package solana

import (
	"github.com/Stone-afk/chain-api-go/apierror"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"testing"
)

func TestNativeSolanaSubmitPayloadAndOptions(t *testing.T) {
	for _, enc := range []ns.EncodingType{"", ns.EncodingBase64, ns.EncodingBase58} {
		for _, skip := range []bool{false, true} {
			t.Run(string(enc)+map[bool]string{false: "false", true: "true"}[skip], func(t *testing.T) {
				retries := uint(0)
				slot := uint64(0)
				data := payload
				if enc == ns.EncodingBase58 {
					data = "123ABCxyz"
				}
				opts := srpc.TransactionOpts{Encoding: enc, SkipPreflight: skip, PreflightCommitment: srpc.CommitmentProcessed, MaxRetries: &retries, MinContextSlot: &slot}
				before := opts
				f := &fakeCaller{reply: `"` + testSignature.String() + `"`}
				s, e := nativeClient(t, f).SendEncodedTransactionWithOpts(t.Context(), data, opts)
				if e != nil {
					t.Fatal(e)
				}
				same(t, s, testSignature)
				same(t, opts, before)
				same(t, retries, uint(0))
				same(t, slot, uint64(0))
				encoding := enc
				if encoding == "" {
					encoding = ns.EncodingBase64
				}
				want := `["` + data + `",{"encoding":"` + string(encoding) + `","skipPreflight":` + map[bool]string{false: "false", true: "true"}[skip] + `,"preflightCommitment":"processed","maxRetries":0,"minContextSlot":0}]`
				wire(t, f, t.Context(), "sendTransaction", want)
			})
		}
	}
}
func TestNativeSolanaInvalidSubmitZeroCalls(t *testing.T) {
	for _, tc := range []struct {
		data string
		enc  ns.EncodingType
	}{{"", ""}, {payload, "json"}, {"AQ", ns.EncodingBase64}, {"====", ns.EncodingBase64}, {" AQIDBA==", ns.EncodingBase64}, {"0OIl", ns.EncodingBase58}, {"abc xyz", ns.EncodingBase58}, {"", ns.EncodingBase58}} {
		t.Run(tc.data+"/"+string(tc.enc), func(t *testing.T) {
			f := &fakeCaller{}
			v, e := nativeClient(t, f).SendEncodedTransactionWithOpts(t.Context(), tc.data, srpc.TransactionOpts{Encoding: tc.enc})
			a := classified(t, e, apierror.InvalidArgument, false)
			same(t, a.Method, "sendTransaction")
			same(t, v, ns.Signature{})
			same(t, f.calls, 0)
		})
	}
}
func TestNativeSolanaMalformedSubmitUnknown(t *testing.T) {
	for _, reply := range []string{`""`, `"abc"`, `"0OIl"`, `"` + (ns.Signature{}).String() + `"`, `null`, `{}`, `1`, `{`} {
		t.Run(reply, func(t *testing.T) {
			f := &fakeCaller{reply: reply}
			v, e := nativeClient(t, f).SendEncodedTransactionWithOpts(t.Context(), payload, srpc.TransactionOpts{})
			a := classified(t, e, apierror.Protocol, true)
			same(t, a.Method, "sendTransaction")
			same(t, v, ns.Signature{})
			same(t, f.calls, 1)
		})
	}
}
