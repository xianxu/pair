package couchtty

import (
	"fmt"
	"image/color"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func postNote(t *testing.T, link string, col, row int, text string) (int, string) {
	t.Helper()
	body := fmt.Sprintf(`{"cols":80,"rows":24,"box":[%d,%d,2,1],"color":0,"text":%q}`, col, row, text)
	resp, err := castClient.Post(link+"note", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, string(buf[:n])
}

// noteUntilSaved posts a note until the session takes it: like a point, a
// note is refused until the hub has seen a frame with the active 👆.
func noteUntilSaved(t *testing.T, link string, col, row int, text string) {
	t.Helper()
	waitFor(t, "note saved", func() bool {
		code, _ := postNote(t, link, col, row, text)
		return code == http.StatusNoContent
	})
}

func (f *consoleFixture) cellText(x, y int) string {
	f.screen.mu.Lock()
	defer f.screen.mu.Unlock()
	if c := f.screen.em.CellAt(x, y); c != nil {
		return c.Content
	}
	return ""
}

func (f *consoleFixture) heldNotes() ([]broadcast.Note, uint64) {
	f.con.pnotes.mu.Lock()
	defer f.con.pnotes.mu.Unlock()
	return f.con.pnotes.notes, f.con.pnotes.version
}

func TestNoteDrawnOnOperatorAndViewers(t *testing.T) {
	f, s, link := pointerOnFixture(t, nil)
	v := openViewer(t, s)
	noteUntilSaved(t, link, 10, 5, "look here")
	waitFor(t, "💭 on the operator's screen", func() bool { return f.cellText(10, 5) == broadcast.NoteGlyph })
	waitFor(t, "💭1 in the tab bar", func() bool { return strings.Contains(f.screenText(), broadcast.NoteGlyph+"1") })
	v.until("💭 in the broadcast", func(string) bool {
		c := v.emu.CellAt(10, 5)
		return c != nil && c.Content == broadcast.NoteGlyph
	})
	notes, _ := s.Notes()
	resp, err := castClient.Post(link+"resolve", "application/json", strings.NewReader(fmt.Sprintf(`{"id":%d}`, notes[0].ID)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	waitFor(t, "💭 gone", func() bool { return f.cellText(10, 5) != broadcast.NoteGlyph })
	waitFor(t, "count gone", func() bool { return !strings.Contains(f.screenText(), broadcast.NoteGlyph) })
}

// Turning pointing off deletes every note on the operator's side too, and
// turning it back on shows none.
func TestNotesGoWhenPointingTurnsOff(t *testing.T) {
	f, s, link := pointerOnFixture(t, nil)
	noteUntilSaved(t, link, 10, 5, "a")
	waitFor(t, "💭 drawn", func() bool { return f.cellText(10, 5) == broadcast.NoteGlyph })
	_, stale := s.Notes()
	staleNotes, _ := s.Notes()
	f.clickPointer(t, 0)
	waitFor(t, "pointing off", func() bool { return f.pointerPhase() == pointerOff })
	waitFor(t, "💭 gone", func() bool { return f.cellText(10, 5) != broadcast.NoteGlyph })
	if notes, _ := f.heldNotes(); len(notes) != 0 {
		t.Fatalf("console still holds %d notes", len(notes))
	}
	// A report in flight from before the off arrives late: dropped.
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error { f.con.applyNotes(s, staleNotes, stale); return nil })
	f.clickPointer(t, 0)
	waitFor(t, "pointing on again", func() bool { return f.pointerPhase() == pointerOn })
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error { f.con.applyNotes(s, staleNotes, stale); return nil })
	if notes, _ := f.heldNotes(); len(notes) != 0 {
		t.Fatal("a pre-off snapshot brought notes back after re-enable")
	}
	if strings.Contains(f.screenText(), broadcast.NoteGlyph) {
		t.Fatal("💭 drawn after re-enable")
	}
}

func TestNotesNewestVersionWins(t *testing.T) {
	f, s, _ := pointerOnFixture(t, nil)
	newer := []broadcast.Note{{ID: 2, Col: 4, Row: 4, Text: "new"}}
	older := []broadcast.Note{{ID: 1, Col: 4, Row: 4, Text: "old"}}
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(s, newer, 10)
		f.con.applyNotes(s, older, 9)
		return nil
	})
	if notes, v := f.heldNotes(); v != 10 || len(notes) != 1 || notes[0].Text != "new" {
		t.Fatalf("held %+v v%d", notes, v)
	}
}

