package ttyio

import (
	"context"
	"sync"
)

// WriteStep is a transport outcome. A positive Limit accepts at most that many
// bytes; Block delays acceptance until opened or the caller cancels.
type WriteStep struct {
	Limit        int
	ZeroProgress bool
	Err          error
	Block        <-chan struct{}
}

// Fake retains accepted bytes and consumes ordered transport outcomes, letting
// integration tests observe real partial progress rather than count calls.
type Fake struct {
	mu      sync.Mutex
	steps   []WriteStep
	bytes   []byte
	calls   int
	started chan struct{}
}

func NewFake() *Fake                     { return &Fake{started: make(chan struct{}, 128)} }
func (f *Fake) Enqueue(s WriteStep)      { f.mu.Lock(); defer f.mu.Unlock(); f.steps = append(f.steps, s) }
func (f *Fake) Started() <-chan struct{} { return f.started }
func (f *Fake) Bytes() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]byte(nil), f.bytes...)
}
func (f *Fake) Calls() int { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }
func (f *Fake) WriteContext(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	f.mu.Lock()
	var s WriteStep
	if len(f.steps) > 0 {
		s = f.steps[0]
		f.steps = f.steps[1:]
	}
	f.calls++
	f.mu.Unlock()
	select {
	case f.started <- struct{}{}:
	default:
	}
	if s.Block != nil {
		select {
		case <-s.Block:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	n := len(p)
	if s.ZeroProgress {
		n = 0
	}
	if s.Limit > 0 && n > s.Limit {
		n = s.Limit
	}
	f.mu.Lock()
	f.bytes = append(f.bytes, p[:n]...)
	f.mu.Unlock()
	return n, s.Err
}
