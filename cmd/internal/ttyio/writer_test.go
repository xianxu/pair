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
