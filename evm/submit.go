package evm

import (
	"context"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"strings"
)

// SendRawTransaction sends the unchanged payload once. A hash means node
// acceptance only, not execution or confirmation.
func (c *Client) SendRawTransaction(ctx context.Context, payload Data) (Hash, error) {
	if len(payload) <= 2 || !validHex(string(payload), -1) {
		return "", invalid()
	}
	const method = "eth_sendRawTransaction"
	raw, err := c.invoke(ctx, method, []any{payload}, true)
	if err != nil {
		return "", err
	}
	hash, err := decode[Hash](raw)
	if err == nil && (!validHex(string(hash), 32) || strings.TrimLeft(string(hash)[2:], "0") == "") {
		err = errors.New("invalid transaction hash result")
	}
	if err != nil {
		return "", apierror.Error{Kind: apierror.Protocol, Method: method, Cause: err, SubmissionUnknown: true}
	}
	return hash, nil
}
