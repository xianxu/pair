package ptychild

import (
	"strings"
	"testing"
)

func feedWhole(data string) *Screen {
	s := &Screen{}
	s.Feed([]byte(data))
	return s
}

func feedByteAtATime(data string) *Screen {
	s := &Screen{}
	for i := 0; i < len(data); i++ {
		s.Feed([]byte(data[i : i+1]))
	}
	return s
}

// Ported from termcmd's TestUpdateMouseMode, which this scanner absorbs. The
// cases must survive the move or the migration silently loses mouse routing.
func TestScreenMouseMode(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"enable basic mouse", "\x1b[?1000h", true},
		// 1006 alone is an ENCODING request, not a request for events. This row
		// used to expect true, which is the conflation pair#172 BR-26 named: a
		// supervisor asking "is this child tracking" got yes for a child that
		// had asked for nothing. Its parsing is still asserted, by SGRMouse
		// below.
		{"sgr encoding alone is not tracking", "\x1b[?1006h", false},
		{"enable multiple modes", "\x1b[?1000;1006h", true},
		{"disable mouse", "\x1b[?1000h\x1b[?1000l", false},
		// And the inverse, which is how BR-22's demotion was reached through
		// the observation: dropping the ENCODING must not read as dropping the
		// tracking.
		{"dropping the encoding leaves tracking on", "\x1b[?1002h\x1b[?1006l", true},
		{"unrelated private mode preserves state", "\x1b[?1000h\x1b[?25l", true},
		{"colon-separated params", "\x1b[?1000:1006h", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := feedWhole(tt.data).Mouse(); got != tt.want {
				t.Fatalf("Mouse() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The encoding is tracked separately, so a supervisor can send a child the form
// it asked for: a child holding 1000 without 1006 cannot parse an SGR report.
func TestScreenSGRMouseIsSeparateFromTracking(t *testing.T) {
	for _, tt := range []struct {
		name     string
		data     string
		tracking bool
		sgr      bool
	}{
		{"neither", "", false, false},
		{"tracking only", "\x1b[?1000h", true, false},
		{"encoding only", "\x1b[?1006h", false, true},
		{"both", "\x1b[?1002;1006h", true, true},
		{"drop the encoding, keep tracking", "\x1b[?1002;1006h\x1b[?1006l", true, false},
		{"drop the tracking, keep the encoding", "\x1b[?1002;1006h\x1b[?1002l", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			screen := feedWhole(tt.data)
			if got := screen.Mouse(); got != tt.tracking {
				t.Errorf("Mouse() = %v, want %v", got, tt.tracking)
			}
			if got := screen.SGRMouse(); got != tt.sgr {
				t.Errorf("SGRMouse() = %v, want %v", got, tt.sgr)
			}
		})
	}
}

func TestScreenAltScreen(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"1049 enter", "\x1b[?1049h", true},
		{"1047 enter", "\x1b[?1047h", true},
		{"47 enter", "\x1b[?47h", true},
		{"1049 leave", "\x1b[?1049h\x1b[?1049l", false},
		{"enter then unrelated mode stays", "\x1b[?1049h\x1b[?25l", true},
		{"never entered", "plain output\r\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := feedWhole(tt.data).AltScreen(); got != tt.want {
				t.Fatalf("AltScreen() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The console pins the host's scrolling region above its reserved row. Anything
// a child does that can drop that region has to be observable, or the row is
// silently overwritten and never comes back.
func TestScreenRowDirty(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"DECSTBM reset", "\x1b[r", true},
		{"DECSTBM explicit", "\x1b[1;24r", true},
		{"RIS full reset", "\x1bc", true},
		{"alt screen enter", "\x1b[?1049h", true},
		{"alt screen leave", "\x1b[?1049h\x1b[?1049l", true},
		// An 'r' final behind the private introducer is DECRSTR, not DECSTBM.
		// Treating every 'r' as a margin change would fire on ordinary output.
		{"cursor position", "\x1b[3;4H", false},
		// DECRSTR: an 'r' final BEHIND the private introducer is a mode
		// restore, not a margin change. The rule was stated in two comments
		// and covered by no case -- removing the introducer branch left the
		// suite green (BR-6).
		{"DECRSTR private r", "\x1b[?1049r", false},
		{"SGR", "\x1b[31m", false},
		{"plain text", "hello\r\nworld", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := feedWhole(tt.data).TakeRowDirty(); got != tt.want {
				t.Fatalf("TakeRowDirty() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Latched, and cleared by the reader: the console re-asserts once per event,
// not once per poll.
func TestScreenRowDirtyIsClearedOnRead(t *testing.T) {
	s := feedWhole("\x1b[r")
	if !s.TakeRowDirty() {
		t.Fatal("first TakeRowDirty() = false")
	}
	if s.TakeRowDirty() {
		t.Fatal("second TakeRowDirty() = true — the event was not consumed")
	}
}

// BEL is the one activity signal available before #147, so a false positive
// makes the status row cry wolf. Every title change ends in BEL.
func TestScreenBellIgnoresOSCTerminators(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"bare BEL", "done\x07", true},
		{"OSC title BEL-terminated", "\x1b]0;my title\x07", false},
		{"OSC title then real BEL", "\x1b]0;my title\x07\x07", true},
		{"OSC ST-terminated", "\x1b]0;my title\x1b\\", false},
		{"no bell", "plain", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := feedWhole(tt.data).TakeBell(); got != tt.want {
				t.Fatalf("TakeBell() = %v, want %v", got, tt.want)
			}
		})
	}
}

// The regression the M1 boundary review found (BR-1). An OSC 52 clipboard write
// is kilobytes and ALWAYS crosses a 4096-byte pty read boundary; with a tight
// pending bound it was abandoned mid-sequence and its terminating BEL was then
// counted as the child ringing -- a false page on every copy.
func TestScreenLongSequenceSplitAcrossReadsRaisesNoBell(t *testing.T) {
	for _, n := range []int{300, 4096, 9000} {
		data := "\x1b]52;c;" + strings.Repeat("A", n) + "\x07"
		if feedWhole(data).TakeBell() {
			t.Fatalf("%d-byte OSC fed whole raised a bell", n)
		}
		if feedByteAtATime(data).TakeBell() {
			t.Fatalf("%d-byte OSC fed one byte at a time raised a bell", n)
		}
	}
}

// A pty read boundary falls wherever the kernel puts it. termcmd's
// updateMouseMode scanned each chunk independently and could not see a sequence
// split across two reads; this must.
func TestScreenSplitReadsReachTheSameState(t *testing.T) {
	streams := []string{
		"\x1b[?1049h\x1b[?1006h\x1b[1;24r",
		"\x1b]0;title\x07\x07",
		"text\x1b[?47h more\x1b[?47l",
	}
	for _, data := range streams {
		whole, split := feedWhole(data), feedByteAtATime(data)
		if whole.AltScreen() != split.AltScreen() || whole.Mouse() != split.Mouse() {
			t.Fatalf("%q: split state {alt:%v mouse:%v} != whole {alt:%v mouse:%v}",
				data, split.AltScreen(), split.Mouse(), whole.AltScreen(), whole.Mouse())
		}
		if whole.TakeRowDirty() != split.TakeRowDirty() || whole.TakeBell() != split.TakeBell() {
			t.Fatalf("%q: split latches differ from whole", data)
		}
	}
}

// The repo's prefix-vs-complete rule: buffer only a real prefix. A malformed
// but COMPLETE control must be consumed, or it swallows the input behind it.
func TestScreenConsumesMalformedCompleteControls(t *testing.T) {
	s := &Screen{}
	s.Feed([]byte("\x1b[@z\x1b[?1049h"))
	if !s.AltScreen() {
		t.Fatal("a malformed complete control swallowed the sequence behind it")
	}
	if n := s.Pending(); n != 0 {
		t.Fatalf("Pending() = %d after a complete stream, want 0", n)
	}
}

// A stream of param bytes with no final byte is not a "prefix" worth holding
// forever -- that is an unbounded buffer fed by arbitrary child output. Past
// the guard the scanner stops BUFFERING but keeps FRAMING, so memory is bounded
// and the stream stays in sync.
//
// The assertion is EQUIVALENCE, not recovery. An unterminated CSI swallows what
// follows in whole-feed too -- `\x1b[` is a final byte to `ansi`'s
// introducer-independent scan -- so demanding recovery here would demand
// behaviour the framing does not have. What must hold is that chunking cannot
// change the answer.
func TestScreenPendingIsBoundedAndChunkingCannotChangeTheAnswer(t *testing.T) {
	junk := make([]byte, 0, maxPending+4096)
	junk = append(junk, 0x1b, '[')
	for i := 0; i < maxPending+2000; i++ {
		junk = append(junk, ';')
	}
	data := string(junk) + "\x1b[?1049h"

	whole := &Screen{}
	whole.Feed([]byte(data))

	split := &Screen{}
	for i := 0; i < len(data); i += 4096 {
		end := i + 4096
		if end > len(data) {
			end = len(data)
		}
		split.Feed([]byte(data[i:end]))
	}

	if whole.Pending() > maxPending || split.Pending() > maxPending {
		t.Fatalf("pending exceeded the bound: whole=%d split=%d", whole.Pending(), split.Pending())
	}
	if whole.AltScreen() != split.AltScreen() {
		t.Fatalf("chunking changed AltScreen: whole=%v split=%v", whole.AltScreen(), split.AltScreen())
	}
	if whole.TakeBell() != split.TakeBell() {
		t.Fatal("chunking changed the bell latch on an over-long unterminated run")
	}
}

// The other half of BR-1's final shape: a TERMINATED sequence longer than the
// bound must still be framed to its real terminator, so a bell AFTER it is
// still counted. Discarding to the next ESC (the round-2 fix) dropped it.
func TestScreenLongTerminatedSequenceStillFramesAndKeepsALaterBell(t *testing.T) {
	data := "\x1b]52;c;" + strings.Repeat("A", maxPending+5000) + "\x07" + "ready\x07"

	whole := &Screen{}
	whole.Feed([]byte(data))
	split := &Screen{}
	for i := 0; i < len(data); i += 4096 {
		end := i + 4096
		if end > len(data) {
			end = len(data)
		}
		split.Feed([]byte(data[i:end]))
	}

	wb, sb := whole.TakeBell(), split.TakeBell()
	if wb != sb {
		t.Fatalf("chunking changed the bell: whole=%v split=%v", wb, sb)
	}
	if !wb {
		t.Fatal("the real bell AFTER an over-long terminated sequence was dropped")
	}
}

// The bound has to be generous enough for the protocol: OSC 52 carries a whole
// clipboard. A tight bound is not a safety measure, it is a false-positive
// generator (BR-1).
func TestScreenPendingBoundFitsARealisticOSC(t *testing.T) {
	if maxPending < 32*1024 {
		t.Fatalf("maxPending = %d is too small for an OSC 52 clipboard payload", maxPending)
	}
}

func FuzzScreenFeed(f *testing.F) {
	for _, s := range []string{
		"", "\x1b", "\x1b[", "\x1b[?1049h", "\x1b]0;t\x07", "\x1b[@z",
		"\x1b[?1000;1006h\x1b[r", "\x1bc", "\x07", "\x1b]4;?\x1b\\",
		"\x1b]52;c;" + strings.Repeat("A", 300) + "\x07",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		whole := &Screen{}
		whole.Feed(in) // must not panic

		split := &Screen{}
		for i := 0; i < len(in); i++ {
			split.Feed(in[i : i+1])
		}
		if whole.AltScreen() != split.AltScreen() || whole.Mouse() != split.Mouse() {
			t.Fatalf("chunking changed the state for %q", in)
		}
		// The LATCHES are covered too. Asserting invariance for AltScreen and
		// Mouse but not for these is precisely why this fuzzer ran 595k execs
		// without finding the false bell (BR-1): the bug lived in the two
		// fields it did not compare.
		if whole.TakeBell() != split.TakeBell() {
			t.Fatalf("chunking changed the bell latch for %q", in)
		}
		if whole.TakeRowDirty() != split.TakeRowDirty() {
			t.Fatalf("chunking changed the row-dirty latch for %q", in)
		}
		if whole.Pending() > maxPending || split.Pending() > maxPending {
			t.Fatalf("pending exceeded the bound for %q", in)
		}
	})
}

// BR-1 round 2: raising the bound cured the everyday OSC 52 case but left
// chunk-invariance broken ABOVE the bound -- whole input discarded the whole
// run, split input discarded the first maxPending bytes and then rescanned the
// remainder as text, where a BEL still counted. The rule that fixes it is
// resync-to-next-ESC, which both paths follow identically.
func TestScreenChunkInvariantAboveThePendingBound(t *testing.T) {
	// An unterminated sequence longer than the guard, then a BEL that belongs
	// to it, then a real sequence the scanner must still pick up afterwards.
	data := "\x1b]52;c;" + strings.Repeat("A", maxPending+5000) + "\x07" + "\x1b[?1049h"

	whole := &Screen{}
	whole.Feed([]byte(data))

	split := &Screen{}
	for i := 0; i < len(data); i += 4096 {
		end := i + 4096
		if end > len(data) {
			end = len(data)
		}
		split.Feed([]byte(data[i:end]))
	}

	if whole.TakeBell() != split.TakeBell() {
		t.Fatal("bell latch differs between whole and 4096-byte-chunked input above the bound")
	}
	if whole.AltScreen() != split.AltScreen() {
		t.Fatal("alt-screen state differs between whole and chunked input above the bound")
	}
	// And the trailing real sequence must survive the resync.
	if !whole.AltScreen() {
		t.Fatal("resync swallowed the sequence that followed the abandoned run")
	}
}

// BR-1's last residual: the OSC scan stops before a trailing ESC so a two-byte
// ST is not split in half -- but the bytes it declined to consume have to be
// HELD, not dropped. Measured by the reviewer at 1 of 70,550 cut positions.
func TestScreenHoldsAnSTSplitAcrossReads(t *testing.T) {
	body := strings.Repeat("A", maxPending+2000)
	// ... long OSC ... ESC | \ ... then a REAL bell that must survive.
	first := "\x1b]52;c;" + body + "\x1b"
	second := "\\" + "ready\x07"

	split := &Screen{}
	split.Feed([]byte(first))
	split.Feed([]byte(second))

	whole := &Screen{}
	whole.Feed([]byte(first + second))

	wb, sb := whole.TakeBell(), split.TakeBell()
	if wb != sb {
		t.Fatalf("a chunk boundary inside the ST changed the bell latch: whole=%v split=%v", wb, sb)
	}
	if !wb {
		t.Fatal("the real bell after the sequence was swallowed by both paths")
	}
}

// The framing must cover every sequence class the "BEL only outside a sequence"
// invariant is claimed over. DCS/APC/PM/SOS fell through to the two-byte case
// and had their payloads scanned as text.
func TestScreenFramesAllStringTerminatedClasses(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"DCS XTGETTCAP", "\x1bP+q616263\x07\x1b\\"},
		{"APC kitty graphics", "\x1b_Ga=T,f=100;PAYLOAD\x07\x1b\\"},
		{"PM", "\x1b^private\x07\x1b\\"},
		{"SOS", "\x1bXstring\x07\x1b\\"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if feedWhole(c.data).TakeBell() {
				t.Fatalf("a BEL inside %s rang the bell; it is a terminator, not the child", c.name)
			}
		})
	}
	// A tmux passthrough carries a whole sequence as DCS payload; it must not
	// take effect from inside the wrapper.
	s := feedWhole("\x1bPtmux;\x1b[?1049h\x1b\\")
	if s.AltScreen() {
		t.Fatal("a DCS payload set alt-screen from inside a sequence")
	}
}

