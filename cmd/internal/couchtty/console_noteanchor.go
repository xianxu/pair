package couchtty

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
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
// Ordering (ARCH-ORDER). A job carries the exact observation it was made
// from: the session, the note as it opened, the slot in view and the frame
// then, whose text under the box must still be the note's (else the screen
// moved between the note opening and the queueing, and the job is not
// made). A resolved anchor applies only to that note of that session: the
// session drops it once the note is resolved or pointing turned off (a
// session never reuses an id; a new broadcast is a new session, so tried
// is keyed by both). It is dropped here when the screen in view is no
// longer that frame's endpoint and geometry (a slot switch or a resize
// mid-pass: the panes no longer match the box). A note opened during a
// pass waits for the next pass. A failed or timed out query leaves the
// anchor as it opened, kind AnchorScreen; nothing is retried.

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
	// once; it keeps only the latest session's open notes (pruned on
	// every snapshot) and nothing once the broadcast ends (forgetAnchors).
	tried map[triedNote]bool
}

// triedNote is one note of one session.
type triedNote struct {
	session anchorSetter
	id      int
}

// anchorJob is one note waiting for its pane path: the note as it opened,
// the slot in view and the layout then, and the frame's geometry epoch.
type anchorJob struct {
	session anchorSetter
	note    broadcast.Note
	slot    couchcore.ThreadAddress
	layout  ScreenLayout
	frame   terminal.Frame
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
	if c.queueAnchorJobs(s, notes, l, slot) {
		c.GoTracked(c.resolveAnchors)
	}
}

// queueAnchorJobs queues each of the session's open notes still waiting
// for a pane path, once, against the frame in view, and reports whether
// the caller must start the resolver (it wasn't running). It takes no lock
// but anchors.mu, and starts nothing itself.
func (c *Console) queueAnchorJobs(s anchorSetter, notes []broadcast.Note, l ScreenLayout, slot couchcore.ThreadAddress) bool {
	c.anchors.mu.Lock()
	defer c.anchors.mu.Unlock()
	open := make(map[triedNote]bool, len(notes))
	for _, n := range notes {
		open[triedNote{s, n.ID}] = true
	}
	for k := range c.anchors.tried {
		if !open[k] {
			delete(c.anchors.tried, k)
		}
	}
	if c.anchors.rects == nil || slot == (couchcore.ThreadAddress{}) {
		return false
	}
	f := c.anchors.frame
	for _, n := range notes {
		k := triedNote{s, n.ID}
		if n.Anchor == nil || n.Anchor.Kind != broadcast.AnchorScreen || c.anchors.tried[k] ||
			len(n.Anchor.Path) == 0 || n.Anchor.Path[0] != "pane "+l.Slot ||
			f.Geometry != n.On || TextUnder(f, n.Box) != n.Anchor.Text {
			continue
		}
		if c.anchors.tried == nil {
			c.anchors.tried = map[triedNote]bool{}
		}
		c.anchors.tried[k] = true
		c.anchors.pending = append(c.anchors.pending, anchorJob{session: s, note: n, slot: slot, layout: l, frame: f})
	}
	if len(c.anchors.pending) == 0 || c.anchors.running {
		return false
	}
	c.anchors.running = true
	return true
}

// resolveAnchors is the resolver: one pass per batch of waiting notes, one
// pane observation per slot in a pass, until none wait.
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
		c.resolveAnchorPass(jobs, rects)
	}
}

// resolveAnchorPass owns each slot's admission token through publication. A
// successful query is evidence, not permission to publish after cancellation.
func (c *Console) resolveAnchorPass(jobs []anchorJob, rects PaneRects) {
	contexts := map[couchcore.ThreadAddress]context.Context{}
	var cancellations []context.CancelFunc
	defer func() {
		for _, cancel := range cancellations {
			cancel()
		}
	}()
	panes := map[couchcore.ThreadAddress][]PaneRect{}
	refused := map[couchcore.ThreadAddress]bool{}
	refuse := func(slot couchcore.ThreadAddress, reason string) {
		if !refused[slot] {
			c.setNotice("Note anchor unchanged: " + reason)
			refused[slot] = true
		}
	}
	for _, j := range jobs {
		ps, asked := panes[j.slot]
		if !asked {
			ctx, cancel := context.WithTimeout(c.lifetime, paneRectsTimeout)
			contexts[j.slot] = ctx
			cancellations = append(cancellations, cancel)
			got, err := rects(ctx, j.slot)
			if err == nil {
				err = ctx.Err()
			}
			if err != nil {
				got = nil // the anchor stays as it opened
				refuse(j.slot, err.Error())
			}
			panes[j.slot], ps = got, got
		}
		if ps == nil {
			continue
		}
		ctx := contexts[j.slot]
		c.anchors.mu.Lock()
		now := c.anchors.frame
		c.anchors.mu.Unlock()
		if now.EndpointID != j.frame.EndpointID || now.GeometryEpoch != j.frame.GeometryEpoch || now.Geometry != j.frame.Geometry || TextUnder(now, j.note.Box) != j.note.Anchor.Text {
			refuse(j.slot, "the observed screen context changed")
			continue // another slot in view, or resized: the panes no longer match the box
		}
		l := j.layout
		l.Panes = ps
		a := AnchorFor(j.note.Box, j.frame, l)
		if a.Kind != broadcast.AnchorPane || !strings.HasPrefix(strings.Join(a.Path, "\x00"), strings.Join(j.note.Anchor.Path, "\x00")) {
			refuse(j.slot, "the pane context could not be confirmed")
			continue
		}
		a.Text = j.note.Anchor.Text // the text as it was when the note opened
		if err := zellijreply.Effect(ctx, func() error {
			j.session.SetNoteAnchor(j.note.ID, a)
			return nil
		}); err != nil {
			refuse(j.slot, err.Error())
		}
	}
}
