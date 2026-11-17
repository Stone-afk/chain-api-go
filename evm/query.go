package evm

import (
	"context"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

func (c *Client) ChainID(ctx context.Context) (*big.Int, error)   { return c.native.ChainID(ctx) }
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) { return c.native.BlockNumber(ctx) }
func (c *Client) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return c.native.BlockByNumber(ctx, number)
}
func (c *Client) BalanceAt(ctx context.Context, account common.Address, number *big.Int) (*big.Int, error) {
	return c.native.BalanceAt(ctx, account, number)
}
func (c *Client) NonceAt(ctx context.Context, account common.Address, number *big.Int) (uint64, error) {
	return c.native.NonceAt(ctx, account, number)
}
func (c *Client) CodeAt(ctx context.Context, account common.Address, number *big.Int) ([]byte, error) {
	return c.native.CodeAt(ctx, account, number)
}
func (c *Client) TransactionByHash(ctx context.Context, hash common.Hash) (*types.Transaction, bool, error) {
	return c.native.TransactionByHash(ctx, hash)
}
func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	return c.native.TransactionReceipt(ctx, hash)
}
func (c *Client) FilterLogs(ctx context.Context, query geth.FilterQuery) ([]types.Log, error) {
	return c.native.FilterLogs(ctx, query)
}
