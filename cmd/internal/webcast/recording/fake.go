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
// catch-up. cancel is idempotent. Its Sink refuses frames (ErrNotLive).
//
// Ended says why a subscription's channel closed, as the fan-out saw it:
// true when the fan-out ended (its Sink closed; the subscriber has every
// record there will be), false when the subscriber was dropped or
// cancelled (a dropped viewer reconnects for a fresh catch-up). The fan-out
// records the reason when it closes the channel, so a drop just before the
// end still reads as a drop.
type Live interface {
	Subscribe() (recs <-chan Record, cancel func())
	Ended(recs <-chan Record) bool
}

// ErrClosed: a record was put after Close.
var ErrClosed = errors.New("recording: sink closed")

// ErrNotLive: a record that never goes to viewers (a frame: pixels never
// reach them) was put into a live fan-out.
var ErrNotLive = errors.New("recording: frames are not streamed to viewers")

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

// FakeHub was Hub's name while it was only the fake of Live (#457).
//
// Deprecated: use Hub, which is the production fan-out (#460). Kept for
// branches written against #457's name.
type FakeHub = Hub

// NewFakeHub is NewHub's old name.
//
// Deprecated: use NewHub.
func NewFakeHub(buffer int) *Hub { return NewHub(buffer) }

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
