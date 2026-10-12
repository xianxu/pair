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

// The operator sees viewers' web notes in Couch (#470): 💭N after the
// Chrome cell; a click opens #447's notes menu with them (author and text);
// a click copies one; a right click copies and resolves it, for viewers too.
func TestWebNotesInCouch(t *testing.T) {
	var rf *recordedFake
	var ro *webcast.RecordOptions
	r := newWebcastRig(t, func(r *webcastRig) {
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
	mem := &recording.MemSink{}
	rf.w = board.NewWriter(recording.Tee(o.Sink, mem), nil)
	b := recording.NewBuilder("rec")
	p := b.El("p", []string{"id", "p"}, b.Txt("Hello"))
	b.Page(recording.SegLive, "http://app/", p)
	for _, rec := range b.Recs {
		if err := rf.w.Put(rec); err != nil {
			t.Fatal(err)
		}
	}
	var link string
	waitFor(t, "link", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		link = r.con.wcast.update.Link
		return link != ""
	})
	body := fmt.Sprintf(`{"viewer":{"id":"ana","name":"Ana"},"seg":1,"node":%d,"box":[1,2,3,4],"color":2,"text":"fix the header"}`, p.ID)
	resp, err := http.Post(link+"note", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("note: %d", resp.StatusCode)
	}
	waitFor(t, "Chrome ▶ 💭1", func() bool { return strings.Contains(r.lastRow(), WebcastIdleLabel+" "+broadcast.NoteGlyph+"1") })
	var span ColumnSpan
	waitFor(t, "the web notes span", func() bool {
		r.con.mu.Lock()
		defer r.con.mu.Unlock()
		span = r.con.statusWebNotes
		return span != ColumnSpan{}
	})
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
	for _, rec := range mem.Records() {
		if s, ok := rec.(*recording.NoteState); ok && s.State == recording.NoteResolved {
			resolved = true
		}
	}
	if !resolved {
		t.Fatal("no note-state in the recording")
	}
}

func TestWebMenuNotes(t *testing.T) {
	items, ids, texts := webMenuNotes([]board.LiveNote{
		{ID: "n1", Author: recording.Author{Name: "Ana"}, Text: "one", Color: 3},
		{ID: "n2", Text: "two", Color: -1},
	})
	want := []broadcast.Note{{ID: 1, Color: 3, Text: "Ana: one"}, {ID: 2, Color: 0, Text: "two"}}
	if fmt.Sprint(items) != fmt.Sprint(want) || fmt.Sprint(ids) != "[n1 n2]" || fmt.Sprint(texts) != "[one two]" {
		t.Errorf("items %v ids %v texts %v", items, ids, texts)
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
