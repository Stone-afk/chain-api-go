package bitcoin

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"reflect"
	"testing"
)

const hash = "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"
const raw = "00AbFF"

type rpcFake struct {
	calls  int
	method string
	params []byte
	ctx    context.Context
	reply  string
	err    error
}

func (f *rpcFake) Call(ctx context.Context, method string, params, result any) error {
	f.calls++
	f.ctx = ctx
	f.method = method
	var err error
	f.params, err = json.Marshal(params)
	if err != nil {
		return err
	}
	if f.err != nil {
		return f.err
	}
	// Raw replies permit malformed JSON to reach the chain decoding boundary.
	if dst, ok := result.(*json.RawMessage); ok {
		*dst = append((*dst)[:0], f.reply...)
		return nil
	}
	return json.Unmarshal([]byte(f.reply), result)
}
func client(t *testing.T, f *rpcFake) *Client {
	t.Helper()
	c, e := New(Config{}, f)
	if e != nil {
		t.Errorf("New: %v", e)
	}
	return c
}
func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}
func kind(t *testing.T, e error, want apierror.Kind) apierror.Error {
	t.Helper()
	var a apierror.Error
	if !errors.As(e, &a) || a.Kind != want {
		t.Errorf("error=%v; want %s", e, want)
	}
	return a
}
func wire(t *testing.T, f *rpcFake, m, p string, ctx context.Context) {
	t.Helper()
	equal(t, f.calls, 1)
	equal(t, f.method, m)
	equal(t, string(f.params), p)
	equal(t, f.ctx, ctx)
}
func TestConstructor(t *testing.T) {
	c, e := New(Config{}, nil)
	equal(t, c, (*Client)(nil))
	kind(t, e, apierror.Configuration)
	f := &rpcFake{}
	c, e = New(Config{}, f)
	if c == nil || e != nil {
		t.Errorf("valid injected caller rejected: %v", e)
	}
	equal(t, f.calls, 0)
}

type operation struct {
	name, params, reply string
	run                 func(*Client, context.Context) (any, error)
	want                any
}

