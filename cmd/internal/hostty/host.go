package hostty

import (
	"io"
	"os"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// Host is the seam over the operator's terminal.
//
// Size uses ptychild.Size deliberately rather than a second identical type: the
// only arithmetic anyone does with it is "host size, minus the reserved row,
// is the child's size", so one type keeps that a subtraction instead of a
// conversion.
type Host interface {
	io.Writer

	// Size reports the terminal's current dimensions.
	Size() (ptychild.Size, error)

	// MakeRaw puts the terminal in raw mode and returns its restore. The
	// restore MUST be idempotent: a console restores on the child-exit path
	// and again from a deferred teardown, and on a signal it may race both.
	MakeRaw() (restore func() error, err error)

	// Resized fires when the terminal changes size. Deliveries COALESCE -- a
	// window drag emits SIGWINCH continuously, and one wake per signal would
	// resize every child dozens of times for one drag.
	Resized() <-chan struct{}

	// Close stops watching for resizes.
	Close() error
}

// TerminationHost is the optional process-lifecycle half needed by couch.
// pair term consumes the same Host seam but owns its lifecycle elsewhere, so
// termination does not inflate the shared base interface.
type TerminationHost interface {
	Terminated() <-chan os.Signal
}
