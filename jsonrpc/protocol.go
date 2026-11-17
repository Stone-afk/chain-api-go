package jsonrpc

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/Stone-afk/chain-api-go/apierror"
)

type request struct {
	Version string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// parseResponse retains the result or validated raw node error fields.
// Field presence is distinct from JSON null in both supported dialects.
func parseResponse(body []byte, version Version, id uint64) (json.RawMessage, apierror.Error, bool) {
	fields, valid, err := parseObject(body)
	if !valid {
		return nil, apierror.Error{Cause: err}, false
	}
	var responseID uint64
	raw := fields["id"]
	if len(raw) == 0 || isNull(raw) {
		return nil, apierror.Error{}, false
	}
	if err := json.Unmarshal(raw, &responseID); err != nil {
		return nil, apierror.Error{Cause: err}, false
	}
	if responseID != id {
		return nil, apierror.Error{}, false
	}
	result, hasResult := fields["result"]
	rpcError, hasError := fields["error"]
	if version == V2 {
		var wireVersion string
		if err := json.Unmarshal(fields["jsonrpc"], &wireVersion); err != nil {
			return nil, apierror.Error{Cause: err}, false
		}
		if wireVersion != V2.ToString() || hasResult == hasError {
			return nil, apierror.Error{}, false
		}
	} else {
		if !hasResult || !hasError {
			return nil, apierror.Error{}, false
		}
		if isNull(rpcError) {
			return result, apierror.Error{}, true
		}
		if !isNull(result) {
			return nil, apierror.Error{}, false
		}
	}
	if !hasError {
		return result, apierror.Error{}, true
	}
	details, valid, err := parseObject(rpcError)
	if !valid {
		return nil, apierror.Error{Cause: err}, false
	}
	var code int
	var message string
	if len(details["code"]) == 0 || isNull(details["code"]) || len(details["message"]) == 0 || isNull(details["message"]) {
		return nil, apierror.Error{}, false
	}
	if err := json.Unmarshal(details["code"], &code); err != nil {
		return nil, apierror.Error{Cause: err}, false
	}
	if err := json.Unmarshal(details["message"], &message); err != nil {
		return nil, apierror.Error{Cause: err}, false
	}
	return nil, apierror.Error{RPCCode: code, HasRPCCode: true, Message: message, Data: details["data"]}, true
}

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

// parseObject rejects ambiguous members instead of silently keeping the last.
func parseObject(raw []byte) (map[string]json.RawMessage, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, false, err
	}
	if token != json.Delim('{') {
		return nil, false, nil
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, false, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, false, nil
		}
		if _, exists := fields[name]; exists {
			return nil, false, nil
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false, err
		}
		fields[name] = value
	}
	token, err = decoder.Token()
	if err != nil {
		return nil, false, err
	}
	if token != json.Delim('}') {
		return nil, false, nil
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false, err
	}
	return fields, true, nil
}
