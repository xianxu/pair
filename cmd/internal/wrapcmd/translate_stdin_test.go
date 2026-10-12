package wrapcmd

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
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

// waitFor polls cond until it holds, bounded by the test's own deadline, not
// a guess at how fast the loop answers (#479); false means it gave up.
func waitFor(t *testing.T, cond func() bool) bool {
	t.Helper()
	return testwait.Wait(t, cond)
}

// bracketedPasteOn is what every profiled harness writes at startup (DECSET
// 2004). Fixtures that model a live agent and exercise paste markers must
// emit it: pair-wrap drops the markers for a child that never enabled paste
// (pair#211), so a fixture without it models a different agent.
const bracketedPasteOn = "\x1b[?2004h"

func proxyForHarness(harness string) *proxy {
	profile, ok := profileForHarness(harness, true)
	if !ok {
		return &proxy{agentBasename: harness}
	}
	return &proxy{agentBasename: harness, ttyProfile: &profile}
}

// claudeProxy yields a proxy wired through the production Claude profile with a
// live composer painted. Claude is positively gated (pair#138), so a proxy with
// no terminal state would fail closed and every Return-remap expectation below
// would read as a bare CR.
func claudeProxy() *proxy {
	p := proxyForHarness("claude")
	if p.ttyProfile == nil {
		return p
	}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		panic("configure claude test proxy: " + err.Error())
	}
	var rolling []byte
	p.handleChunk([]byte(bracketedPasteOn+claudeLiveComposerPaint()), &rolling)
	return p
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
	if !waitFor(t, func() bool { return bytes.Equal(out.Bytes(), []byte("hello")) }) {
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
	if !waitFor(t, func() bool { return bytes.Equal(out.Bytes(), []byte("hi\\\r")) }) {
		t.Fatalf("got %q, want %q", out.Bytes(), "hi\\\r")
	}
	_ = w.Close()
	<-done
}

// TestTranslateStdin_HeldBackEscFlushesAfterTimeout is the core
// invariant from commit 6b657e4. A lone \x1b on stdin (e.g. nvim's
// send_interrupt_to_agent writes a bare ESC for "interrupt the agent")
// gets held back as agentPending — there's no way to know in advance
// whether more bytes are coming to form an Alt+Enter chord or a CSI.
// After `pendingFlushAfter` of idle time, the timer fires and the
// held bytes are written verbatim to the child.
func TestTranslateStdin_HeldBackEscFlushesAfterTimeout(t *testing.T) {
	r, w := io.Pipe()
	out := newDrainBuffer()
	p := claudeProxy()
	flush := newManualTimer()
	p.newStdinFlushTimer = func() loopTimer { return flush }
	done := make(chan struct{})
	flushAfter := 20 * time.Millisecond
	go func() {
		p.translateStdinFrom(r, out, flushAfter)
		close(done)
	}()
	// Send a lone ESC. translateChunk holds it back (could be the
	// start of Alt+Enter, KKP CSI, bpStart, …) so out stays empty.
	_, _ = w.Write([]byte{0x1b})

	// The loop armed the flush when it held the ESC; until the flush's
	// clock reaches flushAfter nothing may reach the child.
	await(func() bool { armed, _ := flush.state(); return armed })
	flush.advance(flushAfter - time.Millisecond)
	if len(out.Bytes()) != 0 {
		t.Fatalf("flushed too early: %q", out.Bytes())
	}

	// After the timer fires, the lone ESC must appear in the output.
	flush.advance(time.Millisecond)
	await(func() bool { return len(out.Bytes()) > 0 })
	if !bytes.Equal(out.Bytes(), []byte{0x1b}) {
		t.Fatalf("ESC flushed as %q", out.Bytes())
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
	flush := newManualTimer()
	p.newStdinFlushTimer = func() loopTimer { return flush }
	done := make(chan struct{})
	flushAfter := 100 * time.Millisecond
	go func() {
		p.translateStdinFrom(r, out, flushAfter)
		close(done)
	}()
	// Half of Alt+Enter: ESC alone. Held in pending, flush armed; the
	// flush's clock stands still, so the second half always beats it.
	_, _ = w.Write([]byte{0x1b})
	await(func() bool { armed, _ := flush.state(); return armed })
	// Second half: \r. Combined with pending → Alt+Enter → altCR (\r).
	_, _ = w.Write([]byte{'\r'})

	// Should land in output as a single \r (claude altCR), not as
	// ESC followed by \r verbatim (which would be the timer-flush
	// case + a subsequent \r).
	await(func() bool { return len(out.Bytes()) > 0 })
	if !bytes.Equal(out.Bytes(), []byte{'\r'}) {
		t.Fatalf("got %q, want %q (Alt+Enter chord)", out.Bytes(), "\r")
	}
	// The loop writes the chord before it disarms the flush; a flush left
	// armed would hang here until the binary's timeout.
	await(func() bool { armed, _ := flush.state(); return !armed })
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
