package termcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// goroutineID is only sound inside a test: it parses the runtime's own stack
// header. It is the cheapest way to answer "how many goroutines touched this
// writer", which is the property M2 exists to establish and which no amount of
// reading the code proves once a channel is involved.
func goroutineID() string {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// "goroutine 17 [running]:" -> "17"
	f := strings.Fields(string(buf[:n]))
	if len(f) < 2 {
		return "unknown"
	}
	return f[1]
}

type writerRecorder struct {
	mu      sync.Mutex
	writers map[string]int
	body    bytes.Buffer
}

func newWriterRecorder() *writerRecorder {
	return &writerRecorder{writers: map[string]int{}}
}

func (w *writerRecorder) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writers[goroutineID()]++
	w.body.Write(p)
	return len(p), nil
}

func (w *writerRecorder) distinct() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.writers)
}

func (w *writerRecorder) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

// THE ENVELOPE, enforced rather than asserted: after M2 exactly one goroutine
// may write the pane. A second writer is how a paint lands inside a child's
// escape sequence -- couch documents the mechanism (atlas/couch.md): "a pty read
// boundary falls wherever the kernel puts it, so a paint written between two
// chunks can land inside one of the child's escape sequences."
//
// Before M2, redrawTab wrote from the Run goroutine while copyActiveOutput wrote
// from the pump, so this sees two.
func TestOnlyOneGoroutineWritesTheHost(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()

	// A tab so isActive() lets chunks through.
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // the pump's source
		defer wg.Done()
		for i := 0; i < 50; i++ {
			m.output <- ptyChunk{id: 1, data: []byte("chunk" + strconv.Itoa(i))}
		}
	}()
	go func() { // a concurrent redraw, as a tab switch would issue
		defer wg.Done()
		for i := 0; i < 50; i++ {
			m.redrawTab([]byte("redraw"))
		}
	}()
	wg.Wait()
	m.drainForTest()

	if got := rec.distinct(); got != 1 {
		t.Fatalf("%d goroutines wrote the pane; M2's envelope is one", got)
	}
}

// couch's lesson restated at the new consumer: a write issued while the child's
// stream is MID-SEQUENCE must be deferred, and OWED. Dropping it leaves a stale
// row that nothing will repaint.
func TestPaintDefersMidSequenceAndIsOwed(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// A chunk that ends INSIDE a CSI: "\x1b[3" is incomplete until its final.
	m.output <- ptyChunk{id: 1, data: []byte("before\x1b[3")}
	m.paintOwn([]byte("PAINT"))
	m.drainForTest()

	if strings.Contains(rec.String(), "PAINT") {
		t.Fatalf("a paint landed inside the child's escape sequence: %q", rec.String())
	}

	// The completing byte ends the sequence; the owed paint must land, once.
	m.output <- ptyChunk{id: 1, data: []byte("m after")}
	m.drainForTest()

	if n := strings.Count(rec.String(), "PAINT"); n != 1 {
		t.Fatalf("owed paint landed %d times, want exactly 1: %q", n, rec.String())
	}
}

// The half couch got wrong FIRST: the gate is fed CHILD bytes only. Feeding our
// own escapes in lets it frame our bytes with the child's partial and report
// safe precisely when it is not.
func TestGateIsNotFedOurOwnWrites(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// DIRECTION 1: our own PARTIAL escape must not leave the gate believing the
	// CHILD's stream is mid-sequence. The child's stream is clean here, so the
	// paint is written rather than deferred -- which is the path that actually
	// reaches the scanner if anyone wires it up wrong.
	m.output <- ptyChunk{id: 1, data: []byte("complete\x1b[0m")}
	m.drainForTest()
	if m.midSequenceForTest() {
		t.Fatal("setup: the child's stream should be at a boundary")
	}
	m.paintOwn([]byte("\x1b[3")) // OUR partial sequence
	m.drainForTest()
	if m.midSequenceForTest() {
		t.Fatal("our own partial escape poisoned the gate; it must see CHILD bytes only")
	}

	// DIRECTION 2: our own write must not COMPLETE a child's partial sequence.
	// Feeding both streams into one scanner lets it frame our bytes with the
	// child's and report safe precisely when it is not.
	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.drainForTest()
	if !m.midSequenceForTest() {
		t.Fatal("setup: the child's partial sequence did not open the gate")
	}
	m.paintOwn([]byte("m")) // textually completes the child's CSI
	m.drainForTest()
	if !m.midSequenceForTest() {
		t.Fatal("our own write closed the gate on the child's behalf")
	}
}

