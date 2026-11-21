package solana

import (
	"context"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

func (c *Client) GetGenesisHash(ctx context.Context) (ns.Hash, error) {
	return c.rpc.GetGenesisHash(ctx)
}
func (c *Client) GetSlotWithOpts(ctx context.Context, opts *srpc.GetSlotOpts) (uint64, error) {
	return c.rpc.GetSlotWithOpts(ctx, opts)
}
func (c *Client) GetBlockWithOpts(ctx context.Context, slot uint64, opts *srpc.GetBlockOpts) (*srpc.GetBlockResult, error) {
	return c.rpc.GetBlockWithOpts(ctx, slot, opts)
}
func (c *Client) GetBlocks(ctx context.Context, startSlot uint64, endSlot *uint64, commitment srpc.CommitmentType) (srpc.BlocksResult, error) {
	return c.rpc.GetBlocks(ctx, startSlot, endSlot, commitment)
}
func (c *Client) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	return c.rpc.GetFirstAvailableBlock(ctx)
}
func (c *Client) GetBalanceWithOpts(ctx context.Context, key ns.PublicKey, opts *srpc.GetBalanceOpts) (*srpc.GetBalanceResult, error) {
	return c.rpc.GetBalanceWithOpts(ctx, key, opts)
}
func (c *Client) GetAccountInfoWithOpts(ctx context.Context, key ns.PublicKey, opts *srpc.GetAccountInfoOpts) (*srpc.GetAccountInfoResult, error) {
	return c.rpc.GetAccountInfoWithOpts(ctx, key, opts)
}
func (c *Client) GetLatestBlockhashWithOpts(ctx context.Context, opts *srpc.GetLatestBlockhashOpts) (*srpc.GetLatestBlockhashResult, error) {
	return c.rpc.GetLatestBlockhashWithOpts(ctx, opts)
}
func (c *Client) GetSignatureStatuses(ctx context.Context, history bool, signatures ...ns.Signature) (*srpc.GetSignatureStatusesResult, error) {
	return c.rpc.GetSignatureStatuses(ctx, history, signatures...)
}
func (c *Client) GetTransaction(ctx context.Context, signature ns.Signature, opts *srpc.GetTransactionOpts) (*srpc.GetTransactionResult, error) {
	return c.rpc.GetTransaction(ctx, signature, opts)
}
