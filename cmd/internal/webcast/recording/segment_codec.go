package recording

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Each object preserves its own additive values; existing segment fields keep
// their historical nullability and optionality. Only page_load adds validation.
type segmentWire struct {
	unknown  map[string]json.RawMessage
	children map[string]*segmentWire
}
type segmentSchema struct {
	fields   map[string]bool
	children map[string]*segmentSchema
}

var segmentTabSchema = &segmentSchema{fields: map[string]bool{"id": true, "url": true, "title": true}}
var segmentVPSchema = &segmentSchema{fields: map[string]bool{"w": true, "h": true, "dpr": true}}
var segmentSchemaV1 = &segmentSchema{fields: map[string]bool{"t": true, "k": true, "seg": true, "why": true, "tab": true, "vp": true, "page_load": true}, children: map[string]*segmentSchema{"tab": segmentTabSchema, "vp": segmentVPSchema}}

func splitSegmentObject(raw []byte, schema *segmentSchema) ([]byte, *segmentWire, error) {
	owned, unknown, err := strictjson.SplitObjectPreserveNumbers(raw, schema.fields, schema.fields)
	if err != nil {
		return nil, nil, err
	}
	if err = checkSketchOwnedKeyUnicode(raw, schema.fields); err != nil {
		return nil, nil, err
	}
	wire := &segmentWire{unknown: unknown, children: map[string]*segmentWire{}}
	for key, value := range owned {
		if key == "page_load" {
			if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return nil, nil, errors.New("null page_load")
			}
			continue
		}
		if child := schema.children[key]; child != nil && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			clean, w, err := splitSegmentObject(value, child)
			if err != nil {
				return nil, nil, err
			}
			owned[key] = clean
			if w != nil {
				wire.children[key] = w
			}
		} else if err := strictjson.ValidateUnicode(value); err != nil {
			return nil, nil, err
		}
	}
	clean, err := strictjson.EncodeObjectPreserveNumbers(owned, nil, schema.fields)
	if len(wire.unknown) == 0 && len(wire.children) == 0 {
		wire = nil
	}
	return clean, wire, err
}
func decodeSegment(raw []byte, r *Seg) error {
	clean, wire, err := splitSegmentObject(raw, segmentSchemaV1)
	if err != nil {
		return err
	}
	// Unknown values have been split out recursively. Unmarshal retains existing
	// legacy field behavior rather than tightening unrelated null/required rules.
	if err = json.Unmarshal(clean, r); err != nil {
		return err
	}
	if r.PageLoad != nil {
		if err = r.PageLoad.Validate(); err != nil {
			return err
		}
		if r.PageLoad.Target != r.Tab.ID {
			return errors.New("page load target differs from tab")
		}
	}
	r.wire = wire
	return nil
}
func mergeSegmentObject(raw []byte, wire *segmentWire, schema *segmentSchema) ([]byte, error) {
	if wire == nil {
		return raw, nil
	}
	var owned map[string]json.RawMessage
	if err := json.Unmarshal(raw, &owned); err != nil {
		return nil, err
	}
	for key, child := range wire.children {
		value := owned[key]
		if len(value) == 0 || bytes.Equal(value, []byte("null")) {
			continue
		}
		merged, err := mergeSegmentObject(value, child, schema.children[key])
		if err != nil {
			return nil, err
		}
		owned[key] = merged
	}
	return strictjson.EncodeObjectPreserveNumbers(owned, wire.unknown, schema.fields)
}
func encodeSegment(r *Seg) ([]byte, error) {
	if r.PageLoad != nil {
		if err := r.PageLoad.Validate(); err != nil {
			return nil, err
		}
		if r.PageLoad.Target != r.Tab.ID {
			return nil, errors.New("page load target differs from tab")
		}
	}
	copy := *r
	copy.K = r.Kind()
	raw, err := json.Marshal(&copy)
	if err != nil {
		return nil, err
	}
	// Outer json.Marshal compacts custom marshaler output. Restore the load's
	// raw extensions after that pass, just like other nested segment objects.
	if r.PageLoad != nil {
		var owned map[string]json.RawMessage
		json.Unmarshal(raw, &owned)
		owned["page_load"], err = r.PageLoad.MarshalJSON()
		if err != nil {
			return nil, err
		}
		raw, err = strictjson.EncodeObjectPreserveNumbers(owned, nil, segmentSchemaV1.fields)
		if err != nil {
			return nil, err
		}
	}
	return mergeSegmentObject(raw, r.wire, segmentSchemaV1)
}
