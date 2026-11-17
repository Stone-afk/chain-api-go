package jsonrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/transport"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type nilDoer struct{}

func (*nilDoer) Do(*http.Request) (*http.Response, error) { panic("typed nil reached") }

func config() jsonrpc.Config {
	return jsonrpc.Config{Endpoint: "https://rpc.invalid/test?api_key=endpoint-secret-019", Headers: http.Header{"X-Test": {"original"}, "Authorization": {"Bearer header-secret-019"}}, MaxResponseBytes: 4096, Version: jsonrpc.V2}
}

// Do not stop at New's error: an unimplemented constructor must not hide the
// distinct Call/RawCall assertions which T019 is intended to make RED.
func client(t *testing.T, cfg jsonrpc.Config, d transport.HTTPDoer) *jsonrpc.Client {
	t.Helper()
	c, err := jsonrpc.New(cfg, d)
	if err != nil {
		t.Errorf("valid New: %v", err)
	}
	if c == nil {
		t.Fatal("valid New returned nil client")
	}
	return c
}

func kind(t *testing.T, err error, want apierror.Kind) apierror.Error {
	t.Helper()
	var e apierror.Error
	if !errors.As(err, &e) {
		t.Errorf("want apierror.Error value kind %s; got %T", want, err)
		return e
	}
	if e.Kind != want {
		t.Errorf("kind = %s, want %s", e.Kind, want)
	}
	if e.NodeCode != "" {
		t.Error("JSON-RPC error fabricated a textual NodeCode")
	}
	return e
}

func errorFields(t *testing.T, e apierror.Error, method string, status, code int, hasCode bool) {
	t.Helper()
	if e.Method != method || e.HTTPStatus != status || e.RPCCode != code || e.HasRPCCode != hasCode {
		t.Error("method/status/RPC-code metadata lost or fabricated")
	}
}

func invoke(c *jsonrpc.Client, raw bool, ctx context.Context, method string, params, result any) error {
	if raw {
		return c.RawCall(ctx, method, params, result)
	}
	return c.Call(ctx, method, params, result)
}

func requestID(t *testing.T, r *http.Request) json.RawMessage {
	t.Helper()
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("request JSON: %v", err)
		return json.RawMessage(`0`)
	}
	id := req["id"]
	if len(id) == 0 || string(id) == "null" {
		t.Error("request must have non-null ID")
		return json.RawMessage(`0`)
	}
	var n uint64
	if err := json.Unmarshal(id, &n); err != nil {
		t.Error("counter ID must be unsigned integer")
	}
	return id
}

func response(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
}

func TestNewRejectsInvalidConfigurationWithoutIO(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*jsonrpc.Config)
		d      transport.HTTPDoer
	}{
		{"nil", nil, nil},
		{"empty-endpoint", func(c *jsonrpc.Config) { c.Endpoint = "" }, nil},
		{"relative-endpoint", func(c *jsonrpc.Config) { c.Endpoint = "/rpc" }, nil},
		{"unsupported-scheme", func(c *jsonrpc.Config) { c.Endpoint = "file:///rpc" }, nil},
		{"missing-host", func(c *jsonrpc.Config) { c.Endpoint = "https:///rpc" }, nil},
		{"zero-limit", func(c *jsonrpc.Config) { c.MaxResponseBytes = 0 }, nil},
		{"negative-limit", func(c *jsonrpc.Config) { c.MaxResponseBytes = -1 }, nil},
		{"missing-version", func(c *jsonrpc.Config) { c.Version = "" }, nil},
		{"unknown-version", func(c *jsonrpc.Config) { c.Version = "unknown" }, nil},
		{"invalid-header-name", func(c *jsonrpc.Config) { c.Headers["Bad\nName"] = []string{"x"} }, nil},
		{"invalid-header-value", func(c *jsonrpc.Config) { c.Headers["X-Test"] = []string{"x\r\ny"} }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config()
			calls := 0
			d := tc.d
			if tc.change != nil {
				tc.change(&cfg)
				d = doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			}
			_, err := jsonrpc.New(cfg, d)
			errorFields(t, kind(t, err, apierror.Configuration), "", 0, 0, false)
			if calls != 0 {
				t.Errorf("constructor Do count = %d", calls)
			}
		})
	}
}

func TestNewDoesNotInspectTypedNilDoer(t *testing.T) {
	var pointer *nilDoer
	var function doerFunc
	for _, tc := range []struct {
		name string
		doer transport.HTTPDoer
	}{{"pointer", pointer}, {"function", function}} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := jsonrpc.New(config(), tc.doer)
			if err != nil || c == nil {
				t.Fatalf("typed-nil doer was defensively rejected: client=%v error=%v", c, err)
			}
		})
	}
}

