package evm

import (
	"context"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"math/big"
)

type Client struct{ rpc jsonrpc.Caller }

func New(jsonrpc.Caller) (*Client, error) { return nil, unsupported() }

func unsupported() error                                    { return apierror.Error{Kind: apierror.Unsupported} }
func (*Client) ChainID(context.Context) (big.Int, error)    { return big.Int{}, unsupported() }
func (*Client) BlockNumber(context.Context) (uint64, error) { return 0, unsupported() }
func (*Client) GetBlock(context.Context, BlockRef) (Block, bool, error) {
	return Block{}, false, unsupported()
}
func (*Client) GetBalance(context.Context, Address, BlockRef) (big.Int, error) {
	return big.Int{}, unsupported()
}
func (*Client) GetTransactionCount(context.Context, Address, BlockRef) (uint64, error) {
	return 0, unsupported()
}
func (*Client) GetCode(context.Context, Address, BlockRef) (Data, error) { return nil, unsupported() }
func (*Client) GetTransaction(context.Context, Hash) (Transaction, bool, error) {
	return Transaction{}, false, unsupported()
}
func (*Client) GetTransactionReceipt(context.Context, Hash) (Receipt, bool, error) {
	return Receipt{}, false, unsupported()
}
func (*Client) GetLogs(context.Context, LogFilter) ([]Log, error)         { return nil, unsupported() }
func (*Client) Call(context.Context, CallRequest, BlockRef) (Data, error) { return nil, unsupported() }
func (*Client) EstimateGas(context.Context, CallRequest) (uint64, error)  { return 0, unsupported() }
func (*Client) GasPrice(context.Context) (big.Int, error)                 { return big.Int{}, unsupported() }
func (*Client) FeeHistory(context.Context, uint64, BlockRef, []float64) (FeeHistoryResult, error) {
	return FeeHistoryResult{}, unsupported()
}
func (*Client) SendRawTransaction(context.Context, Data) (Hash, error) { return Hash{}, unsupported() }
