package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/transport"
)

const scenarioProbeYAML = `max_requests: 40
min_interval: 200ms
total_timeout: 2m
required_chains:
  - ethereum
  - bnb
  - bitcoin
  - solana
  - tron
submit_probe_authorization: synthetic-probe-approval
chains:
  ethereum:
    endpoint: https://ethereum.invalid
  bnb:
    endpoint: https://bnb.invalid
  bitcoin:
    endpoint: https://bitcoin.invalid
  solana:
    endpoint: https://solana.invalid
  tron:
    endpoint: https://tron.invalid
`

type scenarioProbeCall struct {
	chain, path string
	body        []byte
}

type scenarioProbeFake struct {
	calls   []scenarioProbeCall
	reads   int
	accept  string
	failing string
	missing string
	idErr   error
}

func (f *scenarioProbeFake) count(chain string) int {
	n := 0
	for _, c := range f.calls {
		if c.chain == chain {
			n++
		}
	}
	return n
}

func (f *scenarioProbeFake) deps(env map[string]string, yaml string) scenarioProbeDeps {
	identity := map[string]scenarioProbeIdentity{}
	for _, chain := range scenarioChains {
		identity[chain] = func(context.Context, string, scenarioChainConfig, transport.HTTPDoer) error { return f.idErr }
	}
	return scenarioProbeDeps{
		LookupEnv: func(k string) (string, bool) { v, ok := env[k]; return v, ok },
		ReadFile: func(string) ([]byte, error) {
			f.reads++
			return []byte(yaml), nil
		},
		NewDoer: func(c scenarioChainConfig) (transport.HTTPDoer, func(), error) {
			u, _ := url.Parse(c.Endpoint)
			chain := strings.TrimSuffix(u.Hostname(), ".invalid")
			return scenarioDoerFunc(func(r *http.Request) (*http.Response, error) {
				var body []byte
				if r.Body != nil {
					body, _ = io.ReadAll(r.Body)
				}
				f.calls = append(f.calls, scenarioProbeCall{chain: chain, path: r.URL.Path, body: body})
				if chain == f.failing {
					return nil, errors.New("synthetic transport failure")
				}
				return f.respond(chain, body), nil
			}), func() {}, nil
		},
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0) },
		Wait:     func(context.Context, time.Duration) error { return nil },
		Identity: identity,
	}
}

