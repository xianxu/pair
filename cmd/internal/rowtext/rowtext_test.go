package rowtext_test

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// The hazards this package exists for, each measured somewhere in the repo's
// history rather than imagined.
func TestSanitizeRemovesEveryControlPath(t *testing.T) {
	for _, tc := range []struct{ name, in, wantGone, wantKept string }{
		{"complete CSI clears the screen", "ev\x1b[2Jil", "\x1b", "evil"},
		{"SO garbles every following line", "Ev\x0eil", "\x0e", "Evil"},
		{"newline splits the row and can forge a line", "a\nb", "\n", "ab"},
		{"carriage return overwrites the row", "a\rb", "\r", "ab"},
		{"DEL", "a\x7fb", "\x7f", "ab"},
		{"BEL", "a\x07b", "\x07", "ab"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := rowtext.Sanitize(tc.in)
			if strings.Contains(got, tc.wantGone) {
				t.Fatalf("Sanitize(%q) = %q; %q survived", tc.in, got, tc.wantGone)
			}
			if !strings.Contains(got, tc.wantKept) {
				t.Fatalf("Sanitize(%q) = %q; stripping mangled the readable text", tc.in, got)
			}
		})
	}
}

// Width, not length: an emoji is one rune and two columns, and a row that
// overflows wraps onto the child's area.
func TestFitBoundsDisplayWidthNotRuneCount(t *testing.T) {
	if got := rowtext.Fit("😀😀😀", 4); got != "😀😀" {
		t.Fatalf("Fit = %q; want two double-width runes in four columns", got)
	}
	if got := rowtext.Fit("abc", 10); got != "abc" {
		t.Fatalf("Fit shortened a string that already fit: %q", got)
	}
	if got := rowtext.Fit("abc", 0); got != "" {
		t.Fatalf("Fit(_, 0) = %q; want nothing", got)
	}
}

// Order matters: sanitizing can only shorten, so fitting AFTERWARDS is what
// actually bounds the width. Reversed, a stripped escape frees columns the
// result never reclaims -- and worse, the caller's width bound is computed
// against bytes that will not be there.
func TestSanitizeAndFitBoundsTheSANITIZEDWidth(t *testing.T) {
	got := rowtext.SanitizeAndFit("\x1b[31mabcdef\x1b[0m", 4)
	if got != "abcd" {
		t.Fatalf("SanitizeAndFit = %q; want the first four VISIBLE columns", got)
	}
}
