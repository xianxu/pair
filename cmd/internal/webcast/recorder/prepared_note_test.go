package recorder

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
	"time"
)

func preparedFixture(t *testing.T) (*harness, *board.Board, board.AnnotationAdmission, board.NoteReservation) {
	t.Helper()
	h, b, _ := operatorHarness(t)
	a, err := b.AdmitOperatorAnnotationGeneration(b.AnnotationCapability().Generation)
	if err != nil {
		t.Fatal(err)
	}
	var node rec.NodeID
	var doc int
	h.r.ask(context.Background(), func() {
		doc = h.r.m.st.DocSeg
		node, _, err = h.r.m.nodeAt(h.r.m.sessions["A"], h.r.frameIDs["A"], []int{1, 2, 0})
	})
	p, err := a.PrepareNote(board.PrepareNoteIn{Seg: doc, Node: node, Box: rec.Box{1, 2, 30, 40}, Region: rec.Box{3, 4, 10, 10}}, true)
	if err != nil {
		t.Fatal(err)
	}
	n, err := a.ReservePreparedNote(p, board.Viewer{}, 0, "hello", "capture-1", true)
	if err != nil {
		t.Fatal(err)
	}
	return h, b, a, n
}
func TestPreparedWriteAcknowledgedAndObserved(t *testing.T) {
	h, _, a, n := preparedFixture(t)
	results := make(chan board.OperatorOutcome, 2)
	if err := h.r.WritePrepared(context.Background(), a, n, nil, func(v board.OperatorOutcome, e error) {
		if e != nil {
			t.Error(e)
		}
		results <- v
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := h.r.ObserveNote(context.Background(), n.Note.ID)
	if err != nil || !ok || got.Capture != "capture-1" {
		t.Fatalf("observe: %+v %v %v", got, ok, err)
	}
	if v := <-results; v != board.OperatorAccepted {
		t.Fatal(v)
	}
	got.At.Box[0] = 999
	again, ok, err := h.r.ObserveNote(context.Background(), n.Note.ID)
	if err != nil || !ok || again.At.Box[0] == 999 {
		t.Fatal("observation aliases record", err)
	}
	if _, ok, err = h.r.ObserveNote(context.Background(), "absent"); err != nil || ok {
		t.Fatal(ok, err)
	}
	h.r.Close()
	if err = h.r.WritePrepared(context.Background(), a, n, nil, func(board.OperatorOutcome, error) { t.Error("rejected enqueue settled") }); err == nil {
		t.Fatal("closed enqueue accepted")
	}
	select {
	case v := <-results:
		t.Fatal("settled twice", v)
	default:
	}
}
func TestPreparedWriteRefusalsAndUncertainty(t *testing.T) {
	for _, why := range []string{"pause", "navigation", "disabled", "check", "canceled", "sink"} {
		t.Run(why, func(t *testing.T) {
			h, b, a, n := preparedFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var check func() error
			switch why {
			case "pause":
				h.r.Send(Pause{})
			case "navigation":
				n.Document++
			case "disabled":
				if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
					t.Fatal(err)
				}
			case "check":
				check = func() error { return board.ErrStale }
			case "sink":
				h.r.ask(context.Background(), func() { h.r.sink = operatorFailSink{h.r.sink} })
			}
			entered, release := make(chan struct{}), make(chan struct{})
			h.r.enqueue(query{fn: func() { close(entered); <-release }, done: make(chan struct{})}, false)
			<-entered
			result := make(chan board.OperatorOutcome, 2)
			if err := h.r.WritePrepared(ctx, a, n, check, func(v board.OperatorOutcome, e error) {
				if e == nil {
					t.Error("missing refusal reason")
				}
				result <- v
			}); err != nil {
				close(release)
				t.Fatal(err)
			}
			if why == "canceled" {
				cancel()
			}
			close(release)
			want := board.OperatorRefused
			if why == "sink" {
				want = board.OperatorUnknown
			}
			select {
			case v := <-result:
				if v != want {
					t.Fatal(v, want)
				}
			case <-time.After(time.Second):
				t.Fatal("missing settlement")
			}
			_, ok, err := h.r.ObserveNote(context.Background(), n.Note.ID)
			if why == "sink" {
				if err == nil {
					t.Fatal("failed sink claimed observation certainty")
				}
			} else if err != nil || ok {
				t.Fatal(ok, err)
			}
		})
	}
}
func TestPreparedObserveCancellationDoesNotRace(t *testing.T) {
	h, _, _, _ := preparedFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	h.r.enqueue(query{fn: func() { close(entered); <-release }, done: make(chan struct{})}, false)
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := h.r.ObserveNote(ctx, "none"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(release)
	h.settle()
}

func TestPreparedWriteShutdownDrainsImmutableRecordOnce(t *testing.T) {
	h, _, a, n := preparedFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	h.r.enqueue(query{fn: func() { close(entered); <-release }, done: make(chan struct{})}, false)
	<-entered
	outcomes := make(chan board.OperatorOutcome, 2)
	n.Note.At.Classes = []string{"before"}
	n.Note.At.Stable = map[string]string{"id": "before"}
	n.Note.CropUnavailable = "preserved reason"
	if err := h.r.WritePrepared(context.Background(), a, n, nil, func(v board.OperatorOutcome, err error) { outcomes <- v }); err != nil {
		close(release)
		t.Fatal(err)
	}
	n.Note.At.Classes[0] = "after"
	n.Note.At.Stable["id"] = "after"
	n.Note.At.Box[0] = 999
	closed := make(chan struct{})
	go func() { h.r.Close(); close(closed) }()
	close(release)
	<-closed
	if v := <-outcomes; v != board.OperatorAccepted {
		t.Fatal(v)
	}
	select {
	case v := <-outcomes:
		t.Fatal("duplicate settlement", v)
	default:
	}
	for _, v := range h.r.m.st.Notes {
		if v.Note.ID == n.Note.ID {
			if v.Note.At.Classes[0] != "before" || v.Note.At.Stable["id"] != "before" || v.Note.At.Box[0] == 999 || v.Note.CropUnavailable != "preserved reason" {
				t.Fatalf("queued record mutated: %+v", v.Note)
			}
			return
		}
	}
	t.Fatal("drained note missing")
}