func (f *scenarioProbeFake) respond(chain string, body []byte) *http.Response {
	if chain == "tron" {
		out := `{"result":false,"code":"SIGERROR","message":"synthetic"}`
		if f.accept == chain {
			out = `{"result":true,"txid":"` + strings.Repeat("ab", 32) + `"}`
		}
		return scenarioProbeHTTP(out)
	}
	var req struct {
		Version string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(body, &req)
	msg := map[string]any{"id": req.ID}
	if req.Version != "" {
		msg["jsonrpc"] = req.Version
	}
	switch {
	case f.accept == chain:
		msg["result"] = map[string]string{
			"ethereum": "0x" + strings.Repeat("ab", 32),
			"bnb":      "0x" + strings.Repeat("ab", 32),
			"bitcoin":  strings.Repeat("ab", 32),
			"solana":   strings.Repeat("1", 88),
		}[chain]
		if chain == "bitcoin" {
			msg["error"] = nil
		}
	case f.missing == chain:
		msg["error"] = map[string]any{"code": -32601, "message": "Method not found"}
	default:
		msg["error"] = map[string]any{"code": -32000, "message": "synthetic rejection"}
		if chain == "bitcoin" {
			msg["result"] = nil
		}
	}
	out, _ := json.Marshal(msg)
	return scenarioProbeHTTP(string(out))
}

func scenarioProbeHTTP(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func scenarioProbeOn() map[string]string {
	return map[string]string{scenarioSwitch: "1", scenarioProbeSwitch: "1"}
}

func TestScenarioSubmitProbe(t *testing.T) {
	t.Run("payload_constants", testScenarioProbePayloadConstants)
	t.Run("switch_off_zero_calls", testScenarioProbeSwitchOff)
	t.Run("missing_authorization_blocked", testScenarioProbeMissingAuthorization)
	t.Run("identity_failure_zero_submit", testScenarioProbeIdentityFailure)
	t.Run("one_call_per_chain_fixed_payload", testScenarioProbeOneCallPerChain)
	t.Run("node_accept_fails_and_stops", testScenarioProbeAcceptStops)
	t.Run("transport_error_blocked_no_retry", testScenarioProbeTransportError)
	t.Run("method_missing_capability_blocked", testScenarioProbeMethodMissing)
}

// The probe payloads are fixed malformed constants, never constructed transactions.
func testScenarioProbePayloadConstants(t *testing.T) {
	if scenarioProbeEVMPayload != "0x00" {
		t.Errorf("EVM payload = %q", scenarioProbeEVMPayload)
	}
	if scenarioProbeBitcoinPayload != "00" {
		t.Errorf("Bitcoin payload = %q", scenarioProbeBitcoinPayload)
	}
	if scenarioProbeSolanaPayload != base64.StdEncoding.EncodeToString([]byte{0}) {
		t.Errorf("Solana payload = %q", scenarioProbeSolanaPayload)
	}
	var tron map[string]any
	if err := json.Unmarshal([]byte(scenarioProbeTronPayload), &tron); err != nil {
		t.Fatalf("Tron payload is not a JSON object: %v", err)
	}
	for _, k := range []string{"raw_data", "raw_data_hex", "signature", "txID"} {
		if _, ok := tron[k]; ok {
			t.Errorf("Tron payload contains %q", k)
		}
	}
}

func testScenarioProbeSwitchOff(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  map[string]string
	}{
		{"both_unset", map[string]string{}},
		{"scenario_unset", map[string]string{scenarioProbeSwitch: "1"}},
		{"probe_unset", map[string]string{scenarioSwitch: "1"}},
		{"scenario_zero", map[string]string{scenarioSwitch: "0", scenarioProbeSwitch: "1"}},
		{"probe_zero", map[string]string{scenarioSwitch: "1", scenarioProbeSwitch: "0"}},
		{"probe_true", map[string]string{scenarioSwitch: "1", scenarioProbeSwitch: "true"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &scenarioProbeFake{}
			got := runScenarioSubmitProbe(context.Background(), f.deps(tc.env, scenarioProbeYAML))
			if got.Status != "NOT_RUN" || got.Category != "disabled" {
				t.Errorf("report = %s/%s; want NOT_RUN/disabled", got.Status, got.Category)
			}
			if len(f.calls) != 0 || f.reads != 0 || got.Requests != 0 {
				t.Errorf("calls=%d reads=%d requests=%d; want zero", len(f.calls), f.reads, got.Requests)
			}
		})
	}
}

func testScenarioProbeMissingAuthorization(t *testing.T) {
	yaml := strings.Replace(scenarioProbeYAML, "submit_probe_authorization: synthetic-probe-approval\n", "", 1)
	f := &scenarioProbeFake{}
	got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), yaml))
	if got.Status != "BLOCKED" || got.Category != "authorization" {
		t.Errorf("report = %s/%s; want BLOCKED/authorization", got.Status, got.Category)
	}
	if len(f.calls) != 0 || got.Requests != 0 {
		t.Errorf("calls=%d requests=%d; want zero", len(f.calls), got.Requests)
	}
}

func testScenarioProbeIdentityFailure(t *testing.T) {
	f := &scenarioProbeFake{idErr: errors.New("synthetic network mismatch")}
	got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), scenarioProbeYAML))
	if len(f.calls) != 0 || got.Requests != 0 {
		t.Errorf("submit calls=%d requests=%d; want zero", len(f.calls), got.Requests)
	}
	if got.Status != "BLOCKED" && got.Status != "FAIL" {
		t.Errorf("status = %q; want BLOCKED or FAIL", got.Status)
	}
	for _, chain := range scenarioChains {
		if s := got.Chains[chain].Status; s == "" || s == "PASS" {
			t.Errorf("%s status = %q; want non-PASS outcome", chain, s)
		}
	}
}

