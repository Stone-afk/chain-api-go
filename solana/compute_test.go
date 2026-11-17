package solana

import (
	"context"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"testing"
)

func TestNativeSolanaOptions(t *testing.T) {
	z := uint64(0)
	no := false
	k, s := testKey.String(), testSignature.String()
	for _, tc := range []struct {
		name, params string
		call         callFn
	}{
		{"getSlot", `[{"commitment":"processed","minContextSlot":0}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetSlotWithOpts(x, &srpc.GetSlotOpts{Commitment: srpc.CommitmentProcessed, MinContextSlot: &z})
		}},
		{"getBalance", `["` + k + `",{"commitment":"confirmed","minContextSlot":0}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetBalanceWithOpts(x, testKey, &srpc.GetBalanceOpts{Commitment: srpc.CommitmentConfirmed, MinContextSlot: &z})
		}},
		{"getBlock", `[0,{"encoding":"json","commitment":"confirmed","maxSupportedTransactionVersion":0,"rewards":false,"transactionDetails":"full"}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetBlockWithOpts(x, 0, &srpc.GetBlockOpts{Encoding: ns.EncodingJSON, Commitment: srpc.CommitmentConfirmed, MaxSupportedTransactionVersion: &z, Rewards: &no, TransactionDetails: srpc.TransactionDetailsFull})
		}},
		{"getAccountInfo", `["` + k + `",{"encoding":"base64","minContextSlot":0,"dataSlice":{"offset":0,"length":0}}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetAccountInfoWithOpts(x, testKey, &srpc.GetAccountInfoOpts{MinContextSlot: &z, DataSlice: &srpc.DataSlice{Offset: &z, Length: &z}})
		}},
		{"getLatestBlockhash", `[{"commitment":"confirmed","minContextSlot":0}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetLatestBlockhashWithOpts(x, &srpc.GetLatestBlockhashOpts{Commitment: srpc.CommitmentConfirmed, MinContextSlot: &z})
		}},
		{"getSignatureStatuses", `[["` + s + `"],{"searchTransactionHistory":true}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetSignatureStatuses(x, true, testSignature)
		}},
		{"getTransaction", `["` + s + `",{"encoding":"base64","commitment":"finalized","maxSupportedTransactionVersion":0}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetTransaction(x, testSignature, &srpc.GetTransactionOpts{Encoding: ns.EncodingBase64, Commitment: srpc.CommitmentFinalized, MaxSupportedTransactionVersion: &z})
		}},
		{"getFeeForMessage", `["AQIDBA==",{"commitment":"confirmed","minContextSlot":0}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.GetFeeForMessageWithOpts(x, payload, &srpc.GetFeeForMessageOpts{Commitment: srpc.CommitmentConfirmed, MinContextSlot: &z})
		}},
		{"simulateTransaction", `["AQIDBA==",{"encoding":"base64","sigVerify":true,"innerInstructions":true,"minContextSlot":0,"accounts":{"encoding":"base64","addresses":["` + k + `"]}}]`, func(c nativeContract, x context.Context) (any, error) {
			return c.SimulateRawTransactionWithOpts(x, []byte{1, 2, 3, 4}, &srpc.SimulateTransactionOpts{SigVerify: true, InnerInstructions: true, MinContextSlot: &z, Accounts: &srpc.SimulateTransactionAccountsOpts{Addresses: []ns.PublicKey{testKey}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reply string
			for _, base := range methodCases() {
				if base.name == tc.name {
					reply = base.reply
				}
			}
			f := &fakeCaller{reply: reply}
			_, e := tc.call(nativeClient(t, f), t.Context())
			if e != nil {
				t.Fatal(e)
			}
			wire(t, f, t.Context(), tc.name, tc.params)
			same(t, z, uint64(0))
			same(t, no, false)
		})
	}
}
func TestNativeSolanaSimulationSerializedBytes(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(map[bool]string{false: "defaults", true: "replace"}[replace], func(t *testing.T) {
			data := []byte{1, 2, 3, 4}
			f := &fakeCaller{reply: `{"context":{"slot":7},"value":{"err":null}}`}
			r, e := nativeClient(t, f).SimulateRawTransactionWithOpts(t.Context(), data, &srpc.SimulateTransactionOpts{ReplaceRecentBlockhash: replace})
			if e != nil {
				t.Fatal(e)
			}
			same(t, r.Context.Slot, uint64(7))
			same(t, r.Value.Err, nil)
			same(t, data, []byte{1, 2, 3, 4})
			want := `["AQIDBA==",{"encoding":"base64"}]`
			if replace {
				want = `["AQIDBA==",{"encoding":"base64","replaceRecentBlockhash":true}]`
			}
			wire(t, f, t.Context(), "simulateTransaction", want)
		})
	}
}
func TestNativeSolanaUpstreamValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		call callFn
	}{
		{"block encoding", func(c nativeContract, x context.Context) (any, error) {
			return c.GetBlockWithOpts(x, 0, &srpc.GetBlockOpts{Encoding: "invalid"})
		}},
		{"transaction encoding", func(c nativeContract, x context.Context) (any, error) {
			return c.GetTransaction(x, testSignature, &srpc.GetTransactionOpts{Encoding: "invalid"})
		}},
		{"parsed account slice", func(c nativeContract, x context.Context) (any, error) {
			return c.GetAccountInfoWithOpts(x, testKey, &srpc.GetAccountInfoOpts{Encoding: ns.EncodingJSONParsed, DataSlice: &srpc.DataSlice{}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCaller{}
			_, e := tc.call(nativeClient(t, f), t.Context())
			if e == nil {
				t.Fatal("upstream invalid request accepted")
			}
			same(t, f.calls, 0)
		})
	}
}
