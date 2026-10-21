package couchcore

import (
	"bytes"
	"encoding/json"
)

// AttentionByteLimit retains the event payload bound without persistence.
const AttentionByteLimit = 256 << 10

// encodeAttention is the one encoding of the in-memory payload for the reducer's byte bound. HTML escaping is off: it would grow every < > &
// sixfold for nothing.
func encodeAttention(items []AttentionItem) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(items); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
