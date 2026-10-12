package recording

import "sync"

// Hub is the production fan-out of a recording to its viewers (#460): a
// Live and the Sink that feeds it. It keeps only what a late joiner needs:
// at each segment its log shrinks to CatchUp(log), so its memory is bounded
// by one segment plus the assets and the current document's notes (about
// 1.4 MB/min of segment on #453's busy stress page), and it is freed with
// the Hub when the cast ends. It is also the fake other components test
// against: the real thing is small enough to be its own double.
type Hub struct {
	mu     sync.Mutex
	log    []Record
	subs   map[chan Record]struct{}
	buffer int
	closed bool
}

// NewHub: buffer is how many records a subscriber may fall behind before
// it is dropped.
func NewHub(buffer int) *Hub {
	return &Hub{subs: map[chan Record]struct{}{}, buffer: buffer}
}

func (h *Hub) Put(r Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrClosed
	}
	if _, ok := r.(*Frame); ok {
		return ErrNotLive
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
func (h *Hub) Subscribe() (<-chan Record, func()) {
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
func (h *Hub) Close() error {
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
func (h *Hub) Retained() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.log)
}
