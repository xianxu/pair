package board

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func sketchAckRig(t *testing.T) (*Writer, *rec.Hub, *rec.MemSink) {
	t.Helper()
	hub := rec.NewHub(0)
	mem := &rec.MemSink{}
	w := sketchWriterFixture(t, rec.Tee(mem, hub))
	return w, hub, mem
}
func sketchAckBegin(hub *rec.Hub) SketchWrite {
	return SketchWrite{Record: writerSketchBegin(), Document: 1, Check: func() error { return nil }, Source: func(b *rec.SketchBegin) (*rec.SketchSource, error) {
		return hub.CandidateSketchSource(b, rec.SketchSourcePrefixLimit, rec.SketchSourceStateLimit)
	}}
}
func TestSketchWritePreflightRefusesWithoutEffects(t *testing.T) {
	w, hub, mem := sketchAckRig(t)
	in := sketchAckBegin(hub)
	before, n := w.st.T, len(mem.Records())
	in.Source = func(b *rec.SketchBegin) (*rec.SketchSource, error) {
		if b.Time() <= before {
			t.Fatal("source received unstamped begin")
		}
		return nil, rec.ErrSketchSourcePrefixLimit
	}
	result, err := w.WriteSketch(context.Background(), in)
	if err == nil || result.Outcome != SketchRefused || w.st.T != before || len(mem.Records()) != n || w.budget.Used() != 0 {
		t.Fatalf("preflight changed writer: %+v %v", result, err)
	}
	if in.Record.Time() != 0 {
		t.Fatal("writer mutated caller record")
	}
	good := sketchAckBegin(hub)
	revoked := false
	good.Check = func() error {
		if revoked {
			return ErrDisabled
		}
		return nil
	}
	good.Source = func(b *rec.SketchBegin) (*rec.SketchSource, error) {
		source, err := hub.CandidateSketchSource(b, rec.SketchSourcePrefixLimit, rec.SketchSourceStateLimit)
		revoked = true
		return source, err
	}
	if r, e := w.WriteSketch(context.Background(), good); e == nil || r.Outcome != SketchRefused || len(mem.Records()) != n {
		t.Fatal("revocation during source escaped final check")
	}
}
func TestSketchWriteAcceptedSourceReceiptAndDuplicate(t *testing.T) {
	w, hub, mem := sketchAckRig(t)
	in := sketchAckBegin(hub)
	accepted := 0
	in.Accepted = func() { accepted++ }
	out, err := w.WriteSketch(context.Background(), in)
	if err != nil || out.Outcome != SketchAccepted || out.Source == nil || accepted != 1 {
		t.Fatalf("write %+v %v", out, err)
	}
	begin := out.Source.Begin()
	if begin.Time() != 3 {
		t.Fatal("wrong source fence")
	}
	obs, err := w.ObserveSketch(context.Background(), "sk", "begin")
	if err != nil || !obs.Found || obs.Metadata.NextSequence != 1 {
		t.Fatalf("receipt %+v %v", obs, err)
	}
	n := len(mem.Records())
	in.Check = func() error { return ErrDisabled }
	in.Source = func(*rec.SketchBegin) (*rec.SketchSource, error) { t.Fatal("retry rebuilt source"); return nil, nil }
	if got, err := w.WriteSketch(context.Background(), in); err != nil || got.Outcome != SketchAccepted || accepted != 1 || len(mem.Records()) != n {
		t.Fatalf("retry %+v %v", got, err)
	}
	if got, err := w.ObserveSketch(context.Background(), "sk", "absent"); err != nil || got.Found {
		t.Fatalf("absence %+v %v", got, err)
	}
}

type sketchAckSink struct {
	rec.Sink
	entered, allow chan struct{}
	fail, accept   bool
}

func (s *sketchAckSink) Put(r rec.Record) error {
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
func TestSketchWriteQueuedCancelAndStartedJoin(t *testing.T) {
	w, hub, mem := sketchAckRig(t)
	held := &sketchAckSink{Sink: w.sink, entered: make(chan struct{}), allow: make(chan struct{})}
	w.sink = held
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan SketchWriteResult, 1)
	go func() { v, _ := w.WriteSketch(ctx, sketchAckBegin(hub)); first <- v }()
	<-held.entered
	canceled, stop := context.WithCancel(context.Background())
	stop()
	queued := make(chan error, 1)
	go func() { _, err := w.WriteSketch(canceled, sketchAckBegin(hub)); queued <- err }()
	select {
	case err := <-queued:
		if err == nil {
			t.Fatal("queued cancellation accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("queued cancellation blocked behind sink")
	}
	cancel()
	select {
	case <-first:
		t.Fatal("started effect abandoned on cancellation")
	default:
	}
	close(held.allow)
	if got := <-first; got.Outcome != SketchAccepted {
		t.Fatal(got)
	}
	if len(mem.Records()) != 4 {
		t.Fatalf("physical record count %d", len(mem.Records()))
	}
}
func TestSketchWriteUnknownSourceReceiptAndNoFalseAbsence(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject then fail", true: "accept then fail"}[accept], func(t *testing.T) {
			w, hub, mem := sketchAckRig(t)
			w.sink = &sketchAckSink{Sink: w.sink, fail: true, accept: accept}
			in := sketchAckBegin(hub)
			in.Accepted = func() { t.Error("unknown write renewed lease") }
			out, err := w.WriteSketch(context.Background(), in)
			if err == nil || out.Outcome != SketchUnknown || out.Source == nil {
				t.Fatalf("unknown %+v %v", out, err)
			}
			n := len(mem.Records())
			if got, e := w.WriteSketch(context.Background(), in); e == nil || got.Outcome != SketchUnknown || len(mem.Records()) != n {
				t.Fatalf("unknown retry %+v %v", got, e)
			}
			obs, e := w.ObserveSketch(context.Background(), "sk", "begin")
			if e == nil || !obs.Found {
				t.Fatalf("unknown observation %+v %v", obs, e)
			}
			if _, e := w.ObserveSketch(context.Background(), "sk", "absent"); e == nil {
				t.Fatal("failed writer proved absence")
			}
		})
	}
}
func TestSketchWritePausedInterruptAndHistoricalLabel(t *testing.T) {
	w, hub, _ := sketchAckRig(t)
	if _, err := w.WriteSketch(context.Background(), sketchAckBegin(hub)); err != nil {
		t.Fatal(err)
	}
	if err := w.Put(&rec.Stop{Why: rec.StopPause}); err != nil {
		t.Fatal(err)
	}
	for i, r := range []rec.Record{&rec.SketchInterrupt{Sketch: "sk", Reason: rec.SketchInterruptExpiry, SketchOperation: rec.SketchOperation{Operation: "expire", Sequence: 1, Digest: strings.Repeat("b", 64)}}, &rec.SketchLabel{Sketch: "sk", Label: "Partial", SketchOperation: rec.SketchOperation{Operation: "label", Sequence: 2, Digest: strings.Repeat("c", 64)}}} {
		if out, err := w.WriteSketch(context.Background(), SketchWrite{Record: r, Document: 0, Check: func() error { return nil }}); err != nil || out.Outcome != SketchAccepted {
			t.Fatalf("paused metadata %d: %+v %v", i, out, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.ObserveSketch(context.Background(), "sk", "missing"); err == nil {
		t.Fatal("closed writer proved absence")
	}
}
