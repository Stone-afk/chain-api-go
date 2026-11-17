package evm

import (
	"context"
	"net/http"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	native *ethclient.Client
	rpc    *rpc.Client
}

func New(caller jsonrpc.Caller) (*Client, error) {
	if caller == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	client, err := rpc.DialOptions(context.Background(), "http://chain-api-go.invalid", rpc.WithHTTPClient(&http.Client{Transport: bridge{caller: caller}}))
	if err != nil {
		return nil, err
	}
	return &Client{native: ethclient.NewClient(client), rpc: client}, nil
}
