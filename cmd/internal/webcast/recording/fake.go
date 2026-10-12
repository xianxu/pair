package recording

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Live is the fan-out to viewers (the live viewer's server side, #460): each
// subscriber gets the catch-up, then every record as it is put.
//
// Subscribe's channel closes when cancel is called, when the hub closes, or
// when the subscriber falls behind: a slow viewer is dropped, never allowed
// to hold the recorder back. A dropped viewer reconnects and gets a fresh
// catch-up. cancel is idempotent.
type Live interface {
	Subscribe() (recs <-chan Record, cancel func())
}

// ErrClosed: a record was put after Close.
var ErrClosed = errors.New("recording: sink closed")

// MemSink is the stateful fake Sink: it keeps what it was given, in order,
// and refuses what a real sink would (a put after close, time going back).
type MemSink struct {
	mu     sync.Mutex
	recs   []Record
	closed bool
}

func (m *MemSink) Put(r Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	if n := len(m.recs); n > 0 && r.Time() < m.recs[n-1].Time() {
		return fmt.Errorf("t=%d is before the previous record's t=%d", r.Time(), m.recs[n-1].Time())
	}
	m.recs = append(m.recs, r)
	return nil
}

func (m *MemSink) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Records is a copy of what was put.
func (m *MemSink) Records() []Record {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Record(nil), m.recs...)
}

// Closed reports whether Close was called.
func (m *MemSink) Closed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// FakeHub is the stateful fake of Live, and a Sink. It keeps only what a
// late joiner needs: at each segment its log shrinks to CatchUp(log), so it
// is bounded by one segment plus the assets and the document's notes.
type FakeHub struct {
	mu     sync.Mutex
	log    []Record
	subs   map[chan Record]struct{}
	buffer int
	closed bool
}

// NewFakeHub: buffer is how many records a subscriber may fall behind
// before it is dropped.
func NewFakeHub(buffer int) *FakeHub {
	return &FakeHub{subs: map[chan Record]struct{}{}, buffer: buffer}
}

func (h *FakeHub) Put(r Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrClosed
	}
	if _, ok := r.(*Seg); ok {
		h.log = CatchUp(h.log)
	}
	h.log = append(h.log, r)
	for ch := range h.subs {
		select {
		case ch <- r:
		default:
			delete(h.subs, ch)
			close(ch)
		}
	}
	return nil
}

// Subscribe takes the catch-up and registers under one lock, so no record
// falls between them.
func (h *FakeHub) Subscribe() (<-chan Record, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cu := CatchUp(h.log)
	ch := make(chan Record, len(cu)+h.buffer)
	for _, r := range cu {
		ch <- r
	}
	if h.closed {
		close(ch)
		return ch, func() {}
	}
	h.subs[ch] = struct{}{}
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
	}
}

// Close ends every subscription; later puts fail.
func (h *FakeHub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.closed {
		h.closed = true
		for ch := range h.subs {
			close(ch)
		}
		h.subs = nil
	}
	return nil
}

// Retained is how many records the hub holds for late joiners.
func (h *FakeHub) Retained() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.log)
}

// FakeRecorder stands in for the recorder (#458): it plays a recording into
// a sink. Speed 0 is instant; 1 is real time, by the records' t.
type FakeRecorder struct {
	Records []Record
	Speed   float64
}

// Run puts every record, in order, and returns at the end or when ctx is
// done. It doesn't close the sink: the caller owns it.
func (f FakeRecorder) Run(ctx context.Context, sink Sink) error {
	var prev int64
	for i, r := range f.Records {
		if f.Speed > 0 && i > 0 {
			wait := time.Duration(float64(r.Time()-prev) / f.Speed * float64(time.Millisecond))
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := sink.Put(r); err != nil {
			return err
		}
		prev = r.Time()
	}
	return nil
}
