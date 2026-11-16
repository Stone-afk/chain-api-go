package ethereum

import (
	"context"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/evm"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"math/big"
)

type Config struct{ ExpectedChainID big.Int }
type Client struct{ *evm.Client }

func New(Config, jsonrpc.Caller) (*Client, error) {
	return nil, apierror.Error{Kind: apierror.Unsupported}
}
func (*Client) ValidateNetwork(context.Context) error {
	return apierror.Error{Kind: apierror.Unsupported}
}
