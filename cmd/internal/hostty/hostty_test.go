package hostty

import (
	"os"
	"strings"
	"syscall"
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
// This one genuinely needs a pty -- Size and MakeRaw are terminal operations.
// It FAILS rather than skips when one is unavailable, matching how ptychild
// handles the identical constraint: a suite that reports ok while silently
// skipping its conformance check is telling you the wrong thing (BR-15).
func TestOSHostConformsToTheFakeOnSizeAndRawMode(t *testing.T) {
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v -- this check needs a real terminal; run it unsandboxed", err)
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

// Close must release a `for range host.Resized()` consumer, for BOTH hosts.
// A watcher goroutine that never returns is the leak the M1 boundary review
// found (BR-2); asserting it on the fake alone would not have caught it, since
// the fake is not what production ranges over.
func TestCloseReleasesResizedConsumers(t *testing.T) {
	// Deliberately NOT a pty: OSHost's watcher and Close path touch neither the
	// terminal nor its size, so gating this on pty.Open would make the pin skip
	// itself in the sandboxed shell this issue documents as its own (BR-15).
	f, err := os.CreateTemp(t.TempDir(), "hostty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()

	hosts := map[string]Host{
		"OSHost":   NewOSHost(f, f),
		"FakeHost": NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}),
	}
	for name, h := range hosts {
		t.Run(name, func(t *testing.T) {
			returned := make(chan struct{})
			go func() {
				for range h.Resized() {
				}
				close(returned)
			}()
			if err := h.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			select {
			case <-returned:
			case <-time.After(2 * time.Second):
				t.Fatal("the resize consumer never returned after Close")
			}
		})
	}
}

// Coalescing on the REAL side. TestFakeHostResizesCoalesce drives the fake, but
// production depends on OSHost.watch()'s non-blocking send -- and a package
// whose stated purpose is making the SIGWINCH path testable should test it
// where it actually runs (BR-7).
func TestOSHostCoalescesRealSIGWINCH(t *testing.T) {
	// Real signals, no pty: the coalescing is in watch()'s non-blocking send,
	// which does not care what kind of file the host wraps (BR-15).
	f, err := os.CreateTemp(t.TempDir(), "hostty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()

	h := NewOSHost(f, f)
	defer func() { _ = h.Close() }()

	for i := 0; i < 50; i++ {
		_ = syscall.Kill(os.Getpid(), syscall.SIGWINCH)
	}
	time.Sleep(200 * time.Millisecond)

	n := 0
	for {
		select {
		case _, ok := <-h.Resized():
			if !ok {
				t.Fatal("Resized() closed early")
			}
			n++
			continue
		default:
		}
		break
	}
	if n > 1 {
		t.Fatalf("50 real SIGWINCHs delivered %d wakes; they must coalesce", n)
	}
	if n == 0 {
		t.Fatal("50 real SIGWINCHs delivered no wake at all")
	}
}

// BR-18: a post-Close SetSize used to PANIC on the fake ("send on closed
// channel") while OSHost absorbed a SIGWINCH burst inertly -- in the double
// M2's console tests are built on, so it would crash a run rather than fail it.
// Driving both past Close is the class fix; stopping at Close is what missed it.
func TestHostsAgreeAfterClose(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "hostty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()

	real := NewOSHost(f, f)
	fake := NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	for _, h := range []Host{real, fake} {
		if err := h.Close(); err != nil {
			t.Fatalf("%T Close: %v", h, err)
		}
	}

	// The real host's post-Close resize stimulus is a signal it no longer
	// watches; the fake's is SetSize. Neither may panic, and both must stay
	// inert. A burst, because one is not a burst.
	for i := 0; i < 20; i++ {
		_ = syscall.Kill(os.Getpid(), syscall.SIGWINCH)
		fake.SetSize(ptychild.Size{Rows: uint16(25 + i), Cols: 80})
	}
	time.Sleep(100 * time.Millisecond)

	for _, h := range []Host{real, fake} {
		if _, err := h.Write([]byte("post-close")); err != nil {
			t.Fatalf("%T: Write after Close returned %v; the other host does not", h, err)
		}
		if err := h.Close(); err != nil {
			t.Fatalf("%T: second Close returned %v", h, err)
		}
	}
}
