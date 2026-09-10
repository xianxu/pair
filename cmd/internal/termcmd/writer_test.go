package termcmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/hostty"
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

// reset clears the recorded body so a test can assert on what happened AFTER a
// setup step, rather than on everything since the mux started.
func (w *writerRecorder) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.body.Reset()
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
			m.redrawTab([]byte("redraw"), hostty.ChildModes{})
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

	m.redrawTab([]byte("fresh"), hostty.ChildModes{})
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
	m.redrawTab([]byte("replaced"), hostty.ChildModes{})
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

// BR-35: the gate MODELS the terminal, so it must be fed exactly what the
// terminal is shown. Two directions, and the first shipped broken.
func TestTheGateSeesExactlyWhatTheTerminalSees(t *testing.T) {
	t.Run("a background tab cannot pin the gate", func(t *testing.T) {
		rec := newWriterRecorder()
		m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
		defer close(m.done)
		go m.copyActiveOutput()
		m.tabs = append(m.tabs,
			&terminalTab{id: 1, name: "front"},
			&terminalTab{id: 2, name: "back"})
		m.active = 0

		// Tab 2 is NOT active: its bytes are never written, so the terminal
		// never enters that sequence and the gate must not think it did.
		m.output <- ptyChunk{id: 2, data: []byte("noise\x1b[3")}
		m.drainForTest()
		if m.midSequenceForTest() {
			t.Fatal("a background tab's partial escape pinned the gate; paints would defer forever")
		}

		// And a paint must therefore land immediately.
		m.paintOwn([]byte("PAINT"))
		m.drainForTest()
		if !strings.Contains(rec.String(), "PAINT") {
			t.Fatalf("the paint was deferred against a sequence the terminal never saw: %q", rec.String())
		}
	})

	t.Run("a takeover's replay is fed", func(t *testing.T) {
		rec := newWriterRecorder()
		m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
		defer close(m.done)
		go m.copyActiveOutput()
		m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
		m.active = 0

		// The replay ends mid-sequence. Those bytes ARE written, so the gate
		// must know the terminal is inside a sequence -- otherwise the next
		// paint lands in the middle of it.
		m.redrawTab([]byte("restored\x1b[3"), hostty.ChildModes{})
		m.drainForTest()
		if !m.midSequenceForTest() {
			t.Fatal("the gate is blind to a sequence the terminal was shown by the replay")
		}
		m.paintOwn([]byte("PAINT"))
		m.drainForTest()
		if strings.Contains(rec.String(), "PAINT") {
			t.Fatalf("a paint landed inside the replay's own escape sequence: %q", rec.String())
		}
	})
}

// BR-25 (Critical): a handler running ON the writer goroutine must not post to
// the channel that goroutine drains.
//
// removeTab's only caller is handleChunk, on a child's EOF, and it ended with
// redrawTab -> enqueue. With the buffer full -- a child exiting while its
// output is backed up, which is precisely when children exit under load -- the
// send blocks forever, because the only goroutine that could drain it is the
// one blocked in the send. The pane wedges permanently.
//
// The test fills the buffer first, so a reentrant post cannot succeed by luck.
func TestAChildExitingWithAFullBufferDoesNotWedgeThePane(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	m.tabs = append(m.tabs,
		&terminalTab{id: 1, name: "one"},
		&terminalTab{id: 2, name: "two"})
	m.active = 0

	// NO LOOP RUNNING, and the buffer saturated. This is the writer
	// goroutine's own situation at the moment it handles an EOF while output is
	// backed up: nothing is draining, because the drainer is the caller.
	//
	// Deliberately not staged by racing a real loop -- an earlier version did
	// that and passed against the reverted fix, because the loop drained the
	// buffer before removeTab ever posted. A hazard that only reproduces
	// sometimes is not pinned by a test that only reproduces it sometimes.
	for i := 0; i < cap(m.output); i++ {
		m.output <- ptyChunk{id: 1, data: []byte("x")}
	}

	done := make(chan struct{})
	go func() { m.removeTab(2); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("removeTab blocked: a handler on the writer goroutine posted to its own channel")
	}
}

// BR-39: the takeover feeds the replay to the gate, so anything written AFTER
// it must consult the gate too. Writing owed diagnostics straight to the pane
// put them inside the replay's own open sequence -- the corruption this
// milestone exists to prevent, reintroduced by the fix for BR-35.
func TestADiagnosticNeverLandsInsideTheReplaysOpenSequence(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// Owe a diagnostic, then take over with a replay that ends mid-sequence.
	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.reportError(errors.New("owed failure"))
	m.drainForTest()
	m.redrawTab([]byte("restored\x1b[3"), hostty.ChildModes{})
	m.drainForTest()

	if strings.Contains(rec.String(), "owed failure") {
		t.Fatalf("a diagnostic landed inside the replay's open sequence: %q", rec.String())
	}

	// It is OWED, not dropped: the next boundary must deliver it.
	m.output <- ptyChunk{id: 1, data: []byte("m")}
	m.drainForTest()
	if !strings.Contains(rec.String(), "owed failure") {
		t.Fatalf("the diagnostic was dropped rather than deferred: %q", rec.String())
	}
}

