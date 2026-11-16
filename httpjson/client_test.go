package httpjson_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/httpjson"
	"github.com/Stone-afk/chain-api-go/transport"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.Reader
	closes, reads int
}

func (b *trackedBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.reads += n
	return n, err
}
func (b *trackedBody) Close() error { b.closes++; return nil }

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

type badJSON struct{}

func (badJSON) MarshalJSON() ([]byte, error) { return nil, errors.New("request-secret-021") }

func config() httpjson.Config {
	return httpjson.Config{Endpoint: "https://node.invalid/prefix-secret-021", Headers: http.Header{"Authorization": {"Bearer header-secret-021"}}, MaxResponseBytes: 128}
}
func client(t *testing.T, cfg httpjson.Config, d transport.HTTPDoer) *httpjson.Client {
	t.Helper()
	c, err := httpjson.New(cfg, d)
	if err != nil {
		t.Errorf("valid constructor failed: %v", err)
	}
	if c == nil {
		t.Fatal("valid constructor returned nil client")
	}
	return c
}
func invoke(c *httpjson.Client, raw bool, ctx context.Context, path string, req, out any) error {
	if raw {
		return c.RawPost(ctx, path, req, out)
	}
	var p httpjson.Poster = c
	return p.Post(ctx, path, req, out)
}
func checkError(t *testing.T, err error, kind apierror.Kind, method string, status int, sentinel error) {
	t.Helper()
	var e apierror.Error
	if !errors.As(err, &e) {
		t.Errorf("want value apierror.Error %s, got %T", kind, err)
		return
	}
	if e.Kind != kind || e.Method != method || e.HTTPStatus != status || e.HasRPCCode || e.RPCCode != 0 || e.NodeCode != "" || e.SubmissionUnknown {
		t.Errorf("unexpected safe fields: %#v", e)
	}
	if errors.Unwrap(err) != sentinel {
		t.Error("cause must be exact safe sentinel or nil")
	}
	for _, s := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, s) != (s == sentinel) {
			t.Errorf("incorrect errors.Is(%v)", s)
		}
	}
	for _, secret := range []string{"node.invalid", "prefix-secret-021", "header-secret-021", "request-secret-021", "response-secret-021", "transport-secret-021"} {
		if strings.Contains(fmt.Sprintf("%#v %s", e, err), secret) {
			t.Errorf("error retained synthetic sensitive marker %q", secret)
		}
	}
}

func TestNewRejectsInvalidConfigurationWithoutIO(t *testing.T) {
	for _, name := range []string{"nil-doer", "typed-nil-doer", "empty-endpoint", "relative", "scheme", "missing-host", "bad-escape", "fragment", "zero-limit", "negative-limit", "header-name", "header-value"} {
		t.Run(name, func(t *testing.T) {
			cfg, calls := config(), 0
			var d transport.HTTPDoer = doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			switch name {
			case "nil-doer":
				d = nil
			case "typed-nil-doer":
				var p *http.Client
				d = p
			case "empty-endpoint":
				cfg.Endpoint = ""
			case "relative":
				cfg.Endpoint = "/prefix"
			case "scheme":
				cfg.Endpoint = "ftp://node.invalid"
			case "missing-host":
				cfg.Endpoint = "https:///prefix"
			case "bad-escape":
				cfg.Endpoint = "https://node.invalid/%zz"
			case "fragment":
				cfg.Endpoint += "#fragment"
			case "zero-limit":
				cfg.MaxResponseBytes = 0
			case "negative-limit":
				cfg.MaxResponseBytes = -1
			case "header-name":
				cfg.Headers["Bad Name"] = []string{"x"}
			case "header-value":
				cfg.Headers["X-Test"] = []string{"x\r\nAuthorization: secret"}
			}
			c, err := httpjson.New(cfg, d)
			checkError(t, err, apierror.Configuration, "", 0, nil)
			if c != nil || calls != 0 {
				t.Errorf("invalid constructor client=%v calls=%d", c != nil, calls)
			}
		})
	}
}

