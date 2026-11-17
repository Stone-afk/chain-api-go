package evm

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

type rpcFake struct {
	calls    int
	method   string
	params   []byte
	ctx      context.Context
	reply    string
	err      error
	result   any
	rawReply bool
}

func (f *rpcFake) Call(ctx context.Context, method string, params, result any) error {
	f.calls++
	f.method = method
	f.ctx = ctx
	f.result = result
	var err error
	f.params, err = json.Marshal(params)
	if err != nil {
		return err
	}
	if f.err != nil {
		return f.err
	}
	if f.rawReply {
		*(result.(*json.RawMessage)) = append(json.RawMessage(nil), f.reply...)
		return nil
	}
	return json.Unmarshal([]byte(f.reply), result)
}
func intValue(s string) big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("bad test integer")
	}
	return *v
}
func assertKind(t *testing.T, err error, kind apierror.Kind) apierror.Error {
	t.Helper()
	var got apierror.Error
	if !errors.As(err, &got) || got.Kind != kind {
		t.Fatalf("error=%v; want %s", err, kind)
	}
	return got
}
func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
}
func assertWire(t *testing.T, f *rpcFake, method, params string, ctx context.Context) {
	t.Helper()
	var got, want any
	if err := json.Unmarshal(f.params, &got); err != nil {
		t.Fatalf("no valid request: %v (calls=%d)", err, f.calls)
	}
	if err := json.Unmarshal([]byte(params), &want); err != nil {
		t.Fatal(err)
	}
	equal(t, got, want)
	equal(t, f.calls, 1)
	equal(t, f.method, method)
	equal(t, f.ctx, ctx)
}

const addrJSON = `"0x0000000000000000000000000000000000000000"`
const hashJSON = `"0x0000000000000000000000000000000000000000000000000000000000000000"`

const H1 = `"0x0000000000000000000000000000000000000000000000000000000000000001"`
const H2 = `"0x0000000000000000000000000000000000000000000000000000000000000002"`
const H3 = `"0x0000000000000000000000000000000000000000000000000000000000000003"`
const A1 = `"0x0000000000000000000000000000000000000001"`
const A2 = `"0x0000000000000000000000000000000000000002"`
const logJSON = `{"address":` + A1 + `,"topics":[` + H2 + `,` + H3 + `],"data":"0x00ff","removed":true,"blockHash":` + H2 + `,"transactionHash":` + H1 + `,"blockNumber":"0x10","transactionIndex":"0x2","logIndex":"0x3"}`
const blockJSON = `{"number":"0x10","hash":` + H1 + `,"parentHash":` + H2 + `,"timestamp":"0x11","gasLimit":"0x12","gasUsed":"0x13","transactions":[` + H3 + `],"baseFeePerGas":"0x10000000000000001"}`
const transactionJSON = `{"hash":` + H1 + `,"from":` + A1 + `,"to":` + A2 + `,"nonce":"0x2","gas":"0x5208","value":"0x10000000000000001","input":"0x00ff","blockHash":` + H2 + `,"blockNumber":"0x10","transactionIndex":"0x3","gasPrice":"0x10000000000000002","maxFeePerGas":"0x10000000000000003","maxPriorityFeePerGas":"0x10000000000000004","chainId":"0x10000000000000005","type":"0x2"}`

var receiptJSON = `{"transactionHash":` + H1 + `,"blockHash":` + H2 + `,"transactionIndex":"0x3","blockNumber":"0x10","from":` + A1 + `,"to":` + A2 + `,"contractAddress":` + A1 + `,"cumulativeGasUsed":"0x5209","gasUsed":"0x5208","logs":[` + logJSON + `],"logsBloom":"0x` + strings.Repeat("00", 255) + `01","status":"0x1","effectiveGasPrice":"0x10000000000000006","type":"0x2"}`