// Erasing the display takes a reserved row with it even though the scrolling
// region survives -- DECSTBM restricts SCROLLING, not erasing. Missing this
// cost an M2 operator smoke: the row appeared and then vanished a second later
// as pair drew its first full screen.
func TestScreenTreatsAnEraseAsRowDirty(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
	}{
		{"ED 2", "\x1b[2J"},
		{"ED default", "\x1b[J"},
		{"ED 1", "\x1b[1J"},
		{"ED 3", "\x1b[3J"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !feedWhole(tt.data).TakeRowDirty() {
				t.Fatalf("%s did not mark the row dirty; a full-screen child's startup clear would silently eat the row", tt.name)
			}
		})
	}
	// Erase in LINE is not erase in display: it cannot reach a row the child
	// cannot address.
	if feedWhole("\x1b[2K").TakeRowDirty() {
		t.Fatal("erase-in-line marked the row dirty; it repaints on every line a child clears")
	}
}

// A mouse-mode change latches the same edge an alt-screen transition does.
//
// Without it nothing TRIGGERS a supervisor to re-evaluate: mouse reporting is
// terminal-global, so a child's bare `?1000l` left couch's own clicks off until
// some unrelated paint happened to run. The row-dirty latch is how this package
// already says "the console needs to look again".
func TestMouseModeChangeLatchesRowDirty(t *testing.T) {
	for _, data := range []string{
		"\x1b[?1000h", "\x1b[?1002h", "\x1b[?1003h", "\x1b[?1006h",
		"\x1b[?1000l", "\x1b[?1002l",
	} {
		screen := feedWhole(data)
		if !screen.TakeRowDirty() {
			t.Errorf("%q did not latch rowDirty, so nothing re-evaluates the mode", data)
		}
	}
	// And an unrelated private mode still does not.
	if feedWhole("\x1b[?25l").TakeRowDirty() {
		t.Error("hiding the cursor latched rowDirty")
	}
}

