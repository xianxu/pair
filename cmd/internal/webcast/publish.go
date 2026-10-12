package webcast

import "sync"

// publisher fans the latest value out to subscribers. Each subscriber's
// channel holds one value that a newer one replaces, so publishing never
// blocks and a slow reader only ever sees the newest state. Shared by
// FakeBrowser, Chrome and Cast.
type publisher[T any] struct {
	mu     sync.Mutex
	cur    T
	subs   map[chan T]struct{}
	closed bool
}

func newPublisher[T any](initial T) *publisher[T] {
	return &publisher[T]{cur: initial, subs: map[chan T]struct{}{}}
}

func (p *publisher[T]) get() T {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cur
}

func (p *publisher[T]) publish(v T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.cur = v
	for ch := range p.subs {
		replace(ch, v)
	}
}

// finish publishes the final value and closes every subscriber's channel;
// later publishes are dropped and later subscribers get the final value on a
// closed channel.
func (p *publisher[T]) finish(v T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.cur, p.closed = v, true
	for ch := range p.subs {
		replace(ch, v)
		close(ch)
	}
	p.subs = nil
}

func (p *publisher[T]) subscribe() (<-chan T, func()) {
	ch := make(chan T, 1)
	p.mu.Lock()
	defer p.mu.Unlock()
	ch <- p.cur
	if p.closed {
		close(ch)
		return ch, func() {}
	}
	p.subs[ch] = struct{}{}
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if _, ok := p.subs[ch]; ok {
				delete(p.subs, ch)
				close(ch)
			}
		})
	}
}

// replace puts v in a one-slot channel, dropping the stale value. The caller
// holds the publisher's lock, so no other sender races it.
func replace[T any](ch chan T, v T) {
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- v
	}
}