func address(t *testing.T) Address {
	return rawString[Address](t, "0x0000000000000000000000000000000000000001")
}
func hash(t *testing.T) Hash {
	return rawString[Hash](t, "0x0000000000000000000000000000000000000000000000000000000000000001")
}
func query(t *testing.T, c *Client, name string) (any, bool, error) {
	ctx := context.Background()
	switch name {
	case "block":
		return c.GetBlock(ctx, Latest())
	case "pending":
		return c.GetBlock(ctx, Pending())
	case "transaction":
		return c.GetTransaction(ctx, hash(t))
	case "receipt":
		return c.GetTransactionReceipt(ctx, hash(t))
	case "logs":
		v, e := c.GetLogs(ctx, LogFilter{FromBlock: AtNumber(0), ToBlock: Latest()})
		return v, true, e
	}
	t.Fatalf("unknown test operation %s", name)
	return nil, false, nil
}
func edited(raw, key, value string) string {
	var m map[string]json.RawMessage
	if e := json.Unmarshal([]byte(raw), &m); e != nil {
		panic(e)
	}
	if value == "" {
		delete(m, key)
	} else {
		m[key] = json.RawMessage(value)
	}
	b, e := json.Marshal(m)
	if e != nil {
		panic(e)
	}
	return string(b)
}
func TestEVMNew(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rpc   jsonrpc.Caller
		valid bool
	}{{"nil", nil, false}, {"valid", &rpcFake{reply: `"0x1"`}, true}} {
		t.Run(tc.name, func(t *testing.T) {
			c, e := New(tc.rpc)
			if f, ok := tc.rpc.(*rpcFake); ok && f != nil {
				equal(t, f.calls, 0)
			}
			if !tc.valid {
				assertKind(t, e, apierror.Configuration)
				equal(t, c, (*Client)(nil))
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			v, e := c.ChainID(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			equal(t, v.String(), "1")
			equal(t, tc.rpc.(*rpcFake).calls, 1)
		})
	}
}

func TestEVMNewDoesNotInspectTypedNilCaller(t *testing.T) {
	var rpc *rpcFake
	c, err := New(rpc)
	if err != nil || c == nil {
		t.Fatalf("typed-nil caller was defensively rejected: client=%v error=%v", c, err)
	}
}
func TestQueryWireAndValues(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		method, params, reply string
		run                   func(*testing.T, *Client) (any, error)
		want                  any
	}{
		{"eth_chainId", `[]`, `"0x10000000000000001"`, func(t *testing.T, c *Client) (any, error) { v, e := c.ChainID(ctx); return v.String(), e }, "18446744073709551617"},
		{"eth_blockNumber", `[]`, `"0x10"`, func(t *testing.T, c *Client) (any, error) { return c.BlockNumber(ctx) }, uint64(16)},
		{"eth_getBalance", `[` + A1 + `,"latest"]`, `"0x10000000000000001"`, func(t *testing.T, c *Client) (any, error) {
			v, e := c.GetBalance(ctx, address(t), Latest())
			return v.String(), e
		}, "18446744073709551617"},
		{"eth_getTransactionCount", `[` + A1 + `,"pending"]`, `"0x0"`, func(t *testing.T, c *Client) (any, error) { return c.GetTransactionCount(ctx, address(t), Pending()) }, uint64(0)},
		{"eth_getCode", `[` + A1 + `,"0x0"]`, `"0x00ff"`, func(t *testing.T, c *Client) (any, error) {
			v, e := c.GetCode(ctx, address(t), AtNumber(0))
			return reflect.ValueOf(v).String(), e
		}, "0x00ff"},
	} {
		t.Run(tc.method, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			v, e := tc.run(t, &Client{rpc: f})
			if e != nil {
				t.Fatal(e)
			}
			equal(t, v, tc.want)
			assertWire(t, f, tc.method, tc.params, ctx)
		})
	}
}

