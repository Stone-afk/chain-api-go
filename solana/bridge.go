package solana

import (
	"context"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	sjsonrpc "github.com/solana-foundation/solana-go/v2/rpc/jsonrpc"
	"net/http"
)

type rpcBridge struct{ caller jsonrpc.Caller }
type callerError struct{ cause error }

func (e callerError) Error() string { return e.cause.Error() }
func (e callerError) Unwrap() error { return e.cause }
func (b rpcBridge) CallForInto(ctx context.Context, out any, method string, params []any) error {
	if err := b.caller.Call(ctx, method, params, out); err != nil {
		return callerError{err}
	}
	return nil
}
func (rpcBridge) CallWithCallback(context.Context, string, []any, func(*http.Request, *http.Response) error) error {
	return apierror.Error{Kind: apierror.Unsupported}
}
func (rpcBridge) CallBatch(context.Context, sjsonrpc.RPCRequests) (sjsonrpc.RPCResponses, error) {
	return nil, apierror.Error{Kind: apierror.Unsupported}
}
