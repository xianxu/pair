package couchcore

import (
	"errors"
	"time"
)

const MaxDrainMembers = 128

type DrainPhase string

const (
	DrainPreparing  DrainPhase = "preparing"
	DrainActive     DrainPhase = "active"
	DrainCancelling DrainPhase = "cancelling"
	DrainFinished   DrainPhase = "cancelled"
	DrainRecovered  DrainPhase = "recovered"
)

type DrainVisibleState string

const (
	DrainDraining  DrainVisibleState = "draining"
	DrainStuck     DrainVisibleState = "stuck"
	DrainDrained   DrainVisibleState = "drained"
	DrainCancelled DrainVisibleState = "cancelled"
)

type DrainOutcome string

const (
	DrainPending DrainOutcome = "pending"
	DrainDone    DrainOutcome = "done"
	DrainUnknown DrainOutcome = "unknown"
	DrainRefused DrainOutcome = "refused"
)

type DrainEffectKind string

const (
	DrainPause      DrainEffectKind = "pause"
	DrainSendNotice DrainEffectKind = "notice"
	DrainInterrupt  DrainEffectKind = "interrupt"
	DrainRevoke     DrainEffectKind = "revoke"
	DrainResume     DrainEffectKind = "resume"
)

type DrainEventKind string

const (
	DrainQueueObserved   DrainEventKind = "queue"
	DrainMemberObserved  DrainEventKind = "member"
	DrainMemberJoined    DrainEventKind = "joined"
	DrainAcknowledged    DrainEventKind = "ack"
	DrainNoticeResult    DrainEventKind = "notice_result"
	DrainInterruptResult DrainEventKind = "interrupt_result"
	DrainRevoked         DrainEventKind = "revoked"
	DrainCancel          DrainEventKind = "cancel"
	DrainRecover         DrainEventKind = "recover"
	DrainFailed          DrainEventKind = "failed"
)

type DrainStart struct {
	ID, Reason string
	Fast       bool
	Deadline   time.Time
	Members    []DrainMember
}
type DrainState struct {
	ID, Owner, Reason string
	Fast              bool
	Deadline          time.Time
	Phase             DrainPhase
	Members           map[string]DrainMember
	Queue             DrainQueueObservation
	Error             string
	ResumeIssued      bool
}
type DrainMember struct {
	Workspace, Identity       string
	Observation               DrainObservation
	Ack, AckSealed            bool
	AckSubmission             uint64
	Notice, Interrupt, Revoke DrainOutcome
	Error                     string
}
type DrainObservation struct {
	Known, Connected, Idle, Picker, Background bool
	Submission, Sequence                       uint64
}
type DrainActivity struct{ ID, Workspace, Kind, State, Head string }
type DrainQueueObservation struct {
	Known, Complete, Paused bool
	Owners                  []string
	Active                  []DrainActivity
	Detail                  string
	Sequence                uint64
}
type DrainEvent struct {
	Kind                    DrainEventKind
	ID, Workspace, Identity string
	Queue                   DrainQueueObservation
	Observation             DrainObservation
	Outcome                 DrainOutcome
	Detail                  string
}
type DrainEffect struct {
	Kind                                   DrainEffectKind
	ID, Owner, Workspace, Identity, Reason string
	Fast                                   bool
}

// StartDrain constructs persisted intent. The shell must persist it and close
// admission before executing the returned owner-scoped pause effect.
func StartDrain(in DrainStart) (DrainState, []DrainEffect, error) {
	if _, err := DrainNotice(in.ID, in.Reason, in.Fast); err != nil {
		return DrainState{}, nil, err
	}
	if in.Deadline.IsZero() {
		return DrainState{}, nil, errors.New("drain deadline required")
	}
	if len(in.Members) > MaxDrainMembers {
		return DrainState{}, nil, errors.New("drain membership exceeds limit")
	}
	s := DrainState{ID: in.ID, Owner: "couch-drain:" + in.ID, Reason: in.Reason, Fast: in.Fast, Deadline: in.Deadline, Phase: DrainPreparing, Members: make(map[string]DrainMember)}
	for _, m := range in.Members {
		if m.Workspace == "" || m.Identity == "" {
			return DrainState{}, nil, errors.New("drain member identity required")
		}
		if _, ok := s.Members[m.Workspace]; ok {
			return DrainState{}, nil, errors.New("duplicate drain workspace")
		}
		s.Members[m.Workspace] = DrainMember{Workspace: m.Workspace, Identity: m.Identity}
	}
	return s, []DrainEffect{s.effect(DrainPause, DrainMember{})}, nil
}
func (s DrainState) effect(k DrainEffectKind, m DrainMember) DrainEffect {
	return DrainEffect{Kind: k, ID: s.ID, Owner: s.Owner, Workspace: m.Workspace, Identity: m.Identity, Reason: s.Reason, Fast: s.Fast}
}
func cloneDrain(s DrainState) DrainState {
	members := make(map[string]DrainMember, len(s.Members))
	for k, m := range s.Members {
		members[k] = m
	}
	s.Members = members
	s.Queue.Owners = append([]string(nil), s.Queue.Owners...)
	s.Queue.Active = append([]DrainActivity(nil), s.Queue.Active...)
	return s
}
func drainOwnPause(s DrainState) bool {
	if !s.Queue.Known || !s.Queue.Complete || !s.Queue.Paused {
		return false
	}
	for _, o := range s.Queue.Owners {
		if o == s.Owner {
			return true
		}
	}
	return false
}
func drainControlTerminal(o DrainOutcome) bool { return o == DrainDone || o == DrainRefused }

