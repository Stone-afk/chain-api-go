package tron

import (
	"context"
	"encoding/json"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func (c *Client) compute(ctx context.Context, method string, req ContractCallRequest) (json.RawMessage, error) {
	if !validAddress(req.OwnerAddress, req.Visible) || !validAddress(req.ContractAddress, req.Visible) || req.FunctionSelector == "" || len(req.Parameter)%2 != 0 || !validHex(req.Parameter) || req.HasCallValue && req.CallValue < 0 {
		return nil, invalid(method)
	}
	body := map[string]any{"owner_address": req.OwnerAddress, "contract_address": req.ContractAddress, "function_selector": req.FunctionSelector, "parameter": req.Parameter, "visible": req.Visible}
	if req.HasCallValue {
		body["call_value"] = req.CallValue
	}
	raw, fields, err := c.post(ctx, method, body, false)
	if err != nil {
		return nil, err
	}
	inner, err := object(fields["result"])
	if err != nil {
		return nil, protocol(method, err, false)
	}
	accepted, valid := decision(inner["result"])
	if !valid {
		return nil, protocol(method, nil, false)
	}
	var result NodeResult
	if err := json.Unmarshal(fields["result"], &result); err != nil {
		return nil, protocol(method, err, false)
	}
	if !accepted {
		return nil, apierror.Error{Kind: apierror.RPC, Method: method, NodeCode: result.Code, Message: result.Message, Data: raw}
	}
	return raw, nil
}

func (c *Client) TriggerConstantContract(ctx context.Context, req ContractCallRequest) (ConstantResult, error) {
	const method = "/wallet/triggerconstantcontract"
	raw, err := c.compute(ctx, method, req)
	if err != nil {
		return ConstantResult{}, err
	}
	var result ConstantResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return ConstantResult{}, protocol(method, err, false)
	}
	return result, nil
}
func (c *Client) EstimateEnergy(ctx context.Context, req ContractCallRequest) (EnergyEstimate, error) {
	const method = "/wallet/estimateenergy"
	raw, err := c.compute(ctx, method, req)
	if err != nil {
		return EnergyEstimate{}, err
	}
	var result EnergyEstimate
	if err := json.Unmarshal(raw, &result); err != nil {
		return EnergyEstimate{}, protocol(method, err, false)
	}
	return result, nil
}
