package tron

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/httpjson"
)

type Client struct{ poster httpjson.Poster }

func New(poster httpjson.Poster) (*Client, error) {
	return &Client{poster: poster}, nil
}

func protocol(method string, cause error, submission bool) error {
	return apierror.Error{Kind: apierror.Protocol, Method: method, Cause: cause, SubmissionUnknown: submission}
}
func invalid(method string) error {
	return apierror.Error{Kind: apierror.InvalidArgument, Method: method}
}

// post retains the response bytes for node errors and passes context unchanged.
func (c *Client) post(ctx context.Context, method string, request any, submission bool) (json.RawMessage, map[string]json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.poster.Post(ctx, method, request, &raw); err != nil {
		var e apierror.Error
		if !errors.As(err, &e) {
			e.Kind = apierror.Transport
		}
		e.Method, e.Cause = method, err
		e.SubmissionUnknown = submission && e.Kind != apierror.RPC
		return nil, nil, e
	}
	fields, err := object(raw)
	if err != nil {
		return nil, nil, protocol(method, err, submission)
	}
	if fields == nil {
		return nil, nil, protocol(method, nil, submission)
	}
	if value, ok := fields["Error"]; ok {
		var message string
		if len(value) == 0 || value[0] != '"' {
			return nil, nil, protocol(method, nil, submission)
		}
		if err := json.Unmarshal(value, &message); err != nil {
			return nil, nil, protocol(method, err, submission)
		}
		return nil, nil, apierror.Error{Kind: apierror.RPC, Method: method, Message: message, Data: raw}
	}
	return raw, fields, nil
}

func object(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	err := json.Unmarshal(raw, &fields)
	return fields, err
}

func decision(raw json.RawMessage) (bool, bool) {
	switch string(bytes.TrimSpace(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}
