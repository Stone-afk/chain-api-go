package bnb

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type fakeCaller struct {
	calls  int
	method string
	reply  string
	err    error
	params []byte
	ctx    context.Context
}

func (f *fakeCaller) Call(ctx context.Context, m string, params any, result any) error {
	f.ctx = ctx
	var e error
	f.params, e = json.Marshal(params)
	if e != nil {
		return e
	}
	f.calls++
	f.method = m
	if f.err != nil {
		return f.err
	}
	return json.Unmarshal([]byte(f.reply), result)
}
func chainID(n int64) big.Int { return *big.NewInt(n) }
func kind(t *testing.T, e error, want apierror.Kind) {
	t.Helper()
	var a apierror.Error
	if !errors.As(e, &a) || a.Kind != want {
		t.Fatalf("error=%v want=%s", e, want)
	}
}

func TestNewLocalValidation(t *testing.T) {
	var typedNil *fakeCaller
	for _, tc := range []struct {
		name  string
		id    big.Int
		rpc   jsonrpc.Caller
		valid bool
	}{
		{"valid", chainID(56), &fakeCaller{}, true}, {"nil", chainID(56), nil, false},
		{"typed-nil", chainID(56), typedNil, true}, {"zero", chainID(0), &fakeCaller{}, false}, {"negative", chainID(-1), &fakeCaller{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, e := New(Config{ExpectedChainID: tc.id}, tc.rpc)
			if tc.valid {
				if e != nil || c == nil {
					t.Fatalf("valid constructor unavailable: %v", e)
				}
			} else {
				kind(t, e, apierror.Configuration)
				if c != nil {
					t.Error("invalid constructor returned usable client")
				}
			}
			if f, ok := tc.rpc.(*fakeCaller); ok && f != nil && f.calls != 0 {
				t.Fatal("constructor performed I/O")
			}
		})
	}
}

func TestValidateNetwork(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		id          int64
		rpcErr      error
		want        apierror.Kind
	}{
		{"match", `"0x38"`, 56, nil, ""}, {"mismatch", `"0x1"`, 56, nil, apierror.NetworkMismatch},
		{"nonstandard-positive", `"0x539"`, 1337, nil, ""},
		{"rpc-rejection", "", 56, apierror.Error{Kind: apierror.RPC}, apierror.RPC},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{reply: tc.reply, err: tc.rpcErr}
			expected := chainID(tc.id)
			c, e := New(Config{ExpectedChainID: expected}, f)
			if e != nil {
				t.Fatalf("constructor unavailable: %v", e)
			}
			expected.SetBit(&expected, 10, 1)
			e = c.ValidateNetwork(context.Background())
			if tc.want == "" {
				if e != nil {
					t.Fatal(e)
				}
			} else {
				kind(t, e, tc.want)
			}
			if f.calls != 1 || f.method != "eth_chainId" {
				t.Errorf("calls=%d method=%s", f.calls, f.method)
			}
		})
	}
}

type nativeFacade interface {
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
	SendRawTransaction(context.Context, string) (common.Hash, error)
}

func TestNativeEVMFacade(t *testing.T) {
	f := &fakeCaller{reply: `"0x10000000000000001"`}
	c, err := New(Config{ExpectedChainID: chainID(56)}, f)
	if err != nil {
		t.Fatal(err)
	}
	if f.calls != 0 {
		t.Fatal("NS-02: constructor called Caller")
	}
	n, ok := any(c).(nativeFacade)
	if !ok {
		t.Fatal("NS-01: concrete facade lacks native EVM signatures")
	}
	v, err := n.BalanceAt(context.Background(), common.Address{}, nil)
	if err != nil || v.String() != "18446744073709551617" {
		t.Fatalf("native balance=%v error=%v", v, err)
	}
	if f.calls != 1 || f.method != "eth_getBalance" {
		t.Fatal("implicit validation or retry")
	}
}
func TestNativeEVMFacadeRawError(t *testing.T) {
	leaf := errors.New("SYNTHETIC_CAUSE")
	source := apierror.Error{Kind: apierror.RPC, HasRPCCode: true, RPCCode: 0, Message: "  SYNTHETIC_NODE\n", Data: json.RawMessage(` { "n":9007199254740993 } `), ResponseBody: []byte("SYNTHETIC_BODY"), ResponseBodyTruncated: true, Cause: leaf}
	f := &fakeCaller{err: source}
	c, err := New(Config{ExpectedChainID: chainID(56)}, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.BlockNumber(context.Background())
	var got apierror.Error
	if !errors.As(err, &got) || !errors.Is(err, leaf) {
		t.Fatal("NS-07: raw error/Cause unreachable")
	}
	if got.Kind != source.Kind || got.HasRPCCode != source.HasRPCCode || got.RPCCode != source.RPCCode || got.Message != source.Message || !reflect.DeepEqual(got.Data, source.Data) || !reflect.DeepEqual(got.ResponseBody, source.ResponseBody) || !got.ResponseBodyTruncated {
		t.Fatal("NS-07: original error fields changed")
	}
	if f.calls != 1 {
		t.Fatal("automatic validation or retry")
	}
}
