package bitcoin

import (
	"context"
	"encoding/json"
)

func (c *Client) GetBlockchainInfo(ctx context.Context) (BlockchainInfo, error) {
	const method = "getblockchaininfo"
	raw, err := c.call(ctx, method, []any{})
	if err != nil {
		return BlockchainInfo{}, err
	}
	var v BlockchainInfo
	if err = decode(method, raw, &v, '{'); err != nil {
		return BlockchainInfo{}, err
	}
	return v, nil
}

func (c *Client) GetBlockCount(ctx context.Context) (int64, error) {
	const method = "getblockcount"
	raw, err := c.call(ctx, method, []any{})
	if err != nil {
		return 0, err
	}
	var v int64
	if err = json.Unmarshal(raw, &v); err != nil {
		return 0, protocol(method, err)
	}
	if string(raw) == "null" || v < 0 {
		return 0, protocol(method, nil)
	}
	return v, nil
}

func (c *Client) GetBlockHash(ctx context.Context, height int64) (BlockHash, error) {
	const method = "getblockhash"
	if height < 0 {
		return "", invalid(method)
	}
	raw, err := c.call(ctx, method, []any{height})
	if err != nil {
		return "", err
	}
	var v BlockHash
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", protocol(method, err)
	}
	if !validHash(string(v)) {
		return "", protocol(method, nil)
	}
	return v, nil
}

func (c *Client) GetBlock(ctx context.Context, hash BlockHash) (Block, error) {
	const method = "getblock"
	if !validHash(string(hash)) {
		return Block{}, invalid(method)
	}
	raw, err := c.call(ctx, method, []any{hash, 1})
	if err != nil {
		return Block{}, err
	}
	var v Block
	if err = decode(method, raw, &v, '{'); err != nil {
		return Block{}, err
	}
	return v, nil
}

// GetTxOut returns found=false only for the node's top-level null result.
func (c *Client) GetTxOut(ctx context.Context, req GetTxOutRequest) (TxOut, bool, error) {
	const method = "gettxout"
	if !validHash(string(req.TxID)) {
		return TxOut{}, false, invalid(method)
	}
	params := []any{req.TxID, req.Vout}
	if req.HasIncludeMempool {
		params = append(params, req.IncludeMempool)
	}
	raw, err := c.call(ctx, method, params)
	if err != nil {
		return TxOut{}, false, err
	}
	if string(raw) == "null" {
		return TxOut{}, false, nil
	}
	var w struct {
		TxOut
		Value json.RawMessage `json:"value"`
	}
	if err = decode(method, raw, &w, '{'); err != nil {
		return TxOut{}, false, err
	}
	amount, ok := btcAmount(w.Value)
	if !ok {
		return TxOut{}, false, protocol(method, nil)
	}
	w.TxOut.Value = Satoshi(amount)
	return w.TxOut, true, nil
}

// GetRawTransaction preserves node errors for unavailable history or txindex.
func (c *Client) GetRawTransaction(ctx context.Context, req GetRawTransactionRequest) (Transaction, error) {
	const method = "getrawtransaction"
	if !validHash(string(req.TxID)) || req.HasBlockHash && !validHash(string(req.BlockHash)) {
		return Transaction{}, invalid(method)
	}
	params := []any{req.TxID, 1}
	if req.HasBlockHash {
		params = append(params, req.BlockHash)
	}
	raw, err := c.call(ctx, method, params)
	if err != nil {
		return Transaction{}, err
	}
	var w struct {
		Transaction
		Vout []struct {
			TxOutput
			Value json.RawMessage `json:"value"`
		} `json:"vout"`
	}
	if err = decode(method, raw, &w, '{'); err != nil {
		return Transaction{}, err
	}
	if w.Vout != nil {
		w.Transaction.Vout = make([]TxOutput, len(w.Vout))
	}
	for i, out := range w.Vout {
		amount, ok := btcAmount(out.Value)
		if !ok {
			return Transaction{}, protocol(method, nil)
		}
		out.TxOutput.Value = Satoshi(amount)
		w.Transaction.Vout[i] = out.TxOutput
	}
	return w.Transaction, nil
}

func (c *Client) GetMempoolEntry(ctx context.Context, txid TxID) (MempoolEntry, error) {
	const method = "getmempoolentry"
	if !validHash(string(txid)) {
		return MempoolEntry{}, invalid(method)
	}
	raw, err := c.call(ctx, method, []any{txid})
	if err != nil {
		return MempoolEntry{}, err
	}
	var w struct {
		MempoolEntry
		Fees struct {
			Base json.RawMessage `json:"base"`
		} `json:"fees"`
		BaseFee json.RawMessage `json:"base_fee"`
	}
	if err = decode(method, raw, &w, '{'); err != nil {
		return MempoolEntry{}, err
	}
	amount, ok := btcAmount(w.Fees.Base)
	if !ok {
		return MempoolEntry{}, protocol(method, nil)
	}
	w.MempoolEntry.BaseFee = Satoshi(amount)
	return w.MempoolEntry, nil
}
