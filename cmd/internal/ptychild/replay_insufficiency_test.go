package ptychild

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// #209 plan step 1: reproduce, deliberately, the four ways byte-replay-only
// reconstruction fails — before choosing a fix. Each is deterministic here, at
// the seam, rather than "sometimes" in a live session.
//
// The shared setup is a child whose ring is smaller than its history, which is
// the steady state of any long-lived pane: DefaultRingBytes is 128 KiB and an
// actively working agent emits that in roughly two minutes (measured in #171:
// 143 KB over 123 s on a busy pane).

func replayChild(capacity int) *Child {
	return &Child{ring: NewRing(capacity), screen: &Screen{}, done: make(chan struct{}), fake: &fakeState{}}
}

// Mode 1 — the last full paint aged out. The screen is cleared and only the
// increments since the paint are rewritten, so what lands is a partial frame.
// This is the mode the operator's report describes.
func TestReplayLosesTheLastFullPaintOnceItAgesOut(t *testing.T) {
	const paint = "\x1b[H\x1b[2JFULL-FRAME-CONTENT"
	child := replayChild(64)
	child.Feed([]byte(paint))
	// Ordinary incremental output afterwards, more than the ring holds.
	child.Feed([]byte(strings.Repeat("x", 200)))

	replay := child.ReplayThrough(child.ReplaySafeEnd())
	if bytes.Contains(replay, []byte("FULL-FRAME-CONTENT")) {
		t.Fatal("fixture did not age the paint out; the ring is too large for this test")
	}
	if bytes.Contains(replay, []byte("\x1b[2J")) {
		t.Fatal("fixture retained the clear; the paint did not age out")
	}
	// This is the defect: a consumer clears the screen and writes THIS, so the
	// operator sees increments over a blank frame with no full paint anywhere.
	t.Logf("mode 1 reproduced: replay is %d bytes of increments with no paint", len(replay))
}

// Mode 2 — the cutoff is older than the ring, so the replay is EMPTY. A
// consumer that clears then writes nothing leaves a blank screen.
func TestReplayIsEmptyWhenTheCutoffPredatesTheRing(t *testing.T) {
	child := replayChild(16)
	child.Feed([]byte(strings.Repeat("a", 100)))

	if replay := child.ReplayThrough(4); replay != nil {
		t.Fatalf("replay = %q, want nil — cutoff 4 is older than ringStart", replay)
	}
	t.Log("mode 2 reproduced: ReplayThrough returns nil, so clear-then-replay blanks the screen")
}

// Mode 3 — retention bisects an escape sequence, so a replay can begin inside
// one. The bytes that reach the terminal are then not a valid stream.
func TestReplayCanBeginInsideAnEscapeSequence(t *testing.T) {
	child := replayChild(8)
	child.Feed([]byte("\x1b[38;5;42mCOLOURED"))

	replay := child.ReplayThrough(child.ReplaySafeEnd())
	if bytes.HasPrefix(replay, []byte("\x1b")) {
		t.Fatalf("replay = %q, want a tail that starts mid-sequence", replay)
	}
	t.Logf("mode 3 reproduced: replay %q begins inside an SGR sequence, with no introducer", replay)
}

// Mode 4 — mode state is a property of the WHOLE stream, not its tail. An
// alt-screen enter that happened before the window cannot be replayed, so the
// child's actual screen mode and the reconstructed one disagree. #196
// established this class for mouse modes; the screen itself has it too.
func TestReplayCannotCarryModeStateSetBeforeTheWindow(t *testing.T) {
	const enterAlt = "\x1b[?1049h"
	child := replayChild(32)
	child.Feed([]byte(enterAlt))
	child.Feed([]byte(strings.Repeat("y", 100)))

	replay := child.ReplayThrough(child.ReplaySafeEnd())
	if bytes.Contains(replay, []byte(enterAlt)) {
		t.Fatal("fixture did not age the mode set out of the ring")
	}
	t.Log("mode 4 reproduced: the alt-screen enter is outside the window, so a replay cannot restore the mode")
}

