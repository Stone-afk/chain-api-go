package live

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

func verifyManifest(raw []byte, expectedDigest string) (string, error) {
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	if digest != expectedDigest {
		return "", errors.New("manifest")
	}
	if _, err := strictObject(raw); err != nil {
		return "", errors.New("manifest")
	}
	return digest, nil
}

// Token decoding detects duplicate decoded keys at every nesting level.
func strictObject(raw []byte) (map[string]any, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("json")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value func() (any, error)
	value = func() (any, error) {
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t {
		case json.Delim('{'):
			obj := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				k, ok := key.(string)
				if !ok {
					return nil, errors.New("json")
				}
				if _, exists := obj[k]; exists {
					return nil, errors.New("json")
				}
				v, err := value()
				if err != nil {
					return nil, err
				}
				obj[k] = v
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return nil, errors.New("json")
			}
			return obj, nil
		case json.Delim('['):
			arr := []any{}
			for d.More() {
				v, err := value()
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return nil, errors.New("json")
			}
			return arr, nil
		default:
			if _, delimiter := t.(json.Delim); delimiter {
				return nil, errors.New("json")
			}
			return t, nil
		}
	}
	v, err := value()
	if err != nil {
		return nil, errors.New("json")
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("json")
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errors.New("json")
	}
	return obj, nil
}
