package tron

import (
	"context"
	"encoding/json"
)

// query checks only the nullable identity boundary; DTO decoding stays standard.
func query[T any](ctx context.Context, c *Client, method string, request any, identity string) (T, bool, error) {
	var zero T
	raw, fields, err := c.post(ctx, method, request, false)
	if err != nil {
		return zero, false, err
	}
	if identity != "" {
		if len(fields) == 0 {
			return zero, false, nil
		}
		if _, ok := fields[identity]; !ok {
			return zero, false, protocol(method, nil, false)
		}
	}
	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		return zero, false, protocol(method, err, false)
	}
	return result, true, nil
}

func (c *Client) GetNodeInfo(ctx context.Context) (NodeInfo, error) {
	const method = "/wallet/getnodeinfo"
	result, _, err := query[NodeInfo](ctx, c, method, struct{}{}, "")
	return result, err
}

func (c *Client) GetNowBlock(ctx context.Context) (Block, bool, error) {
	const method = "/wallet/getnowblock"
	return query[Block](ctx, c, method, struct{}{}, "blockID")
}

func (c *Client) GetBlockByNum(ctx context.Context, height int64) (Block, bool, error) {
	const method = "/wallet/getblockbynum"
	if height < 0 {
		return Block{}, false, invalid(method)
	}
	return query[Block](ctx, c, method, map[string]any{"num": height}, "blockID")
}

func (c *Client) GetAccount(ctx context.Context, req GetAccountRequest) (Account, bool, error) {
	const method = "/wallet/getaccount"
	if !validAddress(req.Address, req.Visible) {
		return Account{}, false, invalid(method)
	}
	return query[Account](ctx, c, method, map[string]any{"address": req.Address, "visible": req.Visible}, "address")
}

func (c *Client) GetAccountResource(ctx context.Context, req GetAccountRequest) (AccountResource, error) {
	const method = "/wallet/getaccountresource"
	if !validAddress(req.Address, req.Visible) {
		return AccountResource{}, invalid(method)
	}
	result, _, err := query[AccountResource](ctx, c, method, map[string]any{"address": req.Address, "visible": req.Visible}, "")
	return result, err
}

func (c *Client) GetChainParameters(ctx context.Context) (ChainParameters, error) {
	const method = "/wallet/getchainparameters"
	result, _, err := query[ChainParameters](ctx, c, method, struct{}{}, "")
	return result, err
}

func (c *Client) GetTransactionByID(ctx context.Context, id TxID) (Transaction, bool, error) {
	const method = "/wallet/gettransactionbyid"
	if !validTxID(string(id)) {
		return Transaction{}, false, invalid(method)
	}
	return query[Transaction](ctx, c, method, map[string]any{"value": id}, "txID")
}

func (c *Client) GetTransactionInfoByID(ctx context.Context, id TxID) (TransactionInfo, bool, error) {
	const method = "/wallet/gettransactioninfobyid"
	if !validTxID(string(id)) {
		return TransactionInfo{}, false, invalid(method)
	}
	return query[TransactionInfo](ctx, c, method, map[string]any{"value": id}, "id")
}

func (c *Client) GetSolidifiedAccount(ctx context.Context, req GetAccountRequest) (Account, bool, error) {
	const method = "/walletsolidity/getaccount"
	if !validAddress(req.Address, req.Visible) {
		return Account{}, false, invalid(method)
	}
	return query[Account](ctx, c, method, map[string]any{"address": req.Address, "visible": req.Visible}, "address")
}

func (c *Client) GetSolidifiedNowBlock(ctx context.Context) (Block, bool, error) {
	const method = "/walletsolidity/getnowblock"
	return query[Block](ctx, c, method, struct{}{}, "blockID")
}

func (c *Client) GetSolidifiedBlockByNum(ctx context.Context, height int64) (Block, bool, error) {
	const method = "/walletsolidity/getblockbynum"
	if height < 0 {
		return Block{}, false, invalid(method)
	}
	return query[Block](ctx, c, method, map[string]any{"num": height}, "blockID")
}

func (c *Client) GetSolidifiedTransactionByID(ctx context.Context, id TxID) (Transaction, bool, error) {
	const method = "/walletsolidity/gettransactionbyid"
	if !validTxID(string(id)) {
		return Transaction{}, false, invalid(method)
	}
	return query[Transaction](ctx, c, method, map[string]any{"value": id}, "txID")
}

func (c *Client) GetSolidifiedTransactionInfoByID(ctx context.Context, id TxID) (TransactionInfo, bool, error) {
	const method = "/walletsolidity/gettransactioninfobyid"
	if !validTxID(string(id)) {
		return TransactionInfo{}, false, invalid(method)
	}
	return query[TransactionInfo](ctx, c, method, map[string]any{"value": id}, "id")
}
