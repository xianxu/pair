package broadcast

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

// notesOn is a session with pointing on and the operator's 40x4 screen
// showing the pointer marker.
func notesOn(t *testing.T) (*pointerSession, string) {
	t.Helper()
	p := startPointerSession(t)
	link, err := p.enablePointer(t)
	if err != nil {
		t.Fatal(err)
	}
	p.showPointer(t)
	return p, link
}

func noteBody(col, row, color int, text string) string {
	return fmt.Sprintf(`{"cols":40,"rows":4,"box":[%d,%d,2,1],"color":%d,"text":%q}`, col, row, color, text)
}

// notesEvent reads the next notes event from a stream.
func notesEvent(t *testing.T, r *bufio.Reader) wireNotes {
	t.Helper()
	var w wireNotes
	if err := json.Unmarshal([]byte(readEvent(t, r, "notes")), &w); err != nil {
		t.Fatal(err)
	}
	return w
}

func stream(t *testing.T, url string) *bufio.Reader {
	t.Helper()
	resp, err := castClient.Get(url + "events")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return bufio.NewReader(resp.Body)
}

func TestNoteSavedReachesEveryStream(t *testing.T) {
	p, link := notesOn(t)
	ps, vs := stream(t, link), stream(t, p.s.Link())
	for _, r := range []*bufio.Reader{ps, vs} {
		if w := notesEvent(t, r); len(w.Notes) != 0 || w.Guard != StatusGuardCols {
			t.Fatalf("on join: %+v", w)
		}
	}
	if code, body := post(t, link+"note", "application/json", noteBody(3, 1, 2, "look <b>here</b>")); code != http.StatusNoContent {
		t.Fatalf("POST note %d %q", code, body)
	}
	for name, r := range map[string]*bufio.Reader{"pointer": ps, "view": vs} {
		w := notesEvent(t, r)
		if len(w.Notes) != 1 || w.Notes[0].Text != "look <b>here</b>" || w.Notes[0].Col != 3 || w.Notes[0].Color != 2 {
			t.Fatalf("%s stream: %+v", name, w)
		}
	}
	// A page that joins later gets the list straight away.
	if w := notesEvent(t, stream(t, p.s.Link())); len(w.Notes) != 1 {
		t.Fatalf("late join: %+v", w)
	}
	calls := p.notesCalls()
	if len(calls) != 1 || len(calls[0].notes) != 1 {
		t.Fatalf("OnNotes %+v", calls)
	}
	id := calls[0].notes[0].ID
	if code, _ := post(t, link+"resolve", "application/json", fmt.Sprintf(`{"id":%d}`, id)); code != http.StatusNoContent {
		t.Fatalf("resolve %d", code)
	}
	for _, r := range []*bufio.Reader{ps, vs} {
		if w := notesEvent(t, r); len(w.Notes) != 0 {
			t.Fatalf("after resolve: %+v", w)
		}
	}
	if code, _ := post(t, link+"resolve", "application/json", fmt.Sprintf(`{"id":%d}`, id)); code != http.StatusNoContent {
		t.Fatalf("resolving an unknown id: %d", code)
	}
	calls = p.notesCalls()
	if len(calls) != 2 || len(calls[1].notes) != 0 || calls[1].version <= calls[0].version {
		t.Fatalf("OnNotes after resolve %+v", calls)
	}
}

func TestNoteRefusals(t *testing.T) {
	p, link := notesOn(t)
	for i := 0; i < MaxNotesPerColor; i++ {
		if code, body := post(t, link+"note", "application/json", noteBody(i*2, 0, 1, "x")); code != http.StatusNoContent {
			t.Fatalf("note %d: %d %q", i, code, body)
		}
		p.s.pointer.noteLimit = NewRateLimit(1000, 1000) // keep the budget out of this test
	}
	cases := []struct {
		name, body string
		code       int
		text       string
	}{
		{"color full", noteBody(20, 1, 1, "x"), 409, refuseColorFull},
		{"overlap", noteBody(1, 0, 2, "x"), 409, refuseOverlap},
		{"malformed", `{"cols":40}`, 400, refuseNote},
		{"control text", noteBody(20, 1, 2, "a\x1bb"), 400, refuseNote},
		{"guarded cell", noteBody(0, 3, 2, "x"), 400, refuseNote},
	}
	for _, c := range cases {
		code, body := post(t, link+"note", "application/json", c.body)
		if code != c.code || strings.TrimSpace(body) != c.text {
			t.Errorf("%s: %d %q, want %d %q", c.name, code, body, c.code, c.text)
		}
	}
}

// A note lands only on what the operator sees now.
func TestNoteScreenChanged(t *testing.T) {
	cases := map[string]struct {
		frame func(*testing.T) terminal.Frame
		class terminal.FrameClass
		body  string
	}{
		"other grid":     {func(t *testing.T) terminal.Frame { return livePointer(t, "x") }, terminal.FramePublic, `{"cols":80,"rows":24,"box":[3,1,2,1],"color":0,"text":"x"}`},
		"private screen": {func(t *testing.T) terminal.Frame { return livePointer(t, "x") }, terminal.FramePrivate, noteBody(3, 1, 0, "x")},
		"marker hidden":  {func(t *testing.T) terminal.Frame { return live(t, "x") }, terminal.FramePublic, noteBody(3, 1, 0, "x")},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := startPointerSession(t)
			link, _ := p.enablePointer(t)
			p.s.Offer(c.frame(t), c.class)
			p.s.hub.sync()
			code, body := post(t, link+"note", "application/json", c.body)
			if code != http.StatusConflict || strings.TrimSpace(body) != refuseScreen {
				t.Fatalf("%d %q", code, body)
			}
			if len(p.notesCalls()) != 0 {
				t.Fatal("OnNotes called for a refused note")
			}
		})
	}
}

