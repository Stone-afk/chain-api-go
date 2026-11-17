package httpjson_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type badJSON struct{ cause error }

func (b badJSON) MarshalJSON() ([]byte, error) { return nil, b.cause }

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
func checkError(t *testing.T, err error, kind apierror.Kind, method string, status int, cause error) {
	t.Helper()
	var e apierror.Error
	if !errors.As(err, &e) {
		t.Errorf("want value apierror.Error %s, got %T", kind, err)
		return
	}
	if e.Kind != kind || e.Method != method || e.HTTPStatus != status || e.HasRPCCode || e.RPCCode != 0 || e.NodeCode != "" || e.SubmissionUnknown {
		t.Errorf("unexpected classification fields: %#v", e)
	}
	if errors.Unwrap(err) != cause {
		t.Error("cause must be the exact original error or nil")
	}
	for _, s := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, s) != errors.Is(cause, s) {
			t.Errorf("incorrect errors.Is(%v)", s)
		}
	}

}

func TestNewRejectsInvalidConfigurationWithoutIO(t *testing.T) {
	for _, name := range []string{"nil-doer", "empty-endpoint", "relative", "scheme", "missing-host", "bad-escape", "fragment", "zero-limit", "negative-limit", "header-name", "header-value"} {
		t.Run(name, func(t *testing.T) {
			cfg, calls := config(), 0
			var d transport.HTTPDoer = doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			switch name {
			case "nil-doer":
				d = nil
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
			var cause error
			if name == "bad-escape" {
				cause = errors.Unwrap(err)
				var parsed *url.Error
				if !errors.As(err, &parsed) || !errors.Is(err, url.EscapeError("%zz")) {
					t.Error("original URL parse error chain missing")
				}
			}
			checkError(t, err, apierror.Configuration, "", 0, cause)
			if c != nil || calls != 0 {
				t.Errorf("invalid constructor client=%v calls=%d", c != nil, calls)
			}
		})
	}
}

func TestNewDoesNotInspectTypedNilDoer(t *testing.T) {
	var doer *http.Client
	c, err := httpjson.New(config(), doer)
	if err != nil || c == nil {
		t.Fatalf("typed-nil doer was defensively rejected: client=%v error=%v", c, err)
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
				original := errors.New("request-secret-021")
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
					req = badJSON{cause: original}
				}
				err := invoke(c, raw, ctx, "wallet/probe", req, result)
				var cause error
				switch name {
				case "unencodable":
					cause = errors.Unwrap(err)
					var unsupported *json.UnsupportedTypeError
					if !errors.As(err, &unsupported) || unsupported.Type != reflect.TypeOf(req) {
						t.Error("original unsupported-type marshal error missing")
					}
				case "marshal-error":
					cause = errors.Unwrap(err)
					var marshaler *json.MarshalerError
					if !errors.As(err, &marshaler) || marshaler.Err != original || !errors.Is(err, original) {
						t.Error("original marshal sentinel identity missing")
					}
				}
				checkError(t, err, apierror.InvalidArgument, "wallet/probe", 0, cause)
				if calls != 0 || out != 19 {
					t.Error("invalid argument performed I/O or modified result")
				}
			})
		}
	}
}

