package board

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestAnnotationsDefaultOff(t *testing.T) {
	b := New(notes.Structural{}, func(recording.Record, int) error { t.Fatal("unexpected emit"); return nil })
	if err := b.Stroke(StrokeIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Stroke: %v", err)
	}
	if _, err := b.Note(NoteIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Note: %v", err)
	}
	if err := b.SetState(StateIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("SetState: %v", err)
	}
}

func TestOperatorSketchAdmissionIndependentOfViewerAnnotation(t *testing.T) {
	b := New(notes.Structural{}, nil).WithAnnotationCleanup(func(context.Context, uint64) error { return nil })
	generation := b.AnnotationCapability().Generation
	a, err := b.AdmitOperatorSketchGeneration(generation)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Check(); err != nil {
		t.Fatalf("console admission with viewers off: %v", err)
	}
	if _, err = b.AdmitAnnotationGeneration(generation); !errors.Is(err, ErrDisabled) {
		t.Fatalf("viewer admitted while off: %v", err)
	}
	if err = a.Stroke(StrokeIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("sketch admission granted ordinary annotation: %v", err)
	}
	if err = b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err = a.Check(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("old console generation survived toggle: %v", err)
	}
	if _, err = b.AdmitOperatorSketchGeneration(generation); !errors.Is(err, ErrDisabled) {
		t.Fatalf("stale ingress generation: %v", err)
	}
	fresh, err := b.AdmitOperatorSketchGeneration(b.AnnotationCapability().Generation)
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	if err = fresh.Check(); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed board admitted console: %v", err)
	}
	if _, err = b.AdmitOperatorSketchGeneration(b.AnnotationCapability().Generation); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed ingress: %v", err)
	}
}

