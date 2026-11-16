package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"strings"
)

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }
func wireString(raw json.RawMessage) (string, error) {
	var s string
	if isNull(raw) || json.Unmarshal(raw, &s) != nil {
		return "", protocol()
	}
	return s, nil
}
func parseQuantity(raw json.RawMessage) (big.Int, error) {
	s, e := wireString(raw)
	if e != nil || len(s) < 3 || !strings.HasPrefix(s, "0x") || (len(s) > 3 && s[2] == '0') {
		return big.Int{}, protocol()
	}
	for _, c := range s[2:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return big.Int{}, protocol()
		}
	}
	var n big.Int
	if _, ok := n.SetString(s[2:], 16); !ok {
		return big.Int{}, protocol()
	}
	return n, nil
}
func parseUint(raw json.RawMessage) (uint64, error) {
	n, e := parseQuantity(raw)
	if e != nil || !n.IsUint64() {
		return 0, protocol()
	}
	return n.Uint64(), nil
}
func parseData(raw json.RawMessage) (Data, error) {
	s, e := wireString(raw)
	if e != nil {
		return nil, e
	}
	v, e := ParseData(s)
	if e != nil {
		return nil, protocol()
	}
	return v, nil
}
func parseAddress(raw json.RawMessage) (Address, error) {
	s, e := wireString(raw)
	if e != nil {
		return Address{}, e
	}
	v, e := ParseAddress(s)
	if e != nil {
		return Address{}, protocol()
	}
	return v, nil
}
func parseHash(raw json.RawMessage) (Hash, error) {
	s, e := wireString(raw)
	if e != nil {
		return Hash{}, e
	}
	v, e := ParseHash(s)
	if e != nil {
		return Hash{}, protocol()
	}
	return v, nil
}
func parseBloom(raw json.RawMessage) ([256]byte, error) {
	d, e := parseData(raw)
	if e != nil || len(d) != 256 {
		return [256]byte{}, protocol()
	}
	return [256]byte(d), nil
}
func parseBool(raw json.RawMessage) (bool, error) {
	var b bool
	if isNull(raw) || json.Unmarshal(raw, &b) != nil {
		return false, protocol()
	}
	return b, nil
}
func parseArray[T any](raw json.RawMessage, parse func(json.RawMessage) (T, error)) ([]T, error) {
	var items []json.RawMessage
	if isNull(raw) || json.Unmarshal(raw, &items) != nil {
		return nil, protocol()
	}
	out := make([]T, len(items))
	for i, item := range items {
		v, e := parse(item)
		if e != nil {
			return nil, protocol()
		}
		out[i] = v
	}
	return out, nil
}
func parseHashes(raw json.RawMessage) ([]Hash, error) { return parseArray(raw, parseHash) }
func parseLogs(raw json.RawMessage) ([]Log, error)    { return parseArray(raw, parseLog) }
func object(raw json.RawMessage, required string) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, protocol()
	}
	m := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err = decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil, protocol()
		}
		if _, duplicate := m[key]; duplicate {
			return nil, protocol()
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, protocol()
		}
		m[key] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, protocol()
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, protocol()
	}
	for _, key := range strings.Fields(required) {
		if _, ok := m[key]; !ok {
			return nil, protocol()
		}
	}
	return m, nil
}

