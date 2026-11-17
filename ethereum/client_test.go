package ethereum

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
		{"valid", chainID(1), &fakeCaller{}, true}, {"nil", chainID(1), nil, false},
		{"typed-nil", chainID(1), typedNil, true}, {"zero", chainID(0), &fakeCaller{}, false}, {"negative", chainID(-1), &fakeCaller{}, false},
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
		{"match", `"0x1"`, 1, nil, ""}, {"mismatch", `"0x38"`, 1, nil, apierror.NetworkMismatch},
		{"nonstandard-positive", `"0x539"`, 1337, nil, ""},
		{"rpc-rejection", "", 1, apierror.Error{Kind: apierror.RPC}, apierror.RPC},
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

func rawString[T any](t *testing.T, s string) T {
	t.Helper()
	var value T
	v := reflect.ValueOf(&value).Elem()
	if v.Kind() != reflect.String {
		t.Fatalf("raw API: %s kind = %s; want string", v.Type(), v.Kind())
	}
	v.SetString(s)
	return value
}
func TestConcreteCapabilities(t *testing.T) {
	const a = "0x0000000000000000000000000000000000000001"
	const h = "0x0000000000000000000000000000000000000000000000000000000000000003"
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, reply, params string
		run                         func(*testing.T, *Client) (any, error)
		want                        any
	}{
		{"chain", "eth_chainId", `"0x1"`, `[]`, func(t *testing.T, c *Client) (any, error) { v, e := c.ChainID(ctx); return v.String(), e }, "1"},
		{"height", "eth_blockNumber", `"0x10"`, `[]`, func(t *testing.T, c *Client) (any, error) { return c.BlockNumber(ctx) }, uint64(16)},
		{"block", "eth_getBlockByNumber", `null`, `["pending",false]`, func(t *testing.T, c *Client) (any, error) {
			v, p, e := c.GetBlock(ctx, evm.Pending())
			return []any{v, p}, e
		}, []any{evm.Block{}, false}},
		{"balance", "eth_getBalance", `"0x10000000000000001"`, `["` + a + `","latest"]`, func(t *testing.T, c *Client) (any, error) {
			v, e := c.GetBalance(ctx, rawString[evm.Address](t, a), evm.Latest())
			return v.String(), e
		}, "18446744073709551617"},
		{"nonce", "eth_getTransactionCount", `"0x2"`, `["` + a + `","pending"]`, func(t *testing.T, c *Client) (any, error) {
			return c.GetTransactionCount(ctx, rawString[evm.Address](t, a), evm.Pending())
		}, uint64(2)},
		{"code", "eth_getCode", `"0xAB"`, `["` + a + `","0x0"]`, func(t *testing.T, c *Client) (any, error) {
			v, e := c.GetCode(ctx, rawString[evm.Address](t, a), evm.AtNumber(0))
			return reflect.ValueOf(v).Interface(), e
		}, "0xAB"},
		{"transaction", "eth_getTransactionByHash", `null`, `["` + h + `"]`, func(t *testing.T, c *Client) (any, error) {
			v, p, e := c.GetTransaction(ctx, rawString[evm.Hash](t, h))
			return []any{v, p}, e
		}, []any{evm.Transaction{}, false}},
		{"receipt", "eth_getTransactionReceipt", `null`, `["` + h + `"]`, func(t *testing.T, c *Client) (any, error) {
			v, p, e := c.GetTransactionReceipt(ctx, rawString[evm.Hash](t, h))
			return []any{v, p}, e
		}, []any{evm.Receipt{}, false}},
		{"logs", "eth_getLogs", `[]`, `[{"fromBlock":"latest","toBlock":"pending","topics":[]}]`, func(t *testing.T, c *Client) (any, error) {
			return c.GetLogs(ctx, evm.LogFilter{FromBlock: evm.Latest(), ToBlock: evm.Pending()})
		}, []evm.Log{}},
		{"call", "eth_call", `"0xCD"`, `[{},"latest"]`, func(t *testing.T, c *Client) (any, error) { return c.Call(ctx, evm.CallRequest{}, evm.Latest()) }, "0xCD"},
		{"estimate", "eth_estimateGas", `"0x5208"`, `[{}]`, func(t *testing.T, c *Client) (any, error) { return c.EstimateGas(ctx, evm.CallRequest{}) }, uint64(21000)},
		{"price", "eth_gasPrice", `"0x2"`, `[]`, func(t *testing.T, c *Client) (any, error) { v, e := c.GasPrice(ctx); return v.String(), e }, "2"},
		{"fee", "eth_feeHistory", `{"oldestBlock":"0x1","baseFeePerGas":["0x1","0x2"],"gasUsedRatio":[0.5]}`, `["0x1","latest",[]]`, func(t *testing.T, c *Client) (any, error) { return c.FeeHistory(ctx, 1, evm.Latest(), nil) }, nil},
		{"submit", "eth_sendRawTransaction", `"` + h + `"`, `["0x00AB"]`, func(t *testing.T, c *Client) (any, error) {
			return c.SendRawTransaction(ctx, rawString[evm.Data](t, "0x00AB"))
		}, h},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{reply: tc.reply}
			c, e := New(Config{ExpectedChainID: chainID(1)}, f)
			if e != nil {
				t.Fatal(e)
			}
			if f.calls != 0 {
				t.Fatal("constructor performed I/O")
			}
			got, e := tc.run(t, c)
			if e != nil {
				t.Fatal(e)
			}
			if tc.name == "fee" {
				v := reflect.ValueOf(got)
				for name, want := range map[string]any{"OldestBlock": "0x1", "BaseFeePerGas": []string{"0x1", "0x2"}, "GasUsedRatio": []float64{0.5}, "Reward": json.RawMessage(nil)} {
					if !reflect.DeepEqual(v.FieldByName(name).Interface(), want) {
						t.Errorf("FeeHistory.%s = %#v; want %#v", name, v.FieldByName(name).Interface(), want)
					}
				}
			} else if tc.name == "code" || tc.name == "call" || tc.name == "submit" {
				v := reflect.ValueOf(got)
				if v.Kind() != reflect.String || v.String() != tc.want {
					t.Errorf("raw result = %#v; want %s", got, tc.want)
				}
			} else if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v; want %#v", got, tc.want)
			}
			var params, want any
			if e = json.Unmarshal(f.params, &params); e != nil {
				t.Fatal(e)
			}
			if e = json.Unmarshal([]byte(tc.params), &want); e != nil {
				t.Fatal(e)
			}
			if f.calls != 1 || f.method != tc.method || f.ctx != ctx || !reflect.DeepEqual(params, want) {
				t.Errorf("wrong request: method=%s calls=%d params=%s", f.method, f.calls, f.params)
			}
		})
	}
}

func TestEmbeddedRawError(t *testing.T) {
	leaf := errors.New("SYNTHETIC_CAUSE")
	source := apierror.Error{Kind: apierror.RPC, HasRPCCode: true, RPCCode: 0, Message: "  SYNTHETIC_NODE \n", Data: json.RawMessage(`{"n":9007199254740993}`), Cause: leaf}
	f := &fakeCaller{err: source}
	c, err := New(Config{ExpectedChainID: chainID(1)}, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.BlockNumber(context.Background())
	var got apierror.Error
	if !errors.As(err, &got) {
		t.Fatal("missing structured error")
	}
	want := source
	want.Method = "eth_blockNumber"
	want.Cause = source
	if !reflect.DeepEqual(got, want) || !errors.Is(err, leaf) {
		t.Errorf("embedded client lost raw error: %#v", got)
	}
	if f.calls != 1 {
		t.Fatal("automatic validation or retry")
	}
}