// redrawTab is termcmd's WHOLESALE TAKEOVER -- HomeAndClear plus a replay -- so
// it needs couch's third gate rule (console.go:992-995), not just the first two:
// whatever partial sequence the old content left is no longer on screen to be
// corrupted, so the scan resets and any owed paint is dropped rather than
// flushed against a stream that no longer exists.
func TestTakeoverResetsTheGateAndDropsTheOwedPaint(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.paintOwn([]byte("STALE"))
	m.drainForTest()
	if !m.midSequenceForTest() {
		t.Fatal("setup failed: the gate should be mid-sequence")
	}

	m.redrawTab([]byte("fresh"))
	m.drainForTest()

	if m.midSequenceForTest() {
		t.Fatal("a wholesale takeover left the gate mid-sequence against a screen that is gone")
	}
	// POSITIVE CONTROL FIRST. Without it this test passes when nothing is
	// written at all -- a broken writer, a dropped event, a recorder wired to
	// the wrong stream -- and an assert-absent that cannot fail proves nothing.
	if !strings.Contains(rec.String(), "fresh") {
		t.Fatalf("the takeover's own replay never reached the pane: %q", rec.String())
	}
	if strings.Contains(rec.String(), "STALE") {
		t.Fatalf("a paint owed against the OLD screen landed after the takeover: %q", rec.String())
	}
}

// A DIAGNOSTIC is not a paint: it is an event, so it survives both a coalescing
// repaint and a wholesale takeover. Losing it means losing the operator's only
// report that something failed.
func TestDiagnosticsAreQueuedNotCoalescedAndSurviveATakeover(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.reportError(errors.New("first failure"))
	m.reportError(errors.New("second failure"))
	m.paintOwn([]byte("PAINT"))
	m.drainForTest()

	// Nothing lands while the child is mid-sequence.
	if strings.Contains(rec.String(), "failure") {
		t.Fatalf("a diagnostic landed inside the child's escape sequence: %q", rec.String())
	}

	m.output <- ptyChunk{id: 1, data: []byte("m")}
	m.drainForTest()
	got := rec.String()
	for _, want := range []string{"first failure", "second failure"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q was coalesced away; diagnostics queue, they do not replace: %q", want, got)
		}
	}

	// And a takeover drops the owed PAINT but not the diagnostics.
	m.output <- ptyChunk{id: 1, data: []byte("y\x1b[3")}
	m.reportError(errors.New("third failure"))
	m.paintOwn([]byte("DOOMED"))
	m.drainForTest()
	m.redrawTab([]byte("replaced"))
	m.drainForTest()

	got = rec.String()
	if !strings.Contains(got, "third failure") {
		t.Fatalf("a takeover swallowed a diagnostic: %q", got)
	}
	if strings.Contains(got, "DOOMED") {
		t.Fatalf("a paint owed against the old screen survived the takeover: %q", got)
	}
}

// A gate is only as good as the scanner behind it, and ptychild.Screen is the
// one couch already uses. Pinned so a future refactor cannot quietly swap in a
// second implementation of "is this stream mid-sequence".
func TestGateUsesTheSharedScanner(t *testing.T) {
	var s ptychild.Screen
	s.FeedFraming([]byte("\x1b[3"))
	if !s.MidSequence() {
		t.Fatal("ptychild.Screen does not report a partial CSI; the gate's premise is wrong")
	}
	s.FeedFraming([]byte("m"))
	if s.MidSequence() {
		t.Fatal("ptychild.Screen stayed mid-sequence after the final byte")
	}
}

