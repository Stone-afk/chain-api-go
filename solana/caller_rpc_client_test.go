package solana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	sjsonrpc "github.com/solana-foundation/solana-go/v2/rpc/jsonrpc"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

// Controller-approved test-only introspection avoids a production test seam and
// missing-symbol compilation RED. Read the fixed upstream pointer without modifying it.
func privateNative(t *testing.T, c *Client) *srpc.Client {
	t.Helper()
	v := reflect.ValueOf(c).Elem()
	target := reflect.TypeOf((*srpc.Client)(nil))
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Type() != target {
			continue
		}
		if !f.CanAddr() || f.IsNil() {
			t.Fatal("NS-07: private native client not initialized")
		}
		return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Interface().(*srpc.Client)
	}
	t.Fatal("NS-07: Client lacks private native rpc.Client")
	return nil
}
func TestNativeSolanaCallerRPCClientUnsupported(t *testing.T) {
	f := &fakeCaller{}
	n := privateNative(t, client(t, f))
	callbacks, network := 0, 0
	old := http.DefaultTransport
	http.DefaultTransport = rejectTransport{&network}
	t.Cleanup(func() { http.DefaultTransport = old })
	e := n.RPCCallWithCallback(t.Context(), "synthetic_callback", []any{"SYNTHETIC_REQUEST"}, func(*http.Request, *http.Response) error { callbacks++; return nil })
	classified(t, e, apierror.Unsupported, false)
	responses, e := n.RPCCallBatch(t.Context(), sjsonrpc.RPCRequests{&sjsonrpc.RPCRequest{Method: "synthetic_batch", Params: []any{"SYNTHETIC_REQUEST"}, ID: 1, JSONRPC: "2.0"}})
	classified(t, e, apierror.Unsupported, false)
	same(t, len(responses), 0)
	same(t, f.calls, 0)
	same(t, callbacks, 0)
	same(t, network, 0)
}
func TestNativeSolanaCallerRPCClientForwardsIdentity(t *testing.T) {
	f := &fakeCaller{reply: `{"n":7}`}
	n := privateNative(t, client(t, f))
	ctx := context.WithValue(t.Context(), struct{}{}, "identity")
	out := map[string]int{}
	e := n.RPCCallForInto(ctx, &out, "synthetic_method", []any{uint64(9007199254740993)})
	if e != nil {
		t.Fatal(e)
	}
	wire(t, f, ctx, "synthetic_method", `[9007199254740993]`)
	same(t, out, map[string]int{"n": 7})
	if f.out != &out {
		t.Fatal("CallForInto changed output pointer")
	}
}

type taggedError struct{ cause error }

func (e taggedError) Error() string { return "SYNTHETIC_TYPED_DEPENDENCY" }
func (e taggedError) Unwrap() error { return e.cause }
func TestNativeSolanaRawErrorsAndIsolation(t *testing.T) {
	leaf := taggedError{errors.Join(context.Canceled, context.DeadlineExceeded)}
	for _, tc := range methodCases() {
		for _, kind := range []apierror.Kind{apierror.RPC, apierror.HTTP, apierror.Protocol, apierror.Transport, apierror.Unsupported, apierror.Kind("future"), ""} {
			t.Run(tc.name+"/"+string(kind), func(t *testing.T) {
				original := apierror.Error{Kind: kind, Method: "dependency", HTTPStatus: 503, RPCCode: 0, HasRPCCode: true, NodeCode: "SYNTHETIC_NODE", Message: "  SYNTHETIC_MESSAGE\n", Data: json.RawMessage(` { "n":9007199254740993 } `), ResponseBody: []byte("SYNTHETIC_BODY\n"), ResponseBodyTruncated: true, Cause: leaf}
				var dependency error = original
				if kind == "" {
					dependency = leaf
				}
				wrapped := fmt.Errorf("SYNTHETIC_WRAPPER: %w", dependency)
				f := &fakeCaller{err: wrapped}
				c := nativeClient(t, f)
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				_, e := tc.call(c, ctx)
				if !errors.Is(e, wrapped) || !errors.Is(e, context.Canceled) || !errors.Is(e, context.DeadlineExceeded) {
					t.Fatalf("complete wrapper/context Cause lost: %v", e)
				}
				var typed taggedError
				if !errors.As(e, &typed) || typed != leaf {
					t.Fatal("typed leaf lost")
				}
				same(t, f.calls, 1)
				same(t, f.ctx, ctx)
				same(t, f.method, tc.name)
				if kind != "" {
					var a apierror.Error
					if !errors.As(e, &a) {
						t.Fatal("raw apierror lost")
					}
					same(t, a.Kind, kind)
					same(t, a.HTTPStatus, original.HTTPStatus)
					same(t, a.RPCCode, original.RPCCode)
					same(t, a.HasRPCCode, true)
					same(t, a.NodeCode, original.NodeCode)
					same(t, a.Message, original.Message)
					same(t, a.Data, original.Data)
					same(t, a.ResponseBody, original.ResponseBody)
					same(t, a.ResponseBodyTruncated, true)
					if tc.name == "sendTransaction" {
						same(t, a.SubmissionUnknown, kind != apierror.RPC)
						same(t, a.Method, "sendTransaction")
					}
				} else if tc.name == "sendTransaction" {
					classified(t, e, apierror.Transport, true)
				}
				encoded, _ := json.Marshal(e)
				returned := fmt.Sprintf("%v %+v %#v %s", e, e, e, encoded)
				for _, marker := range []string{testKey.String(), testSignature.String(), payload} {
					if strings.Contains(returned, marker) {
						t.Fatal("request-only key/signature/payload appended to error")
					}
				}
			})
		}
	}
}