// The cursor save slot is SHARED: one per terminal. A console with a reserved
// row that paints between the child's DECSC and DECRC clobbers what the child
// saved, and the child's restore then recovers the CONSOLE's position.
//
// Measured in pair#199: zsh draws its right-hand prompt with terminfo sc/rc
// (ESC 7 / ESC 8), so the operator's cursor ended up inside the tab strip and
// the right-prompt was drawn on the strip's row.
func TestHoldsCursorSaveTracksTheChildsOwnSaveRestore(t *testing.T) {
	var s Screen
	if s.HoldsCursorSave() {
		t.Fatal("a fresh screen reports a held save")
	}
	s.FeedFraming([]byte("text\x1b7more"))
	if !s.HoldsCursorSave() {
		t.Fatal("the child's DECSC was not observed")
	}
	s.FeedFraming([]byte("still held\x1b8done"))
	if s.HoldsCursorSave() {
		t.Fatal("the child's DECRC did not clear the debt")
	}
}

// One slot, not a stack: terminals keep a single save, so a second DECSC
// overwrites the first and ONE DECRC settles it. Counting depth instead would
// leave a console deferring forever after any unbalanced pair.
func TestASecondSaveDoesNotDeepenTheDebt(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b7\x1b7"))
	// The state AFTER the second save is what distinguishes "set" from
	// "toggle". Checking only the end state passes for both, which is how a
	// toggling implementation survived this test's first version.
	if !s.HoldsCursorSave() {
		t.Fatal("a second DECSC cleared the debt; it must SET, not toggle")
	}
	s.FeedFraming([]byte("\x1b8"))
	if s.HoldsCursorSave() {
		t.Fatal("two saves needed two restores; the slot is one deep, not a stack")
	}
}

