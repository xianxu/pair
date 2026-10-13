package couchcore

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/orientation"
)

type SwitchContentKind string

const (
	SwitchContentFresh    SwitchContentKind = "fresh"
	SwitchContentAuthored SwitchContentKind = "authored"
	SwitchContentFallback SwitchContentKind = "fallback"
)

type SwitchContent struct {
	Kind SwitchContentKind `json:"kind"`
	Body string            `json:"body,omitempty"`
}

// ValidateSwitchContent is shared by switch admission and its transports.
func ValidateSwitchContent(c SwitchContent) error {
	switch c.Kind {
	case SwitchContentFresh:
		if c.Body == "" {
			return nil
		}
		return errors.New("switch-agent: fresh content must have an empty body")
	case SwitchContentAuthored, SwitchContentFallback:
		return orientation.ValidateBody(c.Body)
	default:
		return errors.New("switch-agent: invalid content kind")
	}
}

type AgentSwitchPhase string

const (
	AgentSwitchWaitingBrief    AgentSwitchPhase = "waiting-brief"
	AgentSwitchWaitingSettled  AgentSwitchPhase = "waiting-settled"
	AgentSwitchSwitching       AgentSwitchPhase = "switching"
	AgentSwitchDeliveryPending AgentSwitchPhase = "delivery-pending"
	AgentSwitchSucceeded       AgentSwitchPhase = "succeeded"
	AgentSwitchRefused         AgentSwitchPhase = "refused"
	AgentSwitchFailure         AgentSwitchPhase = "failed"
	AgentSwitchUnconfirmed     AgentSwitchPhase = "unconfirmed"
)

// AgentSwitchSpec captures admission evidence. Source is the shell's opaque
// authenticated binding plus wrapper session token, never a mutable slot name.
// Zero Content requests an authored callback; Fallback must be explicit.
type AgentSwitchSpec struct {
	RequestID, Source, Agent                   string
	Content, Fallback                          SwitchContent
	AdmittedAt, BriefDeadline, OverallDeadline time.Time
}

// AgentSwitch is a value with no mutable references. Only AdvanceAgentSwitch
// changes its private state; copying it cannot alias selected content.
type AgentSwitch struct {
	requestID, source, agent                   string
	content, fallback                          SwitchContent
	admittedAt, briefDeadline, overallDeadline time.Time
	phase                                      AgentSwitchPhase
	reason                                     string
}

func NewAgentSwitch(spec AgentSwitchSpec) (AgentSwitch, error) {
	for _, v := range []struct {
		value string
		limit int
	}{{spec.RequestID, 1024}, {spec.Source, 16384}, {spec.Agent, 1024}} {
		if strings.TrimSpace(v.value) == "" || len(v.value) > v.limit || !utf8.ValidString(v.value) || strings.IndexFunc(v.value, unicode.IsControl) >= 0 {
			return AgentSwitch{}, errors.New("switch-agent: invalid request identity")
		}
	}
	if spec.AdmittedAt.IsZero() || !spec.BriefDeadline.After(spec.AdmittedAt) || !spec.OverallDeadline.After(spec.BriefDeadline) {
		return AgentSwitch{}, errors.New("switch-agent: invalid deadlines")
	}
	if spec.Fallback.Kind != SwitchContentFallback {
		return AgentSwitch{}, errors.New("switch-agent: fallback content required")
	}
	if err := ValidateSwitchContent(spec.Fallback); err != nil {
		return AgentSwitch{}, err
	}
	phase := AgentSwitchWaitingBrief
	if spec.Content != (SwitchContent{}) {
		if err := ValidateSwitchContent(spec.Content); err != nil {
			return AgentSwitch{}, err
		}
		phase = AgentSwitchWaitingSettled
	}
	return AgentSwitch{requestID: spec.RequestID, source: spec.Source, agent: spec.Agent, content: spec.Content, fallback: spec.Fallback, admittedAt: spec.AdmittedAt, briefDeadline: spec.BriefDeadline, overallDeadline: spec.OverallDeadline, phase: phase}, nil
}
func (s AgentSwitch) Phase() AgentSwitchPhase { return s.phase }
func (s AgentSwitch) Content() SwitchContent  { return s.content }
func (s AgentSwitch) RequestID() string       { return s.requestID }
func (s AgentSwitch) Source() string          { return s.source }
func (s AgentSwitch) Agent() string           { return s.agent }
func (s AgentSwitch) Reason() string          { return s.reason }
func (s AgentSwitch) Terminal() bool {
	switch s.phase {
	case AgentSwitchSucceeded, AgentSwitchRefused, AgentSwitchFailure, AgentSwitchUnconfirmed:
		return true
	}
	return false
}

type AgentSwitchEventKind uint8

const (
	AgentSwitchCallback AgentSwitchEventKind = iota + 1
	AgentSwitchObserve
	AgentSwitchTick
	AgentSwitchCancel
	AgentSwitchRegistered
	AgentSwitchFailed
	AgentSwitchDeliverySubmitted
	AgentSwitchDeliveryUnconfirmed
)

