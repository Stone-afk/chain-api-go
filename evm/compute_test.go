package evm

import (
	"context"
	"math"
	"math/big"
	"strconv"

	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func TestComputeWireAndValues(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, params, reply string
		run                         func(*Client) (any, error)
		want                        any
	}{
		{"call", "eth_call", `[{"to":` + addrJSON + `,"data":"0x00"},"pending"]`, `"0x0001"`, func(c *Client) (any, error) { return c.Call(ctx, CallRequest{HasTo: true, Data: Data{0}}, Pending()) }, Data{0, 1}},
		{"estimate", "eth_estimateGas", `[{"data":"0x"}]`, `"0x5208"`, func(c *Client) (any, error) { return c.EstimateGas(ctx, CallRequest{}) }, uint64(21000)},
		{"gas-price", "eth_gasPrice", `[]`, `"0x10000000000000001"`, func(c *Client) (any, error) { v, e := c.GasPrice(ctx); return v.String(), e }, "18446744073709551617"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			c := &Client{rpc: f}
			got, e := tc.run(c)
			if e != nil {
				t.Fatalf("missing compute behavior: %v", e)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v want %v", got, tc.want)
			}
			assertWire(t, f, tc.method, tc.params, ctx)
		})
	}
}

func TestCallRequestPresenceAndValidation(t *testing.T) {
	for _, operation := range []string{"call", "estimate"} {
		for _, tc := range []struct {
			name    string
			req     CallRequest
			want    string
			invalid bool
		}{
			{"absent-values-ignored", CallRequest{Value: intValue("-1"), GasPrice: intValue("-2"), MaxFeePerGas: intValue("-3"), MaxPriorityFeePerGas: intValue("-4")}, `{"data":"0x"}`, false},
			{"explicit-zero", CallRequest{HasFrom: true, HasTo: true, HasGas: true, HasValue: true, HasGasPrice: true}, `{"from":` + addrJSON + `,"to":` + addrJSON + `,"gas":"0x0","value":"0x0","gasPrice":"0x0","data":"0x"}`, false},
			{"dynamic-zero", CallRequest{HasMaxFeePerGas: true, HasMaxPriorityFeePerGas: true}, `{"maxFeePerGas":"0x0","maxPriorityFeePerGas":"0x0","data":"0x"}`, false},
			{"legacy-dynamic-conflict", CallRequest{HasGasPrice: true, HasMaxFeePerGas: true}, "", true},
			{"legacy-priority-conflict", CallRequest{HasGasPrice: true, HasMaxPriorityFeePerGas: true}, "", true},
			{"negative-value", CallRequest{Value: intValue("-1"), HasValue: true}, "", true},
			{"negative-price", CallRequest{GasPrice: intValue("-1"), HasGasPrice: true}, "", true},
			{"negative-max-fee", CallRequest{MaxFeePerGas: intValue("-1"), HasMaxFeePerGas: true}, "", true},
			{"negative-tip", CallRequest{MaxPriorityFeePerGas: intValue("-1"), HasMaxPriorityFeePerGas: true}, "", true},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				f := &rpcFake{reply: `"0x"`}
				if operation == "estimate" {
					f.reply = `"0x0"`
				}
				c := &Client{rpc: f}
				var err error
				if operation == "call" {
					_, err = c.Call(context.Background(), tc.req, Latest())
				} else {
					_, err = c.EstimateGas(context.Background(), tc.req)
				}
				if tc.invalid {
					assertKind(t, err, apierror.InvalidArgument)
					if f.calls != 0 {
						t.Error("invalid request reached Caller")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				method, params := "eth_call", "["+tc.want+`,"latest"]`
				if operation == "estimate" {
					method, params = "eth_estimateGas", "["+tc.want+"]"
				}
				assertWire(t, f, method, params, context.Background())
			})
		}
	}
}

