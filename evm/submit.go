package evm

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

// SendRawTransaction submits the caller's encoded payload unchanged, once.
// A returned hash indicates acceptance, not execution or confirmation.
func (c *Client) SendRawTransaction(ctx context.Context, payload string) (common.Hash, error) {
	const method = "eth_sendRawTransaction"
	if len(payload) <= 2 || !strings.HasPrefix(payload, "0x") {
		return common.Hash{}, apierror.Error{Kind: apierror.InvalidArgument}
	}
	if _, err := hex.DecodeString(payload[2:]); err != nil {
		return common.Hash{}, apierror.Error{Kind: apierror.InvalidArgument}
	}
	var hash common.Hash
	err := c.rpc.CallContext(ctx, &hash, method, payload)
	if err == nil && hash == (common.Hash{}) {
		err = errors.New("zero transaction hash result")
	}
	if err != nil {
		out := apierror.Error{Kind: apierror.Protocol, Method: method, SubmissionUnknown: true, Cause: err}
		var origin callerError
		if errors.As(err, &origin) {
			out.Kind = apierror.Transport
			var source apierror.Error
			if errors.As(origin, &source) {
				out = source
			}
			out.Method = method
			out.SubmissionUnknown = out.Kind != apierror.RPC
			out.Cause = err
		}
		return common.Hash{}, out
	}
	return hash, nil
}
