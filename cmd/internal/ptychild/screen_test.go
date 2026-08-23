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
		{"enable sgr mouse", "\x1b[?1006h", true},
		{"enable multiple modes", "\x1b[?1000;1006h", true},
		{"disable mouse", "\x1b[?1000h\x1b[?1000l", false},
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
func TestScreenRegionLost(t *testing.T) {
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
			if got := feedWhole(tt.data).TakeRegionLost(); got != tt.want {
				t.Fatalf("TakeRegionLost() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Latched, and cleared by the reader: the console re-asserts once per event,
// not once per poll.
func TestScreenRegionLostIsClearedOnRead(t *testing.T) {
	s := feedWhole("\x1b[r")
	if !s.TakeRegionLost() {
		t.Fatal("first TakeRegionLost() = false")
	}
	if s.TakeRegionLost() {
		t.Fatal("second TakeRegionLost() = true — the event was not consumed")
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
		if whole.TakeRegionLost() != split.TakeRegionLost() || whole.TakeBell() != split.TakeBell() {
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
// the guard the run is DISCARDED rather than rescanned as text, which is what
// keeps the bell latch chunk-invariant (BR-1).
func TestScreenPendingIsBounded(t *testing.T) {
	s := &Screen{}
	junk := make([]byte, 0, maxPending+4096)
	junk = append(junk, 0x1b, '[')
	for i := 0; i < maxPending+2000; i++ {
		junk = append(junk, ';')
	}
	s.Feed(junk)
	if n := s.Pending(); n > maxPending {
		t.Fatalf("Pending() = %d, want <= %d", n, maxPending)
	}
	// A BEL inside an abandoned run is that sequence's terminator, not the
	// child ringing, so discarding must not latch one.
	if s.TakeBell() {
		t.Fatal("discarding an over-long run raised a bell")
	}
	// And it must still recognise a real sequence afterwards.
	s.Feed([]byte("\x1b[?1049h"))
	if !s.AltScreen() {
		t.Fatal("scanner did not recover after dropping an over-long pending run")
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
		if whole.TakeRegionLost() != split.TakeRegionLost() {
			t.Fatalf("chunking changed the region-lost latch for %q", in)
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
