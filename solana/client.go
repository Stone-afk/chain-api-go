package solana

import (
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

type Client struct{ rpc *srpc.Client }

func New(caller jsonrpc.Caller) (*Client, error) {
	return &Client{rpc: srpc.NewWithCustomRPCClient(callerRPCClient{caller})}, nil
}