// BR-39's CLASS half, and the shape it settled into after two attempts.
//
// The first version scanned run.go for `m.stdout` and required each hit to be
// gated or marked. That was weaker than it looked: it missed `fmt.Fprintf`
// (the most idiomatic spelling, and the one this file used pre-M2), read only
// one file while M3 adds a second to the package, and could be satisfied by an
// unrelated comment above the write. Scanning for violations is weaker than
// making them unrepresentable.
//
// So `paneWriter` is deliberately NOT an io.Writer. With no Write method,
// `fmt.Fprintf(m.pane, …)`, `io.WriteString(m.pane, …)` and `m.pane.Write(…)`
// are COMPILE ERRORS -- the door cannot be opened by reflex, in this file or
// any future one. This test states that property so a refactor that adds a
// Write method (making the whole design silently inert) fails here rather than
// in a review three rounds later.
func TestPaneWriterIsNotAnIOWriter(t *testing.T) {
	var p any = paneWriter{w: io.Discard}
	if _, ok := p.(io.Writer); ok {
		t.Fatal("paneWriter satisfies io.Writer; Fprintf and WriteString can now " +
			"open an ungated door to the pane, which is what this type prevents")
	}
}

// And the mux must hold no other route to the fd. A second field of a writer
// type would restore exactly what paneWriter removes.
func TestTheMuxHoldsNoRawWriterToThePane(t *testing.T) {
	typ := reflect.TypeOf(terminalMux{})
	writer := reflect.TypeOf((*io.Writer)(nil)).Elem()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Name == "stderr" {
			continue // startup diagnostics, before the loop exists
		}
		if f.Type.Implements(writer) {
			t.Fatalf("terminalMux.%s is an io.Writer; every route to the pane "+
				"must go through paneWriter", f.Name)
		}
	}
}

// The subprocess gets NO descriptor -- including stdin, which is in raw mode and
// carries the operator's keystrokes. Code and atlas both claimed "neither
// descriptor" while a third one was being handed over (BR-41).
func TestTheZellijSubprocessGetsNoStdinEither(t *testing.T) {
	src, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read run.go: %v", err)
	}
	if strings.Contains(string(src), "cmd.Stdin = os.Stdin") {
		t.Fatal("runZellij hands the subprocess the pane's raw-mode stdin; it can eat keystrokes")
	}
}

// The three mutations round 10 measured GREEN. Each is a real behaviour with no
// test behind it, which is the same "correct but unpinned" shape that has cost
// this milestone several rounds.
func TestTheTakeoverResetIsLoadBearing(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// Leave the gate mid-sequence, then take over with a replay of PURE DIGITS.
	//
	// The digits matter. A first version replayed "clean", and it passed with
	// the reset deleted: `c` is a valid CSI final byte (0x40-0x7e), so the
	// replay silently TERMINATED the stale `\x1b[3` and cleared the gate by
	// accident. Digits are CSI parameters, so they cannot terminate anything --
	// with the reset gone the stale sequence stays open and the paint is
	// deferred, which is the failure this test is for.
	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.drainForTest()
	m.redrawTab([]byte("12345"), hostty.ChildModes{})
	m.drainForTest()
	if m.midSequenceForTest() {
		t.Fatal("the takeover did not reset the scan; the old screen's partial sequence still gates writes")
	}
	m.paintOwn([]byte("PAINT"))
	m.drainForTest()
	if !strings.Contains(rec.String(), "PAINT") {
		t.Fatalf("a paint was deferred against a screen that no longer exists: %q", rec.String())
	}
}

// An owed write must not be stranded by a silent child. flushOwed ran only on
// the child-data branch, so with nothing arriving the owed write waited
// indefinitely -- the "stale row that nothing repaints" failure writeOwn's own
// comment says the owing prevents. Latent in M2, operator-visible in M3.
// A takeover DROPS the owed paint. Deleting `m.owed = nil` was measured green,
// because every existing takeover test then flushed the owed paint at the next
// boundary and could not tell "dropped" from "deferred once more".
func TestATakeoverDropsTheOwedPaintRatherThanDeferringIt(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3")}
	m.paintOwn([]byte("DOOMED"))
	m.drainForTest()
	m.redrawTab([]byte("12345"), hostty.ChildModes{}) // digits: cannot terminate the stale CSI
	m.drainForTest()

	// Drive the stream to a clean boundary. A DEFERRED paint would land here;
	// a DROPPED one never can.
	m.output <- ptyChunk{id: 1, data: []byte("plain text")}
	m.drainForTest()
	if strings.Contains(rec.String(), "DOOMED") {
		t.Fatalf("the owed paint survived a takeover and landed against a screen "+
			"that no longer exists: %q", rec.String())
	}
}