func TestCompleteRawDTOs(t *testing.T) {
	for _, tc := range []struct {
		name, method, params, reply string
		fields                      map[string]any
	}{
		{"block", "eth_getBlockByNumber", `["latest",false]`, blockJSON, map[string]any{
			"Number": json.RawMessage(`"0x10"`), "Hash": json.RawMessage(H1), "ParentHash": H2[1 : len(H2)-1],
			"Timestamp": "0x11", "GasLimit": "0x12", "GasUsed": "0x13", "Transactions": []string{H3[1 : len(H3)-1]}, "BaseFeePerGas": json.RawMessage(`"0x10000000000000001"`),
		}},
		{"transaction", "eth_getTransactionByHash", "[" + H1 + "]", transactionJSON, map[string]any{
			"Hash": H1[1 : len(H1)-1], "From": A1[1 : len(A1)-1], "To": json.RawMessage(A2), "Nonce": "0x2", "Gas": "0x5208", "Value": "0x10000000000000001", "Input": "0x00ff",
			"BlockHash": json.RawMessage(H2), "BlockNumber": json.RawMessage(`"0x10"`), "TransactionIndex": json.RawMessage(`"0x3"`),
			"GasPrice": json.RawMessage(`"0x10000000000000002"`), "MaxFeePerGas": json.RawMessage(`"0x10000000000000003"`), "MaxPriorityFeePerGas": json.RawMessage(`"0x10000000000000004"`), "ChainID": json.RawMessage(`"0x10000000000000005"`), "Type": json.RawMessage(`"0x2"`),
		}},
		{"receipt", "eth_getTransactionReceipt", "[" + H1 + "]", receiptJSON, map[string]any{
			"TransactionHash": H1[1 : len(H1)-1], "BlockHash": H2[1 : len(H2)-1], "TransactionIndex": "0x3", "BlockNumber": "0x10", "From": A1[1 : len(A1)-1],
			"To": json.RawMessage(A2), "ContractAddress": json.RawMessage(A1), "CumulativeGasUsed": "0x5209", "GasUsed": "0x5208",
			"LogsBloom": "0x" + strings.Repeat("00", 255) + "01", "Status": json.RawMessage(`"0x1"`), "Root": json.RawMessage(nil), "EffectiveGasPrice": json.RawMessage(`"0x10000000000000006"`), "Type": json.RawMessage(`"0x2"`),
		}},
		{"logs", "eth_getLogs", `[{"fromBlock":"0x0","toBlock":"latest","topics":[]}]`, "[" + logJSON + "]", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			v, p, e := query(t, &Client{rpc: f}, tc.name)
			if e != nil {
				t.Fatal(e)
			}
			equal(t, p, true)
			assertWire(t, f, tc.method, tc.params, context.Background())
			if tc.name == "logs" {
				logs := v.([]Log)
				equal(t, len(logs), 1)
				assertLog(t, logs[0])
				return
			}
			if tc.name == "receipt" {
				logs := v.(Receipt).Logs
				equal(t, len(logs), 1)
				assertLog(t, logs[0])
				tc.fields["Logs"] = logs
			}
			assertFields(t, v, tc.fields)
		})
	}
}
func assertLog(t *testing.T, v Log) {
	t.Helper()
	assertFields(t, v, map[string]any{
		"Address": A1[1 : len(A1)-1], "Topics": []string{H2[1 : len(H2)-1], H3[1 : len(H3)-1]}, "Data": "0x00ff", "Removed": json.RawMessage("true"),
		"BlockHash": json.RawMessage(H2), "TransactionHash": json.RawMessage(H1), "BlockNumber": json.RawMessage(`"0x10"`), "TransactionIndex": json.RawMessage(`"0x2"`), "LogIndex": json.RawMessage(`"0x3"`),
	})
}

