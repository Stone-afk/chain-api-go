package bitcoin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"testing"
)

func TestRawInputValidation(t *testing.T) {
	for _, payload := range []RawTransaction{"", "0", "xyz0", "0x00", "aa bb"} {
		for _, submit := range []bool{false, true} {
			t.Run(fmt.Sprint(payload, "/", submit), func(t *testing.T) {
				f := &rpcFake{}
				c := client(t, f)
				var e error
				if submit {
					v, err := c.SendRawTransaction(context.Background(), payload)
					e = err
					equal(t, v, TxID(""))
				} else {
					v, err := c.TestMempoolAccept(context.Background(), payload)
					e = err
					equal(t, v, MempoolAcceptResult{})
				}
				a := kind(t, e, apierror.InvalidArgument)
				equal(t, a.SubmissionUnknown, false)
				equal(t, f.calls, 0)
			})
		}
	}
}
func TestSubmitResultValidation(t *testing.T) {
	for _, reply := range []string{`null`, `0`, `{}`, `""`, `"short"`, `"0x` + hash + `"`, `"GBCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"`, `{`} {
		t.Run(reply, func(t *testing.T) {
			f := &rpcFake{reply: reply}
			v, e := client(t, f).SendRawTransaction(context.Background(), RawTransaction(raw))
			a := kind(t, e, apierror.Protocol)
			equal(t, a.SubmissionUnknown, true)
			equal(t, v, TxID(""))
			wire(t, f, "sendrawtransaction", `["`+raw+`"]`, context.Background())
		})
	}
}
func TestSubmitRawErrors(t *testing.T) {
	sentinel := errors.New("synthetic node cause")
	for _, k := range []apierror.Kind{apierror.RPC, apierror.Transport, apierror.HTTP, apierror.Protocol} {
		t.Run(string(k), func(t *testing.T) {
			original := apierror.Error{Kind: k, Method: "old", Message: " synthetic rejection timeout ", Data: json.RawMessage(`{"n":9007199254740993}`), HasRPCCode: true, RPCCode: 0, NodeCode: "native", HTTPStatus: 500, ResponseBody: []byte("synthetic bytes"), ResponseBodyTruncated: true, Cause: sentinel}
			outer := fmt.Errorf("outer: %w", original)
			f := &rpcFake{err: outer}
			v, e := client(t, f).SendRawTransaction(context.Background(), RawTransaction(raw))
			a := kind(t, e, k)
			want := original
			want.Method = "sendrawtransaction"
			want.SubmissionUnknown = k != apierror.RPC
			want.Cause = outer
			equal(t, a, want)
			equal(t, v, TxID(""))
			equal(t, f.calls, 1)
			if !errors.Is(e, sentinel) {
				t.Error("cause lost")
			}
		})
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, errors.New("rejected not found")} {
		t.Run(cause.Error(), func(t *testing.T) {
			outer := fmt.Errorf("outer: %w", errors.Join(cause, sentinel))
			f := &rpcFake{err: outer}
			v, e := client(t, f).SendRawTransaction(context.Background(), RawTransaction(raw))
			a := kind(t, e, apierror.Transport)
			equal(t, a.Cause, outer)
			equal(t, a.SubmissionUnknown, true)
			equal(t, a.Method, "sendrawtransaction")
			equal(t, v, TxID(""))
			equal(t, f.calls, 1)
			if !errors.Is(e, cause) || !errors.Is(e, sentinel) {
				t.Error("joined cause lost")
			}
		})
	}
}
