package couchcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

type drainControlRecord struct {
	Member                          drainLiveMember
	WrapperInstance                 string
	NoticeID, InterruptID           string
	NoticeStarted, InterruptStarted bool
	InterruptObserved               *couchmessage.DrainObservation
}
type drainServiceRecord struct {
	SchemaVersion int                           `json:"schema_version"`
	State         couchcore.DrainState          `json:"state"`
	From          couchmessage.Binding          `json:"from"`
	Released      bool                          `json:"released"`
	PauseStarted  bool                          `json:"pause_started"`
	Controls      map[string]drainControlRecord `json:"controls"`
}

func decodeDrainServiceRecord(raw json.RawMessage) (drainServiceRecord, error) {
	var r drainServiceRecord
	if _, err := drainJSONObject(raw); err != nil {
		return r, err
	}
	if err := validateDrainRecordShape(raw, reflect.TypeOf(r)); err != nil {
		return r, err
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	if r.SchemaVersion != 1 {
		return r, errors.New("unsupported drain record version")
	}
	if r.State.ID == "" || r.State.Owner != "couch-drain:"+r.State.ID || len(r.State.Members) > couchcore.MaxDrainMembers || len(r.Controls) > couchcore.MaxDrainMembers {
		return r, errors.New("invalid persisted drain authority")
	}
	if _, err := couchcore.DrainNotice(r.State.ID, r.State.Reason, r.State.Fast); err != nil {
		return r, err
	}
	switch r.State.Phase {
	case couchcore.DrainPreparing, couchcore.DrainActive, couchcore.DrainCancelling, couchcore.DrainFinished, couchcore.DrainRecovered:
	default:
		return r, errors.New("invalid persisted drain phase")
	}
	if r.Released && r.State.Phase != couchcore.DrainFinished {
		return r, errors.New("released drain has active phase")
	}
	if r.State.Deadline.IsZero() || r.Controls == nil || r.State.Members == nil || len(r.Controls) != len(r.State.Members) {
		return r, errors.New("incomplete drain record")
	}
	if err := r.From.Validate(); err != nil {
		return r, err
	}
	if r.Released {
		if !r.State.ResumeIssued || !r.State.Queue.Known || !r.State.Queue.Complete {
			return r, errors.New("released drain lacks owner-release evidence")
		}
		for _, owner := range r.State.Queue.Owners {
			if owner == r.State.Owner {
				return r, errors.New("released drain still owns queue pause")
			}
		}
		for _, member := range r.State.Members {
			if member.Revoke != couchcore.DrainDone {
				return r, errors.New("released drain has unconfirmed control revocation")
			}
		}
	}
	for workspace, m := range r.State.Members {
		control, ok := r.Controls[workspace]
		if !ok || m.Workspace != workspace || m.Identity != control.Member.Identity() || control.Member.Binding.Slot != workspace || control.NoticeID != drainEffectID(r.State.ID, m.Identity, "notice") || control.InterruptID != drainEffectID(r.State.ID, m.Identity, "interrupt") {
			return r, errors.New("persisted member/control identity mismatch")
		}
		if err := control.Member.Binding.Validate(); err != nil {
			return r, err
		}
		if control.Member.Session == "" {
			return r, errors.New("persisted admitted session missing")
		}
		for _, outcome := range []couchcore.DrainOutcome{m.Notice, m.Interrupt, m.Revoke} {
			switch outcome {
			case "", couchcore.DrainPending, couchcore.DrainDone, couchcore.DrainUnknown, couchcore.DrainRefused:
			default:
				return r, errors.New("invalid persisted control outcome")
			}
		}
	}
	return r, nil
}

// Known fields are mandatory and strictly typed, including recursively inside
// member/control maps. Unknown additive fields are left uninterpreted.
func validateDrainRecordShape(raw json.RawMessage, t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil
		}
		return validateDrainRecordShape(raw, t.Elem())
	}
	if t == reflect.TypeOf(time.Time{}) {
		var v time.Time
		return json.Unmarshal(raw, &v)
	}
	switch t.Kind() {
	case reflect.Struct:
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil || m == nil {
			return errors.New("drain record expected object")
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			for key := range m {
				if key != name && strings.EqualFold(key, name) {
					return fmt.Errorf("case alias shadows known drain field %s", name)
				}
			}
			v, ok := m[name]
			if !ok {
				return fmt.Errorf("missing drain record field %s", name)
			}
			if err := validateDrainRecordShape(v, f.Type); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	case reflect.Map:
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("null drain map")
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return err
		}
		for _, v := range m {
			if err := validateDrainRecordShape(v, t.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil
		}
		var a []json.RawMessage
		if err := json.Unmarshal(raw, &a); err != nil {
			return err
		}
		for _, v := range a {
			if err := validateDrainRecordShape(v, t.Elem()); err != nil {
				return err
			}
		}
	default:
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("null known scalar")
		}
		v := reflect.New(t)
		return json.Unmarshal(raw, v.Interface())
	}
	return nil
}

