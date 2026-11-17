package bitcoin

import (
	"context"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"testing"
)

func TestEstimateInputs(t *testing.T) {
	for _, target := range []int64{0, -1, 1, 1008, 1009} {
		for _, mode := range []EstimateMode{EstimateUnset, EstimateEconomical, EstimateConservative, "invalid"} {
			t.Run(fmt.Sprint(target, "/", mode), func(t *testing.T) {
				f := &rpcFake{reply: `{"feerate":1e-8,"blocks":2}`}
				v, found, e := client(t, f).EstimateSmartFee(context.Background(), EstimateSmartFeeRequest{ConfTarget: target, Mode: mode, HasMode: true})
				if target < 1 || target > 1008 || mode == "invalid" {
					kind(t, e, apierror.InvalidArgument)
					equal(t, f.calls, 0)
					equal(t, v, FeeEstimate{})
					equal(t, found, false)
				} else {
					equal(t, e, nil)
					equal(t, v, FeeEstimate{FeeRate: 1, Blocks: 2})
					equal(t, found, true)
					wire(t, f, "estimatesmartfee", fmt.Sprintf(`[%d,"%s"]`, target, mode), context.Background())
				}
			})
		}
	}
}
func TestEstimateAbsent(t *testing.T) {
	for _, tc := range []struct {
		reply   string
		want    FeeEstimate
		found   bool
		invalid bool
	}{
		{`{"blocks":5,"errors":["insufficient data"]}`, FeeEstimate{Blocks: 5, Errors: []string{"insufficient data"}}, false, false},
		{`{}`, FeeEstimate{}, false, false},
		{`{"feerate":0,"blocks":1}`, FeeEstimate{Blocks: 1}, true, false},
		{`{"feerate":null,"blocks":1}`, FeeEstimate{}, false, true},
		{`[]`, FeeEstimate{}, false, true}, {`null`, FeeEstimate{}, false, true}, {`{"blocks":"bad"}`, FeeEstimate{}, false, true},
	} {
		t.Run(tc.reply, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			v, found, e := client(t, f).EstimateSmartFee(context.Background(), EstimateSmartFeeRequest{ConfTarget: 1})
			if tc.invalid {
				kind(t, e, apierror.Protocol)
			} else {
				equal(t, e, nil)
			}
			equal(t, v, tc.want)
			equal(t, found, tc.found)
			equal(t, f.calls, 1)
		})
	}
}
func TestPreflightResultShape(t *testing.T) {
	for _, reply := range []string{`null`, `{}`, `[]`, `[{},{}]`, `[null]`, `[{}]`, `[{"allowed":null}]`, `[{"allowed":"false"}]`, `[{"allowed":true,"txid":5}]`, `[`} {
		t.Run(reply, func(t *testing.T) {
			f := &rpcFake{reply: reply}
			v, e := client(t, f).TestMempoolAccept(context.Background(), RawTransaction(raw))
			kind(t, e, apierror.Protocol)
			equal(t, v, MempoolAcceptResult{})
			equal(t, f.calls, 1)
		})
	}
}
func TestPreflightStandardDecoding(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			f := &rpcFake{reply: fmt.Sprintf(`[{"allowed":false,"allowed":%t,"unknown":123,"reject-reason":"native policy"}]`, allowed)}
			v, e := client(t, f).TestMempoolAccept(context.Background(), RawTransaction(raw))
			equal(t, e, nil)
			equal(t, v, MempoolAcceptResult{Allowed: allowed, RejectReason: "native policy"})
			equal(t, f.calls, 1)
		})
	}
}
