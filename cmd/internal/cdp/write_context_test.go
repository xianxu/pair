package cdp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// ownedGateWriter models the generic owned-writer contract: Close interrupts
// Write. The production os.Pipe transport additionally supports deadlines.
type ownedGateWriter struct {
	mu        sync.Mutex
	writes    int
	entered   chan struct{}
	release   chan struct{}
	once      sync.Once
	closeOnce sync.Once
	closed    bool
}

func (w *ownedGateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.writes++
	w.mu.Unlock()
	w.once.Do(func() { close(w.entered) })
	<-w.release
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}
func (w *ownedGateWriter) Close() error {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	w.closeOnce.Do(func() { close(w.release) })
	return nil
}
func (w *ownedGateWriter) unblock() { w.closeOnce.Do(func() { close(w.release) }) }
func writeTestConn(t *testing.T, w io.WriteCloser) *Conn {
	t.Helper()
	input, peer := io.Pipe()
	c := New(input, w)
	t.Cleanup(func() {
		c.Close()
		input.Close()
		peer.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := c.Wait(ctx); err != nil {
			t.Error(err)
		}
	})
	return c
}
func writeResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("write did not honor context/join")
		return nil
	}
}
func TestWriteCanceledBeforeAdmissionDoesNotEmit(t *testing.T) {
	w := &ownedGateWriter{entered: make(chan struct{}), release: make(chan struct{})}
	c := writeTestConn(t, w)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() { done <- c.Call(ctx, "", "Never", nil, nil) }()
	// Release is only teardown insurance for the old blocking implementation.
	t.Cleanup(w.unblock)
	if err := writeResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writes != 0 {
		t.Fatal("canceled call emitted a frame")
	}
}
func TestWriteQueuedDeadlineDoesNotEmit(t *testing.T) {
	w := &ownedGateWriter{entered: make(chan struct{}), release: make(chan struct{})}
	c := writeTestConn(t, w)
	first := make(chan error, 1)
	go func() { first <- c.Call(context.Background(), "", "First", nil, nil) }()
	<-w.entered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	second := make(chan error, 1)
	go func() { second <- c.Call(ctx, "", "Never", nil, nil) }()
	if err := writeResult(t, second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued: %v", err)
	}
	w.mu.Lock()
	writes := w.writes
	w.mu.Unlock()
	if writes != 1 {
		t.Fatalf("queued canceled write emitted: %d", writes)
	}
	w.unblock()
	c.dispatch(Message{ID: 1, Result: json.RawMessage(`{}`)})
	if err := writeResult(t, first); err != nil {
		t.Fatal(err)
	}
}
func TestWriteGenericCancellationRetiresOwnedTransport(t *testing.T) {
	w := &ownedGateWriter{entered: make(chan struct{}), release: make(chan struct{})}
	c := writeTestConn(t, w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Call(ctx, "", "Blocked", nil, nil) }()
	<-w.entered
	cancel()
	if err := writeResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("active cancel: %v", err)
	}
	if c.Err() == nil {
		t.Fatal("unsupported deadline transport remained usable after interrupted write")
	}
}

type observedPipe struct {
	*os.File
	entered  chan struct{}
	once     sync.Once
	mu       sync.Mutex
	deadline time.Time
}

