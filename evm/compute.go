package evm

import (
	"context"
	"math"
	"math/big"
)

func validateCall(req CallRequest) error {
	if req.GasPrice != "" && (req.MaxFeePerGas != "" || req.MaxPriorityFeePerGas != "") {
		return invalid()
	}
	for _, s := range []string{req.From, req.To} {
		if s != "" && !validHex(s, 20) {
			return invalid()
		}
	}
	for _, s := range []string{req.Gas, req.Value, req.GasPrice, req.MaxFeePerGas, req.MaxPriorityFeePerGas} {
		if s != "" && !validQuantity(s) {
			return invalid()
		}
	}
	if req.Data != "" && !validHex(req.Data, -1) {
		return invalid()
	}
	return nil
}
func (c *Client) Call(ctx context.Context, req CallRequest, at BlockRef) (Data, error) {
	if at.wire == "" {
		return "", invalid()
	}
	if err := validateCall(req); err != nil {
		return "", err
	}
	return requestValue(c, ctx, "eth_call", []any{req, at}, decode[Data])
}
func (c *Client) EstimateGas(ctx context.Context, req CallRequest) (uint64, error) {
	if err := validateCall(req); err != nil {
		return 0, err
	}
	return requestValue(c, ctx, "eth_estimateGas", []any{req}, parseUint)
}
func (c *Client) GasPrice(ctx context.Context) (big.Int, error) {
	return requestValue(c, ctx, "eth_gasPrice", []any{}, parseQuantity)
}
func (c *Client) FeeHistory(ctx context.Context, count uint64, newest BlockRef, percentiles []float64) (FeeHistoryResult, error) {
	if count == 0 || newest.wire == "" {
		return FeeHistoryResult{}, invalid()
	}
	for i, p := range percentiles {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 100 || (i > 0 && p < percentiles[i-1]) {
			return FeeHistoryResult{}, invalid()
		}
	}
	return requestValue(c, ctx, "eth_feeHistory", []any{quantity(count), newest, append([]float64{}, percentiles...)}, decode[FeeHistoryResult])
}
