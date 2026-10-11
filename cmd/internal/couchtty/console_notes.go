package couchtty

import (
	"sync"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// pointerNotes is the broadcast's sticky notes (#447) as the console last
// accepted them: the Presenter's overlay draws them and the status row
// counts them, from this one snapshot. mu is a leaf lock: it guards these
// fields and nothing else, and is never held across a call to the Presenter,
// the session or the console.
type pointerNotes struct {
	mu      sync.Mutex
	session *broadcast.Session
	notes   []broadcast.Note
	version uint64
}

// onNotes is the session reporting a change. It runs on the changing
// goroutine, which can be the Run loop itself (a click on 👆 turns pointing
// off, and the session reports the emptied board), so it hands the snapshot
// over without waiting; applyNotes' version check makes the order of
// deliveries irrelevant.
func (c *Console) onNotes(s *broadcast.Session, notes []broadcast.Note, version uint64) {
	go func() {
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.applyNotes(s, notes, version)
			return nil
		})
	}()
}

// applyNotes keeps a snapshot only from the live broadcast's session while
// pointing is on, and only if it is newer than the one held: two request
// goroutines can deliver out of order, and a report in flight when pointing
// turned off or the broadcast ended must not bring notes back.
func (c *Console) applyNotes(s *broadcast.Session, notes []broadcast.Note, version uint64) {
	c.mu.Lock()
	ok := c.bcast.phase == broadcastLive && c.bcast.session == s && c.pointer == pointerOn
	c.mu.Unlock()
	if !ok {
		return
	}
	c.pnotes.mu.Lock()
	if c.pnotes.session == s && version <= c.pnotes.version {
		c.pnotes.mu.Unlock()
		return
	}
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = s, notes, version
	c.pnotes.mu.Unlock()
	c.repaint()
}

// resyncNotes takes the session's current notes on a pointer-phase change.
// The session flips (and empties the board on off) before the console's
// phase changes, so this reads the cleared board, and its version drops any
// older snapshot still in flight.
func (c *Console) resyncNotes(s *broadcast.Session) {
	if s == nil {
		return
	}
	notes, version := s.Notes()
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = s, notes, version
	c.pnotes.mu.Unlock()
}

// forgetNotes drops the snapshot when the broadcast ends.
func (c *Console) forgetNotes() {
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = nil, nil, 0
	c.pnotes.mu.Unlock()
}

// drawnNotes is the snapshot the overlay and the status row read. It is
// empty while pointing is off only because every pointer-phase change
// resyncs it from the session, whose board a flip to off empties.
func (c *Console) drawnNotes() []broadcast.Note {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	return c.pnotes.notes
}

// noteOverlay draws the notes on a public frame. It runs on the Presenter
// goroutine.
func (c *Console) noteOverlay(f terminal.Frame, class terminal.FrameClass) terminal.Frame {
	if class != terminal.FramePublic {
		return f
	}
	return broadcast.DrawNotes(f, c.drawnNotes())
}
