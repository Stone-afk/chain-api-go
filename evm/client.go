package evm

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"reflect"
)

type Client struct{ rpc jsonrpc.Caller }

func New(rpc jsonrpc.Caller) (*Client, error) {
	if rpc == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	v := reflect.ValueOf(rpc)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil, apierror.Error{Kind: apierror.Configuration}
		}
	}
	return &Client{rpc: rpc}, nil
}
func invalid() error  { return apierror.Error{Kind: apierror.InvalidArgument} }
func protocol() error { return apierror.Error{Kind: apierror.Protocol} }

// invoke keeps only safe classification and numeric codes from injected errors.
func (c *Client) invoke(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	if ctx == nil {
		return nil, invalid()
	}
	v := reflect.ValueOf(ctx)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return nil, invalid()
		}
	}
	if c == nil || c.rpc == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	var raw json.RawMessage
	if err := c.rpc.Call(ctx, method, params, &raw); err != nil {
		out := apierror.Error{Kind: apierror.Transport, Method: method}.WithContextCause(err)
		var a apierror.Error
		if errors.As(err, &a) {
			switch a.Kind {
			case apierror.Configuration, apierror.InvalidArgument, apierror.Transport, apierror.HTTP, apierror.Protocol, apierror.RPC, apierror.Unsupported, apierror.NetworkMismatch:
				out.Kind = a.Kind
			}
			out.HTTPStatus = a.HTTPStatus
			out.RPCCode = a.RPCCode
			out.HasRPCCode = a.HasRPCCode
		}
		return nil, out
	}
	if !json.Valid(raw) {
		return nil, protocol()
	}
	return raw, nil
}
