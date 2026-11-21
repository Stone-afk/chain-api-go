package solana

import (
	"context"
	"errors"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"testing"
)

type callFn func(nativeContract, context.Context) (any, error)
type methodCase struct {
	name, params, reply string
	call                callFn
	check               func(*testing.T, any)
}

func methodCases() []methodCase {
	k, s, h := testKey.String(), testSignature.String(), testHash.String()
	return []methodCase{
		{"getGenesisHash", `null`, `"` + h + `"`, func(c nativeContract, x context.Context) (any, error) { return c.GetGenesisHash(x) }, func(t *testing.T, v any) { same(t, v, testHash) }},
		{"getSlot", `[]`, `18446744073709551615`, func(c nativeContract, x context.Context) (any, error) { return c.GetSlotWithOpts(x, nil) }, func(t *testing.T, v any) { same(t, v, ^uint64(0)) }},
		{"getBlock", `[9,{"encoding":"base64"}]`, `{"blockhash":"` + h + `","previousBlockhash":"` + h + `","parentSlot":8,"blockTime":null,"blockHeight":0,"transactions":[{"version":"legacy","transaction":["AQID","base64"],"meta":null},{"version":0,"transaction":["BAUG","base64"],"meta":{"fee":9007199254740993}}]}`, func(c nativeContract, x context.Context) (any, error) { return c.GetBlockWithOpts(x, 9, nil) }, func(t *testing.T, v any) {
			b := v.(*srpc.GetBlockResult)
			same(t, b.Blockhash, testHash)
			same(t, b.ParentSlot, uint64(8))
			same(t, b.BlockTime, (*ns.UnixTimeSeconds)(nil))
			same(t, *b.BlockHeight, uint64(0))
			same(t, len(b.Transactions), 2)
			same(t, b.Transactions[0].Version, srpc.LegacyTransactionVersion)
			same(t, b.Transactions[1].Version, srpc.TransactionVersion(0))
			same(t, b.Transactions[1].Meta.Fee, uint64(9007199254740993))
		}},
		{"getBlocks", `[5,10,{"commitment":"finalized"}]`, `[5,7,10]`, func(c nativeContract, x context.Context) (any, error) {
			end := uint64(10)
			return c.GetBlocks(x, 5, &end, srpc.CommitmentFinalized)
		}, func(t *testing.T, v any) { same(t, v, srpc.BlocksResult{5, 7, 10}) }},
		{"getBlocks", `[5]`, `[]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetBlocks(x, 5, nil, "")
		}, func(t *testing.T, v any) { same(t, len(v.(srpc.BlocksResult)), 0) }},
		{"getFirstAvailableBlock", `null`, `18446744073709551615`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetFirstAvailableBlock(x)
		}, func(t *testing.T, v any) { same(t, v, ^uint64(0)) }},
		{"getBalance", `["` + k + `"]`, `{"context":{"slot":19},"value":9007199254740993}`, func(c nativeContract, x context.Context) (any, error) { return c.GetBalanceWithOpts(x, testKey, nil) }, func(t *testing.T, v any) {
			b := v.(*srpc.GetBalanceResult)
			same(t, b.Context.Slot, uint64(19))
			same(t, b.Value, uint64(9007199254740993))
		}},
		{"getAccountInfo", `["` + k + `",{"encoding":"base64"}]`, `{"context":{"slot":20},"value":{"data":["AQID","base64"],"lamports":9007199254740993,"owner":"` + k + `","executable":false,"rentEpoch":0}}`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetAccountInfoWithOpts(x, testKey, nil)
		}, func(t *testing.T, v any) {
			b := v.(*srpc.GetAccountInfoResult)
			same(t, b.Context.Slot, uint64(20))
			same(t, b.Value.Data.GetBinary(), []byte{1, 2, 3})
			same(t, b.Value.Owner, testKey)
			same(t, b.Value.Lamports, uint64(9007199254740993))
		}},
		{"getLatestBlockhash", `[]`, `{"context":{"slot":21},"value":{"blockhash":"` + h + `","lastValidBlockHeight":999}}`, func(c nativeContract, x context.Context) (any, error) { return c.GetLatestBlockhashWithOpts(x, nil) }, func(t *testing.T, v any) {
			b := v.(*srpc.GetLatestBlockhashResult)
			same(t, b.Context.Slot, uint64(21))
			same(t, b.Value.Blockhash, testHash)
			same(t, b.Value.LastValidBlockHeight, uint64(999))
		}},
		{"getSignatureStatuses", `[["` + s + `","` + s + `"]]`, `{"context":{"slot":22},"value":[{"slot":12,"confirmations":null,"err":{"InstructionError":[1,2]},"confirmationStatus":"finalized"},null]}`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetSignatureStatuses(x, false, testSignature, testSignature)
		}, func(t *testing.T, v any) {
			b := v.(*srpc.GetSignatureStatusesResult)
			same(t, b.Context.Slot, uint64(22))
			same(t, len(b.Value), 2)
			same(t, b.Value[1], (*srpc.SignatureStatusesResult)(nil))
			same(t, b.Value[0].Confirmations, (*uint64)(nil))
			same(t, b.Value[0].ConfirmationStatus, srpc.ConfirmationStatusFinalized)
			if b.Value[0].Err == nil {
				t.Fatal("execution error lost")
			}
		}},
		{"getTransaction", `["` + s + `"]`, `{"slot":23,"blockTime":null,"transaction":["AQID","base64"],"meta":{"fee":9007199254740993},"version":0}`, func(c nativeContract, x context.Context) (any, error) { return c.GetTransaction(x, testSignature, nil) }, func(t *testing.T, v any) {
			b := v.(*srpc.GetTransactionResult)
			same(t, b.Slot, uint64(23))
			same(t, b.Version, srpc.TransactionVersion(0))
			same(t, b.Meta.Fee, uint64(9007199254740993))
			same(t, b.BlockTime, (*ns.UnixTimeSeconds)(nil))
		}},
		{"getFeeForMessage", `["AQIDBA=="]`, `{"context":{"slot":24},"value":9007199254740993}`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetFeeForMessageWithOpts(x, payload, nil)
		}, func(t *testing.T, v any) {
			b := v.(*srpc.GetFeeForMessageResult)
			same(t, b.Context.Slot, uint64(24))
			same(t, *b.Value, uint64(9007199254740993))
		}},
		{"simulateTransaction", `["AQIDBA==",{"encoding":"base64"}]`, `{"context":{"slot":25},"value":{"err":{"InstructionError":[0,"Custom"]},"logs":["synthetic"],"accounts":[null],"unitsConsumed":9007199254740993,"replacementBlockhash":{"blockhash":"` + h + `","lastValidBlockHeight":17}}}`, func(c nativeContract, x context.Context) (any, error) {
			return c.SimulateRawTransactionWithOpts(x, []byte{1, 2, 3, 4}, nil)
		}, func(t *testing.T, v any) {
			b := v.(*srpc.SimulateTransactionResponse)
			same(t, b.Context.Slot, uint64(25))
			same(t, *b.Value.UnitsConsumed, uint64(9007199254740993))
			same(t, b.Value.Logs, []string{"synthetic"})
			same(t, b.Value.Accounts[0], (*srpc.Account)(nil))
			same(t, b.Value.ReplacementBlockhash.Blockhash, testHash)
			if b.Value.Err == nil {
				t.Fatal("execution error lost")
			}
		}},
		{"sendTransaction", `["AQIDBA==",{"encoding":"base64","skipPreflight":false}]`, `"` + s + `"`, func(c nativeContract, x context.Context) (any, error) {
			return c.SendEncodedTransactionWithOpts(x, payload, srpc.TransactionOpts{})
		}, func(t *testing.T, v any) { same(t, v, testSignature) }},
	}
}
func TestNativeSolanaWireAndResults(t *testing.T) {
	for _, tc := range methodCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{reply: tc.reply}
			c := nativeClient(t, f)
			ctx := context.WithValue(t.Context(), struct{}{}, "synthetic")
			v, e := tc.call(c, ctx)
			if e != nil {
				t.Fatal(e)
			}
			tc.check(t, v)
			wire(t, f, ctx, tc.name, tc.params)
		})
	}
}
func TestNativeSolanaNullSemantics(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		call        callFn
		want        error
	}{
		{"block", "null", func(c nativeContract, x context.Context) (any, error) { return c.GetBlockWithOpts(x, 0, nil) }, srpc.ErrNotConfirmed},
		{"transaction", "null", func(c nativeContract, x context.Context) (any, error) { return c.GetTransaction(x, testSignature, nil) }, srpc.ErrNotFound},
		{"account", `{"context":{"slot":9},"value":null}`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetAccountInfoWithOpts(x, testKey, nil)
		}, srpc.ErrNotFound},
		{"statuses", "null", func(c nativeContract, x context.Context) (any, error) {
			return c.GetSignatureStatuses(x, true, testSignature)
		}, srpc.ErrNotFound},
		{"statuses-null-value", `{"context":{},"value":null}`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetSignatureStatuses(x, false, testSignature)
		}, srpc.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{reply: tc.reply}
			_, e := tc.call(nativeClient(t, f), t.Context())
			if !errors.Is(e, tc.want) {
				t.Fatalf("sentinel got %v want %v", e, tc.want)
			}
			same(t, f.calls, 1)
		})
	}
	for _, value := range []string{"null", "0"} {
		t.Run("fee/"+value, func(t *testing.T) {
			f := &fakeCaller{reply: `{"context":{"slot":9},"value":` + value + `}`}
			r, e := nativeClient(t, f).GetFeeForMessageWithOpts(t.Context(), payload, nil)
			if e != nil {
				t.Fatal(e)
			}
			same(t, r.Context.Slot, uint64(9))
			if value == "null" {
				same(t, r.Value, (*uint64)(nil))
			} else {
				same(t, *r.Value, uint64(0))
			}
		})
	}
}
