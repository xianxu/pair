package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// DecodeArgumentArray preserves explicit empty argv and rejects null arrays or
// elements at both persisted-preference and operation/protocol boundaries.
func DecodeArgumentArray(raw []byte) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, errors.New("startup parameters must be a JSON string array")
	}
	var elements []json.RawMessage
	if err := strictjson.Decode(raw, &elements); err != nil {
		return nil, err
	}
	argv := make([]string, len(elements))
	for i, element := range elements {
		value := bytes.TrimSpace(element)
		if len(value) == 0 || value[0] != '"' {
			return nil, errors.New("startup parameters must contain only strings")
		}
		if err := strictjson.Decode(element, &argv[i]); err != nil {
			return nil, err
		}
	}
	return argv, nil
}