func (p *observedPipe) Write(b []byte) (int, error) {
	p.once.Do(func() { close(p.entered) })
	return p.File.Write(b)
}
func (p *observedPipe) SetWriteDeadline(d time.Time) error {
	p.mu.Lock()
	p.deadline = d
	p.mu.Unlock()
	return p.File.SetWriteDeadline(d)
}
func newObservedPipe(t *testing.T) (*os.File, *observedPipe) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	return r, &observedPipe{File: w, entered: make(chan struct{})}
}
func TestWriteRealPipePartialCancellationRetiresConnection(t *testing.T) {
	r, w := newObservedPipe(t)
	c := writeTestConn(t, w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- c.Call(ctx, "", "Large", map[string]string{"payload": strings.Repeat("x", 2<<20)}, nil)
	}()
	// Reading a prefix proves that interruption follows actual frame emission.
	prefix := make([]byte, 128)
	if _, err := io.ReadFull(r, prefix); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := writeResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("partial cancel: %v", err)
	}
	if c.Err() == nil {
		t.Fatal("partial frame left connection usable")
	}
	if err := c.Call(context.Background(), "", "MustNotFollowPartial", nil, nil); err == nil {
		t.Fatal("command followed partial frame")
	}
}
func fillCommandPipe(t *testing.T, w *os.File) int64 {
	t.Helper()
	total := int64(0)
	for _, size := range []int{4096, 1} {
		if err := w.SetWriteDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
		for {
			n, err := w.Write(bytes.Repeat([]byte{'x'}, size))
			total += int64(n)
			if err != nil {
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					t.Fatal(err)
				}
				break
			}
		}
	}
	if err := w.SetWriteDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	return total
}
func TestWriteRealPipeZeroByteCancellationResetsDeadline(t *testing.T) {
	r, w := newObservedPipe(t)
	filled := fillCommandPipe(t, w.File)
	c := writeTestConn(t, w)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Call(ctx, "", "Blocked", nil, nil) }()
	<-w.entered
	cancel()
	if err := writeResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("zero-byte cancel: %v", err)
	}
	if c.Err() != nil {
		t.Fatalf("zero-byte deadline cancellation poisoned connection: %v", c.Err())
	}
	w.mu.Lock()
	deadline := w.deadline
	w.mu.Unlock()
	if !deadline.IsZero() {
		t.Fatal("cancellation deadline leaked to next owner")
	}
	if _, err := io.CopyN(io.Discard, r, filled); err != nil {
		t.Fatal(err)
	}
	read := make(chan error, 1)
	go func() {
		raw, err := newFrameReader(r, maxFrame).next()
		if err == nil {
			var m Message
			err = json.Unmarshal(raw, &m)
			if err == nil {
				c.dispatch(Message{ID: m.ID, Result: json.RawMessage(`{}`)})
			}
		}
		read <- err
	}()
	nextCtx, nextCancel := context.WithTimeout(context.Background(), time.Second)
	defer nextCancel()
	if err := c.Call(nextCtx, "", "AfterReset", nil, nil); err != nil {
		t.Fatalf("reused connection: %v", err)
	}
	if err := writeResult(t, read); err != nil {
		t.Fatal(err)
	}
}

type shortFrameWriter struct{ closed bool }

func (w *shortFrameWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func (w *shortFrameWriter) Close() error                { w.closed = true; return nil }
func TestWriteShortFrameWithoutErrorRetiresConnection(t *testing.T) {
	w := &shortFrameWriter{}
	c := writeTestConn(t, w)
	if err := c.Call(context.Background(), "", "Short", nil, nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short frame: %v", err)
	}
	if !w.closed || c.Err() == nil {
		t.Fatal("short frame did not retire and close owned writer")
	}
}

type joinedDeadlineWriter struct {
	mu                                                  sync.Mutex
	entered, writeRelease, cancelEntered, cancelRelease chan struct{}
	canceled, callbackFinished, resetEarly              bool
	writes                                              int
}

func (w *joinedDeadlineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.writes++
	w.mu.Unlock()
	close(w.entered)
	<-w.writeRelease
	return 0, os.ErrDeadlineExceeded
}
func (w *joinedDeadlineWriter) Close() error { return nil }
func (w *joinedDeadlineWriter) SetWriteDeadline(d time.Time) error {
	if !d.IsZero() && !d.After(time.Now()) {
		w.mu.Lock()
		w.canceled = true
		w.mu.Unlock()
		close(w.cancelEntered)
		<-w.cancelRelease
		w.mu.Lock()
		w.callbackFinished = true
		w.mu.Unlock()
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if d.IsZero() && w.canceled && !w.callbackFinished {
		w.resetEarly = true
	}
	return nil
}
func TestWriteCancellationCallbackJoinsBeforeResetAndNextAdmission(t *testing.T) {
	w := &joinedDeadlineWriter{entered: make(chan struct{}), writeRelease: make(chan struct{}), cancelEntered: make(chan struct{}), cancelRelease: make(chan struct{})}
	c := writeTestConn(t, w)
	var releaseWrite, releaseCancel sync.Once
	t.Cleanup(func() {
		releaseWrite.Do(func() { close(w.writeRelease) })
		releaseCancel.Do(func() { close(w.cancelRelease) })
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Call(ctx, "", "Interrupted", nil, nil) }()
	<-w.entered
	cancel()
	<-w.cancelEntered
	releaseWrite.Do(func() { close(w.writeRelease) })
	nextCtx, nextCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer nextCancel()
	next := make(chan error, 1)
	go func() { next <- c.Call(nextCtx, "", "NoNewOwnerYet", nil, nil) }()
	if err := writeResult(t, next); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued while callback active: %v", err)
	}
	w.mu.Lock()
	early, writes := w.resetEarly, w.writes
	w.mu.Unlock()
	if early || writes != 1 {
		t.Fatalf("deadline reset or token released before callback join: early=%v writes=%d", early, writes)
	}
	select {
	case err := <-done:
		t.Fatalf("returned before cancellation callback joined: %v", err)
	default:
	}
	releaseCancel.Do(func() { close(w.cancelRelease) })
	if err := writeResult(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if c.Err() != nil {
		t.Fatalf("zero-byte canceled frame retired deadline transport: %v", c.Err())
	}
}
