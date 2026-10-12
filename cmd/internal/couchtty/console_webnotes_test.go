package couchtty

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// recordedFake is a FakeBrowser whose recording's one writer is the
// reference board.Writer, as the recorder's queue is for Chrome.
type recordedFake struct {
	*webcast.FakeBrowser
	w *board.Writer
}

func (r *recordedFake) Emit(rec recording.Record, doc int) error { return r.w.Emit(rec, doc) }

// webNotesCast is a console with a recorded web cast showing a page, and
// a way to post a viewer's note over the cast link.
type webNotesCast struct {
	*webcastRig
	mem  *recording.MemSink
	link string
	p    recording.Node
}

func newWebNotesCast(t *testing.T, screen bool) *webNotesCast {
	t.Helper()
	var rf *recordedFake
	var ro *webcast.RecordOptions
	r := newWebcastRig(t, func(r *webcastRig) {
		r.screen = screen
		r.wrap = func(f *webcast.FakeBrowser, o *webcast.RecordOptions) webcast.Browser {
			ro, rf = o, &recordedFake{FakeBrowser: f}
			return rf
		}
	})
	r.child.Feed([]byte("\x1b[?1003;1006h"))
	waitFor(t, "child tracks the mouse", func() bool { return r.child.Endpoint().Modes().Tracking != 0 })
	r.open(t)
	o, err := ro.Out() // the go-live opens the recording
	if err != nil {
		t.Fatal(err)
	}
	c := &webNotesCast{webcastRig: r, mem: &recording.MemSink{}}
	rf.w = board.NewWriter(recording.Tee(o.Sink, c.mem), nil)
	b := recording.NewBuilder("rec")
	c.p = b.El("p", []string{"id", "p"}, b.Txt("Hello"))
	b.Page(recording.SegLive, "http://app/", c.p)
	for _, rec := range b.Recs {
		if err := rf.w.Put(rec); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "link", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		c.link = r.con.wcast.update.Link
		return c.link != ""
	})
	return c
}

func (c *webNotesCast) note(t *testing.T, name, text string) {
	t.Helper()
	body := fmt.Sprintf(`{"viewer":{"id":"v-%s","name":%q},"seg":1,"node":%d,"box":[1,2,3,4],"color":2,"text":%q}`, name, name, c.p.ID, text)
	resp, err := http.Post(c.link+"note", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("note: %d", resp.StatusCode)
	}
}

func (c *webNotesCast) webSpan(t *testing.T) ColumnSpan {
	t.Helper()
	var span ColumnSpan
	waitFor(t, "the web notes span", func() bool {
		c.con.mu.Lock()
		defer c.con.mu.Unlock()
		span = c.con.statusWebNotes
		return span != ColumnSpan{}
	})
	return span
}

// The operator sees viewers' web notes in Couch (#470): 💭N after the
// Chrome cell; a click opens #447's notes menu with them (author and text);
// a click copies one; a right click copies and resolves it, for viewers too.
func TestWebNotesInCouch(t *testing.T) {
	r := newWebNotesCast(t, false)
	r.note(t, "Ana", "fix the header")
	waitFor(t, "Chrome ▶ 💭1", func() bool { return strings.Contains(r.lastRow(), WebcastIdleLabel+" "+broadcast.NoteGlyph+"1") })
	span := r.webSpan(t)
	if r.notesSpan() != (ColumnSpan{}) {
		t.Fatal("the screen broadcast's 💭N must not show for web notes")
	}
	r.clickStatus(span.Start + 1)
	var m NotesMenu
	waitFor(t, "menu open with the web note", func() bool { open, mm, _ := r.notesMenu(); m = mm; return open && m.Open() })
	waitFor(t, "author and text on screen", func() bool { return strings.Contains(r.screenText(), "Ana: fix the header") })
	item := m.items[0]
	r.mouse(0, m.X+8, item.Top, false)
	r.mouse(0, m.X+8, item.Top, true)
	waitFor(t, "the text copied, without the author", func() bool { return copies(r.consoleFixture, "fix the header") == 1 })
	// A right click resolves it: the count goes, the menu closes.
	r.mouse(2, m.X+8, item.Top, false)
	r.mouse(2, m.X+8, item.Top, true)
	waitFor(t, "resolved notice", func() bool { return strings.Contains(r.notice(), "copied and resolved") })
	waitFor(t, "💭 gone", func() bool { return !strings.Contains(r.lastRow(), broadcast.NoteGlyph) })
	waitFor(t, "menu closed", func() bool { open, _, _ := r.notesMenu(); return !open })
	if r.childGot(fmt.Sprintf(";%d;%dM", m.X+9, item.Top+1)) {
		t.Fatal("a click over the menu reached the child")
	}
	// The resolve is in the recording, as the operator's.
	resolved := false
	for _, rec := range r.mem.Records() {
		if s, ok := rec.(*recording.NoteState); ok && s.State == recording.NoteResolved {
			resolved = true
		}
	}
	if !resolved {
		t.Fatal("no note-state in the recording")
	}
}

