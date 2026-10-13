// Package capability models admission and cleanup independently of their IO owners.
package capability

import "fmt"

// Phase is a capability's lifecycle. Its zero value is Off.
type Phase uint8

const (
	Off Phase = iota
	On
	Disabling
	Ended
)

// CleanupOutcome is present only while Disabling.
type CleanupOutcome uint8

const (
	NoCleanup CleanupOutcome = iota
	Running
	Retryable
	Broken
)

// CapabilityState is a comparable snapshot owned by one capability authority.
// Attempt remains allocated across Off/On cycles, so old completions cannot
// match a later disable. Generation fences previously admitted work.
type CapabilityState struct {
	Phase      Phase
	Generation uint64
	Attempt    uint64
	Outcome    CleanupOutcome
}

type EventKind uint8

const (
	Enable EventKind = iota
	Disable
	Retry
	CleanupSucceeded
	CleanupFailed
	End
)

// Event is tagged by Kind. Only cleanup completions carry Attempt; only
// CleanupFailed carries Outcome, which must be Retryable or Broken.
type Event struct {
	Kind    EventKind
	Attempt uint64
	Outcome CleanupOutcome
}

// Permit binds work to one admission generation. It is local to its issuing
// authority; owners must not exchange permits across capabilities or casts.
type Permit struct {
	generation uint64
	issued     bool
}

// Permit captures admission; the owner must recheck it at mutation dispatch.
func (s CapabilityState) Permit() (Permit, bool) {
	if !s.valid() || s.Phase != On {
		return Permit{}, false
	}
	return Permit{generation: s.Generation, issued: true}, true
}

// Admits refuses work whenever admission is closed or its generation is stale.
func (s CapabilityState) Admits(p Permit) bool {
	return p.issued && s.valid() && s.Phase == On && s.Generation == p.generation
}

// NextCapability returns a new snapshot without IO. Invalid states, malformed
// events and forbidden commands return the original state and an error.
// Completions apply only to the current Running attempt: duplicates, late
// replies and replies after failure/end cannot change settled outcomes.
func NextCapability(s CapabilityState, e Event) (CapabilityState, error) {
	if !s.valid() {
		return s, fmt.Errorf("invalid capability state: %+v", s)
	}
	if !e.valid() {
		return s, fmt.Errorf("invalid capability event: %+v", e)
	}
	next := s
	switch e.Kind {
	case Enable:
		if s.Phase == On {
			return s, nil
		}
		if s.Phase != Off {
			return s, refused(s, e)
		}
		next.Phase = On
		next.Generation++
	case Disable:
		if s.Phase == Off {
			return s, nil
		}
		if s.Phase != On {
			return s, refused(s, e)
		}
		next.Phase = Disabling
		next.Outcome = Running
		next.Generation++
		next.Attempt++
	case Retry:
		if s.Phase != Disabling || s.Outcome != Retryable {
			return s, refused(s, e)
		}
		next.Outcome = Running
		next.Attempt++
	case CleanupSucceeded, CleanupFailed:
		if s.Phase != Disabling || s.Outcome != Running || e.Attempt != s.Attempt {
			return s, nil
		}
		if e.Kind == CleanupSucceeded {
			next.Phase = Off
			next.Outcome = NoCleanup
		} else {
			next.Outcome = e.Outcome
		}
	case End:
		if s.Phase == Ended {
			return s, nil
		}
		next.Phase = Ended
		next.Outcome = NoCleanup
		next.Generation++
	}
	if next.Generation < s.Generation || next.Attempt < s.Attempt {
		return s, fmt.Errorf("capability identity exhausted")
	}
	return next, nil
}

func (s CapabilityState) valid() bool {
	if s.Phase > Ended {
		return false
	}
	if s.Phase == Disabling {
		return s.Attempt != 0 && s.Outcome >= Running && s.Outcome <= Broken
	}
	return s.Outcome == NoCleanup
}

func (e Event) valid() bool {
	switch e.Kind {
	case Enable, Disable, Retry, End:
		return e.Attempt == 0 && e.Outcome == NoCleanup
	case CleanupSucceeded:
		return e.Attempt != 0 && e.Outcome == NoCleanup
	case CleanupFailed:
		return e.Attempt != 0 && (e.Outcome == Retryable || e.Outcome == Broken)
	default:
		return false
	}
}

func refused(s CapabilityState, e Event) error {
	return fmt.Errorf("capability event %d refused in phase %d, outcome %d", e.Kind, s.Phase, s.Outcome)
}
