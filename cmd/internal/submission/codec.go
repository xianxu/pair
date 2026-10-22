package submission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// manifestDocument is immutable original wire evidence, retained only when a
// forward extension exists. Unknown values never participate in permissions or
// evidence-file enumeration and are not converted through float64.
type manifestDocument struct{ raw []byte }
type manifestWire Submission

// manifestSchema declares the wire shapes of owned fields. The typed Go
// decoder still owns numeric range/semantic validation; this preflight rejects
// missing required members, null and wrong-shaped values before json.Unmarshal
// can invent zero values.
// Unknown extensions remain raw JSON, outside this schema and its authority.
type manifestSchema struct {
	fields            map[string]*manifestSchema
	required          []string
	element           *manifestSchema
	tupleLength       int
	scalar            byte // s:string, b:boolean, i:integer, n:number
	nullable          bool
	dictionary        bool
	stringAlternative bool // PolicyEvidence's intentional string/object/null union
}

func scalarSchema(kind byte) *manifestSchema { return &manifestSchema{scalar: kind} }
func objectSchema(stringsOnly string, children map[string]*manifestSchema) *manifestSchema {
	s := &manifestSchema{fields: map[string]*manifestSchema{}}
	for _, name := range strings.Fields(stringsOnly) {
		s.fields[name] = scalarSchema('s')
	}
	for name, child := range children {
		s.fields[name] = child
	}
	return s
}
func typedObject(text, integers, unsigned, numbers, booleans string, children map[string]*manifestSchema) *manifestSchema {
	s := objectSchema(text, children)
	for _, group := range []struct {
		names string
		kind  byte
	}{{integers, 'i'}, {unsigned, 'i'}, {numbers, 'n'}, {booleans, 'b'}} {
		for _, name := range strings.Fields(group.names) {
			s.fields[name] = scalarSchema(group.kind)
		}
	}
	return s
}
func nullableSchema(s *manifestSchema) *manifestSchema {
	copy := *s
	copy.nullable = true
	return &copy
}

// requireMembers is context-specific: new sketch evidence requires explicit
// facts, while shared legacy note structures retain their omission defaults.
func requireMembers(s *manifestSchema, names string) *manifestSchema {
	copy := *s
	copy.required = strings.Fields(names)
	return &copy
}
func arraySchema(element *manifestSchema) *manifestSchema {
	return &manifestSchema{element: element, nullable: true}
}
func stringMapSchema() *manifestSchema {
	return &manifestSchema{element: scalarSchema('s'), dictionary: true, nullable: true}
}
func tupleSchema(length int, nullable bool) *manifestSchema {
	return &manifestSchema{tupleLength: length, nullable: nullable}
}

var submissionSchema = makeSubmissionSchema()

func makeSubmissionSchema() *manifestSchema {
	privacy := nullableSchema(objectSchema("origin privacy", nil))
	privacy.stringAlternative = true
	author := objectSchema("id name color", nil)
	page := nullableSchema(objectSchema("url title provenance", nil))
	viewport := requireMembers(typedObject("", "w h", "", "dpr", "", nil), "w h dpr")
	anchor := typedObject("tag route kind sel text", "node", "", "", "", map[string]*manifestSchema{"box": tupleSchema(4, true), "classes": arraySchema(scalarSchema('s')), "path": arraySchema(scalarSchema('s')), "stable": stringMapSchema()})
	fact := requireMembers(typedObject("tag id unavailable", "node parent", "", "", "truncated", map[string]*manifestSchema{"data": stringMapSchema()}), "node")
	facts := typedObject("", "inspected_nodes bytes", "", "", "truncated missing_layout", map[string]*manifestSchema{"anchors": arraySchema(fact), "ancestors": arraySchema(fact), "intersections": arraySchema(fact)})
	facts = requireMembers(facts, "anchors ancestors intersections truncated missing_layout inspected_nodes bytes")
	path := requireMembers(typedObject("id", "", "", "width", "truncated", map[string]*manifestSchema{"at": requireMembers(anchor, "node box"), "pts": arraySchema(tupleSchema(3, false))}), "id at width pts truncated")
	plan := typedObject("sketch", "width height", "", "scale", "clipped", map[string]*manifestSchema{"paths": arraySchema(path), "region": tupleSchema(4, false)})
	plan = requireMembers(plan, "sketch region scale width height clipped paths")
	image := requireMembers(objectSchema("file provenance", map[string]*manifestSchema{"limitations": arraySchema(scalarSchema('s')), "plan": plan}), "file provenance plan")
	sketchSource := typedObject("tab url title provenance", "t record_index doc", "", "", "", map[string]*manifestSchema{"vp": viewport, "scroll": tupleSchema(2, false)})
	sketchSource = requireMembers(sketchSource, "t record_index doc tab url title vp scroll provenance")
	sketch := typedObject("id recording_id digest note phase reason privacy privacy_origin", "history_t history_record_index", "next_sequence", "", "", map[string]*manifestSchema{"privacy_rule": requireMembers(privacy, "origin privacy"), "author": requireMembers(author, "id color"), "source": sketchSource, "elements": facts, "image": image})
	sketch = requireMembers(sketch, "id recording_id next_sequence digest author note history_t history_record_index phase source elements image privacy privacy_rule")
	entry := objectSchema("selector element text kind role", nil)
	boxText := typedObject("scope unavailable", "", "", "", "truncated", map[string]*manifestSchema{"region": tupleSchema(4, false), "visible_text": arraySchema(entry), "accessible_names": arraySchema(entry), "skipped": arraySchema(scalarSchema('s'))})
	crop := typedObject("file captured_at unavailable", "document", "", "", "clipped_to_viewport", map[string]*manifestSchema{"settlement": nullableSchema(objectSchema("status reason", nil)), "box": tupleSchema(4, false)})
	note := typedObject("privacy privacy_origin capture_id id text element replay", "source_document t", "revision", "", "", map[string]*manifestSchema{"privacy_rule": privacy, "page": page, "author": author, "elements": nullableSchema(facts), "box_text": boxText, "at": anchor, "before": arraySchema(scalarSchema('s')), "context": typedObject("text", "node landmark", "", "", "", map[string]*manifestSchema{"box": tupleSchema(4, true)}), "crop": crop})
	source := typedObject("kind recording_id recording_path tab_id", "document", "", "", "", map[string]*manifestSchema{"page": page})
	return typedObject("privacy privacy_origin how_to_read id delivery", "version", "", "", "superseded", map[string]*manifestSchema{"privacy_rule": privacy, "target": objectSchema("scope tag session nonce", nil), "source": source, "notes": arraySchema(note), "sketches": arraySchema(sketch)})
}

