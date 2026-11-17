package bitcoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"reflect"
	"strings"
	"testing"
)

func TestQueryInputs(t *testing.T) {
	for _, bad := range []string{"", strings.Repeat("a", 63), strings.Repeat("a", 65), "0x" + hash, strings.Repeat("g", 64)} {
		for _, name := range []string{"block", "txout", "transaction", "history", "mempool"} {
			t.Run(name+"/"+bad, func(t *testing.T) {
				f := &rpcFake{}
				c := client(t, f)
				var e error
				switch name {
				case "block":
					_, e = c.GetBlock(context.Background(), BlockHash(bad))
				case "txout":
					_, _, e = c.GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(bad)})
				case "transaction":
					_, e = c.GetRawTransaction(context.Background(), GetRawTransactionRequest{TxID: TxID(bad)})
				case "history":
					_, e = c.GetRawTransaction(context.Background(), GetRawTransactionRequest{TxID: TxID(hash), BlockHash: BlockHash(bad), HasBlockHash: true})
				case "mempool":
					_, e = c.GetMempoolEntry(context.Background(), TxID(bad))
				}
				kind(t, e, apierror.InvalidArgument)
				equal(t, f.calls, 0)
			})
		}
	}
	for _, height := range []int64{-1, 0, 9223372036854775807} {
		t.Run(fmt.Sprint("height/", height), func(t *testing.T) {
			f := &rpcFake{reply: `"` + hash + `"`}
			v, e := client(t, f).GetBlockHash(context.Background(), height)
			if height < 0 {
				kind(t, e, apierror.InvalidArgument)
				equal(t, f.calls, 0)
				equal(t, v, BlockHash(""))
			} else {
				equal(t, e, nil)
				equal(t, v, BlockHash(hash))
				equal(t, f.calls, 1)
			}
		})
	}
}
func TestQueryProtocolFailures(t *testing.T) {
	for _, op := range inventory()[:7] {
		for _, reply := range []string{`{`, `[]`, `true`, `"bad"`, `null`} {
			if op.name == "gettxout" && reply == "null" {
				continue
			}
			t.Run(op.name+"/"+reply, func(t *testing.T) {
				f := &rpcFake{reply: reply}
				got, e := op.run(client(t, f), context.Background())
				kind(t, e, apierror.Protocol)
				equal(t, f.calls, 1)
				assertZeroResult(t, got)
			})
		}
	}
	for _, tc := range []struct {
		index int
		reply string
	}{{0, `{"blocks":"x"}`}, {1, `-1`}, {1, `1.5`}, {2, `"short"`}, {2, `"0x` + hash + `"`}, {3, `{"height":true}`}, {4, `{"value":1,"confirmations":"x"}`}, {5, `{"version":2147483648}`}, {5, `{"vin":[{"sequence":-1}]}`}, {6, `{"vsize":"x","fees":{"base":1}}`}} {
		t.Run(fmt.Sprint(tc.index, tc.reply), func(t *testing.T) {
			op := inventory()[tc.index]
			f := &rpcFake{reply: tc.reply}
			v, e := op.run(client(t, f), context.Background())
			kind(t, e, apierror.Protocol)
			assertZeroResult(t, v)
			equal(t, f.calls, 1)
		})
	}
}
func assertZeroResult(t *testing.T, v any) {
	t.Helper()
	if parts, ok := v.([]any); ok {
		for _, part := range parts {
			assertZeroResult(t, part)
		}
		return
	}
	if !reflect.ValueOf(v).IsZero() {
		t.Errorf("failure returned partial value: %#v", v)
	}
}
func TestNullableTxOut(t *testing.T) {
	f := &rpcFake{reply: `null`}
	v, found, e := client(t, f).GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(hash)})
	equal(t, e, nil)
	equal(t, v, TxOut{})
	equal(t, found, false)
	equal(t, f.calls, 1)
}
func TestExactMoney(t *testing.T) {
	for _, tc := range []struct {
		name, number string
		want         int64
		valid        bool
	}{
		{"one", "1e-8", 1, true}, {"trailing_zero", "0.000000010", 1, true}, {"negative_zero", "-0", 0, true}, {"zero", "0", 0, true}, {"maximum", "92233720368.54775807", 9223372036854775807, true},
		{"subsat", "1e-9", 0, false}, {"negative", "-0.00000001", 0, false}, {"overflow", "92233720368.54775808", 0, false}, {"string", `"0.1"`, 0, false}, {"null", "null", 0, false}, {"huge_exponent", "1e999999999", 0, false}, {"tiny_exponent", "1e-999999999", 0, false}, {"missing", "", 0, false},
	} {
		for _, location := range []string{"txout", "vout", "fees.base", "feerate"} {
			if location == "feerate" && tc.number == "" {
				continue
			}
			t.Run(location+"/"+tc.name, func(t *testing.T) {
				field := ""
				if tc.number != "" {
					field = `"value":` + tc.number
				}
				f := &rpcFake{}
				var got int64
				var e error
				var result any
				switch location {
				case "txout":
					f.reply = `{` + field + `}`
					v, found, err := client(t, f).GetTxOut(context.Background(), GetTxOutRequest{TxID: TxID(hash)})
					got = int64(v.Value)
					e = err
					result = v
					if tc.valid {
						equal(t, found, true)
					} else {
						equal(t, found, false)
					}
				case "vout":
					f.reply = `{"vout":[{` + field + `}]}`
					v, err := client(t, f).GetRawTransaction(context.Background(), GetRawTransactionRequest{TxID: TxID(hash)})
					e = err
					result = v
					if len(v.Vout) == 1 {
						got = int64(v.Vout[0].Value)
					} else if tc.valid {
						t.Errorf("output absent: %#v", v)
					}
				case "fees.base":
					if tc.number == "" {
						f.reply = `{"fees":{}}`
					} else {
						f.reply = `{"fees":{"base":` + tc.number + `}}`
					}
					v, err := client(t, f).GetMempoolEntry(context.Background(), TxID(hash))
					got = int64(v.BaseFee)
					e = err
					result = v
				case "feerate":

					f.reply = `{"feerate":` + tc.number + `}`
					v, found, err := client(t, f).EstimateSmartFee(context.Background(), EstimateSmartFeeRequest{ConfTarget: 1})
					got = int64(v.FeeRate)
					e = err
					result = v
					if tc.valid {
						equal(t, found, true)
					} else {
						equal(t, found, false)
					}
				}
				if tc.valid {
					equal(t, e, nil)
					equal(t, got, tc.want)
				} else {
					kind(t, e, apierror.Protocol)
					assertZeroResult(t, result)
				}
				equal(t, f.calls, 1)
			})
		}
	}
}
func TestStandardQueryDecoding(t *testing.T) {
	for _, tc := range []struct {
		index int
		reply string
		want  any
	}{
		{0, `{"blocks":1,"blocks":2,"unknown":{"anything":true}}`, BlockchainInfo{Blocks: 2}},
		{3, `{"tx":["not-a-hash"],"extra":true}`, Block{TxIDs: []string{"not-a-hash"}}},
		{5, `{"vin":[{"txid":null,"vout":0,"scriptSig":{"hex":"odd"},"txinwitness":["native"]}],"blockhash":null,"in_active_chain":false}`, Transaction{Vin: []TxIn{{TxID: json.RawMessage(`null`), Vout: json.RawMessage(`0`), ScriptSig: json.RawMessage(`{"hex":"odd"}`), Witness: json.RawMessage(`["native"]`)}}, BlockHash: json.RawMessage(`null`), InActiveChain: json.RawMessage(`false`)}},
	} {
		t.Run(fmt.Sprint(tc.index), func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			got, e := inventory()[tc.index].run(client(t, f), context.Background())
			equal(t, e, nil)
			equal(t, got, tc.want)
			equal(t, f.calls, 1)
		})
	}
}
func TestReturnedStorageIndependent(t *testing.T) {
	f := &rpcFake{reply: `{"tx":["first"],"nextblockhash":"next"}`}
	c := client(t, f)
	a, e := c.GetBlock(context.Background(), BlockHash(hash))
	equal(t, e, nil)
	b, e := c.GetBlock(context.Background(), BlockHash(hash))
	equal(t, e, nil)
	if len(a.TxIDs) > 0 {
		a.TxIDs[0] = "changed"
	}
	if len(a.NextBlockHash) > 0 {
		a.NextBlockHash[0] = 'x'
	}
	equal(t, b.TxIDs, []string{"first"})
	equal(t, string(b.NextBlockHash), `"next"`)
	equal(t, f.calls, 2)
}
func TestQueryComputeRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic cause")
	for _, k := range []apierror.Kind{apierror.RPC, apierror.HTTP, apierror.Protocol, apierror.Transport, apierror.Configuration, apierror.Unsupported, apierror.NetworkMismatch, apierror.Kind("future-kind")} {
		for _, data := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`{"n":9007199254740993}`)} {
			original := apierror.Error{Kind: k, Method: "old", Message: "  synthetic unauthorized not found timeout  ", Data: data, HTTPStatus: 503, HasRPCCode: data != nil, RPCCode: 0, NodeCode: "native", ResponseBody: []byte("synthetic raw body"), ResponseBodyTruncated: true, SubmissionUnknown: true, Cause: sentinel}
			outer := fmt.Errorf("outer: %w", errors.Join(original, context.Canceled))
			for _, op := range inventory()[:9] {
				t.Run(op.name+"/"+string(k)+"/"+string(data), func(t *testing.T) {
					f := &rpcFake{err: outer}
					v, e := op.run(client(t, f), context.Background())
					got := kind(t, e, k)
					want := original
					want.Method = op.name
					want.SubmissionUnknown = false
					want.Cause = outer
					equal(t, got, want)
					assertZeroResult(t, v)
					equal(t, f.calls, 1)
					if !errors.Is(e, sentinel) || !errors.Is(e, context.Canceled) {
						t.Error("complete joined cause chain lost")
					}
				})
			}
		}
	}
	for _, op := range inventory()[:9] {
		t.Run(op.name+"/unstructured", func(t *testing.T) {
			outer := fmt.Errorf("not found: %w", sentinel)
			f := &rpcFake{err: outer}
			_, e := op.run(client(t, f), context.Background())
			got := kind(t, e, apierror.Transport)
			equal(t, got.Cause, outer)
			equal(t, got.Method, op.name)
			equal(t, got.SubmissionUnknown, false)
			equal(t, f.calls, 1)
		})
	}
}
