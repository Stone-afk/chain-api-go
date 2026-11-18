package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
	"github.com/Stone-afk/chain-api-go/bnb"
	"github.com/Stone-afk/chain-api-go/ethereum"
	"github.com/Stone-afk/chain-api-go/httpjson"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/solana"
	"github.com/Stone-afk/chain-api-go/transport"
	"github.com/Stone-afk/chain-api-go/tron"
)

const (
	scenarioPathSecret   = "SYNTHETIC_PATH_TOKEN_7f3a"
	scenarioHeaderSecret = "SYNTHETIC_HEADER_VALUE_91c2"
)

const scenarioBaseYAML = `# synthetic scenario config
max_requests: 40
min_interval: 200ms
total_timeout: 2m
required_chains:
  - ethereum
  - tron
known_unsupported:
  - ethereum/FeeHistory
chains:
  ethereum:
    endpoint: https://ethereum.invalid/v3/` + scenarioPathSecret + `
    params:
      expected_chain_id: 1
  tron:
    endpoint: https://tron.invalid
    headers:
      TRON-PRO-API-KEY: ` + scenarioHeaderSecret + `
`

type scenarioDoerFunc func(*http.Request) (*http.Response, error)

func (f scenarioDoerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func scenarioRedacted(t *testing.T, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	text := string(b) + fmt.Sprintf("%v %+v %#v", v, v, v)
	if e, ok := v.(error); ok && e != nil {
		text += e.Error()
	}
	for _, secret := range []string{scenarioPathSecret, scenarioHeaderSecret} {
		if strings.Contains(text, secret) {
			t.Error("output retained synthetic secret")
		}
	}
}

func scenarioReadIDs(chains ...string) []string {
	var ids []string
	for _, e := range scenarioCatalog() {
		for _, c := range chains {
			if e.Chain == c && e.Kind != "submit" {
				ids = append(ids, e.ID)
			}
		}
	}
	return ids
}

func TestScenario(t *testing.T) {
	t.Run("yaml_subset", testScenarioYAML)
	t.Run("config_validation", testScenarioConfig)
	t.Run("switch_off_reads_only_switch", testScenarioSwitchOff)
	t.Run("switch_on_invalid_config_blocked", testScenarioSwitchOnBlocked)
	t.Run("budget", testScenarioBudget)
	t.Run("ledger", testScenarioLedger)
	t.Run("catalog", testScenarioCatalog)
	t.Run("redaction", testScenarioRedaction)
	t.Run("template", testScenarioTemplate)
}

// LS-02/03 risk: unsupported YAML must be rejected rather than guessed.
func testScenarioYAML(t *testing.T) {
	positive := "# head\n\ntop: value\nnested:\n  key: Basic abc  # trailing\n  url: https://x.invalid/a#b\n  deeper:\n    n: 1\nlist:\n  - a\n  - b\n"
	got, err := parseScenarioYAML([]byte(positive))
	want := map[string]any{"top": "value", "nested": map[string]any{"key": "Basic abc", "url": "https://x.invalid/a#b", "deeper": map[string]any{"n": "1"}}, "list": []string{"a", "b"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("positive subset = %#v, %v; want %#v", got, err, want)
	}
	for _, tc := range []struct{ name, raw string }{
		{"tab", "a:\n\tb: 1\n"},
		{"anchor_alias", "a: &x 1\nb: *x\n"},
		{"flow_collection", "a: [x, y]\n"},
		{"multiline_scalar", "a: |\n  one\n  two\n"},
		{"multi_document", "---\na: 1\n---\nb: 2\n"},
		{"duplicate_key", "a: 1\na: 2\n"},
		{"bad_indent", "a:\n  b: 1\n   c: 2\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseScenarioYAML([]byte(tc.raw)); !errors.Is(err, errScenarioYAML) {
				t.Errorf("err = %v; want errScenarioYAML", err)
			}
		})
	}
}

