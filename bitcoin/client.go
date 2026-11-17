package bitcoin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
)

type Client struct{ rpc jsonrpc.Caller }

func New(_ Config, rpc jsonrpc.Caller) (*Client, error) {
	if rpc == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	return &Client{rpc: rpc}, nil
}

func (c *Client) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.rpc.Call(ctx, method, params, &raw); err != nil {
		var e apierror.Error
		if !errors.As(err, &e) {
			e.Kind = apierror.Transport
		}
		e.Method, e.Cause = method, err
		e.SubmissionUnknown = method == "sendrawtransaction" && e.Kind != apierror.RPC
		return nil, e
	}
	return bytes.TrimSpace(raw), nil
}

func invalid(method string) error {
	return apierror.Error{Kind: apierror.InvalidArgument, Method: method}
}
func protocol(method string, cause error) error {
	return apierror.Error{Kind: apierror.Protocol, Method: method, Cause: cause, SubmissionUnknown: method == "sendrawtransaction"}
}

func decode(method string, raw json.RawMessage, target any, shape byte) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return protocol(method, err)
	}
	if len(raw) == 0 || raw[0] != shape {
		return protocol(method, nil)
	}
	return nil
}

func validHex(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
func validHash(s string) bool { return len(s) == 64 && validHex(s) }