func TestAnOwedWriteIsNotStrandedByASilentChild(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// Mid-sequence, then the child goes quiet forever.
	m.output <- ptyChunk{id: 1, data: []byte("x\x1b[3m")} // completes; gate clear
	m.drainForTest()
	m.paintOwn([]byte("FIRST"))
	m.drainForTest()
	if !strings.Contains(rec.String(), "FIRST") {
		t.Fatalf("a paint on a clear gate did not land: %q", rec.String())
	}

	// Now genuinely mid-sequence, owe a paint, and deliver only CONSOLE events.
	m.output <- ptyChunk{id: 1, data: []byte("\x1b[3")}
	m.paintOwn([]byte("OWED"))
	m.drainForTest()
	if strings.Contains(rec.String(), "OWED") {
		t.Fatalf("the paint landed mid-sequence: %q", rec.String())
	}
	// A takeover is a console event and clears the gate; the owed paint is
	// dropped by design there, so use a completing child chunk instead and
	// assert the flush happens without waiting for a SECOND chunk.
	m.output <- ptyChunk{id: 1, data: []byte("m")}
	m.drainForTest()
	if !strings.Contains(rec.String(), "OWED") {
		t.Fatalf("the owed paint was stranded: %q", rec.String())
	}
}

// resizeThroughWriter must actually run on the loop. Reverting it to a direct
// inheritSize call was measured green: nothing observed that resize and paint
// serialize, which is what ARCH-ORDER asserts.
func TestResizeRunsOnTheWriterGoroutine(t *testing.T) {
	rec := newWriterRecorder()
	m := newTerminalMux("sh", nil, rec, io.Discard, &fakeRuntime{})
	defer close(m.done)
	go m.copyActiveOutput()
	m.tabs = append(m.tabs, &terminalTab{id: 1, name: "one"})
	m.active = 0

	// resizeThroughWriter must SERIALIZE against the loop, which is what
	// ARCH-ORDER asserts and what calling inheritSize directly would break.
	// Observed by the goroutine identity the work actually runs on: the loop's.
	loopID := make(chan string, 1)
	m.enqueue(ptyChunk{onWriter: func() { loopID <- goroutineID() }})
	m.drainForTest()

	// The PRODUCTION entry point, driven with a fake host -- an earlier version
	// called the injectable helper directly, so mutating resizeThroughWriter
	// (the caller the resize goroutine actually uses) left it green. Testing the
	// seam is not testing the path.
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80})
	resizeID := make(chan string, 1)
	m.captureIDForTest = func() { resizeID <- goroutineID() }
	m.resizeThroughWriter(host)
	m.drainForTest()

	got, want := <-resizeID, <-loopID
	if got != want {
		t.Fatalf("resize ran on goroutine %q, the writer loop is %q; they do not "+
			"serialize, so a resize can land between a paint's bytes", got, want)
	}
}

// The resize GOROUTINE must post through the loop, not call inheritSize
// directly. Pinning resizeThroughWriter's behaviour does not pin that its
// caller uses it -- reverting the call site was measured green while the
// behaviour test stayed happy, which is "testing the seam, not the path" one
// level up.
//
// A source check rather than an integration test: driving runShell's resize
// goroutine needs a real pty and a real host, and the property here is
// structural — no caller outside the writer loop reaches inheritSize directly.
func TestTheResizeGoroutinePostsThroughTheLoop(t *testing.T) {
	src, err := os.ReadFile("run.go")
	if err != nil {
		t.Fatalf("read run.go: %v", err)
	}
	lines := strings.Split(string(src), "\n")
	var offenders []string
	for i, line := range lines {
		if !strings.Contains(line, "inheritSize(") {
			continue
		}
		// The definition, and the one call inside the posted closure, are the
		// legitimate mentions.
		if strings.Contains(line, "func (m *terminalMux) inheritSize") ||
			strings.Contains(line, "m.inheritSize(host)") {
			continue
		}
		// A `loop-exempt: <reason>` marker in the comment block above, same
		// vocabulary as the gate's exemptions.
		exempt := false
		for j := i; j >= 0 && j > i-6; j-- {
			if strings.Contains(lines[j], "loop-exempt:") {
				exempt = true
				break
			}
			if t := strings.TrimSpace(lines[j]); j < i && t != "" && !strings.HasPrefix(t, "//") {
				break
			}
		}
		if exempt {
			continue
		}
		offenders = append(offenders,
			"run.go:"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
	}
	if len(offenders) > 0 {
		t.Fatalf("inheritSize reached outside the writer loop; a resize can then "+
			"land between a paint's bytes:\n  %s", strings.Join(offenders, "\n  "))
	}
	if !strings.Contains(string(src), "mux.resizeThroughWriter(host)") {
		t.Fatal("the resize goroutine no longer posts through the writer loop")
	}
}
