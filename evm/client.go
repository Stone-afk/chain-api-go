package evm

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Client struct {
	native *ethclient.Client
	rpc    *rpc.Client
}

func New(rawURL string, caller jsonrpc.Caller) (*Client, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Fragment != "" {
		return nil, apierror.Error{Kind: apierror.Configuration, Cause: err}
	}
	client, err := rpc.DialOptions(context.Background(), rawURL, rpc.WithHTTPClient(&http.Client{Transport: callerRoundTripper{caller: caller}}))
	if err != nil {
		return nil, err
	}
	return &Client{native: ethclient.NewClient(client), rpc: client}, nil
}