func TestPostRequestPrefixHeadersAndCopy(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, suffix := range []string{"", "/"} {
			for _, path := range []string{"wallet/probe", "/wallet/probe"} {
				t.Run(fmt.Sprintf("raw=%v/suffix=%q/path=%q", raw, suffix, path), func(t *testing.T) {
					cfg, calls := config(), 0
					cfg.Endpoint += suffix
					cfg.Headers["X-Multi"] = []string{"one", "two"}
					ctx := context.WithValue(context.Background(), struct{}{}, "caller")
					bodies := []*trackedBody{}
					c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Method != "POST" || r.URL.Scheme != "https" || r.URL.Host != "node.invalid" || r.URL.Path != "/prefix-secret-021/wallet/probe" || r.URL.RawQuery != "" || r.URL.Fragment != "" {
							t.Error("POST or same-origin prefix contract violated")
						}
						if r.Context() != ctx || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Authorization") != "Bearer header-secret-021" || !reflect.DeepEqual(r.Header.Values("X-Multi"), []string{"one", "two"}) {
							t.Error("context or copied headers changed")
						}
						data, err := io.ReadAll(r.Body)
						if err != nil || !json.Valid(data) || string(data) != `{"payload":"request-secret-021"}` {
							t.Error("request JSON changed")
						}
						r.Header["X-Multi"][0] = "doer mutation"
						b := &trackedBody{Reader: strings.NewReader(`{"n":7}`)}
						bodies = append(bodies, b)
						return &http.Response{StatusCode: 200, Body: b}, nil
					}))
					if calls != 0 {
						t.Error("constructor performed I/O")
					}
					cfg.Headers["X-Multi"][0] = "caller mutation"
					cfg.Headers.Set("Authorization", "changed")
					cfg.Endpoint = "https://other.invalid"
					for i := 0; i < 2; i++ {
						var out struct {
							N int `json:"n"`
						}
						if err := invoke(c, raw, ctx, path, map[string]string{"payload": "request-secret-021"}, &out); err != nil || out.N != 7 {
							t.Errorf("Post result=%d err=%v", out.N, err)
						}
					}
					if calls != 2 {
						t.Errorf("Do count=%d, want exactly one per call", calls)
					}
					for _, b := range bodies {
						if b.closes != 1 {
							t.Error("success body not closed exactly once")
						}
					}
				})
			}
		}
	}
}

func TestInvalidArgumentsAndUnsafePathsDoNoIO(t *testing.T) {
	paths := []string{"", " ", "https://other.invalid/wallet", "https://node.invalid/wallet", "//other.invalid/wallet", ".", "..", "wallet/../probe", "wallet/./probe", "%2e%2e/probe", "wallet/%2E%2E/probe", "wallet%2f..%2fprobe", "%252e%252e/probe", "wallet?token=request-secret-021", "wallet#fragment", "wallet\\..\\probe"}
	paths = append(paths, "%25252e%25252e/probe", "wallet%25252f%25252e%25252e%25252fprobe", "wallet%2f.%252e%252fprobe", "wallet/%2525252e%2525252e/probe", "wallet%252f..%2fprobe")
	for _, raw := range []bool{false, true} {
		for _, path := range paths {
			t.Run(fmt.Sprintf("raw=%v/path=%q", raw, path), func(t *testing.T) {
				calls := 0
				c := client(t, config(), doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
				out := 19
				checkError(t, invoke(c, raw, context.Background(), path, nil, &out), apierror.InvalidArgument, "", 0, nil)
				if calls != 0 || out != 19 {
					t.Error("unsafe path performed I/O or modified result")
				}
			})
		}
		for _, name := range []string{"nil-context", "nil-result", "typed-nil-result", "value-result", "unencodable", "marshal-error"} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, name), func(t *testing.T) {
				calls, out := 0, 19
				c := client(t, config(), doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
				ctx := context.Background()
				var result any = &out
				var req any
				switch name {
				case "nil-context":
					ctx = nil
				case "nil-result":
					result = nil
				case "typed-nil-result":
					var p *int
					result = p
				case "value-result":
					result = out
				case "unencodable":
					req = make(chan int)
				case "marshal-error":
					req = badJSON{}
				}
				checkError(t, invoke(c, raw, ctx, "wallet/probe", req, result), apierror.InvalidArgument, "wallet/probe", 0, nil)
				if calls != 0 || out != 19 {
					t.Error("invalid argument performed I/O or modified result")
				}
			})
		}
	}
}

