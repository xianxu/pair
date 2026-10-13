package recorder

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestClearAnnotationsAcknowledgesOrderedPausedWrite(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	top := h.id("A", h.fc.pages["A"].doc.backend)
	h.r.Send(Emit{Record: &rec.Note{ID: "old", Author: rec.Author{ID: "v"}, At: rec.NoteAt{Node: top}}, Doc: 1})
	h.r.Send(Emit{Record: &rec.Stroke{ID: "old", Author: rec.Author{ID: "v"}, Pts: [][3]float64{{1, 2, 0}}}, Doc: 1})
	h.r.Send(Pause{})
	if err := h.r.ClearAnnotations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if got := h.kinds(); !strings.HasSuffix(got, "stop:pause annotation-clear") {
		t.Fatalf("returned before write: %s", got)
	}
	if err := h.r.ClearAnnotations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if strings.Count(h.kinds(), "annotation-clear") != 1 {
		t.Fatal("duplicate clear")
	}
	recs := h.out.sink.Records()
	files := map[string][]byte{}
	for _, r := range recs {
		line, err := rec.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		name := rec.FileOf(r.Kind())
		files[name] = append(append(files[name], line...), '\n')
	}
	findings, end := rec.Validate(files)
	if len(findings) != 0 || len(end.Notes) != 0 || len(end.Strokes) != 0 {
		t.Fatalf("replay resurrected: %v notes=%d strokes=%d", findings, len(end.Notes), len(end.Strokes))
	}
	if err := h.r.ClearAnnotations(context.Background(), 1); err == nil {
		t.Fatal("old cleanup accepted")
	}
}

func TestCapabilityCleanupBeforeOpenAndCanceledRetry(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := h.r.ClearAnnotations(ctx, 1)
	if !errors.Is(err, context.Canceled) || capability.FailureOutcome(err) != capability.Retryable {
		t.Fatalf("cancellation: %v", err)
	}
	if err := h.r.ClearAnnotations(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if h.out.opened != 0 {
		t.Fatal("cleanup opened recording")
	}
	h.r.Close()
	if err := h.r.ClearAnnotations(context.Background(), 2); err == nil || capability.FailureOutcome(err) != capability.Broken {
		t.Fatalf("closed recorder: %v", err)
	}
}

func TestResetControlAcknowledgesFullReset(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	a, b := rec.Author{ID: "v1"}, rec.Author{ID: "v2"}
	h.r.Send(Emit{Record: &rec.Ctl{Op: rec.CtlRequest, Who: a, Control: rec.Control{Pending: []rec.Author{a}}}})
	h.r.Send(Emit{Record: &rec.Ctl{Op: rec.CtlGrant, Who: a, Control: rec.Control{Driver: &a}}})
	h.r.Send(Emit{Record: &rec.Ctl{Op: rec.CtlRequest, Who: b, Control: rec.Control{Driver: &a, Pending: []rec.Author{b}}}})
	if err := h.r.ResetControl(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	c := h.last("ctl").(*rec.Ctl)
	if c.Op != rec.CtlReset || !c.Control.Equal(rec.Control{}) {
		t.Fatalf("incomplete reset: %+v", c)
	}
	before := len(h.out.sink.Records())
	if err := h.r.ResetControl(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if len(h.out.sink.Records()) != before {
		t.Fatal("reset retry wrote twice")
	}
}

// cleanupSink retains the actual accepted stream while delaying the clear's
// acknowledgement or failing after acceptance, like a partial tee write.
type cleanupSink struct {
	rec.Sink
	entered chan struct{}
	release chan struct{}
	fail    error
}

func (s *cleanupSink) Put(r rec.Record) error {
	if r.Kind() != "annotation-clear" {
		return s.Sink.Put(r)
	}
	if s.entered != nil {
		close(s.entered)
		<-s.release
		s.entered = nil
	}
	if err := s.Sink.Put(r); err != nil {
		return err
	}
	return s.fail
}

func TestClearLostAcknowledgementReconcilesWithoutDuplicate(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	sink := &cleanupSink{Sink: &h.out.sink, entered: make(chan struct{}), release: make(chan struct{})}
	entered := sink.entered
	if err := h.r.ask(context.Background(), func() { h.r.sink = sink }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- h.r.ClearAnnotations(ctx, 2) }()
	<-entered
	cancel()
	err := <-result
	close(sink.release)
	if !errors.Is(err, context.Canceled) || capability.FailureOutcome(err) != capability.Retryable {
		t.Fatalf("lost ack: %v", err)
	}
	if err := h.r.ClearAnnotations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if strings.Count(h.kinds(), "annotation-clear") != 1 {
		t.Fatal("lost ack duplicated effect")
	}
}

func TestClearSinkFailureIsTerminalEvenIfItAcceptedRecord(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	want := &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("disk write failed")}
	sink := &cleanupSink{Sink: &h.out.sink, fail: want}
	if err := h.r.ask(context.Background(), func() { h.r.sink = sink }); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		err := h.r.ClearAnnotations(context.Background(), 2)
		if !errors.Is(err, want) || capability.FailureOutcome(err) != capability.Broken {
			t.Fatalf("partial write: %v", err)
		}
	}
	if strings.Count(h.kinds(), "annotation-clear") != 1 {
		t.Fatal("terminal failure replayed effect")
	}
}
