package bitcoin

import (
	"context"
	"encoding/json"
)

// EstimateSmartFee returns hasEstimate=false when feerate is absent, preserving blocks and errors.
func (c *Client) EstimateSmartFee(ctx context.Context, req EstimateSmartFeeRequest) (FeeEstimate, bool, error) {
	const method = "estimatesmartfee"
	if req.ConfTarget < 1 || req.ConfTarget > 1008 || req.HasMode && req.Mode != EstimateUnset && req.Mode != EstimateEconomical && req.Mode != EstimateConservative {
		return FeeEstimate{}, false, invalid(method)
	}
	params := []any{req.ConfTarget}
	if req.HasMode {
		params = append(params, req.Mode)
	}
	raw, err := c.call(ctx, method, params)
	if err != nil {
		return FeeEstimate{}, false, err
	}
	var w struct {
		FeeEstimate
		FeeRate json.RawMessage `json:"feerate"`
	}
	if err = decode(method, raw, &w, '{'); err != nil {
		return FeeEstimate{}, false, err
	}
	if w.FeeRate == nil {
		return w.FeeEstimate, false, nil
	}
	amount, ok := btcAmount(w.FeeRate)
	if !ok {
		return FeeEstimate{}, false, protocol(method, nil)
	}
	w.FeeEstimate.FeeRate = SatoshiPerKVByte(amount)
	return w.FeeEstimate, true, nil
}

func (c *Client) TestMempoolAccept(ctx context.Context, payload RawTransaction) (MempoolAcceptResult, error) {
	const method = "testmempoolaccept"
	if !validHex(string(payload)) {
		return MempoolAcceptResult{}, invalid(method)
	}
	raw, err := c.call(ctx, method, []any{[]RawTransaction{payload}})
	if err != nil {
		return MempoolAcceptResult{}, err
	}
	var items []json.RawMessage
	if err = decode(method, raw, &items, '['); err != nil {
		return MempoolAcceptResult{}, err
	}
	if len(items) != 1 {
		return MempoolAcceptResult{}, protocol(method, nil)
	}
	var w struct {
		MempoolAcceptResult
		Allowed json.RawMessage `json:"allowed"`
	}
	if err = decode(method, items[0], &w, '{'); err != nil {
		return MempoolAcceptResult{}, err
	}
	if string(w.Allowed) != "true" && string(w.Allowed) != "false" {
		return MempoolAcceptResult{}, protocol(method, nil)
	}
	w.MempoolAcceptResult.Allowed = string(w.Allowed) == "true"
	return w.MempoolAcceptResult, nil
}