// THE SAVE-SLOT ENUMERATION, one case per spelling.
//
// A missing arm here is a silent hole in the paint gate, not a visible failure,
// which is why this is a table rather than a handful of examples. The previous
// version of this test asserted that ENTERING the alt screen clears the save --
// encoding the bug it was meant to catch. `?1049h` is defined as
// DECSC-then-switch: it TAKES the slot, and it is how nearly every full-screen
// child takes it.
func TestEverySaveSlotSpellingIsAccountedFor(t *testing.T) {
	const (
		takes    = "takes the slot"
		releases = "releases it"
		inert    = "leaves it alone"
	)
	for _, tc := range []struct{ name, seq, effect string }{
		{"DECSC", "\x1b7", takes},
		{"DECRC", "\x1b8", releases},
		{"SCOSC", "\x1b[s", takes},
		{"SCORC", "\x1b[u", releases},
		{"save cursor (1048h)", "\x1b[?1048h", takes},
		{"restore cursor (1048l)", "\x1b[?1048l", releases},
		{"alt screen WITH save (1049h)", "\x1b[?1049h", takes},
		{"leave alt screen (1049l)", "\x1b[?1049l", releases},
		{"alt screen only (1047h)", "\x1b[?1047h", inert},
		{"alt screen only (1047l)", "\x1b[?1047l", inert},
		{"alt screen only (47h)", "\x1b[?47h", inert},
		{"RIS", "\x1bc", releases},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// From EMPTY: does it take?
			var empty Screen
			empty.FeedFraming([]byte(tc.seq))
			if got := empty.HoldsCursorSave(); got != (tc.effect == takes) {
				t.Fatalf("from empty, %s left HoldsCursorSave=%v; it %s", tc.name, got, tc.effect)
			}
			// From HELD: does it release, or leave it alone?
			var held Screen
			held.FeedFraming([]byte("\x1b7"))
			held.FeedFraming([]byte(tc.seq))
			want := tc.effect != releases // takes and inert both leave it held
			if got := held.HoldsCursorSave(); got != want {
				t.Fatalf("from held, %s left HoldsCursorSave=%v; it %s", tc.name, got, tc.effect)
			}
		})
	}
}