// ReduceDrain is pure: effects carry the exact generation/incarnation authority.
// Zero observation sequences are for already serialized shell observations; a
// producer capable of reordering must provide strictly increasing sequences.
func ReduceDrain(before DrainState, e DrainEvent) (DrainState, []DrainEffect) {
	if e.ID != before.ID || before.ID == "" || before.Phase == DrainFinished {
		return before, nil
	}
	s := cloneDrain(before)
	var fx []DrainEffect
	switch e.Kind {
	case DrainQueueObserved:
		if e.Queue.Sequence != 0 && e.Queue.Sequence <= s.Queue.Sequence {
			return before, nil
		}
		s.Queue = e.Queue
		s.Queue.Owners = append([]string(nil), e.Queue.Owners...)
		s.Queue.Active = append([]DrainActivity(nil), e.Queue.Active...)
		if s.Phase == DrainPreparing && drainOwnPause(s) {
			s.Phase = DrainActive
			s.Error = ""
		}
	case DrainCancel:
		if s.Phase != DrainCancelling {
			s.Phase = DrainCancelling
			s.Error = ""
			s.ResumeIssued = false
			for k, m := range s.Members {
				m.Ack = false
				m.AckSealed = false
				m.Revoke = DrainPending
				s.Members[k] = m
				fx = append(fx, s.effect(DrainRevoke, m))
			}
		}
	case DrainRecover:
		s.Phase = DrainRecovered
		s.Error = "recovered drain; cancel before a new drain"
		for k, m := range s.Members {
			m.Ack = false
			m.AckSealed = false
			m.Observation = DrainObservation{}
			s.Members[k] = m
		}
	case DrainFailed:
		s.Error = e.Detail
	case DrainMemberJoined:
		if s.Phase == DrainCancelling || s.Phase == DrainRecovered {
			return before, nil
		}
		if e.Workspace == "" || e.Identity == "" {
			s.Error = "invalid member identity"
			break
		}
		m, exists := s.Members[e.Workspace]
		if exists && m.Identity == e.Identity {
			return before, nil
		}
		if !exists && len(s.Members) >= MaxDrainMembers {
			s.Error = "membership exceeds limit"
			break
		}
		s.Members[e.Workspace] = DrainMember{Workspace: e.Workspace, Identity: e.Identity}
	default:
		m, ok := s.Members[e.Workspace]
		if !ok || m.Identity != e.Identity {
			return before, nil
		}
		switch e.Kind {
		case DrainMemberObserved:
			if e.Observation.Sequence != 0 && e.Observation.Sequence <= m.Observation.Sequence {
				return before, nil
			}
			o := e.Observation
			if m.AckSealed && (!o.Known || !o.Connected || !o.Idle || o.Picker || o.Background || o.Submission != m.AckSubmission) {
				m.Ack = false
				m.AckSealed = false
			}
			if m.Ack && (!o.Known || !o.Connected || o.Submission != m.AckSubmission) {
				m.Ack = false
				m.AckSealed = false
			}
			m.Observation = o
			if m.Ack && o.Known && o.Connected && o.Idle && !o.Picker && !o.Background {
				m.AckSealed = true
			}
		case DrainAcknowledged:
			if s.Phase != DrainActive || !drainOwnPause(s) || !drainControlTerminal(m.Notice) || !e.Observation.Known || !e.Observation.Connected {
				return before, nil
			}
			if !m.Ack {
				m.Ack = true
				m.AckSealed = false
				// The shell supplies identity-fenced activity sampled for this ack,
				// independently of the previous reconciliation observation.
				m.AckSubmission = e.Observation.Submission
			}
		case DrainNoticeResult:
			if m.Notice != DrainPending && m.Notice != DrainUnknown {
				return before, nil
			}
			m.Notice = e.Outcome
			m.Error = e.Detail
		case DrainInterruptResult:
			if m.Interrupt != DrainPending && m.Interrupt != DrainUnknown {
				return before, nil
			}
			m.Interrupt = e.Outcome
			m.Error = e.Detail
		case DrainRevoked:
			if s.Phase != DrainCancelling {
				return before, nil
			}
			m.Revoke = e.Outcome
			m.Error = e.Detail
		default:
			return before, nil
		}
		s.Members[e.Workspace] = m
	}
	if s.Phase == DrainActive && drainOwnPause(s) {
		for k, m := range s.Members {
			if s.Fast && m.Interrupt == "" {
				m.Interrupt = DrainPending
				fx = append(fx, s.effect(DrainInterrupt, m))
			}
			if m.Notice == "" && (!s.Fast || m.Interrupt == DrainDone) {
				m.Notice = DrainPending
				fx = append(fx, s.effect(DrainSendNotice, m))
			}
			s.Members[k] = m
		}
	}
	if s.Phase == DrainCancelling {
		terminal := true
		for _, m := range s.Members {
			if !drainControlTerminal(m.Revoke) {
				terminal = false
				break
			}
		}
		if terminal && (!s.ResumeIssued || e.Kind == DrainCancel) {
			s.ResumeIssued = true
			s.Queue = DrainQueueObservation{}
			fx = append(fx, s.effect(DrainResume, DrainMember{}))
		} else if terminal && s.ResumeIssued && s.Queue.Known && s.Queue.Complete {
			own := false
			for _, o := range s.Queue.Owners {
				own = own || o == s.Owner
			}
			if !own {
				s.Phase = DrainFinished
			}
		}
	}
	return s, fx
}
