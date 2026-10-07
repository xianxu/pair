package terminalcapture

import "errors"

// lifecycle exposes observations and events, never writable authoritative state.
// Recorder serializes access with its mutex and executes the returned effects.
type lifecycle interface {
	snapshot() lifecycleSnapshot
	transition(lifecycleEvent) lifecycleEffects
}

type lifecycleSnapshot struct {
	phase Phase
	err   error
}

type lifecycleEventKind uint8

const (
	closeRequested lifecycleEventKind = iota
	captureFailed
	drainTimedOut
	workerCompleted
)

type lifecycleEvent struct {
	kind lifecycleEventKind
	err  error // Present only for captureFailed; timeout has its own fixed error.
}

type lifecycleEffects struct {
	closeAdmission bool
	notify         bool
}

type captureLifecycle struct{ state lifecycleSnapshot }

func newLifecycle() lifecycle {
	return &captureLifecycle{state: lifecycleSnapshot{phase: Recording}}
}

func (m *captureLifecycle) snapshot() lifecycleSnapshot { return m.state }

func (m *captureLifecycle) transition(event lifecycleEvent) lifecycleEffects {
	next, effects := nextLifecycle(m.state, event)
	m.state = next
	return effects
}

// nextLifecycle is the sole transition authority. No clocks, locks, channels or
// IO enter this model. Close stops admission once; only confirmed worker
// completion closes a healthy drain. Failure stays failed after late completion.
// Disabled/Closed ignore late events, and duplicate failures do not grow errors
// or generate wakeups. A failed capture may additionally report a drain timeout.
func nextLifecycle(state lifecycleSnapshot, event lifecycleEvent) (lifecycleSnapshot, lifecycleEffects) {
	if state.phase == Disabled || state.phase == Closed {
		return state, lifecycleEffects{}
	}
	previous := state.phase
	changedError := false
	switch event.kind {
	case closeRequested:
		if state.phase == Recording {
			state.phase = Draining
		}
	case workerCompleted:
		if state.phase == Draining {
			state.phase = Closed
		}
	case captureFailed, drainTimedOut:
		failure := event.err
		if event.kind == drainTimedOut {
			if state.phase == Recording {
				return state, lifecycleEffects{}
			}
			failure = ErrCloseTimeout
		}
		if failure == nil {
			return state, lifecycleEffects{}
		}
		state.phase = Failed
		if !errors.Is(state.err, failure) {
			state.err = errors.Join(state.err, failure)
			changedError = true
		}
	}
	return state, lifecycleEffects{
		closeAdmission: previous == Recording && state.phase != Recording,
		notify:         previous != state.phase || changedError,
	}
}
