package historyarchive

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var labelJSONFields = []string{"version", "id", "kind", "from", "to", "file_count", "raw_bytes", "bytes", "compression", "recipients", "sha256"}

// labelJSON avoids invoking BundleLabel's methods recursively.
type labelJSON BundleLabel

// DecodeLabel admits additive metadata while validating every known label field.
func DecodeLabel(raw []byte) (BundleLabel, error) {
	var label BundleLabel
	if err := label.UnmarshalJSON(raw); err != nil {
		return label, err
	}
	return label, nil
}

func (l *BundleLabel) UnmarshalJSON(raw []byte) error {
	known, extensions, err := decodeExtensibleObject(raw, labelJSONFields)
	if err != nil {
		return err
	}
	knownJSON, err := json.Marshal(known)
	if err != nil {
		return err
	}
	var decoded labelJSON
	if err = json.Unmarshal(knownJSON, &decoded); err != nil {
		return err
	}
	decoded.Extensions = extensions
	candidate := BundleLabel(decoded)
	if err = ValidateLabel(candidate); err != nil {
		return err
	}
	*l = candidate
	return nil
}

func (l BundleLabel) MarshalJSON() ([]byte, error) {
	knownJSON, err := json.Marshal(labelJSON(l))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObject(knownJSON, l.Extensions, labelJSONFields)
}

func decodeExtensibleObject(raw []byte, known []string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	return decodeExtensibleObjectLimit(raw, known, MaxMetadataBytes)
}

// decodeExtensibleObjectLimit bounds bytes and nesting, rejects duplicate keys
// at every depth, and requires exact spellings and non-null known fields.
func decodeExtensibleObjectLimit(raw []byte, known []string, limit int) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	return decodeExtensibleObjectSchema(raw, known, nil, limit)
}

// Optional known fields permit additive schema evolution without treating their
// recognized values as opaque extensions or weakening required-field checks.
func decodeExtensibleObjectSchema(raw []byte, required, optional []string, limit int) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	if len(raw) > limit {
		return nil, nil, errors.New("metadata exceeds byte limit")
	}
	var object map[string]json.RawMessage
	if err := decodeJSONNoDuplicateKeys(raw, &object, false); err != nil {
		return nil, nil, err
	}
	if object == nil {
		return nil, nil, errors.New("metadata must be an object")
	}
	fields := make(map[string]json.RawMessage, len(required)+len(optional))
	for _, key := range required {
		value, ok := object[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, nil, fmt.Errorf("required metadata field %q missing or null", key)
		}
		fields[key] = value
		delete(object, key)
	}
	for _, key := range optional {
		if value, ok := object[key]; ok {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, nil, fmt.Errorf("optional metadata field %q cannot be null", key)
			}
			fields[key] = value
			delete(object, key)
		}
	}
	known := append(append([]string(nil), required...), optional...)
	for key := range object {
		for _, name := range known {
			if strings.EqualFold(key, name) {
				return nil, nil, fmt.Errorf("invalid spelling for metadata field %q", key)
			}
		}
	}
	if len(object) == 0 {
		object = nil
	}
	return fields, object, nil
}

func marshalExtensibleObject(knownJSON []byte, extensions map[string]json.RawMessage, known []string) ([]byte, error) {
	return marshalExtensibleObjectLimit(knownJSON, extensions, known, MaxMetadataBytes)
}

// marshalExtensibleObjectLimit retains the known struct's field order and adds
// deterministic unknown fields, rejecting collisions and invalid raw values.
func marshalExtensibleObjectLimit(knownJSON []byte, extensions map[string]json.RawMessage, known []string, limit int) ([]byte, error) {
	if len(knownJSON) > limit {
		return nil, errors.New("metadata exceeds byte limit")
	}
	if len(extensions) == 0 {
		return knownJSON, nil
	}
	out := append([]byte(nil), knownJSON[:len(knownJSON)-1]...)
	keys := make([]string, 0, len(extensions))
	for key := range extensions {
		for _, name := range known {
			if strings.EqualFold(key, name) {
				return nil, fmt.Errorf("extension collides with known field %q", key)
			}
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := extensions[key]
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		if len(out)+len(encodedKey)+len(value)+3 > limit {
			return nil, errors.New("metadata exceeds byte limit")
		}
		var checked json.RawMessage
		if err := decodeJSONNoDuplicateKeys(value, &checked, false); err != nil {
			return nil, err
		}
		out = append(out, ',')
		out = append(out, encodedKey...)
		out = append(out, ':')
		out = append(out, value...)
	}
	out = append(out, '}')
	var object map[string]json.RawMessage
	if err := decodeJSONNoDuplicateKeys(out, &object, false); err != nil {
		return nil, err
	}
	return out, nil
}