// DecodePreserving rejects ambiguous keys and invalid known wire shapes before
// typed decoding. New evidence requires explicit member presence; legacy
// omission defaults remain supported. Validate owns semantic values and
// dependencies at each store boundary.
func DecodePreserving(data []byte) (Submission, error) {
	if len(data) > MaxBundleBytes {
		return Submission{}, fmt.Errorf("manifest exceeds 64 MiB")
	}
	if err := checkJSONKeys(data, func(_ int, key string) error {
		// Retain the existing all-depth permission/source alias protection, including
		// inside extensions. Other known aliases are rejected at their exact path.
		for _, canonical := range []string{"privacy", "privacy_rule", "privacy_origin", "version", "source", "page", "source_document", "document", "url", "title", "provenance"} {
			if strings.EqualFold(key, canonical) && key != canonical {
				return fmt.Errorf("noncanonical privacy JSON key")
			}
		}
		return nil
	}); err != nil {
		return Submission{}, err
	}
	known, extended, err := filterManifest(data, submissionSchema)
	if err != nil {
		return Submission{}, err
	}
	var wire manifestWire
	if err = json.Unmarshal(known, &wire); err != nil {
		return Submission{}, err
	}
	out := Submission(wire)
	if extended {
		out.preserved = &manifestDocument{raw: bytes.Clone(data)}
	}
	return out, nil
}
func filterManifest(raw []byte, schema *manifestSchema) ([]byte, bool, error) {
	if schema == nil {
		return raw, false, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		if schema.nullable {
			return raw, false, nil
		}
		return nil, false, fmt.Errorf("null for nonnullable manifest value")
	}
	if len(trimmed) == 0 {
		return nil, false, fmt.Errorf("empty manifest value")
	}
	if schema.scalar != 0 {
		valid := false
		switch schema.scalar {
		case 's':
			valid = trimmed[0] == '"'
		case 'b':
			valid = bytes.Equal(trimmed, []byte("true")) || bytes.Equal(trimmed, []byte("false"))
		case 'n':
			valid = trimmed[0] == '-' || trimmed[0] >= '0' && trimmed[0] <= '9'
		case 'i':
			valid = trimmed[0] == '-' || trimmed[0] >= '0' && trimmed[0] <= '9'
			for _, c := range trimmed {
				if c != '-' && (c < '0' || c > '9') {
					valid = false
					break
				}
			}
		}
		if !valid {
			return nil, false, fmt.Errorf("wrong type for known manifest scalar")
		}
		return raw, false, nil
	}
	if schema.tupleLength > 0 {
		if trimmed[0] != '[' {
			return nil, false, fmt.Errorf("invalid fixed geometry tuple")
		}
		var tuple []json.RawMessage
		if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) != schema.tupleLength {
			return nil, false, fmt.Errorf("invalid fixed geometry tuple")
		}
		for i, v := range tuple {
			if _, _, err := filterManifest(v, scalarSchema('n')); err != nil {
				return nil, false, fmt.Errorf("geometry member %d: %w", i, err)
			}
		}
		return raw, false, nil
	}
	if schema.fields != nil || schema.dictionary {
		if schema.stringAlternative && trimmed[0] == '"' {
			return raw, false, nil
		}
		if trimmed[0] != '{' {
			return nil, false, fmt.Errorf("wrong type for known manifest object")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, false, err
		}
		for _, name := range schema.required {
			if _, present := fields[name]; !present {
				return nil, false, fmt.Errorf("missing required manifest field %q", name)
			}
		}
		extended := false
		for name, value := range fields {
			child, known := schema.fields[name]
			if schema.dictionary {
				child, known = schema.element, true
			}
			if !known {
				for canonical := range schema.fields {
					if strings.EqualFold(name, canonical) {
						return nil, false, fmt.Errorf("noncanonical manifest field %q", name)
					}
				}
				delete(fields, name)
				extended = true
				continue
			}
			filtered, more, err := filterManifest(value, child)
			if err != nil {
				return nil, false, fmt.Errorf("manifest field %s: %w", name, err)
			}
			fields[name] = filtered
			extended = extended || more
		}
		data, err := json.Marshal(fields)
		return data, extended, err
	}
	if schema.element != nil {
		if trimmed[0] != '[' {
			return nil, false, fmt.Errorf("wrong type for known manifest array")
		}
		var elements []json.RawMessage
		if err := json.Unmarshal(raw, &elements); err != nil {
			return nil, false, err
		}
		extended := false
		for i, value := range elements {
			filtered, more, err := filterManifest(value, schema.element)
			if err != nil {
				return nil, false, fmt.Errorf("manifest array member %d: %w", i, err)
			}
			elements[i] = filtered
			extended = extended || more
		}
		data, err := json.Marshal(elements)
		return data, extended, err
	}
	return nil, false, fmt.Errorf("missing manifest wire schema")
}

