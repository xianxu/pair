package main

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"
)

// drainBuffer is an io.Writer that records everything written and
// notifies a single waiter via a channel each time a Write lands.
// Used to observe the byte stream emitted by translateStdinFrom
// without racing on a bytes.Buffer.
type drainBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	wch chan struct{}
}

func newDrainBuffer() *drainBuffer {
	return &drainBuffer{wch: make(chan struct{}, 16)}
}

func (d *drainBuffer) Write(p []byte) (int, error) {
	d.mu.Lock()
	n, err := d.buf.Write(p)
	d.mu.Unlock()
	select {
	case d.wch <- struct{}{}:
	default:
	}
	return n, err
}

func (d *drainBuffer) Bytes() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	b := make([]byte, d.buf.Len())
	copy(b, d.buf.Bytes())
	return b
}

// waitFor calls cond() until it returns true or `timeout` elapses.
// Polls every 1 ms — cheap and avoids tying tests to specific
// scheduling order.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(1 * time.Millisecond)
	}
	return cond()
}

// claudeProxy yields a *proxy wired with the claude keymap (plain
// Enter → backslash-Enter, Alt+Enter → plain Enter), matching what
// pair-wrap's sendKeymapByAgent[claude] resolves to in production.
func claudeProxy() *proxy {
	return &proxy{sendKM: sendKeymap{
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
	}}
}

// TestTranslateStdin_PassthroughPlainBytes exercises the happy path:
// no carryover, no held-back sequence, just bytes in → bytes out.
func TestTranslateStdin_PassthroughPlainBytes(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	done := make(chan struct{})
	go func() {
		p.translateStdinFrom(r, out, 10*time.Millisecond)
		close(done)
	}()
	_, _ = w.Write([]byte("hello"))
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), []byte("hello")) }) {
		t.Fatalf("got %q, want %q", out.Bytes(), "hello")
	}
	_ = w.Close()
	<-done
}

// TestTranslateStdin_RewritesEnter exercises the keymap translation
// through the goroutine + select-loop machinery (not just
// translateChunk). Plain \r should become \\r per the claude keymap.
func TestTranslateStdin_RewritesEnter(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	done := make(chan struct{})
	go func() {
		p.translateStdinFrom(r, out, 10*time.Millisecond)
		close(done)
	}()
	_, _ = w.Write([]byte("hi\r"))
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), []byte("hi\\\r")) }) {
		t.Fatalf("got %q, want %q", out.Bytes(), "hi\\\r")
	}
	_ = w.Close()
	<-done
}

// TestTranslateStdin_HeldBackEscFlushesAfterTimeout is the core
// invariant from commit 6b657e4. A lone \x1b on stdin (e.g. nvim's
// send_esc_to_agent writes a bare ESC for "interrupt the agent")
// gets held back as agentPending — there's no way to know in advance
// whether more bytes are coming to form an Alt+Enter chord or a CSI.
// After `pendingFlushAfter` of idle time, the timer fires and the
// held bytes are written verbatim to the child.
func TestTranslateStdin_HeldBackEscFlushesAfterTimeout(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	done := make(chan struct{})
	flushAfter := 20 * time.Millisecond
	go func() {
		p.translateStdinFrom(r, out, flushAfter)
		close(done)
	}()
	// Send a lone ESC. translateChunk holds it back (could be the
	// start of Alt+Enter, KKP CSI, bpStart, …) so out stays empty.
	_, _ = w.Write([]byte{0x1b})

	// Confirm nothing flushed yet — sample a few times within the
	// pre-timeout window.
	time.Sleep(5 * time.Millisecond)
	if len(out.Bytes()) != 0 {
		t.Fatalf("flushed too early: %q", out.Bytes())
	}

	// After the timer fires, the lone ESC must appear in the output.
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), []byte{0x1b}) }) {
		t.Fatalf("ESC never flushed: out=%q", out.Bytes())
	}

	_ = w.Close()
	<-done
}

// TestTranslateStdin_ContinuationBeatsTimer covers the dual case: if
// the second half of a chord arrives before the timer fires, the
// chord resolves normally and no verbatim flush happens — the timer
// should be disarmed by the chord-completing chunk.
func TestTranslateStdin_ContinuationBeatsTimer(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	done := make(chan struct{})
	flushAfter := 100 * time.Millisecond
	go func() {
		p.translateStdinFrom(r, out, flushAfter)
		close(done)
	}()
	// Half of Alt+Enter: ESC alone. Held in pending.
	_, _ = w.Write([]byte{0x1b})
	time.Sleep(10 * time.Millisecond) // well under flushAfter
	// Second half: \r. Combined with pending → Alt+Enter → altCR (\r).
	_, _ = w.Write([]byte{'\r'})

	// Should land in output as a single \r (claude altCR), not as
	// ESC followed by \r verbatim (which would be the timer-flush
	// case + a subsequent \r).
	if !waitFor(200*time.Millisecond, func() bool { return bytes.Equal(out.Bytes(), []byte{'\r'}) }) {
		t.Fatalf("got %q, want %q (Alt+Enter chord)", out.Bytes(), "\r")
	}
	_ = w.Close()
	<-done
}

// TestTranslateStdin_EOFFlushesPending covers the close-side: even
// without the timer, EOF on stdin should flush any held-back bytes
// before returning so they're never silently lost.
func TestTranslateStdin_EOFFlushesPending(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	done := make(chan struct{})
	// Long flush timeout — confirm it's the EOF path, not the timer,
	// that wins here.
	go func() {
		p.translateStdinFrom(r, out, 10*time.Second)
		close(done)
	}()
	_, _ = w.Write([]byte{0x1b}) // partial sequence, held in pending
	time.Sleep(5 * time.Millisecond)
	_ = w.Close() // EOF
	<-done
	if !bytes.Equal(out.Bytes(), []byte{0x1b}) {
		t.Fatalf("EOF didn't flush pending: %q", out.Bytes())
	}
}