func testScenarioProbeOneCallPerChain(t *testing.T) {
	f := &scenarioProbeFake{}
	got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), scenarioProbeYAML))
	if got.Status != "PASS" {
		t.Errorf("status = %s/%s; want PASS", got.Status, got.Category)
	}
	if got.Requests != len(scenarioChains) {
		t.Errorf("requests = %d; want %d", got.Requests, len(scenarioChains))
	}
	for _, chain := range scenarioChains {
		if n := f.count(chain); n != 1 {
			t.Errorf("%s calls = %d; want 1", chain, n)
		}
		if s := got.Chains[chain].Status; s != "PASS" {
			t.Errorf("%s status = %q; want PASS", chain, s)
		}
	}
	for _, c := range f.calls {
		scenarioProbeCheckBody(t, c)
	}
}

func scenarioProbeCheckBody(t *testing.T, c scenarioProbeCall) {
	t.Helper()
	if c.chain == "tron" {
		if c.path != "/wallet/broadcasttransaction" {
			t.Errorf("tron path = %q", c.path)
		}
		var got, want any
		if json.Unmarshal(c.body, &got) != nil || json.Unmarshal([]byte(scenarioProbeTronPayload), &want) != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("tron body = %s; want %s", c.body, scenarioProbeTronPayload)
		}
		return
	}
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(c.body, &req); err != nil || len(req.Params) == 0 {
		t.Errorf("%s body = %s; want JSON-RPC with params", c.chain, c.body)
		return
	}
	var first string
	_ = json.Unmarshal(req.Params[0], &first)
	want := map[string][2]string{
		"ethereum": {"eth_sendRawTransaction", scenarioProbeEVMPayload},
		"bnb":      {"eth_sendRawTransaction", scenarioProbeEVMPayload},
		"bitcoin":  {"sendrawtransaction", scenarioProbeBitcoinPayload},
		"solana":   {"sendTransaction", scenarioProbeSolanaPayload},
	}[c.chain]
	method, payload := want[0], want[1]
	if req.Method != method || first != payload {
		t.Errorf("%s request = %s(%q); want %s(%q)", c.chain, req.Method, first, method, payload)
	}
	if c.chain == "solana" && !bytes.Contains(c.body, []byte(`"encoding":"base64"`)) {
		t.Errorf("solana body = %s; want base64 encoding", c.body)
	}
}

func testScenarioProbeAcceptStops(t *testing.T) {
	for i, accepting := range scenarioChains {
		t.Run(accepting, func(t *testing.T) {
			f := &scenarioProbeFake{accept: accepting}
			got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), scenarioProbeYAML))
			if got.Status != "FAIL" {
				t.Errorf("status = %s/%s; want FAIL", got.Status, got.Category)
			}
			if s := got.Chains[accepting].Status; s != "FAIL" {
				t.Errorf("%s status = %q; want FAIL", accepting, s)
			}
			for j, chain := range scenarioChains {
				want := 1
				if j > i {
					want = 0
				}
				if n := f.count(chain); n != want {
					t.Errorf("%s calls = %d; want %d", chain, n, want)
				}
			}
		})
	}
}

func testScenarioProbeTransportError(t *testing.T) {
	for _, failing := range scenarioChains {
		t.Run(failing, func(t *testing.T) {
			f := &scenarioProbeFake{failing: failing}
			got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), scenarioProbeYAML))
			if got.Status != "BLOCKED" {
				t.Errorf("status = %s/%s; want BLOCKED", got.Status, got.Category)
			}
			if s := got.Chains[failing].Status; s != "BLOCKED" {
				t.Errorf("%s status = %q; want BLOCKED", failing, s)
			}
			if n := f.count(failing); n != 1 {
				t.Errorf("%s calls = %d; want exactly 1 (no retry)", failing, n)
			}
		})
	}
}

func testScenarioProbeMethodMissing(t *testing.T) {
	f := &scenarioProbeFake{missing: "ethereum"}
	got := runScenarioSubmitProbe(context.Background(), f.deps(scenarioProbeOn(), scenarioProbeYAML))
	if issue := got.Chains["ethereum"]; issue != (scenarioIssue{"BLOCKED", "capability"}) {
		t.Errorf("ethereum = %+v; want BLOCKED/capability", issue)
	}
	if got.Status != "BLOCKED" {
		t.Errorf("status = %s/%s; want BLOCKED", got.Status, got.Category)
	}
	if n := f.count("ethereum"); n != 1 {
		t.Errorf("ethereum calls = %d; want 1", n)
	}
}
