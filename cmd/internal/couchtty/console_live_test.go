package couchtty

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// Live console checks: the real Console, a REAL pty child running a REAL
// full-screen app, and a real terminal emulator reading the screen.
//
// Why this exists rather than more fake-child tests: M2's first operator smoke
// found the reserved row vanishing as pair drew its first screen, and every
// emulator test I had was green. The gap was that a FAKE child only emits what
// the test feeds it, so it never emitted the startup clear a real full-screen
// app always does. nvim is in the real stack, does clear on startup, and emits
// the margin reset on exit -- the two things that actually broke.
//
// Gated on PAIR_LIVE_COUCH=1 with t.Skip and deliberately NO build tag, so it
// keeps compiling under `go test ./cmd/...` rather than rotting invisibly.
// Reachable via `make test-live`.
func liveConsoleOnly(t *testing.T) string {
	t.Helper()
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to run against a real pty child")
	}
	path, err := exec.LookPath("nvim")
	if err != nil {
		t.Fatalf("nvim not on PATH: %v -- this check needs the real app, not a stand-in", err)
	}
	return path
}

// startLiveChild wires the real thing: a pty child under the real Console, with
// a vt emulator as the screen.
func startLiveChild(t *testing.T, argv []string, rows, cols uint16) (*vtHost, *ptychild.Child, *Console) {
	t.Helper()
	host := newVTHost(rows, cols)
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	con := New(host, stdinR)

	child, err2 := ptychild.Start(ptychild.Options{
		Dir:  t.TempDir(),
		Argv: argv,
		Env:  []string{"TERM=xterm-256color"},
		Size: con.ChildSize(),
		Sink: func(chunk []byte) { con.Deliver("c1", chunk) },
	})
	if err2 != nil {
		t.Fatalf("start %v: %v", argv, err2)
	}
	con.Attach("c1", "brain", child)

	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		con.Stop()
		_ = child.Close()
		_ = stdinW.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})
	return host, child, con
}

// A real full-screen app, judged on the BYTE STREAM rather than the rendered
// screen.
//
// Scope, stated because it bounds what this proves: the vt harness does not
// faithfully render an alt-screen app -- nvim's own content comes back
// truncated, which is a limitation of reading `vt` this way, not of couch. So
// the nvim case asserts what the harness CAN judge, and the rendered-screen
// question stays an operator smoke item (Task 2.7) rather than being claimed
// here.
//
// What it does prove is the bug this file was written to find: a real
// full-screen app emits escape sequences that SPLIT across pty reads, and the
// console must not write its status row into the gap. The first run of this
// test produced
//
//	\x1b7\x1b[12;1H\x1b[2K[brain]\x1b8;82;88m
//
// -- a paint spliced into the middle of nvim's `\x1b[38;2;76;82;88m`. No
// fake-child test could produce it, because a fake only emits what the test
// hands it whole.
//
// It FINDS that class of bug; it does not PIN it. Reverting the fix leaves this
// test green, because whether a read boundary lands inside a sequence depends
// on kernel timing. The deterministic guard is
// TestConsoleNeverInjectsInsideAChildEscapeSequence, which constructs the
// boundary rather than hoping for one -- verified red on the revert. Both are
// worth having, and it is worth being explicit about which does which: a live
// test treated as a regression guard is a gated-only pin that also cannot
// fail.
func TestLiveConsoleNeverSplicesIntoARealChildsSequences(t *testing.T) {
	nvim := liveConsoleOnly(t)
	host, child, _ := startLiveChild(t, []string{nvim, "-u", "NONE"}, 12, 60)

	waitLong(t, "nvim to emit its first screen", func() bool {
		return len(child.Snapshot()) > 500
	})
	// Let it settle, then make it redraw so paints and output interleave.
	time.Sleep(500 * time.Millisecond)
	for _, keys := range []string{"\x1b:set number\r", "\x1bihello\x1b", "\x1b:set nonumber\r"} {
		if _, err := child.Write([]byte(keys)); err != nil {
			t.Fatalf("write: %v", err)
		}
		time.Sleep(250 * time.Millisecond)
	}

	if bad, ok := splicedPaint(host.Written()); ok {
		t.Fatalf("a status-row paint was spliced into the child's stream: %q", bad)
	}
	if !strings.Contains(host.Written(), "[brain]") {
		t.Fatal("the console never painted the reserved row at all")
	}
}

// splicedPaint looks for a paint that begins while an escape sequence is still
// open. A paint is `\x1b7`; it is legitimate only when the bytes before it end
// at a sequence boundary.
func splicedPaint(stream string) (string, bool) {
	for i := 0; i < len(stream); {
		j := strings.Index(stream[i:], "\x1b7")
		if j < 0 {
			return "", false
		}
		at := i + j
		var sc ptychild.Screen
		sc.Feed([]byte(stream[:at]))
		if sc.Pending() > 0 {
			lo := at - 24
			if lo < 0 {
				lo = 0
			}
			return stream[lo:minInt(at+24, len(stream))], true
		}
		i = at + 2
	}
	return "", false
}

// A child that scrolls hard, for real, through a real pty.
func TestLiveReservedRowSurvivesRealScrolling(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to run against a real pty child")
	}
	host, _, _ := startLiveChild(t, []string{"sh", "-c", "i=0; while [ $i -lt 200 ]; do echo real-scroll-$i; i=$((i+1)); done; sleep 5"}, 12, 60)

	waitLong(t, "the child to scroll", func() bool {
		return strings.Contains(host.childArea(), "real-scroll-199")
	})
	if got := host.row(12); !strings.Contains(got, "brain") {
		t.Fatalf("200 lines of real scrolling ate the row: %q", got)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