// Stopping the broadcast forgets notes the console holds: no 💭, no count.
func TestNotesForgottenWhenBroadcastEnds(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	noteUntilSaved(t, link, 10, 5, "a")
	waitFor(t, "count drawn", func() bool { return strings.Contains(f.screenText(), broadcast.NoteGlyph+"1") })
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	if notes, v := f.heldNotes(); len(notes) != 0 || v != 0 {
		t.Fatalf("console still holds %d notes (v%d)", len(notes), v)
	}
	waitFor(t, "💭 gone", func() bool { return !strings.Contains(f.screenText(), broadcast.NoteGlyph) })
}

func TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession(t *testing.T) {
	f, s, _ := pointerOnFixture(t, nil)
	other := &broadcast.Session{}
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(other, []broadcast.Note{{ID: 1, Col: 4, Row: 4, Text: "x"}}, 99)
		return nil
	})
	if notes, _ := f.heldNotes(); len(notes) != 0 {
		t.Fatal("a snapshot from another session was kept")
	}
	_, _ = f.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(s, []broadcast.Note{{ID: 1, Col: 4, Row: 4, Text: "x"}}, 99)
		return nil
	})
	if notes, _ := f.heldNotes(); len(notes) != 0 {
		t.Fatal("a snapshot after the broadcast ended was kept")
	}
}

func TestNoteOverlayOnlyOnPublicFrames(t *testing.T) {
	f, s, _ := pointerOnFixture(t, nil)
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(s, []broadcast.Note{{ID: 1, Col: 4, Row: 1, Text: "x"}}, 99)
		return nil
	})
	cells, err := terminal.StyledRows("hello world", 20, 3)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := terminal.PanelFrame(terminal.Geometry{Cols: 20, Rows: 3}, cells, terminal.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	frame.ChromeRows = 1
	if got := f.con.screenOverlay(frame, terminal.FramePublic); got.Cells[1*20+4].Content != broadcast.NoteGlyph {
		t.Fatal("no 💭 on a public frame")
	}
	if got := f.con.screenOverlay(frame, terminal.FramePrivate); got.Cells[1*20+4].Content == broadcast.NoteGlyph {
		t.Fatal("💭 drawn on a private frame")
	}
}

func (f *consoleFixture) notesTimer() bool {
	f.con.pnotes.mu.Lock()
	defer f.con.pnotes.mu.Unlock()
	return f.con.pnotes.timer != nil
}

