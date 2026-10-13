package drive

import (
	"context"
	"errors"
)

// Injector dispatches a driver's input into the cast tab's page (#466). The
// real one (#467) runs as an input of the recorder's queue, the record
// stream's one writer: CDP Input.dispatchMouseEvent, dispatchKeyEvent,
// insertText and imeSetComposition into the active tab, hit tests through
// DOM.getNodeForLocation and the recorder's id map (Recorder.IDOf,
// NodeOf), and what the observer reports while it dispatches carries by.
//
// An injector follows the record stream it injects into: it sees every
// record, ctl included, in stream order (the real one is on the recorder's
// queue; a fake is fed as a recording.Sink). So it knows who drives at the
// moment it dispatches, and a batch overtaken by a revoke is refused there.
//
// Keys are dispatched from Key alone: the virtual key code and the text
// derive from Key; Code only fills the DOM event's code, and Confine holds
// it to Key's own key.
type Injector interface {
	// Inject dispatches b's acts in order, as viewer by. It refuses the
	// whole batch, dispatching nothing, when by isn't the driver in the
	// stream it follows (ErrNotDriver), when the cast isn't live or b.Seg
	// isn't the current segment (ErrNotLive), or when any act fails Confine
	// (ErrRefused). A down, click or wheel whose element isn't at its point
	// any more stops the batch there (ErrStale): the acts before it stand.
	// Records it causes carry by (recording.Source).
	Inject(ctx context.Context, by string, b Batch) error
}

var (
	// ErrNotLive: the cast is paused or ended, or the batch was aimed at a
	// segment that has gone.
	ErrNotLive = errors.New("drive: the cast isn't showing that page")
	// ErrStale: the element the viewer aimed at isn't at that point in the
	// tab any more.
	ErrStale = errors.New("drive: the page changed")
	// ErrDialog: the cast tab is showing a JavaScript dialog, which stops
	// input and the page's script until the operator answers it (#467).
	ErrDialog = errors.New("drive: the page is showing a dialog")
	// ErrInput: the tab failed to take an act it was allowed (a CDP input
	// call or hit test failed). It wraps the cause, for the operator's log;
	// the viewer is told only that it didn't work (pair#505).
	ErrInput = errors.New("drive: the cast couldn't take the input")
)

// confineAll checks a whole batch before anything is dispatched.
func confineAll(b Batch) error {
	for _, a := range b.Acts {
		if err := Confine(a); err != nil {
			return err
		}
	}
	return nil
}
