package tron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

// Synthetic object deliberately has no valid transaction schema or credentials.
const syntheticPayload = ` { "raw_data" : {"n":9007199254740993}, "signature":["synthetic-sig","second"], "unknown":"synthetic payload marker" } `

func TestBroadcastRawObjectOnce(t *testing.T) {
	for _, response := range []string{`{"result":true}`, `{"result":true,"txid":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789","code":"raw","message":" accepted "}`} {
		t.Run(response, func(t *testing.T) {
			p := &fakePoster{response: response}
			ctx := context.Background()
			payload := TransactionPayload{JSON: json.RawMessage(syntheticPayload)}
			got, accepted, err := newClient(t, p).BroadcastTransaction(ctx, payload)
			if err != nil || !accepted || !got.Result {
				t.Error("accepted result rejected")
			}
			want := BroadcastResult{Result: true}
			if strings.Contains(response, "txid") {
				want.TxID = string(txID)
				want.Code = "raw"
				want.Message = " accepted "
			}
			if got != want {
				t.Error("broadcast DTO changed or txid synthesized")
			}
			checkWire(t, p, ctx, "/wallet/broadcasttransaction", syntheticPayload)
			raw, ok := p.request.(json.RawMessage)
			if !ok {
				t.Error("Poster request is not RawMessage")
			} else if string(raw) != syntheticPayload {
				t.Error("payload rewritten before Poster")
			}
			if string(payload.JSON) != syntheticPayload {
				t.Error("caller payload mutated")
			}
			if !strings.Contains(string(p.body), "9007199254740993") {
				t.Error("large integer lost precision")
			}
		})
	}
	// No transaction field inspection: an empty object is a valid JSON object.
	p := &fakePoster{response: `{"result":true}`}
	_, ok, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(`{}`)})
	if err != nil || !ok || p.calls != 1 {
		t.Error("empty object was subjected to transaction schema")
	}
}
func TestBroadcastLocalValidation(t *testing.T) {
	for _, payload := range []string{"", " ", `null`, `[]`, `"object"`, `0`, `true`, `{`, `{} {}`} {
		t.Run(payload, func(t *testing.T) {
			p := &fakePoster{}
			got, present, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(payload)})
			checkError(t, err, apierror.InvalidArgument, false)
			if present || got != (BroadcastResult{}) {
				t.Error("invalid payload returned result")
			}
			if p.calls != 0 {
				t.Error("invalid payload sent")
			}
		})
	}
}
func TestBroadcastResponseBoundary(t *testing.T) {
	for _, response := range []string{`null`, `[]`, `{}`, `{"result":null}`, `{"result":"true"}`, `{"result":1}`, `{"result":true,"txid":"bad"}`, `{"result":true,"txid":7}`, `{"result":true,"txid":"0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`, `{"Error":false}`} {
		t.Run(response, func(t *testing.T) {
			p := &fakePoster{response: response}
			got, accepted, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(syntheticPayload)})
			checkError(t, err, apierror.Protocol, true)
			if accepted || got != (BroadcastResult{}) {
				t.Error("malformed result accepted")
			}
			if p.calls != 1 {
				t.Error("unknown submission retried or omitted")
			}
		})
	}
}
func TestBroadcastRejectionOriginalFields(t *testing.T) {
	for _, message := range []string{" unsupported ", "already known", "not found", "synthetic endpoint/token marker"} {
		t.Run(message, func(t *testing.T) {
			response := `{"result":false,"code":"CONTRACT_VALIDATE_ERROR","message":"` + message + `","detail":{"n":9007199254740993}}`
			p := &fakePoster{response: response}
			got, accepted, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(syntheticPayload)})
			e := checkError(t, err, apierror.RPC, false)
			if e.Message != message || e.NodeCode != "CONTRACT_VALIDATE_ERROR" || string(e.Data) != response {
				t.Error("rejection details changed")
			}
			if e.HTTPStatus != 0 || len(e.ResponseBody) != 0 || e.Cause != nil {
				t.Error("fabricated HTTP/cause fields")
			}
			if e.Method != "/wallet/broadcasttransaction" {
				t.Error("method missing")
			}
			if accepted || got != (BroadcastResult{}) {
				t.Error("rejection returned success")
			}
			if p.calls != 1 {
				t.Error("rejection retried or omitted")
			}
			if strings.Contains(errString(err), "synthetic payload marker") {
				t.Error("request payload appended to error")
			}
		})
	}
	p := &fakePoster{response: `{"Error":"raw top-level rejection"}`}
	_, accepted, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(`{}`)})
	e := checkError(t, err, apierror.RPC, false)
	if accepted || e.Message != "raw top-level rejection" || string(e.Data) != p.response || p.calls != 1 {
		t.Error("top-level rejection lost")
	}
}
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func TestBroadcastDependencyUnknown(t *testing.T) {
	sentinel := errors.New("synthetic transport marker")
	for _, kind := range []apierror.Kind{apierror.Transport, apierror.HTTP, apierror.Protocol, apierror.RPC} {
		t.Run(kind.ToString(), func(t *testing.T) {
			original := apierror.Error{Kind: kind, HTTPStatus: 502, NodeCode: "raw-code", HasRPCCode: true, RPCCode: 0, Message: "  already known  ", Data: json.RawMessage(`null`), ResponseBody: []byte("synthetic raw body"), ResponseBodyTruncated: true, Cause: sentinel}
			wrapped := fmt.Errorf("outer: %w", original)
			p := &fakePoster{err: wrapped}
			_, accepted, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(syntheticPayload)})
			e := checkError(t, err, kind, kind != apierror.RPC)
			if accepted || p.calls != 1 {
				t.Error("dependency failure accepted/retried")
			}
			if e.Cause != wrapped || !errors.Is(err, sentinel) {
				t.Error("dependency chain lost")
			}
			if e.HTTPStatus != 502 || e.NodeCode != original.NodeCode || !e.HasRPCCode || e.RPCCode != 0 || e.Message != original.Message || string(e.Data) != "null" || string(e.ResponseBody) != string(original.ResponseBody) || !e.ResponseBodyTruncated {
				t.Error("dependency fields lost")
			}
			if e.Method != "/wallet/broadcasttransaction" {
				t.Error("method missing")
			}
		})
	}
	for _, cause := range []error{sentinel, fmt.Errorf("canceled wrapper: %w", context.Canceled), errors.Join(context.Canceled, context.DeadlineExceeded)} {
		t.Run("unstructured", func(t *testing.T) {
			p := &fakePoster{err: cause}
			_, _, err := newClient(t, p).BroadcastTransaction(context.Background(), TransactionPayload{JSON: json.RawMessage(`{}`)})
			e := checkError(t, err, apierror.Transport, true)
			if e.Cause != cause || !errors.Is(err, cause) || p.calls != 1 {
				t.Error("unstructured error chain/count lost")
			}
		})
	}
}