func TestNoteWhilePointingOff(t *testing.T) {
	p, link := notesOn(t)
	p.s.DisablePointer()
	for _, route := range []string{"note", "resolve"} {
		body := noteBody(3, 1, 0, "x")
		if route == "resolve" {
			body = `{"id":1}`
		}
		code, text := post(t, link+route, "application/json", body)
		if code != http.StatusForbidden || strings.TrimSpace(text) != refusePointerOff {
			t.Errorf("%s while off: %d %q", route, code, text)
		}
	}
	if len(p.notesCalls()) != 0 {
		t.Fatal("OnNotes called while off")
	}
}

func TestNoteRateLimit(t *testing.T) {
	_, link := notesOn(t)
	limited := 0
	for i := 0; i < NoteBurst+5; i++ {
		if code, _ := post(t, link+"resolve", "application/json", `{"id":1}`); code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited < 4 {
		t.Fatalf("only %d of %d rapid note requests were limited", limited, NoteBurst+5)
	}
}

// Turning pointing off deletes every note, by the operator's click or by
// the 👆-hidden watch; turning it on again starts empty.
func TestNotesClearedWhenPointingTurnsOff(t *testing.T) {
	offs := map[string]func(p *pointerSession){
		"click": func(p *pointerSession) { p.s.DisablePointer() },
		"watch": func(p *pointerSession) { p.s.pointerHidden(p.s.pointer.generation()) },
	}
	for name, off := range offs {
		t.Run(name, func(t *testing.T) {
			p, link := notesOn(t)
			vs := stream(t, p.s.Link())
			notesEvent(t, vs)
			post(t, link+"note", "application/json", noteBody(3, 1, 0, "a"))
			post(t, link+"note", "application/json", noteBody(9, 1, 1, "b"))
			notesEvent(t, vs)
			notesEvent(t, vs)
			before := p.notesCalls()
			off(p)
			if w := notesEvent(t, vs); len(w.Notes) != 0 {
				t.Fatalf("view stream after off: %+v", w)
			}
			calls := p.notesCalls()
			last := calls[len(calls)-1]
			if len(calls) != len(before)+1 || len(last.notes) != 0 || last.version <= before[len(before)-1].version {
				t.Fatalf("OnNotes after off: %+v", calls)
			}
			p.enablePointer(t)
			if notes, _ := p.s.Notes(); len(notes) != 0 {
				t.Fatal("re-enabled pointing brought notes back")
			}
		})
	}
}

// Saving a note waits on the hub loop (the screen check); the hub loop's
// pointer watch takes the pointer lock. With no lock held across the screen
// check, notes posted while the watch fires never deadlock.
func TestNotesDoNotDeadlockWithPointerWatch(t *testing.T) {
	p := startPointerSessionWith(t, HubOptions{PointerAfter: func(time.Duration) (<-chan time.Time, func() bool) {
		tm := time.NewTimer(time.Millisecond)
		return tm.C, tm.Stop
	}})
	p.s.pointer.noteLimit = NewRateLimit(100000, 100000)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 40; i++ {
			link, _ := p.enablePointer(t)
			p.showPointer(t)
			post(t, link+"note", "application/json", noteBody((i%10)*2, 1, i%PointerColors, "x"))
			// Hide the marker: the watch turns pointing off after its grace.
			p.s.Offer(live(t, "x"), terminal.FramePublic)
		}
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("notes and the pointer watch deadlocked")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if on, _ := p.s.pointer.On(); !on {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the watch never turned pointing off")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if notes, _ := p.s.Notes(); len(notes) != 0 {
		t.Fatalf("%d notes left after the watch turned pointing off", len(notes))
	}
}

// A view page without a pointer capability gets no notes event at all.
func TestNotesNeedAPointerCapability(t *testing.T) {
	h, srv := testServer(t, HubOptions{}, 20*time.Millisecond)
	_ = h
	ts := httpServer(t, srv)
	resp, events := openEvents(t, ts.URL)
	defer resp.Body.Close()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if ev := events.next(); ev.name == "notes" {
			t.Fatal("a server without a pointer capability sent notes")
		}
	}
}

// The draw-time anchor is captured as the note opens and never reaches a page.
func TestNoteAnchorLifecycle(t *testing.T) {
	p, link := notesOn(t)
	var asked []Note
	p.s.cfg.AnchorNote = func(n Note, cols, rows int) NoteAnchor {
		asked = append(asked, n)
		return NoteAnchor{Kind: AnchorScreen, Path: []string{"pane pair:3"}, Text: fmt.Sprintf("%dx%d", cols, rows)}
	}
	vs := stream(t, p.s.Link())
	notesEvent(t, vs)
	if code, body := post(t, link+"note", "application/json", noteBody(3, 1, 0, "a")); code != http.StatusNoContent {
		t.Fatalf("POST note %d %q", code, body)
	}
	raw := readEvent(t, vs, "notes")
	if strings.Contains(raw, "anchor") || strings.Contains(raw, "pane pair:3") {
		t.Fatalf("the anchor reached a page: %s", raw)
	}
	notes, _ := p.s.Notes()
	if len(asked) != 1 || len(notes) != 1 || notes[0].Anchor == nil || notes[0].Anchor.Text != "40x4" || notes[0].On != (terminal.Geometry{Cols: 40, Rows: 4}) {
		t.Fatalf("asked %+v, notes %+v", asked, notes)
	}
}
