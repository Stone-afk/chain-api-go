package evm

import (
	"context"
	"encoding/json"
	"github.com/Stone-afk/chain-api-go/apierror"
	"math"
	"reflect"
	"strconv"
	"testing"
)

func callRequest(t *testing.T, fields map[string]string) CallRequest {
	t.Helper()
	var req CallRequest
	v := reflect.ValueOf(&req).Elem()
	for name, s := range fields {
		f := v.FieldByName(name)
		if !f.IsValid() || f.Kind() != reflect.String {
			t.Fatalf("CallRequest.%s must be a raw string", name)
		}
		f.SetString(s)
	}
	return req
}

func TestCallRequestWireAndValidation(t *testing.T) {
	for _, op := range []string{"call", "estimate"} {
		for _, tc := range []struct {
			name    string
			fields  map[string]string
			want    string
			invalid bool
		}{
			{"omitted", nil, `{}`, false},
			{"explicit-zero", map[string]string{"From": A1[1 : len(A1)-1], "To": A2[1 : len(A2)-1], "Gas": "0x0", "Value": "0x0", "GasPrice": "0x0", "Data": "0x"}, `{"from":` + A1 + `,"to":` + A2 + `,"gas":"0x0","value":"0x0","gasPrice":"0x0","data":"0x"}`, false},
			{"dynamic", map[string]string{"MaxFeePerGas": "0xAb", "MaxPriorityFeePerGas": "0x0", "Data": "0xABcd"}, `{"maxFeePerGas":"0xAb","maxPriorityFeePerGas":"0x0","data":"0xABcd"}`, false},
			{"legacy-dynamic-conflict", map[string]string{"GasPrice": "0x0", "MaxFeePerGas": "0x0"}, "", true},
			{"legacy-priority-conflict", map[string]string{"GasPrice": "0x0", "MaxPriorityFeePerGas": "0x0"}, "", true},
			{"bad-from", map[string]string{"From": "0x01"}, "", true},
			{"bad-to", map[string]string{"To": "0xgg"}, "", true},
			{"bad-gas", map[string]string{"Gas": "0x00"}, "", true},
			{"bad-value", map[string]string{"Value": "-1"}, "", true},
			{"bad-price", map[string]string{"GasPrice": "0x"}, "", true},
			{"bad-max-fee", map[string]string{"MaxFeePerGas": "0xgg"}, "", true},
			{"bad-tip", map[string]string{"MaxPriorityFeePerGas": "1"}, "", true},
			{"odd-data", map[string]string{"Data": "0x0"}, "", true},
			{"bad-data", map[string]string{"Data": "0xgg"}, "", true},
		} {
			t.Run(op+"/"+tc.name, func(t *testing.T) {
				req := callRequest(t, tc.fields)
				before := req
				f := &rpcFake{reply: `"0xABcd"`}
				c := &Client{rpc: f}
				ctx := context.Background()
				var e error
				if op == "call" {
					var v Data
					v, e = c.Call(ctx, req, Pending())
					if !tc.invalid && e == nil {
						equal(t, v, rawString[Data](t, "0xABcd"))
					}
				} else {
					f.reply = `"0x5208"`
					var v uint64
					v, e = c.EstimateGas(ctx, req)
					if !tc.invalid && e == nil {
						equal(t, v, uint64(21000))
					}
				}
				equal(t, req, before)
				if tc.invalid {
					assertKind(t, e, apierror.InvalidArgument)
					equal(t, f.calls, 0)
					return
				}
				if e != nil {
					t.Fatal(e)
				}
				method, params := "eth_call", "["+tc.want+`,"pending"]`
				if op == "estimate" {
					method, params = "eth_estimateGas", "["+tc.want+"]"
				}
				assertWire(t, f, method, params, ctx)
			})
		}
	}
}

const feeJSON = `{"oldestBlock":"0xf","baseFeePerGas":["0x10000000000000001","0x10000000000000002","0x10000000000000003"],"gasUsedRatio":[0.5,1],"reward":[["0x10000000000000004","0x10000000000000005"],["0x10000000000000006","0x10000000000000007"]]}`

