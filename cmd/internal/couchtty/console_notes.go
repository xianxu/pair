package couchtty

import (
	"slices"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
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
	// The notes menu (M4). menuOpen is the operator's choice; menu is its
	// layout on the frame now on screen, the only thing the mouse is
	// hit-tested against, and zero whenever that frame doesn't show it (a
	// private panel), so the menu never takes a click it isn't drawn under.
	// hover is view state, kept apart from style, which is replaced
	// wholesale; pressed is a press inside the menu awaiting its release.
	menuOpen bool
	menu     NotesMenu
	hover    int
	pressed  bool
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
	c.settleMenuLocked()
	c.pnotes.mu.Unlock()
	c.repaint()
	c.scheduleNotes()
}

// settleMenuLocked keeps the menu consistent with the notes: closed with
// none left, and no hover on a note that is gone. Callers hold pnotes.mu.
func (c *Console) settleMenuLocked() {
	if len(c.pnotes.notes) == 0 {
		c.pnotes.menuOpen, c.pnotes.menu = false, NotesMenu{}
	}
	if !slices.ContainsFunc(c.pnotes.notes, func(n broadcast.Note) bool { return n.ID == c.pnotes.hover }) {
		c.pnotes.hover = 0
	}
}

// noteStyleLocked is the background the chip and glow blend into, as marks
// fade into it: the operator's real background when the terminal reported
// it and has truecolor. Callers hold c.mu.
func (c *Console) noteStyleLocked() broadcast.NoteStyle {
	return broadcast.NoteStyle{BG: c.menu.Palette.BG, Blend: blendable(c.menu.Palette)}
}

// blendable reports whether marks and notes can fade into the operator's
// real background: the terminal reported it (OSC 11) and has truecolor.
func blendable(p Palette) bool { return p.Known && p.TrueColor && !p.NoColor }

// restyleNotes takes a new palette (an OSC 10/11 reply, a theme change) into
// the notes' style, as marks take it on their next batch. The overlay can't
// read c.mu itself: it runs on the Presenter goroutine, which callers holding
// c.mu may wait on.
func (c *Console) restyleNotes() {
	c.mu.Lock()
	style := c.noteStyleLocked()
	c.mu.Unlock()
	c.pnotes.mu.Lock()
	c.pnotes.style = style
	c.pnotes.mu.Unlock()
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
	d, ok := broadcast.NoteNextChange(c.pnotes.notes, c.now(), c.pnotes.hover)
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
	c.settleMenuLocked()
	c.pnotes.mu.Unlock()
	c.scheduleNotes()
}

// forgetNotes drops the snapshot when the broadcast ends.
func (c *Console) forgetNotes() {
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = nil, nil, 0
	c.pnotes.menuOpen, c.pnotes.menu, c.pnotes.pressed, c.pnotes.hover = false, NotesMenu{}, false, 0
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
	defer c.pnotes.mu.Unlock()
	return broadcast.DrawNotes(f, c.pnotes.notes, c.drawStyleLocked())
}

// menuOverlay draws the open notes menu on a public frame, over everything
// else, and records its layout on that frame for hit-testing. On any other
// frame the layout is cleared: the menu is not on screen, so it takes no
// clicks. It runs on the Presenter goroutine.
func (c *Console) menuOverlay(f terminal.Frame, class terminal.FrameClass) terminal.Frame {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	if class != terminal.FramePublic || !c.pnotes.menuOpen {
		c.pnotes.menu = NotesMenu{}
		return f
	}
	c.pnotes.menu = RenderNotesMenu(c.pnotes.notes, f.Geometry.Cols, f.Geometry.Rows)
	return c.pnotes.menu.Draw(f, c.pnotes.hover, c.drawStyleLocked())
}

// drawStyleLocked is the style to draw with now: the palette facts, the
// clock and the hovered note. Callers hold pnotes.mu.
func (c *Console) drawStyleLocked() broadcast.NoteStyle {
	style := c.pnotes.style
	style.Now, style.Hover = c.now(), c.pnotes.hover
	return style
}

// toggleNotesMenu is a click on 💭N.
func (c *Console) toggleNotesMenu() {
	c.pnotes.mu.Lock()
	c.pnotes.menuOpen = !c.pnotes.menuOpen && len(c.pnotes.notes) > 0
	if !c.pnotes.menuOpen {
		c.pnotes.menu, c.pnotes.hover = NotesMenu{}, 0
	}
	c.pnotes.mu.Unlock()
	c.refreshNotes()
}

// refreshNotes repaints the overlay and re-plans the flash timer (a hover
// changes which note is steady).
func (c *Console) refreshNotes() {
	c.terminalError(c.presenter.Refresh(c.lifetime))
	c.scheduleNotes()
}

// routeNotesMenu takes the mouse while it is over the open menu: motion
// hovers an item, a click copies it or closes the menu, and nothing over the
// menu reaches the child -- a press's release included. It reports whether
// it took the event. Motion outside the menu clears the hover and goes on.
func (c *Console) routeNotesMenu(raw []byte) bool {
	hit, _, _, ok := mouseinput.ParsePrefix(raw)
	if !ok {
		return false
	}
	x, y := hit.X-1, hit.Y-1
	c.pnotes.mu.Lock()
	open, menu, pressed := c.pnotes.menuOpen, c.pnotes.menu, c.pnotes.pressed
	if hit.Release {
		c.pnotes.pressed = false
		c.pnotes.mu.Unlock()
		return pressed
	}
	if !open {
		c.pnotes.mu.Unlock()
		return false
	}
	motion := hit.Button&32 != 0
	if !menu.Contains(x, y) {
		changed := motion && c.pnotes.hover != 0
		if changed {
			c.pnotes.hover = 0
		}
		c.pnotes.mu.Unlock()
		if changed {
			c.refreshNotes()
		}
		return false
	}
	id := menu.ItemAt(x, y)
	switch {
	case motion:
		changed := c.pnotes.hover != id
		c.pnotes.hover = id
		c.pnotes.mu.Unlock()
		if changed {
			c.refreshNotes()
		}
	case hit.Button == 0:
		c.pnotes.pressed = true
		text := ""
		for _, n := range c.pnotes.notes {
			if n.ID == id {
				text = n.Text
			}
		}
		closing := menu.CloseAt(x, y)
		c.pnotes.mu.Unlock()
		switch {
		case closing:
			c.toggleNotesMenu()
		case text != "":
			c.copyLink(text)
			c.setNotice("Note copied.")
		}
	default: // wheel, other buttons: the menu's, not the child's
		c.pnotes.pressed = hit.Button < 64
		c.pnotes.mu.Unlock()
	}
	return true
}