func testScenarioConfig(t *testing.T) {
	got, err := parseScenarioConfig([]byte(scenarioBaseYAML))
	want := scenarioConfig{MaxRequests: 40, MinInterval: 200 * time.Millisecond, TotalTimeout: 2 * time.Minute, RequiredChains: []string{"ethereum", "tron"}, KnownUnsupported: []string{"ethereum/FeeHistory"}, Chains: map[string]scenarioChainConfig{
		"ethereum": {Endpoint: "https://ethereum.invalid/v3/" + scenarioPathSecret, Params: map[string]string{"expected_chain_id": "1"}},
		"tron":     {Endpoint: "https://tron.invalid", Headers: map[string]string{"TRON-PRO-API-KEY": scenarioHeaderSecret}},
	}}
	if err != nil || fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Errorf("valid config rejected or mismatched (err category %v)", errors.Is(err, errScenarioConfig))
	}
	tronEndpoint := "    endpoint: https://tron.invalid\n"
	tronHeaders := "    headers:\n      TRON-PRO-API-KEY: " + scenarioHeaderSecret + "\n"
	for _, tc := range []struct{ name, old, new string }{
		{"unknown_top_key", "max_requests: 40\n", "max_requests: 40\nextra: 1\n"},
		{"unknown_chain_key", tronEndpoint, tronEndpoint + "    bogus: x\n"},
		{"unknown_chain_name", "chains:\n", "chains:\n  polygon:\n    endpoint: https://polygon.invalid\n"},
		{"type_integer", "max_requests: 40", "max_requests: many"},
		{"type_sequence", "required_chains:\n  - ethereum\n  - tron\n", "required_chains: ethereum\n"},
		{"type_mapping", tronHeaders, "    headers: none\n"},
		{"scheme_not_http", "https://tron.invalid", "ftp://tron.invalid"},
		{"missing_host", "https://tron.invalid", "https:///rpc"},
		{"fragment", "https://tron.invalid", "https://tron.invalid/rpc#frag"},
		{"userinfo", "https://tron.invalid", "https://user:pw@tron.invalid"},
		{"header_name", "TRON-PRO-API-KEY:", "TRON@KEY:"},
		{"header_value", "TRON-PRO-API-KEY: " + scenarioHeaderSecret, "TRON-PRO-API-KEY: bad\x7f" + scenarioHeaderSecret},
		{"limit_zero", "max_requests: 40", "max_requests: 0"},
		{"limit_negative", "max_requests: 40", "max_requests: -1"},
		{"duration_text", "min_interval: 200ms", "min_interval: soon"},
		{"duration_zero", "min_interval: 200ms", "min_interval: 0s"},
		{"duration_negative", "total_timeout: 2m", "total_timeout: -2m"},
		{"known_unsupported_unknown_id", "ethereum/FeeHistory", "ethereum/NoSuchMethod"},
		{"required_not_configured", "  - tron\n", "  - solana\n"},
		{"missing_endpoint", "    endpoint: https://ethereum.invalid/v3/" + scenarioPathSecret + "\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(scenarioBaseYAML, tc.old, tc.new, 1)
			if raw == scenarioBaseYAML {
				t.Fatal("mutation did not apply")
			}
			_, err := parseScenarioConfig([]byte(raw))
			if !errors.Is(err, errScenarioConfig) || errors.Is(err, errScenarioYAML) {
				t.Errorf("err = %v; want semantic errScenarioConfig", err)
			}
			scenarioRedacted(t, err)
		})
	}
}

// LS-02: with the switch off only the switch itself may be looked up.
func testScenarioSwitchOff(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		present     bool
	}{{"unset", "", false}, {"zero", "0", true}, {"empty", "", true}, {"true_word", "true", true}, {"yes_word", "yes", true}} {
		t.Run(tc.name, func(t *testing.T) {
			var keys []string
			forbid := func(name string) { t.Errorf("disabled scenario touched %s", name) }
			deps := scenarioDeps{
				LookupEnv: func(k string) (string, bool) { keys = append(keys, k); return tc.value, tc.present },
				ReadFile:  func(string) ([]byte, error) { forbid("ReadFile"); return nil, fs.ErrNotExist },
				NewDoer: func(scenarioChainConfig) (transport.HTTPDoer, func(), error) {
					forbid("NewDoer")
					return nil, func() {}, errors.New("forbidden")
				},
				Now:  func() time.Time { forbid("Now"); return time.Time{} },
				Wait: func(context.Context, time.Duration) error { forbid("Wait"); return nil },
			}
			r := runScenario(context.Background(), deps)
			if !reflect.DeepEqual(keys, []string{scenarioSwitch}) {
				t.Errorf("LookupEnv keys = %q; want exactly [%s]", keys, scenarioSwitch)
			}
			if r.Status != "NOT_RUN" || r.Category != "disabled" || r.Requests != 0 || scenarioOutcome(r) != "SKIP" {
				t.Errorf("report = %s/%s requests=%d outcome=%s; want NOT_RUN/disabled 0 SKIP", r.Status, r.Category, r.Requests, scenarioOutcome(r))
			}
		})
	}
}