func TestNewCopiesConfigurationAndDoesNoIO(t *testing.T) {
	cfg := config()
	calls := 0
	d := doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://rpc.invalid/test?api_key=endpoint-secret-019" || r.Header.Get("X-Test") != "original" || r.Header.Get("X-New") != "" {
			t.Error("constructor did not copy configuration deeply")
		}
		id := requestID(t, r)
		return response(fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":7}`, id)), nil
	})
	c := client(t, cfg, d)
	if calls != 0 {
		t.Errorf("New made %d requests", calls)
	}
	cfg.Headers["X-Test"][0] = "changed"
	cfg.Headers["X-New"] = []string{"added"}
	cfg.Endpoint = "https://changed.invalid"
	cfg.Version = jsonrpc.Bitcoin
	cfg.MaxResponseBytes = 1
	var result int
	if err := c.Call(context.Background(), "probe", nil, &result); err != nil {
		t.Errorf("Call: %v", err)
	}
	if calls != 1 || result != 7 {
		t.Errorf("Do count/result = %d/%d, want 1/7", calls, result)
	}
}

func TestCallRejectsInvalidArgumentsBeforeIO(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, name := range []string{"nil-context", "empty-method", "nil-result", "typed-nil-result", "nonpointer-result", "unencodable-params"} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, name), func(t *testing.T) {
				calls := 0
				c := client(t, config(), doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
				ctx := context.Background()
				method := "probe"
				var out any
				var result any = &out
				var params any = []any{}
				switch name {
				case "nil-context":
					ctx = nil
				case "empty-method":
					method = ""
				case "nil-result":
					result = nil
				case "typed-nil-result":
					var p *int
					result = p
				case "nonpointer-result":
					result = 1
				case "unencodable-params":
					params = make(chan int)
				}
				errorFields(t, kind(t, invoke(c, raw, ctx, method, params, result), apierror.InvalidArgument), method, 0, 0, false)
				if calls != 0 {
					t.Errorf("invalid input issued %d requests", calls)
				}
			})
		}
	}
}

func TestContextCancellation(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
			t.Run(fmt.Sprintf("raw=%v/%v", raw, sentinel), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				cause := fmt.Errorf("synthetic transport-secret-019: %w", sentinel)
				c := client(t, config(), doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Context() != ctx {
						t.Error("request did not carry caller context")
					}
					cancel()
					return nil, cause
				}))
				var out any
				err := invoke(c, raw, ctx, "probe", nil, &out)
				errorFields(t, kind(t, err, apierror.Transport), "probe", 0, 0, false)
				if !errors.Is(err, sentinel) {
					t.Errorf("missing context sentinel %v", sentinel)
				}
				if errors.Unwrap(err) != cause {
					t.Error("must retain original context wrapper")
				}
				if err != nil && !strings.Contains(err.Error(), cause.Error()) {
					t.Error("original context wrapper text lost")
				}
				if calls != 1 {
					t.Errorf("Do count = %d, want 1", calls)
				}
			})
		}
	}
}

func TestAlreadyCanceledContextDoesNoIO(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("raw=%v/expired=%v", raw, expired), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				if expired {
					cancel()
					ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
				}
				defer cancel()
				cancel()
				calls := 0
				c := client(t, config(), doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
				var out any
				err := invoke(c, raw, ctx, "probe", nil, &out)
				errorFields(t, kind(t, err, apierror.Transport), "probe", 0, 0, false)
				if !errors.Is(err, ctx.Err()) || errors.Unwrap(err) != ctx.Err() {
					t.Error("missing pre-canceled context sentinel")
				}
				if calls != 0 {
					t.Errorf("pre-canceled call issued %d requests", calls)
				}
			})
		}
	}
}

func TestConcurrentCallsHaveUniqueIDs(t *testing.T) {
	const count = 64
	var mu sync.Mutex
	ids := make(map[string]bool)
	calls := 0
	c := client(t, config(), doerFunc(func(r *http.Request) (*http.Response, error) {
		id := requestID(t, r)
		mu.Lock()
		calls++
		if ids[string(id)] {
			t.Error("duplicate request ID")
		}
		ids[string(id)] = true
		mu.Unlock()
		return response(fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":7}`, id)), nil
	}))
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(raw bool) {
			defer wg.Done()
			var out int
			if err := invoke(c, raw, context.Background(), "probe", nil, &out); err != nil {
				t.Errorf("concurrent call: %v", err)
			}
			if out != 7 {
				t.Error("concurrent result lost")
			}
		}(i%2 == 0)
	}
	wg.Wait()
	if calls != count || len(ids) != count {
		t.Errorf("Do/unique IDs = %d/%d, want %d", calls, len(ids), count)
	}
}
