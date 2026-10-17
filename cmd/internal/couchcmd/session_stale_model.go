package couchcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

const staleReminderEvery = 30 * time.Minute

type staleReminderObservation struct {
	document            sessionsync.DocumentInfo
	live                couchmessage.SlotLiveness
	hub, known, pending bool
}

func (o staleReminderObservation) eligible() bool {
	return o.document.ID != "" && o.document.SessionSync && o.document.Owner == o.live.Binding.Slot && o.document.Holder != sessionsync.HolderOperator && o.hub && o.known && !o.pending && o.live.Settled != nil && *o.live.Settled && o.live.Picker != nil && !*o.live.Picker
}

func (o staleReminderObservation) sameGeneration(other staleReminderObservation) bool {
	return o.document.ID == other.document.ID && o.document.File == other.document.File && o.document.Owner == other.document.Owner && o.live.Binding == other.live.Binding && o.live.Session == other.live.Session
}

type staleReminderState struct {
	next           map[string]time.Time
	active         *staleReminderObservation
	failed, closed bool
}

func (s *staleReminderState) due(id string, at time.Time) bool {
	return !s.closed && !at.Before(s.next[id])
}

type staleReminderEventKind uint8

const (
	staleRegistry staleReminderEventKind = iota
	staleTick
	staleChecked
	staleSent
	staleCancelled
	staleFault
)

type staleReminderEvent struct {
	kind        staleReminderEventKind
	at          time.Time
	ids         map[string]bool
	observation staleReminderObservation
	body        string
	err         error
}

type staleReminderEffect struct{ check, send, log bool }

// advanceStaleReminder is the sole owner of scheduling/diagnostic state. The
// worker supplies observations and results; this deterministic core performs no IO.
func advanceStaleReminder(s *staleReminderState, e staleReminderEvent) staleReminderEffect {
	if s.closed {
		return staleReminderEffect{}
	}
	switch e.kind {
	case staleRegistry:
		for id := range s.next {
			if !e.ids[id] {
				delete(s.next, id)
			}
		}
		if s.active != nil && !e.ids[s.active.document.ID] {
			s.active = nil
		}
	case staleTick:
		if s.active != nil || !e.observation.eligible() || !s.due(e.observation.document.ID, e.at) {
			break
		}
		observation := e.observation
		s.active = &observation
		return staleReminderEffect{check: true}
	case staleChecked:
		if s.active == nil {
			break
		}
		original := *s.active
		s.active = nil
		if e.err == nil && (!e.observation.eligible() || !original.sameGeneration(e.observation)) {
			break
		}
		if s.next == nil {
			s.next = map[string]time.Time{}
		}
		s.next[original.document.ID] = e.at.Add(staleReminderEvery)
		if e.err != nil {
			log := !s.failed
			s.failed = true
			return staleReminderEffect{log: log}
		}
		s.failed = false
		return staleReminderEffect{send: e.body != ""}
	case staleFault:
		log := !s.failed
		s.failed = true
		return staleReminderEffect{log: log}
	case staleSent:
		// Admission uncertainty/refusal never refunds the consumed check allowance.
	case staleCancelled:
		s.active = nil
		s.closed = true
	}
	return staleReminderEffect{}
}

// staleReminderBody recognizes tl-lead's framing without reimplementing its
// freshness budgets. Reject incomplete diagnostics instead of claiming freshness.
func staleReminderBody(file, output string) (string, error) {
	if len(file) > couchmessage.MaxBodyBytes || len(output) > couchmessage.MaxBodyBytes {
		return "", fmt.Errorf("tl-lead status output exceeds message limit")
	}
	if err := couchmessage.ValidateBody(output); err != nil {
		return "", err
	}
	output = strings.TrimSpace(output)
	if output == "fresh: every budgeted section" {
		return "", nil
	}
	if output == "" {
		return "", fmt.Errorf("tl-lead status check returned no freshness result")
	}
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "stale: ## ") {
			return "", fmt.Errorf("tl-lead status check returned an incomplete or unknown result")
		}
	}
	body := sessionsync.PingPrefix + file + "\n" + output
	if err := couchmessage.ValidateBody(body); err != nil {
		return "", err
	}
	return body, nil
}
