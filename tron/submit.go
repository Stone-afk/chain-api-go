package tron

import (
	"context"
	"encoding/json"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func (c *Client) BroadcastTransaction(ctx context.Context, payload TransactionPayload) (BroadcastResult, bool, error) {
	const method = "/wallet/broadcasttransaction"
	fields, err := object(payload.JSON)
	if err != nil || fields == nil {
		return BroadcastResult{}, false, apierror.Error{Kind: apierror.InvalidArgument, Method: method, Cause: err}
	}
	raw, fields, err := c.post(ctx, method, payload.JSON, true)
	if err != nil {
		return BroadcastResult{}, false, err
	}
	accepted, valid := decision(fields["result"])
	if !valid {
		return BroadcastResult{}, false, protocol(method, nil, true)
	}
	var result BroadcastResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return BroadcastResult{}, false, protocol(method, err, true)
	}
	if !accepted {
		return BroadcastResult{}, false, apierror.Error{Kind: apierror.RPC, Method: method, NodeCode: result.Code, Message: result.Message, Data: raw}
	}
	if _, exists := fields["txid"]; exists && !validTxID(result.TxID) {
		return BroadcastResult{}, false, protocol(method, nil, true)
	}
	return result, true, nil
}