// Where each reproduced mode is ANSWERED. The four above are reproductions —
// they demonstrate the defect. These assert the new path handles it, so the
// reproductions cannot quietly become decoration (#209 BR-9).
//
// Mode 1 is deliberately not asserted here: no composition can restore a frame
// the ring no longer holds, which is the whole reason a switch asks the child
// to repaint. It is pinned where it happens —
// termcmd's TestTabSwitchIssuesARepaintRequestAndRestoresTheSize and couchtty's
// TestSwitchAsksTheIncomingChildToRepaint — and the assumption that the request
// produces a frame is measured against a real zellij by cmd/probes/zellijrepaint.
func TestTheAgedOutPaintIsNotRecoverableFromBytesAlone(t *testing.T) {
	// The premise of the repaint request, stated as a test so it cannot rot:
	// once the paint is out of the ring, no consumer of ReplayThrough can get
	// it back, however it composes what remains.
	const paint = "\x1b[H\x1b[2JFULL-FRAME-CONTENT"
	child := replayChild(64)
	child.Feed([]byte(paint))
	child.Feed([]byte(strings.Repeat("x", 200)))

	if bytes.Contains(child.ReplayThrough(child.ReplaySafeEnd()), []byte("FULL-FRAME-CONTENT")) {
		t.Fatal("the paint survived the ring; this premise no longer holds and the repaint request needs re-justifying")
	}
}

// Mode 4's answer: Screen tracks the mode across the WHOLE stream, so the
// buffer is known even when the sequence that set it has aged out.
func TestScreenStillKnowsTheBufferAfterTheModeSetAgesOut(t *testing.T) {
	const enterAlt = "\x1b[?1049h"
	child := replayChild(32)
	child.Feed([]byte(enterAlt))
	child.Feed([]byte(strings.Repeat("y", 100)))

	if bytes.Contains(child.ReplayThrough(child.ReplaySafeEnd()), []byte(enterAlt)) {
		t.Fatal("fixture did not age the mode set out")
	}
	alt, observed := child.RepaintModes()
	if !observed {
		t.Fatal("Screen did not observe the alt-screen enter, so a repaint could not assert it")
	}
	if !alt {
		t.Fatal("Screen lost the alt-screen state once the sequence aged out — mode 4 is not answered")
	}
}

// Mode 2's answer lives in hostty.Repaint (an empty replay emits nothing rather
// than blanking); this asserts the input half that makes it reachable.
func TestAnEmptyReplayIsReachableAndDistinguishableFromNoOutput(t *testing.T) {
	child := replayChild(16)
	child.Feed([]byte(strings.Repeat("a", 100)))
	if child.ReplayThrough(4) != nil {
		t.Fatal("a cutoff older than the ring no longer yields an empty replay; hostty.Repaint's empty case is unreachable")
	}
}

// The settle is the load-bearing half of the repaint request, and the boundary
// review was right to refuse it on a probe that measured a different sequence
// (#209 BR-3). Re-measured against the real binary with the PRODUCTION
// sequence, back-to-back ioctls repainted 6 of 12 runs — signals do not queue,
// so zellij can take one SIGWINCH, read a winsize already restored, and
// re-render nothing.
//
// A fake cannot tell us zellij repaints; that is cmd/probes/zellijrepaint's job.
// What it CAN pin is the property the probe measured the fix needs: the shrink
// is left standing, not erased in the same instant. Verified by mutation —
// dropping the sleep from nudge fails here.
func TestRequestRepaintLeavesTheShrinkStandingLongEnoughToBeSeen(t *testing.T) {
	child := NewFakeChild(nil)
	size := Size{Rows: 24, Cols: 80}
	if err := child.Resize(size); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	child.RequestRepaint()
	waitForResizes(t, child, 3)
	elapsed := time.Since(start)

	if elapsed < RepaintSettle {
		t.Errorf("the nudge completed after %v, want at least the measured settle %v — "+
			"a restore issued in the same instant as the shrink is coalesced away",
			elapsed, RepaintSettle)
	}
	got := child.Resizes()[1:]
	want := []Size{{Rows: size.Rows - 1, Cols: size.Cols}, size}
	if got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("resizes = %v, want %v — shrink by a row, restore, columns untouched", got, want)
	}
}

