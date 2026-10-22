package couchtty

import (
	"sync"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// Notes use only Couch's public frame and its own screen layout.
// Hosted native pane/tab provenance is unavailable, so no later query refines it.
type noteAnchors struct {
	mu    sync.Mutex
	frame terminal.Frame
}

// keepNoteFrame keeps the public frame notes are placed on. It runs on the
// Presenter goroutine.
func (c *Console) keepNoteFrame(f terminal.Frame) {
	c.anchors.mu.Lock()
	c.anchors.frame = f
	c.anchors.mu.Unlock()
}

// noteLayoutLocked is the screen's layout as Couch drew it last: the tab
// bar's chips and the slot in view. Callers hold c.mu; no IO.
func (c *Console) noteLayoutLocked(f terminal.Frame) ScreenLayout {
	var l ScreenLayout
	for _, ch := range c.statusChips {
		l.Chips = append(l.Chips, ch)
	}
	var p *pane
	for _, candidate := range c.panes {
		if candidate.child.Endpoint().ID() == f.EndpointID {
			p = candidate
			break
		}
	}
	if p == nil {
		return l
	}
	for _, a := range c.statusModelLocked().Actors {
		if a.Thread == p.thread {
			l.Slot = a.Label
		}
	}
	return l
}

// anchorNote is the broadcast's AnchorNote: a note's anchor from memory
// as it opens. It runs on the request's goroutine.
func (c *Console) anchorNote(n broadcast.Note, cols, rows int) broadcast.NoteAnchor {
	// Selection and chrome hold this existing publication owner through both
	// presenter publication and hit-test metadata commit. Do not pair a new
	// frame with the prior publication's chips or active slot.
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	if c.presenter == nil {
		return broadcast.NoteAnchor{Kind: broadcast.AnchorScreen}
	}
	if _, ok := c.confirmedChromeStatePublishing(); !ok {
		return broadcast.NoteAnchor{Kind: broadcast.AnchorScreen}
	}
	c.anchors.mu.Lock()
	f := c.anchors.frame
	c.anchors.mu.Unlock()
	c.mu.Lock()
	l := c.noteLayoutLocked(f)
	c.mu.Unlock()
	if f.Geometry != (terminal.Geometry{Cols: cols, Rows: rows}) {
		return broadcast.NoteAnchor{Kind: broadcast.AnchorScreen}
	}
	return AnchorFor(n.Box, f, l)
}