func TestResponseBoundariesAndResources(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, name := range []string{"ok", "exact-limit", "over-limit", "empty", "malformed", "trailing", "decode", "http", "http-over-limit", "http-malformed", "read", "transport", "response-and-error", "nil-response", "nil-body", "typed-nil-body"} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, name), func(t *testing.T) {
				cfg, calls, status := config(), 0, 200
				payload, want := `{"n":7,"s":"ok"}`, apierror.Protocol
				switch name {
				case "exact-limit":
					cfg.MaxResponseBytes = int64(len(payload))
				case "over-limit":
					cfg.MaxResponseBytes = 8
				case "empty":
					payload = ""
				case "malformed":
					payload = `{"n":7`
				case "trailing":
					payload += ` {"secret":"response-secret-021"}`
				case "decode":
					payload = `{"n":7,"s":{}}`
				case "http", "http-over-limit":
					status = 503
					want = apierror.HTTP
					payload = `{"message":"response-secret-021"}`
					if name == "http-over-limit" {
						cfg.MaxResponseBytes = 8
					}
				case "http-malformed":
					status = 401
					want = apierror.HTTP
					payload = "response-secret-021"
				case "read", "transport", "response-and-error", "nil-response":
					want = apierror.Transport
				}
				b := &trackedBody{Reader: strings.NewReader(payload)}
				if name == "read" {
					b.Reader = readerFunc(func([]byte) (int, error) { return 0, errors.New("transport-secret-021") })
				}
				obtained := false
				c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					data, err := io.ReadAll(r.Body)
					if err != nil || string(data) != "null" {
						t.Error("nil request must encode JSON null")
					}
					if name == "transport" {
						return nil, errors.New("transport-secret-021")
					}
					if name == "nil-response" {
						return nil, nil
					}
					resp := &http.Response{StatusCode: status, Body: b}
					if name == "nil-body" {
						resp.Body = nil
						return resp, nil
					}
					if name == "typed-nil-body" {
						var p *trackedBody
						resp.Body = p
						return resp, nil
					}
					obtained = true
					if name == "response-and-error" {
						return resp, errors.New("transport-secret-021")
					}
					return resp, nil
				}))
				out := struct {
					N int    `json:"n"`
					S string `json:"s"`
				}{19, "unchanged"}
				before := out
				err := invoke(c, raw, context.Background(), "wallet/probe", nil, &out)
				if name == "ok" || name == "exact-limit" {
					if err != nil || out.N != 7 || out.S != "ok" {
						t.Errorf("valid response: result=%v err=%v", out, err)
					}
				} else {
					wantStatus := status
					if name == "transport" || name == "nil-response" || name == "response-and-error" {
						wantStatus = 0
					}
					checkError(t, err, want, "wallet/probe", wantStatus, nil)
					if out != before {
						t.Error("failure partially modified caller result")
					}
				}
				if calls != 1 {
					t.Errorf("Do count=%d, want 1", calls)
				}
				if obtained && b.closes != 1 {
					t.Error("obtained body not closed exactly once")
				}
				if b.reads > int(cfg.MaxResponseBytes)+1 {
					t.Error("response read beyond limit plus detection byte")
				}
			})
		}
	}
}

func TestContextSentinelsAndBodyClosure(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded, errors.Join(context.DeadlineExceeded, context.Canceled)} {
			for _, stage := range []string{"before", "do", "read", "after-read"} {
				t.Run(fmt.Sprintf("raw=%v/%v/%s", raw, sentinel, stage), func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					if stage == "before" && sentinel == context.DeadlineExceeded {
						ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
						defer cancel()
					}
					if stage == "before" {
						cancel()
					}
					calls := 0
					b := &trackedBody{Reader: readerFunc(func(p []byte) (int, error) {
						if stage == "after-read" {
							cancel()
							return copy(p, `{"n":7}`), io.EOF
						}
						return 0, fmt.Errorf("transport-secret-021: %w", sentinel)
					})}
					c := client(t, config(), doerFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Context() != ctx {
							t.Error("caller context not propagated")
						}
						if stage == "do" {
							return &http.Response{StatusCode: 200, Body: b}, fmt.Errorf("transport-secret-021: %w", sentinel)
						}
						return &http.Response{StatusCode: 200, Body: b}, nil
					}))
					out := map[string]int{"unchanged": 19}
					err := invoke(c, raw, ctx, "wallet/probe", nil, &out)
					want, status, wantCalls := sentinel, 200, 1
					if errors.Is(sentinel, context.Canceled) {
						want = context.Canceled
					}
					if stage == "before" {
						status = 0
						wantCalls = 0
					}
					if stage == "do" {
						status = 0
					}
					if stage == "after-read" {
						want = context.Canceled
					}
					checkError(t, err, apierror.Transport, "wallet/probe", status, want)
					if calls != wantCalls || b.closes != wantCalls {
						t.Errorf("Do/Close=%d/%d want %d", calls, b.closes, wantCalls)
					}
					if !reflect.DeepEqual(out, map[string]int{"unchanged": 19}) {
						t.Error("context failure mutated result")
					}
				})
			}
		}
	}
}
