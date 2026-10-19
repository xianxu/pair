package couchcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type lifecycleFailureWire LifecycleFailure

// failureKnownFields is derived from owned tags so opaque members cannot shadow them.
func failureKnownFields(t reflect.Type) map[string]bool {
	m := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "-" || t.Field(i).PkgPath != "" {
			continue
		}
		name := tag
		for j, c := range name {
			if c == ',' {
				name = name[:j]
				break
			}
		}
		if name == "" {
			name = t.Field(i).Name
		}
		m[name] = true
	}
	return m
}
func failureObject(raw []byte, known map[string]bool, nullable map[string]bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil, fmt.Errorf("failure object cannot be null")
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, nil, err
	}
	if all == nil {
		return nil, nil, fmt.Errorf("failure object required")
	}
	owned := map[string]json.RawMessage{}
	unknown := map[string]json.RawMessage{}
	for k, v := range all {
		if known[k] {
			if !nullable[k] && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return nil, nil, fmt.Errorf("failure field %s cannot be null", k)
			}
			owned[k] = v
		} else {
			unknown[k] = append(json.RawMessage(nil), v...)
		}
	}
	return owned, unknown, nil
}
func failureEncodeObject(owned, unknown map[string]json.RawMessage, known map[string]bool) ([]byte, error) {
	merged := cloneFailureUnknown(owned)
	if merged == nil {
		merged = map[string]json.RawMessage{}
	}
	for k, v := range unknown {
		if known[k] {
			continue
		}
		if err := strictjson.RejectDuplicateKeys(v); err != nil {
			return nil, err
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
func decodeLifecycleFailureHistory(raw []byte) (LifecycleFailureHistory, error) {
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("failure history must be an array")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	if len(entries) > launchFailureHistoryLimit {
		return nil, fmt.Errorf("launch diagnostics exceeds attempt limit")
	}
	var h LifecycleFailureHistory
	for _, entry := range entries {
		known := failureKnownFields(reflect.TypeOf(lifecycleFailureWire{}))
		owned, unknown, err := failureObject(entry, known, map[string]bool{"before_exit": true, "after_exit": true})
		if err != nil {
			return nil, err
		}
		for _, k := range []string{"nonce", "failed_at"} {
			if _, ok := owned[k]; !ok {
				return nil, fmt.Errorf("missing failure field %s", k)
			}
		}
		var addressUnknown map[string]json.RawMessage
		if address, ok := owned["address"]; ok {
			a, u, e := failureObject(address, map[string]bool{"repo_scope": true, "tag": true}, nil)
			if e != nil {
				return nil, e
			}
			addressUnknown = u
			for _, k := range []string{"repo_scope", "tag"} {
				if _, ok := a[k]; !ok {
					return nil, fmt.Errorf("missing address field %s", k)
				}
			}
			owned["address"], _ = json.Marshal(a)
		}
		var holders []ProcessIdentityLine
		if holderRaw, ok := owned["holders"]; ok {
			var hs []json.RawMessage
			if err := json.Unmarshal(holderRaw, &hs); err != nil {
				return nil, err
			}
			for _, r := range hs {
				m, u, e := failureObject(r, map[string]bool{"process": true, "kind": true}, nil)
				if e != nil {
					return nil, e
				}
				p, ok := m["process"]
				if !ok {
					return nil, fmt.Errorf("missing holder process")
				}
				pm, pu, e := failureObject(p, map[string]bool{"PID": true, "Identity": true}, nil)
				if e != nil {
					return nil, e
				}
				for _, k := range []string{"PID", "Identity"} {
					if _, ok := pm[k]; !ok {
						return nil, fmt.Errorf("missing process identity %s", k)
					}
				}
				m["process"], _ = json.Marshal(pm)
				clean, _ := json.Marshal(m)
				var line ProcessIdentityLine
				if err := strictjson.Decode(clean, &line); err != nil {
					return nil, err
				}
				line.Unknown = u
				line.processUnknown = pu
				holders = append(holders, line)
			}
			delete(owned, "holders")
		}
		clean, _ := json.Marshal(owned)
		var f lifecycleFailureWire
		if err := strictjson.Decode(clean, &f); err != nil {
			return nil, err
		}
		failure := LifecycleFailure(f)
		failure.Unknown = unknown
		failure.addressUnknown = addressUnknown
		failure.Holders = holders
		normalized, err := normalizeLaunchFailure(failure)
		if err != nil {
			return nil, err
		}
		for _, old := range h {
			if old.Address == normalized.Address && old.Nonce == normalized.Nonce && lifecycleFailureTargetKey(failureTarget(old)) == lifecycleFailureTargetKey(failureTarget(normalized)) {
				return nil, fmt.Errorf("duplicate launch diagnostic identity")
			}
		}
		h = append(h, normalized)
	}
	sort.SliceStable(h, func(i, j int) bool { return compareLifecycleFailureRecency(h[i], h[j]) > 0 })
	return h, nil
}

// encodeLifecycleFailureRow is shared by size selection and final publication.
func encodeLifecycleFailureRow(f LifecycleFailure) ([]byte, error) {
	normalized, err := normalizeLaunchFailure(f)
	if err != nil {
		return nil, err
	}
	f = normalized
	base, err := json.Marshal(lifecycleFailureWire(f))
	if err != nil {
		return nil, err
	}
	var owned map[string]json.RawMessage
	_ = json.Unmarshal(base, &owned)
	if f.Address == (ThreadAddress{}) {
		delete(owned, "address")
	} else {
		var a map[string]json.RawMessage
		_ = json.Unmarshal(owned["address"], &a)
		owned["address"], err = failureEncodeObject(a, f.addressUnknown, map[string]bool{"repo_scope": true, "tag": true})
		if err != nil {
			return nil, err
		}
	}
	if len(f.Holders) > 0 {
		var hb bytes.Buffer
		hb.WriteByte('[')
		for j, line := range f.Holders {
			if j > 0 {
				hb.WriteByte(',')
			}
			p, _ := json.Marshal(line.Process)
			var pm map[string]json.RawMessage
			_ = json.Unmarshal(p, &pm)
			p, err = failureEncodeObject(pm, line.processUnknown, map[string]bool{"PID": true, "Identity": true})
			if err != nil {
				return nil, err
			}
			kind, _ := json.Marshal(line.Kind)
			r, err := failureEncodeObject(map[string]json.RawMessage{"process": p, "kind": kind}, line.Unknown, map[string]bool{"process": true, "kind": true})
			if err != nil {
				return nil, err
			}
			hb.Write(r)
		}
		hb.WriteByte(']')
		owned["holders"] = hb.Bytes()
	}
	return failureEncodeObject(owned, f.Unknown, failureKnownFields(reflect.TypeOf(lifecycleFailureWire{})))
}
func encodeLifecycleFailureHistory(h LifecycleFailureHistory) ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, f := range h {
		raw, err := encodeLifecycleFailureRow(f)
		if err != nil {
			return nil, err
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(raw)
	}
	b.WriteByte(']')
	if b.Len() > launchFailureEncodedLimit {
		return nil, fmt.Errorf("launch failure history exceeds %d bytes", launchFailureEncodedLimit)
	}
	return b.Bytes(), nil
}

// fitLifecycleFailureHistory retains a whole-entry prefix within its byte budget.
func fitLifecycleFailureHistory(h LifecycleFailureHistory, limit int, encodeRow func(LifecycleFailure) ([]byte, error)) (LifecycleFailureHistory, error) {
	lengths := make([]int, len(h))
	size := 2
	for i, f := range h {
		raw, err := encodeRow(f)
		if err != nil {
			return nil, err
		}
		lengths[i] = len(raw)
		size += len(raw)
		if i > 0 {
			size++
		}
	}
	retained := len(h)
	for size > limit && retained > 1 {
		retained--
		size -= lengths[retained] + 1
	}
	if size > limit {
		return nil, fmt.Errorf("launch failure history exceeds %d bytes", limit)
	}
	return h[:retained], nil
}
