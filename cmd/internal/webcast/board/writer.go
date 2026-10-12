package board

import (
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
	mu   sync.Mutex
	st   *recording.State
	sink recording.Sink
	now  func() int64
}

// NewWriter writes into sink. now is the recording's clock in ms; nil
// stamps a board's records with the last record's t.
func NewWriter(sink recording.Sink, now func() int64) *Writer {
	return &Writer{st: recording.NewState(), sink: sink, now: now}
}

// Put writes one of the recorder's records as it is.
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
	if err := w.st.Apply(r); err != nil {
		return err
	}
	return w.sink.Put(r)
}
