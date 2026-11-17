package bitcoin

import (
	"context"
	"encoding/json"
)

// SendRawTransaction submits the unchanged payload once; acceptance is not confirmation.
func (c *Client) SendRawTransaction(ctx context.Context, payload RawTransaction) (TxID, error) {
	const method = "sendrawtransaction"
	if !validHex(string(payload)) {
		return "", invalid(method)
	}
	raw, err := c.call(ctx, method, []any{payload})
	if err != nil {
		return "", err
	}
	var v TxID
	if err = json.Unmarshal(raw, &v); err != nil {
		return "", protocol(method, err)
	}
	if !validHash(string(v)) {
		return "", protocol(method, nil)
	}
	return v, nil
}
