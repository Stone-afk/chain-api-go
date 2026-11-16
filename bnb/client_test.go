package bnb

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/evm"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
)

type fakeCaller struct {
	calls  int
	method string
	reply  string
	err    error
}

func (f *fakeCaller) Call(_ context.Context, m string, _ any, result any) error {
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
		{"typed-nil", chainID(56), typedNil, false}, {"zero", chainID(0), &fakeCaller{}, false}, {"negative", chainID(-1), &fakeCaller{}, false},
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

func TestConcreteCapabilities(t *testing.T) {
	ctx := context.Background()
	a := evm.Address{19: 1}
	h := evm.Hash{31: 3}
	for _, tc := range []struct {
		name, rpc, reply string
		args             []any
		want             []any
	}{
		{"ChainID", "eth_chainId", `"0x38"`, nil, []any{chainID(56), nil}},
		{"BlockNumber", "eth_blockNumber", `"0x10"`, nil, []any{uint64(16), nil}},
		{"GetBlock", "eth_getBlockByNumber", `null`, []any{evm.Pending()}, []any{evm.Block{}, false, nil}},
		{"GetBalance", "eth_getBalance", `"0x1"`, []any{a, evm.Latest()}, []any{chainID(1), nil}},
		{"GetTransactionCount", "eth_getTransactionCount", `"0x2"`, []any{a, evm.Pending()}, []any{uint64(2), nil}},
		{"GetCode", "eth_getCode", `"0x00"`, []any{a, evm.Latest()}, []any{evm.Data{0}, nil}},
		{"GetTransaction", "eth_getTransactionByHash", `null`, []any{h}, []any{evm.Transaction{}, false, nil}},
		{"GetTransactionReceipt", "eth_getTransactionReceipt", `null`, []any{h}, []any{evm.Receipt{}, false, nil}},
		{"GetLogs", "eth_getLogs", `[]`, []any{evm.LogFilter{FromBlock: evm.Latest(), ToBlock: evm.Latest()}}, []any{[]evm.Log{}, nil}},
		{"Call", "eth_call", `"0x01"`, []any{evm.CallRequest{}, evm.Latest()}, []any{evm.Data{1}, nil}},
		{"EstimateGas", "eth_estimateGas", `"0x5208"`, []any{evm.CallRequest{}}, []any{uint64(21000), nil}},
		{"GasPrice", "eth_gasPrice", `"0x2"`, nil, []any{chainID(2), nil}},
		{"FeeHistory", "eth_feeHistory", `{"oldestBlock":"0x1","baseFeePerGas":["0x1","0x2"],"gasUsedRatio":[0.5]}`, []any{uint64(1), evm.Latest(), []float64{}}, []any{evm.FeeHistoryResult{OldestBlock: 1, BaseFeePerGas: []big.Int{chainID(1), chainID(2)}, GasUsedRatio: []float64{0.5}}, nil}},
		{"SendRawTransaction", "eth_sendRawTransaction", `"0x0000000000000000000000000000000000000000000000000000000000000003"`, []any{evm.Data{0, 1}}, []any{h, nil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{reply: tc.reply}
			c, e := New(Config{ExpectedChainID: chainID(56)}, f)
			if e != nil {
				t.Fatalf("concrete capability unavailable: %v", e)
			}
			args := []reflect.Value{reflect.ValueOf(ctx)}
			for _, a := range tc.args {
				args = append(args, reflect.ValueOf(a))
			}
			values := reflect.ValueOf(c).MethodByName(tc.name).Call(args)
			var got []any
			for _, v := range values {
				got = append(got, v.Interface())
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v want %#v", got, tc.want)
			}
			if f.calls != 1 || f.method != tc.rpc {
				t.Fatalf("calls=%d method=%s", f.calls, f.method)
			}
		})
	}
}
