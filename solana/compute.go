package solana

import (
	"context"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

func (c *Client) GetFeeForMessageWithOpts(ctx context.Context, message string, opts *srpc.GetFeeForMessageOpts) (*srpc.GetFeeForMessageResult, error) {
	return c.rpc.GetFeeForMessageWithOpts(ctx, message, opts)
}
func (c *Client) SimulateRawTransactionWithOpts(ctx context.Context, data []byte, opts *srpc.SimulateTransactionOpts) (*srpc.SimulateTransactionResponse, error) {
	return c.rpc.SimulateRawTransactionWithOpts(ctx, data, opts)
}
