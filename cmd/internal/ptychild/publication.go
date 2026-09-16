package ptychild

import (
	"context"
	"errors"
	"sync"
)

// Sink acknowledges one already-ingested output batch. Cancellation must be
// honored both before enqueueing UI work and while waiting for its completion.
type Sink func(context.Context, OutputBatch) error

const publicationPackets = 128
const publicationBytes = 1 << 20

type delivery struct {
	batch  OutputBatch
	sink   Sink
	weight int
}
type publication struct {
	mu      sync.Mutex
	queue   []delivery
	bytes   int
	changed chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	err     error
}

func newPublication() *publication {
	ctx, cancel := context.WithCancel(context.Background())
	p := &publication{ctx: ctx, cancel: cancel, done: make(chan struct{}), changed: make(chan struct{})}
	go p.run()
	return p
}
func (p *publication) signalLocked() { close(p.changed); p.changed = make(chan struct{}) }
func outputWeight(b OutputBatch) int {
	n := len(b.Raw)
	for _, effect := range b.Terminal.Effects {
		n += len(effect.Text) + len(effect.Selection) + len(effect.Data)
	}
	return n
}
func (p *publication) enqueue(batch OutputBatch, sink Sink) error {
	if sink == nil {
		return nil
	}
	weight := outputWeight(batch)
	if weight > publicationBytes {
		return errors.New("ptychild: publication exceeds byte limit")
	}
	for {
		p.mu.Lock()
		if p.err != nil {
			err := p.err
			p.mu.Unlock()
			return err
		}
		if err := p.ctx.Err(); err != nil {
			p.mu.Unlock()
			return err
		}
		if len(p.queue) < publicationPackets && weight <= publicationBytes-p.bytes {
			p.queue = append(p.queue, delivery{batch, sink, weight})
			p.bytes += weight
			p.signalLocked()
			p.mu.Unlock()
			return nil
		}
		changed := p.changed
		p.mu.Unlock()
		select {
		case <-changed:
		case <-p.ctx.Done():
			// Recheck the latched consumer error before cancellation.
			continue
		}
	}
}
func (p *publication) run() {
	defer close(p.done)
	for {
		p.mu.Lock()
		if p.ctx.Err() != nil {
			p.queue = nil
			p.bytes = 0
			p.signalLocked()
			p.mu.Unlock()
			return
		}
		if len(p.queue) == 0 {
			changed := p.changed
			p.mu.Unlock()
			select {
			case <-changed:
			case <-p.ctx.Done():
			}
			continue
		}
		d := p.queue[0]
		p.mu.Unlock()
		err := d.sink(p.ctx, d.batch)
		p.mu.Lock()
		p.queue[0] = delivery{}
		p.queue = p.queue[1:]
		p.bytes -= d.weight
		if err != nil {
			p.err = err
			p.cancel()
			p.queue = nil
			p.bytes = 0
			p.signalLocked()
			p.mu.Unlock()
			return
		}
		p.signalLocked()
		p.mu.Unlock()
	}
}
func (p *publication) flush(ctx context.Context) error {
	for {
		p.mu.Lock()
		err := p.err
		if err == nil {
			err = p.ctx.Err()
		}
		empty := len(p.queue) == 0
		changed := p.changed
		p.mu.Unlock()
		if err != nil {
			return err
		}
		if empty {
			return nil
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		case <-p.ctx.Done():
			// Recheck the latched consumer error before cancellation.
			continue
		}
	}
}
func (p *publication) close() { p.cancel(); <-p.done }
