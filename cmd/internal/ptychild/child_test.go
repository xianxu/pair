package ptychild

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func startSh(t *testing.T, script string, opts ...func(*Options)) *Child {
	t.Helper()
	o := Options{
		Argv: []string{"sh", "-c", script},
		Size: Size{Rows: 24, Cols: 80},
	}
	for _, f := range opts {
		f(&o)
	}
	c, err := Start(o)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestChildWriteReachesTheChild(t *testing.T) {
	c := startSh(t, "cat")
	if _, err := c.Write([]byte("marker-one\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	waitFor(t, "the child to echo what was written", func() bool {
		return bytes.Contains(c.Snapshot(), []byte("marker-one"))
	})
}

// The child must OBSERVE the resize. Asserting that Resize returned nil proves
// only that the ioctl was accepted -- it would stay green with the size never
// reaching the process, which is the bug that matters for a one-row-shorter pty.
func TestChildResizeIsObservedByTheChild(t *testing.T) {
	c := startSh(t, "while :; do stty size; sleep 0.05; done")
	waitFor(t, "the initial size", func() bool {
		return bytes.Contains(c.Snapshot(), []byte("24 80"))
	})

	if err := c.Resize(Size{Rows: 30, Cols: 100}); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	waitFor(t, "the child to report the new size", func() bool {
		return bytes.Contains(c.Snapshot(), []byte("30 100"))
	})
}

func TestChildWaitReportsTheExitCode(t *testing.T) {
	c := startSh(t, "exit 7")
	if got := c.Wait(); got != 7 {
		t.Fatalf("Wait() = %d, want 7", got)
	}
}

// Exit must close the pump, or a console waiting on this child hangs forever.
func TestChildExitClosesThePump(t *testing.T) {
	var mu sync.Mutex
	chunks := 0
	c := startSh(t, "printf hello; exit 0", func(o *Options) {
		o.Sink = func([]byte) { mu.Lock(); chunks++; mu.Unlock() }
	})
	c.Wait()
	waitFor(t, "the pump to finish", func() bool { return c.Done() })

	mu.Lock()
	defer mu.Unlock()
	if chunks == 0 {
		t.Fatal("the sink never saw the child's output")
	}
}

func TestChildLatchesBell(t *testing.T) {
	c := startSh(t, `printf '\007'; sleep 5`)
	waitFor(t, "the bell", func() bool { return c.TakeBell() })
	if c.TakeBell() {
		t.Fatal("TakeBell did not clear the latch")
	}
}

// The sink is the live path and the ring is the replay path; a chunk must reach
// BOTH, and the ring must be current before the sink runs -- otherwise a switch
// racing a chunk replays a screen missing bytes the operator already saw.
func TestChildSinkSeesEveryChunkAndTheRingIsCurrentFirst(t *testing.T) {
	var mu sync.Mutex
	var ringAtSink [][]byte
	var child *Child

	o := Options{
		Argv: []string{"sh", "-c", "printf one; sleep 0.1; printf two; sleep 5"},
		Size: Size{Rows: 24, Cols: 80},
		Sink: func(p []byte) {
			mu.Lock()
			defer mu.Unlock()
			ringAtSink = append(ringAtSink, child.Snapshot())
		},
	}
	child, err := Start(o)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = child.Close() })

	waitFor(t, "both writes", func() bool {
		return bytes.Contains(child.Snapshot(), []byte("two"))
	})
	mu.Lock()
	defer mu.Unlock()
	if len(ringAtSink) == 0 {
		t.Fatal("the sink never ran")
	}
	// Every snapshot taken from inside Sink must already contain the chunk that
	// triggered it: the ring is updated before the sink runs, so a caller that
	// switches away inside Sink still repaints a current screen.
	if !bytes.Contains(ringAtSink[0], []byte("one")) {
		t.Fatalf("ring was behind the sink: first chunk saw ring %q", ringAtSink[0])
	}
}

func TestChildScreenTracksAltScreen(t *testing.T) {
	c := startSh(t, `printf '\033[?1049h'; sleep 5`)
	waitFor(t, "alt screen", func() bool { return c.AltScreen() })
}

func TestChildStartRejectsEmptyArgv(t *testing.T) {
	if _, err := Start(Options{Size: Size{Rows: 24, Cols: 80}}); err == nil {
		t.Fatal("Start with no argv returned nil error")
	}
}

// A repaint reads through StripQueries, so a child that emitted a capability
// query at startup cannot re-ask the host terminal on landing (#127).
func TestChildReplayStripsQueries(t *testing.T) {
	c := startSh(t, `printf '\033[cvisible'; sleep 5`)
	waitFor(t, "the output", func() bool {
		return bytes.Contains(c.Snapshot(), []byte("visible"))
	})
	if got := string(c.Replay()); strings.Contains(got, "\x1b[c") {
		t.Fatalf("Replay() contains a capability query: %q", got)
	}
}

// The snapshot/append race, moved here with the ring it belongs to. termcmd
// used to pin this at the mux level (TestRedrawSnapshotIsRaceFree); the lock is
// the Child's now, so the assertion follows the code rather than testing a call
// into a lock the caller does not own.
func TestChildSnapshotDuringPumpIsRaceFree(t *testing.T) {
	c := startSh(t, "i=0; while [ $i -lt 400 ]; do printf 'output\033[c'; i=$((i+1)); done; sleep 5")

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				_ = c.Replay()
				_ = c.AltScreen()
				_ = c.TakeBell()
			}
		}()
	}
	wg.Wait()
}
