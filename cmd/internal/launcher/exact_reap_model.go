package launcher

import (
	"fmt"
	"syscall"
)

// ExactReapState is invocation-local progress. Only AdvanceExactReap changes it.
type ExactReapState struct {
	rows        []exactReapRow
	phase       exactReapPhase
	outstanding uint64
	serial      uint64
	started     bool
}
type exactReapPhase uint8

const (
	exactTermDispatch exactReapPhase = iota
	exactTermWait
	exactKillDispatch
	exactKillWait
	exactDone
)

type exactReapRowState uint8

const (
	exactPendingTerm exactReapRowState = iota
	exactSendingTerm
	exactWaitingTerm
	exactPendingKill
	exactSendingKill
	exactWaitingKill
	exactRowDone
)

type exactReapRow struct {
	step    ReapStep
	state   exactReapRowState
	outcome ExactReapOutcome
}
type exactReapEventKind uint8

const (
	exactBegin exactReapEventKind = iota
	exactNext
	exactSignalReply
	exactObservation
	exactCancel
)

type exactReapEvent struct {
	kind         exactReapEventKind
	steps        []ReapStep
	token        uint64
	index        int
	status       string
	detail       string
	observations []exactPresence
	deadline     bool
}
type exactPresence uint8

const (
	exactUnknown exactPresence = iota
	exactAbsent
	exactAlive
)

type exactReapEffectKind uint8

const (
	exactSignalEffect exactReapEffectKind = iota
	exactProbeEffect
)

type exactReapEffect struct {
	kind   exactReapEffectKind
	token  uint64
	index  int
	step   ReapStep
	signal syscall.Signal
	phase  exactReapPhase
	first  bool
}

// AdvanceExactReap is pure: it owns all progress and produces at most one
// outstanding effect. Invalid, duplicated and out-of-order events do nothing.
func AdvanceExactReap(s ExactReapState, e exactReapEvent) (ExactReapState, []exactReapEffect) {
	s.rows = append([]exactReapRow(nil), s.rows...)
	finish := func(i int, status, detail string) {
		if status == "failed" || status == "refused" || status == "skipped" {
			progress := "no signal acknowledged"
			switch s.rows[i].state {
			case exactSendingTerm:
				progress = "TERM outcome unconfirmed"
			case exactWaitingTerm, exactPendingKill:
				progress = "TERM acknowledged"
			case exactSendingKill:
				progress = "TERM acknowledged; KILL outcome unconfirmed"
			case exactWaitingKill:
				progress = "TERM and KILL acknowledged"
			}
			detail = fmt.Sprintf("%s; %s", progress, detail)
		}
		s.rows[i].state = exactRowDone
		s.rows[i].outcome = ExactReapOutcome{Step: s.rows[i].step, Status: status, Detail: detail}
	}
	if e.kind == exactBegin {
		if s.started {
			return s, nil
		}
		s.started = true
		seen := map[ReapStep]bool{}
		for _, step := range e.steps {
			step.depth = 0
			if seen[step] {
				continue
			}
			seen[step] = true
			s.rows = append(s.rows, exactReapRow{step: step})
			if step.PID <= 0 || step.Identity == "" {
				finish(len(s.rows)-1, "refused", "exact process identity is required")
			}
		}
		return s, nil
	}
	if !s.started || s.phase == exactDone {
		return s, nil
	}
	if e.kind == exactCancel {
		for i, r := range s.rows {
			if r.state != exactRowDone {
				finish(i, "failed", e.detail)
			}
		}
		s.phase = exactDone
		s.outstanding = 0
		return s, nil
	}
	if e.kind == exactSignalReply {
		if s.outstanding == 0 || e.token != s.outstanding || e.index < 0 || e.index >= len(s.rows) {
			return s, nil
		}
		r := &s.rows[e.index]
		if r.state != exactSendingTerm && r.state != exactSendingKill {
			return s, nil
		}
		s.outstanding = 0
		switch e.status {
		case "sent":
			if r.state == exactSendingTerm {
				r.state = exactWaitingTerm
			} else {
				r.state = exactWaitingKill
			}
		case "skipped", "refused", "failed":
			finish(e.index, e.status, e.detail)
		default:
			finish(e.index, "failed", "invalid signal acknowledgment")
		}
		return s, nil
	}
	if e.kind == exactObservation {
		if s.outstanding == 0 || e.token != s.outstanding || (s.phase != exactTermWait && s.phase != exactKillWait) || len(e.observations) != len(s.rows) {
			return s, nil
		}
		s.outstanding = 0
		for i, r := range s.rows {
			if r.state != exactWaitingTerm && r.state != exactWaitingKill {
				continue
			}
			switch e.observations[i] {
			case exactAbsent:
				finish(i, "reaped", "exact process is absent after acknowledged signal")
			case exactAlive:
				if e.deadline {
					if r.state == exactWaitingTerm {
						s.rows[i].state = exactPendingKill
					} else {
						finish(i, "failed", "exact process survived SIGKILL deadline")
					}
				}
			default:
				finish(i, "failed", "process presence is unknown after acknowledged signal")
			}
		}
		if e.deadline {
			if s.phase == exactTermWait {
				s.phase = exactKillDispatch
			} else {
				s.phase = exactDone
			}
		}
		return s, nil
	}
	if e.kind != exactNext || s.outstanding != 0 {
		return s, nil
	}
	effect := func(v exactReapEffect) (ExactReapState, []exactReapEffect) {
		s.serial++
		s.outstanding = s.serial
		v.token = s.serial
		return s, []exactReapEffect{v}
	}
	for {
		switch s.phase {
		case exactTermDispatch, exactKillDispatch:
			pending, sending, sig := exactPendingTerm, exactSendingTerm, syscall.SIGTERM
			if s.phase == exactKillDispatch {
				pending, sending, sig = exactPendingKill, exactSendingKill, syscall.SIGKILL
			}
			for i, r := range s.rows {
				if r.state == pending {
					s.rows[i].state = sending
					return effect(exactReapEffect{kind: exactSignalEffect, index: i, step: r.step, signal: sig, phase: s.phase})
				}
			}
			if s.phase == exactTermDispatch {
				s.phase = exactTermWait
			} else {
				s.phase = exactKillWait
			}
			any := false
			for _, r := range s.rows {
				if r.state == exactWaitingTerm || r.state == exactWaitingKill {
					any = true
				}
			}
			if !any {
				if s.phase == exactTermWait {
					s.phase = exactKillDispatch
					continue
				}
				s.phase = exactDone
				return s, nil
			}
			return effect(exactReapEffect{kind: exactProbeEffect, phase: s.phase, first: true})
		case exactTermWait, exactKillWait:
			any := false
			for _, r := range s.rows {
				if r.state == exactWaitingTerm || r.state == exactWaitingKill {
					any = true
				}
			}
			if !any {
				if s.phase == exactTermWait {
					s.phase = exactKillDispatch
					continue
				}
				s.phase = exactDone
				return s, nil
			}
			return effect(exactReapEffect{kind: exactProbeEffect, phase: s.phase})
		default:
			return s, nil
		}
	}
}
