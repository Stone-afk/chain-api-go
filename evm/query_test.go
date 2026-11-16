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
	calls  int
	method string
	params []byte
	ctx    context.Context
	reply  string
	err    error
	result any
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

var h1, h2, h3 = Hash{31: 1}, Hash{31: 2}, Hash{31: 3}
var a1, a2 = Address{19: 1}, Address{19: 2}

const H1 = `"0x0000000000000000000000000000000000000000000000000000000000000001"`
const H2 = `"0x0000000000000000000000000000000000000000000000000000000000000002"`
const H3 = `"0x0000000000000000000000000000000000000000000000000000000000000003"`
const A1 = `"0x0000000000000000000000000000000000000001"`
const A2 = `"0x0000000000000000000000000000000000000002"`
const logJSON = `{"address":` + A1 + `,"topics":[` + H2 + `,` + H3 + `],"data":"0x00ff","removed":true,"blockHash":` + H2 + `,"transactionHash":` + H1 + `,"blockNumber":"0x10","transactionIndex":"0x2","logIndex":"0x3"}`
const blockJSON = `{"number":"0x10","hash":` + H1 + `,"parentHash":` + H2 + `,"timestamp":"0x11","gasLimit":"0x12","gasUsed":"0x13","transactions":[` + H3 + `],"baseFeePerGas":"0x10000000000000001"}`
const transactionJSON = `{"hash":` + H1 + `,"from":` + A1 + `,"to":` + A2 + `,"nonce":"0x2","gas":"0x5208","value":"0x10000000000000001","input":"0x00ff","blockHash":` + H2 + `,"blockNumber":"0x10","transactionIndex":"0x3","gasPrice":"0x10000000000000002","maxFeePerGas":"0x10000000000000003","maxPriorityFeePerGas":"0x10000000000000004","chainId":"0x10000000000000005","type":"0x2"}`