// Rewrites preserve unknown fields at every object depth. Maps of members are
// intentionally bounded by current keys; retired member keys are not a ledger.
func encodeDrainServiceRecord(r drainServiceRecord, old json.RawMessage) (json.RawMessage, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if len(old) == 0 {
		return raw, nil
	}
	return mergeDrainRecordUnknown(raw, old, reflect.TypeOf(r))
}
func mergeDrainRecordUnknown(fresh, old json.RawMessage, t reflect.Type) (json.RawMessage, error) {
	if t.Kind() == reflect.Pointer {
		if bytes.Equal(fresh, []byte("null")) {
			return fresh, nil
		}
		return mergeDrainRecordUnknown(fresh, old, t.Elem())
	}
	if t.Kind() == reflect.Slice {
		// Activity order is observational. Only its stable operation authority
		// can carry an extension across reordering, completion or replacement.
		if t.Elem() == reflect.TypeOf(couchcore.DrainActivity{}) {
			var current, previous []json.RawMessage
			if json.Unmarshal(fresh, &current) != nil || json.Unmarshal(old, &previous) != nil {
				return fresh, nil
			}
			prior := make(map[[3]string]json.RawMessage, len(previous))
			for _, raw := range previous {
				key, err := drainRecordActivityKey(raw)
				if err != nil {
					return nil, err
				}
				if _, duplicate := prior[key]; duplicate {
					return nil, errors.New("ambiguous prior drain activity identity")
				}
				prior[key] = raw
			}
			seen := make(map[[3]string]bool, len(current))
			for i, raw := range current {
				key, err := drainRecordActivityKey(raw)
				if err != nil {
					return nil, err
				}
				if seen[key] {
					return nil, errors.New("ambiguous current drain activity identity")
				}
				seen[key] = true
				if previous, found := prior[key]; found {
					merged, err := mergeDrainRecordUnknown(raw, previous, t.Elem())
					if err != nil {
						return nil, err
					}
					current[i] = merged
				}
			}
			return json.Marshal(current)
		}
		// Scalar arrays have no extension fields. For a future unkeyed object
		// array, retain opaque fields only when all known values are unchanged;
		// an index alone is never evidence that an element is the same entity.
		if t.Elem().Kind() != reflect.Struct && t.Elem().Kind() != reflect.Pointer {
			return fresh, nil
		}
		known := reflect.New(t)
		if json.Unmarshal(old, known.Interface()) == nil {
			projected, err := json.Marshal(known.Elem().Interface())
			if err == nil && bytes.Equal(projected, fresh) {
				return append(json.RawMessage(nil), old...), nil
			}
		}
		return fresh, nil
	}
	if t.Kind() != reflect.Struct && t.Kind() != reflect.Map {
		return fresh, nil
	}
	if t == reflect.TypeOf(time.Time{}) {
		return fresh, nil
	}
	var current, previous map[string]json.RawMessage
	if json.Unmarshal(fresh, &current) != nil || current == nil {
		return fresh, nil
	}
	_ = json.Unmarshal(old, &previous)
	if t.Kind() == reflect.Map {
		for k, v := range current {
			if prior, ok := previous[k]; ok {
				merged, err := mergeDrainRecordUnknown(v, prior, t.Elem())
				if err != nil {
					return nil, err
				}
				current[k] = merged
			}
		}
	} else {
		known := map[string]reflect.Type{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "" {
				name = f.Name
			}
			known[name] = f.Type
		}
		for k, v := range previous {
			ft, ok := known[k]
			if !ok {
				current[k] = v
				continue
			}
			if newer, exists := current[k]; exists {
				merged, err := mergeDrainRecordUnknown(newer, v, ft)
				if err != nil {
					return nil, err
				}
				current[k] = merged
			}
		}
	}
	return json.Marshal(current)
}

func drainRecordActivityKey(raw json.RawMessage) ([3]string, error) {
	var activity struct{ ID, Kind, Workspace string }
	if err := json.Unmarshal(raw, &activity); err != nil {
		return [3]string{}, err
	}
	if activity.ID == "" || activity.Kind == "" {
		return [3]string{}, errors.New("drain activity has no stable identity")
	}
	return [3]string{activity.ID, activity.Kind, activity.Workspace}, nil
}