// LS-03: enabled with missing or invalid config is BLOCKED and fails the test.
func testScenarioSwitchOnBlocked(t *testing.T) {
	invalid := strings.Replace(scenarioBaseYAML, "max_requests: 40", "max_requests: 0", 1)
	for _, tc := range []struct {
		name string
		raw  []byte
		err  error
	}{
		{"missing_file", nil, fs.ErrNotExist},
		{"read_error", nil, errors.New("permission denied " + scenarioPathSecret)},
		{"yaml_invalid", []byte("a:\n\tb: " + scenarioHeaderSecret + "\n"), nil},
		{"config_invalid", []byte(invalid), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			doers := 0
			deps := scenarioDeps{
				LookupEnv: func(k string) (string, bool) {
					if k == scenarioSwitch {
						return "1", true
					}
					return "", false
				},
				ReadFile: func(p string) ([]byte, error) { paths = append(paths, p); return tc.raw, tc.err },
				NewDoer: func(scenarioChainConfig) (transport.HTTPDoer, func(), error) {
					doers++
					return nil, func() {}, errors.New("forbidden")
				},
				Now:  time.Now,
				Wait: func(context.Context, time.Duration) error { return nil },
			}
			r := runScenario(context.Background(), deps)
			if !reflect.DeepEqual(paths, []string{scenarioConfigPath}) {
				t.Errorf("ReadFile paths = %q; want [%s]", paths, scenarioConfigPath)
			}
			if r.Status != "BLOCKED" || scenarioOutcome(r) != "FAIL" || doers != 0 || r.Requests != 0 {
				t.Errorf("report = %s outcome=%s doers=%d requests=%d; want BLOCKED FAIL 0 0", r.Status, scenarioOutcome(r), doers, r.Requests)
			}
			scenarioRedacted(t, r)
		})
	}
}

// LS-04: the N+1th request never reaches downstream; pacing uses the injected Wait.
func testScenarioBudget(t *testing.T) {
	const interval = 750 * time.Millisecond
	type step struct {
		canceled bool
		waitErr  error
		want     string
	}
	for _, tc := range []struct {
		name                       string
		limit                      int
		steps                      []step
		downstream, waits, counted int
	}{
		{"within_limit_waits_between", 3, []step{{want: "ok"}, {want: "ok"}, {want: "ok"}}, 3, 2, 3},
		{"n_plus_one_blocked", 2, []step{{want: "ok"}, {want: "ok"}, {want: "budget"}}, 2, 1, 2},
		{"canceled_not_counted", 1, []step{{canceled: true, want: "canceled"}, {want: "ok"}}, 1, 0, 1},
		{"wait_cancel_not_counted", 3, []step{{want: "ok"}, {waitErr: context.Canceled, want: "canceled"}, {want: "ok"}}, 2, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			downstream, waits, current := 0, 0, 0
			next := scenarioDoerFunc(func(*http.Request) (*http.Response, error) {
				downstream++
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
			})
			wait := func(_ context.Context, d time.Duration) error {
				waits++
				if d != interval {
					t.Errorf("wait duration = %v; want %v", d, interval)
				}
				return tc.steps[current].waitErr
			}
			b := newScenarioBudget(next, tc.limit, interval, wait)
			for i, s := range tc.steps {
				current = i
				ctx, cancel := context.WithCancel(context.Background())
				if s.canceled {
					cancel()
				}
				req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://budget.invalid/rpc", nil)
				resp, err := b.Do(req)
				cancel()
				if resp != nil {
					resp.Body.Close()
				}
				switch s.want {
				case "ok":
					if err != nil {
						t.Errorf("step %d err = %v; want nil", i, err)
					}
				case "budget":
					if !errors.Is(err, errScenarioBudget) || classifyScenarioError(err) != (scenarioIssue{"BLOCKED", "budget"}) {
						t.Errorf("step %d err = %v; want BLOCKED budget", i, err)
					}
				case "canceled":
					if !errors.Is(err, context.Canceled) {
						t.Errorf("step %d err = %v; want context.Canceled", i, err)
					}
				}
			}
			if downstream != tc.downstream || waits != tc.waits || b.Requests() != tc.counted {
				t.Errorf("downstream=%d waits=%d counted=%d; want %d %d %d", downstream, waits, b.Requests(), tc.downstream, tc.waits, tc.counted)
			}
		})
	}
}

