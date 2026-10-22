package broadcast

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTerminalNoteEditOwnSessionOnly(t *testing.T) {
	p, link := notesOn(t)
	p.s.pointer.noteLimit = NewRateLimit(1000, 1000)
	a, b := jarClient(t), jarClient(t)
	for _, c := range []*http.Client{a, b} {
		if status, _, _ := unlockWith(t, c, link, codeOf(t, p.s, LinkPointer)); status != http.StatusOK {
			t.Fatalf("unlock %d", status)
		}
	}
	if status, body, _ := do(t, a, http.MethodPost, link+"note", "application/json", noteBody(3, 1, 2, "original"), nil); status != http.StatusNoContent {
		t.Fatalf("create %d %s", status, body)
	}
	ns, _ := p.s.Notes()
	original := ns[0]
	body := fmt.Sprintf(`{"id":%d,"text":"changed"}`, original.ID)
	if status, msg, _ := do(t, a, http.MethodPost, link+"note-edit", "application/json", body, nil); status != http.StatusNoContent {
		t.Fatalf("own edit %d %s", status, msg)
	}
	if status, _, _ := do(t, b, http.MethodPost, link+"note-edit", "application/json", body, nil); status != http.StatusForbidden {
		t.Fatalf("other session edit %d", status)
	}
	ns, _ = p.s.Notes()
	if len(ns) != 1 || ns[0].Text != "changed" || ns[0].ID != original.ID || ns[0].Author != original.Author || ns[0].Box != original.Box || ns[0].Col != original.Col || ns[0].Row != original.Row {
		t.Fatalf("edit changed identity/placement: %+v", ns)
	}
	if status, _, _ := do(t, b, http.MethodPost, link+"resolve", "application/json", fmt.Sprintf(`{"id":%d}`, original.ID), nil); status != http.StatusNoContent {
		t.Fatalf("other session resolve %d", status)
	}
	if status, _, _ := do(t, a, http.MethodPost, link+"note-edit", "application/json", body, nil); status != http.StatusConflict {
		t.Fatalf("retired edit %d", status)
	}
	if ns, _ := p.s.Notes(); len(ns) != 0 {
		t.Fatal("edit recreated resolved note")
	}
}

