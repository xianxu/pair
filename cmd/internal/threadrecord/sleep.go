package threadrecord

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// SleepCause is why a park put its thread to sleep (pair#830).
type SleepCause string

// SleepFullQuit is a confirmed full quit (Alt+x, Leave with LeavePark).
const SleepFullQuit SleepCause = "full_quit"

// SleepMark records that the park named by Nonce was a full shutdown's, not
// the operator's choice for this one thread. It is written in the same CAS
// that begins that park and means "asleep" only while the record's verified
// park carries the same nonce (couchcore.ThreadAsleep): every path that ends
// or replaces the park leaves a stale mark inert, even one an older build
// carried through a rewrite. Born additive, like TranscriptBinding.
type SleepMark struct {
	Nonce string     `json:"nonce"`
	At    time.Time  `json:"at"`
	Cause SleepCause `json:"cause"`
	// Unknown carries members a newer build added inside the mark (pair#695 §2a).
	Unknown map[string]json.RawMessage `json:"-"`
}

type sleepWire SleepMark

var sleepKnownFields = strictjson.KnownFields(reflect.TypeOf(sleepWire{}))

// UnmarshalJSON decodes the owned members strictly and keeps the rest.
func (m *SleepMark) UnmarshalJSON(raw []byte) error {
	var wire sleepWire
	unknown, err := strictjson.DecodeKnown(raw, sleepKnownFields, &wire)
	if err != nil {
		return err
	}
	*m = SleepMark(wire)
	if len(unknown) > 0 {
		m.Unknown = unknown
	}
	return nil
}

// MarshalJSON writes the owned members, then the preserved ones verbatim.
func (m SleepMark) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(sleepWire(m))
	if err != nil {
		return nil, err
	}
	return strictjson.AppendMembers(raw, m.Unknown, sleepKnownFields)
}

// Validate is the mark's structural acceptance rule.
func (m SleepMark) Validate() error {
	if m.Nonce == "" {
		return errors.New("sleep mark has no park nonce")
	}
	if m.At.IsZero() {
		return errors.New("sleep mark has no time")
	}
	// Any non-empty cause is accepted: a newer build's cause (a drain, #722)
	// must not make this build refuse the record. Readers only show it.
	if m.Cause == "" || len(m.Cause) > 64 {
		return fmt.Errorf("invalid sleep cause %q", m.Cause)
	}
	return nil
}

// CloneSleepMark copies a mark pointer.
func CloneSleepMark(m *SleepMark) *SleepMark {
	if m == nil {
		return nil
	}
	out := *m
	out.Unknown = strictjson.CloneRaw(m.Unknown)
	return &out
}
