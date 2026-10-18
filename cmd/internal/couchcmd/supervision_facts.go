package couchcmd

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchtty"
)

const maxFactBytes = 32 << 10

// supervisionFact contains only typed, bounded observations, never source prose.
type supervisionFact struct {
	Version     int                       `json:"version"`
	Generation  string                    `json:"generation"`
	Seq         uint64                    `json:"seq"`
	At          time.Time                 `json:"at"`
	ElapsedMS   int64                     `json:"elapsed_ms"`
	Dropped     uint64                    `json:"dropped"`
	Event       string                    `json:"event"`
	Subject     string                    `json:"subject,omitempty"`
	Incarnation string                    `json:"incarnation,omitempty"`
	Sender      string                    `json:"sender,omitempty"`
	Message     string                    `json:"message,omitempty"`
	Hint        string                    `json:"hint,omitempty"`
	Outcome     string                    `json:"outcome,omitempty"`
	Generated   *bool                     `json:"generated,omitempty"`
	Submission  uint64                    `json:"submission,omitempty"`
	Connected   *bool                     `json:"connected,omitempty"`
	Settled     *bool                     `json:"settled"`
	Picker      *bool                     `json:"picker"`
	Clipboard   *couchtty.CopyObservation `json:"clipboard,omitempty"`
	Slots       []supervisionFactSlot     `json:"slots,omitempty"`

	ClipboardKey *couchtty.CopyKeyObservation `json:"clipboard_key,omitempty"`
}
type supervisionFactSlot struct {
	Subject     string  `json:"subject"`
	Incarnation string  `json:"incarnation"`
	Known       bool    `json:"known"`
	Resting     *bool   `json:"resting"`
	Settled     *bool   `json:"settled"`
	Picker      *bool   `json:"picker"`
	Pending     *bool   `json:"pending"`
	QuietMS     *int64  `json:"quiet_ms"`
	Submission  *uint64 `json:"submission"`
}

func factKindHint(body string) string {
	if completionHint(body) {
		return "landed"
	}
	head, _, _ := strings.Cut(body, "\n")
	head = strings.ToLower(strings.TrimSpace(head[:min(len(head), 256)]))
	if strings.HasPrefix(head, "waiting for operator") || strings.HasPrefix(head, "waiting-for-operator") {
		return "waiting-for-operator"
	}
	if strings.HasPrefix(head, "progress:") {
		return "progress"
	}
	return "unknown"
}
func encodeFact(f supervisionFact) ([]byte, error) {
	// A non-nil empty snapshot must survive omitempty as an explicit empty list.
	var value any = f
	if f.Event == "snapshot" {
		slots := f.Slots
		if slots == nil {
			slots = []supervisionFactSlot{}
		}
		value = struct {
			supervisionFact
			Slots []supervisionFactSlot `json:"slots"`
		}{f, slots}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(raw)+1 > maxFactBytes {
		return nil, errors.New("supervision fact exceeds row bound")
	}
	return append(raw, '\n'), nil
}

type factLogLifecycle uint8

const (
	factLogOpening factLogLifecycle = iota
	factLogActive
	factLogUnavailable
	factLogClosed
)

type factLogEvent uint8

const (
	factLogOpened factLogEvent = iota
	factLogOpenFailed
	factLogWriteFailed
	factLogClose
)

func (s factLogLifecycle) advance(e factLogEvent) factLogLifecycle {
	if s == factLogClosed || e == factLogClose {
		return factLogClosed
	}
	if s == factLogOpening {
		if e == factLogOpened {
			return factLogActive
		}
		if e == factLogOpenFailed {
			return factLogUnavailable
		}
	}
	return s
}
