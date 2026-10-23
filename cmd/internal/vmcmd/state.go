package vmcmd

import "fmt"

// Effect is one bounded operation executed by the runtime owner.
type Effect string

const (
	NoEffect   Effect = "none"
	Acquire    Effect = "acquire"
	Prepare    Effect = "prepare"
	Start      Effect = "start"
	AwaitGuest Effect = "await-guest"
	Execute    Effect = "execute"
	Collect    Effect = "collect"
	Stop       Effect = "stop"
	Release    Effect = "release"
	Complete   Effect = "complete"
)

type phase uint8

const (
	phaseIdle phase = iota
	phaseAcquiring
	phasePreparing
	phaseStarting
	phaseAwaitingGuest
	phaseExecuting
	phaseCollecting
	phaseStopping
	phaseReleasing
	phaseFinished
)

// EventKind identifies an observed operation outcome, never a desired state.
type EventKind uint8

const (
	Begin EventKind = iota
	Acquired
	Prepared
	Started
	GuestReady
	CommandEnded
	Collected
	Stopped
	Released
	Failed
	Cancelled
	TimedOut
)

type Event struct {
	Kind   EventKind
	Exit   int
	Detail string
}

type resultKind uint8

const (
	unknownResult resultKind = iota
	commandResult
	failureResult
	cancelResult
)

// RunState owns orchestration decisions. Fields are private so callers cannot
// skip cleanup by changing phase or publishing success themselves.
type RunState struct {
	phase  phase
	result resultKind
	exit   int
	detail string
}

func NewRunState() RunState       { return RunState{exit: 125} }
func (s RunState) Done() bool     { return s.phase == phaseFinished }
func (s RunState) ExitCode() int  { return s.exit }
func (s RunState) Detail() string { return s.detail }

// Advance is pure. A failure during cleanup advances cleanup, preserving an
// earlier command failure; cancellation never releases a held position early.
func Advance(s RunState, e Event) (RunState, Effect, error) {
	before := s
	refuse := func() (RunState, Effect, error) {
		return before, NoEffect, fmt.Errorf("VM event %d is invalid in phase %d", e.Kind, s.phase)
	}
	if s.Done() {
		return refuse()
	}
	if e.Kind == Cancelled || e.Kind == TimedOut {
		if !(s.result == commandResult && s.exit != 0) {
			s.exit, s.result = 130, cancelResult
			if e.Kind == TimedOut {
				s.exit = 124
			}
		}
		s = s.withDetail(e.Detail)
		switch s.phase {
		case phaseIdle, phaseAcquiring:
			s.phase = phaseFinished
			return s, Complete, nil
		case phaseCollecting, phaseStopping, phaseReleasing:
			return s, NoEffect, nil
		default:
			s.phase = phaseCollecting
			return s, Collect, nil
		}
	}
	if e.Kind == Failed {
		if s.result != cancelResult && !(s.result == commandResult && s.exit != 0) {
			s.exit, s.result = 125, failureResult
		}
		s = s.withDetail(e.Detail)
		switch s.phase {
		case phaseIdle, phaseAcquiring:
			s.phase = phaseFinished
			return s, Complete, nil
		case phaseCollecting:
			s.phase = phaseStopping
			return s, Stop, nil
		case phaseStopping:
			s.phase = phaseReleasing
			return s, Release, nil
		case phaseReleasing:
			s.phase = phaseFinished
			return s, Complete, nil
		default:
			s.phase = phaseCollecting
			return s, Collect, nil
		}
	}
	steps := [...]struct {
		from, to phase
		event    EventKind
		effect   Effect
	}{
		{phaseIdle, phaseAcquiring, Begin, Acquire},
		{phaseAcquiring, phasePreparing, Acquired, Prepare},
		{phasePreparing, phaseStarting, Prepared, Start},
		{phaseStarting, phaseAwaitingGuest, Started, AwaitGuest},
		{phaseAwaitingGuest, phaseExecuting, GuestReady, Execute},
		{phaseExecuting, phaseCollecting, CommandEnded, Collect},
		{phaseCollecting, phaseStopping, Collected, Stop},
		{phaseStopping, phaseReleasing, Stopped, Release},
		{phaseReleasing, phaseFinished, Released, Complete},
	}
	for _, step := range steps {
		if step.from != s.phase || step.event != e.Kind {
			continue
		}
		if e.Kind == CommandEnded {
			if e.Exit < 0 || e.Exit > 255 {
				return refuse()
			}
			s.exit, s.result = e.Exit, commandResult
		}
		s.phase = step.to
		return s, step.effect, nil
	}
	return refuse()
}

func (s RunState) withDetail(detail string) RunState {
	if detail == "" {
		return s
	}
	if s.detail != "" {
		s.detail += "; "
	}
	s.detail += detail
	if len(s.detail) > 4096 {
		s.detail = s.detail[:4096]
	}
	return s
}
