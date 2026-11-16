package evm

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
)

// SendRawTransaction sends the supplied serialized bytes once. A returned hash
// indicates node acceptance only; execution and confirmation require queries.
func (c *Client) SendRawTransaction(ctx context.Context, payload Data) (Hash, error) {
	if ctx == nil || len(payload) == 0 {
		return Hash{}, invalid()
	}
	raw, e := c.invoke(ctx, "eth_sendRawTransaction", []any{"0x" + hex.EncodeToString(payload)})
	var hash Hash
	if e == nil {
		hash, e = parseHash(raw)
	}
	if e != nil {
		var a apierror.Error
		if errors.As(e, &a) {
			a.Method = "eth_sendRawTransaction"
			switch a.Kind {
			case apierror.Transport, apierror.HTTP, apierror.Protocol:
				a.SubmissionUnknown = true
			}
			return Hash{}, a
		}
		return Hash{}, apierror.Error{Kind: apierror.Protocol, Method: "eth_sendRawTransaction", SubmissionUnknown: true}
	}
	return hash, nil
}
