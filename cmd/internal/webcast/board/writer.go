package board

import (
	"context"
	"errors"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"sync"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// Writer is the reference single writer: the recorder's records (Put) and a
// board's (Emit) go through it, in order, into one sink. It stamps a board's
// records with the clock (never before the last record) and checks
// recording.State.OnDoc as it orders them, so a record checked against a
// document a navigation has since replaced is dropped. The recorder's queue
// does the same (recorder.Emit's Doc); tests and fakes use this.
type Writer struct {
	mu           sync.Mutex
	clearID      uint64
	clearFailure error
	st           *recording.State
	lastKind     string
	sink         recording.Sink
	now          func() int64
}

// NewWriter writes into sink. now is the recording's clock in ms; nil
// stamps a board's records with the last record's t.
func NewWriter(sink recording.Sink, now func() int64) *Writer {
	return &Writer{st: recording.NewState(), sink: sink, now: now}
}

// Put admits generated live recorder input. Like the recorder mirror it
// stamps timestamps monotonically and preserves file-merge order. Raw stored
// records are validated by recording.State.Apply/Validate, without stamping.
func (w *Writer) Put(r recording.Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeLocked(r)
}

// Emit is a Board's Emit: it writes r stamped now, or refuses it when the
// document that began at doc isn't current and live any more: ErrPaused
// during a pause (pair#499), ErrStale otherwise.
func (w *Writer) Emit(r recording.Record, doc int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.st.OnDoc(doc) {
		if w.st.Paused() && w.st.DocSeg == doc {
			return ErrPaused
		}
		return ErrStale
	}
	if n, ok := r.(*recording.NoteState); ok && n.IfRevision != nil && !w.st.NoteUnchanged(n.ID, *n.IfRevision) {
		return ErrStale
	}
	t := w.st.T
	if w.now != nil {
		t = max(t, w.now())
	}
	r.SetTime(t)
	return w.writeLocked(r)
}

// Close closes the sink.
func (w *Writer) Close() error { return w.sink.Close() }

func (w *Writer) writeLocked(r recording.Record) error {
	r.SetTime(recording.OrderedTime(r.Time(), w.st.T, w.lastKind, r.Kind()))
	if err := w.st.Apply(r); err != nil {
		return err
	}
	w.lastKind = r.Kind()
	return w.sink.Put(r)
}

// ClearAnnotations serializes with Emit and acknowledges the sink's clear.
// Repeating a successful id is safe, including when no recording has begun.
func (w *Writer) ClearAnnotations(ctx context.Context, id uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.clearFailure != nil {
		return w.clearFailure
	}
	if id == 0 || id < w.clearID {
		return &capability.Failure{Outcome: capability.Broken, Cause: errors.New("board: invalid cleanup identity")}
	}
	if id == w.clearID {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return &capability.Failure{Outcome: capability.Retryable, Cause: err}
	}
	if w.st.Head == nil {
		w.clearID = id
		return nil
	}
	t := w.st.T + 1
	if w.now != nil {
		t = max(t, w.now())
	}
	r := &recording.AnnotationClear{Header: recording.Header{T: t}, Who: recording.Author{ID: recording.OperatorID}}
	if err := w.writeLocked(r); err != nil {
		// The reducer applies before the sink: failure cannot safely be retried.
		w.clearFailure = &capability.Failure{Outcome: capability.Broken, Cause: err}
		return w.clearFailure
	}
	w.clearID = id
	return nil
}