// 1047/47 switch screens WITHOUT the save half, so clearing the flag there would
// hand a console permission to paint inside a save the child still holds. This
// is the arm most likely to be written as "an alt-screen transition abandons the
// save", which is true of 1049 and false of these.
func TestAltScreenWithoutTheSaveHalfDoesNotReleaseTheSlot(t *testing.T) {
	for _, seq := range []string{"\x1b[?1047h", "\x1b[?1047l", "\x1b[?47h", "\x1b[?47l"} {
		var s Screen
		s.FeedFraming([]byte("\x1b7"))
		s.FeedFraming([]byte(seq))
		if !s.HoldsCursorSave() {
			t.Fatalf("%q released a save it does not touch", seq)
		}
	}
}

// The two gates are INDEPENDENT. A console must defer on either, and an earlier
// design that folded them into one flag could not tell "mid-escape" from
// "inside the child's save" -- they clear on different bytes.
func TestMidSequenceAndHeldSaveAreSeparateConditions(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b7"))
	if s.MidSequence() {
		t.Fatal("a COMPLETE DECSC left the framer mid-sequence")
	}
	if !s.HoldsCursorSave() {
		t.Fatal("the save was not recorded")
	}
	s.FeedFraming([]byte("\x1b[3"))
	if !s.MidSequence() || !s.HoldsCursorSave() {
		t.Fatal("a partial CSI must not disturb the save debt, or vice versa")
	}
}