// EncodePreserving overlays current owned fields onto their original wire
// objects. A known omitted field is removed; only genuinely unknown fields are
// retained. Immutable evidence arrays must keep their order and identity.
func EncodePreserving(s Submission) ([]byte, error) {
	current, err := json.Marshal(manifestWire(s))
	if err != nil {
		return nil, err
	}
	if s.preserved == nil {
		return current, nil
	}
	return mergeManifest(s.preserved.raw, current, submissionSchema)
}
func mergeManifest(original, current []byte, schema *manifestSchema) ([]byte, error) {
	if schema == nil {
		return current, nil
	}
	oldTrim, newTrim := bytes.TrimSpace(original), bytes.TrimSpace(current)
	if schema.fields != nil && len(newTrim) > 0 && newTrim[0] == '{' {
		previous := map[string]json.RawMessage{}
		if len(oldTrim) > 0 && oldTrim[0] == '{' {
			if err := json.Unmarshal(original, &previous); err != nil {
				return nil, err
			}
		}
		var fresh map[string]json.RawMessage
		if err := json.Unmarshal(current, &fresh); err != nil {
			return nil, err
		}
		for name := range previous {
			if _, known := schema.fields[name]; known {
				if _, present := fresh[name]; !present {
					delete(previous, name)
				}
			}
		}
		for name, value := range fresh {
			merged, err := mergeManifest(previous[name], value, schema.fields[name])
			if err != nil {
				return nil, err
			}
			previous[name] = merged
		}
		return json.Marshal(previous)
	}
	if schema.element != nil && len(oldTrim) > 0 && oldTrim[0] == '[' && len(newTrim) > 0 && newTrim[0] == '[' {
		var previous, fresh []json.RawMessage
		if err := json.Unmarshal(original, &previous); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(current, &fresh); err != nil {
			return nil, err
		}
		if len(previous) != len(fresh) {
			return nil, fmt.Errorf("cannot reshape preserved evidence array")
		}
		for i := range fresh {
			if !sameManifestIdentity(previous[i], fresh[i]) {
				return nil, fmt.Errorf("cannot reassign preserved evidence array")
			}
			merged, err := mergeManifest(previous[i], fresh[i], schema.element)
			if err != nil {
				return nil, err
			}
			fresh[i] = merged
		}
		return json.Marshal(fresh)
	}
	return current, nil
}
func sameManifestIdentity(a, b []byte) bool {
	var left, right map[string]json.RawMessage
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return true
	}
	for _, key := range []string{"id", "node"} {
		av, aok := left[key]
		bv, bok := right[key]
		if aok != bok {
			return false
		}
		if !aok {
			continue
		}
		var astr, bstr string
		if json.Unmarshal(av, &astr) == nil && json.Unmarshal(bv, &bstr) == nil {
			if astr != bstr {
				return false
			}
		} else if !bytes.Equal(bytes.TrimSpace(av), bytes.TrimSpace(bv)) {
			return false
		}
	}
	return true
}
func (s Submission) MarshalJSON() ([]byte, error) { return EncodePreserving(s) }
