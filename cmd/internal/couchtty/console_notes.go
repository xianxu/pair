package couchtty

import (
	"sync"
	"time"

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
	// style is the operator's background facts for the chip and glow (M3),
	// taken when a snapshot is accepted; timer repaints the next flash step.
	style broadcast.NoteStyle
	timer *time.Timer
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
	style := c.noteStyleLocked()
	c.mu.Unlock()
	if !ok {
		return
	}
	c.pnotes.mu.Lock()
	if c.pnotes.session == s && version <= c.pnotes.version {
		c.pnotes.mu.Unlock()
		return
	}
	c.pnotes.session, c.pnotes.notes, c.pnotes.version, c.pnotes.style = s, notes, version, style
	c.pnotes.mu.Unlock()
	c.repaint()
	c.scheduleNotes()
}

// noteStyleLocked is the background the chip and glow blend into, as marks
// fade into it: the operator's real background when the terminal reported
// it and has truecolor. Callers hold c.mu.
func (c *Console) noteStyleLocked() broadcast.NoteStyle {
	p := c.menu.Palette
	return broadcast.NoteStyle{BG: p.BG, Blend: p.Known && p.TrueColor && !p.NoColor}
}

// scheduleNotes arms one timer for the notes' next flash step, replacing
// any earlier one; with no notes there is none. The timer repaints through
// the overlay and schedules the next.
func (c *Console) scheduleNotes() {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	if c.pnotes.timer != nil {
		c.pnotes.timer.Stop()
		c.pnotes.timer = nil
	}
	d, ok := broadcast.NoteNextChange(c.pnotes.notes, c.now(), c.pnotes.style.Hover)
	if !ok {
		return
	}
	c.pnotes.timer = time.AfterFunc(d, func() {
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.terminalError(c.presenter.Refresh(c.lifetime))
			c.scheduleNotes()
			return nil
		})
	})
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
	c.mu.Lock()
	style := c.noteStyleLocked()
	c.mu.Unlock()
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version, c.pnotes.style = s, notes, version, style
	c.pnotes.mu.Unlock()
	c.scheduleNotes()
}

// forgetNotes drops the snapshot when the broadcast ends.
func (c *Console) forgetNotes() {
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = nil, nil, 0
	if c.pnotes.timer != nil {
		c.pnotes.timer.Stop()
		c.pnotes.timer = nil
	}
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
	c.pnotes.mu.Lock()
	notes, style := c.pnotes.notes, c.pnotes.style
	c.pnotes.mu.Unlock()
	style.Now = c.now()
	return broadcast.DrawNotes(f, notes, style)
}
