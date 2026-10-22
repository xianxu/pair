package editorendpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// ErrEndpointCapacity is a deterministic encoded-state byte-cap refusal.
var ErrEndpointCapacity = errors.New("endpoint state exceeds 8 MiB")

func decodeObject(raw []byte, target any) (map[string]json.RawMessage, error) {
	known := strictjson.KnownFields(reflect.TypeOf(target).Elem())
	owned, unknown, e := strictjson.SplitObjectPreserveNumbers(raw, known, map[string]bool{"Current": true})
	if e != nil {
		return nil, e
	}
	for k := range known {
		if _, ok := owned[k]; !ok && k != "Current" {
			return nil, fmt.Errorf("missing required field %s", k)
		}
	}
	body, e := strictjson.EncodeObjectPreserveNumbers(owned, nil, known)
	if e != nil {
		return nil, e
	}
	// Exact owned keys, required members, non-null values and all nested
	// duplicates have already been checked. Typed Unmarshal retains raw child
	// messages without imposing float64 range on their opaque future fields.
	if e = json.Unmarshal(body, target); e != nil {
		return nil, e
	}
	return unknown, nil
}
func encodeObject(value any, unknown map[string]json.RawMessage, override map[string]json.RawMessage) ([]byte, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	var owned map[string]json.RawMessage
	if e = json.Unmarshal(raw, &owned); e != nil {
		return nil, e
	}
	for k, v := range override {
		owned[k] = v
	}
	return strictjson.EncodeObjectPreserveNumbers(owned, unknown, strictjson.KnownFields(reflect.TypeOf(value)))
}
func rawArray(items [][]byte) []byte {
	return append(append([]byte{'['}, bytes.Join(items, []byte{','})...), ']')
}

// DecodeEndpointState rejects invalid authority rather than treating it as empty.
// Additive unknown members at all owned object levels survive exact raw rewrites.
func DecodeEndpointState(raw []byte) (EndpointState, error) {
	var s EndpointState
	if len(raw) > MaxStateBytes {
		return s, errors.New("endpoint state exceeds 8 MiB")
	}
	// Decode children individually so their additive fields are retained as raw JSON.
	type root struct {
		Version   int               `json:"Version"`
		Namespace string            `json:"Namespace"`
		Revision  uint64            `json:"Revision"`
		Records   []json.RawMessage `json:"Records"`
	}
	var r root
	unknown, e := decodeObject(raw, &r)
	if e != nil {
		return s, e
	}
	s = EndpointState{Version: r.Version, Namespace: r.Namespace, Revision: r.Revision, Records: make([]EndpointRecord, 0, len(r.Records)), Unknown: unknown}
	if len(r.Records) > MaxRecords {
		return EndpointState{}, errors.New("too many endpoint records")
	}
	for _, rawRecord := range r.Records {
		type wire struct {
			Binding json.RawMessage `json:"Binding"`
			Current json.RawMessage `json:"Current"`
			Status  string          `json:"Status"`
		}
		var w wire
		u, e := decodeObject(rawRecord, &w)
		if e != nil {
			return EndpointState{}, e
		}
		var b EndpointBinding
		bu, e := decodeObject(w.Binding, &b)
		if e != nil {
			return EndpointState{}, e
		}
		b.Unknown = bu
		record := EndpointRecord{Binding: b, Status: w.Status, Unknown: u}
		if len(w.Current) > 0 && !bytes.Equal(bytes.TrimSpace(w.Current), []byte("null")) {
			var cb EndpointBinding
			cu, e := decodeObject(w.Current, &cb)
			if e != nil {
				return EndpointState{}, e
			}
			cb.Unknown = cu
			record.Current = &cb
		}
		s.Records = append(s.Records, record)
	}
	if e := validateState(s); e != nil {
		return EndpointState{}, e
	}
	return s, nil
}
func EncodeEndpointState(s EndpointState) ([]byte, error) {
	if e := validateState(s); e != nil {
		return nil, e
	}
	items := make([][]byte, 0, len(s.Records))
	for _, x := range s.Records {
		binding, e := encodeObject(x.Binding, x.Binding.Unknown, nil)
		if e != nil {
			return nil, e
		}
		overrides := map[string]json.RawMessage{"Binding": binding}
		if x.Current != nil {
			current, e := encodeObject(*x.Current, x.Current.Unknown, nil)
			if e != nil {
				return nil, e
			}
			overrides["Current"] = current
		}
		record, e := encodeObject(x, x.Unknown, overrides)
		if e != nil {
			return nil, e
		}
		items = append(items, record)
	}
	raw, e := encodeObject(s, s.Unknown, map[string]json.RawMessage{"Records": rawArray(items)})
	if e != nil {
		return nil, e
	}
	if len(raw) > MaxStateBytes {
		return nil, ErrEndpointCapacity
	}
	return raw, nil
}
func validateState(s EndpointState) error {
	if s.Version != 1 || !canonicalPath(s.Namespace) || s.Records == nil || len(s.Records) > MaxRecords {
		return errors.New("invalid endpoint state header or bounds")
	}
	pairs := map[string]bool{}
	current := map[string]bool{}
	active := map[string]int{}
	identities := map[string]ProcessEvidence{}
	for _, x := range s.Records {
		b := x.Binding
		r := RegisterRequest{Version: 1, Namespace: b.Namespace, DocumentID: b.DocumentID, EditorIncarnation: b.EditorIncarnation, AttachID: b.AttachID, PID: b.PID, Socket: b.Socket, Buffer: b.Buffer, Mode: b.Mode}
		if e := ValidateEndpointRequest(r); e != nil {
			return e
		}
		if b.Namespace != s.Namespace || !validHex(b.BindingGeneration) || !canonicalPath(b.File) || b.ProcessIdentity == "" || len(b.ProcessIdentity) > 4096 {
			return errors.New("invalid endpoint binding")
		}
		if x.Status != "current" && x.Status != "superseded" && x.Status != "retired" {
			return errors.New("invalid endpoint status")
		}
		key := b.EditorIncarnation + ":" + b.DocumentID
		if pairs[key] {
			return errors.New("duplicate process/document pair")
		}
		pairs[key] = true
		identity := ProcessEvidence{EditorIncarnation: b.EditorIncarnation, PID: b.PID, ProcessIdentity: b.ProcessIdentity}
		if prev, ok := identities[b.EditorIncarnation]; ok && prev != identity {
			return errors.New("conflicting process identity")
		}
		identities[b.EditorIncarnation] = identity
		if x.Current != nil {
			cb := *x.Current
			oldID, oe := attach(cb.AttachID)
			highID, _ := attach(b.AttachID)
			if x.Status != "retired" || b.Mode != "headless" || oe != nil || oldID >= highID || cb.Mode != "desktop" || cb.Namespace != b.Namespace || cb.DocumentID != b.DocumentID || cb.EditorIncarnation != b.EditorIncarnation || cb.PID != b.PID || cb.ProcessIdentity != b.ProcessIdentity {
				return errors.New("invalid retained current binding")
			}
			if e := validateState(EndpointState{Version: 1, Namespace: s.Namespace, Records: []EndpointRecord{{Binding: cb, Status: "current"}}}); e != nil {
				return e
			}
		}
		if CurrentBinding(x) != nil {
			if current[b.DocumentID] {
				return errors.New("multiple current endpoints for document")
			}
			current[b.DocumentID] = true
			active[b.EditorIncarnation]++
			if active[b.EditorIncarnation] > MaxProcessActive {
				return errors.New("too many active endpoints for process")
			}
		}
	}
	return nil
}
