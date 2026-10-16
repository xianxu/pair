package nativelock

import "fmt"

// Phase is the wrapper's conversation-lock lifecycle, independent of IO.
// pair:m5-concept pure
type Phase uint8

const (
	Unknown Phase = iota
	Acquiring
	Held
	Stopping
	Exited
)

// State is immutable to callers; its zero value is an unknown-ID wrapper.
// The wrapper owner serializes Advance and executes its returned IO effects.
// Concrete handles stay with that owner until Release, after group cleanup.
// pair:m5-concept pure
type State struct {
	phase   Phase
	known   map[Key]bool
	pending map[Key]bool
}

func (s State) Phase() Phase     { return s.phase }
func (s State) Known(k Key) bool { return s.known[k] }

// pair:m5-concept pure
type EventKind uint8

const (
	Discovered EventKind = iota
	Acquired
	Failed
	Stop
	CleanupComplete
)

// pair:m5-concept pure
type Event struct {
	Kind EventKind
	Key  Key
}

// Effects are instructions, not IO. Release closes every handle the owner has,
// including a late handle returned after cleanup has already finished.
// pair:m5-concept pure
type Effects struct {
	Acquire *Key
	Stop    bool
	Release bool
}

// Advance retains known locks through stopping, rejects late discoveries and
// idempotently handles repeated reports. Only CleanupComplete (or completion
// after that boundary) permits releasing handles.
// pair:m5-concept pure
func Advance(s State, e Event) (State, Effects, error) {
	unchanged := s
	invalid := func(reason string) (State, Effects, error) {
		return unchanged, Effects{}, fmt.Errorf("native lock lifecycle: %s", reason)
	}
	switch e.Kind {
	case Discovered:
		if s.phase == Stopping || s.phase == Exited {
			return invalid("discovery after stop")
		}
		if _, err := NewKey(e.Key.agent, e.Key.id); err != nil {
			return invalid("invalid discovered key")
		}
		if s.known[e.Key] || s.pending[e.Key] {
			return s, Effects{}, nil
		}
		s.pending = copySet(s.pending)
		s.pending[e.Key] = true
		s.phase = Acquiring
		key := e.Key
		return s, Effects{Acquire: &key}, nil
	case Acquired, Failed:
		if !s.pending[e.Key] {
			return invalid("completion without pending acquisition")
		}
		s.pending = copySet(s.pending)
		delete(s.pending, e.Key)
		if e.Kind == Acquired {
			if s.phase == Exited {
				return s, Effects{Release: true}, nil
			}
			s.known = copySet(s.known)
			s.known[e.Key] = true
		}
		if s.phase == Stopping || s.phase == Exited {
			return s, Effects{}, nil
		}
		if e.Kind == Failed {
			s.phase = Stopping
			return s, Effects{Stop: true}, nil
		}
		s.phase = Held
		if len(s.pending) > 0 {
			s.phase = Acquiring
		}
		return s, Effects{}, nil
	case Stop:
		if s.phase == Stopping || s.phase == Exited {
			return s, Effects{}, nil
		}
		s.phase = Stopping
		return s, Effects{Stop: true}, nil
	case CleanupComplete:
		if s.phase == Exited {
			return s, Effects{}, nil
		}
		if s.phase != Stopping {
			return invalid("cleanup completed before stop")
		}
		s.phase = Exited
		s.known = nil
		return s, Effects{Release: true}, nil
	default:
		return invalid("unknown event")
	}
}
func copySet(in map[Key]bool) map[Key]bool {
	out := make(map[Key]bool, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}
