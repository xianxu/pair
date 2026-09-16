package termcmd

import (
	"testing"
)

func TestCsiEndLenientFramingIsPinned(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"plain CSI", "\x1b[31m", 5},
		{"unterminated", "\x1b[31", -1},
		{"out-of-range param byte still frames", "\x1b[\x00A", 4},
		{"private-mode query", "\x1b[?1006h", 8},
		{"SS3", "\x1bOX", 3},
		{"SS3 with @ final", "\x1bO@", 3},
	}
	for _, c := range cases {
		if got := csiEnd([]byte(c.in)); got != c.want {
			t.Errorf("%s: csiEnd(%q) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

func TestMalformedEscapeSizeNeverReturnsZeroOnNonEmptyInput(t *testing.T) {
	for _, in := range []string{"\x1b[", "\x1b[\x00A", "\x1bZ", "\x1b", "\x1b[31m", "\x1bOX"} {
		if got := malformedEscapeSize([]byte(in)); got <= 0 {
			t.Errorf("malformedEscapeSize(%q) = %d — the decoder loop would not advance", in, got)
		}
	}
}
