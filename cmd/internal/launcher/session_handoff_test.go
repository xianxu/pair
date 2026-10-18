package launcher

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type handoffWorld struct {
	mu        sync.Mutex
	events    []string
	admission chan error
	exit      chan HandoffExit
	released  chan struct{}
	started   chan struct{}
	checkErr  error
}

func newHandoffWorld() *handoffWorld {
	return &handoffWorld{admission: make(chan error, 1), exit: make(chan HandoffExit, 1), released: make(chan struct{}), started: make(chan struct{})}
}
func (w *handoffWorld) event(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.events = append(w.events, s)
}
func (w *handoffWorld) Acquire(context.Context) (io.Closer, error) {
	w.event("acquire")
	return handoffLease{w}, nil
}
func (w *handoffWorld) Check(context.Context) error { w.event("check"); return w.checkErr }
func (w *handoffWorld) Start(ctx context.Context) (HandoffClient, error) {
	w.event("start")
	close(w.started)
	return handoffChild{w, ctx}, nil
}
func (w *handoffWorld) Admit(ctx context.Context) error {
	select {
	case e := <-w.admission:
		w.event("admit")
		return e
	case <-ctx.Done():
		w.event("admission-cancelled")
		return ctx.Err()
	}
}

type handoffLease struct{ w *handoffWorld }

func (l handoffLease) Close() error { l.w.event("release"); close(l.w.released); return nil }

type handoffChild struct {
	w   *handoffWorld
	ctx context.Context
}

func (c handoffChild) Wait() HandoffExit {
	select {
	case e := <-c.w.exit:
		c.w.event("reaped")
		return e
	case <-c.ctx.Done():
		c.w.event("reaped")
		return HandoffExit{Code: -1}
	}
}

func TestTerminalHandoffReleasesAfterAdmissionBeforeClientExit(t *testing.T) {
	w := newHandoffWorld()
	done := make(chan error, 1)
	go func() { _, err := CoordinateTerminalHandoff(context.Background(), w, time.Minute); done <- err }()
	<-w.started
	select {
	case <-w.released:
		t.Fatal("released before admission")
	default:
	}
	w.admission <- nil
	<-w.released
	select {
	case <-done:
		t.Fatal("did not wait for interactive client")
	default:
	}
	w.exit <- HandoffExit{Code: 0}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestTerminalHandoffRefusesBeforeStartingWhenOwnerContested(t *testing.T) {
	w := newHandoffWorld()
	w.checkErr = errors.New("contested")
	if _, err := CoordinateTerminalHandoff(context.Background(), w, time.Minute); !errors.Is(err, w.checkErr) {
		t.Fatalf("got %v", err)
	}
	select {
	case <-w.started:
		t.Fatal("started contested session")
	default:
	}
	<-w.released
}
func TestTerminalHandoffFailureReapsBeforeReleasing(t *testing.T) {
	w := newHandoffWorld()
	w.admission <- errors.New("unverified")
	if _, err := CoordinateTerminalHandoff(context.Background(), w, time.Minute); err == nil {
		t.Fatal("accepted unknown admission")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, e := range w.events {
		if e == "release" && (i == 0 || w.events[i-1] != "reaped") {
			t.Fatalf("released before own client was reaped: %v", w.events)
		}
	}
}
func TestTerminalHandoffCancelledAdmissionJoinsBeforeRelease(t *testing.T) {
	w := newHandoffWorld()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := CoordinateTerminalHandoff(ctx, w, time.Minute); done <- err }()
	<-w.started
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancel reported success")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.events[len(w.events)-1] != "release" {
		t.Fatalf("work after release: %v", w.events)
	}
}

func TestTerminalHandoffCanceledBeforeStart(t *testing.T) {
	w := newHandoffWorld()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CoordinateTerminalHandoff(ctx, w, time.Minute)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	select {
	case <-w.started:
		t.Fatal("started client with canceled admission")
	default:
	}
}
