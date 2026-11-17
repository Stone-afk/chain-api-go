package evm

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestNativeEVMSubmissionValidation(t *testing.T) {
	for _, payload := range []string{"", "0x", "00", "0X00", "0x0", "0xgg", " 0xaa", "0xaa\n"} {
		t.Run(payload, func(t *testing.T) {
			f := &rpcFake{}
			c := nativeClient(t, f)
			id, err := c.SendRawTransaction(context.Background(), payload)
			a := assertKind(t, err, apierror.InvalidArgument)
			equal(t, a.SubmissionUnknown, false)
			equal(t, id, common.Hash{})
			equal(t, len(f.calls), 0)
		})
	}
}
func TestNativeEVMSubmissionOnce(t *testing.T) {
	leaf := errors.New("SYNTHETIC_CAUSE")
	source := apierror.Error{Kind: apierror.RPC, Method: "eth_sendRawTransaction", HasRPCCode: true, RPCCode: -32000, Message: "  SYNTHETIC_NODE\n", Data: json.RawMessage(`{"n":9007199254740993}`), ResponseBody: []byte("SYNTHETIC_BODY"), ResponseBodyTruncated: true, Cause: leaf}
	for _, tc := range []struct {
		name, reply string
		err         error
		kind        apierror.Kind
		unknown     bool
	}{
		{"accepted", `"` + h1 + `"`, nil, "", false},
		{"node-rejection", "", source, apierror.RPC, false},
		{"transport", "", apierror.Error{Kind: apierror.Transport, Cause: leaf}, apierror.Transport, true},
		{"http", "", apierror.Error{Kind: apierror.HTTP, HTTPStatus: 500, Cause: leaf}, apierror.HTTP, true},
		{"untyped-cause", "", leaf, apierror.Transport, true},
		{"zero", `"0x0000000000000000000000000000000000000000000000000000000000000000"`, nil, apierror.Protocol, true},
		{"bad-hash", `"0x12"`, nil, apierror.Protocol, true},
		{"null", "null", nil, apierror.Protocol, true},
		{"wrong-type", "123", nil, apierror.Protocol, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply, err: tc.err}
			c := nativeClient(t, f)
			id, err := c.SendRawTransaction(context.Background(), "0x00ABcd")
			assertWire(t, f, "eth_sendRawTransaction", `["0x00ABcd"]`, context.Background())
			if tc.kind == "" {
				if err != nil {
					t.Fatal(err)
				}
				equal(t, id, common.HexToHash(h1))
				return
			}
			a := assertKind(t, err, tc.kind)
			equal(t, a.SubmissionUnknown, tc.unknown)
			equal(t, id, common.Hash{})
			if tc.err != nil && !errors.Is(err, leaf) {
				t.Fatal("NS-07: submission Cause lost")
			}
			if tc.name == "node-rejection" {
				equal(t, a.RPCCode, source.RPCCode)
				equal(t, a.HasRPCCode, true)
				equal(t, a.Message, source.Message)
				equal(t, a.Data, source.Data)
				equal(t, a.ResponseBody, source.ResponseBody)
				equal(t, a.ResponseBodyTruncated, true)
			}
		})
	}
}