// settle waits for the Run loop to finish the step in progress: commands run
// one at a time between input chunks, so a no-op one returns only after it.
func (f *consoleFixture) settle(t *testing.T) {
	t.Helper()
	if err := f.con.runTerminalCommand(f.con.lifetime, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

// M3: the 💭 sits on a chip of its author's color (color 0 here, the amber
// marks use), and a flash timer runs while notes are open, stopped when
// pointing turns off.
func TestNoteChipAndFlashTimer(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	noteUntilSaved(t, link, 10, 5, "a")
	waitFor(t, "chip drawn", func() bool { return f.screenBg(10, 5) == markTint })
	waitFor(t, "glow beside the chip", func() bool { return f.screenBg(9, 5) != nil })
	waitFor(t, "flash timer armed", f.notesTimer)
	f.clickPointer(t, 0)
	waitFor(t, "pointing off", func() bool { return f.pointerPhase() == pointerOff })
	waitFor(t, "timer stopped", func() bool { return !f.notesTimer() })
	waitFor(t, "chip gone", func() bool { return f.screenBg(10, 5) != markTint })
}

// Every timer the console arms for the collaboration layer re-arms after it
// fires and is torn down on each way the layer ends (#447 M3 review, BR-14):
// the marks' fade timer and the notes' flash timer, on pointer-off and on
// broadcast end.
func TestCollaborationTimersRearmAndTearDown(t *testing.T) {
	marksTimer := func(f *consoleFixture) *time.Timer {
		f.con.pmarks.mu.Lock()
		defer f.con.pmarks.mu.Unlock()
		return f.con.pmarks.timer
	}
	notesTimer := func(f *consoleFixture) *time.Timer {
		f.con.pnotes.mu.Lock()
		defer f.con.pnotes.mu.Unlock()
		return f.con.pnotes.timer
	}
	timers := map[string]struct {
		arm   func(*testing.T, *consoleFixture, string)
		timer func(*consoleFixture) *time.Timer
	}{
		"marks": {func(t *testing.T, f *consoleFixture, link string) { tapUntilMarked(t, f, link, 5, 3) }, marksTimer},
		"notes": {func(t *testing.T, f *consoleFixture, link string) { noteUntilSaved(t, link, 10, 5, "a") }, notesTimer},
	}
	ends := map[string]func(*testing.T, *consoleFixture){
		"pointer off": func(t *testing.T, f *consoleFixture) {
			f.clickPointer(t, 0)
			waitFor(t, "pointing off", func() bool { return f.pointerPhase() == pointerOff })
		},
		"broadcast end": func(t *testing.T, f *consoleFixture) {
			_, _ = f.stdin.Write([]byte(ctrlAltB))
			waitFor(t, "broadcast off", func() bool { return f.phase() == broadcastOff })
		},
	}
	for tname, tm := range timers {
		t.Run(tname+"/re-arms after firing", func(t *testing.T) {
			f, _, link := pointerOnFixture(t, nil)
			tm.arm(t, f, link)
			var first *time.Timer
			waitFor(t, "timer armed", func() bool { first = tm.timer(f); return first != nil })
			// The first step comes at the end of the hold (NoteFlashHold); the timer
			// that fires arms the next one.
			waitFor(t, "timer re-armed", func() bool { next := tm.timer(f); return next != nil && next != first })
		})
		for ename, end := range ends {
			t.Run(tname+"/"+ename, func(t *testing.T) {
				f, _, link := pointerOnFixture(t, nil)
				tm.arm(t, f, link)
				waitFor(t, "timer armed", func() bool { return tm.timer(f) != nil })
				end(t, f)
				// The phase flips before the same loop step tears the layer
				// down, so let that step finish (#473). Then at once, not
				// eventually: a stale timer would clear itself when it next
				// fires, which a wait would mistake for teardown.
				f.settle(t)
				if tm.timer(f) != nil {
					t.Fatal("timer still armed after the layer ended")
				}
			})
		}
	}
}

// A palette reply after a note is up restyles it at once (M3 review): the
// blend background isn't frozen at the last note change.
func TestNotesTakeANewPalette(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	noteUntilSaved(t, link, 10, 5, "a")
	waitFor(t, "note held", func() bool { n, _ := f.heldNotes(); return len(n) == 1 })
	f.con.mu.Lock()
	f.con.menu.Palette.TrueColor, f.con.menu.Palette.Known = true, true
	f.con.menu.Palette.BG = color.RGBA{0x20, 0x20, 0x20, 0xff}
	f.con.mu.Unlock()
	f.con.restyleNotes()
	f.con.pnotes.mu.Lock()
	style := f.con.pnotes.style
	f.con.pnotes.mu.Unlock()
	if !style.Blend || style.BG != (color.RGBA{0x20, 0x20, 0x20, 0xff}) {
		t.Fatalf("style not refreshed: %+v", style)
	}
}

func (f *consoleFixture) notesSpan() ColumnSpan {
	f.con.mu.Lock()
	defer f.con.mu.Unlock()
	return f.con.statusNotes
}

func (f *consoleFixture) notesMenu() (bool, NotesMenu, int) {
	f.con.pnotes.mu.Lock()
	defer f.con.pnotes.mu.Unlock()
	return f.con.pnotes.menuOpen, f.con.pnotes.menu, f.con.pnotes.hover
}

func (f *consoleFixture) mouse(button, x, y int, release bool) {
	end := "M"
	if release {
		end = "m"
	}
	// SGR reports are one-based.
	_, _ = fmt.Fprintf(f.stdin, "\x1b[<%d;%d;%d%s", button, x+1, y+1, end)
}

func (f *consoleFixture) childGot(report string) bool {
	for _, w := range f.child.Writes() {
		if strings.Contains(string(w), report) {
			return true
		}
	}
	return false
}

// M4: 💭N opens the notes menu; hovering an item highlights it and holds its
// 💭 at full glow; a click copies its text; nothing over the menu reaches a
// child that tracks the mouse, while a click elsewhere still does.
func TestNotesMenuOpenHoverCopy(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	f.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child tracks the mouse", func() bool { return f.child.Endpoint().Modes().Tracking != 0 })
	noteUntilSaved(t, link, 10, 5, "copy me please")
	var span ColumnSpan
	waitFor(t, "💭1 drawn", func() bool { span = f.notesSpan(); return span != (ColumnSpan{}) })
	f.clickStatus(span.Start + 1)
	var m NotesMenu
	waitFor(t, "menu open and drawn", func() bool { open, mm, _ := f.notesMenu(); m = mm; return open && m.Open() })
	waitFor(t, "item text on screen", func() bool { return strings.Contains(f.screenText(), "copy me please") })
	item := m.items[0]
	// Hover.
	f.mouse(35, m.X+8, item.Top, false)
	waitFor(t, "hovered", func() bool { _, _, h := f.notesMenu(); return h != 0 })
	// Click copies the full text.
	f.mouse(0, m.X+8, item.Top, false)
	f.mouse(0, m.X+8, item.Top, true)
	waitFor(t, "copied", func() bool { return copies(f, "copy me please") == 1 })
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "Note copied") })
	for _, r := range []string{fmt.Sprintf(";%d;%dM", m.X+9, item.Top+1), fmt.Sprintf(";%d;%dm", m.X+9, item.Top+1)} {
		if f.childGot(r) {
			t.Fatalf("a report over the menu reached the child: %q", r)
		}
	}
	// Control: a click outside the menu still reaches the child.
	f.mouse(0, 2, 2, false)
	f.mouse(0, 2, 2, true)
	waitFor(t, "a click elsewhere reaches the child", func() bool { return f.childGot("\x1b[<0;3;3M") })
	// Motion off the items clears the hover.
	f.mouse(35, 2, 2, false)
	waitFor(t, "hover cleared", func() bool { _, _, h := f.notesMenu(); return h == 0 })
	// The close mark closes it.
	f.mouse(0, m.X+m.W-1, m.Y, false)
	f.mouse(0, m.X+m.W-1, m.Y, true)
	waitFor(t, "menu closed", func() bool { open, _, _ := f.notesMenu(); return !open })
}

