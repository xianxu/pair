package couchtty

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// fakeSession records the anchors a resolver sets; open is the notes the
// session still has (a resolved note's anchor is dropped).
type fakeSession struct {
	mu   sync.Mutex
	open map[int]bool
	set  map[int]broadcast.NoteAnchor
}

func (s *fakeSession) SetNoteAnchor(id int, a broadcast.NoteAnchor) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.open[id] {
		return false
	}
	s.set[id] = a
	return true
}

// paneLayout is a stateful fake of a slot's zellij session: its panes now,
// which a test changes between queries, and how often it was asked.
type paneLayout struct {
	mu      sync.Mutex
	panes   []PaneRect
	err     error
	queries int
	// during runs inside a query, as the world moves on while zellij
	// answers (a resize, a resolve).
	during func()
}

func (p *paneLayout) rects(_ context.Context, _ couchcore.ThreadAddress) ([]PaneRect, error) {
	p.mu.Lock()
	p.queries++
	during, panes, err := p.during, append([]PaneRect(nil), p.panes...), p.err
	p.mu.Unlock()
	if during != nil {
		during()
	}
	return panes, err
}

// anchorRig is a console holding the 40×10 anchorScreen as its last
// public frame, with notes opened on it waiting for their pane path.
type anchorRig struct {
	c      *Console
	s      *fakeSession
	layout *paneLayout
	frame  terminal.Frame
}

func newAnchorRig(t *testing.T) *anchorRig {
	t.Helper()
	f, l := anchorScreen()
	f.GeometryEpoch, f.EndpointID = 7, "pair:3"
	panes := l.Panes
	r := &anchorRig{c: &Console{lifetime: context.Background()}, s: &fakeSession{open: map[int]bool{}, set: map[int]broadcast.NoteAnchor{}},
		layout: &paneLayout{panes: panes}, frame: f}
	r.c.keepNoteFrame(f)
	r.c.SetPaneRects(r.layout.rects)
	return r
}

// open queues note id on box as the console would: its anchor as it
// opened, from memory (no panes yet).
func (r *anchorRig) open(id int, box broadcast.CellBox) {
	_, l := anchorScreen()
	l.Panes = nil
	n := broadcast.Note{ID: id, Box: box, On: r.frame.Geometry}
	a := AnchorFor(box, r.frame, l)
	n.Anchor = &a
	r.s.open[id] = true
	r.c.anchors.pending = append(r.c.anchors.pending, anchorJob{session: r.s, note: n, slot: couchcore.ThreadAddress{RepoScope: "pair", Tag: "3"}, layout: l, frame: r.frame})
}

func (r *anchorRig) path(id int) string {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	a, ok := r.s.set[id]
	if !ok {
		return ""
	}
	return strings.Join(a.Path, " › ")
}

func TestResolverFillsThePanePathAndKeepsTheOpeningText(t *testing.T) {
	r := newAnchorRig(t)
	r.open(1, cb(0, 0, 24, 3))
	r.open(2, cb(0, 6, 20, 2))
	r.c.resolveAnchors()
	if got := r.path(1); got != "pane pair:3 › agent (claude)" {
		t.Errorf("note 1: %q", got)
	}
	if got := r.path(2); got != "pane pair:3 › draft nvim › draft-pair-3.md, lines 12–13 › section: Review" {
		t.Errorf("note 2: %q", got)
	}
	if r.layout.queries != 1 {
		t.Errorf("two notes in one pass asked zellij %d times, want once", r.layout.queries)
	}
	if a := r.s.set[1]; a.Kind != broadcast.AnchorPane || !strings.HasPrefix(a.Text, "> fix the header color") {
		t.Errorf("note 1's anchor %+v", a)
	}
}

// The events the resolver can't block: a note resolved while zellij
// answers, a resize meanwhile, a query that fails.
func TestResolverOrdering(t *testing.T) {
	t.Run("a note resolved during the pass stays resolved", func(t *testing.T) {
		r := newAnchorRig(t)
		r.open(1, cb(0, 0, 24, 3))
		r.layout.during = func() { r.s.mu.Lock(); delete(r.s.open, 1); r.s.mu.Unlock() }
		r.c.resolveAnchors()
		if got := r.path(1); got != "" {
			t.Errorf("a resolved note got an anchor: %q", got)
		}
	})
	t.Run("a resize during the pass drops the answer", func(t *testing.T) {
		r := newAnchorRig(t)
		r.open(1, cb(0, 0, 24, 3))
		r.layout.during = func() {
			g := r.frame
			g.GeometryEpoch++
			r.c.keepNoteFrame(g)
		}
		r.c.resolveAnchors()
		if got := r.path(1); got != "" {
			t.Errorf("an answer for the old layout applied: %q", got)
		}
	})
	t.Run("a slot switch during the pass drops the answer", func(t *testing.T) {
		r := newAnchorRig(t)
		r.open(1, cb(0, 6, 20, 2))
		r.layout.during = func() {
			g, _ := anchorScreen() // another slot, same size and epoch
			g.GeometryEpoch, g.EndpointID = 7, "pair:1"
			r.c.keepNoteFrame(g)
		}
		r.c.resolveAnchors()
		if got := r.path(1); got != "" {
			t.Errorf("an answer for the slot no longer in view applied: %q", got)
		}
	})
	t.Run("a failed query leaves the anchor as it opened, without a retry", func(t *testing.T) {
		r := newAnchorRig(t)
		r.open(1, cb(0, 0, 24, 3))
		r.layout.err = errors.New("zellij: no session")
		r.c.resolveAnchors()
		if got := r.path(1); got != "" || r.layout.queries != 1 {
			t.Errorf("after a failure: %q, %d queries", got, r.layout.queries)
		}
	})
	t.Run("a note opened during a pass waits for the next pass", func(t *testing.T) {
		r := newAnchorRig(t)
		r.open(1, cb(0, 0, 24, 3))
		r.layout.during = func() {
			r.layout.during = nil
			r.c.anchors.mu.Lock()
			defer r.c.anchors.mu.Unlock()
			r.open(2, cb(0, 6, 20, 2)) // queued while the first pass waits on zellij
		}
		r.c.resolveAnchors()
		if r.path(1) == "" || r.path(2) == "" || r.layout.queries != 2 {
			t.Errorf("paths %q %q after %d queries; want both, from two passes", r.path(1), r.path(2), r.layout.queries)
		}
		if r.c.anchors.running {
			t.Error("the resolver still marked running after it returned")
		}
	})
}