// Ledger priority: FAIL > BLOCKED > NOT_RUN > PASS over required chains.
func testScenarioLedger(t *testing.T) {
	results := func(status string, chains ...string) map[string]string {
		m := map[string]string{}
		for _, id := range scenarioReadIDs(chains...) {
			m[id] = status
		}
		return m
	}
	with := func(m map[string]string, id, status string) map[string]string { m[id] = status; return m }
	all := []string{"ethereum", "bnb", "bitcoin", "solana", "tron"}
	for _, tc := range []struct {
		name     string
		required []string
		results  map[string]string
		want     string
	}{
		{"all_pass", all, results("PASS", all...), "PASS"},
		{"submit_not_authorized_ignored", all, with(results("PASS", all...), "tron/BroadcastTransaction", "NOT_AUTHORIZED"), "PASS"},
		{"fail_beats_blocked", all, with(with(results("PASS", all...), "ethereum/ChainID", "BLOCKED"), "tron/GetNodeInfo", "FAIL"), "FAIL"},
		{"blocked_beats_not_run", all, with(with(results("PASS", all...), "bnb/ChainID", "NOT_RUN"), "bitcoin/GetBlockCount", "BLOCKED"), "BLOCKED"},
		{"required_missing_not_pass", all, results("PASS", "ethereum", "bnb", "bitcoin", "solana"), "NOT_RUN"},
		{"narrowed_ignores_unrun", []string{"tron"}, results("PASS", "tron"), "PASS"},
		{"narrowed_ignores_blocked", []string{"tron"}, with(results("PASS", "tron"), "ethereum/ChainID", "BLOCKED"), "PASS"},
		{"narrowed_still_fails", []string{"tron"}, with(results("PASS", "tron"), "solana/GetGenesisHash", "FAIL"), "FAIL"},
		{"empty_required_blocked", nil, results("PASS", all...), "BLOCKED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarizeScenario(tc.required, tc.results); got != tc.want {
				t.Errorf("summary = %q; want %q", got, tc.want)
			}
		})
	}
}

// LS-05: the catalog mirrors the five clients' exported method sets plus two Raw entries.
func testScenarioCatalog(t *testing.T) {
	clients := []struct {
		chain string
		typ   reflect.Type
	}{
		{"ethereum", reflect.TypeOf((*ethereum.Client)(nil))},
		{"bnb", reflect.TypeOf((*bnb.Client)(nil))},
		{"bitcoin", reflect.TypeOf((*bitcoin.Client)(nil))},
		{"solana", reflect.TypeOf((*solana.Client)(nil))},
		{"tron", reflect.TypeOf((*tron.Client)(nil))},
	}
	submit := map[string]bool{"ethereum/SendRawTransaction": true, "bnb/SendRawTransaction": true, "bitcoin/SendRawTransaction": true, "solana/SendEncodedTransactionWithOpts": true, "tron/BroadcastTransaction": true}
	var reflected []string
	for _, c := range clients {
		for i := 0; i < c.typ.NumMethod(); i++ {
			reflected = append(reflected, c.chain+"/"+c.typ.Method(i).Name)
		}
	}
	nonSubmit := 0
	for _, id := range reflected {
		if !submit[id] {
			nonSubmit++
		}
	}
	if nonSubmit != 62 || len(reflected) != 67 {
		t.Fatalf("SDK reflection drifted: %d non-submit, %d total", nonSubmit, len(reflected))
	}
	if _, ok := reflect.TypeOf((*jsonrpc.Client)(nil)).MethodByName("RawCall"); !ok {
		t.Fatal("jsonrpc.Client.RawCall missing")
	}
	if _, ok := reflect.TypeOf((*httpjson.Client)(nil)).MethodByName("RawPost"); !ok {
		t.Fatal("httpjson.Client.RawPost missing")
	}
	raw := map[string]bool{"ethereum/jsonrpc.RawCall": true, "tron/httpjson.RawPost": true}
	var methods []string
	seen, rawSeen := map[string]bool{}, 0
	for _, e := range scenarioCatalog() {
		if seen[e.ID] || e.ID != e.Chain+"/"+e.Method {
			t.Errorf("catalog entry %q duplicated or malformed", e.ID)
		}
		seen[e.ID] = true
		wantKind, wantStatus := "read", "NOT_RUN"
		switch {
		case raw[e.ID]:
			wantKind, rawSeen = "raw", rawSeen+1
		case submit[e.ID]:
			wantKind, wantStatus = "submit", "NOT_AUTHORIZED"
		}
		if e.Kind != wantKind || e.Status != wantStatus {
			t.Errorf("%s = %s/%s; want %s/%s", e.ID, e.Kind, e.Status, wantKind, wantStatus)
		}
		if !raw[e.ID] {
			methods = append(methods, e.ID)
		}
	}
	sort.Strings(reflected)
	sort.Strings(methods)
	if !reflect.DeepEqual(methods, reflected) || rawSeen != len(raw) {
		t.Errorf("catalog has %d method IDs and %d raw IDs; want %d and %d", len(methods), rawSeen, len(reflected), len(raw))
	}
}