// openMenuWith saves a note, waits until 💭1 is drawn for it, and opens the
// menu with a click, as the operator would.
func openMenuWith(t *testing.T, f *consoleFixture, link string, col int) {
	t.Helper()
	noteUntilSaved(t, link, col, 5, "a")
	var span ColumnSpan
	waitFor(t, "💭1 drawn", func() bool {
		held, _ := f.heldNotes()
		span = f.notesSpan()
		return len(held) == 1 && span != (ColumnSpan{}) && strings.Contains(f.lastRow(), broadcast.NoteGlyph+"1")
	})
	f.clickStatus(span.Start + 1)
	waitFor(t, "menu open and drawn", func() bool { open, m, _ := f.notesMenu(); return open && m.Open() })
}

func TestNotesMenuToggles(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	openMenuWith(t, f, link, 10)
	f.clickStatus(f.notesSpan().Start + 1)
	waitFor(t, "closed by a second click", func() bool { open, m, _ := f.notesMenu(); return !open && !m.Open() })
}

// A right click on an item copies it and resolves it, for every viewer too;
// it never reaches the child.
func TestNotesMenuRightClickResolves(t *testing.T) {
	f, s, link := pointerOnFixture(t, nil)
	f.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child tracks the mouse", func() bool { return f.child.Endpoint().Modes().Tracking != 0 })
	openMenuWith(t, f, link, 10)
	_, m, _ := f.notesMenu()
	item := m.items[0]
	text, _ := s.Notes()
	f.mouse(2, m.X+8, item.Top, false)
	f.mouse(2, m.X+8, item.Top, true)
	waitFor(t, "copied", func() bool { return copies(f, text[0].Text) == 1 })
	waitFor(t, "resolved on the session", func() bool { notes, _ := s.Notes(); return len(notes) == 0 })
	waitFor(t, "notice", func() bool { return strings.Contains(f.notice(), "Note copied and resolved") })
	waitFor(t, "menu closed with no notes left", func() bool { open, mm, _ := f.notesMenu(); return !open && !mm.Open() })
	if f.childGot(fmt.Sprintf("\x1b[<2;%d;%d", m.X+9, item.Top+1)) {
		t.Fatal("the right click reached the child")
	}
}

