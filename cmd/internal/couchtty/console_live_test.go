package couchtty

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
)

// Live conformance runs real applications on a PTY with the packaged virtual
// terminal profile. The parent screen is observed independently below.
// Gated at runtime so these paths keep compiling in ordinary CI.
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

	profileEnv, err := runtimebundle.TerminalEnvironment(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	child, err2 := ptychild.Start(ptychild.Options{
		Dir:  t.TempDir(),
		Argv: argv,
		Env:  profileEnv,
		Size: con.ChildSize(),
		Sink: func(ctx context.Context, batch ptychild.OutputBatch) error { return con.Deliver(ctx, "c1", batch) },
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

// Startup clears, alternate-screen use, cursor motion and redraws from a real
// editor must preserve both its buffer and the Console's reserved chrome.
func TestLiveConsoleNvimPreservesContentAndChrome(t *testing.T) {
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

	waitLong(t, "nvim buffer and reserved chrome", func() bool {
		return strings.Contains(host.childArea(), "hello") && strings.Contains(host.row(12), "brain")
	})
	assertNativeOracle(t, host.Written(), 60, 12, "hello", "brain")
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
