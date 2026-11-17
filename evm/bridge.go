package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"io"
	"net/http"
)

// callerError distinguishes dependency failures from native response decoding.
type callerError struct{ cause error }

func (e callerError) Error() string { return e.cause.Error() }
func (e callerError) Unwrap() error { return e.cause }

type bridge struct{ caller jsonrpc.Caller }
type bridgeRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}
type bridgeResponse struct {
	Version string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
}

// RoundTrip is an in-memory protocol bridge; it never uses a network transport.
func (b bridge) RoundTrip(req *http.Request) (*http.Response, error) {
	defer req.Body.Close()
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var requests []bridgeRequest
	batch := bytes.HasPrefix(bytes.TrimSpace(raw), []byte("["))
	if batch {
		err = json.Unmarshal(raw, &requests)
	} else {
		var request bridgeRequest
		err = json.Unmarshal(raw, &request)
		requests = []bridgeRequest{request}
	}
	if err != nil {
		return nil, err
	}
	responses := make([]bridgeResponse, 0, len(requests))
	for _, request := range requests {
		var result json.RawMessage
		call := func(ctx context.Context) error { return b.caller.Call(ctx, request.Method, request.Params, &result) }
		var err error
		if gate, ok := req.Context().(interface {
			CallRPC(func(context.Context) error) error
		}); ok {
			err = gate.CallRPC(call)
		} else {
			err = call(req.Context())
		}
		if err != nil {
			return nil, callerError{err}
		}
		responses = append(responses, bridgeResponse{"2.0", request.ID, result})
	}
	var body []byte
	if batch {
		body, err = json.Marshal(responses)
	} else {
		body, err = json.Marshal(responses[0])
	}
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: req}, nil
}
