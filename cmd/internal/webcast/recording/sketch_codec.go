package recording

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// sketchWire keeps additive values beside, rather than inside, authoritative
// typed fields. Each object owns its own unknown raw values.
type sketchWire struct {
	unknown  map[string]json.RawMessage
	children map[string]*sketchWire
	known    map[string]bool
}
type sketchSchema struct {
	fields, required []string
	children         map[string]*sketchSchema
	stable           bool
}

var sketchAuthorSchema = &sketchSchema{fields: []string{"id", "name", "color"}, required: []string{"id", "color"}}
var sketchVPSchema = &sketchSchema{fields: []string{"w", "h", "dpr"}, required: []string{"w", "h", "dpr"}}
var sketchStableSchema = &sketchSchema{stable: true}
var sketchAnchorSchema = &sketchSchema{fields: []string{"tag", "classes", "stable", "route", "node", "kind", "path", "sel", "text", "box"}, required: []string{"node", "box"}, children: map[string]*sketchSchema{"stable": sketchStableSchema}}

func sketchRecordSchema(kind string) *sketchSchema {
	fields := []string{"t", "k", "operation", "sequence", "digest"}
	var payload []string
	var children map[string]*sketchSchema
	switch kind {
	case "sketch-begin":
		payload = []string{"id", "author", "owner", "doc", "tab", "vp"}
		children = map[string]*sketchSchema{"author": sketchAuthorSchema, "vp": sketchVPSchema}
	case "sketch-stroke-begin":
		payload = []string{"sketch", "id", "at", "width"}
		children = map[string]*sketchSchema{"at": sketchAnchorSchema}
	case "sketch-points":
		payload = []string{"sketch", "id", "chunk", "pts"}
	case "sketch-stroke-end", "sketch-undo", "sketch-redo":
		payload = []string{"sketch", "id"}
	case "sketch-label":
		payload = []string{"sketch", "label"}
	case "sketch-interrupt":
		payload = []string{"sketch", "reason"}
	default:
		return nil
	}
	fields = append(fields, payload...)
	return &sketchSchema{fields: fields, required: fields, children: children}
}
func sketchFieldSet(fields []string) map[string]bool {
	m := map[string]bool{}
	for _, k := range fields {
		m[k] = true
	}
	return m
}
func splitSketchObject(raw []byte, schema *sketchSchema) ([]byte, *sketchWire, error) {
	known := sketchFieldSet(schema.fields)
	if schema.stable {
		for _, k := range []string{"id", "data-testid", "data-test", "data-cy"} {
			known[k] = true
		}
		// Stable is an open string map today. A future structured member belongs
		// to its future reader and remains raw; string entries remain semantic.
		var all map[string]json.RawMessage
		if err := json.Unmarshal(raw, &all); err != nil {
			return nil, nil, err
		}
		for k, v := range all {
			var s string
			if len(v) > 0 && v[0] == '"' && json.Unmarshal(v, &s) == nil {
				known[k] = true
			}
		}
	}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := checkSketchOwnedKeyUnicode(raw, known); err != nil {
		return nil, nil, err
	}
	for _, k := range schema.required {
		if _, ok := owned[k]; !ok {
			return nil, nil, fmt.Errorf("missing %s", k)
		}
	}
	wire := &sketchWire{unknown: unknown, known: known, children: map[string]*sketchWire{}}
	for k, sub := range schema.children {
		if v, ok := owned[k]; ok {
			clean, child, err := splitSketchObject(v, sub)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", k, err)
			}
			owned[k] = clean
			if child != nil {
				wire.children[k] = child
			}
		}
	}
	for key, value := range owned {
		if err := checkSketchUnicode(value); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", key, err)
		}
	}
	if len(unknown) == 0 {
		wire.unknown = nil
	}
	if len(wire.children) == 0 {
		wire.children = nil
	}
	clean, err := strictjson.EncodeObject(owned, nil, known)
	if err != nil {
		return nil, nil, err
	}
	// With no additive values, do not attach wire state. Plain typed values
	// and a decoded equivalent then remain deeply equal.
	if len(wire.unknown) == 0 && len(wire.children) == 0 {
		return clean, nil, nil
	}
	return clean, wire, nil
}

// checkSketchUnicode inspects original JSON string bytes before encoding/json
// can replace malformed UTF-8 or unpaired UTF-16 escapes with U+FFFD. Its input
// contains only owned values; additive unknown values retain their raw bytes.
func checkSketchUnicode(raw []byte) error {
	if !utf8.Valid(raw) {
		return errors.New("invalid UTF-8 in owned sketch string")
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for i < len(raw) && raw[i] != '"' {
			if raw[i] != '\\' {
				i++
				continue
			}
			if i+1 >= len(raw) {
				return errors.New("truncated JSON escape")
			}
			if raw[i+1] != 'u' {
				i += 2
				continue
			}
			if i+6 > len(raw) {
				return errors.New("truncated Unicode escape")
			}
			unit, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err != nil {
				return err
			}
			i += 6
			if unit >= 0xdc00 && unit <= 0xdfff {
				return errors.New("unpaired low surrogate in owned sketch string")
			}
			if unit < 0xd800 || unit > 0xdbff {
				continue
			}
			if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
				return errors.New("unpaired high surrogate in owned sketch string")
			}
			low, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err != nil {
				return err
			}
			if low < 0xdc00 || low > 0xdfff {
				return errors.New("unpaired high surrogate in owned sketch string")
			}
			i += 6
		}
	}
	return nil
}