// The overlay the menu draws in is shared (syncOverlay): stopping the
// screen broadcast leaves it for the web cast, whose menu still draws; the
// web cast's end removes it.
func TestWebNotesMenuOutlivesTheScreenBroadcast(t *testing.T) {
	r := newWebNotesCast(t, true)
	_, _ = r.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "broadcast live", func() bool { return r.phase() == broadcastLive })
	r.note(t, "Ana", "still here")
	_, _ = r.stdin.Write([]byte(ctrlAltB))
	waitFor(t, "broadcast off", func() bool { return r.phase() == broadcastOff })
	r.clickStatus(r.webSpan(t).Start + 1)
	waitFor(t, "the web menu drawn after the broadcast stopped", func() bool {
		open, m, _ := r.notesMenu()
		return open && m.Open() && strings.Contains(r.screenText(), "Ana: still here")
	})
	r.clickCell(t, 16) // Ctrl+click ends the cast
	waitFor(t, "cast ended, overlay released", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		return r.con.wcast.phase == webcastOff && r.con.overlayUsers == 0
	})
}

// One menu, two sources: a click on the other 💭N switches it, and the
// hover never glows a screen note while the menu shows web notes. A web
// note keeps its menu id while open, so a click on a row drawn before the
// list changed acts on that row's note or on nothing, never on another.
func TestWebNotesMenuSourceAndStableIDs(t *testing.T) {
	r := newWebNotesCast(t, false)
	c := r.con
	live := func(ids ...string) []board.LiveNote {
		var out []board.LiveNote
		for _, id := range ids {
			out = append(out, board.LiveNote{ID: id, Text: "text " + id})
		}
		return out
	}
	c.pnotes.mu.Lock()
	c.pnotes.notes = []broadcast.Note{{ID: 1, Text: "screen"}}
	c.pnotes.webItems, c.pnotes.web = webMenuNotes(live("a", "b"), c.webMenuIDLocked)
	idA, idB := c.pnotes.webItems[0].menuID, c.pnotes.webItems[1].menuID
	c.pnotes.mu.Unlock()
	run := func(fn func()) {
		_ = c.runTerminalCommand(c.lifetime, func() error { fn(); return nil })
	}
	run(func() { c.toggleNotesMenu(screenNotes) })
	c.pnotes.mu.Lock()
	c.pnotes.hover = 1
	src, hover := c.pnotes.source, c.screenHoverLocked()
	c.pnotes.mu.Unlock()
	if src != screenNotes || hover != 1 {
		t.Fatalf("screen menu: source %d hover %d", src, hover)
	}
	run(func() { c.toggleNotesMenu(webNotes) })
	c.pnotes.mu.Lock()
	open, src, hover := c.pnotes.menuOpen, c.pnotes.source, c.pnotes.hover
	c.pnotes.hover = idA
	glow := c.screenHoverLocked()
	c.pnotes.mu.Unlock()
	if !open || src != webNotes || hover != 0 || glow != 0 {
		t.Fatalf("switched to web: open %v source %d hover %d, screen glow %d", open, src, hover, glow)
	}
	// The list changes under a drawn menu: a is resolved elsewhere.
	c.pnotes.mu.Lock()
	c.pnotes.webItems, c.pnotes.web = webMenuNotes(live("b"), c.webMenuIDLocked)
	textA, resolveA, textB := c.noteTextLocked(idA), c.resolverLocked(idA), c.noteTextLocked(idB)
	newB := c.pnotes.webItems[0].menuID
	c.pnotes.mu.Unlock()
	if textA != "" || resolveA != nil || textB != "text b" || newB != idB {
		t.Fatalf("after a's resolve: a %q %v, b %q (id %d, was %d)", textA, resolveA != nil, textB, newB, idB)
	}
}

func TestWebMenuNotes(t *testing.T) {
	ids := map[string]int{"n1": 7}
	next := 10
	items, notes := webMenuNotes([]board.LiveNote{
		{ID: "n1", Author: recording.Author{Name: "Ana"}, Text: "one", Color: 3},
		{ID: "n2", Text: "two", Color: -1},
	}, func(id string) int {
		if v, ok := ids[id]; ok {
			return v
		}
		next++
		ids[id] = next
		return next
	})
	want := []broadcast.Note{{ID: 7, Color: 3, Text: "Ana: one"}, {ID: 11, Color: 0, Text: "two"}}
	if fmt.Sprint(notes) != fmt.Sprint(want) || fmt.Sprint(items) != "[{7 n1 one} {11 n2 two}]" {
		t.Errorf("items %v notes %v", items, notes)
	}
}

// The Chrome cell's 💭N follows it, with its own span.
func TestStatusRowWebNotes(t *testing.T) {
	row := RenderStatusRow(80, StatusModel{Webcast: WebcastLive, WebNotes: 3})
	plain := ansi.Strip(row.Body)
	if !strings.Contains(plain, WebcastLiveLabel+" "+broadcast.NoteGlyph+"3") || row.WebNotes == (ColumnSpan{}) || row.Notes != (ColumnSpan{}) ||
		row.WebNotes != (ColumnSpan{Start: 9, End: 12}) { // after `Chrome ⏸ `; 💭 is two columns
		t.Errorf("row %q spans %+v %+v", row.Body, row.WebNotes, row.Notes)
	}
	if row := RenderStatusRow(80, StatusModel{Webcast: WebcastOff, WebNotes: 3}); row.WebNotes != (ColumnSpan{}) {
		t.Errorf("no Chrome, no 💭N: %+v", row.WebNotes)
	}
}
