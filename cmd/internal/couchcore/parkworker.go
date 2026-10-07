package couchcore

import (
	"context"
	"errors"
	"sync"
)

type parkWork func(context.Context) (ParkResult, error)

type parkWorkResult struct {
	result ParkResult
	err    error
}

type parkFuture struct {
	done chan struct{}
	once sync.Once
	work parkWorkResult
}

func (f *parkFuture) Await(ctx context.Context) (ParkResult, error) {
	if f == nil {
		return ParkResult{}, errors.New("nil park future")
	}
	select {
	case <-ctx.Done():
		return ParkResult{}, ctx.Err()
	case <-f.done:
		return f.work.result, f.work.err
	}
}

type activeParkWork struct {
	nonce  string
	future *parkFuture
}

// parkWorker bounds both goroutines and accepted work by the supplied Couch
// admission capacity. At most one operation exists per address; duplicate
// nonce submissions share its future rather than creating parallel teardown.
//
// Capacity is a throughput bound that submitters WAIT on, never a refusal
// (pair#205): every park submitter shares this worker -- Leave's fan-out,
// operator and remote parks, relaunch, switch-agent, continuations and boot
// recovery -- so a refusal at capacity would fail whichever arrived last, at
// random. Only a second transaction on an active address is refused.
type parkWorker struct {
	mu       sync.Mutex
	capacity int
	active   map[ThreadAddress]activeParkWork
	// freed is closed and replaced whenever a unit is released, waking
	// submitters waiting for capacity.
	freed chan struct{}
	// onWait, when set, runs as a submitter starts waiting for capacity, so a
	// test synchronizes on the wait instead of sleeping. Tests only.
	onWait func()
}

func newParkWorker(capacity int) *parkWorker {
	return &parkWorker{capacity: capacity, active: map[ThreadAddress]activeParkWork{}, freed: make(chan struct{})}
}

func (w *parkWorker) Submit(ctx context.Context, address ThreadAddress, nonce string, work parkWork) (*parkFuture, error) {
	if w == nil || w.capacity <= 0 || work == nil || nonce == "" {
		return nil, errors.New("park worker submission is incomplete")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w.mu.Lock()
	for {
		if current, ok := w.active[address]; ok {
			w.mu.Unlock()
			if current.nonce == nonce {
				return current.future, nil
			}
			return nil, errors.New("another park transaction already owns this address")
		}
		if len(w.active) < w.capacity {
			break
		}
		freed, onWait := w.freed, w.onWait
		w.mu.Unlock()
		if onWait != nil {
			onWait()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-freed:
		}
		w.mu.Lock()
	}
	future := &parkFuture{done: make(chan struct{})}
	w.active[address] = activeParkWork{nonce: nonce, future: future}
	w.mu.Unlock()

	go func() {
		future.work.result, future.work.err = work(ctx)
		// Free the address BEFORE signalling done: a caller that has seen its
		// park finish may submit the next park on this address at once, and a
		// stale entry would refuse it as "another park transaction" (pair#205).
		w.mu.Lock()
		delete(w.active, address)
		close(w.freed)
		w.freed = make(chan struct{})
		w.mu.Unlock()
		close(future.done)
	}()
	return future, nil
}
