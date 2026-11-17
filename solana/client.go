package solana

import (
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

type Config struct{}
type Client struct{ rpc *srpc.Client }

func New(_ Config, caller jsonrpc.Caller) (*Client, error) {
	if caller == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	return &Client{rpc: srpc.NewWithCustomRPCClient(rpcBridge{caller})}, nil
}
