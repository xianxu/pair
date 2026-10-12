package webcast

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Browser is the browser-control seam: one cast Chrome. Couch's UI, the Cast
// and its Server use only this; no CDP type crosses it. The recorder (#458)
// is not a client of it: it is part of the real Browser (Chrome), on the CDP
// side, fed the loop's CDP events and live phase (LaunchOptions.Record).
type Browser interface {
	// State is the current state.
	State() State
	// Subscribe delivers the latest State, starting with the current one.
	// The channel holds one value and is replaced, never blocking the
	// publisher; it closes after the final (PhaseEnded) state. cancel stops
	// delivery and is idempotent.
	Subscribe() (states <-chan State, cancel func())
	// SetLive moves between PhaseIdle and PhaseLive. ErrEnded once ended.
	SetLive(live bool) error
	// Focus brings the cast tab and its window to the front (#501): the
	// operator's own click, so it may take focus. ErrEnded once ended.
	Focus() error
	// Close ends the cast and closes Chrome. Idempotent; returns once Chrome
	// is gone.
	Close() error
	// Done closes when Chrome is gone, however it went.
	Done() <-chan struct{}
	// Err is why it ended: nil for Close, ErrClosedByOperator when Chrome was
	// quit by hand, else the failure. Nil while running.
	Err() error
}

var (
	// ErrEnded: the Browser has ended.
	ErrEnded = errors.New("webcast: Chrome has closed")
	// ErrClosedByOperator: Chrome was quit, or its last window closed, by hand.
	ErrClosedByOperator = errors.New("webcast: Chrome closed")
	// ErrNotRecording: the cast isn't recorded, so there is no stream to
	// write into.
	ErrNotRecording = errors.New("webcast: the cast isn't recorded")
)

// Emitter is a Browser that records: Emit passes a record made outside the
// recorder (#470's board; #467's control) into the record stream's one
// writer. doc is the document the record was made on, 0 for none; the
// writer drops the record if that document has gone (recording.State.OnDoc).
type Emitter interface {
	Emit(r recording.Record, doc int) error
}