// Every way the menu's world ends closes it, layout included (M4 review).
func TestNotesMenuLifecycle(t *testing.T) {
	ends := map[string]func(*testing.T, *consoleFixture, *broadcast.Session, string){
		"last note resolved": func(t *testing.T, f *consoleFixture, s *broadcast.Session, link string) {
			notes, _ := s.Notes()
			resp, err := castClient.Post(link+"resolve", "application/json", strings.NewReader(fmt.Sprintf(`{"id":%d}`, notes[0].ID)))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
		},
		"pointer off": func(t *testing.T, f *consoleFixture, _ *broadcast.Session, _ string) { f.clickPointer(t, 0) },
		"broadcast end": func(t *testing.T, f *consoleFixture, _ *broadcast.Session, _ string) {
			_, _ = f.stdin.Write([]byte(ctrlAltB))
		},
	}
	for name, end := range ends {
		t.Run(name, func(t *testing.T) {
			f, s, link := pointerOnFixture(t, nil)
			openMenuWith(t, f, link, 10)
			end(t, f, s, link)
			waitFor(t, "menu closed", func() bool { open, m, h := f.notesMenu(); return !open && !m.Open() && h == 0 })
		})
	}
}

// Under a private panel the menu isn't drawn, so it takes no clicks: a click
// where it was must not copy a note nobody can see (M4 review, Critical).
func TestNotesMenuTakesNoClickUnderAPanel(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	openMenuWith(t, f, link, 10)
	_, m, _ := f.notesMenu()
	item := m.items[0]
	_, _ = f.stdin.Write([]byte{0}) // the switcher: a private frame
	waitFor(t, "switcher open", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	waitFor(t, "menu layout cleared", func() bool { _, mm, _ := f.notesMenu(); return !mm.Open() })
	f.mouse(0, m.X+8, item.Top, false)
	f.mouse(0, m.X+8, item.Top, true)
	time.Sleep(100 * time.Millisecond)
	if copies(f, "a") != 0 || strings.Contains(f.notice(), "Note copied") {
		t.Fatal("a hidden menu copied a note")
	}
}

// The menu is drawn over everything the collaboration layer adds: a mark
// inside its box leaves the box untinted (M4 review).
func TestNotesMenuDrawnOverMarks(t *testing.T) {
	f, s, _ := pointerOnFixture(t, nil)
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(s, []broadcast.Note{{ID: 1, Col: 4, Row: 4, Text: "x"}}, 99)
		return nil
	})
	f.con.pnotes.mu.Lock()
	f.con.pnotes.menuOpen = true
	f.con.pnotes.mu.Unlock()
	frame := menuFrame(t, 80, 24)
	frame.ChromeRows = 1
	m := RenderNotesMenu([]broadcast.Note{{ID: 1, Text: "x"}}, 80, 24)
	f.con.pmarks.mu.Lock()
	f.con.pmarks.marks = broadcast.NewMarks()
	f.con.pmarks.marks.Add([][2]int{{m.X + 10, m.Y + 1}}, 80, 24, 1, 0, f.con.now())
	f.con.pmarks.mu.Unlock()
	out := f.con.screenOverlay(frame, terminal.FramePublic)
	if bg := out.Cells[(m.Y+1)*80+m.X+10].Style.Bg; bg == markTint {
		t.Fatal("a mark was drawn over the notes menu")
	}
}

// offBeat waits until the shared flash beat isn't showing, so a full glow can
// only come from the hover.
func offBeat() {
	every := int64(broadcast.NoteFlashEvery)
	quiet := int64(broadcast.NoteFlashHold + broadcast.NoteFlashFade + 200*time.Millisecond)
	if phase := time.Now().UnixNano() % every; phase < quiet {
		time.Sleep(time.Duration(quiet - phase))
	}
	if phase := time.Now().UnixNano() % every; phase > every-int64(time.Second) {
		time.Sleep(time.Duration(every-phase) + time.Duration(quiet))
	}
}

