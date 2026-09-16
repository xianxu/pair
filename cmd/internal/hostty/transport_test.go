package hostty

import (
	"context"
	"errors"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"os"
	"testing"
	"time"
)

func TestOSHostOwnsFlagsThroughMeasurementAndJoinedRead(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	fd := int(slave.Fd())
	before, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := NewOSHost(slave, slave)
	defer h.Close()
	restore, err := h.MakeRaw()
	if err != nil {
		t.Fatal(err)
	}
	defer restore()
	if _, err = h.Size(); err != nil {
		t.Fatal(err)
	}
	acquired, _ := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if acquired&unix.O_NONBLOCK == 0 {
		t.Fatal("measurement/raw mode surrendered nonblocking ownership")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { var b [1]byte; _, err := h.ReadContext(ctx, b[:]); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("input read did not cancel")
	}
	if err = h.Close(); err != nil {
		t.Fatal(err)
	}
	after, _ := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if before != after {
		t.Fatalf("flags before%x after%x", before, after)
	}
	if _, err = h.WriteContext(context.Background(), []byte("late")); !errors.Is(err, os.ErrClosed) {
		t.Fatal(err)
	}
}
