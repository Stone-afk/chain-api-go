package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
)

type Client struct{ rpc jsonrpc.Caller }

func New(rpc jsonrpc.Caller) (*Client, error) {
	if rpc == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	return &Client{rpc: rpc}, nil
}
func invalid() error { return apierror.Error{Kind: apierror.InvalidArgument} }
func (c *Client) invoke(ctx context.Context, method string, params []any, submission bool) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.rpc.Call(ctx, method, params, &raw); err != nil {
		out := apierror.Error{Kind: apierror.Transport}
		var source apierror.Error
		if errors.As(err, &source) {
			out = source
		}
		out.Method = method
		out.Cause = err
		out.SubmissionUnknown = submission && out.Kind != apierror.RPC
		return nil, out
	}
	return raw, nil
}
func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

// decode delegates DTO semantics to encoding/json and preserves decode errors.
func decode[T any](raw json.RawMessage) (T, error) {
	var value T
	err := json.Unmarshal(raw, &value)
	if err == nil && isNull(raw) {
		err = errors.New("unexpected null result")
	}
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}
func requestValue[T any](c *Client, ctx context.Context, method string, params []any, parse func(json.RawMessage) (T, error)) (T, error) {
	var zero T
	raw, err := c.invoke(ctx, method, params, false)
	if err != nil {
		return zero, err
	}
	value, err := parse(raw)
	if err != nil {
		return zero, apierror.Error{Kind: apierror.Protocol, Method: method, Cause: err}
	}
	return value, nil
}
func requestNullable[T any](c *Client, ctx context.Context, method string, params []any) (T, bool, error) {
	var zero T
	raw, err := c.invoke(ctx, method, params, false)
	if err != nil {
		return zero, false, err
	}
	if isNull(raw) {
		return zero, false, nil
	}
	value, err := decode[T](raw)
	if err != nil {
		return zero, false, apierror.Error{Kind: apierror.Protocol, Method: method, Cause: err}
	}
	return value, true, nil
}
