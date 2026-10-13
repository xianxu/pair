package drive

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"slices"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Tab is the cast tab as an injector sees it, during one run on the record
// stream's writer (#467). The real one is the recorder's loop: CDP input
// into the cast tab, hit tests through its id map. The fake is a reduced
// recording.State.
type Tab interface {
	// Page is the state of the stream the tab follows, now: live, the
	// segment, who drives, the DOM.
	Page() *recording.State
	// At is the element at a top-page CSS point in the tab, as a recording
	// id; 0 when the tab has none the recording knows.
	At(x, y float64) (recording.NodeID, error)
	// Dispatch performs one act Confine has allowed. Records it causes
	// carry the run's by.
	Dispatch(a Act) error
}

// TabRunner runs fn with the cast tab, on the stream's writer, in stream
// order; what the run causes is attributed to by. WithTab returns only after
// fn settles, or after guaranteeing fn will never start. Cancellation before
// dispatch returns a definitely-no-effect capability.Failure (Retryable).
type TabRunner interface {
	WithTab(ctx context.Context, by string, fn func(Tab) error) error
}

// TabInjector is the one statement of Injector's rules (#466), over any
// TabRunner: the real recorder and the fake tab share it.
//
// It also owns what a driver leaves pressed (#467): a button down without
// its up, keys down without theirs. Whatever the driver, the page or the
// viewer's shell does next (control moves, the page navigates, a refusal
// drops the viewer's queued ups), the tab never keeps them: they are
// released before anyone else's input, before input on another page, when
// the driver's own batch is refused (the shell drops what it queued after
// a refusal, ups included), and at once when control changes (Release,
// called by the controller).
type TabInjector struct {
	Tabs       TabRunner
	mu         sync.Mutex // guards held; runs are serialized by the tab anyway
	held       pressed
	releaseErr error
}

// pressed is what one driver holds down in the tab.
type pressed struct {
	by     string
	seg    int
	button bool
	x, y   float64
	keys   []Act // key downs, in order
}

func (p pressed) empty() bool { return !p.button && len(p.keys) == 0 }

// NewTabInjector is the injector over tabs.
func NewTabInjector(tabs TabRunner) *TabInjector { return &TabInjector{Tabs: tabs} }

// Releaser is an injector that can let go of what a driver left pressed;
// the controller calls it on every control change, naming the outgoing
// driver.
type Releaser interface {
	Release(ctx context.Context, by string) error
}

// Release ends what driver by left pressed, as nobody's input. It names
// the hold it ends: by the time it reaches the tab, a newer driver's batch
// may have run (and let go of by's hold itself), and that driver's press
// isn't by's to end.
func (t *TabInjector) Release(ctx context.Context, by string) error {
	t.mu.Lock()
	prior := t.releaseErr
	t.mu.Unlock()
	if prior != nil && capability.FailureOutcome(prior) == capability.Broken {
		return prior
	}
	if err := ctx.Err(); err != nil {
		return &ReleaseFailure{&capability.Failure{Outcome: capability.Retryable, Cause: err}}
	}
	return t.Tabs.WithTab(ctx, "", func(tab Tab) error {
		t.mu.Lock()
		mine := t.held.by == by
		prior := t.releaseErr
		t.mu.Unlock()
		// Text/click failures can be uncertain without owning any held key.
		// The terminal evidence belongs to the injector, not a particular hold.
		if prior != nil && capability.FailureOutcome(prior) == capability.Broken {
			return prior
		}
		if !mine {
			return nil
		}
		return t.release(ctx, tab)
	})
}

// ReleaseFailure distinguishes failed cleanup from a refused viewer action.
// Its cause retains the adapter's definitely-no-effect or uncertain classification.
type ReleaseFailure struct{ Cause error }

func (e *ReleaseFailure) Error() string { return "drive release: " + e.Cause.Error() }
func (e *ReleaseFailure) Unwrap() error { return e.Cause }

// release removes each hold only after its up is acknowledged. Stop at the
// first failure: uncertain transport effects must never be blindly replayed.
func (t *TabInjector) release(ctx context.Context, tab Tab) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.releaseErr != nil && capability.FailureOutcome(t.releaseErr) == capability.Broken {
		return t.releaseErr
	}
	h := &t.held
	check := func() error {
		if err := ctx.Err(); err != nil {
			t.releaseErr = &ReleaseFailure{&capability.Failure{Outcome: capability.Retryable, Cause: err}}
			return t.releaseErr
		}
		return nil
	}
	if h.button {
		if err := check(); err != nil {
			return err
		}
		if err := tab.Dispatch(Act{A: ActUp, X: h.x, Y: h.y, Button: ButtonLeft}); err != nil {
			t.releaseErr = &ReleaseFailure{err}
			return t.releaseErr
		}
		h.button = false
	}
	for len(h.keys) > 0 {
		if err := check(); err != nil {
			return err
		}
		k := h.keys[len(h.keys)-1]
		if err := tab.Dispatch(Act{A: ActKey, Type: KeyUp, Key: k.Key, Code: k.Code}); err != nil {
			t.releaseErr = &ReleaseFailure{err}
			return t.releaseErr
		}
		h.keys = h.keys[:len(h.keys)-1]
	}
	*h = pressed{}
	t.releaseErr = nil
	return nil
}

