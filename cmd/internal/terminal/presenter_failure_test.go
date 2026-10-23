package terminal

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// heldFailureWriter models a transport that can retain a physical failure when
// its caller also cancels. ttyio.Fake deliberately returns cancellation first.
type heldFailureWriter struct {
	*ttyio.Fake
	armed    atomic.Bool
	entered  chan struct{}
	release  chan struct{}
	physical error
}

func (w *heldFailureWriter) WriteContext(ctx context.Context, data []byte) (int, error) {
	if !w.armed.CompareAndSwap(true, false) {
		return w.Fake.WriteContext(ctx, data)
	}
	close(w.entered)
	select {
	case <-w.release:
	case <-ctx.Done():
	}
	return 0, errors.Join(ctx.Err(), w.physical)
}

func receiveFailure(t *testing.T, what string, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-testwait.After(t):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

func assertFailureCause(t *testing.T, err, physical error, canceled bool) {
	t.Helper()
	if err == nil || errors.Is(err, context.Canceled) != canceled || (physical != nil && !errors.Is(err, physical)) {
		t.Errorf("lost cause: got %v, canceled=%t physical=%v", err, canceled, physical)
	}
	var failure *WriteFailure
	if !errors.As(err, &failure) || failure.Op != "parent output" {
		t.Errorf("lost physical write evidence: %v", err)
	}
	// errors.Is alone would accept a synthetic refusal joined with cancellation.
	var leaves func(error)
	leaves = func(node error) {
		if node == nil {
			return
		}
		if joined, ok := node.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				leaves(child)
			}
		} else if wrapped, ok := node.(interface{ Unwrap() error }); ok {
			leaves(wrapped.Unwrap())
		} else if node != physical && !(canceled && node == context.Canceled) {
			t.Errorf("unexpected error leaf %v in %v", node, err)
		}
	}
	leaves(err)
}

func TestPresenterFailureCauseSurvivesQueuedAndLaterRequests(t *testing.T) {
	for _, mode := range []string{"cancel", "physical", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			w := &heldFailureWriter{Fake: ttyio.NewFake(), entered: make(chan struct{}), release: make(chan struct{})}
			if mode != "cancel" {
				w.physical = errors.New("physical paint failure")
			}
			w.armed.Store(true)
			p := NewPresenter(w, ChildRequested)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(func() { cancel(); p.Release(context.Background()) })
			paint := make(chan error, 1)
			go func() { paint <- p.Panel(ctx, Frame{Geometry: Geometry{8, 5}, Cells: make([]Cell, 40)}, FramePrivate) }()
			select {
			case <-w.entered:
			case <-testwait.After(t):
				t.Fatal("paint did not enter writer")
			}
			// The actor is blocked in the writer, so this request is necessarily
			// queued before failure. Its callback must never execute.
			var called atomic.Bool
			queued := presentRequest{ctx: context.Background(), done: make(chan error, 1), run: func(context.Context) error { called.Store(true); return nil }}
			p.requests <- queued
			if mode == "physical" {
				close(w.release)
			} else {
				cancel()
			}
			assertFailureCause(t, receiveFailure(t, "paint failure", paint), w.physical, mode != "physical")
			assertFailureCause(t, receiveFailure(t, "queued refusal", queued.done), w.physical, mode != "physical")
			if p.View().State != Failed || p.Failure() == nil {
				t.Fatalf("incoherent failed presenter: %+v, %v", p.View(), p.Failure())
			}
			first, writes := p.Failure(), w.Calls()
			for _, n := range []int{8, 16} {
				t.Run(fmt.Sprint(n), func(t *testing.T) {
					for i := 0; i < n; i++ {
						_, err := p.ChromeState(context.Background())
						assertFailureCause(t, err, w.physical, mode != "physical")
						assertFailureCause(t, p.Present(context.Background(), nil), w.physical, mode != "physical")
					}
					if called.Load() || w.Calls() != writes || p.Failure() != first {
						t.Fatal("rejected requests executed effects or changed the first failure")
					}
				})
			}
			canceled, stop := context.WithCancel(context.Background())
			stop()
			if err := p.Present(canceled, nil); err != context.Canceled || p.Failure() != first {
				t.Errorf("changed caller cancellation precedence or lost retained cause: %v, %v", err, p.Failure())
			}
			if err := p.Release(context.Background()); err != nil {
				t.Fatal(err)
			}
			_, err := p.ChromeState(context.Background())
			assertFailureCause(t, err, w.physical, mode != "physical")
		})
	}
}

func TestPresenterFailureCauseSurvivesReleasedCompletion(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			// Isolate call's completed-owner branch: the request is admitted
			// before owner completion, but never receives an individual result.
			p := &Presenter{requests: make(chan presentRequest, 1), done: make(chan struct{})}
			result := make(chan error, 1)
			go func() {
				result <- p.call(context.Background(), func(context.Context) error { t.Error("released callback ran"); return nil })
			}()
			select {
			case <-p.requests:
			case <-testwait.After(t):
				t.Fatal("request was not admitted")
			}
			p.mu.Lock()
			p.view.State = Released
			p.closing = true
			if failed {
				p.failureErr = &WriteFailure{Op: "parent output", Err: context.Canceled}
			}
			p.mu.Unlock()
			close(p.done)
			err := receiveFailure(t, "released request", result)
			if failed {
				assertFailureCause(t, err, nil, true)
			} else if err == nil || err.Error() != "terminal: presenter released" {
				t.Errorf("healthy release refusal changed: %v", err)
			}
		})
	}
}