func TestFeeHistoryRawValues(t *testing.T) {
	f := &rpcFake{reply: feeJSON}
	p := []float64{25, 75}
	v, e := (&Client{rpc: f}).FeeHistory(context.Background(), 2, Latest(), p)
	if e != nil {
		t.Fatal(e)
	}
	assertFields(t, v, map[string]any{
		"OldestBlock": "0xf", "BaseFeePerGas": []string{"0x10000000000000001", "0x10000000000000002", "0x10000000000000003"},
		"GasUsedRatio": []float64{0.5, 1}, "Reward": json.RawMessage(`[["0x10000000000000004","0x10000000000000005"],["0x10000000000000006","0x10000000000000007"]]`),
	})
	assertWire(t, f, "eth_feeHistory", `["0x2","latest",[25,75]]`, context.Background())
	equal(t, p, []float64{25, 75})
}

func TestDataResultsRemainRaw(t *testing.T) {
	for _, op := range []string{"code", "call"} {
		t.Run(op, func(t *testing.T) {
			v, _, e := operation(t, &Client{rpc: &rpcFake{reply: `"0x0"`}}, op)
			if e != nil {
				t.Fatalf("raw Data result must not undergo nested hex validation: %v", e)
			}
			equal(t, v, rawString[Data](t, "0x0"))
		})
	}
}
func TestFeeHistoryPermissiveAndPresence(t *testing.T) {
	for _, reward := range []string{"", "null", "[]", `[["0x00"]]`} {
		t.Run("reward-"+reward, func(t *testing.T) {
			raw := edited(`{"oldestBlock":"0x00","baseFeePerGas":["0x01"],"gasUsedRatio":[1.2,-0.1]}`, "reward", reward)
			f := &rpcFake{reply: raw}
			v, e := (&Client{rpc: f}).FeeHistory(context.Background(), 3, Latest(), nil)
			if e != nil {
				t.Fatalf("raw fee history must not enforce dimensions or nested quantities: %v", e)
			}
			var want json.RawMessage
			if reward != "" {
				want = json.RawMessage(reward)
			}
			assertFields(t, v, map[string]any{"OldestBlock": "0x00", "BaseFeePerGas": []string{"0x01"}, "GasUsedRatio": []float64{1.2, -0.1}, "Reward": want})
			assertWire(t, f, "eth_feeHistory", `["0x3","latest",[]]`, context.Background())
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

func operation(t *testing.T, c *Client, name string) (any, bool, error) {
	ctx := context.Background()
	switch name {
	case "chain":
		v, e := c.ChainID(ctx)
		return v, true, e
	case "height":
		v, e := c.BlockNumber(ctx)
		return v, true, e
	case "balance":
		v, e := c.GetBalance(ctx, address(t), Latest())
		return v, true, e
	case "nonce":
		v, e := c.GetTransactionCount(ctx, address(t), Latest())
		return v, true, e
	case "code":
		v, e := c.GetCode(ctx, address(t), Latest())
		return v, true, e
	case "call":
		v, e := c.Call(ctx, CallRequest{}, Latest())
		return v, true, e
	case "estimate":
		v, e := c.EstimateGas(ctx, CallRequest{})
		return v, true, e
	case "price":
		v, e := c.GasPrice(ctx)
		return v, true, e
	case "fee":
		v, e := c.FeeHistory(ctx, 1, Latest(), nil)
		return v, true, e
	case "submit":
		v, e := c.SendRawTransaction(ctx, rawString[Data](t, "0x01"))
		return v, true, e
	default:
		return query(t, c, name)
	}
}

func TestResultTypeAndNullBoundaries(t *testing.T) {
	for _, op := range []string{"chain", "height", "balance", "nonce", "code", "call", "estimate", "price", "fee", "submit", "block", "transaction", "receipt", "logs"} {
		for _, bad := range []string{"null", "false", `{"broken":`} {
			if bad == "null" && (op == "block" || op == "transaction" || op == "receipt") {
				continue
			}
			t.Run(op+"/"+bad, func(t *testing.T) {
				f := &rpcFake{reply: bad, rawReply: true}
				v, p, e := operation(t, &Client{rpc: f}, op)
				assertKind(t, e, apierror.Protocol)
				equal(t, reflect.ValueOf(v).IsZero(), true)
				if op == "block" || op == "transaction" || op == "receipt" {
					equal(t, p, false)
				}
				equal(t, f.calls, 1)
			})
		}
	}
}
func TestScalarQuantityValidation(t *testing.T) {
	for _, op := range []string{"chain", "height", "balance", "nonce", "estimate", "price"} {
		for _, bad := range []string{`"0x00"`, `"0x"`, `"1"`, `"0xgg"`} {
			t.Run(op+"/"+bad, func(t *testing.T) {
				v, _, e := operation(t, &Client{rpc: &rpcFake{reply: bad}}, op)
				assertKind(t, e, apierror.Protocol)
				equal(t, reflect.ValueOf(v).IsZero(), true)
			})
		}
	}
	for _, op := range []string{"height", "nonce", "estimate"} {
		t.Run(op+"/overflow", func(t *testing.T) {
			v, _, e := operation(t, &Client{rpc: &rpcFake{reply: `"0x10000000000000000"`}}, op)
			assertKind(t, e, apierror.Protocol)
			equal(t, reflect.ValueOf(v).IsZero(), true)
		})
	}
}
func TestRequestBoundaries(t *testing.T) {
	for _, op := range []string{"block", "balance", "nonce", "code", "call", "fee", "filter-from", "filter-to", "address", "hash", "topics", "filter-address", "filter-topic"} {
		t.Run(op, func(t *testing.T) {
			f := &rpcFake{}
			c := &Client{rpc: f}
			ctx := context.Background()
			var e error
			switch op {
			case "block":
				_, _, e = c.GetBlock(ctx, BlockRef{})
			case "balance":
				_, e = c.GetBalance(ctx, address(t), BlockRef{})
			case "nonce":
				_, e = c.GetTransactionCount(ctx, address(t), BlockRef{})
			case "code":
				_, e = c.GetCode(ctx, address(t), BlockRef{})
			case "call":
				_, e = c.Call(ctx, CallRequest{}, BlockRef{})
			case "fee":
				_, e = c.FeeHistory(ctx, 1, BlockRef{}, nil)
			case "address":
				_, e = c.GetBalance(ctx, rawString[Address](t, "0x01"), Latest())
			case "hash":
				_, _, e = c.GetTransaction(ctx, rawString[Hash](t, "0x01"))
			default:
				filter := LogFilter{FromBlock: Latest(), ToBlock: Latest()}
				if op == "filter-from" {
					filter.FromBlock = BlockRef{}
				}
				if op == "filter-to" {
					filter.ToBlock = BlockRef{}
				}
				if op == "topics" || op == "filter-topic" || op == "filter-address" {
					v := reflect.ValueOf(&filter).Elem()
					var name string
					var value any
					switch op {
					case "topics":
						name, value = "Topics", make([][]string, 5)
					case "filter-topic":
						name, value = "Topics", [][]string{{"0x01"}}
					default:
						name, value = "Addresses", []string{"0x01"}
					}
					setField(t, v, name, value)
				}
				_, e = c.GetLogs(ctx, filter)
			}
			assertKind(t, e, apierror.InvalidArgument)
			equal(t, f.calls, 0)
		})
	}
}
func setField(t *testing.T, v reflect.Value, name string, value any) {
	t.Helper()
	f := v.FieldByName(name)
	r := reflect.ValueOf(value)
	if !f.IsValid() || f.Type() != r.Type() {
		t.Fatalf("%s.%s must have type %s", v.Type(), name, r.Type())
	}
	f.Set(r)
}
func TestLogFilterORWildcardWire(t *testing.T) {
	for _, withAddress := range []bool{false, true} {
		t.Run(strconv.FormatBool(withAddress), func(t *testing.T) {
			req := LogFilter{FromBlock: AtNumber(0), ToBlock: Pending()}
			v := reflect.ValueOf(&req).Elem()
			setField(t, v, "Topics", [][]string{{H2[1 : len(H2)-1], H3[1 : len(H3)-1]}, nil, {}})
			extra := ""
			if withAddress {
				setField(t, v, "Addresses", []string{A1[1 : len(A1)-1]})
				extra = `,"address":[` + A1 + `]`
			}
			before, e := json.Marshal(req)
			if e != nil {
				t.Fatal(e)
			}
			f := &rpcFake{reply: "[]"}
			_, e = (&Client{rpc: f}).GetLogs(context.Background(), req)
			if e != nil {
				t.Fatal(e)
			}
			assertWire(t, f, "eth_getLogs", `[{"fromBlock":"0x0","toBlock":"pending","topics":[[`+H2+","+H3+`],null,null]`+extra+"}]", context.Background())
			after, e := json.Marshal(req)
			if e != nil {
				t.Fatal(e)
			}
			equal(t, string(after), string(before))
		})
	}
}
