package evm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
)

func parseQuantity(raw json.RawMessage) (big.Int, error) {
	s, err := decode[string](raw)
	if err != nil {
		return big.Int{}, err
	}
	if !validQuantity(s) {
		return big.Int{}, errors.New("invalid quantity result")
	}
	var value big.Int
	value.SetString(s[2:], 16)
	return value, nil
}
func parseUint(raw json.RawMessage) (uint64, error) {
	s, err := decode[string](raw)
	if err != nil {
		return 0, err
	}
	if !validQuantity(s) {
		return 0, errors.New("invalid quantity result")
	}
	return strconv.ParseUint(s[2:], 16, 64)
}
func (c *Client) ChainID(ctx context.Context) (big.Int, error) {
	return requestValue(c, ctx, "eth_chainId", []any{}, parseQuantity)
}
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return requestValue(c, ctx, "eth_blockNumber", []any{}, parseUint)
}
func (c *Client) GetBalance(ctx context.Context, address Address, at BlockRef) (big.Int, error) {
	if at.wire == "" || !validHex(string(address), 20) {
		return big.Int{}, invalid()
	}
	return requestValue(c, ctx, "eth_getBalance", []any{address, at}, parseQuantity)
}
func (c *Client) GetTransactionCount(ctx context.Context, address Address, at BlockRef) (uint64, error) {
	if at.wire == "" || !validHex(string(address), 20) {
		return 0, invalid()
	}
	return requestValue(c, ctx, "eth_getTransactionCount", []any{address, at}, parseUint)
}
func (c *Client) GetCode(ctx context.Context, address Address, at BlockRef) (Data, error) {
	if at.wire == "" || !validHex(string(address), 20) {
		return "", invalid()
	}
	return requestValue(c, ctx, "eth_getCode", []any{address, at}, decode[Data])
}

// GetBlock returns present=false for a node's null result.
func (c *Client) GetBlock(ctx context.Context, at BlockRef) (Block, bool, error) {
	if at.wire == "" {
		return Block{}, false, invalid()
	}
	return requestNullable[Block](c, ctx, "eth_getBlockByNumber", []any{at, false})
}

// GetTransaction returns present=false for a node's null result.
func (c *Client) GetTransaction(ctx context.Context, hash Hash) (Transaction, bool, error) {
	if !validHex(string(hash), 32) {
		return Transaction{}, false, invalid()
	}
	return requestNullable[Transaction](c, ctx, "eth_getTransactionByHash", []any{hash})
}

// GetTransactionReceipt returns present=false for a node's null result.
func (c *Client) GetTransactionReceipt(ctx context.Context, hash Hash) (Receipt, bool, error) {
	if !validHex(string(hash), 32) {
		return Receipt{}, false, invalid()
	}
	return requestNullable[Receipt](c, ctx, "eth_getTransactionReceipt", []any{hash})
}
func (c *Client) GetLogs(ctx context.Context, filter LogFilter) ([]Log, error) {
	if filter.FromBlock.wire == "" || filter.ToBlock.wire == "" || len(filter.Topics) > 4 {
		return nil, invalid()
	}
	for _, address := range filter.Addresses {
		if !validHex(address, 20) {
			return nil, invalid()
		}
	}
	topics := make([]any, len(filter.Topics))
	for i, group := range filter.Topics {
		for _, topic := range group {
			if !validHex(topic, 32) {
				return nil, invalid()
			}
		}
		if len(group) > 0 {
			topics[i] = group
		}
	}
	params := map[string]any{"fromBlock": filter.FromBlock, "toBlock": filter.ToBlock, "topics": topics}
	if len(filter.Addresses) > 0 {
		params["address"] = filter.Addresses
	}
	return requestValue(c, ctx, "eth_getLogs", []any{params}, decode[[]Log])
}
