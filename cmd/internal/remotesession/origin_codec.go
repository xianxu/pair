package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"strings"
)

const maxOriginsBytes = 1 << 20
const maxOriginMarkers = 4096

type originRecord struct {
	root      map[string]json.RawMessage
	documents map[string]map[string]json.RawMessage
}

func remoteObject(raw []byte, known ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, errors.New("remote record requires object")
	}
	for key := range fields {
		for _, name := range known {
			if key != name && strings.EqualFold(key, name) {
				return nil, errors.New("remote record case alias")
			}
		}
	}
	return fields, nil
}
func requiredRemoteField(fields map[string]json.RawMessage, key string, target any) error {
	raw, present := fields[key]
	if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("remote required field missing/null")
	}
	return json.Unmarshal(raw, target)
}
func validDocumentID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func decodeOrigins(raw []byte) (originRecord, error) {
	var record originRecord
	if err := validateRemoteJSON(raw, maxOriginsBytes); err != nil {
		return record, err
	}
	root, err := remoteObject(raw, "version", "documents")
	if err != nil {
		return record, err
	}
	var version int
	if err := requiredRemoteField(root, "version", &version); err != nil {
		return record, err
	}
	if version != 1 {
		return record, errors.New("unsupported remote origin version")
	}
	var documents map[string]json.RawMessage
	if err := requiredRemoteField(root, "documents", &documents); err != nil {
		return record, err
	}
	if len(documents) > maxOriginMarkers {
		return record, errors.New("remote origin capacity reached")
	}
	record = originRecord{root: root, documents: make(map[string]map[string]json.RawMessage, len(documents))}
	for id, rawMarker := range documents {
		if !validDocumentID(id) {
			return originRecord{}, errors.New("invalid remote document ID")
		}
		fields, err := remoteObject(rawMarker, "remote_edited", "forget_pending", "submit_seen")
		if err != nil {
			return originRecord{}, err
		}
		if len(fields["remote_edited"]) == 0 && len(fields["forget_pending"]) == 0 && len(fields["submit_seen"]) == 0 {
			return originRecord{}, errors.New("remote origin record requires marker or forget intent")
		}
		for _, key := range []string{"remote_edited", "forget_pending", "submit_seen"} {
			if _, present := fields[key]; !present {
				continue
			}
			var flag bool
			if err := requiredRemoteField(fields, key, &flag); err != nil {
				return originRecord{}, err
			}
			if !flag {
				return originRecord{}, errors.New("remote origin flags cannot be false")
			}
		}
		record.documents[id] = fields
	}
	return record, nil
}

// Retained raw members are emitted verbatim; no float conversion or opaque
// nested-object rewrite occurs. Only this owner's fields are replaced.
func marshalRemoteFields(fields map[string]json.RawMessage) []byte {
	// All members passed here were admitted by the bounded preserving decoder or
	// encoded from typed owned values. Reuse the shared verbatim object writer.
	raw, _ := strictjson.EncodeObject(fields, nil, nil)
	return raw
}
func encodeOrigins(record originRecord) ([]byte, error) {
	documents := make(map[string]json.RawMessage, len(record.documents))
	for id, fields := range record.documents {
		documents[id] = marshalRemoteFields(fields)
	}
	root := make(map[string]json.RawMessage, len(record.root)+2)
	for key, value := range record.root {
		root[key] = value
	}
	root["version"] = json.RawMessage("1")
	root["documents"] = marshalRemoteFields(documents)
	raw := marshalRemoteFields(root)
	if _, err := decodeOrigins(raw); err != nil {
		return nil, err
	}
	return raw, nil
}
