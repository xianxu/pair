package hostty

import (
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestControlSequences(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"set region", SetRegion(1, 23), "\x1b[1;23r"},
		{"reset region", ResetRegion, "\x1b[r"},
		{"move to", MoveTo(24, 1), "\x1b[24;1H"},
		{"home and clear", HomeAndClear, "\x1b[1;1H\x1b[J"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestFakeHostReportsSizeAndCapturesWrites(t *testing.T) {
	h := NewFakeHost(ptychild.Size{Rows: 40, Cols: 120})
	got, err := h.Size()
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if got.Rows != 40 || got.Cols != 120 {
		t.Fatalf("Size() = %+v, want 40x120", got)
	}
	if _, err := h.Write([]byte("painted")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if !strings.Contains(h.Written(), "painted") {
		t.Fatalf("Written() = %q", h.Written())
	}
}

// Without a fireable resize channel, no console test can cover the SIGWINCH
// path -- which is the gap that made the first draft of #146's console tasks
// unbuildable.
func TestFakeHostResizeIsObservable(t *testing.T) {
	h := NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	h.SetSize(ptychild.Size{Rows: 30, Cols: 100})

	select {
	case <-h.Resized():
	case <-time.After(time.Second):
		t.Fatal("SetSize did not deliver a resize")
	}
	got, _ := h.Size()
	if got.Rows != 30 {
		t.Fatalf("Size() = %+v after SetSize", got)
	}
}

// A console restores on the child-exit path AND from a deferred teardown. A
// restore that is not idempotent turns the second call into a broken terminal.
func TestFakeHostRestoreIsIdempotent(t *testing.T) {
	h := NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	restore, err := h.MakeRaw()
	if err != nil {
		t.Fatalf("MakeRaw: %v", err)
	}
	if err := restore(); err != nil {
		t.Fatalf("first restore: %v", err)
	}
	if err := restore(); err != nil {
		t.Fatalf("second restore: %v", err)
	}
	if h.RawDepth() != 0 {
		t.Fatalf("RawDepth() = %d after two restores, want 0", h.RawDepth())
	}
}

// A window drag fires SIGWINCH continuously. Queueing one wake per signal would
// make the console resize every child N times for one drag.
func TestFakeHostResizesCoalesce(t *testing.T) {
	h := NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	for i := 0; i < 50; i++ {
		h.SetSize(ptychild.Size{Rows: uint16(25 + i), Cols: 80})
	}
	n := 0
	for {
		select {
		case <-h.Resized():
			n++
			continue
		default:
		}
		break
	}
	if n > 1 {
		t.Fatalf("50 resizes delivered %d wakes; they must coalesce", n)
	}
}

// Conformance: OSHost against a REAL pty. The fake models size reporting and
// raw-mode restore; this pins that the real thing agrees, rather than asserting
// whatever each happens to do separately.
func TestOSHostConformsToTheFakeOnSizeAndRawMode(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Skipf("pty.Open: %v (sandboxed?)", err)
	}
	defer func() { _ = ptmx.Close(); _ = tty.Close() }()

	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 33, Cols: 111}); err != nil {
		t.Fatalf("Setsize: %v", err)
	}

	real := NewOSHost(tty, tty)
	fake := NewFakeHost(ptychild.Size{Rows: 33, Cols: 111})

	rs, err := real.Size()
	if err != nil {
		t.Fatalf("OSHost.Size: %v", err)
	}
	fs, _ := fake.Size()
	if rs != fs {
		t.Fatalf("OSHost.Size() = %+v, FakeHost.Size() = %+v", rs, fs)
	}

	// Both must go raw and come back, twice, without erroring.
	for _, h := range []Host{real, fake} {
		restore, err := h.MakeRaw()
		if err != nil {
			t.Fatalf("%T MakeRaw: %v", h, err)
		}
		if err := restore(); err != nil {
			t.Fatalf("%T restore: %v", h, err)
		}
		if err := restore(); err != nil {
			t.Fatalf("%T second restore: %v", h, err)
		}
	}
}

func TestOSHostSizeErrorsOnANonTerminal(t *testing.T) {
	h := NewOSHost(nil, nil)
	if _, err := h.Size(); err == nil {
		t.Fatal("Size() on a nil terminal returned nil error")
	}
}
