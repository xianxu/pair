package drive

import (
	"context"
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
// order; what the run causes is attributed to by.
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
	Tabs TabRunner
	mu   sync.Mutex // guards held; runs are serialized by the tab anyway
	held pressed
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
// the controller calls it on every control change.
type Releaser interface {
	Release(ctx context.Context) error
}

// Release ends whatever the last driver left pressed, as nobody's input.
func (t *TabInjector) Release(ctx context.Context) error {
	return t.Tabs.WithTab(ctx, "", func(tab Tab) error { return t.release(tab) })
}

// release dispatches the ups for what is held, then forgets it whatever
// happened: an up that fails (the page went) leaves nothing to hold.
func (t *TabInjector) release(tab Tab) error {
	t.mu.Lock()
	h := t.held
	t.held = pressed{}
	t.mu.Unlock()
	var err error
	if h.button {
		err = tab.Dispatch(Act{A: ActUp, X: h.x, Y: h.y, Button: ButtonLeft})
	}
	for i := len(h.keys) - 1; i >= 0; i-- {
		k := h.keys[i]
		if e := tab.Dispatch(Act{A: ActKey, Type: KeyUp, Key: k.Key, Code: k.Code}); err == nil {
			err = e
		}
	}
	return err
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
		h.keys = append(h.keys, a)
	case a.A == ActKey && a.Type == KeyUp:
		h.keys = slices.DeleteFunc(h.keys, func(k Act) bool { return k.Key == a.Key })
	}
}

func (t *TabInjector) Inject(ctx context.Context, by string, b Batch) error {
	return t.Tabs.WithTab(ctx, by, func(tab Tab) error {
		err := t.inject(ctx, tab, by, b)
		t.mu.Lock()
		mine := t.held.by == by
		t.mu.Unlock()
		if err != nil && mine {
			t.release(tab)
		}
		return err
	})
}

func (t *TabInjector) inject(ctx context.Context, tab Tab, by string, b Batch) error {
	page := tab.Page()
	t.mu.Lock()
	h := t.held
	t.mu.Unlock()
	if !h.empty() && (h.by != page.Control.Driving() || h.seg != page.Seg) {
		t.release(tab) // another driver now, or another page: let go first
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
		if err := ctx.Err(); err != nil {
			return err
		}
		if a.A == ActDown || a.A == ActClick || a.A == ActWheel {
			hit, err := tab.At(a.X, a.Y)
			if err != nil {
				return err
			}
			if !onTarget(tab.Page().Node, hit, a.ID) {
				return ErrStale
			}
		}
		if err := tab.Dispatch(a); err != nil {
			return err
		}
		t.note(by, page.Seg, a)
	}
	return nil
	return nil
}

// onTarget is the stale rule: the element the tab has at the point (hit)
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