func TestRawPresenceAndPermissiveDecoding(t *testing.T) {
	for _, tc := range []struct {
		name, op, reply, field string
		want                   any
	}{
		{"pending-number", "pending", edited(blockJSON, "number", "null"), "Number", json.RawMessage("null")},
		{"missing-base-fee", "block", edited(blockJSON, "baseFeePerGas", ""), "BaseFeePerGas", json.RawMessage(nil)},
		{"creation-to", "transaction", edited(transactionJSON, "to", "null"), "To", json.RawMessage("null")},
		{"pending-block-hash", "transaction", edited(transactionJSON, "blockHash", "null"), "BlockHash", json.RawMessage("null")},
		{"failed-status", "receipt", edited(receiptJSON, "status", `"0x0"`), "Status", json.RawMessage(`"0x0"`)},
		{"historical-root", "receipt", edited(edited(receiptJSON, "status", ""), "root", H3), "Root", json.RawMessage(H3)},
		{"missing-plain-string", "block", edited(blockJSON, "gasUsed", ""), "GasUsed", ""},
		{"null-plain-string", "block", edited(blockJSON, "gasUsed", "null"), "GasUsed", ""},
		{"noncanonical-quantity", "block", edited(blockJSON, "gasUsed", `"0x00"`), "GasUsed", "0x00"},
		{"noncanonical-hash", "block", edited(blockJSON, "parentHash", `"0xAb"`), "ParentHash", "0xAb"},
		{"unknown-field", "block", edited(blockJSON, "future", `{"x":1}`), "GasUsed", "0x13"},
		{"duplicate-standard-last-wins", "block", strings.TrimSuffix(blockJSON, "}") + `,"gasUsed":"0x20"}`, "GasUsed", "0x20"},
		{"case-folded-key", "block", edited(edited(blockJSON, "gasUsed", ""), "GASUSED", `"0x21"`), "GasUsed", "0x21"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, p, e := query(t, &Client{rpc: &rpcFake{reply: tc.reply}}, tc.op)
			if e != nil {
				t.Fatalf("raw DTO must decode without schema validation: %v", e)
			}
			equal(t, p, true)
			field := reflect.ValueOf(v).FieldByName(tc.field)
			equal(t, field.Interface(), tc.want)
		})
	}
	for _, raw := range []string{"", "null", "false"} {
		t.Run("removed-"+raw, func(t *testing.T) {
			v, _, e := query(t, &Client{rpc: &rpcFake{reply: "[" + edited(logJSON, "removed", raw) + "]"}}, "logs")
			if e != nil {
				t.Fatal(e)
			}
			got := reflect.ValueOf(v.([]Log)[0]).FieldByName("Removed").Interface()
			var want json.RawMessage
			if raw != "" {
				want = json.RawMessage(raw)
			}
			equal(t, got, want)
		})
	}
}

func TestNullableThreeStates(t *testing.T) {
	for _, name := range []string{"block", "transaction", "receipt"} {
		for _, failed := range []bool{false, true} {
			t.Run(name+"/"+map[bool]string{false: "null", true: "rpc"}[failed], func(t *testing.T) {
				f := &rpcFake{reply: "null"}
				if failed {
					f.err = apierror.Error{Kind: apierror.RPC, RPCCode: -32000, HasRPCCode: true}
				}
				v, p, e := query(t, &Client{rpc: f}, name)
				equal(t, p, false)
				equal(t, reflect.ValueOf(v).IsZero(), true)
				if !failed {
					equal(t, e, nil)
				} else {
					a := assertKind(t, e, apierror.RPC)
					equal(t, a.RPCCode, -32000)
					equal(t, a.HasRPCCode, true)
				}
				equal(t, f.calls, 1)
			})
		}
	}
}

func TestScalarBigIntOwnership(t *testing.T) {
	for _, op := range []string{"chain", "balance", "gas-price"} {
		t.Run(op, func(t *testing.T) {
			f := &rpcFake{reply: `"0x10000000000000001"`}
			c := &Client{rpc: f}
			run := func() (big.Int, error) {
				switch op {
				case "chain":
					return c.ChainID(context.Background())
				case "balance":
					return c.GetBalance(context.Background(), address(t), Latest())
				default:
					return c.GasPrice(context.Background())
				}
			}
			first, e := run()
			if e != nil {
				t.Fatal(e)
			}
			second, e := run()
			if e != nil {
				t.Fatal(e)
			}
			first.SetBit(&first, 0, 0)
			equal(t, second.String(), "18446744073709551617")
			third, e := run()
			if e != nil {
				t.Fatal(e)
			}
			equal(t, third.String(), "18446744073709551617")
		})
	}
}