func TestResponseBoundariesAndResources(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, name := range []string{"ok", "exact-limit", "over-limit", "empty", "malformed", "trailing", "decode", "http", "http-over-limit", "http-malformed", "read", "transport", "response-and-error", "nil-response", "nil-body"} {
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
				original := errors.New("transport-secret-021")
				b := &trackedBody{Reader: strings.NewReader(payload)}
				if name == "read" {
					b.Reader = readerFunc(func([]byte) (int, error) { return 0, original })
				}
				obtained := false
				c := client(t, cfg, doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					data, err := io.ReadAll(r.Body)
					if err != nil || string(data) != "null" {
						t.Error("nil request must encode JSON null")
					}
					if name == "transport" {
						return nil, original
					}
					if name == "nil-response" {
						return nil, nil
					}
					resp := &http.Response{StatusCode: status, Body: b}
					if name == "nil-body" {
						resp.Body = nil
						return resp, nil
					}
					obtained = true
					if name == "response-and-error" {
						return resp, original
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
					if name == "transport" || name == "nil-response" {
						wantStatus = 0
					}
					var cause error
					if name == "read" || name == "transport" || name == "response-and-error" {
						cause = original
					}
					switch name {
					case "empty", "malformed", "trailing":
						cause = errors.Unwrap(err)
						var syntax *json.SyntaxError
						if !errors.As(err, &syntax) {
							t.Error("original JSON syntax error missing")
						}
					case "decode":
						cause = errors.Unwrap(err)
						var mismatch *json.UnmarshalTypeError
						if !errors.As(err, &mismatch) || mismatch.Field != "s" || mismatch.Type != reflect.TypeOf("") {
							t.Error("original JSON type error missing")
						}
					}
					checkError(t, err, want, "wallet/probe", wantStatus, cause)
					switch name {
					case "empty", "malformed", "trailing", "decode", "over-limit":
						var e apierror.Error
						if errors.As(err, &e) {
							body := payload
							if name == "over-limit" {
								body = body[:cfg.MaxResponseBytes]
							}
							if string(e.ResponseBody) != body || e.ResponseBodyTruncated != (name == "over-limit") {
								t.Error("Protocol response body/truncation not retained")
							}
						}
					}
					if name == "response-and-error" && b.reads != 0 {
						t.Error("failed Do response must not be read for diagnostics")
					}
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
					original := fmt.Errorf("transport-secret-021: %w", sentinel)
					b := &trackedBody{Reader: readerFunc(func(p []byte) (int, error) {
						if stage == "after-read" {
							cancel()
							return copy(p, `{"n":7}`), io.EOF
						}
						return 0, original
					})}
					c := client(t, config(), doerFunc(func(r *http.Request) (*http.Response, error) {
						calls++
						if r.Context() != ctx {
							t.Error("caller context not propagated")
						}
						if stage == "do" {
							return &http.Response{StatusCode: 200, Body: b}, original
						}
						return &http.Response{StatusCode: 200, Body: b}, nil
					}))
					out := map[string]int{"unchanged": 19}
					err := invoke(c, raw, ctx, "wallet/probe", nil, &out)
					want, status, wantCalls := original, 200, 1
					if stage == "before" {
						want = ctx.Err()
						status = 0
						wantCalls = 0
					}
					if stage == "do" && b.reads != 0 {
						t.Error("failed Do response must not be read for diagnostics")
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

// All markers in these fixtures are synthetic; no real endpoint is contacted.
func TestRawResponseBodyAndLimits(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, tc := range []struct {
			name, payload        string
			limit                int64
			readError, truncated bool
		}{
			{"non-json", "response-secret-021", 128, false, false},
			{"exact-limit", "12345678", 8, false, false},
			{"limit-plus-one", "123456789", 8, false, true},
			{"partial-read", "partial", 8, true, false},
			{"read-error-only", "", 8, true, false},
			{"exact-limit-read-error", "12345678", 8, true, false},
			{"probe-byte-and-error", "123456789", 8, true, true},
		} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, tc.name), func(t *testing.T) {
				cfg := config()
				cfg.MaxResponseBytes = tc.limit
				original := fmt.Errorf("synthetic read: %w", context.Canceled)
				b := &trackedBody{Reader: strings.NewReader(tc.payload)}
				if tc.readError {
					b.Reader = readerFunc(func(p []byte) (int, error) { return copy(p, tc.payload), original })
				}
				if tc.name == "probe-byte-and-error" {
					first := true
					b.Reader = readerFunc(func(p []byte) (int, error) {
						if first {
							first = false
							return copy(p, tc.payload[:tc.limit]), nil
						}
						return copy(p, tc.payload[tc.limit:]), original
					})
				}
				calls := 0
				c := client(t, cfg, doerFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: 503, Body: b}, nil
				}))
				out := "unchanged"
				err := invoke(c, raw, context.Background(), "wallet/probe", map[string]string{"payload": "request-secret-021"}, &out)
				kind := apierror.HTTP
				var cause error
				if tc.readError {
					kind, cause = apierror.Transport, original
				}
				checkError(t, err, kind, "wallet/probe", 503, cause)
				var e apierror.Error
				if !errors.As(err, &e) {
					t.Fatal("missing structured error")
				}
				want := tc.payload
				if int64(len(want)) > tc.limit {
					want = want[:tc.limit]
				}
				if string(e.ResponseBody) != want {
					t.Error("original response bytes not retained exactly")
				}
				if e.ResponseBodyTruncated != tc.truncated {
					t.Errorf("truncation=%v want %v", e.ResponseBodyTruncated, tc.truncated)
				}
				for _, marker := range []string{"node.invalid", "prefix-secret-021", "header-secret-021", "request-secret-021"} {
					fields := e.Message + string(e.Data) + string(e.ResponseBody) + e.Error()
					if e.Cause != nil {
						fields += e.Cause.Error()
					}
					if strings.Contains(fields, marker) {
						t.Errorf("SDK appended absent configuration/request marker %q", marker)
					}
				}
				if calls != 1 || b.closes != 1 || b.reads > int(tc.limit)+1 || out != "unchanged" {
					t.Error("request count, close, bounded read or result ownership violated")
				}
			})
		}
	}
}

type dependencyError struct{ message string }

func (e dependencyError) Error() string { return e.message }

func TestOriginalTypedCauseDespiteCanceledContext(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, stage := range []string{"do", "read"} {
			t.Run(fmt.Sprintf("raw=%v/%s", raw, stage), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				original := fmt.Errorf("synthetic wrapper: %w", dependencyError{"transport-secret-021"})
				calls := 0
				b := &trackedBody{Reader: readerFunc(func([]byte) (int, error) { cancel(); return 0, original })}
				c := client(t, config(), doerFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if stage == "do" {
						cancel()
						return &http.Response{StatusCode: 200, Body: b}, original
					}
					return &http.Response{StatusCode: 200, Body: b}, nil
				}))
				out := 19
				err := invoke(c, raw, ctx, "wallet/probe", nil, &out)
				status := 200
				if stage == "do" && b.reads != 0 {
					t.Error("failed Do response must not be read for diagnostics")
				}
				checkError(t, err, apierror.Transport, "wallet/probe", status, original)
				var typed dependencyError
				if !errors.Is(err, original) || !errors.As(err, &typed) || typed.message != "transport-secret-021" {
					t.Error("original typed dependency error is not reachable")
				}
				if calls != 1 || b.closes != 1 || out != 19 {
					t.Error("request count, body closure or result ownership violated")
				}
			})
		}
	}
}
