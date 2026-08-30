package couchcore

import (
	"context"
	"errors"
	"sync"
)

var ErrParkWorkerOverloaded = errors.New("park worker admission is full")

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
type parkWorker struct {
	mu       sync.Mutex
	capacity int
	active   map[ThreadAddress]activeParkWork
}

func newParkWorker(capacity int) *parkWorker {
	return &parkWorker{capacity: capacity, active: map[ThreadAddress]activeParkWork{}}
}

func (w *parkWorker) Submit(ctx context.Context, address ThreadAddress, nonce string, work parkWork) (*parkFuture, error) {
	if w == nil || w.capacity <= 0 || work == nil || nonce == "" {
		return nil, errors.New("park worker submission is incomplete")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w.mu.Lock()
	if current, ok := w.active[address]; ok {
		w.mu.Unlock()
		if current.nonce == nonce {
			return current.future, nil
		}
		return nil, errors.New("another park transaction already owns this address")
	}
	if len(w.active) >= w.capacity {
		w.mu.Unlock()
		return nil, ErrParkWorkerOverloaded
	}
	future := &parkFuture{done: make(chan struct{})}
	w.active[address] = activeParkWork{nonce: nonce, future: future}
	w.mu.Unlock()

	go func() {
		future.work.result, future.work.err = work(ctx)
		close(future.done)
		w.mu.Lock()
		delete(w.active, address)
		w.mu.Unlock()
	}()
	return future, nil
}