type AgentSwitchEvent struct {
	Kind                                          AgentSwitchEventKind
	At                                            time.Time
	RequestID, Source, Agent                      string
	Body                                          string
	Alive, SettledKnown, Settled, GitKnown, Clean bool
	Reason                                        string
}
type AgentSwitchEffect uint8

const (
	AgentSwitchNoEffect AgentSwitchEffect = iota
	AgentSwitchFallbackNotice
	AgentSwitchEnqueue
	AgentSwitchPublish
)

// AdvanceAgentSwitch returns intent only; the shell executes effects and sends
// correlated results. Enqueue transfers source revalidation to the transaction:
// the shell must recheck the binding and settled/clean state before parking.
func AdvanceAgentSwitch(s AgentSwitch, e AgentSwitchEvent) (AgentSwitch, AgentSwitchEffect, error) {
	refuseEvent := func(reason string) (AgentSwitch, AgentSwitchEffect, error) {
		return s, AgentSwitchNoEffect, errors.New("switch-agent: " + reason)
	}
	if s.phase == "" {
		return refuseEvent("uninitialized request")
	}
	if e.RequestID != s.requestID {
		return refuseEvent("request ID mismatch")
	}
	if s.Terminal() {
		return refuseEvent("request is terminal")
	}
	if e.Kind < AgentSwitchCallback || e.Kind > AgentSwitchDeliveryUnconfirmed {
		return refuseEvent("unknown event")
	}
	if e.At.IsZero() || e.At.Before(s.admittedAt) {
		return refuseEvent("event predates admission")
	}
	if e.Kind == AgentSwitchCallback && (e.Source != s.source || e.Agent != s.agent) {
		return refuseEvent("callback source or agent mismatch")
	}
	waiting := s.phase == AgentSwitchWaitingBrief || s.phase == AgentSwitchWaitingSettled
	if waiting && !e.At.Before(s.overallDeadline) {
		s.phase, s.reason = AgentSwitchRefused, "source did not settle before the switch deadline"
		return s, AgentSwitchPublish, nil
	}
	if e.Kind == AgentSwitchCallback && !e.At.Before(s.overallDeadline) {
		return refuseEvent("callback expired")
	}
	if e.Kind == AgentSwitchCancel {
		if waiting {
			s.phase = AgentSwitchRefused
		} else {
			s.phase = AgentSwitchUnconfirmed
		}
		s.reason = "switch cancelled"
		return s, AgentSwitchPublish, nil
	}
	if waiting && e.Kind == AgentSwitchObserve && (e.Source != s.source || !e.Alive || (e.GitKnown && !e.Clean) || (e.SettledKnown && e.Settled && !e.GitKnown)) {
		s.phase, s.reason = AgentSwitchRefused, "source changed, exited, or could not be verified clean"
		return s, AgentSwitchPublish, nil
	}
	if s.phase == AgentSwitchWaitingBrief && !e.At.Before(s.briefDeadline) {
		s.phase, s.content = AgentSwitchWaitingSettled, s.fallback
		if e.Kind == AgentSwitchCallback {
			return s, AgentSwitchFallbackNotice, errors.New("switch-agent: brief deadline expired")
		}
		return s, AgentSwitchFallbackNotice, nil
	}
	switch e.Kind {
	case AgentSwitchCallback:
		content := SwitchContent{Kind: SwitchContentAuthored, Body: e.Body}
		if err := ValidateSwitchContent(content); err != nil {
			return s, AgentSwitchNoEffect, err
		}
		if s.content == content {
			return s, AgentSwitchNoEffect, nil
		}
		if s.phase != AgentSwitchWaitingBrief {
			return refuseEvent("brief already selected")
		}
		s.content, s.phase = content, AgentSwitchWaitingSettled
	case AgentSwitchObserve:
		if s.phase == AgentSwitchWaitingSettled && e.SettledKnown && e.Settled && e.GitKnown && e.Clean {
			s.phase = AgentSwitchSwitching
			return s, AgentSwitchEnqueue, nil
		}
	case AgentSwitchRegistered:
		if s.phase == AgentSwitchSwitching {
			s.phase = AgentSwitchDeliveryPending
			if s.content.Kind == SwitchContentFresh {
				s.phase = AgentSwitchSucceeded
			}
			return s, AgentSwitchPublish, nil
		}
	case AgentSwitchFailed:
		if s.phase == AgentSwitchSwitching {
			s.phase, s.reason = AgentSwitchFailure, "switch transaction failed"
			return s, AgentSwitchPublish, nil
		}
	case AgentSwitchDeliverySubmitted, AgentSwitchDeliveryUnconfirmed:
		if s.phase == AgentSwitchDeliveryPending {
			s.phase = AgentSwitchSucceeded
			if e.Kind == AgentSwitchDeliveryUnconfirmed {
				s.phase, s.reason = AgentSwitchUnconfirmed, "handoff submission unconfirmed; inspect the target before sending manually"
			}
			return s, AgentSwitchPublish, nil
		}
	}
	return s, AgentSwitchNoEffect, nil
}
