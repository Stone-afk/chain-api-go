package jsonrpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
)

func TestVersionToString(t *testing.T) {
	for _, v := range []jsonrpc.Version{jsonrpc.V2, jsonrpc.Bitcoin, jsonrpc.Version("custom")} {
		if v.ToString() != string(v) {
			t.Errorf("Version.ToString lost value %q", string(v))
		}
	}
}

func TestRequestDialectsAndResults(t *testing.T) {
	for _, version := range []jsonrpc.Version{jsonrpc.V2, jsonrpc.Bitcoin} {
		for _, raw := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/raw=%v", version, raw), func(t *testing.T) {
				cfg := config()
				cfg.Version = version
				calls := 0
				params := json.RawMessage(`["payload",0,{"flag":false}]`)
				c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
						t.Error("expected JSON POST")
					}
					var req map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Errorf("request JSON: %v", err)
					}
					wantVersion := `"2.0"`
					if version == jsonrpc.Bitcoin {
						wantVersion = `"1.0"`
					}
					if string(req["jsonrpc"]) != wantVersion || string(req["method"]) != `"probe"` || string(req["params"]) != string(params) {
						t.Error("request dialect/method/params changed")
					}
					id := req["id"]
					if len(id) == 0 || string(id) == "null" {
						t.Error("missing request ID")
						id = json.RawMessage(`0`)
					}
					extra := `"jsonrpc":"2.0",`
					if version == jsonrpc.Bitcoin {
						extra = `"error":null,`
					}
					return response(fmt.Sprintf(`{%s"id":%s,"result":{"n":0,"extra":true}}`, extra, id)), nil
				}))
				var out json.RawMessage
				if err := invoke(c, raw, context.Background(), "probe", params, &out); err != nil {
					t.Errorf("call: %v", err)
				}
				if string(out) != `{"n":0,"extra":true}` {
					t.Error("raw result not preserved")
				}
				if string(params) != `["payload",0,{"flag":false}]` {
					t.Error("caller params mutated")
				}
				if calls != 1 {
					t.Errorf("Do count=%d, want 1", calls)
				}
			})
		}
	}
}

type trackedBody struct {
	io.Reader
	closes int
	reads  int
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.reads += n
	return n, e
}
func (b *trackedBody) Close() error { b.closes++; return nil }

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("Authorization: Bearer read-secret-019 sensitive-body-error")
}

type cancelReader struct {
	cancel context.CancelFunc
	ctx    context.Context
}

func (r cancelReader) Read([]byte) (int, error) {
	r.cancel()
	return 0, fmt.Errorf("Authorization: Bearer read-secret-019 sensitive-read: %w", r.ctx.Err())
}

func TestCancellationWhileReadingClosesBody(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(fmt.Sprintf("raw=%v", raw), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			body := &trackedBody{Reader: cancelReader{cancel, ctx}}
			calls := 0
			c := client(t, config(), doerFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: body}, nil
			}))
			var out any
			err := invoke(c, raw, ctx, "probe", nil, &out)
			errorFields(t, kind(t, err, apierror.Transport), "probe", 200, 0, false)
			if !errors.Is(err, context.Canceled) || errors.Unwrap(err) != context.Canceled {
				t.Error("body cancellation sentinel lost")
			}
			if err != nil && strings.Contains(err.Error(), "sensitive-read") {
				t.Error("body cancellation text leaked")
			}
			if calls != 1 || body.closes != 1 {
				t.Errorf("Do/Close=%d/%d, want 1/1", calls, body.closes)
			}
		})
	}
}

