package couchtty

import (
	"slices"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/mouseinput"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
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
	// The web cast's notes (#470), as the cast last delivered them. Each
	// note keeps one menu id while it is open (webMenuIDs, from webNext), so
	// a click on a drawn row acts on that note even if the list changed
	// since the draw. web is the menu's items (Text "name: text").
	webCast    *webcast.Cast
	webItems   []webItem
	web        []broadcast.Note
	webMenuIDs map[string]int
	webNext    int
	source     notesSource
	// drive is remote driving's control (#467) as the cast last delivered
	// it, and its menu rows.
	drive driveState
}

// webItem is one web note in the menu: its menu id, its note id and its
// plain text (what a click copies).
type webItem struct {
	menuID int
	noteID string
	text   string
}

// notesSource is the list the notes menu shows: the screen broadcast's
// sticky notes (#447) or the web cast's (#470). One menu, one at a time.
type notesSource uint8

const (
	screenNotes notesSource = iota
	webNotes
	// driveMenu is remote driving's menu (#467): the driver and who asks.
	driveMenu
)

// menuNotesLocked is the open menu's list. Callers hold pnotes.mu.
func (c *Console) menuNotesLocked() []broadcast.Note {
	switch c.pnotes.source {
	case webNotes:
		return c.pnotes.web
	case driveMenu:
		return c.pnotes.drive.items
	}
	return withAuthors(c.pnotes.notes)
}

// withAuthors is screen notes as the menu shows them: "name: text" when the
// viewer gave a name (pair#495), like the web cast's notes. The board keeps
// the plain text, which is what a click copies.
func withAuthors(notes []broadcast.Note) []broadcast.Note {
	out := make([]broadcast.Note, len(notes))
	for i, n := range notes {
		if n.Author != "" {
			n.Text = n.Author + ": " + n.Text
		}
		out[i] = n
	}
	return out
}

