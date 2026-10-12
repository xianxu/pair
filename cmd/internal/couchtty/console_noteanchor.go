package couchtty

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// Terminal notes' anchors (#488), the shell around AnchorFor. A note opens
// with what memory answers (anchorNote, on the request's goroutine): the
// text under its box from the last public frame, and on the last row the
// tab bar or status line. A box above it is in the slot's zellij session,
// whose pane layout Couch learns only by asking zellij (about 590 ms on
// 0.44.3, #220), so the pane path comes after: a single-flight resolver
// asks once per pass, after every note it serves opened, so the answer is
// fresh for them, and sets each note's anchor through the session.
//
// Ordering (ARCH-ORDER). A resolved anchor applies only to the note it was
// asked for: the session drops it once the note is resolved or pointing
// turned off (ids are never reused). It is dropped here when the screen's
// geometry changed since the note opened (a resize mid-pass: the pane
// layout no longer matches the box) or the slot in view isn't the note's.
// A note opened during a pass waits for the next pass. A failed or timed
// out query leaves the anchor as it opened, kind AnchorScreen; nothing is
// retried.

// paneRectsTimeout bounds one pane query: about 3× list-panes' measured cost.
const paneRectsTimeout = 2 * time.Second

// PaneRects lists a slot's panes in frame cells, with what each pair nvim
// shows. couchcmd's asks the slot's zellij session (paneRectsOf).
type PaneRects func(context.Context, couchcore.ThreadAddress) ([]PaneRect, error)

// noteAnchors is the resolver's state. mu is a leaf lock.
type noteAnchors struct {
	mu sync.Mutex
	// frame is the last public frame before overlays, kept by the overlay
	// on the Presenter goroutine; frames are immutable once composed.
	frame   terminal.Frame
	rects   PaneRects
	pending []anchorJob
	running bool
	// tried is the open notes already queued, so a note is asked about
	// once; it keeps only ids still open (pruned on every snapshot).
	tried map[int]bool
}

// anchorJob is one note waiting for its pane path: the note as it opened,
// the slot in view and the layout then, and the frame's geometry epoch.
type anchorJob struct {
	session anchorSetter
	note    broadcast.Note
	slot    couchcore.ThreadAddress
	layout  ScreenLayout
	epoch   uint64
}

// anchorSetter is where a resolved anchor goes: the broadcast session.
type anchorSetter interface {
	SetNoteAnchor(id int, a broadcast.NoteAnchor) bool
}

// SetPaneRects supplies the pane query; without one, notes above the last
// row keep the anchor they opened with.
func (c *Console) SetPaneRects(r PaneRects) {
	c.anchors.mu.Lock()
	defer c.anchors.mu.Unlock()
	c.anchors.rects = r
}

// keepNoteFrame keeps the public frame notes are placed on. It runs on the
// Presenter goroutine.
func (c *Console) keepNoteFrame(f terminal.Frame) {
	c.anchors.mu.Lock()
	c.anchors.frame = f
	c.anchors.mu.Unlock()
}

// noteLayoutLocked is the screen's layout as Couch drew it last: the tab
// bar's chips, the slot in view and its agent. Callers hold c.mu; no IO.
func (c *Console) noteLayoutLocked() (ScreenLayout, couchcore.ThreadAddress) {
	var l ScreenLayout
	for _, ch := range c.statusChips {
		l.Chips = append(l.Chips, ColumnSpan{ch.Start, ch.End})
	}
	p := c.panes[c.active]
	if p == nil {
		return l, couchcore.ThreadAddress{}
	}
	for _, a := range c.statusModelLocked().Actors {
		if a.Thread == p.thread {
			l.Slot = a.Label
		}
	}
	if row, ok := menuThread(c.menu, p.thread); ok {
		l.Agent = row.Agent
	}
	return l, p.thread
}

// anchorNote is the broadcast's AnchorNote: a note's anchor from memory
// as it opens. It runs on the request's goroutine.
func (c *Console) anchorNote(n broadcast.Note, cols, rows int) broadcast.NoteAnchor {
	c.mu.Lock()
	l, _ := c.noteLayoutLocked()
	c.mu.Unlock()
	c.anchors.mu.Lock()
	f := c.anchors.frame
	c.anchors.mu.Unlock()
	if f.Geometry != (terminal.Geometry{Cols: cols, Rows: rows}) {
		return broadcast.NoteAnchor{Kind: broadcast.AnchorScreen}
	}
	return AnchorFor(n.Box, f, l)
}

// queueAnchors takes a notes snapshot from the live session: each open
// note still waiting for its pane path is queued once, and the resolver
// started if it isn't running. It runs on the Run loop (applyNotes).
func (c *Console) queueAnchors(s *broadcast.Session, notes []broadcast.Note) {
	c.mu.Lock()
	l, slot := c.noteLayoutLocked()
	c.mu.Unlock()
	c.anchors.mu.Lock()
	defer c.anchors.mu.Unlock()
	open := make(map[int]bool, len(notes))
	for _, n := range notes {
		open[n.ID] = true
	}
	for id := range c.anchors.tried {
		if !open[id] {
			delete(c.anchors.tried, id)
		}
	}
	if c.anchors.rects == nil || slot == (couchcore.ThreadAddress{}) {
		return
	}
	for _, n := range notes {
		if n.Anchor == nil || n.Anchor.Kind != broadcast.AnchorScreen || c.anchors.tried[n.ID] ||
			len(n.Anchor.Path) == 0 || n.Anchor.Path[0] != "pane "+l.Slot {
			continue
		}
		if c.anchors.tried == nil {
			c.anchors.tried = map[int]bool{}
		}
		c.anchors.tried[n.ID] = true
		c.anchors.pending = append(c.anchors.pending, anchorJob{session: s, note: n, slot: slot, layout: l, epoch: c.anchors.frame.GeometryEpoch})
	}
	if len(c.anchors.pending) > 0 && !c.anchors.running {
		c.anchors.running = true
		c.GoTracked(c.resolveAnchors)
	}
}

// resolveAnchors is the resolver: one pass per batch of waiting notes, one
// pane query per slot in a pass, until none wait.
func (c *Console) resolveAnchors() {
	for {
		c.anchors.mu.Lock()
		jobs, rects := c.anchors.pending, c.anchors.rects
		c.anchors.pending = nil
		if len(jobs) == 0 || rects == nil {
			c.anchors.running = false
			c.anchors.mu.Unlock()
			return
		}
		c.anchors.mu.Unlock()
		panes := map[couchcore.ThreadAddress][]PaneRect{}
		for _, j := range jobs {
			ps, asked := panes[j.slot]
			if !asked {
				ctx, cancel := context.WithTimeout(c.lifetime, paneRectsTimeout)
				got, err := rects(ctx, j.slot)
				cancel()
				if err != nil {
					got = nil // the anchor stays as it opened
				}
				panes[j.slot], ps = got, got
			}
			if ps == nil {
				continue
			}
			c.anchors.mu.Lock()
			f := c.anchors.frame
			c.anchors.mu.Unlock()
			if f.GeometryEpoch != j.epoch || f.Geometry != j.note.On {
				continue // resized since: the panes no longer match the box
			}
			l := j.layout
			l.Panes = ps
			a := AnchorFor(j.note.Box, f, l)
			if a.Kind != broadcast.AnchorPane || !strings.HasPrefix(strings.Join(a.Path, "\x00"), strings.Join(j.note.Anchor.Path, "\x00")) {
				continue
			}
			a.Text = j.note.Anchor.Text // the text as it was when the note opened
			j.session.SetNoteAnchor(j.note.ID, a)
		}
	}
}