// SCOSC/SCORC is the CSI spelling of the SAME slot, and this repo's own probe
// (probes/cursorsaveslots) established zellij gives it no second storage. A
// gate watching only ESC 7 lets the collision straight back in for any child
// that uses the CSI form.
func TestTheCSISpellingOfSaveRestoreHoldsTheSameSlot(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b[s"))
	if !s.HoldsCursorSave() {
		t.Fatal("CSI s did not register as a save; a child using it would be unguarded")
	}
	s.FeedFraming([]byte("\x1b[u"))
	if s.HoldsCursorSave() {
		t.Fatal("CSI u did not clear the save")
	}
}

// The two spellings share ONE slot, so they must settle each other -- a child
// is free to save with one and restore with the other, and terminals honour it.
func TestTheTwoSpellingsSettleEachOther(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b7"))
	s.FeedFraming([]byte("\x1b[u"))
	if s.HoldsCursorSave() {
		t.Fatal("CSI u did not clear a save made with ESC 7; they are one slot")
	}
	s.FeedFraming([]byte("\x1b[s"))
	s.FeedFraming([]byte("\x1b8"))
	if s.HoldsCursorSave() {
		t.Fatal("ESC 8 did not clear a save made with CSI s")
	}
}

// `CSI <n> s` is DECSLRM (set left/right margins), NOT a save. Treating it as
// one would defer every paint forever against a child that sets margins and
// never issues a restore -- a permanently stale row from a sequence that had
// nothing to do with the cursor.
func TestParameterisedCSIsIsMarginsNotASave(t *testing.T) {
	for _, seq := range []string{"\x1b[1;80s", "\x1b[5s", "\x1b[?69h\x1b[1;40s"} {
		var s Screen
		s.FeedFraming([]byte(seq))
		if s.HoldsCursorSave() {
			t.Fatalf("%q was treated as a cursor save; it is DECSLRM", seq)
		}
	}
}

// BR-79: gating on a save the child holds for its whole ALT-SCREEN lifetime
// freezes the console's row for minutes. That is the common case (every nvim
// session), and it trades a rare one-frame cursor glitch for a permanently
// stale row -- strictly worse.
//
// The hazard the gate exists for is a child that saves and restores SOON: a
// line-oriented child drawing a prompt. A full-screen child repaints from its
// own model every frame and repositions absolutely, so a disturbed cursor does
// not survive to be seen.
func TestTheSaveGateDoesNotFreezeThroughAnAltScreenSession(t *testing.T) {
	var s Screen
	if !s.SafeToPaint() {
		t.Fatal("a fresh screen refuses paints")
	}

	// nvim starting: 1049h takes the slot AND enters the alt screen.
	s.FeedFraming([]byte("\x1b[?1049h"))
	if !s.HoldsCursorSave() {
		t.Fatal("1049h did not take the slot; it is DECSC-then-switch")
	}
	if !s.SafeToPaint() {
		t.Fatal("the row is frozen for the child's whole alt-screen session")
	}

	// Still paintable after the child writes for a while.
	s.FeedFraming([]byte("lots of nvim output\x1b[1;1H"))
	if !s.SafeToPaint() {
		t.Fatal("the row froze partway through the session")
	}

	// nvim quitting restores and leaves the alt screen.
	s.FeedFraming([]byte("\x1b[?1049l"))
	if s.HoldsCursorSave() || !s.SafeToPaint() {
		t.Fatal("1049l did not release the slot")
	}
}

// And the case the gate DOES exist for is unchanged: a line-oriented child
// holding a save outside the alt screen -- zsh drawing its right-hand prompt.
func TestTheSaveGateStillClosesOutsideTheAltScreen(t *testing.T) {
	for _, seq := range []string{"\x1b7", "\x1b[s", "\x1b[?1048h"} {
		var s Screen
		s.FeedFraming([]byte(seq))
		if s.SafeToPaint() {
			t.Fatalf("%q left the gate open; a paint here corrupts the child's restore", seq)
		}
	}
}

// BR-81: altScreen is a SAFETY input to SafeToPaint, so every path that leaves
// the alt screen must clear it. RIS resets the terminal to its power-on state,
// which is the primary screen; leaving the flag set turned a missing reset into
// a permanently disabled save gate.
func TestRISLeavesTheAltScreenAndReArmsTheSaveGate(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b[?1049h")) // in the alt screen, holding the save
	s.FeedFraming([]byte("\x1bc"))       // RIS

	if s.AltScreen() {
		t.Fatal("RIS left altScreen set; the terminal's power-on state is the primary screen")
	}
	// And the gate must arm again: a save taken after the reset closes it.
	s.FeedFraming([]byte("\x1b7"))
	if s.SafeToPaint() {
		t.Fatal("after RIS the save gate never closes again; altScreen is stuck on")
	}
}

// Every field SafeToPaint consults must be cleared by every reset that clears
// the terminal state it models. Stated as its own test because promoting a
// field to a safety input is what created the hole: the audit belongs with the
// promotion.
func TestEverySafetyInputIsClearedByRIS(t *testing.T) {
	var s Screen
	s.FeedFraming([]byte("\x1b[?1049h\x1b[3")) // alt screen, save held, mid-sequence
	s.FeedFraming([]byte("m"))                 // close the sequence
	s.FeedFraming([]byte("\x1bc"))             // RIS
	if !s.SafeToPaint() {
		t.Fatal("RIS left a safety input set; a reset must return the gate to open")
	}
	if s.HoldsCursorSave() || s.AltScreen() || s.MidSequence() {
		t.Fatalf("after RIS: save=%v alt=%v mid=%v; all must be clear",
			s.HoldsCursorSave(), s.AltScreen(), s.MidSequence())
	}
}