// screenHoverLocked is the hovered screen note, whose 💭 glows: none while
// the menu shows the web cast's notes. Callers hold pnotes.mu.
func (c *Console) screenHoverLocked() int {
	if c.pnotes.source == webNotes {
		return 0
	}
	return c.pnotes.hover
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
	list := c.menuNotesLocked()
	if len(list) == 0 {
		c.pnotes.menuOpen, c.pnotes.menu = false, NotesMenu{}
	}
	if !slices.ContainsFunc(list, func(n broadcast.Note) bool { return n.ID == c.pnotes.hover }) {
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
	d, ok := broadcast.NoteNextChange(c.pnotes.notes, c.now(), c.screenHoverLocked())
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

// forgetNotes drops the snapshot when the broadcast ends; the web cast's
// notes are its own.
func (c *Console) forgetNotes() {
	c.pnotes.mu.Lock()
	c.pnotes.session, c.pnotes.notes, c.pnotes.version = nil, nil, 0
	c.settleMenuLocked()
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
	c.keepNoteFrame(f)
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
	title := "Notes"
	if c.pnotes.source == driveMenu {
		title = "Remote driving"
	}
	c.pnotes.menu = RenderTitledMenu(title, c.menuNotesLocked(), f.Geometry.Cols, f.Geometry.Rows-f.ChromeRows+1)
	return c.pnotes.menu.Draw(f, c.pnotes.hover, c.drawStyleLocked())
}

// drawStyleLocked is the style to draw with now: the palette facts, the
// clock and the hovered note. Callers hold pnotes.mu.
func (c *Console) drawStyleLocked() broadcast.NoteStyle {
	style := c.pnotes.style
	style.Now, style.Hover = c.now(), c.screenHoverLocked()
	return style
}

// toggleNotesMenu is a click on a 💭N: the screen broadcast's or the web
// cast's (src). A click on the other one's 💭N switches the menu to it.
func (c *Console) toggleNotesMenu(src notesSource) {
	c.pnotes.mu.Lock()
	open := c.pnotes.menuOpen && c.pnotes.source == src
	if c.pnotes.source != src {
		c.pnotes.hover = 0
	}
	c.pnotes.source = src
	c.pnotes.menuOpen = !open && len(c.menuNotesLocked()) > 0
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

// noteTextLocked is the text of the held note id, or "" when it is gone.
// Callers hold pnotes.mu.
func (c *Console) noteTextLocked(id int) string {
	if c.pnotes.source == webNotes {
		if it, ok := c.webItemLocked(id); ok {
			return it.text
		}
		return ""
	}
	for _, n := range c.pnotes.notes {
		if n.ID == id {
			return n.Text
		}
	}
	return ""
}

// routeNotesMenu takes the mouse while it is over the open menu: motion
// hovers an item, a click copies it or closes the menu, a right click
// copies and resolves it, and nothing over the
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
	// A gesture that starts in the menu is the menu's until its release:
	// drag motion after a press there never reaches anything else.
	if motion && pressed {
		c.pnotes.mu.Unlock()
		return true
	}
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
	case c.pnotes.source == driveMenu && (hit.Button == 0 || hit.Button == 2):
		// Remote driving's rows act (#467): a left click on the driver
		// takes control back, on an asker lets them drive; a right click
		// turns an asker down (and takes control back from the driver too).
		c.pnotes.pressed = true
		closing := menu.CloseAt(x, y)
		act := c.driveActionLocked(id, hit.Button == 2)
		c.pnotes.mu.Unlock()
		switch {
		case closing:
			c.toggleNotesMenu(driveMenu)
		case act != nil:
			act()
		}
	case hit.Button == 0:
		c.pnotes.pressed = true
		text := c.noteTextLocked(id)
		closing := menu.CloseAt(x, y)
		c.pnotes.mu.Unlock()
		switch {
		case closing:
			c.toggleNotesMenu(c.pnotesSource())
		case text != "":
			c.copyLink(text)
			c.setNotice("Note copied.")
		}
	case hit.Button == 2:
		// A right click resolves the note, copying it first so resolving
		// never loses its text. The session reports the emptied slot back
		// through onNotes, like a viewer's Resolve.
		c.pnotes.pressed = true
		text, resolve := c.noteTextLocked(id), c.resolverLocked(id)
		c.pnotes.mu.Unlock()
		if text != "" && resolve != nil {
			c.copyLink(text)
			if err := resolve(); err != nil {
				c.setNotice("Couldn't resolve the note.")
			} else {
				c.setNotice("Note copied and resolved.")
			}
		}
	default: // wheel, other buttons: the menu's, not the child's
		c.pnotes.pressed = hit.Button < 64
		c.pnotes.mu.Unlock()
	}
	return true
}

func (c *Console) pnotesSource() notesSource {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	return c.pnotes.source
}

// resolverLocked resolves the menu's item id in its source: the screen
// broadcast's session, or the web cast's board as the operator. nil when
// there is nothing to resolve it in. Callers hold pnotes.mu.
func (c *Console) resolverLocked(id int) func() error {
	if c.pnotes.source == webNotes {
		cast := c.pnotes.webCast
		it, ok := c.webItemLocked(id)
		if cast == nil || !ok {
			return nil
		}
		return func() error { return cast.ResolveNote(it.noteID) }
	}
	s := c.pnotes.session
	if s == nil {
		return nil
	}
	return func() error { return s.ResolveNote(id) }
}

// webItemLocked is the web note drawn with menu id id, if it is still
// open. Callers hold pnotes.mu.
func (c *Console) webItemLocked(id int) (webItem, bool) {
	for _, it := range c.pnotes.webItems {
		if it.menuID == id {
			return it, true
		}
	}
	return webItem{}, false
}

// applyWebNotes takes the web cast's open notes (#470) for its 💭N and the
// menu; a delivery from a cast that has since ended is dropped. It runs on
// the terminal loop, as webcastEnded does, so the current-cast check and the
// write can't be overtaken by the cast ending.
func (c *Console) applyWebNotes(cast *webcast.Cast, live []board.LiveNote) {
	c.mu.Lock()
	current := c.wcast.cast == cast
	c.mu.Unlock()
	if !current {
		return
	}
	c.pnotes.mu.Lock()
	c.pnotes.webCast = cast
	c.pnotes.webItems, c.pnotes.web = webMenuNotes(live, c.webMenuIDLocked)
	for noteID, id := range c.pnotes.webMenuIDs {
		if _, open := c.webItemLocked(id); !open {
			delete(c.pnotes.webMenuIDs, noteID)
		}
	}
	c.settleMenuLocked()
	c.pnotes.mu.Unlock()
	c.repaint()
	c.terminalError(c.presenter.Refresh(c.lifetime))
}

// webMenuIDLocked is a note's menu id, assigned once while it is open;
// ids of notes no longer open are dropped by the next delivery (the map
// holds the open notes only). Callers hold pnotes.mu.
func (c *Console) webMenuIDLocked(noteID string) int {
	if id, ok := c.pnotes.webMenuIDs[noteID]; ok {
		return id
	}
	if c.pnotes.webMenuIDs == nil {
		c.pnotes.webMenuIDs = map[string]int{}
	}
	c.pnotes.webNext++
	c.pnotes.webMenuIDs[noteID] = c.pnotes.webNext
	return c.pnotes.webNext
}

// forgetWebNotes drops the web cast's notes when it ends.
func (c *Console) forgetWebNotes() {
	c.pnotes.mu.Lock()
	c.pnotes.webCast, c.pnotes.webItems, c.pnotes.web, c.pnotes.webMenuIDs = nil, nil, nil, nil
	c.settleMenuLocked()
	c.pnotes.mu.Unlock()
}

// webNoteCount is the 💭N the Chrome cell shows.
func (c *Console) webNoteCount() int {
	c.pnotes.mu.Lock()
	defer c.pnotes.mu.Unlock()
	return len(c.pnotes.web)
}

// webMenuNotes maps the web cast's notes onto the menu: each item's id is
// the note's stable menu id (menuID), its color the palette index (the same
// palette, #446), and its text shows the author; the items keep the note id
// and plain text. menuID is also asked to forget nothing: the caller prunes.
func webMenuNotes(live []board.LiveNote, menuID func(noteID string) int) ([]webItem, []broadcast.Note) {
	var items []webItem
	var notes []broadcast.Note
	for _, n := range live {
		color := n.Color
		if color < 0 || color >= broadcast.PointerColors {
			color = 0
		}
		text := n.Text
		if n.Author.Name != "" {
			text = n.Author.Name + ": " + n.Text
		}
		id := menuID(n.ID)
		items = append(items, webItem{menuID: id, noteID: n.ID, text: n.Text})
		notes = append(notes, broadcast.Note{ID: id, Color: color, Text: text})
	}
	return items, notes
}
