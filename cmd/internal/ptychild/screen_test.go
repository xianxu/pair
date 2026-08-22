package ptychild

import "testing"

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
// forever — that is an unbounded buffer fed by arbitrary child output.
func TestScreenPendingIsBounded(t *testing.T) {
	s := &Screen{}
	junk := make([]byte, 0, 8192)
	junk = append(junk, 0x1b, '[')
	for i := 0; i < 8000; i++ {
		junk = append(junk, ';')
	}
	s.Feed(junk)
	if n := s.Pending(); n > maxPending {
		t.Fatalf("Pending() = %d, want <= %d", n, maxPending)
	}
	// And it must still recognise a real sequence afterwards.
	s.Feed([]byte("\x1b[?1049h"))
	if !s.AltScreen() {
		t.Fatal("scanner did not recover after dropping an over-long pending run")
	}
}

func FuzzScreenFeed(f *testing.F) {
	for _, s := range []string{
		"", "\x1b", "\x1b[", "\x1b[?1049h", "\x1b]0;t\x07", "\x1b[@z",
		"\x1b[?1000;1006h\x1b[r", "\x1bc", "\x07", "\x1b]4;?\x1b\\",
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
		if whole.Pending() > maxPending || split.Pending() > maxPending {
			t.Fatalf("pending exceeded the bound for %q", in)
		}
	})
}
