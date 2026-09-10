package ptychild

import (
	"bytes"
	"strings"
	"testing"
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