func TestResponseEnvelopes(t *testing.T) {
	// ID is inserted from the observed request; mismatch fixtures deliberately
	// replace it. No test assumes a particular initial counter value.
	cases := []struct {
		name    string
		version jsonrpc.Version
		body    string
		status  int
		want    apierror.Kind
		code    int
		hasCode bool
	}{
		{"v2-result", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":0}`, 200, "", 0, false},
		{"v2-null", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":null}`, 200, "", 0, false},
		{"v2-error", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":-32601,"message":"Authorization: Bearer body-secret-019 sensitive-node","data":"api_key=body-secret-019 sensitive-data"}}`, 200, apierror.RPC, -32601, true},
		{"v2-zero-code", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":0,"message":"zero"}}`, 200, apierror.RPC, 0, true},
		{"bitcoin-result", jsonrpc.Bitcoin, `{"id":ID,"result":0,"error":null}`, 200, "", 0, false},
		{"bitcoin-null", jsonrpc.Bitcoin, `{"id":ID,"result":null,"error":null}`, 200, "", 0, false},
		{"bitcoin-error-http500", jsonrpc.Bitcoin, `{"id":ID,"result":null,"error":{"code":-8,"message":"sensitive-node"}}`, 500, apierror.RPC, -8, true},
		{"v2-error-http500", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":-1,"message":"sensitive-node"}}`, 500, apierror.RPC, -1, true},
		{"http-failure", jsonrpc.V2, `Authorization: Bearer body-secret-019 sensitive-http-body`, 503, apierror.HTTP, 0, false},
		{"http-success-envelope-on-error-status", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":0}`, 503, apierror.HTTP, 0, false},
		{"v2-missing-version", jsonrpc.V2, `{"id":ID,"result":0}`, 200, apierror.Protocol, 0, false},
		{"v2-wrong-version", jsonrpc.V2, `{"jsonrpc":"1.0","id":ID,"result":0}`, 200, apierror.Protocol, 0, false},
		{"missing-id", jsonrpc.V2, `{"jsonrpc":"2.0","result":0}`, 200, apierror.Protocol, 0, false},
		{"null-id", jsonrpc.V2, `{"jsonrpc":"2.0","id":null,"result":0}`, 200, apierror.Protocol, 0, false},
		{"wrong-id", jsonrpc.V2, `{"jsonrpc":"2.0","id":"other","result":0}`, 200, apierror.Protocol, 0, false},
		{"wrong-numeric-id", jsonrpc.V2, `{"jsonrpc":"2.0","id":-1,"result":0}`, 200, apierror.Protocol, 0, false},
		{"stringified-id", jsonrpc.V2, `{"jsonrpc":"2.0","id":"ID","result":0}`, 200, apierror.Protocol, 0, false},
		{"missing-result-error", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID}`, 200, apierror.Protocol, 0, false},
		{"both-result-error", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":0,"error":{"code":1,"message":"x"}}`, 200, apierror.Protocol, 0, false},
		{"v2-result-null-error", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":0,"error":null}`, 200, apierror.Protocol, 0, false},
		{"v2-null-error-only", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":null}`, 200, apierror.Protocol, 0, false},
		{"error-no-code", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"message":"x"}}`, 200, apierror.Protocol, 0, false},
		{"error-fractional-code", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":1.5,"message":"x"}}`, 200, apierror.Protocol, 0, false},
		{"error-string-code", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":"1","message":"x"}}`, 200, apierror.Protocol, 0, false},
		{"error-no-message", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":{"code":1}}`, 200, apierror.Protocol, 0, false},
		{"error-wrong-type", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"error":[]}`, 200, apierror.Protocol, 0, false},
		{"bitcoin-conflict", jsonrpc.Bitcoin, `{"id":ID,"result":0,"error":{"code":1,"message":"x"}}`, 200, apierror.Protocol, 0, false},
		{"bitcoin-missing-result", jsonrpc.Bitcoin, `{"id":ID,"error":null}`, 200, apierror.Protocol, 0, false},
		{"bitcoin-missing-error", jsonrpc.Bitcoin, `{"id":ID,"result":0}`, 200, apierror.Protocol, 0, false},
		{"truncated-json", jsonrpc.V2, `{"jsonrpc":`, 200, apierror.Protocol, 0, false},
		{"trailing-json", jsonrpc.V2, `{"jsonrpc":"2.0","id":ID,"result":0} {}`, 200, apierror.Protocol, 0, false},
		{"array-envelope", jsonrpc.V2, `[]`, 200, apierror.Protocol, 0, false},
		{"null-envelope", jsonrpc.V2, `null`, 200, apierror.Protocol, 0, false},
	}
	for _, raw := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, tc.name), func(t *testing.T) {
				cfg := config()
				cfg.Version = tc.version
				calls := 0
				body := &trackedBody{}
				c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					id := requestID(t, r)
					body.Reader = strings.NewReader(strings.ReplaceAll(tc.body, "ID", string(id)))
					return &http.Response{StatusCode: tc.status, Body: body}, nil
				}))
				out := json.RawMessage(`"unchanged"`)
				err := invoke(c, raw, context.Background(), "probe", nil, &out)
				if tc.want == "" {
					if err != nil {
						t.Errorf("valid envelope: %v", err)
					}
					want := "0"
					if strings.Contains(tc.name, "null") {
						want = "null"
					}
					if string(out) != want {
						t.Errorf("result=%s, want %s", out, want)
					}
				} else {
					e := kind(t, err, tc.want)
					errorFields(t, e, "probe", tc.status, tc.code, tc.hasCode)
					if err != nil && (strings.Contains(err.Error(), "sensitive") || errors.Unwrap(err) != nil) {
						t.Error("unsafe remote error retained")
					}
				}
				if calls != 1 || body.closes != 1 {
					t.Errorf("Do/Close=%d/%d, want 1/1", calls, body.closes)
				}
			})
		}
	}
}

func TestResponseLimitsAndFailures(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, name := range []string{"exact-limit", "over-limit", "over-limit-http", "read-error", "transport-error", "nil-response", "nil-body", "result-type"} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, name), func(t *testing.T) {
				cfg := config()
				cfg.MaxResponseBytes = 128
				calls := 0
				body := &trackedBody{}
				want := apierror.Protocol
				if name == "read-error" || name == "transport-error" || name == "nil-response" {
					want = apierror.Transport
				}
				if name == "exact-limit" {
					want = ""
				}
				if name == "over-limit-http" {
					want = apierror.HTTP
				}
				c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					id := requestID(t, r)
					if name == "transport-error" {
						return nil, errors.New("Authorization: Bearer transport-secret-019 sensitive-transport")
					}
					if name == "nil-response" {
						return nil, nil
					}
					if name == "nil-body" {
						return &http.Response{StatusCode: 200}, nil
					}
					s := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":7}`, id)
					if name == "exact-limit" || strings.HasPrefix(name, "over-limit") {
						s += strings.Repeat(" ", 128-len(s))
						if strings.HasPrefix(name, "over-limit") {
							s += strings.Repeat(" ", 4096)
						}
					}
					body.Reader = strings.NewReader(s)
					if name == "read-error" {
						body.Reader = failingReader{}
					}
					status := 200
					if name == "over-limit-http" {
						status = 503
					}
					return &http.Response{StatusCode: status, Body: body}, nil
				}))
				var out any
				var result any = &out
				if name == "result-type" {
					var s struct{ N int }
					result = &s
				}
				err := invoke(c, raw, context.Background(), "probe", nil, result)
				if want == "" {
					if err != nil {
						t.Errorf("exact limit rejected: %v", err)
					}
				} else {
					status := 200
					if name == "transport-error" || name == "nil-response" {
						status = 0
					}
					if name == "over-limit-http" {
						status = 503
					}
					errorFields(t, kind(t, err, want), "probe", status, 0, false)
				}
				if err != nil && (strings.Contains(err.Error(), "sensitive") || errors.Unwrap(err) != nil) {
					t.Error("unsafe transport/body cause retained")
				}
				if calls != 1 {
					t.Errorf("Do count=%d, want 1", calls)
				}
				if name != "nil-response" && name != "nil-body" && name != "transport-error" && body.closes != 1 {
					t.Errorf("Close count=%d, want 1", body.closes)
				}
				if body.reads > 129 {
					t.Errorf("read %d bytes beyond limit+1", body.reads)
				}
			})
		}
	}
}
