package evm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
)

func callParams(req CallRequest) (map[string]any, error) {
	if req.HasGasPrice && (req.HasMaxFeePerGas || req.HasMaxPriorityFeePerGas) {
		return nil, invalid()
	}
	m := map[string]any{"data": "0x" + hex.EncodeToString(req.Data)}
	if req.HasFrom {
		m["from"] = req.From
	}
	if req.HasTo {
		m["to"] = req.To
	}
	if req.HasGas {
		m["gas"] = quantity(req.Gas)
	}
	for _, f := range []struct {
		name  string
		value big.Int
		has   bool
	}{
		{"value", req.Value, req.HasValue}, {"gasPrice", req.GasPrice, req.HasGasPrice},
		{"maxFeePerGas", req.MaxFeePerGas, req.HasMaxFeePerGas}, {"maxPriorityFeePerGas", req.MaxPriorityFeePerGas, req.HasMaxPriorityFeePerGas},
	} {
		if f.has {
			if f.value.Sign() < 0 {
				return nil, invalid()
			}
			m[f.name] = "0x" + f.value.Text(16)
		}
	}
	return m, nil
}
func (c *Client) Call(ctx context.Context, req CallRequest, at BlockRef) (Data, error) {
	if at.wire == "" {
		return nil, invalid()
	}
	m, e := callParams(req)
	if e != nil {
		return nil, e
	}
	return requestValue(c, ctx, "eth_call", []any{m, at}, parseData)
}
func (c *Client) EstimateGas(ctx context.Context, req CallRequest) (uint64, error) {
	m, e := callParams(req)
	if e != nil {
		return 0, e
	}
	return requestValue(c, ctx, "eth_estimateGas", []any{m}, parseUint)
}
func (c *Client) GasPrice(ctx context.Context) (big.Int, error) {
	return requestValue(c, ctx, "eth_gasPrice", []any{}, parseQuantity)
}
func parseRatio(raw json.RawMessage) (float64, error) {
	var v float64
	if isNull(raw) || json.Unmarshal(raw, &v) != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
		return 0, protocol()
	}
	return v, nil
}
func parseAmounts(raw json.RawMessage) ([]big.Int, error)   { return parseArray(raw, parseQuantity) }
func parseRewards(raw json.RawMessage) ([][]big.Int, error) { return parseArray(raw, parseAmounts) }
func parseRatios(raw json.RawMessage) ([]float64, error)    { return parseArray(raw, parseRatio) }
func (c *Client) FeeHistory(ctx context.Context, count uint64, newest BlockRef, percentiles []float64) (FeeHistoryResult, error) {
	if count == 0 || newest.wire == "" {
		return FeeHistoryResult{}, invalid()
	}
	for i, p := range percentiles {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 100 || (i > 0 && p < percentiles[i-1]) {
			return FeeHistoryResult{}, invalid()
		}
	}
	raw, e := c.invoke(ctx, "eth_feeHistory", []any{quantity(count), newest, append([]float64{}, percentiles...)})
	if e != nil {
		return FeeHistoryResult{}, e
	}
	m, e := object(raw, "oldestBlock baseFeePerGas gasUsedRatio")
	if e != nil {
		return FeeHistoryResult{}, e
	}
	var v FeeHistoryResult
	v.OldestBlock, _, e = field(m, "oldestBlock", parseUint, false)
	if e != nil {
		return FeeHistoryResult{}, e
	}
	v.BaseFeePerGas, _, e = field(m, "baseFeePerGas", parseAmounts, false)
	if e != nil {
		return FeeHistoryResult{}, e
	}
	v.GasUsedRatio, _, e = field(m, "gasUsedRatio", parseRatios, false)
	if e != nil {
		return FeeHistoryResult{}, e
	}
	v.Reward, v.HasReward, e = field(m, "reward", parseRewards, false)
	if e != nil {
		return FeeHistoryResult{}, e
	}
	n := len(v.GasUsedRatio)
	if n < 1 || uint64(n) > count || len(v.BaseFeePerGas) != n+1 {
		return FeeHistoryResult{}, protocol()
	}
	if len(percentiles) > 0 {
		if !v.HasReward || len(v.Reward) != n {
			return FeeHistoryResult{}, protocol()
		}
	} else if len(v.Reward) != 0 && len(v.Reward) != n {
		return FeeHistoryResult{}, protocol()
	}
	for _, row := range v.Reward {
		if len(row) != len(percentiles) {
			return FeeHistoryResult{}, protocol()
		}
	}
	return v, nil
}
