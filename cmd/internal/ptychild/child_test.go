package ptychild

import (
	"bytes"
	"context"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"os"
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
		o.Sink = func(context.Context, OutputBatch) error { mu.Lock(); chunks++; mu.Unlock(); return nil }
	})
	c.Wait()
	waitFor(t, "the pump to finish", func() bool { return c.Done() })

	mu.Lock()
	defer mu.Unlock()
	if chunks == 0 {
		t.Fatal("the sink never saw the child's output")
	}
}

func TestChildPublishesBellOnce(t *testing.T) {
	var mu sync.Mutex
	count := 0
	c := startSh(t, `printf '\007'; exit 0`, func(o *Options) {
		o.Sink = func(_ context.Context, b OutputBatch) error {
			mu.Lock()
			defer mu.Unlock()
			for _, e := range b.Terminal.Effects {
				if e.Kind == terminal.BellEffect {
					count++
				}
			}
			return nil
		}
	})
	c.Wait()
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("bell effects=%d", count)
	}
}

// Diagnostic snapshots include output before its publication callback runs.
func TestChildSinkSeesEveryChunkAndTheRingIsCurrentFirst(t *testing.T) {
	var mu sync.Mutex
	var ringAtSink [][]byte
	var child *Child
	registered := make(chan struct{})

	o := Options{
		Argv: []string{"sh", "-c", "printf one; sleep 0.1; printf two; sleep 5"},
		Size: Size{Rows: 24, Cols: 80},
		Sink: func(ctx context.Context, batch OutputBatch) error {
			select {
			case <-registered:
			case <-ctx.Done():
				return ctx.Err()
			}
			mu.Lock()
			defer mu.Unlock()
			ringAtSink = append(ringAtSink, child.Snapshot())
			return nil
		},
	}
	child, err := Start(o)
	close(registered)
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
	// triggered it: the ring is updated before the sink runs.
	if !bytes.Contains(ringAtSink[0], []byte("one")) {
		t.Fatalf("ring was behind the sink: first chunk saw ring %q", ringAtSink[0])
	}
}

func TestChildScreenTracksAltScreen(t *testing.T) {
	c := startSh(t, `printf '\033[?1049h'; sleep 5`)
	waitFor(t, "alt screen", func() bool { return c.Endpoint().Modes().AltScreen })
}

func TestChildStartRejectsEmptyArgv(t *testing.T) {
	if _, err := Start(Options{Size: Size{Rows: 24, Cols: 80}}); err == nil {
		t.Fatal("Start with no argv returned nil error")
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
				_ = c.Snapshot()
				_, _ = c.Endpoint().Snapshot(time.Now())
				_ = c.Endpoint().Modes()
			}
		}()
	}
	wg.Wait()
}

// ARCH-MOCK conformance: the fake and a real Child must agree on the lifecycle,
// driven through ONE shared scenario rather than asserted separately.
//
// The M1 boundary review found the fake's documented Wait/Done semantics were
// the opposite of the real thing's (BR-3), which no test could catch because no
// test compared them. A test written from that doc would have HUNG in M2/M3
// rather than failed. hostty_test.go's OSHost-vs-FakeHost check is the template.
func TestFakeChildConformsToRealChildLifecycle(t *testing.T) {
	// `sleep 5` stands in for "running"; Exit(0)/Close ends each the same way.
	real := startSh(t, "sleep 5")
	fake := NewFakeChild(nil)
	t.Cleanup(func() { _ = fake.Close() })

	for name, c := range map[string]*Child{"real": real, "fake": fake} {
		if c.Done() {
			t.Fatalf("%s: Done() == true while still running", name)
		}
		waited := make(chan int, 1)
		go func(c *Child) { waited <- c.Wait() }(c)
		select {
		case <-waited:
			t.Fatalf("%s: Wait() returned while still running", name)
		case <-time.After(150 * time.Millisecond):
		}

		if err := c.Close(); err != nil && name == "fake" {
			t.Fatalf("%s: Close: %v", name, err)
		}
		select {
		case <-waited:
		case <-time.After(3 * time.Second):
			t.Fatalf("%s: Wait() did not return after Close", name)
		}
		if !c.Done() {
			t.Fatalf("%s: Done() == false after the child ended", name)
		}
	}
}

// Exit is the fake's stand-in for the process exiting, so it must carry a code
// the way a real child's exit does.
func TestFakeChildExitReportsItsCode(t *testing.T) {
	c := NewFakeChild(nil)
	c.Exit(7)
	if got := c.Wait(); got != 7 {
		t.Fatalf("Wait() = %d, want 7", got)
	}
	if !c.Done() {
		t.Fatal("Done() == false after Exit")
	}
}

// BR-18's class: a conformance test that stops AT the terminal transition
// proves nothing about what happens past it, and past it is where a fake most
// easily diverges. This drives the WHOLE post-terminal stimulus set through
// both implementations and requires them to agree on error-vs-success.
func TestFakeAndRealChildAgreeAfterTheChildHasEnded(t *testing.T) {
	real := startSh(t, "sleep 5")
	fake := NewFakeChild(nil)

	if err := real.Close(); err != nil {
		t.Fatalf("real Close: %v", err)
	}
	real.Wait()
	if err := fake.Close(); err != nil {
		t.Fatalf("fake Close: %v", err)
	}
	fake.Wait()

	stimuli := []struct {
		name string
		call func(*Child) error
	}{
		{"Write", func(c *Child) error { _, err := c.Write([]byte("x")); return err }},
		{"Resize", func(c *Child) error { return c.Resize(Size{Rows: 10, Cols: 10}) }},
		{"Signal", func(c *Child) error { return c.Signal(os.Interrupt) }},
	}
	for _, st := range stimuli {
		t.Run(st.name, func(t *testing.T) {
			realErr := st.call(real) != nil
			fakeErr := st.call(fake) != nil
			if realErr != fakeErr {
				t.Fatalf("post-exit %s: real errors=%v, fake errors=%v — the fake lets through a call production rejects",
					st.name, realErr, fakeErr)
			}
			if !realErr {
				t.Fatalf("post-exit %s succeeded on a real child; the expectation itself is wrong", st.name)
			}
		})
	}

	// Idempotence, on both.
	for name, c := range map[string]*Child{"real": real, "fake": fake} {
		if err := c.Close(); err != nil {
			t.Fatalf("%s: second Close returned %v", name, err)
		}
	}
}

// Both real and fake children have valid geometry before their first resize.
func TestFakeAndRealChildAgreeOnGeometryBeforeAnyoneResizes(t *testing.T) {
	real := startSh(t, "sleep 5")
	t.Cleanup(func() { _ = real.Close() })
	fake := NewFakeChild(nil)
	t.Cleanup(func() { _ = fake.Close() })

	for name, c := range map[string]*Child{"real": real, "fake": fake} {
		if got := c.Size(); got.Rows < 2 || got.Cols == 0 {
			t.Fatalf("%s: a fresh child reports Size() = %v — a child with no geometry "+
				"cannot model production terminal state", name, got)
		}
	}
	if got, want := real.Size(), (Size{Rows: 24, Cols: 80}); got != want {
		t.Fatalf("real child started at %v, want %v — the size startSh asked for", got, want)
	}

}
