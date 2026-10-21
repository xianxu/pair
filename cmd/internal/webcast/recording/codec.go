package recording

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// Decode reads one record line. A kind this version doesn't know decodes to
// *Unknown, kept verbatim; a line without t or k is an error.
func Decode(line []byte) (Record, error) {
	var env struct {
		T *int64  `json:"t"`
		K *string `json:"k"`
	}
	if err := json.Unmarshal(line, &env); err != nil {
		return nil, err
	}
	if env.T == nil || env.K == nil || *env.K == "" {
		return nil, errors.New("a record needs t and k")
	}
	def, ok := kinds[*env.K]
	if !ok {
		return &Unknown{Header: Header{T: *env.T, K: *env.K}, Raw: bytes.Clone(line)}, nil
	}
	r := def.new()
	if sketchRecordSchema(*env.K) != nil {
		if err := decodeSketch(line, r); err != nil {
			return nil, fmt.Errorf("%s: %w", *env.K, err)
		}
		return r, nil
	}
	if err := json.Unmarshal(line, r); err != nil {
		return nil, fmt.Errorf("%s: %w", *env.K, err)
	}
	return r, nil
}

// Encode writes one record line (without the newline). The envelope's k is
// set from the record's kind, so a record never names the wrong one.
func Encode(r Record) ([]byte, error) {
	if u, ok := r.(*Unknown); ok {
		return bytes.Clone(u.Raw), nil
	}
	if sketchMetadata(r) != nil {
		return encodeSketch(r)
	}
	// Records are shared by fan-out readers. Copy the concrete envelope before
	// stamping its kind; the payload remains read-only and need not be cloned.
	v := reflect.ValueOf(r)
	copy := reflect.New(v.Elem().Type())
	copy.Elem().Set(v.Elem())
	wire := copy.Interface().(Record)
	wire.header().K = r.Kind()
	return json.Marshal(wire)
}

// Known reports whether a kind is part of this version.
func Known(kind string) bool {
	_, ok := kinds[kind]
	return ok
}

// FileOf is the recording file a kind lives in; an unknown kind stays with
// the events.
func FileOf(kind string) string {
	if def, ok := kinds[kind]; ok {
		return def.file
	}
	return FileEvents
}