func TestAnnotationAdmissionCannotSurviveOffOn(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	a, err := r.bd.AdmitAnnotation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.note(r.p1.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err = r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.Live()) != 0 {
		t.Fatal("notes survived clear")
	}
	if err = r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err = a.Stroke(StrokeIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("old Stroke: %v", err)
	}
	if _, err = a.Note(NoteIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("old Note: %v", err)
	}
	if err = a.SetState(StateIn{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("old state: %v", err)
	}
}

func TestDisableOrdersClearAfterInFlightEmit(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	entered, release := make(chan struct{}), make(chan struct{})
	r.before = func() { close(entered); <-release }
	emitted := make(chan error, 1)
	go func() { _, err := r.note(r.p1.ID, 0); emitted <- err }()
	<-entered
	updates, stop := r.bd.SubscribeAnnotationCapability()
	defer stop()
	<-updates
	done := make(chan error, 1)
	go func() { done <- r.bd.SetAnnotationAllowed(context.Background(), false) }()
	if s := <-updates; s.Phase != capability.Disabling {
		t.Fatalf("phase: %+v", s)
	}
	if _, err := r.bd.AdmitAnnotation(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("admission: %v", err)
	}
	close(release)
	if err := <-emitted; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	recs := r.mem.Records()
	if recs[len(recs)-1].Kind() != "annotation-clear" {
		t.Fatal("clear did not follow mutation")
	}
	if len(r.bd.Live()) != 0 || len(r.bd.pending) != 0 {
		t.Fatal("annotations survived clear")
	}
}

func TestAnnotationCleanupFailureRequiresExplicitRetry(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	var ids []uint64
	r.bd.WithAnnotationCleanup(func(ctx context.Context, id uint64) error {
		ids = append(ids, id)
		if len(ids) == 1 {
			return &capability.Failure{Outcome: capability.Retryable, Cause: errors.New("ack lost")}
		}
		return r.w.ClearAnnotations(ctx, id)
	})
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err == nil {
		t.Fatal("failure hidden")
	}
	s := r.bd.AnnotationCapability()
	if s.Outcome != capability.Retryable || s.Problem == "" {
		t.Fatalf("snapshot: %+v", s)
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), true); err == nil {
		t.Fatal("enabled through failure")
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err == nil {
		t.Fatal("duplicate disable retried")
	}
	if len(ids) != 1 {
		t.Fatal("implicit retry")
	}
	if err := r.bd.RetryAnnotations(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != ids[1] {
		t.Fatalf("cleanup identities: %v", ids)
	}
	s = r.bd.AnnotationCapability()
	if s.Phase != capability.Off || s.Problem != "" {
		t.Fatalf("snapshot: %+v", s)
	}
}

func TestAnnotationMissingCleanupIsTerminal(t *testing.T) {
	b := New(notes.Structural{}, nil)
	if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if err := b.SetAnnotationAllowed(context.Background(), false); err == nil {
		t.Fatal("missing cleanup succeeded")
	}
	if b.AnnotationCapability().Outcome != capability.Broken {
		t.Fatal("missing cleanup was not terminal")
	}
	if err := b.RetryAnnotations(context.Background()); err == nil {
		t.Fatal("terminal retry accepted")
	}
}

func TestWriterAnnotationClearDedupAndTime(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	if err := r.w.ClearAnnotations(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if err := r.w.ClearAnnotations(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if err := r.w.ClearAnnotations(context.Background(), 6); err == nil {
		t.Fatal("older clear accepted")
	}
	recs := r.mem.Records()
	if len(r.kinds("annotation-clear")) != 1 || recs[len(recs)-1].Time() <= recs[len(recs)-2].Time() {
		t.Fatal("clear is duplicated or not strictly later")
	}
}

func TestAnnotationRunningCleanupIsSingleFlightAndCloseWins(t *testing.T) {
	r := newRig(t)
	entered, release := make(chan struct{}), make(chan struct{})
	r.bd.WithAnnotationCleanup(func(context.Context, uint64) error { close(entered); <-release; return nil })
	done := make(chan error, 1)
	go func() { done <- r.bd.SetAnnotationAllowed(context.Background(), false) }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.bd.RetryAnnotations(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("joining attempt: %v", err)
	}
	updates, stop := r.bd.SubscribeAnnotationCapability()
	defer stop()
	<-updates
	if err := r.bd.Close(); err != nil {
		t.Fatal(err)
	}
	if s := <-updates; s.Phase != capability.Ended {
		t.Fatalf("close state: %+v", s)
	}
	if _, ok := <-updates; ok {
		t.Fatal("subscription not closed")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if r.bd.AnnotationCapability().Phase != capability.Ended {
		t.Fatal("late cleanup changed Ended")
	}
}

func TestAnnotationClearDropsPendingNotes(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.bd.emit = func(recording.Record, int) error { return nil }
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.pending) != 1 {
		t.Fatal("expected pending note")
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(r.bd.pending) != 0 {
		t.Fatal("clear retained pending note")
	}
}

func TestResolveSelectionRefusesAnnotationsOff(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Fatal(err)
	}
	selection, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	r.bd.WithAnnotationCleanup(func(context.Context, uint64) error { return errors.New("broken sink") })
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err == nil {
		t.Fatal("missing failure")
	}
	if err := r.bd.ResolveSelection(context.Background(), selection); !errors.Is(err, ErrDisabled) {
		t.Fatalf("resolve with admission closed: %v", err)
	}
}

func TestWriterClearBeforeHeadAndWhilePaused(t *testing.T) {
	r := newRig(t)
	if err := r.w.ClearAnnotations(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if len(r.mem.Records()) != 0 {
		t.Fatal("clear wrote before head")
	}
	r.page(recording.SegLive)
	r.add(&recording.Stop{Why: recording.StopPause})
	if err := r.w.ClearAnnotations(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if len(r.kinds("annotation-clear")) != 1 {
		t.Fatal("paused clear absent")
	}
}

type annotationFailSink struct{ err error }

func (s annotationFailSink) Put(recording.Record) error { return s.err }
func (s annotationFailSink) Close() error               { return nil }

func TestWriterClearSinkFailureRemainsTerminal(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	r.w.sink = annotationFailSink{errors.New("partial sink write")}
	err := r.w.ClearAnnotations(context.Background(), 1)
	if err == nil || capability.FailureOutcome(err) != capability.Broken {
		t.Fatalf("sink failure: %v", err)
	}
	r.w.sink = r.mem
	if err := r.w.ClearAnnotations(context.Background(), 1); err == nil {
		t.Fatal("uncertain sink outcome retried")
	}
}

func TestAnnotationAdmissionUsesOriginatingGeneration(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	g := r.bd.AnnotationCapability().Generation
	if _, err := r.bd.AdmitAnnotationGeneration(g + 1); !errors.Is(err, ErrDisabled) {
		t.Fatalf("future generation: %v", err)
	}
	a, err := r.bd.AdmitAnnotationGeneration(g)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := r.bd.AdmitAnnotationGeneration(r.bd.AnnotationCapability().Generation); !errors.Is(err, ErrDisabled) {
		t.Fatalf("off authority: %v", err)
	}
	if err := r.bd.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := r.bd.AdmitAnnotationGeneration(g); !errors.Is(err, ErrDisabled) {
		t.Fatalf("retired generation: %v", err)
	}
	if err := a.Check(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("retired permit: %v", err)
	}
	if _, err := r.bd.AdmitAnnotationGeneration(r.bd.AnnotationCapability().Generation); err != nil {
		t.Fatal(err)
	}
}
