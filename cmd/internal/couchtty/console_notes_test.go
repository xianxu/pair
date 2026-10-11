package couchtty

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func postNote(t *testing.T, link string, col, row int, text string) (int, string) {
	t.Helper()
	body := fmt.Sprintf(`{"cols":80,"rows":24,"col":%d,"row":%d,"color":0,"text":%q}`, col, row, text)
	resp, err := http.Post(link+"note", "application/json", strings.NewReader(body))
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
	v := openViewer(t, s.Link())
	noteUntilSaved(t, link, 10, 5, "look here")
	waitFor(t, "💭 on the operator's screen", func() bool { return f.cellText(10, 5) == broadcast.NoteGlyph })
	waitFor(t, "💭1 in the tab bar", func() bool { return strings.Contains(f.screenText(), broadcast.NoteGlyph+"1") })
	v.until("💭 in the broadcast", func(string) bool {
		c := v.emu.CellAt(10, 5)
		return c != nil && c.Content == broadcast.NoteGlyph
	})
	notes, _ := s.Notes()
	resp, err := http.Post(link+"resolve", "application/json", strings.NewReader(fmt.Sprintf(`{"id":%d}`, notes[0].ID)))
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
