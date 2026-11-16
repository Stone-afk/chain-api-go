package evm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func TestSubmissionSingleUnchangedPayload(t *testing.T) {
	payload := Data{0, 1, 128, 255}
	f := &rpcFake{reply: H3}
	ctx := context.Background()
	got, e := (&Client{rpc: f}).SendRawTransaction(ctx, payload)
	if e != nil {
		t.Fatalf("missing submit behavior: %v", e)
	}
	if got != h3 || !reflect.DeepEqual(payload, Data{0, 1, 128, 255}) {
		t.Fatal("hash or original payload changed")
	}
	assertWire(t, f, "eth_sendRawTransaction", `["0x000180ff"]`, ctx)
}

func TestSubmissionUnknownAndSafeErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		reply   string
		kind    apierror.Kind
		unknown bool
		cause   error
	}{
		{"transport", apierror.Error{Kind: apierror.Transport}, "", apierror.Transport, true, nil},
		{"http", apierror.Error{Kind: apierror.HTTP, HTTPStatus: 503}, "", apierror.HTTP, true, nil},
		{"protocol", apierror.Error{Kind: apierror.Protocol}, "", apierror.Protocol, true, nil},
		{"rpc-rejection", apierror.Error{Kind: apierror.RPC, RPCCode: 0, HasRPCCode: true}, "", apierror.RPC, false, nil},
		{"canceled", fmt.Errorf("SYNTHETIC_SECRET: %w", context.Canceled), "", apierror.Transport, true, context.Canceled},
		{"deadline", fmt.Errorf("SYNTHETIC_SECRET: %w", context.DeadlineExceeded), "", apierror.Transport, true, context.DeadlineExceeded},
		{"invalid-hash", nil, `"0x01"`, apierror.Protocol, true, nil},
		{"null-hash", nil, `null`, apierror.Protocol, true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{err: tc.err, reply: tc.reply}
			got, e := (&Client{rpc: f}).SendRawTransaction(context.Background(), Data{1})
			a := assertKind(t, e, tc.kind)
			if got != (Hash{}) || a.SubmissionUnknown != tc.unknown || f.calls != 1 {
				t.Errorf("hash=%v unknown=%v calls=%d", got, a.SubmissionUnknown, f.calls)
			}
			if tc.cause != nil && !errors.Is(e, tc.cause) {
				t.Error("context sentinel lost")
			}
			if tc.kind == apierror.RPC && (!a.HasRPCCode || a.RPCCode != 0) {
				t.Error("zero node RPC code lost")
			}
			if strings.Contains(fmt.Sprintf("%v %+v %#v", e, e, e), "SYNTHETIC_SECRET") {
				t.Error("unsafe cause retained")
			}
		})
	}
}

func TestSubmissionLocalRejection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		payload Data
	}{
		{"empty", context.Background(), nil}, {"nil-context", nil, Data{1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{}
			_, e := (&Client{rpc: f}).SendRawTransaction(tc.ctx, tc.payload)
			a := assertKind(t, e, apierror.InvalidArgument)
			if a.SubmissionUnknown || f.calls != 0 {
				t.Fatal("local rejection sent or marked unknown")
			}
		})
	}
}
