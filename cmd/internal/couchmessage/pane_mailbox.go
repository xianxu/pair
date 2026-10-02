package couchmessage

import "sync"

// PaneMailbox carries the Console's pane state per thread to the message
// service without ever blocking the Console (#365). It keeps only the latest
// pane per thread ("" = none) — Registry treats PaneChanged as a state, so
// coalescing loses nothing — and is bounded by the number of threads.
type PaneMailbox struct {
	mu     sync.Mutex
	latest map[ThreadKey]PaneHandle
	wake   chan struct{}
}

func NewPaneMailbox() *PaneMailbox {
	return &PaneMailbox{latest: map[ThreadKey]PaneHandle{}, wake: make(chan struct{}, 1)}
}

func (m *PaneMailbox) Post(thread ThreadKey, pane PaneHandle) {
	m.mu.Lock()
	m.latest[thread] = pane
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *PaneMailbox) Wake() <-chan struct{} { return m.wake }

// Drain hands over every thread's latest pane posted since the last drain.
func (m *PaneMailbox) Drain() map[ThreadKey]PaneHandle {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.latest
	m.latest = map[ThreadKey]PaneHandle{}
	return out
}
