// Package operationalrun owns immutable Couch operational run bundles and their retention.
package operationalrun

import (
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const RecordLimit = 64 << 10

var component = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,127}$`)

type RunKey struct {
	Hub     string                     `json:"hub"`
	Task    string                     `json:"task"`
	ID      string                     `json:"id"`
	Unknown map[string]json.RawMessage `json:"-"`
}

func (k RunKey) valid() bool {
	return component.MatchString(k.Hub) && component.MatchString(k.Task) && component.MatchString(k.ID)
}
func sameKey(a, b RunKey) bool { return a.Hub == b.Hub && a.Task == b.Task && a.ID == b.ID }

type Record struct {
	Version             int                        `json:"version"`
	Key                 RunKey                     `json:"key"`
	DeclarationRevision string                     `json:"declaration_revision"`
	Source              string                     `json:"source"`
	ScheduledAt         time.Time                  `json:"scheduled_at"`
	StartedAt           time.Time                  `json:"started_at"`
	EndedAt             time.Time                  `json:"ended_at"`
	Exit                *int                       `json:"exit"`
	Outcome             string                     `json:"outcome"`
	Notice              string                     `json:"notice"`
	Unknown             map[string]json.RawMessage `json:"-"`
}
type keyWire RunKey
type recordWire Record

var keyKnown = strictjson.KnownFields(reflect.TypeFor[keyWire]())
var recordKnown = strictjson.KnownFields(reflect.TypeFor[recordWire]())

func (k *RunKey) UnmarshalJSON(raw []byte) error {
	var w keyWire
	u, e := decodeOwned(raw, keyKnown, nil, &w)
	if e != nil {
		return e
	}
	*k = RunKey(w)
	k.Unknown = u
	return nil
}
func (r *Record) UnmarshalJSON(raw []byte) error {
	var w recordWire
	u, e := decodeOwned(raw, recordKnown, map[string]bool{"exit": true}, &w)
	if e != nil {
		return e
	}
	*r = Record(w)
	r.Unknown = u
	return r.validate()
}
func (r Record) validate() error {
	if r.Version != 1 || !r.Key.valid() || r.Outcome == "" || len(r.DeclarationRevision) > 4096 || len(r.Source) > 4096 || len(r.Notice) > 4096 {
		return errors.New("invalid operational run record")
	}
	return nil
}
func encodeRecord(r Record) ([]byte, error) {
	if e := r.validate(); e != nil {
		return nil, e
	}
	raw, e := json.Marshal(recordWire(r))
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		return nil, e
	}
	kr, e := json.Marshal(keyWire(r.Key))
	if e != nil {
		return nil, e
	}
	var keys map[string]json.RawMessage
	if e = json.Unmarshal(kr, &keys); e != nil {
		return nil, e
	}
	fields["key"], e = strictjson.EncodeObject(keys, r.Key.Unknown, keyKnown)
	if e != nil {
		return nil, e
	}
	raw, e = strictjson.EncodeObject(fields, r.Unknown, recordKnown)
	if len(raw) > RecordLimit {
		return nil, errors.New("operational record exceeds byte cap")
	}
	return raw, e
}
func terminal(r Record) bool {
	if r.EndedAt.IsZero() {
		return false
	}
	switch r.Outcome {
	case "quiet", "wake", "failed", "timeout", "cancelled", "skipped", "refused", "completed":
		return true
	}
	return false
}

// Completed identifies a confirmed terminal result. It is shared by retention
// and interrupted authority publication; uncertainty never authorizes either.
func (r Record) Completed() bool              { return terminal(r) }
func (r Record) MarshalJSON() ([]byte, error) { return encodeRecord(r) }
func (k RunKey) MarshalJSON() ([]byte, error) {
	raw, e := json.Marshal(keyWire(k))
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		return nil, e
	}
	return strictjson.EncodeObject(fields, k.Unknown, keyKnown)
}

// Every field in a published envelope is present, including zero timestamps and
// a nullable exit status. Unknown members carry no interpretation authority.
func decodeOwned(raw []byte, known, nullable map[string]bool, target any) (map[string]json.RawMessage, error) {
	owned, unknown, e := strictjson.SplitObject(raw, known, nullable)
	if e != nil {
		return nil, e
	}
	for name := range known {
		if _, ok := owned[name]; !ok {
			return nil, errors.New("missing operational member: " + name)
		}
	}
	encoded, e := strictjson.EncodeObject(owned, nil, known)
	if e != nil {
		return nil, e
	}
	if e = strictjson.Decode(encoded, target); e != nil {
		return nil, e
	}
	return unknown, nil
}
