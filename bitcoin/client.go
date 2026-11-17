package bitcoin

import (
	"context"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
)

type Client struct{ rpc jsonrpc.Caller }

func New(_ Config, rpc jsonrpc.Caller) (*Client, error) {
	if rpc == nil {
		return nil, apierror.Error{Kind: apierror.Configuration}
	}
	return &Client{rpc: rpc}, nil
}
func (c *Client) GetBlockchainInfo(context.Context) (BlockchainInfo, error) {
	return BlockchainInfo{}, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetBlockCount(context.Context) (int64, error) {
	return 0, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetBlockHash(context.Context, int64) (BlockHash, error) {
	return "", apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetBlock(context.Context, BlockHash) (Block, error) {
	return Block{}, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetTxOut(context.Context, GetTxOutRequest) (TxOut, bool, error) {
	return TxOut{}, false, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetRawTransaction(context.Context, GetRawTransactionRequest) (Transaction, error) {
	return Transaction{}, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) GetMempoolEntry(context.Context, TxID) (MempoolEntry, error) {
	return MempoolEntry{}, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) EstimateSmartFee(context.Context, EstimateSmartFeeRequest) (FeeEstimate, bool, error) {
	return FeeEstimate{}, false, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) TestMempoolAccept(context.Context, RawTransaction) (MempoolAcceptResult, error) {
	return MempoolAcceptResult{}, apierror.Error{Kind: apierror.Unsupported}
}
func (c *Client) SendRawTransaction(context.Context, RawTransaction) (TxID, error) {
	return "", apierror.Error{Kind: apierror.Unsupported}
}