var receiptJSON = `{"transactionHash":` + H1 + `,"blockHash":` + H2 + `,"transactionIndex":"0x3","blockNumber":"0x10","from":` + A1 + `,"to":` + A2 + `,"contractAddress":` + A1 + `,"cumulativeGasUsed":"0x5209","gasUsed":"0x5208","logs":[` + logJSON + `],"logsBloom":"0x` + strings.Repeat("00", 255) + `01","status":"0x1","effectiveGasPrice":"0x10000000000000006","type":"0x2"}`
var wantLog = Log{Address: a1, Topics: []Hash{h2, h3}, Data: Data{0, 255}, Removed: true, BlockHash: h2, TransactionHash: h1, HasBlockHash: true, HasTransactionHash: true, BlockNumber: 16, TransactionIndex: 2, LogIndex: 3, HasBlockNumber: true, HasTransactionIndex: true, HasLogIndex: true}
var wantBlock = Block{Number: 16, HasNumber: true, Hash: h1, HasHash: true, ParentHash: h2, Timestamp: 17, GasLimit: 18, GasUsed: 19, Transactions: []Hash{h3}, BaseFeePerGas: intValue("18446744073709551617"), HasBaseFeePerGas: true}
var wantTransaction = Transaction{Hash: h1, From: a1, To: a2, HasTo: true, Nonce: 2, Gas: 21000, Value: intValue("18446744073709551617"), Input: Data{0, 255}, BlockHash: h2, HasBlockHash: true, BlockNumber: 16, HasBlockNumber: true, TransactionIndex: 3, HasTransactionIndex: true, GasPrice: intValue("18446744073709551618"), MaxFeePerGas: intValue("18446744073709551619"), MaxPriorityFeePerGas: intValue("18446744073709551620"), ChainID: intValue("18446744073709551621"), HasGasPrice: true, HasMaxFeePerGas: true, HasMaxPriorityFeePerGas: true, HasChainID: true, Type: 2, HasType: true}
var wantReceipt = Receipt{TransactionHash: h1, BlockHash: h2, TransactionIndex: 3, BlockNumber: 16, From: a1, To: a2, ContractAddress: a1, HasTo: true, HasContractAddress: true, CumulativeGasUsed: 21001, GasUsed: 21000, Logs: []Log{wantLog}, LogsBloom: [256]byte{255: 1}, Status: 1, HasStatus: true, EffectiveGasPrice: intValue("18446744073709551622"), HasEffectiveGasPrice: true, Type: 2, HasType: true}

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
func query(c *Client, ctx context.Context, name string) (any, bool, error) {
	switch name {
	case "block":
		return c.GetBlock(ctx, Latest())
	case "pending-block":
		return c.GetBlock(ctx, Pending())
	case "transaction":
		return c.GetTransaction(ctx, h1)
	case "receipt":
		return c.GetTransactionReceipt(ctx, h1)
	case "logs", "logs-defaults":
		addresses := []Address{a1}
		if name == "logs-defaults" {
			addresses = nil
		}
		v, e := c.GetLogs(ctx, LogFilter{FromBlock: AtNumber(0), ToBlock: Latest(), Addresses: addresses, Topics: [][]Hash{{h2, h3}, nil, {}}})
		return v, true, e
	}
	panic(name)
}
func TestEVMNew(t *testing.T) {
	var n *rpcFake
	for _, tc := range []struct {
		name  string
		rpc   jsonrpc.Caller
		valid bool
	}{{"nil", nil, false}, {"typed-nil", n, false}, {"valid", &rpcFake{reply: `"0x1"`}, true}} {
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
func TestQueryWireAndValues(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		method, params, reply string
		run                   func(*Client) (any, error)
		want                  any
	}{
		{"eth_chainId", `[]`, `"0x10000000000000001"`, func(c *Client) (any, error) { v, e := c.ChainID(ctx); return v.String(), e }, "18446744073709551617"},
		{"eth_blockNumber", `[]`, `"0x10"`, func(c *Client) (any, error) { return c.BlockNumber(ctx) }, uint64(16)},
		{"eth_getBalance", `[` + A1 + `,"latest"]`, `"0x10000000000000001"`, func(c *Client) (any, error) { v, e := c.GetBalance(ctx, a1, Latest()); return v.String(), e }, "18446744073709551617"},
		{"eth_getTransactionCount", `[` + A1 + `,"pending"]`, `"0x0"`, func(c *Client) (any, error) { return c.GetTransactionCount(ctx, a1, Pending()) }, uint64(0)},
		{"eth_getCode", `[` + A1 + `,"0x0"]`, `"0x00ff"`, func(c *Client) (any, error) { return c.GetCode(ctx, a1, AtNumber(0)) }, Data{0, 255}},
	} {
		t.Run(tc.method, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			v, e := tc.run(&Client{rpc: f})
			if e != nil {
				t.Fatal(e)
			}
			equal(t, v, tc.want)
			assertWire(t, f, tc.method, tc.params, ctx)
		})
	}
}
func TestCompleteDTOs(t *testing.T) {
	defaults := wantLog
	defaults.Removed = false
	for _, tc := range []struct {
		name, method, params, reply string
		want                        any
	}{
		{"block", "eth_getBlockByNumber", `["latest",false]`, edited(blockJSON, "futureExtension", `{"ignored":true}`), wantBlock},
		{"transaction", "eth_getTransactionByHash", `[` + H1 + `]`, transactionJSON, wantTransaction},
		{"receipt", "eth_getTransactionReceipt", `[` + H1 + `]`, receiptJSON, wantReceipt},
		{"logs-defaults", "eth_getLogs", `[{"fromBlock":"0x0","toBlock":"latest","topics":[[` + H2 + `,` + H3 + `],null,null]}]`, `[` + edited(logJSON, "removed", "") + `]`, []Log{defaults}},
		{"logs", "eth_getLogs", `[{"fromBlock":"0x0","toBlock":"latest","address":[` + A1 + `],"topics":[[` + H2 + `,` + H3 + `],null,null]}]`, `[` + logJSON + `]`, []Log{wantLog}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			v, p, e := query(&Client{rpc: f}, context.Background(), tc.name)
			if e != nil {
				t.Fatal(e)
			}
			equal(t, p, true)
			equal(t, v, tc.want)
			assertWire(t, f, tc.method, tc.params, context.Background())
		})
	}
}
func TestPendingHistoricalAndZero(t *testing.T) {
	legacy := wantTransaction
	legacy.GasPrice, legacy.MaxFeePerGas, legacy.MaxPriorityFeePerGas, legacy.ChainID = big.Int{}, big.Int{}, big.Int{}, big.Int{}
	legacy.HasGasPrice, legacy.HasMaxFeePerGas, legacy.HasMaxPriorityFeePerGas, legacy.HasChainID, legacy.HasType = false, false, false, false, false
	legacy.Type = 0
	b := wantBlock
	b.Number = 0
	b.Hash = Hash{}
	b.HasNumber = false
	b.HasHash = false
	b.BaseFeePerGas = big.Int{}
	b.HasBaseFeePerGas = false
	tx := wantTransaction
	tx.To = Address{}
	tx.HasTo = false
	tx.BlockHash = Hash{}
	tx.BlockNumber = 0
	tx.TransactionIndex = 0
	tx.HasBlockHash = false
	tx.HasBlockNumber = false
	tx.HasTransactionIndex = false
	r := wantReceipt
	r.Status = 0
	r.HasStatus = false
	r.Root = h3
	r.HasRoot = true
	z := wantReceipt
	z.Status = 0
	z.To = Address{}
	z.ContractAddress = Address{}
	z.HasTo = false
	z.HasContractAddress = false
	z.EffectiveGasPrice = big.Int{}
	z.HasEffectiveGasPrice = false
	z.Type = 0
	z.HasType = false
	l := wantLog
	l.BlockHash = Hash{}
	l.TransactionHash = Hash{}
	l.BlockNumber = 0
	l.TransactionIndex = 0
	l.LogIndex = 0
	l.HasBlockHash = false
	l.HasTransactionHash = false
	l.HasBlockNumber = false
	l.HasTransactionIndex = false
	l.HasLogIndex = false
	for _, tc := range []struct {
		name, method, reply string
		want                any
	}{
		{"pending", "pending-block", edited(edited(edited(blockJSON, "number", "null"), "hash", "null"), "baseFeePerGas", ""), b},
		{"legacy-optional-absent", "transaction", edited(edited(edited(edited(edited(transactionJSON, "gasPrice", ""), "maxFeePerGas", ""), "maxPriorityFeePerGas", ""), "chainId", ""), "type", ""), legacy},
		{"creation", "transaction", edited(edited(edited(edited(transactionJSON, "to", "null"), "blockHash", "null"), "blockNumber", "null"), "transactionIndex", "null"), tx},
		{"historical", "receipt", edited(edited(receiptJSON, "status", ""), "root", H3), r},
		{"failed-creation", "receipt", edited(edited(edited(edited(edited(receiptJSON, "status", `"0x0"`), "to", "null"), "contractAddress", "null"), "effectiveGasPrice", ""), "type", ""), z},
		{"pending-log", "logs", "[" + edited(edited(edited(edited(edited(logJSON, "blockHash", "null"), "transactionHash", "null"), "blockNumber", "null"), "transactionIndex", "null"), "logIndex", "null") + "]", []Log{l}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, p, e := query(&Client{rpc: &rpcFake{reply: tc.reply}}, context.Background(), tc.method)
			if e != nil {
				t.Fatal(e)
			}
			equal(t, p, true)
			equal(t, v, tc.want)
		})
	}
}
func TestRequiredFieldsAndNullable(t *testing.T) {
	for _, tc := range []struct{ name, raw, keys string }{
		{"block", blockJSON, "number hash parentHash timestamp gasLimit gasUsed transactions"},
		{"transaction", transactionJSON, "hash from to nonce gas value input blockHash blockNumber transactionIndex"},
		{"receipt", receiptJSON, "transactionHash blockHash transactionIndex blockNumber from to contractAddress cumulativeGasUsed gasUsed logs logsBloom status"},
		{"logs", logJSON, "address topics data blockHash transactionHash blockNumber transactionIndex logIndex"},
	} {
		for _, key := range strings.Fields(tc.keys) {
			t.Run(tc.name+"/missing-"+key, func(t *testing.T) {
				raw := edited(tc.raw, key, "")
				if tc.name == "logs" {
					raw = "[" + raw + "]"
				}
				v, p, e := query(&Client{rpc: &rpcFake{reply: raw}}, context.Background(), tc.name)
				assertKind(t, e, apierror.Protocol)
				equal(t, reflect.ValueOf(v).IsZero(), true)
				if tc.name != "logs" {
					equal(t, p, false)
				}
			})
		}
		if tc.name == "logs" {
			continue
		}
		for _, err := range []error{nil, apierror.Error{Kind: apierror.RPC, HasRPCCode: true, RPCCode: -32000}} {
			t.Run(tc.name+"/null-or-rpc", func(t *testing.T) {
				v, p, e := query(&Client{rpc: &rpcFake{reply: "null", err: err}}, context.Background(), tc.name)
				equal(t, p, false)
				equal(t, reflect.ValueOf(v).IsZero(), true)
				if err == nil {
					equal(t, e, nil)
				} else {
					a := assertKind(t, e, apierror.RPC)
					equal(t, a.RPCCode, -32000)
					equal(t, a.HasRPCCode, true)
				}
			})
		}
	}
}
func TestDTOInvalidWire(t *testing.T) {
	for _, tc := range []struct{ name, raw, key, bad string }{
		{"block", blockJSON, "number", "null"}, {"block", blockJSON, "hash", `"0x01"`}, {"block", blockJSON, "timestamp", `"0x00"`}, {"block", blockJSON, "gasUsed", `"0x10000000000000000"`}, {"block", blockJSON, "transactions", `[{}]`}, {"block", blockJSON, "transactions", "null"}, {"block", blockJSON, "baseFeePerGas", "null"},
		{"transaction", transactionJSON, "from", `"0x01"`}, {"transaction", transactionJSON, "input", `"0x0"`}, {"transaction", transactionJSON, "value", `"0x01"`}, {"transaction", transactionJSON, "blockHash", "null"}, {"transaction", transactionJSON, "gasPrice", "null"}, {"transaction", transactionJSON, "type", `"0x10000000000000000"`},
		{"receipt", receiptJSON, "blockHash", "null"}, {"receipt", receiptJSON, "logsBloom", `"0x00"`}, {"receipt", receiptJSON, "status", `"0x2"`}, {"receipt", receiptJSON, "root", H3}, {"receipt", receiptJSON, "gasUsed", `"0x00"`}, {"receipt", receiptJSON, "logs", "null"}, {"receipt", receiptJSON, "logs", "[" + edited(logJSON, "logIndex", "null") + "]"}, {"receipt", receiptJSON, "effectiveGasPrice", `"0x"`},
		{"logs", logJSON, "address", `"0x01"`}, {"logs", logJSON, "topics", `["0x01"]`}, {"logs", logJSON, "topics", "null"}, {"logs", logJSON, "topics", "[" + H1 + "," + H1 + "," + H1 + "," + H1 + "," + H1 + "]"}, {"logs", logJSON, "data", `"0x0"`}, {"logs", logJSON, "removed", "null"}, {"logs", logJSON, "logIndex", `"0x00"`},
	} {
		t.Run(tc.name+"/"+tc.key+"/"+tc.bad, func(t *testing.T) {
			raw := edited(tc.raw, tc.key, tc.bad)
			if tc.name == "logs" {
				raw = "[" + raw + "]"
			}
			v, _, e := query(&Client{rpc: &rpcFake{reply: raw}}, context.Background(), tc.name)
			assertKind(t, e, apierror.Protocol)
			equal(t, reflect.ValueOf(v).IsZero(), true)
		})
	}
}
func amounts(v any) []big.Int {
	switch x := v.(type) {
	case big.Int:
		return []big.Int{x}
	case Block:
		return []big.Int{x.BaseFeePerGas}
	case Transaction:
		return []big.Int{x.Value, x.GasPrice, x.MaxFeePerGas, x.MaxPriorityFeePerGas, x.ChainID}
	case Receipt:
		return []big.Int{x.EffectiveGasPrice}
	case FeeHistoryResult:
		r := append([]big.Int{}, x.BaseFeePerGas...)
		for _, row := range x.Reward {
			r = append(r, row...)
		}
		return r
	}
	panic("not monetary")
}
func amountStrings(v any) []string {
	var r []string
	for _, n := range amounts(v) {
		r = append(r, n.String())
	}
	return r
}
func TestReturnedBigIntOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		run         func(*Client) (any, error)
	}{
		{"chain", `"0x10000000000000001"`, func(c *Client) (any, error) { return c.ChainID(context.Background()) }},
		{"balance", `"0x10000000000000001"`, func(c *Client) (any, error) { return c.GetBalance(context.Background(), a1, Latest()) }},
		{"gas-price", `"0x10000000000000001"`, func(c *Client) (any, error) { return c.GasPrice(context.Background()) }},
		{"block", blockJSON, func(c *Client) (any, error) { v, _, e := c.GetBlock(context.Background(), Latest()); return v, e }},
		{"transaction", transactionJSON, func(c *Client) (any, error) { v, _, e := c.GetTransaction(context.Background(), h1); return v, e }},
		{"receipt", receiptJSON, func(c *Client) (any, error) {
			v, _, e := c.GetTransactionReceipt(context.Background(), h1)
			return v, e
		}},
		{"fee-history", feeJSON, func(c *Client) (any, error) {
			return c.FeeHistory(context.Background(), 2, Latest(), []float64{25, 75})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			c := &Client{rpc: f}
			first, e := tc.run(c)
			if e != nil {
				t.Fatal(e)
			}
			original := amountStrings(first)
			saved := f.result
			fakeBytes, e := json.Marshal(saved)
			if e != nil {
				t.Fatal(e)
			}
			second, e := tc.run(c)
			if e != nil {
				t.Fatal(e)
			}
			for i := range amounts(first) {
				n := amounts(first)
				n[i].SetBit(&n[i], 0, 1-n[i].Bit(0))
				for j, s := range amountStrings(first) {
					if j != i {
						equal(t, s, original[j])
					}
				}
				equal(t, amountStrings(second), original)
				during, e := json.Marshal(saved)
				if e != nil {
					t.Fatal(e)
				}
				equal(t, string(during), string(fakeBytes))
				third, e := tc.run(c)
				if e != nil {
					t.Fatal(e)
				}
				equal(t, amountStrings(third), original)
				n[i].SetBit(&n[i], 0, 1-n[i].Bit(0))
			}
			equal(t, f.reply, tc.reply)
		})
	}
}