// A field may be absent only after the object's required-key check. Null is
// accepted solely at the native nullable fields selected by each DTO parser.
func field[T any](m map[string]json.RawMessage, key string, parse func(json.RawMessage) (T, error), nullable bool) (T, bool, error) {
	var zero T
	raw, ok := m[key]
	if !ok {
		return zero, false, nil
	}
	if isNull(raw) && nullable {
		return zero, false, nil
	}
	v, e := parse(raw)
	if e != nil {
		return zero, false, protocol()
	}
	return v, true, nil
}
func parseBlock(raw json.RawMessage) (Block, error) {
	m, e := object(raw, "number hash parentHash timestamp gasLimit gasUsed transactions")
	if e != nil {
		return Block{}, e
	}
	var v Block
	v.Number, v.HasNumber, e = field(m, "number", parseUint, true)
	if e != nil {
		return Block{}, e
	}
	v.Hash, v.HasHash, e = field(m, "hash", parseHash, true)
	if e != nil {
		return Block{}, e
	}
	v.ParentHash, _, e = field(m, "parentHash", parseHash, false)
	if e != nil {
		return Block{}, e
	}
	v.Timestamp, _, e = field(m, "timestamp", parseUint, false)
	if e != nil {
		return Block{}, e
	}
	v.GasLimit, _, e = field(m, "gasLimit", parseUint, false)
	if e != nil {
		return Block{}, e
	}
	v.GasUsed, _, e = field(m, "gasUsed", parseUint, false)
	if e != nil {
		return Block{}, e
	}
	v.Transactions, _, e = field(m, "transactions", parseHashes, false)
	if e != nil {
		return Block{}, e
	}
	v.BaseFeePerGas, v.HasBaseFeePerGas, e = field(m, "baseFeePerGas", parseQuantity, false)
	if e != nil {
		return Block{}, e
	}
	if v.HasNumber != v.HasHash {
		return Block{}, protocol()
	}
	return v, nil
}
func parseTransaction(raw json.RawMessage) (Transaction, error) {
	m, e := object(raw, "hash from to nonce gas value input blockHash blockNumber transactionIndex")
	if e != nil {
		return Transaction{}, e
	}
	var v Transaction
	v.Hash, _, e = field(m, "hash", parseHash, false)
	if e != nil {
		return Transaction{}, e
	}
	v.From, _, e = field(m, "from", parseAddress, false)
	if e != nil {
		return Transaction{}, e
	}
	v.To, v.HasTo, e = field(m, "to", parseAddress, true)
	if e != nil {
		return Transaction{}, e
	}
	v.Nonce, _, e = field(m, "nonce", parseUint, false)
	if e != nil {
		return Transaction{}, e
	}
	v.Gas, _, e = field(m, "gas", parseUint, false)
	if e != nil {
		return Transaction{}, e
	}
	v.Value, _, e = field(m, "value", parseQuantity, false)
	if e != nil {
		return Transaction{}, e
	}
	v.Input, _, e = field(m, "input", parseData, false)
	if e != nil {
		return Transaction{}, e
	}
	v.BlockHash, v.HasBlockHash, e = field(m, "blockHash", parseHash, true)
	if e != nil {
		return Transaction{}, e
	}
	v.BlockNumber, v.HasBlockNumber, e = field(m, "blockNumber", parseUint, true)
	if e != nil {
		return Transaction{}, e
	}
	v.TransactionIndex, v.HasTransactionIndex, e = field(m, "transactionIndex", parseUint, true)
	if e != nil {
		return Transaction{}, e
	}
	v.GasPrice, v.HasGasPrice, e = field(m, "gasPrice", parseQuantity, false)
	if e != nil {
		return Transaction{}, e
	}
	v.MaxFeePerGas, v.HasMaxFeePerGas, e = field(m, "maxFeePerGas", parseQuantity, false)
	if e != nil {
		return Transaction{}, e
	}
	v.MaxPriorityFeePerGas, v.HasMaxPriorityFeePerGas, e = field(m, "maxPriorityFeePerGas", parseQuantity, false)
	if e != nil {
		return Transaction{}, e
	}
	v.ChainID, v.HasChainID, e = field(m, "chainId", parseQuantity, false)
	if e != nil {
		return Transaction{}, e
	}
	v.Type, v.HasType, e = field(m, "type", parseUint, false)
	if e != nil {
		return Transaction{}, e
	}
	if v.HasBlockHash != v.HasBlockNumber || v.HasBlockHash != v.HasTransactionIndex {
		return Transaction{}, protocol()
	}
	return v, nil
}
func parseLog(raw json.RawMessage) (Log, error) {
	m, e := object(raw, "address topics data blockHash transactionHash blockNumber transactionIndex logIndex")
	if e != nil {
		return Log{}, e
	}
	var v Log
	v.Address, _, e = field(m, "address", parseAddress, false)
	if e != nil {
		return Log{}, e
	}
	v.Topics, _, e = field(m, "topics", parseHashes, false)
	if e != nil {
		return Log{}, e
	}
	v.Data, _, e = field(m, "data", parseData, false)
	if e != nil {
		return Log{}, e
	}
	v.Removed, _, e = field(m, "removed", parseBool, false)
	if e != nil {
		return Log{}, e
	}
	v.BlockHash, v.HasBlockHash, e = field(m, "blockHash", parseHash, true)
	if e != nil {
		return Log{}, e
	}
	v.TransactionHash, v.HasTransactionHash, e = field(m, "transactionHash", parseHash, true)
	if e != nil {
		return Log{}, e
	}
	v.BlockNumber, v.HasBlockNumber, e = field(m, "blockNumber", parseUint, true)
	if e != nil {
		return Log{}, e
	}
	v.TransactionIndex, v.HasTransactionIndex, e = field(m, "transactionIndex", parseUint, true)
	if e != nil {
		return Log{}, e
	}
	v.LogIndex, v.HasLogIndex, e = field(m, "logIndex", parseUint, true)
	if e != nil {
		return Log{}, e
	}
	if len(v.Topics) > 4 {
		return Log{}, protocol()
	}
	return v, nil
}
func parseReceipt(raw json.RawMessage) (Receipt, error) {
	m, e := object(raw, "transactionHash blockHash transactionIndex blockNumber from to contractAddress cumulativeGasUsed gasUsed logs logsBloom")
	if e != nil {
		return Receipt{}, e
	}
	var v Receipt
	v.TransactionHash, _, e = field(m, "transactionHash", parseHash, false)
	if e != nil {
		return Receipt{}, e
	}
	v.BlockHash, _, e = field(m, "blockHash", parseHash, false)
	if e != nil {
		return Receipt{}, e
	}
	v.TransactionIndex, _, e = field(m, "transactionIndex", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	v.BlockNumber, _, e = field(m, "blockNumber", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	v.From, _, e = field(m, "from", parseAddress, false)
	if e != nil {
		return Receipt{}, e
	}
	v.To, v.HasTo, e = field(m, "to", parseAddress, true)
	if e != nil {
		return Receipt{}, e
	}
	v.ContractAddress, v.HasContractAddress, e = field(m, "contractAddress", parseAddress, true)
	if e != nil {
		return Receipt{}, e
	}
	v.CumulativeGasUsed, _, e = field(m, "cumulativeGasUsed", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	v.GasUsed, _, e = field(m, "gasUsed", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	v.Logs, _, e = field(m, "logs", parseLogs, false)
	if e != nil {
		return Receipt{}, e
	}
	v.LogsBloom, _, e = field(m, "logsBloom", parseBloom, false)
	if e != nil {
		return Receipt{}, e
	}
	v.Status, v.HasStatus, e = field(m, "status", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	v.Root, v.HasRoot, e = field(m, "root", parseHash, false)
	if e != nil {
		return Receipt{}, e
	}
	v.EffectiveGasPrice, v.HasEffectiveGasPrice, e = field(m, "effectiveGasPrice", parseQuantity, false)
	if e != nil {
		return Receipt{}, e
	}
	v.Type, v.HasType, e = field(m, "type", parseUint, false)
	if e != nil {
		return Receipt{}, e
	}
	if v.HasStatus == v.HasRoot || (v.HasStatus && v.Status > 1) {
		return Receipt{}, protocol()
	}
	for _, l := range v.Logs {
		if !l.HasBlockHash || !l.HasTransactionHash || !l.HasBlockNumber || !l.HasTransactionIndex || !l.HasLogIndex {
			return Receipt{}, protocol()
		}
	}
	return v, nil
}

func requestValue[T any](c *Client, ctx context.Context, method string, params []any, parse func(json.RawMessage) (T, error)) (T, error) {
	var zero T
	raw, e := c.invoke(ctx, method, params)
	if e != nil {
		return zero, e
	}
	return parse(raw)
}
func (c *Client) ChainID(ctx context.Context) (big.Int, error) {
	return requestValue(c, ctx, "eth_chainId", []any{}, parseQuantity)
}
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return requestValue(c, ctx, "eth_blockNumber", []any{}, parseUint)
}
func (c *Client) GetBalance(ctx context.Context, address Address, at BlockRef) (big.Int, error) {
	if at.wire == "" {
		return big.Int{}, invalid()
	}
	return requestValue(c, ctx, "eth_getBalance", []any{address, at}, parseQuantity)
}
func (c *Client) GetTransactionCount(ctx context.Context, address Address, at BlockRef) (uint64, error) {
	if at.wire == "" {
		return 0, invalid()
	}
	return requestValue(c, ctx, "eth_getTransactionCount", []any{address, at}, parseUint)
}
func (c *Client) GetCode(ctx context.Context, address Address, at BlockRef) (Data, error) {
	if at.wire == "" {
		return nil, invalid()
	}
	return requestValue(c, ctx, "eth_getCode", []any{address, at}, parseData)
}
func (c *Client) GetBlock(ctx context.Context, at BlockRef) (Block, bool, error) {
	if at.wire == "" {
		return Block{}, false, invalid()
	}
	raw, e := c.invoke(ctx, "eth_getBlockByNumber", []any{at, false})
	if e != nil {
		return Block{}, false, e
	}
	if isNull(raw) {
		return Block{}, false, nil
	}
	v, e := parseBlock(raw)
	if e != nil {
		return Block{}, false, e
	}
	if !v.HasNumber && at.wire != "pending" {
		return Block{}, false, protocol()
	}
	return v, true, nil
}
func (c *Client) GetTransaction(ctx context.Context, hash Hash) (Transaction, bool, error) {
	raw, e := c.invoke(ctx, "eth_getTransactionByHash", []any{hash})
	if e != nil {
		return Transaction{}, false, e
	}
	if isNull(raw) {
		return Transaction{}, false, nil
	}
	v, e := parseTransaction(raw)
	if e != nil {
		return Transaction{}, false, e
	}
	return v, true, nil
}
func (c *Client) GetTransactionReceipt(ctx context.Context, hash Hash) (Receipt, bool, error) {
	raw, e := c.invoke(ctx, "eth_getTransactionReceipt", []any{hash})
	if e != nil {
		return Receipt{}, false, e
	}
	if isNull(raw) {
		return Receipt{}, false, nil
	}
	v, e := parseReceipt(raw)
	if e != nil {
		return Receipt{}, false, e
	}
	return v, true, nil
}
func (c *Client) GetLogs(ctx context.Context, filter LogFilter) ([]Log, error) {
	if filter.FromBlock.wire == "" || filter.ToBlock.wire == "" || len(filter.Topics) > 4 {
		return nil, invalid()
	}
	m := map[string]any{"fromBlock": filter.FromBlock, "toBlock": filter.ToBlock}
	if len(filter.Addresses) > 0 {
		m["address"] = append([]Address{}, filter.Addresses...)
	}
	topics := make([]any, len(filter.Topics))
	for i, t := range filter.Topics {
		if len(t) > 0 {
			topics[i] = append([]Hash{}, t...)
		}
	}
	m["topics"] = topics
	return requestValue(c, ctx, "eth_getLogs", []any{m}, parseLogs)
}
