package notedraft

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"sync"
	"testing"
	"time"
)

func preparation(t *testing.T) (board.AnnotationAdmission, board.PreparedNote) {
	t.Helper()
	_, a, p := preparationAuthority(t, false)
	return a, p
}

func preparationAuthority(t *testing.T, operator bool) (*board.Board, board.AnnotationAdmission, board.PreparedNote) {
	t.Helper()
	bd := board.New(notes.Structural{}, nil)
	if !operator {
		if err := bd.SetAnnotationAllowed(context.Background(), true); err != nil {
			t.Fatal(err)
		}
	}
	b := recording.NewBuilder("test")
	el := b.El("p", []string{"id", "card"}, b.Txt("Original card"))
	b.Page(recording.SegLive, "https://example.test/", el)
	for _, r := range b.Recs {
		bd.Put(r)
	}
	var a board.AnnotationAdmission
	var err error
	if operator {
		a, err = bd.AdmitOperatorAnnotationGeneration(0)
	} else {
		a, err = bd.AdmitAnnotation()
	}
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.PrepareNote(board.PrepareNoteIn{Seg: 1, Node: el.ID, Box: recording.Box{10, 10, 100, 50}, Region: recording.Box{10, 10, 100, 50}}, operator)
	if err != nil {
		t.Fatal(err)
	}
	return bd, a, p
}

type heldCapture struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
}

func (f *heldCapture) Capture(ctx context.Context, p board.PreparedNote) (Evidence, []byte, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	select {
	case f.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return Evidence{}, nil, ctx.Err()
}

func TestServiceQueueBoundSavesUnavailablePreparation(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	owner := Owner{Scope: "viewer", Generation: 1}
	a, p := preparation(t)
	first, err := s.Prepare(context.Background(), owner, "first", a, p, false)
	if err != nil || first.State != "pending" {
		t.Fatal(first, err)
	}
	<-cap.started
	for i := 0; i < 8; i++ {
		if _, err = s.Prepare(context.Background(), owner, string(rune('a'+i)), a, p, false); err != nil {
			t.Fatal(err)
		}
	}
	overflow, err := s.Prepare(context.Background(), owner, "overflow", a, p, false)
	if err != nil || overflow.State != "ready" || overflow.Draft == "" || overflow.Unavailable != CaptureBusy {
		t.Fatal(overflow, err)
	}
	if _, err = s.Status(Owner{Scope: "foreign", Generation: 1}, overflow.Draft); err == nil {
		t.Fatal("foreign owner read draft")
	}
}

func TestServiceCancelAndCloseBoundCapture(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap})
	if err != nil {
		t.Fatal(err)
	}
	owner := Owner{Scope: "viewer", Generation: 1}
	a, p := preparation(t)
	status, err := s.Prepare(context.Background(), owner, "cancel", a, p, false)
	if err != nil {
		t.Fatal(err)
	}
	<-cap.started
	if err = s.Cancel(owner, status.Draft); err != nil {
		t.Fatal(err)
	}
	got, err := s.Status(owner, status.Draft)
	if err != nil || got.State != "cancelled" {
		t.Fatal(got, err)
	}
	done := make(chan struct{})
	go func() { s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("close blocked on capture")
	}
}

func TestCancelBeforePrepareAndUnavailableReply(t *testing.T) {
	cap := &heldCapture{started: make(chan struct{}, 1)}
	s, err := New(Options{Root: t.TempDir(), RecordingPath: "/recording", Capture: cap})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, p := preparation(t)
	owner := Owner{Scope: "viewer", Generation: 1}
	if err = s.Cancel(owner, "abandoned"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Prepare(context.Background(), owner, "abandoned", a, p, false)
	if err != nil || got.State != "cancelled" {
		t.Fatal(got, err)
	}
	now := s.opts.Now()
	token, _ := s.token(&captureRecord{ID: opaqueID(), Owner: owner, Request: "token", RecordingPath: s.opts.RecordingPath, State: State{Phase: Ready, Unavailable: StoreFull}, Expires: now.Add(DraftTTL)})
	if err = s.Cancel(owner, token); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Status(owner, token); err == nil {
		t.Fatal("cancelled unavailable token remained usable")
	}
}
