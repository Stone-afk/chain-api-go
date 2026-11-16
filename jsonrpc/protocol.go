package jsonrpc

import (
	"bytes"
	"encoding/json"
	"io"
)

type request struct {
	Version string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// parseResponse retains only the result or a validated numeric node code.
// Field presence is distinct from JSON null in both supported dialects.
func parseResponse(body []byte, version Version, id uint64) (json.RawMessage, int, bool, bool) {
	fields, valid := parseObject(body)
	if !valid {
		return nil, 0, false, false
	}
	var responseID uint64
	if raw := fields["id"]; len(raw) == 0 || isNull(raw) || json.Unmarshal(raw, &responseID) != nil || responseID != id {
		return nil, 0, false, false
	}
	result, hasResult := fields["result"]
	rpcError, hasError := fields["error"]
	if version == V2 {
		var wireVersion string
		if json.Unmarshal(fields["jsonrpc"], &wireVersion) != nil || wireVersion != V2.ToString() || hasResult == hasError {
			return nil, 0, false, false
		}
	} else {
		if !hasResult || !hasError {
			return nil, 0, false, false
		}
		if isNull(rpcError) {
			return result, 0, false, true
		}
		if !isNull(result) {
			return nil, 0, false, false
		}
	}
	if !hasError {
		return result, 0, false, true
	}
	details, valid := parseObject(rpcError)
	if !valid {
		return nil, 0, false, false
	}
	var code int
	var message string
	if len(details["code"]) == 0 || isNull(details["code"]) || json.Unmarshal(details["code"], &code) != nil || len(details["message"]) == 0 || isNull(details["message"]) || json.Unmarshal(details["message"], &message) != nil {
		return nil, 0, false, false
	}
	return nil, code, true, true
}

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }

// parseObject rejects ambiguous members instead of silently keeping the last.
func parseObject(raw []byte) (map[string]json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, false
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, false
		}
		name, ok := token.(string)
		if !ok {
			return nil, false
		}
		if _, exists := fields[name]; exists {
			return nil, false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, false
		}
		fields[name] = value
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false
	}
	return fields, true
}