// note records what an act dispatched for by leaves pressed.
func (t *TabInjector) note(by string, seg int, a Act) {
	t.mu.Lock()
	defer t.mu.Unlock()
	h := &t.held
	h.by, h.seg = by, seg
	switch {
	case a.A == ActDown:
		h.button, h.x, h.y = true, a.X, a.Y
	case a.A == ActUp, a.A == ActClick:
		h.button = false
	case a.A == ActMove && h.button:
		h.x, h.y = a.X, a.Y
	case a.A == ActKey && a.Type == KeyDown:
		// Keyed by the key: an auto-repeated down holds it once.
		h.keys = append(slices.DeleteFunc(h.keys, func(k Act) bool { return k.Key == a.Key }), a)
	case a.A == ActKey && a.Type == KeyUp:
		h.keys = slices.DeleteFunc(h.keys, func(k Act) bool { return k.Key == a.Key })
	}
}

func (t *TabInjector) Inject(ctx context.Context, by string, b Batch) error {
	entered := false
	err := t.Tabs.WithTab(ctx, by, func(tab Tab) error {
		entered = true
		err := t.inject(ctx, tab, by, b)
		t.mu.Lock()
		mine := t.held.by == by
		t.mu.Unlock()
		var releaseErr *ReleaseFailure
		if err != nil && mine && !errors.As(err, &releaseErr) {
			if e := t.release(ctx, tab); e != nil {
				return e
			}
		}
		return err
	})
	if err != nil && !entered {
		// A queued up canceled before the callback never reaches refusal cleanup.
		// Keep its existing hold visible and require an explicit cleanup attempt.
		t.mu.Lock()
		if t.held.by == by && !t.held.empty() {
			if t.releaseErr == nil {
				t.releaseErr = &ReleaseFailure{err}
			}
			err = t.releaseErr
		}
		t.mu.Unlock()
	}
	return err
}

func (t *TabInjector) inject(ctx context.Context, tab Tab, by string, b Batch) error {
	t.mu.Lock()
	priorFailure := t.releaseErr
	t.mu.Unlock()
	if priorFailure != nil {
		return priorFailure
	}
	if b.Admit != nil && !b.Admit() {
		return ErrDriveUnavailable
	}
	page := tab.Page()
	t.mu.Lock()
	h := t.held
	t.mu.Unlock()
	if !h.empty() && (h.by != page.Control.Driving() || h.seg != page.Seg) {
		if err := t.release(ctx, tab); err != nil {
			return err
		} // another driver/page must wait for release
	}
	if by == "" || page.Control.Driving() != by {
		return ErrNotDriver
	}
	if !page.Live || page.Ended || b.Seg != page.Seg {
		return ErrNotLive
	}
	if err := confineAll(b); err != nil {
		return err
	}
	for _, a := range b.Acts {
		if b.Admit != nil && !b.Admit() {
			return ErrDriveUnavailable
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.A == ActDown || a.A == ActClick || a.A == ActWheel {
			hit, err := tab.At(a.X, a.Y)
			if err != nil {
				return err
			}
			// A wheel only scrolls whatever is under the pointer, so it
			// skips the identity check (pair#516, the operator's call: a
			// flick scrolls the tab ahead of the viewer, and its later
			// events still name what the viewer saw). It keeps the bounds
			// check: a point with nothing under it is refused.
			stale := !onTarget(tab.Page().Node, hit, a.ID)
			if a.A == ActWheel {
				stale = hit == 0
			}
			if stale {
				return ErrStale
			}
		}
		if b.Admit != nil && !b.Admit() {
			return ErrDriveUnavailable
		}
		if err := tab.Dispatch(a); err != nil {
			if capability.FailureOutcome(err) == capability.Broken {
				// An unacknowledged down may have taken effect. Keep that evidence;
				// an unacknowledged up must not be sent again as refusal cleanup.
				if a.A == ActDown || (a.A == ActKey && a.Type == KeyDown) {
					t.note(by, page.Seg, a)
				}
				t.mu.Lock()
				t.releaseErr = &ReleaseFailure{err}
				failure := t.releaseErr
				t.mu.Unlock()
				return failure
			}
			return err
		}
		t.note(by, page.Seg, a)
	}
	return nil
}

// onTarget is a down's and click's stale rule: the element the tab has at the point (hit)
// is the one the viewer aimed at (id) or inside it; or hit is a frame's
// owner and id is inside that frame (the tab's hit test stops at an
// out-of-process frame, where the viewer's nodeAt went on).
func onTarget(node func(recording.NodeID) *recording.DOMNode, hit, id recording.NodeID) bool {
	h, n := node(hit), node(id)
	if h == nil || n == nil {
		return false
	}
	if within(h, n) {
		return true
	}
	return (h.N == "iframe" || h.N == "frame") && h.Y == recording.YEl && within(n, h)
}

// within: n is a or inside it, across shadow roots and frames.
func within(n, a *recording.DOMNode) bool {
	for ; n != nil; n = up(n) {
		if n == a {
			return true
		}
	}
	return false
}

func up(n *recording.DOMNode) *recording.DOMNode {
	if n.Parent != nil {
		return n.Parent
	}
	return n.Host
}
