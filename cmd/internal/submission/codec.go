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

// manifestSchema lists owned object fields, not an alternate value decoder.
// The ordinary typed decoder remains responsible for known value types. A nil
// child denotes an opaque scalar/map/tuple, whose contents have no struct fields.
type manifestSchema struct {
	fields      map[string]*manifestSchema
	element     *manifestSchema
	tupleLength int
	nullable    bool
}

func objectSchema(fields string, children map[string]*manifestSchema) *manifestSchema {
	s := &manifestSchema{fields: map[string]*manifestSchema{}}
	for _, name := range strings.Fields(fields) {
		s.fields[name] = nil
	}
	for name, child := range children {
		s.fields[name] = child
	}
	return s
}
func arraySchema(element *manifestSchema) *manifestSchema { return &manifestSchema{element: element} }

func tupleSchema(length int, nullable bool) *manifestSchema {
	return &manifestSchema{tupleLength: length, nullable: nullable}
}

var submissionSchema = makeSubmissionSchema()

func makeSubmissionSchema() *manifestSchema {
	privacy := objectSchema("origin privacy", nil)
	author := objectSchema("id name color", nil)
	page := objectSchema("url title provenance", nil)
	viewport := objectSchema("w h dpr", nil)
	anchor := objectSchema("tag classes stable route node kind path sel text", map[string]*manifestSchema{"box": tupleSchema(4, true)})
	fact := objectSchema("node parent tag id data unavailable truncated", nil)
	facts := objectSchema("truncated missing_layout inspected_nodes bytes", map[string]*manifestSchema{"anchors": arraySchema(fact), "ancestors": arraySchema(fact), "intersections": arraySchema(fact)})
	path := objectSchema("id width truncated", map[string]*manifestSchema{"at": anchor, "pts": arraySchema(tupleSchema(3, false))})
	plan := objectSchema("sketch region scale width height clipped", map[string]*manifestSchema{"paths": arraySchema(path), "region": tupleSchema(4, false)})
	image := objectSchema("file provenance limitations", map[string]*manifestSchema{"plan": plan})
	sketchSource := objectSchema("t record_index doc tab url title scroll provenance", map[string]*manifestSchema{"vp": viewport, "scroll": tupleSchema(2, false)})
	sketch := objectSchema("id recording_id next_sequence digest note history_t history_record_index phase reason privacy privacy_origin", map[string]*manifestSchema{"privacy_rule": privacy, "author": author, "source": sketchSource, "elements": facts, "image": image})
	entry := objectSchema("selector element text kind role", nil)
	boxText := objectSchema("region truncated scope skipped unavailable", map[string]*manifestSchema{"visible_text": arraySchema(entry), "accessible_names": arraySchema(entry)})
	crop := objectSchema("file clipped_to_viewport box captured_at document unavailable", map[string]*manifestSchema{"settlement": objectSchema("status reason", nil), "box": tupleSchema(4, false)})
	note := objectSchema("privacy privacy_origin capture_id source_document revision id text t element before replay", map[string]*manifestSchema{"privacy_rule": privacy, "page": page, "author": author, "elements": facts, "box_text": boxText, "at": anchor, "context": objectSchema("node landmark text", map[string]*manifestSchema{"box": tupleSchema(4, true)}), "crop": crop})
	source := objectSchema("kind recording_id recording_path tab_id document", map[string]*manifestSchema{"page": page})
	return objectSchema("privacy privacy_origin how_to_read version id delivery superseded", map[string]*manifestSchema{"privacy_rule": privacy, "target": objectSchema("scope tag session nonce", nil), "source": source, "notes": arraySchema(note), "sketches": arraySchema(sketch)})
}

// DecodePreserving rejects duplicate keys and case aliases before decoding
// known fields with their original Go types. Validate enforces semantic required
// fields at each store boundary, as for earlier versions.
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
	if schema.tupleLength > 0 {
		if schema.nullable && bytes.Equal(trimmed, []byte("null")) {
			return raw, false, nil
		}
		var tuple []json.RawMessage
		if err := json.Unmarshal(raw, &tuple); err != nil || len(tuple) != schema.tupleLength {
			return nil, false, fmt.Errorf("invalid fixed geometry tuple")
		}
		return raw, false, nil
	}
	if schema.fields != nil && len(trimmed) > 0 && trimmed[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, false, err
		}
		extended := false
		for name, value := range fields {
			child, known := schema.fields[name]
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
				return nil, false, err
			}
			fields[name] = filtered
			extended = extended || more
		}
		data, err := json.Marshal(fields)
		return data, extended, err
	}
	if schema.element != nil && len(trimmed) > 0 && trimmed[0] == '[' {
		var elements []json.RawMessage
		if err := json.Unmarshal(raw, &elements); err != nil {
			return nil, false, err
		}
		extended := false
		for i, value := range elements {
			filtered, more, err := filterManifest(value, schema.element)
			if err != nil {
				return nil, false, err
			}
			elements[i] = filtered
			extended = extended || more
		}
		data, err := json.Marshal(elements)
		return data, extended, err
	}
	return raw, false, nil
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