// M2.3b — the subprocess writers, asserted by the FD they are handed rather
// than by the method name.
//
// A `Quiet`-only refactor passes a method-name assertion while `cmd.Stderr`
// still points at the pane, which is how this survived three review rounds
// (BR-4). The subprocess is a different process: no goroutine-id test can see
// it, so this is the assertion that replaces the one that structurally cannot.
func TestNeitherZellijMethodHandsTheSubprocessThePanesDescriptors(t *testing.T) {
	// A real `zellij` is not needed: exec fails, and the failure is exactly
	// the case that used to write the pane's stderr per wheel tick.
	var stdout, stderr bytes.Buffer
	_ = runZellij([]string{"action", "nonexistent-verb"}, &stdout, &stderr)

	// The point is the plumbing: whatever the subprocess emits lands in the
	// writers it was given. If runZellij ignored them for os.Stdout/os.Stderr,
	// this test would still pass -- so assert the wiring directly too.
	// BOTH verbs, because each descriptor is only exercised by one of them:
	// a succeeding action writes stdout and nothing to stderr, a failing one
	// the reverse. Testing one verb leaves the other descriptor unchecked --
	// which is how the stderr wiring survived its first mutation check.
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"Action/succeeds", func() error { return OSRuntime{}.RunZellijAction("list-clients") }},
		{"Action/fails", func() error { return OSRuntime{}.RunZellijAction("nonexistent-verb") }},
		{"Quiet/succeeds", func() error { return OSRuntime{}.RunZellijActionQuiet("list-clients") }},
		{"Quiet/fails", func() error { return OSRuntime{}.RunZellijActionQuiet("nonexistent-verb") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// os.Stdout/os.Stderr are redirected to a pipe; anything the
			// subprocess writes there is a byte on the operator's pane.
			out, restoreOut := captureFD(t, &os.Stdout)
			errOut, restoreErr := captureFD(t, &os.Stderr)
			// POSITIVE CONTROL: prove the capture can SEE a write before
			// concluding from its silence. Without this, a captureFD that
			// silently failed would make every arm below pass.
			fmt.Fprint(os.Stdout, "control-out")
			fmt.Fprint(os.Stderr, "control-err")
			_ = tc.run()
			restoreOut()
			restoreErr()
			if !bytes.Contains(out(), []byte("control-out")) {
				t.Fatal("captureFD did not observe a direct stdout write; the assertions below are vacuous")
			}
			if !bytes.Contains(errOut(), []byte("control-err")) {
				t.Fatal("captureFD did not observe a direct stderr write; the assertions below are vacuous")
			}
			// Minus the control bytes, the subprocess must have written NOTHING.
			if got := bytes.ReplaceAll(out(), []byte("control-out"), nil); len(got) != 0 {
				t.Fatalf("%s wrote %d bytes to the pane's stdout: %q", tc.name, len(got), got)
			}
			if got := bytes.ReplaceAll(errOut(), []byte("control-err"), nil); len(got) != 0 {
				t.Fatalf("%s wrote %d bytes to the pane's stderr: %q", tc.name, len(got), got)
			}
		})
	}
}

// captureFD swaps a *os.File for a pipe and returns a reader plus a restore.
func captureFD(t *testing.T, target **os.File) (func() []byte, func()) {
	t.Helper()
	saved := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	*target = w
	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	var captured []byte
	var once sync.Once
	restore := func() {
		once.Do(func() {
			*target = saved
			_ = w.Close()
			captured = <-done
			_ = r.Close()
		})
	}
	return func() []byte { return captured }, restore
}

// BR-32: the fix for BR-27 routes an EXTERNAL PROCESS's bytes toward the pane.
// Keeping the pane clean and then handing it an arbitrary escape would be a
// worse bug than the silence it replaced -- `\x1b[2J` from a subprocess clears
// the operator's screen just as effectively as one from an agent's label.
func TestAZellijErrorCannotPutAnEscapeOrAnUnboundedLineOnThePane(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()

	hostile := errors.New("boom: \x1b[2Jcleared\x0egarbled\nforged-line " +
		strings.Repeat("x", 400))
	m.reportError(hostile)
	m.drainForTest()

	got := rec.String()
	// Positive control: the readable part must actually arrive, or the
	// assertions below pass because nothing was written.
	if !strings.Contains(got, "boom") {
		t.Fatalf("the diagnostic never reached the pane: %q", got)
	}
	for _, bad := range []string{"\x1b", "\x0e", "\n\x66orged", "\r"} {
		if strings.Contains(strings.TrimSuffix(got, "\r\n"), bad) {
			t.Fatalf("control byte %q from an external process reached the pane: %q", bad, got)
		}
	}
	if n := len(got); n > 300 {
		t.Fatalf("an external process put %d bytes on the pane; it must be bounded", n)
	}
}
