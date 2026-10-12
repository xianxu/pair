package ttyio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func TestFileBlockedWritePreservesConcurrentReadAndFlags(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	raw, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(slave.Fd()), raw)
	fd := int(master.Fd())
	before, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	transport, err := NewFile(master, master, false)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()
	type result struct {
		data string
		err  error
	}
	read := make(chan result, 1)
	started := time.Now() // before either goroutine can poll
	go func() { buf := make([]byte, 16); n, e := transport.Read(buf); read <- result{string(buf[:n]), e} }()
	// Cancel once the write has made progress, not at a fixed deadline: a
	// writer goroutine scheduled late would otherwise meet an expired context
	// before writing a byte (#480).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	written := make(chan error, 1)
	go func() {
		n, e := transport.WriteContext(ctx, bytes.Repeat([]byte("x"), 1<<20))
		if n <= 0 || n >= 1<<20 {
			written <- errors.New("expected known partial progress")
			return
		}
		written <- e
	}()
	// Nothing reads the slave, so bytes queued there are progress the 1 MiB
	// write can never finish past.
	for !readable(t, slave) {
		time.Sleep(time.Millisecond)
	}
	if _, err = slave.Write([]byte("reader-alive")); err != nil {
		t.Fatal(err)
	}
	if got := <-read; got.err != nil || got.data != "reader-alive" {
		t.Fatalf("%+v", got)
	}
	cancel()
	if err = <-written; !errors.Is(err, context.Canceled) {
		t.Fatalf("write: %v", err)
	}
	// The reader and the writer each sleep a poll slice per wakeup, so the
	// count scales with the time they were held, however long that was.
	if limit := 2*uint64(time.Since(started)/pollSlice) + 4; transport.PollWaits() > limit {
		t.Fatalf("busy poll: %d waits, at most %d expected", transport.PollWaits(), limit)
	}
	if err = transport.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	// Darwin records a kernel-owned written bit after any ordinary PTY write.
	// Verify mutable flags, not that unrelated kernel observation.
	mask := unix.O_NONBLOCK | unix.O_ASYNC | unix.O_APPEND
	if err != nil || after&mask != before&mask {
		t.Fatalf("flags: %x -> %x (%v)", before, after, err)
	}
}

func TestFileCloseCancelsReadAndRejectsLateIO(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	f, err := NewFile(master, master, false)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, e := f.Read(make([]byte, 1)); done <- e }()
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-done:
		if !errors.Is(e, os.ErrClosed) {
			t.Fatalf("%v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("read not joined")
	}
	if _, e := f.WriteContext(context.Background(), []byte("late")); !errors.Is(e, os.ErrClosed) {
		t.Fatalf("%v", e)
	}
}

func TestFakeModelsPartialErrorAndCancellation(t *testing.T) {
	f := NewFake()
	f.Enqueue(WriteStep{Limit: 2, Err: io.ErrUnexpectedEOF})
	n, err := f.WriteContext(context.Background(), []byte("abcd"))
	if n != 2 || !errors.Is(err, io.ErrUnexpectedEOF) || string(f.Bytes()) != "ab" {
		t.Fatalf("%d %v %q", n, err, f.Bytes())
	}
	hold := make(chan struct{})
	f.Enqueue(WriteStep{Block: hold})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n, e := f.WriteContext(ctx, []byte("lost")); n != 0 || !errors.Is(e, context.Canceled) {
		t.Fatalf("%d %v", n, e)
	}
	if string(f.Bytes()) != "ab" {
		t.Fatal("cancelled write changed state")
	}
}

// stalledOutput wraps a raw PTY slave as the output a presenter writes to. Its
// master is the host terminal: until the test reads it, output backs up exactly
// as it does when the host stops draining (#383).
func stalledOutput(t *testing.T) (host, *File) {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { master.Close(); slave.Close() })
	raw, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { term.Restore(int(slave.Fd()), raw) })
	out, err := NewFile(nil, slave, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { out.Close() })
	reader, err := NewFile(master, master, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reader.Close() })
	return host{reader}, out
}

// patterned bytes make a duplicated or replayed span visible in a comparison.
func patterned(n int) []byte {
	p := make([]byte, n)
	for i := range p {
		p[i] = byte('a' + i%23)
	}
	return p
}

// host reads the master side through its own File: Darwin PTY masters do not
// support os.File read deadlines.
type host struct{ f *File }

