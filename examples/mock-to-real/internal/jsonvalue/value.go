// Package jsonvalue decodes bounded, unambiguous JSON without losing number types.
package jsonvalue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const MaxBytes = 1 << 20

func Decode(data []byte) (any, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("JSON exceeds 1 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	v, e := value(d, 0)
	if e != nil {
		return nil, e
	}
	if _, e := d.Token(); e != io.EOF {
		return nil, fmt.Errorf("trailing JSON")
	}
	return v, nil
}
func value(d *json.Decoder, depth int) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("JSON exceeds depth 64")
	}
	t, e := d.Token()
	if e != nil {
		return nil, e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	switch delim {
	case '{':
		m := map[string]any{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return nil, e
			}
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("object key")
			}
			if _, ok := m[key]; ok {
				return nil, fmt.Errorf("duplicate key %q", key)
			}
			v, e := value(d, depth+1)
			if e != nil {
				return nil, e
			}
			m[key] = v
		}
		if _, e := d.Token(); e != nil {
			return nil, e
		}
		return m, nil
	case '[':
		a := []any{}
		for d.More() {
			v, e := value(d, depth+1)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		if _, e := d.Token(); e != nil {
			return nil, e
		}
		return a, nil
	}
	return nil, fmt.Errorf("unexpected delimiter")
}
