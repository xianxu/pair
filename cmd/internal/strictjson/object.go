package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Additive records (pair#695 §2a): a reader tolerates members it does not own
// and carries them, byte for byte, through its own rewrite, so an older build
// that rewrites a record cannot erase a newer build's field. Duplicate keys,
// invalid JSON and null owned members stay refused; the owner still decodes
// its own members strictly (Decode over the owned subset).

// KnownFields is the set of JSON member names a struct type owns: its exported
// fields, by json tag name, skipping `json:"-"`.
func KnownFields(t reflect.Type) map[string]bool {
	m := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "-" || field.PkgPath != "" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" {
			name = field.Name
		}
		m[name] = true
	}
	return m
}

// SplitObject parses one JSON object into the members the caller owns and
// the rest, which it copies verbatim. It refuses duplicate keys at any depth,
// a null or non-object document, and a null owned member not in nullable.
func SplitObject(raw []byte, known, nullable map[string]bool) (owned, unknown map[string]json.RawMessage, err error) {
	return splitObjectMode(raw, known, nullable, false)
}

// SplitObjectPreserveNumbers retains opaque JSON number tokens without a
// float64 range check. Known values still require the caller's typed decoder.
// Duplicate and nullability checks have the same rules as SplitObject.
func SplitObjectPreserveNumbers(raw []byte, known, nullable map[string]bool) (owned, unknown map[string]json.RawMessage, err error) {
	return splitObjectMode(raw, known, nullable, true)
}

func splitObjectMode(raw []byte, known, nullable map[string]bool, preserveNumbers bool) (owned, unknown map[string]json.RawMessage, err error) {
	if err := rejectDuplicateKeysMode(raw, preserveNumbers); err != nil {
		return nil, nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, nil, errors.New("JSON object required")
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &all); err != nil {
		return nil, nil, err
	}
	owned = map[string]json.RawMessage{}
	unknown = map[string]json.RawMessage{}
	for k, v := range all {
		if !known[k] {
			unknown[k] = append(json.RawMessage(nil), v...)
			continue
		}
		if !nullable[k] && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return nil, nil, fmt.Errorf("JSON member %s cannot be null", k)
		}
		owned[k] = v
	}
	return owned, unknown, nil
}

// EncodeObject writes owned members and the preserved unknown ones as one
// object, sorted by name. Unknown values are appended verbatim, never
// re-marshaled (which would compact them); an unknown member named like an
// owned one is dropped, so it can never shadow the owner's value. A preserved
// value that is not one clean JSON value refuses the write.
func EncodeObject(owned, unknown map[string]json.RawMessage, known map[string]bool) ([]byte, error) {
	return encodeObjectMode(owned, unknown, known, false)
}

// EncodeObjectPreserveNumbers writes opaque unknown numbers verbatim, including
// values outside float64 range. As with EncodeObject, owned values come from the
// caller's typed encoder; unknown values are checked for syntax and duplicates.
func EncodeObjectPreserveNumbers(owned, unknown map[string]json.RawMessage, known map[string]bool) ([]byte, error) {
	return encodeObjectMode(owned, unknown, known, true)
}

func encodeObjectMode(owned, unknown map[string]json.RawMessage, known map[string]bool, preserveNumbers bool) ([]byte, error) {
	merged := make(map[string]json.RawMessage, len(owned)+len(unknown))
	for k, v := range owned {
		merged[k] = v
	}
	for k, v := range unknown {
		if known[k] {
			continue
		}
		if err := rejectDuplicateKeysMode(v, preserveNumbers); err != nil {
			return nil, fmt.Errorf("preserved JSON member %s: %w", k, err)
		}
		merged[k] = v
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		name, _ := json.Marshal(k)
		b.Write(name)
		b.WriteByte(':')
		b.Write(merged[k])
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// CloneRaw deep-copies a preserved member map; nil stays nil.
func CloneRaw(m map[string]json.RawMessage) map[string]json.RawMessage {
	if m == nil {
		return nil
	}
	n := make(map[string]json.RawMessage, len(m))
	for k, v := range m {
		n[k] = append(json.RawMessage(nil), v...)
	}
	return n
}

// MergeRaw copies a and then b over it: a newer value replaces only the same
// member, and members b lacks survive. nil when both are nil.
func MergeRaw(a, b map[string]json.RawMessage) map[string]json.RawMessage {
	n := CloneRaw(a)
	if n == nil && b != nil {
		n = map[string]json.RawMessage{}
	}
	for k, v := range b {
		n[k] = append(json.RawMessage(nil), v...)
	}
	return n
}

// AppendMembers adds preserved unknown members to an already encoded object,
// keeping its formatting (for example json.MarshalIndent output) and writing
// each value verbatim, sorted by name. Members named like owned ones are
// dropped; a value that is not one clean JSON value refuses.
func AppendMembers(encoded []byte, unknown map[string]json.RawMessage, known map[string]bool) ([]byte, error) {
	keys := make([]string, 0, len(unknown))
	for k, v := range unknown {
		if known[k] {
			continue
		}
		if err := rejectDuplicateKeys(v); err != nil {
			return nil, fmt.Errorf("preserved JSON member %s: %w", k, err)
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return encoded, nil
	}
	body := bytes.TrimRight(encoded, " \t\r\n")
	if len(body) == 0 || body[len(body)-1] != '}' {
		return nil, errors.New("encoded value is not a JSON object")
	}
	body = bytes.TrimRight(body[:len(body)-1], " \t\r\n")
	empty := len(body) > 0 && body[len(body)-1] == '{'
	sort.Strings(keys)
	out := append([]byte(nil), body...)
	for i, k := range keys {
		if i > 0 || !empty {
			out = append(out, ',')
		}
		name, _ := json.Marshal(k)
		out = append(out, "\n  "...)
		out = append(out, name...)
		out = append(out, ": "...)
		out = append(out, unknown[k]...)
	}
	out = append(out, "\n}"...)
	return out, nil
}

// DecodeKnown is every preserving decoder's one sequence: split raw, decode
// the known members strictly into target (known members may not be null),
// and return the members it does not own, nil when there are none.
func DecodeKnown(raw []byte, known map[string]bool, target any) (map[string]json.RawMessage, error) {
	owned, unknown, err := SplitObject(raw, known, known)
	if err != nil {
		return nil, err
	}
	ownedRaw, err := EncodeObject(owned, nil, known)
	if err != nil {
		return nil, err
	}
	if err := Decode(ownedRaw, target); err != nil {
		return nil, err
	}
	if len(unknown) == 0 {
		return nil, nil
	}
	return unknown, nil
}
