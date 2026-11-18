package ethereum

import (
	"context"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/evm"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"math/big"
)

type Config struct {
	RawURL          string
	ExpectedChainID big.Int
}
type Client struct {
	*evm.Client
	expectedChainID big.Int
}

func New(cfg Config, rpc jsonrpc.Caller) (*Client, error) {
	if cfg.ExpectedChainID.Sign() <= 0 {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	base, err := evm.New(cfg.RawURL, rpc)
	if err != nil {
		return nil, err
	}
	c := &Client{Client: base}
	c.expectedChainID.Set(&cfg.ExpectedChainID)
	return c, nil
}
func (c *Client) ValidateNetwork(ctx context.Context) error {
	actual, err := c.ChainID(ctx)
	if err != nil {
		return err
	}
	if actual.Cmp(&c.expectedChainID) != 0 {
		return apierror.Error{Kind: apierror.NetworkMismatch}
	}
	return nil
}