// LS-11: classified output never carries URL path tokens or header values.
func testScenarioRedaction(t *testing.T) {
	urlErr := &url.Error{Op: "Post", URL: "https://ethereum.invalid/v3/" + scenarioPathSecret, Err: errors.New("connection refused")}
	for _, tc := range []struct {
		name string
		err  error
		want scenarioIssue
	}{
		{"transport", apierror.Error{Kind: apierror.Transport, Method: "eth_chainId", Cause: urlErr}, scenarioIssue{"BLOCKED", "environment"}},
		{"http_body", apierror.Error{Kind: apierror.HTTP, HTTPStatus: 401, ResponseBody: []byte("bad key " + scenarioHeaderSecret), Cause: urlErr}, scenarioIssue{"BLOCKED", "environment"}},
		{"rpc_message", apierror.Error{Kind: apierror.RPC, Message: "TRON-PRO-API-KEY=" + scenarioHeaderSecret, Data: json.RawMessage(`"` + scenarioPathSecret + `"`)}, scenarioIssue{"FAIL", "rpc"}},
		{"protocol_wrapped", fmt.Errorf("decode %s: %w", scenarioPathSecret, apierror.Error{Kind: apierror.Protocol}), scenarioIssue{"FAIL", "protocol"}},
		{"network_mismatch", apierror.Error{Kind: apierror.NetworkMismatch, Cause: urlErr}, scenarioIssue{"FAIL", "network_mismatch"}},
		{"canceled", fmt.Errorf("post %s: %w", urlErr.URL, context.Canceled), scenarioIssue{"BLOCKED", "canceled"}},
		{"deadline", fmt.Errorf("post %s: %w", urlErr.URL, context.DeadlineExceeded), scenarioIssue{"BLOCKED", "deadline"}},
		{"budget", fmt.Errorf("%w %s", errScenarioBudget, scenarioPathSecret), scenarioIssue{"BLOCKED", "budget"}},
		{"unclassified", errors.New(scenarioHeaderSecret), scenarioIssue{"FAIL", "unclassified"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyScenarioError(tc.err)
			if got != tc.want {
				t.Errorf("classification = %+v; want %+v", got, tc.want)
			}
			diag := scenarioDiagnostic("ethereum/ChainID", tc.err)
			for _, part := range []string{"ethereum/ChainID", tc.want.Status, tc.want.Category} {
				if !strings.Contains(diag, part) {
					t.Errorf("diagnostic missing %q", part)
				}
			}
			scenarioRedacted(t, got)
			scenarioRedacted(t, diag)
		})
	}
}

// The committed template must parse and point only at .invalid hosts with placeholders.
func testScenarioTemplate(t *testing.T) {
	raw, err := os.ReadFile("config.example.yaml")
	if err != nil {
		t.Fatalf("template unreadable: %v", err)
	}
	cfg, err := parseScenarioConfig(raw)
	if err != nil {
		t.Fatalf("template rejected: %v", err)
	}
	if len(cfg.Chains) != 5 || len(cfg.RequiredChains) != 5 || cfg.SubmitProbeAuthorization != "" {
		t.Errorf("template chains=%d required=%d probe authorized=%v", len(cfg.Chains), len(cfg.RequiredChains), cfg.SubmitProbeAuthorization != "")
	}
	for name, c := range cfg.Chains {
		u, err := url.Parse(c.Endpoint)
		if err != nil || !strings.HasSuffix(u.Hostname(), ".invalid") {
			t.Errorf("%s endpoint host is not .invalid", name)
		}
		for k, v := range c.Headers {
			if !strings.Contains(v, "REPLACE_ME") {
				t.Errorf("%s header %s is not a placeholder", name, k)
			}
		}
	}
}
