package evm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// This test-local interface keeps RED executable before the native migration.
type nativeReadContract interface {
	ChainID(context.Context) (*big.Int, error)
	BlockNumber(context.Context) (uint64, error)
	BlockByNumber(context.Context, *big.Int) (*types.Block, error)
	BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error)
	NonceAt(context.Context, common.Address, *big.Int) (uint64, error)
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	FilterLogs(context.Context, geth.FilterQuery) ([]types.Log, error)
	CallContract(context.Context, geth.CallMsg, *big.Int) ([]byte, error)
	EstimateGas(context.Context, geth.CallMsg) (uint64, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
	FeeHistory(context.Context, uint64, *big.Int, []float64) (*geth.FeeHistory, error)
}

// Also pin the dependency's approved signatures and module graph before T038.
var _ nativeReadContract = (*ethclient.Client)(nil)

type nativeContract interface {
	nativeReadContract
	SendRawTransaction(context.Context, string) (common.Hash, error)
}

type nativeCall struct {
	method string
	params json.RawMessage
	ctx    context.Context
}
type rpcFake struct {
	calls   []nativeCall
	reply   string
	err     error
	respond func(int, string, json.RawMessage) (string, error)
}

func (f *rpcFake) Call(ctx context.Context, method string, params, out any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, nativeCall{method, raw, ctx})
	reply := f.reply
	if f.respond != nil {
		reply, err = f.respond(len(f.calls), method, raw)
	} else {
		err = f.err
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(reply), out)
}
func nativeClient(t *testing.T, f *rpcFake) nativeContract {
	t.Helper()
	c, err := New(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 0 {
		t.Fatal("NS-02: constructor called Caller")
	}
	n, ok := any(c).(nativeContract)
	if !ok {
		t.Fatal("NS-01: Client does not implement plan 10.3 native EVM signatures")
	}
	return n
}
func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
}
func assertKind(t *testing.T, err error, kind apierror.Kind) apierror.Error {
	t.Helper()
	var a apierror.Error
	if !errors.As(err, &a) || a.Kind != kind {
		t.Fatalf("error=%v; want %s", err, kind)
	}
	return a
}
func assertWire(t *testing.T, f *rpcFake, method, params string, ctx context.Context) {
	t.Helper()
	equal(t, len(f.calls), 1)
	equal(t, f.calls[0].method, method)
	equal(t, f.calls[0].ctx, ctx)
	var got, want any
	if err := json.Unmarshal(f.calls[0].params, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(params), &want); err != nil {
		t.Fatal(err)
	}
	// geth omits params for zero-argument methods; an empty list is equivalent.
	if params == "[]" && got == nil {
		got = []any{}
	}
	equal(t, got, want)
}

const h1 = "0x0000000000000000000000000000000000000000000000000000000000000001"
const a1 = "0x0000000000000000000000000000000000000001"

func TestNativeEVMConstructor(t *testing.T) {
	f := &rpcFake{}
	c, err := New(f)
	if err != nil || c == nil {
		t.Fatalf("constructor: %v", err)
	}
	equal(t, len(f.calls), 0)
	c, err = New(nil)
	assertKind(t, err, apierror.Configuration)
	equal(t, c, (*Client)(nil))
}
func TestNativeEVMMethodContract(t *testing.T) { nativeClient(t, &rpcFake{}) }

func TestNativeEVMRestrictedSurface(t *testing.T) {
	c, err := New(&rpcFake{})
	if err != nil {
		t.Fatal(err)
	}
	typ := reflect.TypeOf(c)
	for _, name := range []string{"Client", "RPC", "Close", "SendTransaction", "GetBlock", "GetBalance", "GetTransactionCount", "GetCode", "GetTransaction", "GetTransactionReceipt", "GetLogs", "Call", "GasPrice"} {
		if _, exists := typ.MethodByName(name); exists {
			t.Errorf("NS-01: forbidden/legacy method %s is exposed", name)
		}
	}
}
