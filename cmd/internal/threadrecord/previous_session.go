package threadrecord

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

// PreviousSession retains the terminal recording from one outgoing incarnation.
// It is replaced by the next source park, never by target registration or retry.
type PreviousSession struct {
	Source ParkIdentity `json:"source"`
	// Deprecated context fields are decode-only compatibility. MarshalJSON validates
	// them before omitting them; new descriptors carry only tty coordinates.
	Binding       *couchidentity.SessionBinding      `json:"binding,omitempty"`
	Agent         string                             `json:"agent"`
	NativeSession string                             `json:"native_session,omitempty"`
	Transcripts   []string                           `json:"transcripts,omitempty"`
	Capture       *pairlifecycle.PreservedScrollback `json:"capture,omitempty"`
	CapturedAt    time.Time                          `json:"captured_at"`
	Unavailable   []string                           `json:"unavailable,omitempty"`
}

func (p PreviousSession) Validate(address Address) error {
	if err := validateParkIdentity(p.Source, address); err != nil {
		return err
	}
	if err := validateSessionBinding(p.Binding, address); err != nil {
		return err
	}
	if !componentPattern.MatchString(p.Agent) || len(p.Agent) > 64 || p.CapturedAt.IsZero() {
		return fmt.Errorf("previous session needs an agent and capture time")
	}
	if err := pairlifecycle.ValidatePreservedScrollback(p.Capture); err != nil {
		return err
	}
	if p.Capture != nil && (p.Capture.Agent != p.Agent || len(p.Capture.Token) > 256) {
		return fmt.Errorf("previous capture does not match source agent or token budget")
	}
	textOK := func(s string, limit int) bool {
		return len(s) <= limit && utf8.ValidString(s) && strings.IndexFunc(s, unicode.IsControl) < 0
	}
	if !textOK(p.NativeSession, 1024) || len(p.Transcripts) > 8 || len(p.Unavailable) > 16 {
		return fmt.Errorf("previous session references exceed limits")
	}
	for _, path := range p.Transcripts {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || !textOK(path, 4096) {
			return fmt.Errorf("previous transcript must be a bounded absolute path")
		}
	}
	for _, reason := range p.Unavailable {
		if !textOK(reason, 1024) {
			return fmt.Errorf("invalid previous-session unavailability reason")
		}
	}
	type wire PreviousSession
	raw, err := json.Marshal(wire(p))
	if err != nil || len(raw) > 32*1024 {
		return fmt.Errorf("previous session exceeds 32 KiB")
	}
	return nil
}

func ClonePreviousSession(p *PreviousSession) *PreviousSession {
	if p == nil {
		return nil
	}
	out := *p
	if p.Binding != nil {
		b := *p.Binding
		out.Binding = &b
	}
	out.Capture = pairlifecycle.ClonePreservedScrollback(p.Capture)
	out.Transcripts = append([]string(nil), p.Transcripts...)
	out.Unavailable = append([]string(nil), p.Unavailable...)
	return &out
}

// MarshalJSON retires old context without letting malformed stored values evade
// validation. Keep normal decoding so strict persisted decoding still rejects
// unknown or duplicate fields.
func (p PreviousSession) MarshalJSON() ([]byte, error) {
	if err := p.Validate(p.Source.Address); err != nil {
		return nil, err
	}
	type wire PreviousSession
	p.Binding, p.NativeSession, p.Transcripts, p.Unavailable = nil, "", nil, nil
	return json.Marshal(wire(p))
}