func inventory() []operation {
	return []operation{
		{"getblockchaininfo", "[]", `{"chain":"regtest","blocks":7,"headers":8,"bestblockhash":"best","pruned":true,"pruneheight":2}`, func(c *Client, x context.Context) (any, error) { return c.GetBlockchainInfo(x) }, BlockchainInfo{Chain: "regtest", Blocks: 7, Headers: 8, BestBlockHash: "best", Pruned: true, PruneHeight: json.RawMessage(`2`)}},
		{"getblockcount", "[]", `7`, func(c *Client, x context.Context) (any, error) { return c.GetBlockCount(x) }, int64(7)},
		{"getblockhash", "[7]", `"` + hash + `"`, func(c *Client, x context.Context) (any, error) { return c.GetBlockHash(x, 7) }, BlockHash(hash)},
		{"getblock", `["` + hash + `",1]`, `{"hash":"block","height":7,"confirmations":3,"time":123,"tx":["tx"],"previousblockhash":null}`, func(c *Client, x context.Context) (any, error) { return c.GetBlock(x, BlockHash(hash)) }, Block{Hash: "block", Height: 7, Confirmations: 3, Time: 123, TxIDs: []string{"tx"}, PreviousBlockHash: json.RawMessage(`null`)}},
		{"gettxout", `["` + hash + `",2]`, `{"bestblock":"b","confirmations":4,"value":0.00000001,"scriptPubKey":{"hex":"not-recursively-validated","type":"pubkey"},"coinbase":true}`, func(c *Client, x context.Context) (any, error) {
			v, ok, e := c.GetTxOut(x, GetTxOutRequest{TxID: TxID(hash), Vout: 2})
			return []any{v, ok}, e
		}, []any{TxOut{BestBlock: "b", Confirmations: 4, Value: 1, ScriptPubKey: ScriptPubKey{Hex: "not-recursively-validated", Type: "pubkey"}, Coinbase: true}, true}},
		{"getrawtransaction", `["` + hash + `",1]`, `{"txid":"tx","hash":"wtx","hex":"native","version":2,"locktime":3,"size":4,"vsize":5,"weight":6,"vin":[{"sequence":8,"coinbase":"aa"}],"vout":[{"n":0,"value":0.00000002,"scriptPubKey":{}}],"confirmations":0}`, func(c *Client, x context.Context) (any, error) {
			return c.GetRawTransaction(x, GetRawTransactionRequest{TxID: TxID(hash)})
		}, Transaction{TxID: "tx", WTxID: "wtx", Hex: "native", Version: 2, LockTime: 3, Size: 4, VSize: 5, Weight: 6, Vin: []TxIn{{Sequence: 8, Coinbase: json.RawMessage(`"aa"`)}}, Vout: []TxOutput{{Value: 2}}, Confirmations: json.RawMessage(`0`)}},
		{"getmempoolentry", `["` + hash + `"]`, `{"vsize":3,"weight":12,"time":42,"height":7,"fees":{"base":0.00000003},"depends":["a"],"spentby":["b"]}`, func(c *Client, x context.Context) (any, error) { return c.GetMempoolEntry(x, TxID(hash)) }, MempoolEntry{VSize: 3, Weight: 12, Time: 42, Height: 7, BaseFee: 3, Depends: []string{"a"}, SpentBy: []string{"b"}}},
		{"estimatesmartfee", "[6]", `{"feerate":0.00001000,"blocks":6}`, func(c *Client, x context.Context) (any, error) {
			v, ok, e := c.EstimateSmartFee(x, EstimateSmartFeeRequest{ConfTarget: 6})
			return []any{v, ok}, e
		}, []any{FeeEstimate{FeeRate: 1000, Blocks: 6}, true}},
		{"testmempoolaccept", `[["` + raw + `"]]`, `[{"txid":"tx","wtxid":"wtx","allowed":false,"reject-reason":"policy","reject-details":"native details","vsize":null,"fees":{"base":1e-8}}]`, func(c *Client, x context.Context) (any, error) { return c.TestMempoolAccept(x, RawTransaction(raw)) }, MempoolAcceptResult{TxID: "tx", WTxID: "wtx", Allowed: false, RejectReason: "policy", RejectDetails: "native details", VSize: json.RawMessage(`null`), Fees: json.RawMessage(`{"base":1e-8}`)}},
		{"sendrawtransaction", `["` + raw + `"]`, `"` + hash + `"`, func(c *Client, x context.Context) (any, error) { return c.SendRawTransaction(x, RawTransaction(raw)) }, TxID(hash)},
	}
}
func TestMethodInventory(t *testing.T) {
	for _, op := range inventory() {
		t.Run(op.name, func(t *testing.T) {
			f := &rpcFake{reply: op.reply}
			ctx := context.WithValue(context.Background(), struct{}{}, "identity")
			got, e := op.run(client(t, f), ctx)
			if e != nil {
				t.Errorf("legal request: %v", e)
			}
			equal(t, got, op.want)
			wire(t, f, op.name, op.params, ctx)
		})
	}
}
func TestContextPassedUnchanged(t *testing.T) {
	for _, op := range inventory() {
		t.Run(op.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			f := &rpcFake{err: context.Canceled}
			_, e := op.run(client(t, f), ctx)
			wire(t, f, op.name, op.params, ctx)
			if !errors.Is(e, context.Canceled) {
				t.Errorf("cancellation cause lost: %v", e)
			}
		})
	}
}
func TestOptionalPositions(t *testing.T) {
	for _, tc := range []struct {
		name, params, reply string
		run                 func(*Client) error
	}{
		{"mempool_false", `["` + hash + `",0,false]`, `null`, func(c *Client) error {
			_, _, e := c.GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(hash), HasIncludeMempool: true})
			return e
		}},
		{"mempool_true", `["` + hash + `",0,true]`, `null`, func(c *Client) error {
			_, _, e := c.GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(hash), HasIncludeMempool: true, IncludeMempool: true})
			return e
		}},
		{"mempool_ignored", `["` + hash + `",0]`, `null`, func(c *Client) error {
			_, _, e := c.GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(hash), IncludeMempool: true})
			return e
		}},
		{"blockhash", `["` + hash + `",1,"` + hash + `"]`, `{}`, func(c *Client) error {
			_, e := c.GetRawTransaction(context.Background(), GetRawTransactionRequest{TxID: TxID(hash), HasBlockHash: true, BlockHash: BlockHash(hash)})
			return e
		}},
		{"blockhash_ignored", `["` + hash + `",1]`, `{}`, func(c *Client) error {
			_, e := c.GetRawTransaction(context.Background(), GetRawTransactionRequest{TxID: TxID(hash), BlockHash: "invalid"})
			return e
		}},
		{"mode_ignored", `[1]`, `{"blocks":1}`, func(c *Client) error {
			_, _, e := c.EstimateSmartFee(context.Background(), EstimateSmartFeeRequest{ConfTarget: 1, Mode: "invalid"})
			return e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			e := tc.run(client(t, f))
			equal(t, e, nil)
			equal(t, string(f.params), tc.params)
			equal(t, f.calls, 1)
		})
	}
}
