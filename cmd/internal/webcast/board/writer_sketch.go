package board

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"sync"
)

// writerMutex gives canceled queued reference operations a definite refusal
// without starting background workers behind a blocked stateful sink.
type writerMutex struct {
	once  sync.Once
	token chan struct{}
}

func (m *writerMutex) lockContext(ctx context.Context) error {
	m.once.Do(func() { m.token = make(chan struct{}, 1) })
	select {
	case m.token <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (m *writerMutex) Lock()   { _ = m.lockContext(context.Background()) }
func (m *writerMutex) Unlock() { <-m.token }

// SketchWrite owns one prepared event. Check is a final lock-free authority
// check; Source runs only for a stamped prospective begin before any mutation.
// Callbacks must not synchronously reenter the ordered writer.
type SketchWrite struct {
	Record   recording.Record
	Document int
	Check    func() error
	Source   func(*recording.SketchBegin) (*recording.SketchSource, error)
	// Accepted is a leaf callback at the successful persistence boundary,
	// before the writer advances or replies. Retries never renew a lease.
	Accepted func()
}

type SketchWriteResult struct {
	Outcome SketchOutcome
	Source  *recording.SketchSource
	Err     error
}

// SketchObservation is evidence behind a healthy writer barrier. A receipt
// proves durable acceptance only when ObserveSketch also returns no error.
type SketchObservation struct {
	Receipt  recording.SketchReceipt
	Found    bool
	Metadata recording.SketchMetadata
	Tops     recording.SketchStackTops
}

// Clone owns the event before queueing, preserving nested wire members.
func (in SketchWrite) Clone() (SketchWrite, error) {
	if in.Record == nil || recording.SketchReceiptKey(in.Record) == "" {
		return in, ErrShape
	}
	line, err := recording.Encode(in.Record)
	if err != nil {
		return in, err
	}
	in.Record, err = recording.Decode(line)
	return in, err
}

// Preflight is shared by the reference writer and recorder actor. It stamps
// only the privately owned event; no reducer, clock, budget or sink is changed.
// An exact receipt bypasses new drawing checks without creating another effect.
func (in SketchWrite) Preflight(st *recording.State, lastKind string, now int64) (SketchWriteResult, bool, error) {
	out := SketchWriteResult{Outcome: SketchRefused}
	duplicate, err := st.Sketches.Reconcile(in.Record)
	if err != nil {
		return out, false, err
	}
	if duplicate {
		return SketchWriteResult{Outcome: SketchAccepted}, true, nil
	}
	if in.Check == nil {
		return out, false, ErrDisabled
	}
	if err = in.Check(); err != nil {
		return out, false, err
	}
	metadata := false
	switch in.Record.(type) {
	case *recording.SketchLabel, *recording.SketchInterrupt:
		metadata = in.Document == 0
	}
	if !metadata && !st.OnDoc(in.Document) {
		if st.Paused() && st.DocSeg == in.Document {
			return out, false, ErrPaused
		}
		return out, false, ErrStale
	}
	t, err := recording.OrderedTime(now, st.T, lastKind, in.Record.Kind())
	if err != nil {
		return out, false, err
	}
	in.Record.SetTime(t)
	if begin, ok := in.Record.(*recording.SketchBegin); ok {
		if in.Source == nil {
			return out, false, ErrNotLive
		}
		source, err := in.Source(begin)
		if err != nil {
			return out, false, err
		}
		if source == nil {
			return out, false, ErrShape
		}
		candidate := source.Begin()
		want, e := recording.SketchSemantic(begin)
		if e != nil {
			return out, false, e
		}
		got, e := recording.SketchSemantic(&candidate)
		if e != nil {
			return out, false, e
		}
		if candidate.Time() != begin.Time() || want != got {
			return out, false, errors.New("sketch source does not match ordered begin")
		}
		out.Source = source
	}
	if err = in.Check(); err != nil {
		return SketchWriteResult{Outcome: SketchRefused}, false, err
	}
	return out, false, nil
}

func sketchWriteFailure(out SketchWriteResult, err error) (SketchWriteResult, error) {
	out.Err = err
	return out, err
}

// WriteSketch serializes preflight, persistence and lease renewal. Cancellation
// refuses work waiting for the lock; an effect already writing owns its result.
func (w *Writer) WriteSketch(ctx context.Context, in SketchWrite) (SketchWriteResult, error) {
	refused := SketchWriteResult{Outcome: SketchRefused}
	if err := ctx.Err(); err != nil {
		return sketchWriteFailure(refused, err)
	}
	owned, err := in.Clone()
	if err != nil {
		return sketchWriteFailure(refused, err)
	}
	in = owned
	if err = w.mu.lockContext(ctx); err != nil {
		return sketchWriteFailure(refused, err)
	}
	defer w.mu.Unlock()
	if err = ctx.Err(); err != nil {
		return sketchWriteFailure(refused, err)
	}
	if w.closed {
		return sketchWriteFailure(refused, ErrClosed)
	}
	if w.failure != nil {
		duplicate, e := w.st.Sketches.Reconcile(in.Record)
		if e != nil {
			return sketchWriteFailure(refused, e)
		}
		if duplicate {
			return sketchWriteFailure(SketchWriteResult{Outcome: SketchUnknown}, w.failure)
		}
		return sketchWriteFailure(refused, w.failure)
	}
	now := w.st.T
	if w.now != nil {
		now = max(now, w.now())
	}
	out, duplicate, err := in.Preflight(w.st, w.lastKind, now)
	if err != nil {
		return sketchWriteFailure(out, err)
	}
	if duplicate {
		return out, nil
	}
	if err = ctx.Err(); err != nil {
		return sketchWriteFailure(refused, err)
	}
	if err = w.writeReconciledAt(in.Record, in.Record.Time()); err != nil {
		accepted, _ := w.st.Sketches.Reconcile(in.Record)
		if accepted {
			out.Outcome = SketchUnknown
		} else {
			out.Source = nil
		}
		return sketchWriteFailure(out, err)
	}
	out.Outcome = SketchAccepted
	if in.Accepted != nil {
		in.Accepted()
	}
	return out, nil
}

// SketchObservationOf reads only bounded receipt/metadata/stack values, never
// copies accumulated pressure paths. The caller owns the state serialization.
func SketchObservationOf(st *recording.State, sketch, operation string) SketchObservation {
	r, found := st.Sketches.Receipt(sketch, operation)
	m, _ := st.Sketches.Metadata(sketch)
	tops, _ := st.Sketches.StackTops(sketch)
	return SketchObservation{Receipt: r, Found: found, Metadata: m, Tops: tops}
}

func (w *Writer) ObserveSketch(ctx context.Context, sketch, operation string) (SketchObservation, error) {
	if err := w.mu.lockContext(ctx); err != nil {
		return SketchObservation{}, err
	}
	defer w.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return SketchObservation{}, err
	}
	out := SketchObservationOf(w.st, sketch, operation)
	if w.closed {
		return out, ErrClosed
	}
	return out, w.failure
}
