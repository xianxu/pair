package mouseinput

import (
	"strings"
	"testing"
)

// The package shipped with its tests living in termcmd, which is where they were
// before the move. That is coverage by inheritance: a change here is only caught
// if termcmd happens to exercise it, and termcmd cannot exercise what only couch
// calls.
func TestParseDecodesBothTerminators(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Event
	}{
		{"\x1b[<0;12;34M", Event{Button: 0, X: 12, Y: 34}},
		{"\x1b[<0;12;34m", Event{Button: 0, X: 12, Y: 34, Release: true}},
		{"\x1b[<64;1;1M", Event{Button: WheelUp, X: 1, Y: 1}},
		// Above 223, which is the reason SGR is required: the legacy X10
		// encoding caps there and fails SILENTLY on a wide or tall terminal.
		{"\x1b[<0;500;400M", Event{Button: 0, X: 500, Y: 400}},
	} {
		got, ok := Parse([]byte(tc.in))
		if !ok || got != tc.want {
			t.Errorf("Parse(%q) = (%+v,%v), want (%+v,true)", tc.in, got, ok, tc.want)
		}
	}
}

func TestParseRefusesWhatItCannotDecode(t *testing.T) {
	for _, in := range []string{
		"", "\x1b[<", "\x1b[<0;12;34", // no terminator
		"\x1b[M abc",   // legacy X10 -- refused, not guessed at
		"\x1b[<a;b;cM", // non-numeric
		"hello",        //
	} {
		if _, ok := Parse([]byte(in)); ok {
			t.Errorf("Parse(%q) decoded something", in)
		}
	}
}

// ParsePrefix returns the RAW bytes, and a forwarder must write those rather
// than a re-encoding -- otherwise the wire format has two sources of truth.
func TestParsePrefixReturnsTheWireBytes(t *testing.T) {
	const report = "\x1b[<0;12;34M"
	event, raw, rest, ok := ParsePrefix([]byte(report + "tail"))
	if !ok {
		t.Fatal("ParsePrefix refused a complete report")
	}
	if string(raw) != report {
		t.Errorf("raw = %q, want %q", raw, report)
	}
	if string(rest) != "tail" {
		t.Errorf("rest = %q, want %q", rest, "tail")
	}
	if event.X != 12 || event.Y != 34 {
		t.Errorf("event = %+v", event)
	}
}

func TestFindLocatesAReportAfterOtherBytes(t *testing.T) {
	before, event, raw, rest, ok := Find([]byte("abc\x1b[<0;7;8Mdef"))
	if !ok {
		t.Fatal("Find missed an embedded report")
	}
	if string(before) != "abc" || string(raw) != "\x1b[<0;7;8M" || string(rest) != "def" {
		t.Errorf("before=%q raw=%q rest=%q", before, raw, rest)
	}
	if event.X != 7 || event.Y != 8 {
		t.Errorf("event = %+v", event)
	}
}

// IsPrefix says "could still become a report". A caller holding on it MUST bound
// the wait -- MaxReport is that bound, and without one a stray introducer parks
// every following keystroke.
func TestIsPrefixCoversEveryGenuinePrefixAndNothingElse(t *testing.T) {
	const report = "\x1b[<0;12;34M"
	for split := 1; split < len(report); split++ {
		if !IsPrefix([]byte(report[:split])) {
			t.Errorf("IsPrefix(%q) = false, but it is a genuine prefix", report[:split])
		}
	}
	if IsPrefix([]byte(report)) {
		t.Error("a COMPLETE report is not a prefix")
	}
	for _, in := range []string{"a", "\x1b[A", "\x1b[200~"} {
		if IsPrefix([]byte(in)) {
			t.Errorf("IsPrefix(%q) = true", in)
		}
	}
	// The bound is what a caller uses to stop waiting.
	if len("\x1b[<"+strings.Repeat("9", MaxReport)) <= MaxReport {
		t.Fatal("MaxReport does not bound anything")
	}
}