// Hovering a menu item, by mouse motion through the console, highlights its
// rows and holds its 💭 at full glow; without the hover the same note rests
// (M4 review: the wiring, not just the state, is tested).
func TestNotesMenuHoverGlowsItsNote(t *testing.T) {
	f, s, _ := pointerOnFixture(t, nil)
	old := time.Now().Add(-time.Hour) // long past its arrival flash
	_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
		f.con.applyNotes(s, []broadcast.Note{{ID: 1, Col: 10, Row: 5, Color: 1, Text: "x", At: old}}, 99)
		return nil
	})
	var span ColumnSpan
	waitFor(t, "💭1 drawn", func() bool { span = f.notesSpan(); return span != (ColumnSpan{}) })
	f.clickStatus(span.Start + 1)
	var m NotesMenu
	waitFor(t, "menu drawn", func() bool { _, mm, _ := f.notesMenu(); m = mm; return m.Open() })
	f.mouse(35, m.X+8, m.items[0].Top, false)
	waitFor(t, "hovered", func() bool { _, _, h := f.notesMenu(); return h == 1 })

	frame := menuFrame(t, 80, 24)
	frame.ChromeRows = 1
	f.con.pnotes.mu.Lock()
	style := f.con.pnotes.style
	f.con.pnotes.mu.Unlock()
	full := broadcast.NoteChipColor(1, style)
	offBeat()
	hovered := f.con.screenOverlay(frame, terminal.FramePublic)
	if bg := hovered.Cells[5*80+9].Style.Bg; bg != full {
		t.Fatalf("hovered note's glow %v, want the full color %v", bg, full)
	}
	if bg := hovered.Cells[m.items[0].Top*80+m.X+6].Style.Bg; bg != notesMenuHover {
		t.Fatalf("hovered item row %v, want the hover background", bg)
	}
	// Control: the same note, unhovered, at the same moment of the beat rests.
	f.mouse(35, 2, 2, false)
	waitFor(t, "hover cleared", func() bool { _, _, h := f.notesMenu(); return h == 0 })
	offBeat()
	resting := f.con.screenOverlay(frame, terminal.FramePublic)
	if bg := resting.Cells[5*80+9].Style.Bg; bg == full {
		t.Fatal("an unhovered note glowed at full color off the beat")
	}
}

// A press in the menu owns the gesture until its release: drag motion out of
// the box is the menu's (M4 review). The presenter happens to drop motion
// from a press it never saw, so the child can't show the difference; the
// routing decision is what is asserted.
func TestNotesMenuDragStaysTheMenus(t *testing.T) {
	f, _, link := pointerOnFixture(t, nil)
	openMenuWith(t, f, link, 10)
	_, m, _ := f.notesMenu()
	took := func(button, x, y int, release bool) (got bool) {
		end := "M"
		if release {
			end = "m"
		}
		_ = f.con.runTerminalCommand(f.con.lifetime, func() error {
			got = f.con.routeNotesMenu([]byte(fmt.Sprintf("\x1b[<%d;%d;%d%s", button, x+1, y+1, end)))
			return nil
		})
		return got
	}
	if took(32, 3, 3, false) {
		t.Fatal("motion outside the menu, with no press in it, is not the menu's")
	}
	if !took(0, m.X+8, m.items[0].Top, false) {
		t.Fatal("a press in the menu is the menu's")
	}
	if !took(32, 3, 3, false) {
		t.Fatal("drag motion after a press in the menu is the menu's")
	}
	if !took(0, 3, 3, true) {
		t.Fatal("the press's release is the menu's")
	}
	if took(32, 3, 3, false) {
		t.Fatal("after the release, motion outside goes on")
	}
}

// pair#495: the menu names a screen note's author like a web note's; the
// board keeps the plain text a click copies.
func TestScreenNotesShowTheirAuthorInTheMenu(t *testing.T) {
	notes := []broadcast.Note{{ID: 1, Text: "fix this", Author: "Ada"}, {ID: 2, Text: "anon"}}
	got := withAuthors(notes)
	if got[0].Text != "Ada: fix this" || got[1].Text != "anon" {
		t.Fatalf("menu texts %q %q", got[0].Text, got[1].Text)
	}
	if notes[0].Text != "fix this" {
		t.Fatal("the board's note was changed")
	}
}