// A stable-map key is itself owned when its value is a semantic string.
// Inspect the original key token too; a decoded map key has already lost
// malformed source bytes. Unknown member names remain outside this check.
func checkSketchOwnedKeyUnicode(raw []byte, known map[string]bool) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return err
	}
	for dec.More() {
		start := dec.InputOffset()
		token, err := dec.Token()
		if err != nil {
			return err
		}
		end := dec.InputOffset()
		key, ok := token.(string)
		if !ok {
			return errors.New("JSON member name required")
		}
		if known[key] {
			if err := checkSketchUnicode(bytes.TrimLeft(raw[start:end], ", \t\r\n")); err != nil {
				return err
			}
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
	}
	return nil
}

func checkSketchTuples(raw []byte, r Record) error {
	var m map[string]json.RawMessage
	json.Unmarshal(raw, &m)
	switch r.(type) {
	case *SketchPoints:
		var points []json.RawMessage
		if err := json.Unmarshal(m["pts"], &points); err != nil {
			return err
		}
		for _, p := range points {
			var tuple []json.RawMessage
			if err := json.Unmarshal(p, &tuple); err != nil {
				return err
			}
			if len(tuple) != 3 {
				return errors.New("pressure point requires exactly three numbers")
			}
			for _, v := range tuple {
				if string(v) == "null" {
					return errors.New("null pressure coordinate")
				}
			}
		}
	case *SketchStrokeBegin:
		var at map[string]json.RawMessage
		json.Unmarshal(m["at"], &at)
		for _, key := range []string{"path", "classes"} {
			if value, ok := at[key]; ok {
				var entries []json.RawMessage
				if err := json.Unmarshal(value, &entries); err != nil {
					return err
				}
				for _, entry := range entries {
					if string(entry) == "null" {
						return fmt.Errorf("null anchor %s entry", key)
					}
				}
			}
		}
		var box []json.RawMessage
		if err := json.Unmarshal(at["box"], &box); err != nil {
			return err
		}
		if len(box) != 4 {
			return errors.New("anchor box requires exactly four numbers")
		}
		for _, v := range box {
			if string(v) == "null" {
				return errors.New("null box coordinate")
			}
		}
	}
	return nil
}
func decodeSketch(raw []byte, r Record) error {
	clean, wire, err := splitSketchObject(raw, sketchRecordSchema(r.Kind()))
	if err != nil {
		return err
	}
	if err = checkSketchTuples(clean, r); err != nil {
		return err
	}
	if err = strictjson.Decode(clean, r); err != nil {
		return err
	}
	if err = r.check(); err != nil {
		return err
	}
	sketchMetadata(r).wire = wire
	return nil
}
func mergeSketchObject(raw []byte, wire *sketchWire) ([]byte, error) {
	if wire == nil {
		return raw, nil
	}
	var owned map[string]json.RawMessage
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	for k, child := range wire.children {
		sub, ok := owned[k]
		if !ok {
			sub = []byte("{}")
		}
		merged, err := mergeSketchObject(sub, child)
		if err != nil {
			return nil, err
		}
		owned[k] = merged
	}
	known := make(map[string]bool, len(wire.known)+len(owned))
	for k, v := range wire.known {
		known[k] = v
	}
	for k := range owned {
		known[k] = true
	}
	return strictjson.EncodeObject(owned, wire.unknown, known)
}
func encodeSketch(r Record) ([]byte, error) {
	v := reflect.ValueOf(r)
	copy := reflect.New(v.Elem().Type())
	copy.Elem().Set(v.Elem())
	record := copy.Interface().(Record)
	record.header().K = r.Kind()
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return mergeSketchObject(raw, sketchMetadata(r).wire)
}

// SketchCanonical is the canonical known request projection used by receipts
// and digest generation. It excludes writer time, kind, claimed digest and
// additive data. encoding/json sorts keys and uses ECMAScript number cutoffs.
func SketchCanonical(r Record) ([]byte, error) {
	if sketchMetadata(r) == nil {
		return nil, errors.New("not a sketch record")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	delete(m, "t")
	delete(m, "k")
	delete(m, "digest")
	normalizeSketchZero(m)
	return json.Marshal(m)
}
func normalizeSketchZero(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if f, ok := v.(float64); ok && f == 0 {
				x[k] = float64(0)
			} else {
				normalizeSketchZero(v)
			}
		}
	case []any:
		for i, v := range x {
			if f, ok := v.(float64); ok && f == 0 {
				x[i] = float64(0)
			} else {
				normalizeSketchZero(v)
			}
		}
	}
}

// SketchDigest generates the claimed lowercase SHA256 for a semantic request.
// Reducers separately compare actual semantics, so a forged same digest never
// makes a different request an identical retry.
func SketchDigest(r Record) (string, error) {
	raw, err := SketchCanonical(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// SketchSemantic includes the concrete kind, which is outside the digest.
func SketchSemantic(r Record) (string, error) {
	raw, err := SketchCanonical(r)
	if err != nil {
		return "", err
	}
	return r.Kind() + ":" + string(raw), nil
}
