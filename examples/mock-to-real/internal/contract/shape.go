// Package contract compares required recursive JSON shapes, never fixture values.
package contract

import (
	"encoding/json"
	"example.com/mock-to-real/internal/jsonvalue"
	"fmt"
)

type Shape struct {
	kind   string
	fields map[string]*Shape
	item   *Shape
}

func Infer(fixture []byte) (*Shape, error) {
	v, e := jsonvalue.Decode(fixture)
	if e != nil {
		return nil, e
	}
	return inferWithVisit(v, nil)
}

// The optional callback counts each visited input or comparison node. Production
// callers share this exact traversal with instrumentation disabled.
func inferWithVisit(v any, visit func()) (*Shape, error) {
	if visit != nil {
		visit()
	}
	s := &Shape{kind: kind(v)}
	switch v := v.(type) {
	case map[string]any:
		s.fields = map[string]*Shape{}
		for k, x := range v {
			c, e := inferWithVisit(x, visit)
			if e != nil {
				return nil, fmt.Errorf("%s: %w", k, e)
			}
			s.fields[k] = c
		}
	case []any:
		if len(v) == 0 {
			return nil, fmt.Errorf("fixture arrays need a representative item")
		}
		var e error
		s.item, e = inferWithVisit(v[0], visit)
		if e != nil {
			return nil, e
		}
		for _, x := range v[1:] {
			c, e := inferWithVisit(x, visit)
			if e != nil {
				return nil, e
			}
			if !sameShape(s.item, c, visit) {
				return nil, fmt.Errorf("heterogeneous fixture array")
			}
		}
	}
	return s, nil
}
func kind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case json.Number:
		return "number"
	case []any:
		return "array"
	default:
		return "object"
	}
}
func (s *Shape) Match(actual []byte) error {
	v, e := jsonvalue.Decode(actual)
	if e != nil {
		return e
	}
	return s.matchWithVisit(v, "$", nil)
}
func (s *Shape) matchWithVisit(v any, path string, visit func()) error {
	if visit != nil {
		visit()
	}
	if s == nil {
		return fmt.Errorf("nil shape")
	}
	if kind(v) != s.kind {
		return fmt.Errorf("%s: need %s, got %s", path, s.kind, kind(v))
	}
	switch v := v.(type) {
	case map[string]any:
		for k, c := range s.fields {
			x, ok := v[k]
			if !ok {
				return fmt.Errorf("%s.%s: required field missing", path, k)
			}
			if e := c.matchWithVisit(x, path+"."+k, visit); e != nil {
				return e
			}
		}
	case []any:
		for i, x := range v {
			if e := s.item.matchWithVisit(x, fmt.Sprintf("%s[%d]", path, i), visit); e != nil {
				return e
			}
		}
	}
	return nil
}

// sameShape checks prototype compatibility without an opaque reflection walk,
// allowing the inference work guard to account for comparisons as well.
func sameShape(a, b *Shape, visit func()) bool {
	if visit != nil {
		visit()
	}
	if a == nil || b == nil {
		return a == b
	}
	if a.kind != b.kind || len(a.fields) != len(b.fields) {
		return false
	}
	for key, field := range a.fields {
		other, ok := b.fields[key]
		if !ok || !sameShape(field, other, visit) {
			return false
		}
	}
	if a.kind == "array" {
		return sameShape(a.item, b.item, visit)
	}
	return true
}