// Through the broadcast: a note opens with the text under its box, then
// gets its pane path from the slot's panes (the stateful fake standing in
// for zellij); the bottom row's anchor needs no query.
func TestNoteAnchorThroughTheBroadcast(t *testing.T) {
	f, s, link := pointerOnFixture(t, nil)
	layout := &paneLayout{panes: []PaneRect{{Role: PaneAgent, X: 0, Y: 0, W: 80, H: 23}}}
	f.con.SetPaneRects(layout.rects)
	noteUntilSaved(t, link, 10, 5, "this line")
	waitFor(t, "the pane path", func() bool {
		notes, _ := s.Notes()
		return len(notes) == 1 && notes[0].Anchor != nil && notes[0].Anchor.Kind == broadcast.AnchorPane
	})
	notes, _ := s.Notes()
	if a := notes[0].Anchor; len(a.Path) != 2 || !strings.HasPrefix(a.Path[0], "pane ") || !strings.HasPrefix(a.Path[1], "agent") {
		t.Errorf("anchor %+v", a)
	}
	if layout.count() != 1 {
		t.Errorf("zellij asked %d times for one note", layout.count())
	}
	noteUntilSaved(t, link, 40, 23, "the tab bar")
	waitFor(t, "two notes", func() bool { n, _ := s.Notes(); return len(n) == 2 })
	notes, _ = s.Notes()
	if a := notes[1].Anchor; a == nil || (a.Kind != broadcast.AnchorTabBar && a.Kind != broadcast.AnchorStatusLine) {
		t.Errorf("a last-row note's anchor %+v", a)
	}
	if layout.count() != 1 {
		t.Errorf("a last-row note asked zellij (%d queries)", layout.count())
	}
}

func (p *paneLayout) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.queries
}

// A new broadcast is a new session whose ids start again: its note 1 is
// queued though the last session's note 1 was (BR-15). A note whose text
// under the box changed before it was queued is not (the screen moved).
func TestQueueAnchorJobsPerSession(t *testing.T) {
	r := newAnchorRig(t)
	_, l := anchorScreen()
	l.Panes = nil
	note := func(id int, box broadcast.CellBox) broadcast.Note {
		a := AnchorFor(box, r.frame, l)
		return broadcast.Note{ID: id, Box: box, On: r.frame.Geometry, Anchor: &a}
	}
	slot := couchcore.ThreadAddress{RepoScope: "pair", Tag: "3"}
	first, second := &fakeSession{}, &fakeSession{}
	if !r.c.queueAnchorJobs(first, []broadcast.Note{note(1, cb(0, 0, 24, 3))}, l, slot) {
		t.Fatal("the first job must start the resolver")
	}
	if r.c.queueAnchorJobs(first, []broadcast.Note{note(1, cb(0, 0, 24, 3))}, l, slot) || len(r.c.anchors.pending) != 1 {
		t.Fatalf("a note is queued once: %d jobs", len(r.c.anchors.pending))
	}
	r.c.queueAnchorJobs(second, []broadcast.Note{note(1, cb(0, 6, 20, 2))}, l, slot)
	if len(r.c.anchors.pending) != 2 || r.c.anchors.pending[1].session != second {
		t.Fatalf("the next session's note 1 must be queued: %d jobs", len(r.c.anchors.pending))
	}
	if len(r.c.anchors.tried) != 1 {
		t.Errorf("tried keeps only the latest session's open notes: %v", r.c.anchors.tried)
	}
	moved := note(2, cb(0, 0, 24, 3))
	moved.Anchor.Text = "what was there before"
	r.c.queueAnchorJobs(second, []broadcast.Note{note(1, cb(0, 6, 20, 2)), moved}, l, slot)
	if len(r.c.anchors.pending) != 2 {
		t.Errorf("a note whose screen moved was queued")
	}
}

func TestForgetAnchorsLeavesNothingOfTheSession(t *testing.T) {
	r := newAnchorRig(t)
	r.open(1, cb(0, 0, 24, 3))
	r.c.anchors.tried = map[triedNote]bool{{r.s, 1}: true}
	r.c.forgetAnchors()
	if len(r.c.anchors.pending) != 0 || len(r.c.anchors.tried) != 0 {
		t.Errorf("after the broadcast: %d jobs, %d tried", len(r.c.anchors.pending), len(r.c.anchors.tried))
	}
}
