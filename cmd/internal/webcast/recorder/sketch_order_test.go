package recorder

import (
	"context"
	"errors"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"math"
	"strings"
	"testing"
)

func mirrorSketchBegin() *rec.SketchBegin {
	return &rec.SketchBegin{SketchOperation: rec.SketchOperation{Operation: "begin", Digest: strings.Repeat("a", 64)}, ID: "sk", Author: rec.Author{ID: rec.OperatorID, Color: "#123456"}, Owner: rec.OperatorID, Doc: 1, Tab: "tab1", VP: rec.Viewport{W: 100, H: 100, DPR: 1}}
}

func TestRecorderEmitSketchHistoryAndLateLabel(t *testing.T) {
	h := newHarness(t)
	h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
	h.settle()
	begin := mirrorSketchBegin()
	begin.Tab = "T-A"
	h.r.Send(Emit{Record: begin, Doc: 1})
	h.settle()
	if h.last("sketch-begin") == nil {
		t.Fatal("shell refused sketch begin")
	}
	h.r.Send(Emit{Record: &rec.SketchInterrupt{SketchOperation: rec.SketchOperation{Operation: "interrupt", Sequence: 1, Digest: strings.Repeat("b", 64)}, Sketch: "sk", Reason: rec.SketchInterruptExpiry}, Doc: 1})
	h.r.Send(Pause{})
	h.settle()
	retry := *begin
	retry.SetTime(999)
	h.r.Send(Emit{Record: &retry, Doc: 1})
	h.r.Send(Emit{Record: &rec.SketchLabel{SketchOperation: rec.SketchOperation{Operation: "label", Sequence: 2, Digest: strings.Repeat("c", 64)}, Sketch: "sk", Label: "Saved partial"}, Doc: 0})
	h.settle()
	if h.last("sketch-label") == nil {
		t.Fatal("shell refused late label")
	}
	if stats, err := h.r.Stats(context.Background()); err != nil || len(stats.Dropped) != 0 {
		t.Fatalf("shell dropped legal sketch input: %+v %v", stats, err)
	}
}
func TestMirrorSketchRetryDoesNotRestampChargeOrQueue(t *testing.T) {
	m := newMirror()
	b := rec.NewBuilder("sketch ordering")
	b.Page(rec.SegLive, "http://x/")
	for _, r := range b.Recs {
		if !m.Emit(2, r) {
			t.Fatal(m.Examples)
		}
	}
	begin := mirrorSketchBegin()
	if !m.Emit(2, begin) {
		t.Fatal(m.Examples)
	}
	if begin.Time() != 3 {
		t.Fatal("missing begin fence")
	}
	if !m.Emit(2, &rec.Title{V: "after begin"}) || m.st.T != 4 {
		t.Fatal("missing post-begin fence")
	}
	n, used, reserved := len(m.out), m.budget.Used(), m.budget.Reserved()
	retry := *begin
	retry.SetTime(999)
	if !m.Emit(999, &retry) {
		t.Fatal(m.Examples)
	}
	if retry.Time() != 999 || m.st.T != 4 || len(m.out) != n || m.budget.Used() != used || m.budget.Reserved() != reserved {
		t.Fatal("retry changed timestamp, state, budget or queue")
	}
}
func TestMirrorRejectsSnapshotAndOverflowBeforeEffects(t *testing.T) {
	m := newMirror()
	if !m.Emit(math.MaxInt64, &rec.Head{V: rec.Version, Format: rec.Format, ID: "overflow", Started: "2026-10-11T00:00:00Z"}) {
		t.Fatal(m.Examples)
	}
	for _, r := range []rec.Record{mirrorSketchBegin(), &rec.AnnotationClear{Who: rec.Author{ID: rec.OperatorID}}, &rec.Unknown{Header: rec.Header{K: "sketch-snapshot"}, Raw: []byte(`{"t":0,"k":"sketch-snapshot"}`)}} {
		if m.Emit(math.MaxInt64, r) {
			t.Fatalf("accepted %s", r.Kind())
		}
		if m.st.T != math.MaxInt64 || m.lastKind != "head" || len(m.out) != 1 {
			t.Fatal("refusal changed mirror")
		}
	}
}

// The sink fails only its first sketch write. A mistaken retry would then
// produce a visible extra record, including in the reject-before-write case.
type recorderUncertainSketchSink struct {
	rec.Sink
	accept, failed bool
}

func (s *recorderUncertainSketchSink) Put(r rec.Record) error {
	if r.Kind() == "sketch-begin" && !s.failed {
		s.failed = true
		if s.accept {
			if err := s.Sink.Put(r); err != nil {
				return err
			}
		}
		return errors.New("sketch persistence outcome unknown")
	}
	return s.Sink.Put(r)
}
func TestRecorderSketchSinkUncertaintySurvivesRetry(t *testing.T) {
	for _, accept := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject before write", true: "accept then error"}[accept], func(t *testing.T) {
			h := newHarness(t)
			h.page("A", "http://a/", "L1")
			h.r.Send(Live{Tab: rec.Tab{ID: "T-A"}, Session: "A"})
			h.settle()
			if err := h.r.ask(context.Background(), func() { h.r.sink = &recorderUncertainSketchSink{Sink: h.r.sink, accept: accept} }); err != nil {
				t.Fatal(err)
			}
			begin := mirrorSketchBegin()
			begin.Tab = "T-A"
			h.r.Send(Emit{Record: begin, Doc: 1})
			h.settle()
			before := h.kinds()
			var used int
			var firstErr error
			if err := h.r.ask(context.Background(), func() { used = h.r.m.budget.Used(); firstErr = h.r.err }); err != nil {
				t.Fatal(err)
			}
			if used == 0 || firstErr == nil {
				t.Fatal("sink failure rolled back accepted state or lost uncertainty")
			}
			retry := *begin
			retry.SetTime(999)
			h.r.Send(Emit{Record: &retry, Doc: 1})
			h.settle()
			if h.kinds() != before {
				t.Fatal("uncertain retry attempted another physical record")
			}
			if err := h.r.ask(context.Background(), func() {
				if h.r.err != firstErr || h.r.m.budget.Used() != used {
					t.Error("retry lost uncertainty or charged twice")
				}
			}); err != nil {
				t.Fatal(err)
			}
			if err := h.r.Close(); !errors.Is(err, firstErr) {
				t.Fatalf("Close hid sink uncertainty: %v", err)
			}
		})
	}
}
