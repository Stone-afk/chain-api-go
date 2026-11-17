package solana

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/Stone-afk/chain-api-go/apierror"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"strings"
)

// SendEncodedTransactionWithOpts submits an already encoded payload once.
// Acceptance does not imply execution or confirmation.
func (c *Client) SendEncodedTransactionWithOpts(ctx context.Context, payload string, opts srpc.TransactionOpts) (ns.Signature, error) {
	const method = "sendTransaction"
	valid := payload != ""
	switch opts.Encoding {
	case "", ns.EncodingBase64:
		_, err := base64.StdEncoding.DecodeString(payload)
		valid = valid && err == nil
	case ns.EncodingBase58:
		for _, r := range payload {
			if !strings.ContainsRune("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", r) {
				valid = false
				break
			}
		}
	default:
		valid = false
	}
	if !valid {
		return ns.Signature{}, apierror.Error{Kind: apierror.InvalidArgument, Method: method}
	}
	signature, err := c.rpc.SendEncodedTransactionWithOpts(ctx, payload, opts)
	if err == nil && signature == (ns.Signature{}) {
		err = errors.New("zero transaction signature result")
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
		return ns.Signature{}, out
	}
	return signature, nil
}
