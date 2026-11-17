package evm

import (
	"context"
	geth "github.com/ethereum/go-ethereum"
	"math/big"
)

func (c *Client) CallContract(ctx context.Context, msg geth.CallMsg, number *big.Int) ([]byte, error) {
	return c.native.CallContract(ctx, msg, number)
}
func (c *Client) EstimateGas(ctx context.Context, msg geth.CallMsg) (uint64, error) {
	return c.native.EstimateGas(ctx, msg)
}
func (c *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return c.native.SuggestGasPrice(ctx)
}
func (c *Client) FeeHistory(ctx context.Context, count uint64, last *big.Int, percentiles []float64) (*geth.FeeHistory, error) {
	return c.native.FeeHistory(ctx, count, last, percentiles)
}
