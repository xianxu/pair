package drive

import (
	"context"

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
type TabInjector struct{ Tabs TabRunner }

// NewTabInjector is the injector over tabs.
func NewTabInjector(tabs TabRunner) *TabInjector { return &TabInjector{Tabs: tabs} }

func (t *TabInjector) Inject(ctx context.Context, by string, b Batch) error {
	return t.Tabs.WithTab(ctx, by, func(tab Tab) error {
		page := tab.Page()
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
		}
		return nil
	})
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
