package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func TestSubmissionSingleUnchangedPayload(t *testing.T) {
	payload := rawString[Data](t, "0x000180FF")
	const returnedHash = "0xAB00000000000000000000000000000000000000000000000000000000000003"
	f := &rpcFake{reply: `"` + returnedHash + `"`}
	ctx := context.Background()
	got, e := (&Client{rpc: f}).SendRawTransaction(ctx, payload)
	if e != nil {
		t.Fatalf("missing submit behavior: %v", e)
	}
	if got != rawString[Hash](t, returnedHash) || !reflect.DeepEqual(payload, rawString[Data](t, "0x000180FF")) {
		t.Fatal("hash or original payload changed")
	}
	assertWire(t, f, "eth_sendRawTransaction", `["0x000180FF"]`, ctx)
}

// dependencyFailure is a typeable leaf; its marker is synthetic response content.
type dependencyFailure string

func (e dependencyFailure) Error() string { return string(e) }

func TestRawErrorFieldsAndSubmissionClassification(t *testing.T) {
	leaf := dependencyFailure("SYNTHETIC_CAUSE")
	for _, tc := range []struct {
		name string
		kind apierror.Kind
		code int
		has  bool
		data string
	}{
		{"rpc-zero", apierror.RPC, 0, true, "null"},
		{"rpc-negative", apierror.RPC, -32000, true, ` {"n":9007199254740993,"marker":"SYNTHETIC_NODE"} `},
		{"transport", apierror.Transport, 0, false, ""},
		{"http", apierror.HTTP, 0, false, "null"},
		{"protocol", apierror.Protocol, 0, false, ""},
	} {
		ops := []string{"query", "compute", "submit"}
		if tc.name == "rpc-zero" {
			ops = append(ops, "submit-other-message")
		}
		for _, op := range ops {
			t.Run(op+"/"+tc.name, func(t *testing.T) {
				source := apierror.Error{Kind: tc.kind, Method: "dependency_method", RPCCode: tc.code, HasRPCCode: tc.has, NodeCode: "SYNTHETIC_NODE_CODE", Message: "  already known SYNTHETIC_NODE \n", Data: json.RawMessage(tc.data), Cause: leaf}
				if op == "submit-other-message" {
					op = "submit"
					source.Message = "  transaction rejected SYNTHETIC_NODE \n"
				}
				if tc.kind == apierror.HTTP {
					source.HTTPStatus = 503
					source.ResponseBody = []byte(" SYNTHETIC_BODY \n")
					source.ResponseBodyTruncated = true
				}
				dependency := fmt.Errorf("SYNTHETIC_WRAPPER: %w", source)
				f := &rpcFake{err: dependency}
				c := &Client{rpc: f}
				var err error
				method := "eth_blockNumber"
				switch op {
				case "query":
					_, err = c.BlockNumber(context.Background())
				case "compute":
					method = "eth_call"
					_, err = c.Call(context.Background(), CallRequest{}, Latest())
				case "submit":
					method = "eth_sendRawTransaction"
					_, err = c.SendRawTransaction(context.Background(), rawString[Data](t, "0xabcdef1234"))
				}
				got := assertKind(t, err, tc.kind)
				want := source
				want.Method = method
				want.Cause = dependency
				want.SubmissionUnknown = op == "submit" && tc.kind != apierror.RPC
				if !reflect.DeepEqual(got, want) {
					t.Errorf("raw error fields changed: got %#v; want %#v", got, want)
				}
				var typed dependencyFailure
				if got.Cause != dependency || !errors.Is(err, dependency) || !errors.Is(err, leaf) || !errors.As(err, &typed) || typed != leaf {
					t.Error("complete dependency chain lost")
				}
				if strings.Contains(fmt.Sprintf("%+v %#v", err, got), "0xabcdef1234") {
					t.Error("request-only payload appended")
				}
				equal(t, f.calls, 1)
			})
		}
	}
}

func TestSubmissionUnknownContextAndMalformedHash(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
		reply string
	}{
		{"cancel", context.Canceled, ""}, {"deadline", context.DeadlineExceeded, ""},
		{"invalid-hash", nil, `"0x01"`}, {"null-hash", nil, `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			if tc.cause != nil {
				f.err = fmt.Errorf("SYNTHETIC_CONTEXT: %w", tc.cause)
			}
			got, err := (&Client{rpc: f}).SendRawTransaction(context.Background(), rawString[Data](t, "0x01"))
			kind := apierror.Protocol
			if tc.cause != nil {
				kind = apierror.Transport
			}
			a := assertKind(t, err, kind)
			if !reflect.ValueOf(got).IsZero() || !a.SubmissionUnknown {
				t.Error("failed submission must return zero hash and unknown")
			}
			if tc.cause != nil && (a.Cause != f.err || !errors.Is(err, f.err) || !errors.Is(err, tc.cause)) {
				t.Error("context wrapper normalized or lost")
			}
			assertWire(t, f, "eth_sendRawTransaction", `["0x01"]`, context.Background())
		})
	}
}

func TestSubmissionLocalRejection(t *testing.T) {
	for _, raw := range []string{"", "0x", "0x0", "0xgg", "01"} {
		t.Run(raw, func(t *testing.T) {
			f := &rpcFake{}
			_, e := (&Client{rpc: f}).SendRawTransaction(context.Background(), rawString[Data](t, raw))
			a := assertKind(t, e, apierror.InvalidArgument)
			if a.SubmissionUnknown || f.calls != 0 {
				t.Fatal("local rejection sent or marked unknown")
			}
		})
	}
}
func TestQueryDependencyTextAndRequestIsolation(t *testing.T) {
	const requestOnly = "0x1111222233334444555566667777888899990000"
	dependency := fmt.Errorf("SYNTHETIC_DEPENDENCY: %w", context.Canceled)
	f := &rpcFake{err: dependency}
	_, err := (&Client{rpc: f}).GetBalance(context.Background(), rawString[Address](t, requestOnly), Latest())
	got := assertKind(t, err, apierror.Transport)
	if got.Cause != dependency || !errors.Is(err, dependency) || !errors.Is(err, context.Canceled) {
		t.Error("original dependency lost")
	}
	if !strings.Contains(err.Error(), "SYNTHETIC_DEPENDENCY") {
		t.Error("dependency text lost")
	}
	if strings.Contains(fmt.Sprintf("%v %#v", err, got), requestOnly) {
		t.Error("request-only address appended")
	}
	equal(t, got.Method, "eth_getBalance")
	equal(t, got.SubmissionUnknown, false)
	equal(t, f.calls, 1)
}
func TestCanceledContextIsForwarded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &rpcFake{reply: `"0x10"`}
	v, e := (&Client{rpc: f}).BlockNumber(ctx)
	if e != nil {
		t.Fatal(e)
	}
	equal(t, v, uint64(16))
	equal(t, f.ctx, ctx)
	equal(t, f.calls, 1)
}