func TestParseNoteEditRejectsAmbiguousIdentity(t *testing.T) {
	for _, body := range []string{
		`{"id":1,"id":2,"text":"x"}`, `{"id":1,"ID":2,"text":"x"}`,
		`{"id":1,"text":"x","text":"y"}`, `{"id":1,"text":"x","owner":"operator"}`,
		`{"id":1,"text":" "}`, `{"id":null,"text":"x"}`,
	} {
		if _, err := ParseNoteEditRequest([]byte(body)); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}

func TestNoteBoardEditPreservesNoteAndRefusals(t *testing.T) {
	var b NoteBoard
	n, err := b.Add(Note{Box: CellBox{3, 1, 2, 1}, Text: "old", Owner: "one", Author: "Ana"}, 40, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"", "two", OperatorNoteOwner} {
		before := b.List()
		if err := b.Edit(NoteEditRequest{ID: n.ID, Owner: owner, Text: "stolen"}); !errors.Is(err, ErrNoteOwner) {
			t.Fatalf("owner %s: %v", owner, err)
		}
		if !reflect.DeepEqual(before, b.List()) {
			t.Fatal("refusal mutated board")
		}
	}
	if err := b.Edit(NoteEditRequest{ID: n.ID, Owner: "one", Text: "new"}); err != nil {
		t.Fatal(err)
	}
	want := n
	want.Text, want.Edited = "new", true
	if !reflect.DeepEqual(b.List(), []Note{want}) {
		t.Fatalf("edited %+v", b.List())
	}
}

func TestTerminalResolveAcceptsAttributionFromAnotherViewer(t *testing.T) {
	p, link := notesOn(t)
	a, b := jarClient(t), jarClient(t)
	for _, c := range []*http.Client{a, b} {
		unlockWith(t, c, link, codeOf(t, p.s, LinkPointer))
	}
	if status, _, _ := do(t, a, http.MethodPost, link+"note", "application/json", noteBody(3, 1, 2, "original"), nil); status != 204 {
		t.Fatalf("create %d", status)
	}
	ns, _ := p.s.Notes()
	body := fmt.Sprintf(`{"id":%d,"author":"Bo"}`, ns[0].ID)
	if status, msg, _ := do(t, b, http.MethodPost, link+"resolve", "application/json", body, nil); status != 204 {
		t.Fatalf("resolve %d %s", status, msg)
	}
	events := p.s.Resolutions()
	if len(events) != 1 || events[0].ID != ns[0].ID || events[0].Author != "Bo" || events[0].At.IsZero() {
		t.Fatalf("resolution %+v", events)
	}
	if err := p.s.ResolveNote(ns[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(p.s.Resolutions()) != 1 {
		t.Fatal("repeat resolve recorded another event")
	}
}

func TestTerminalResolutionRetentionAndEditGeneration(t *testing.T) {
	p := onPointer(t)
	s := &Session{pointer: p}
	gen := p.generation()
	for i := 0; i < MaxNoteResolutions+3; i++ {
		n, err := p.addNote(Note{Box: CellBox{3, 1, 2, 1}, Text: "text", Owner: "one"}, 40, 4, 1, gen)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.resolveNote(NoteResolution{ID: n.ID, Author: "Bo"}); err != nil {
			t.Fatal(err)
		}
	}
	events := s.Resolutions()
	if len(events) != MaxNoteResolutions || events[0].ID != 4 {
		t.Fatalf("retention %+v", events)
	}
	events[0].Author = "changed"
	if s.Resolutions()[0].Author != "Bo" {
		t.Fatal("inspection exposed mutable storage")
	}
	p.set(false, nil)
	p.set(true, nil)
	n, err := p.addNote(Note{Box: CellBox{3, 1, 2, 1}, Text: "text", Owner: "one"}, 40, 4, 1, p.generation())
	if err != nil {
		t.Fatal(err)
	}
	if err := p.editNote(NoteEditRequest{ID: n.ID, Owner: "one", Text: "late"}, gen); !errors.Is(err, ErrPointerOff) {
		t.Fatalf("stale generation %v", err)
	}
	p.stop()
	if len(s.Resolutions()) != 0 {
		t.Fatal("cast end retained history")
	}
}

// rotateNoteBody holds back the body until the already-admitted session is
// revoked. No cancellation-goroutine scheduling is needed to trigger the race.
type rotateNoteBody struct {
	io.Reader
	rotate func()
}

func (b *rotateNoteBody) Read(p []byte) (int, error) {
	if b.rotate != nil {
		f := b.rotate
		b.rotate = nil
		f()
	}
	return b.Reader.Read(p)
}

func TestTerminalNoteEditRefusesRotationDuringBodyRead(t *testing.T) {
	p := onPointer(t)
	l, err := NewLinkLock(4, nil)
	if err != nil {
		t.Fatal(err)
	}
	session, _ := l.Unlock(time.Now(), l.Code())
	n, err := p.addNote(Note{Box: CellBox{3, 1, 2, 1}, Text: "original", Owner: sessionNoteOwner(session)}, 40, 4, 1, p.generation())
	if err != nil {
		t.Fatal(err)
	}
	body := &rotateNoteBody{Reader: strings.NewReader(fmt.Sprintf(`{"id":%d,"text":"late"}`, n.ID)), rotate: func() {
		if _, err := l.Rotate(); err != nil {
			t.Fatal(err)
		}
	}}
	req := httptest.NewRequest(http.MethodPost, "/"+p.token+"/note-edit", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	w := httptest.NewRecorder()
	admitted, done := gate(w, req, p.token, "note-edit", l, false)
	if admitted == nil {
		t.Fatal("initial session admission failed")
	}
	defer done()
	srv := NewServer(ServerOptions{Pointer: p, OnNoteEdit: func(in NoteEditRequest) error { return p.editNote(in, p.generation()) }})
	srv.ServeHTTP(w, admitted)
	if w.Code != http.StatusForbidden {
		t.Fatalf("revoked edit: %d %s", w.Code, w.Body.String())
	}
	if got := p.board.List()[0].Text; got != "original" {
		t.Fatalf("revoked edit changed note: %q", got)
	}
}
