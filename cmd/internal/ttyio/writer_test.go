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
	go func() { buf := make([]byte, 16); n, e := transport.Read(buf); read <- result{string(buf[:n]), e} }()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
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
	if _, err = slave.Write([]byte("reader-alive")); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-read:
		if got.err != nil || got.data != "reader-alive" {
			t.Fatalf("%+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("reader starved by writer")
	}
	select {
	case err = <-written:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("write: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("write did not cancel")
	}
	if transport.PollWaits() > 100 {
		t.Fatalf("busy poll: %d", transport.PollWaits())
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

// drainUntilQuiet reads the host side until it has been silent for quiet. It
// reports with t.Error, so it is safe on a goroutine.
func (h host) drainUntilQuiet(t *testing.T, quiet time.Duration) []byte {
	var got []byte
	buf := make([]byte, 64<<10)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), quiet)
		n, err := h.f.ReadContext(ctx, buf)
		cancel()
		got = append(got, buf[:n]...)
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Error(err)
			}
			return got
		}
	}
}

// The caller's deadline is the only one: a host that stalls past two seconds
// and resumes inside the caller's budget receives the whole write once.
func TestFileWriteSurvivesTransientHostStallWithinCallerDeadline(t *testing.T) {
	master, out := stalledOutput(t)
	payload := patterned(256 << 10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	drained := make(chan []byte, 1)
	go func() {
		time.Sleep(2500 * time.Millisecond)
		drained <- master.drainUntilQuiet(t, 300*time.Millisecond)
	}()
	start := time.Now()
	n, err := out.WriteContext(ctx, payload)
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
// earlier, and the accepted count is exactly what reached the host.
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
	if elapsed < 2900*time.Millisecond || elapsed > 4*time.Second {
		t.Fatalf("failed after %s, want the caller's 3s deadline", elapsed)
	}
	if n <= 0 || n >= len(payload) {
		t.Fatalf("accepted %d/%d, want a partial prefix", n, len(payload))
	}
	if got := master.drainUntilQuiet(t, 300*time.Millisecond); !bytes.Equal(got, payload[:n]) {
		t.Fatalf("host received %d bytes, write reported %d", len(got), n)
	}
}

// Cancellation stays responsive mid-stall and keeps the same accounting.
func TestFileCancelDuringHostStallReturnsPromptlyWithExactPrefix(t *testing.T) {
	master, out := stalledOutput(t)
	payload := patterned(256 << 10)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	n, err := out.WriteContext(ctx, payload)
	if elapsed := time.Since(start); !errors.Is(err, context.Canceled) || elapsed > time.Second {
		t.Fatalf("err = %v after %s, want prompt cancellation", err, elapsed)
	}
	if got := master.drainUntilQuiet(t, 300*time.Millisecond); !bytes.Equal(got, payload[:n]) {
		t.Fatalf("host received %d bytes, write reported %d", len(got), n)
	}
}
