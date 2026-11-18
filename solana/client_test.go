package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

// Synthetic identifiers and pre-serialized bytes, never constructed or signed transactions.
var testKey = ns.PublicKey{1}
var testHash = ns.Hash{2}
var testSignature = ns.Signature{3}

const payload = "AQIDBA=="

type nativeContract interface {
	GetGenesisHash(context.Context) (ns.Hash, error)
	GetSlotWithOpts(context.Context, *srpc.GetSlotOpts) (uint64, error)
	GetBlockWithOpts(context.Context, uint64, *srpc.GetBlockOpts) (*srpc.GetBlockResult, error)
	GetBalanceWithOpts(context.Context, ns.PublicKey, *srpc.GetBalanceOpts) (*srpc.GetBalanceResult, error)
	GetAccountInfoWithOpts(context.Context, ns.PublicKey, *srpc.GetAccountInfoOpts) (*srpc.GetAccountInfoResult, error)
	GetLatestBlockhashWithOpts(context.Context, *srpc.GetLatestBlockhashOpts) (*srpc.GetLatestBlockhashResult, error)
	GetSignatureStatuses(context.Context, bool, ...ns.Signature) (*srpc.GetSignatureStatusesResult, error)
	GetTransaction(context.Context, ns.Signature, *srpc.GetTransactionOpts) (*srpc.GetTransactionResult, error)
	GetFeeForMessageWithOpts(context.Context, string, *srpc.GetFeeForMessageOpts) (*srpc.GetFeeForMessageResult, error)
	SimulateRawTransactionWithOpts(context.Context, []byte, *srpc.SimulateTransactionOpts) (*srpc.SimulateTransactionResponse, error)
	SendEncodedTransactionWithOpts(context.Context, string, srpc.TransactionOpts) (ns.Signature, error)
}
type fakeCaller struct {
	calls  int
	ctx    context.Context
	method string
	params []byte
	out    any
	reply  string
	err    error
}

func (f *fakeCaller) Call(ctx context.Context, method string, params, out any) error {
	f.calls++
	f.ctx = ctx
	f.method = method
	f.out = out
	var e error
	f.params, e = json.Marshal(params)
	if e != nil {
		return e
	}
	if f.err != nil {
		return f.err
	}
	if err := json.Unmarshal([]byte(f.reply), out); err != nil {
		return apierror.Error{Kind: apierror.Protocol, Method: method, Cause: err}
	}
	return nil
}
func client(t *testing.T, f *fakeCaller) *Client {
	t.Helper()
	c, e := New(f)
	if e != nil || c == nil {
		t.Fatalf("constructor: %v", e)
	}
	return c
}
func nativeClient(t *testing.T, f *fakeCaller) nativeContract {
	t.Helper()
	n, ok := any(client(t, f)).(nativeContract)
	if !ok {
		t.Fatal("NS-01/05: Client lacks frozen eleven-method native Solana contract")
	}
	return n
}
func same(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
func sameJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	decode := func(b []byte) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if e := d.Decode(&v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	same(t, decode(got), decode([]byte(want)))
}
func wire(t *testing.T, f *fakeCaller, ctx context.Context, method, params string) {
	t.Helper()
	same(t, f.calls, 1)
	same(t, f.ctx, ctx)
	same(t, f.method, method)
	sameJSON(t, f.params, params)
}
func classified(t *testing.T, err error, kind apierror.Kind, unknown bool) apierror.Error {
	t.Helper()
	var e apierror.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected apierror, got %v", err)
	}
	same(t, e.Kind, kind)
	same(t, e.SubmissionUnknown, unknown)
	return e
}

type rejectTransport struct{ calls *int }

func (r rejectTransport) RoundTrip(*http.Request) (*http.Response, error) {
	*r.calls++
	return nil, errors.New("SYNTHETIC forbidden default network")
}
func TestNativeSolanaConstructor(t *testing.T) {
	calls := 0
	old := http.DefaultTransport
	http.DefaultTransport = rejectTransport{&calls}
	t.Cleanup(func() { http.DefaultTransport = old })
	f := &fakeCaller{reply: "1"}
	c := client(t, f)
	same(t, f.calls, 0)
	same(t, calls, 0)
	n, ok := any(c).(nativeContract)
	if !ok {
		t.Fatal("NS-02: native injected client unavailable")
	}
	_, e := n.GetSlotWithOpts(t.Context(), nil)
	if e != nil {
		t.Fatal(e)
	}
	same(t, f.calls, 1)
	same(t, calls, 0)
}
