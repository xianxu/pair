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
type Injector interface {
	// Inject dispatches b's acts in order, as viewer by. It refuses the
	// whole batch, dispatching nothing, when the cast isn't live or b.Seg
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