// drainExactly reads the host side until it holds want bytes, then checks
// that nothing more is queued. The write under test has returned (or will
// finish within want), so its every accepted byte is already in the kernel:
// no quiet window has to guess when it ends. It reports with t.Error, so it is
// safe on a goroutine.
func (h host) drainExactly(t *testing.T, want int) []byte {
	got := make([]byte, 0, want)
	buf := make([]byte, 64<<10)
	for len(got) < want {
		n, err := h.f.ReadContext(context.Background(), buf[:min(len(buf), want-len(got))])
		got = append(got, buf[:n]...)
		if err != nil {
			t.Error(err)
			return got
		}
	}
	if n, err := unix.Read(h.f.input, buf); n > 0 || !errors.Is(err, unix.EAGAIN) {
		t.Errorf("host has %d more bytes queued past the %d written (%v)", max(n, 0), want, err)
	}
	return got
}

// readable reports whether bytes wait unread on f.
func readable(t *testing.T, f *os.File) bool {
	t.Helper()
	fds := []unix.PollFd{{Fd: int32(f.Fd()), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	if err != nil && !errors.Is(err, unix.EINTR) {
		t.Fatal(err)
	}
	return n > 0 && fds[0].Revents&unix.POLLIN != 0
}

// The caller's deadline is the only one: a host that stalls past two seconds
// and resumes receives the whole write once. The caller sets no deadline, so a
// transport that imposed its own (#383) fails the write.
func TestFileWriteSurvivesTransientHostStallWithinCallerDeadline(t *testing.T) {
	master, out := stalledOutput(t)
	payload := patterned(256 << 10)
	drained := make(chan []byte, 1)
	go func() {
		time.Sleep(2500 * time.Millisecond)
		drained <- master.drainExactly(t, len(payload))
	}()
	start := time.Now()
	n, err := out.WriteContext(context.Background(), payload)
	elapsed := time.Since(start)
	got := <-drained
	if err != nil || n != len(payload) {
		t.Fatalf("write accepted %d/%d after %s: %v", n, len(payload), elapsed, err)
	}
	if elapsed < 2500*time.Millisecond {
		t.Fatalf("write finished in %s, before the host resumed: stall not exercised", elapsed)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("host received %d bytes, want the %d written exactly once", len(got), len(payload))
	}
}

// A stall that outlasts the caller's deadline fails at that deadline, not
// earlier, and the accepted count is exactly what reached the host. A write
// that outlived its deadline would block on the stalled host until the test
// binary's timeout.
func TestFilePersistentStallFailsAtCallerDeadlineWithExactPrefix(t *testing.T) {
	master, out := stalledOutput(t)
	payload := patterned(256 << 10)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	n, err := out.WriteContext(ctx, payload)
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline", err)
	}
	if elapsed < 2900*time.Millisecond {
		t.Fatalf("failed after %s, before the caller's 3s deadline", elapsed)
	}
	if n <= 0 || n >= len(payload) {
		t.Fatalf("accepted %d/%d, want a partial prefix", n, len(payload))
	}
	if got := master.drainExactly(t, n); !bytes.Equal(got, payload[:n]) {
		t.Fatalf("host received %d bytes, write reported %d", len(got), n)
	}
}

// Cancellation mid-stall returns at the write's next wakeup and keeps the same
// accounting. Prompt is counted in poll waits, not wall time: a writer that
// kept waiting after the cancel would poll again (or never return).
func TestFileCancelDuringHostStallReturnsPromptlyWithExactPrefix(t *testing.T) {
	master, out := stalledOutput(t)
	payload := patterned(256 << 10)
	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		n   int
		err error
	}
	done := make(chan result, 1)
	go func() { n, err := out.WriteContext(ctx, payload); done <- result{n, err} }()
	for out.PollWaits() == 0 { // the write is blocked on the full host
		time.Sleep(time.Millisecond)
	}
	// Cancel first, then count: any wait that began before the count is in
	// it, so at most the one the writer was entering can follow.
	cancel()
	before := out.PollWaits()
	r := <-done
	if !errors.Is(r.err, context.Canceled) || r.n <= 0 {
		t.Fatalf("accepted %d: %v, want a partial prefix and cancellation", r.n, r.err)
	}
	if after := out.PollWaits(); after > before+1 {
		t.Fatalf("write waited %d more times after the cancel", after-before)
	}
	if got := master.drainExactly(t, r.n); !bytes.Equal(got, payload[:r.n]) {
		t.Fatalf("host received %d bytes, write reported %d", len(got), r.n)
	}
}