func TestComputeInputOwnership(t *testing.T) {
	for _, op := range []string{"call", "estimate"} {
		for _, dynamic := range []bool{false, true} {
			t.Run(op+"/"+strconv.FormatBool(dynamic), func(t *testing.T) {
				req := CallRequest{Value: intValue("18446744073709551617"), HasValue: true, GasPrice: intValue("18446744073709551618"), HasGasPrice: !dynamic, MaxFeePerGas: intValue("18446744073709551619"), HasMaxFeePerGas: dynamic, MaxPriorityFeePerGas: intValue("18446744073709551620"), HasMaxPriorityFeePerGas: dynamic, Data: Data{0, 255}}
				f := &rpcFake{reply: `"0x"`}
				c := &Client{rpc: f}
				var e error
				if op == "call" {
					_, e = c.Call(context.Background(), req, Latest())
				} else {
					f.reply = `"0x0"`
					_, e = c.EstimateGas(context.Background(), req)
				}
				if e != nil {
					t.Fatal(e)
				}
				equal(t, []string{req.Value.String(), req.GasPrice.String(), req.MaxFeePerGas.String(), req.MaxPriorityFeePerGas.String()}, []string{"18446744073709551617", "18446744073709551618", "18446744073709551619", "18446744073709551620"})
				equal(t, req.Data, Data{0, 255})
			})
		}
	}
}

const feeJSON = `{"oldestBlock":"0xf","baseFeePerGas":["0x10000000000000001","0x10000000000000002","0x10000000000000003"],"gasUsedRatio":[0.5,1],"reward":[["0x10000000000000004","0x10000000000000005"],["0x10000000000000006","0x10000000000000007"]]}`

func TestFeeHistoryShapes(t *testing.T) {
	want := FeeHistoryResult{OldestBlock: 15, BaseFeePerGas: []big.Int{intValue("18446744073709551617"), intValue("18446744073709551618"), intValue("18446744073709551619")}, GasUsedRatio: []float64{0.5, 1}, Reward: [][]big.Int{{intValue("18446744073709551620"), intValue("18446744073709551621")}, {intValue("18446744073709551622"), intValue("18446744073709551623")}}, HasReward: true}
	for _, count := range []uint64{2, 3} {
		t.Run("count-"+strconv.Itoa(int(count)), func(t *testing.T) {
			f := &rpcFake{reply: feeJSON}
			v, e := (&Client{rpc: f}).FeeHistory(context.Background(), count, Latest(), []float64{25, 25})
			if e != nil {
				t.Fatal(e)
			}
			equal(t, v, want)
			if count == 2 {
				assertWire(t, f, "eth_feeHistory", `["0x2","latest",[25,25]]`, context.Background())
			}
		})
	}
	for _, reward := range []string{"", `[]`, `[[],[]]`} {
		t.Run("reward-"+reward, func(t *testing.T) {
			f := &rpcFake{reply: edited(feeJSON, "reward", reward)}
			v, e := (&Client{rpc: f}).FeeHistory(context.Background(), 2, Latest(), nil)
			if e != nil {
				t.Fatal(e)
			}
			equal(t, v.HasReward, reward != "")
			assertWire(t, f, "eth_feeHistory", `["0x2","latest",[]]`, context.Background())
		})
	}
	for _, tc := range []struct{ key, bad string }{
		{"oldestBlock", ""}, {"oldestBlock", `"0x00"`}, {"baseFeePerGas", "null"}, {"baseFeePerGas", `["0x1"]`}, {"baseFeePerGas", `["0x1","0x00","0x3"]`},
		{"gasUsedRatio", `[]`}, {"gasUsedRatio", `[0,0,0]`}, {"gasUsedRatio", `[-0.1,0]`}, {"gasUsedRatio", `[0,1.1]`}, {"gasUsedRatio", `[0,1e999]`}, {"gasUsedRatio", "null"},
		{"reward", ""}, {"reward", "null"}, {"reward", `[["0x1","0x2"]]`}, {"reward", `[["0x1"],["0x2"]]`}, {"reward", `[["0x1","0x2"],["0x3","0x00"]]`},
	} {
		t.Run(tc.key+"/"+tc.bad, func(t *testing.T) {
			v, e := (&Client{rpc: &rpcFake{reply: edited(feeJSON, tc.key, tc.bad)}}).FeeHistory(context.Background(), 2, Latest(), []float64{25, 75})
			assertKind(t, e, apierror.Protocol)
			equal(t, reflect.ValueOf(v).IsZero(), true)
		})
	}
}
func TestFeeHistoryValidation(t *testing.T) {
	for i, p := range [][]float64{{-1}, {101}, {75, 25}, {math.NaN()}, {math.Inf(1)}, {math.Inf(-1)}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := &rpcFake{}
			_, e := (&Client{rpc: f}).FeeHistory(context.Background(), 1, Latest(), p)
			assertKind(t, e, apierror.InvalidArgument)
			equal(t, f.calls, 0)
		})
	}
	f := &rpcFake{}
	_, e := (&Client{rpc: f}).FeeHistory(context.Background(), 0, Latest(), nil)
	assertKind(t, e, apierror.InvalidArgument)
	equal(t, f.calls, 0)
}

func TestIndependentResponseBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []any
		bad  string
	}{
		{"ChainID", nil, `"0x00"`}, {"BlockNumber", nil, `"0x10000000000000000"`}, {"GetBalance", []any{a1, Latest()}, `"0x00"`}, {"GetTransactionCount", []any{a1, Latest()}, `"0x10000000000000000"`},
		{"GetCode", []any{a1, Latest()}, `"0x0"`}, {"Call", []any{CallRequest{}, Latest()}, `"0x0"`}, {"EstimateGas", []any{CallRequest{}}, `"0x10000000000000000"`}, {"GasPrice", nil, `"0x00"`},
		{"GetBlock", []any{Latest()}, `[]`}, {"GetTransaction", []any{h1}, `[]`}, {"GetTransactionReceipt", []any{h1}, `[]`}, {"GetLogs", []any{LogFilter{FromBlock: Latest(), ToBlock: Latest()}}, `[null]`}, {"FeeHistory", []any{uint64(1), Latest(), []float64{}}, `[]`}, {"SendRawTransaction", []any{Data{1}}, `"0x01"`},
	} {
		for _, mode := range []string{"nil-context", "invalid-response", "null-response"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				if mode == "null-response" && (tc.name == "GetBlock" || tc.name == "GetTransaction" || tc.name == "GetTransactionReceipt") {
					return
				}
				f := &rpcFake{reply: tc.bad}
				ctx := reflect.ValueOf(context.Background())
				want := apierror.Protocol
				if mode == "null-response" {
					f.reply = "null"
				}
				if mode == "nil-context" {
					ctx = reflect.Zero(reflect.TypeOf((*context.Context)(nil)).Elem())
					want = apierror.InvalidArgument
				}
				args := []reflect.Value{ctx}
				for _, a := range tc.args {
					args = append(args, reflect.ValueOf(a))
				}
				values := reflect.ValueOf(&Client{rpc: f}).MethodByName(tc.name).Call(args)
				e, _ := values[len(values)-1].Interface().(error)
				assertKind(t, e, want)
				for _, v := range values[:len(values)-1] {
					equal(t, v.IsZero(), true)
				}
				if mode == "nil-context" {
					equal(t, f.calls, 0)
				}
			})
		}
	}
}
func TestInvalidBlockAndTopicInputs(t *testing.T) {
	for _, run := range []func(*Client) error{
		func(c *Client) error { _, _, e := c.GetBlock(context.Background(), BlockRef{}); return e },
		func(c *Client) error { _, e := c.GetBalance(context.Background(), a1, BlockRef{}); return e },
		func(c *Client) error { _, e := c.GetTransactionCount(context.Background(), a1, BlockRef{}); return e },
		func(c *Client) error { _, e := c.GetCode(context.Background(), a1, BlockRef{}); return e },
		func(c *Client) error { _, e := c.Call(context.Background(), CallRequest{}, BlockRef{}); return e },
		func(c *Client) error { _, e := c.FeeHistory(context.Background(), 1, BlockRef{}, nil); return e },
		func(c *Client) error {
			_, e := c.GetLogs(context.Background(), LogFilter{FromBlock: Latest(), ToBlock: Latest(), Topics: make([][]Hash, 5)})
			return e
		},
	} {
		f := &rpcFake{}
		assertKind(t, run(&Client{rpc: f}), apierror.InvalidArgument)
		equal(t, f.calls, 0)
	}
}
