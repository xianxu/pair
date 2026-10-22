package recorder

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func sketchActorFixture(t *testing.T) (*harness, *rec.Hub, board.SketchWrite) {
	t.Helper()
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	hub := rec.NewHub(0)
	for _, r := range h.out.sink.Records() {
		if err := hub.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := h.r.ask(context.Background(), func() { h.r.sink = rec.Tee(h.r.sink, hub) }); err != nil {
		t.Fatal(err)
	}
	begin := mirrorSketchBegin()
	begin.Tab = "T-A"
	in := board.SketchWrite{Record: begin, Document: 1, Check: func() error { return nil }, Source: func(b *rec.SketchBegin) (*rec.SketchSource, error) {
		return hub.CandidateSketchSource(b, rec.SketchSourcePrefixLimit, rec.SketchSourceStateLimit)
	}}
	return h, hub, in
}
func TestSketchWriteActorSourceAndReceiptBarrier(t *testing.T) {
	h, _, in := sketchActorFixture(t)
	var accepted atomic.Int32
	in.Accepted = func() { accepted.Add(1) }
	out, err := h.r.WriteSketch(context.Background(), in)
	if err != nil || out.Outcome != board.SketchAccepted || out.Source == nil || accepted.Load() != 1 {
		t.Fatalf("write %+v %v", out, err)
	}
	observation, err := h.r.ObserveSketch(context.Background(), "sk", "begin")
	if err != nil || !observation.Found || observation.Metadata.NextSequence != 1 {
		t.Fatalf("observe %+v %v", observation, err)
	}
	begin := out.Source.Begin()
	if begin.Time() != observation.Metadata.T {
		t.Fatal("source differs from accepted time")
	}
	in.Check = func() error { return board.ErrDisabled }
	in.Source = func(*rec.SketchBegin) (*rec.SketchSource, error) {
		t.Fatal("duplicate rebuilt original source")
		return nil, nil
	}
	before := h.kinds()
	if got, err := h.r.WriteSketch(context.Background(), in); err != nil || got.Outcome != board.SketchAccepted || h.kinds() != before || accepted.Load() != 1 {
		t.Fatalf("retry %+v %v", got, err)
	}
	if got, err := h.r.ObserveSketch(context.Background(), "sk", "missing"); err != nil || got.Found {
		t.Fatalf("absent %+v %v", got, err)
	}
}
func TestSketchWriteActorSourceFailureAndRevocationAreAtomic(t *testing.T) {
	h, _, in := sketchActorFixture(t)
	var beforeTime int64
	var beforeKind string
	h.r.ask(context.Background(), func() { beforeTime = h.r.m.st.T; beforeKind = h.r.m.lastKind })
	before := h.kinds()
	in.Source = func(*rec.SketchBegin) (*rec.SketchSource, error) { return nil, rec.ErrSketchSourceStateLimit }
	out, err := h.r.WriteSketch(context.Background(), in)
	if err == nil || out.Outcome != board.SketchRefused || h.kinds() != before {
		t.Fatalf("refusal %+v %v", out, err)
	}
	h.r.ask(context.Background(), func() {
		if h.r.m.st.T != beforeTime || h.r.m.lastKind != beforeKind || h.r.m.budget.Used() != 0 {
			t.Error("preflight advanced actor")
		}
	})
	if in.Record.Time() != 0 {
		t.Fatal("caller record was restamped")
	}
}

type sketchActorSink struct {
	rec.Sink
	entered, allow chan struct{}
	fail, accept   bool
}

func (s *sketchActorSink) Put(r rec.Record) error {
	if r.Kind() != "sketch-begin" {
		return s.Sink.Put(r)
	}
	if s.entered != nil {
		close(s.entered)
		<-s.allow
	}
	if s.accept || !s.fail {
		if err := s.Sink.Put(r); err != nil {
			return err
		}
	}
	if s.fail {
		return errors.New("sketch sink outcome unknown")
	}
	return nil
}
func TestSketchWriteActorUnknownRetainsSourceAndNeverProvesAbsence(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject before error", true: "accept then error"}[accept], func(t *testing.T) {
			h, _, in := sketchActorFixture(t)
			h.r.ask(context.Background(), func() { h.r.sink = &sketchActorSink{Sink: h.r.sink, fail: true, accept: accept} })
			in.Accepted = func() { t.Error("unknown write renewed source lease") }
			out, err := h.r.WriteSketch(context.Background(), in)
			if err == nil || out.Outcome != board.SketchUnknown || out.Source == nil {
				t.Fatalf("unknown %+v %v", out, err)
			}
			before := h.kinds()
			if got, err := h.r.WriteSketch(context.Background(), in); err == nil || got.Outcome != board.SketchUnknown || h.kinds() != before {
				t.Fatalf("unknown retry %+v %v", got, err)
			}
			if got, err := h.r.ObserveSketch(context.Background(), "sk", "begin"); err == nil || !got.Found {
				t.Fatalf("unknown receipt %+v %v", got, err)
			}
			if _, err := h.r.ObserveSketch(context.Background(), "sk", "missing"); err == nil {
				t.Fatal("failed actor proved absence")
			}
		})
	}
}
func TestSketchWriteActorQueuedCancelAndStartedJoin(t *testing.T) {
	h, _, in := sketchActorFixture(t)
	held := &sketchActorSink{entered: make(chan struct{}), allow: make(chan struct{})}
	h.r.ask(context.Background(), func() { held.Sink = h.r.sink; h.r.sink = held })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan board.SketchWriteResult, 1)
	go func() { out, _ := h.r.WriteSketch(ctx, in); done <- out }()
	<-held.entered
	queuedCtx, stop := context.WithCancel(context.Background())
	stop()
	queued := make(chan error, 1)
	go func() { _, err := h.r.WriteSketch(queuedCtx, in); queued <- err }()
	select {
	case err := <-queued:
		if err == nil {
			t.Fatal("canceled queued work accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("canceled queued work blocked")
	}
	cancel()
	select {
	case <-done:
		t.Fatal("started actor effect abandoned")
	default:
	}
	close(held.allow)
	if out := <-done; out.Outcome != board.SketchAccepted {
		t.Fatal(out)
	}
	observation, err := h.r.ObserveSketch(context.Background(), "sk", "begin")
	if err != nil || !observation.Found {
		t.Fatal("started result was not retained", err)
	}
}
func TestSketchWriteActorPausedInternalInterruptAndLabel(t *testing.T) {
	h, _, in := sketchActorFixture(t)
	if _, err := h.r.WriteSketch(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	h.r.Send(Pause{})
	h.settle()
	for _, r := range []rec.Record{&rec.SketchInterrupt{Sketch: "sk", Reason: rec.SketchInterruptExpiry, SketchOperation: rec.SketchOperation{Operation: "expire", Sequence: 1, Digest: strings.Repeat("b", 64)}}, &rec.SketchLabel{Sketch: "sk", Label: "Partial", SketchOperation: rec.SketchOperation{Operation: "label", Sequence: 2, Digest: strings.Repeat("c", 64)}}} {
		out, err := h.r.WriteSketch(context.Background(), board.SketchWrite{Record: r, Document: 0, Check: func() error { return nil }})
		if err != nil || out.Outcome != board.SketchAccepted {
			t.Fatalf("paused metadata %+v %v", out, err)
		}
	}
	h.r.Close()
	if _, err := h.r.ObserveSketch(context.Background(), "sk", "missing"); err == nil {
		t.Fatal("closed actor proved absence")
	}
}
