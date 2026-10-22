package board

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func sketchWriterFixture(t *testing.T, sink rec.Sink) *Writer {
	t.Helper()
	w := NewWriter(sink, func() int64 { return 2 })
	b := rec.NewBuilder("sketch ordering")
	b.At(2).Page(rec.SegLive, "http://x/")
	for _, r := range b.Recs {
		if err := w.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	return w
}
func writerSketchBegin() *rec.SketchBegin {
	return &rec.SketchBegin{SketchOperation: rec.SketchOperation{Operation: "begin", Digest: strings.Repeat("a", 64)}, ID: "sk", Author: rec.Author{ID: rec.OperatorID, Color: "#123456"}, Owner: rec.OperatorID, Doc: 1, Tab: "tab1", VP: rec.Viewport{W: 100, H: 100, DPR: 1}}
}
func TestWriterSketchRetryDoesNotRestampChargeOrWrite(t *testing.T) {
	sink := &rec.MemSink{}
	w := sketchWriterFixture(t, sink)
	begin := writerSketchBegin()
	if err := w.Emit(begin, 1); err != nil {
		t.Fatal(err)
	}
	if begin.Time() != 3 {
		t.Fatalf("begin fence %d", begin.Time())
	}
	if err := w.Put(&rec.Title{V: "after begin"}); err != nil {
		t.Fatal(err)
	}
	if w.st.T != 4 {
		t.Fatal("missing post-begin fence")
	}
	n, used, reserved := len(sink.Records()), w.budget.Used(), w.budget.Reserved()
	retry := *begin
	retry.SetTime(999)
	if err := w.Emit(&retry, 1); err != nil {
		t.Fatal(err)
	}
	if retry.Time() != 999 || w.st.T != 4 || len(sink.Records()) != n || w.budget.Used() != used || w.budget.Reserved() != reserved {
		t.Fatal("retry changed timestamp, state, bytes or sink")
	}
}
func TestWriterRejectsSnapshotAndOverflowBeforeEffects(t *testing.T) {
	sink := &rec.MemSink{}
	w := NewWriter(sink, nil)
	if err := w.Put(&rec.Head{Header: rec.Header{T: math.MaxInt64}, V: rec.Version, Format: rec.Format, ID: "overflow", Started: "2026-10-11T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	for _, r := range []rec.Record{writerSketchBegin(), &rec.Unknown{Header: rec.Header{K: "sketch-snapshot"}, Raw: []byte(`{"t":0,"k":"sketch-snapshot"}`)}} {
		if w.Put(r) == nil {
			t.Fatalf("accepted %s", r.Kind())
		}
		if w.st.T != math.MaxInt64 || w.lastKind != "head" || len(sink.Records()) != 1 {
			t.Fatal("refusal changed writer")
		}
	}
	if w.ClearAnnotations(context.Background(), 1) == nil {
		t.Fatal("overflowing clear accepted")
	}
	if len(sink.Records()) != 1 {
		t.Fatal("overflowing clear emitted")
	}
}

type uncertainSketchSink struct {
	rec.MemSink
	fail   bool
	reject bool
}

func (s *uncertainSketchSink) Put(r rec.Record) error {
	if s.fail && s.reject {
		return errors.New("write outcome unknown")
	}
	if err := s.MemSink.Put(r); err != nil {
		return err
	}
	if s.fail {
		return errors.New("ack lost")
	}
	return nil
}
func TestWriterSketchSinkErrorRetainsReceiptAndBudget(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted then error", true: "refused then error"}[reject], func(t *testing.T) {
			sink := &uncertainSketchSink{}
			w := sketchWriterFixture(t, sink)
			sink.fail = true
			sink.reject = reject
			begin := writerSketchBegin()
			if w.Put(begin) == nil {
				t.Fatal("missing sink failure")
			}
			n, used := len(sink.Records()), w.budget.Used()
			if used == 0 {
				t.Fatal("unknown outcome rolled back budget")
			}
			retry := *begin
			retry.SetTime(999)
			if err := w.Put(&retry); err == nil {
				t.Fatal("unknown persistence became successful retry")
			}
			if len(sink.Records()) != n || w.budget.Used() != used {
				t.Fatal("unknown outcome retry wrote or charged twice")
			}
			if err := w.Emit(&retry, 1); err == nil {
				t.Fatal("Emit hid unknown persistence")
			}
		})
	}
}

func TestWriterSketchHistoricalRetryAndLateLabel(t *testing.T) {
	sink := &rec.MemSink{}
	w := sketchWriterFixture(t, sink)
	begin := writerSketchBegin()
	if err := w.Emit(begin, 1); err != nil {
		t.Fatal(err)
	}
	interrupt := &rec.SketchInterrupt{SketchOperation: rec.SketchOperation{Operation: "interrupt", Sequence: 1, Digest: strings.Repeat("b", 64)}, Sketch: "sk", Reason: rec.SketchInterruptExpiry}
	if err := w.Put(interrupt); err != nil {
		t.Fatal(err)
	}
	if err := w.Put(&rec.Stop{Why: rec.StopPause}); err != nil {
		t.Fatal(err)
	}
	n := len(sink.Records())
	retry := *begin
	if err := w.Emit(&retry, 1); err != nil {
		t.Fatalf("historical receipt retry: %v", err)
	}
	if len(sink.Records()) != n {
		t.Fatal("historical retry emitted")
	}
	label := &rec.SketchLabel{SketchOperation: rec.SketchOperation{Operation: "label", Sequence: 2, Digest: strings.Repeat("c", 64)}, Sketch: "sk", Label: "Saved partial"}
	if err := w.Emit(label, 0); err != nil {
		t.Fatalf("historical label: %v", err)
	}
	if w.budget.Reserved() != 0 {
		t.Fatal("late label retained reservation")
	}
}