// The nudge asks for NO size, and that is the fix rather than a convenience
// (#209 C2). Passing one made a stale restore expressible: a host resize
// landing inside the settle window was overwritten by a restore leg carrying a
// size sampled before it, leaving the child permanently mis-sized. The first
// answer was a rule in a doc comment ("callers must stay on the goroutine that
// serializes their other resizes") and couch broke it in the same round.
//
// So this drives the race from a DIFFERENT goroutine, which is what a caller
// would do wrong, and asserts the child still ends at the newest size.
func TestAResizeDuringANudgeWinsWhicheverGoroutineItComesFrom(t *testing.T) {
	child := NewFakeChild(nil)
	before := Size{Rows: 24, Cols: 80}
	grown := Size{Rows: 40, Cols: 120}
	if err := child.Resize(before); err != nil {
		t.Fatal(err)
	}

	child.RequestRepaint()
	// Land the competing resize INSIDE the settle window, from a goroutine that
	// knows nothing about the nudge. WAIT for the shrink rather than sleeping
	// toward it: a sleep assumes the spawned nudge has already run, which is
	// the same "did not wait for the other party" mistake this test's own
	// comment is about, and it fails spuriously under -race or load.
	waitForResizes(t, child, 2)
	done := make(chan error, 1)
	go func() { done <- child.Resize(grown) }()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// WAIT FOR THE NUDGE TO FINISH before asking. Reading the size while the
	// restore leg is still pending passes either way — the first version of
	// this test did exactly that and was measured green against the defect it
	// names.
	//
	// The restore leg is SKIPPED here rather than reordered: the racing Resize
	// bumped the generation, so the nudge sees the world moved and declines to
	// write back a size nobody asked for (#209 I-1). Three resizes total, and
	// then a settle's grace to catch a restore that should not come.
	waitForResizes(t, child, 3)
	time.Sleep(2 * RepaintSettle)

	if got := child.Size(); got != grown {
		t.Fatalf("child left at %v after a resize raced a nudge, want the newest size %v — "+
			"the restore leg overwrote a size it did not read", got, grown)
	}
	// And the nudge still did its job rather than being skipped wholesale.
	got := child.Resizes()
	if len(got) < 3 || got[1].Rows != before.Rows-1 {
		t.Fatalf("resizes = %v, want the shrink still issued before the race", got)
	}
	if len(got) != 3 {
		t.Fatalf("resizes = %v, want exactly three — a superseded restore must not "+
			"be written at all, not written and then corrected", got)
	}
}

// A second request while one is in flight is DROPPED, not queued. Two nudges
// produce one repaint, and a held tab-switch key would otherwise spend the
// settle once per keystroke while the child's own reflow bytes queue behind it.
func TestASecondRepaintRequestDuringOneInFlightIsDropped(t *testing.T) {
	child := NewFakeChild(nil)
	size := Size{Rows: 24, Cols: 80}
	if err := child.Resize(size); err != nil {
		t.Fatal(err)
	}

	for range 5 {
		child.RequestRepaint()
	}
	waitForResizes(t, child, 3)
	time.Sleep(RepaintSettle)

	// 1 for the setup Resize plus exactly one shrink/restore pair.
	if got := child.Resizes(); len(got) != 3 {
		t.Fatalf("five requests produced %v, want one shrink-and-restore pair", got)
	}
}

// A child with no room to shrink is not nudged at all. One row for the child
// plus the reserved row is the floor; below it a "shrink" would be a resize to
// zero rows, which is a different event with different consequences.
func TestRequestRepaintDeclinesWhenThereIsNoRowToGive(t *testing.T) {
	for _, rows := range []uint16{0, 1} {
		child := NewFakeChild(nil)
		if err := child.Resize(Size{Rows: rows, Cols: 80}); err != nil {
			t.Fatal(err)
		}
		before := len(child.Resizes())
		child.RequestRepaint()
		time.Sleep(2 * RepaintSettle)
		if got := child.Resizes(); len(got) != before {
			t.Errorf("rows=%d issued %v, want no nudge at all", rows, got[before:])
		}
	}
}

func waitForResizes(t *testing.T, child *Child, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(child.Resizes()) >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d resizes; got %v", n, child.Resizes())
}
