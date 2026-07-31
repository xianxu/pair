package ansi

import (
	"bytes"
	"regexp"
	"testing"
)

// otherEscRe is the regex retired from wrapcmd/wrap.go:189, preserved HERE as the
// differential oracle.
//
// #128's Done-when says wrapcmd's three call sites must behave identically after
// the extraction. The only honest way to hold that is to compare against what the
// code actually used to run — not to re-read the regex and the scanner and reason
// that they agree. Every disagreement below is the scanner's bug, never the
// oracle's.
var otherEscRe = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()*+][@-~]|\x1b[@-Z\\-_]`)

// regexLenAt is the oracle's answer to SequenceLen: the length of a match anchored
// at offset 0, else 0. Mirrors wrap.go:1018's `FindIndex(...) && loc[0] == 0`.
func regexLenAt(buf []byte) int {
	loc := otherEscRe.FindIndex(buf)
	if loc != nil && loc[0] == 0 {
		return loc[1]
	}
	return 0
}

var oracleCorpus = []string{
	"\x1b[31mX", "\x1b]0;t\x07", "\x1b(B", "\x1bM", "plain",
	"\x1b[", "\x1b]0;unterminated", "\x1b]0;a\x1bZ\x07", "\x1b[>1u", "\x1b[>7u",
	"\x1b[?1006h", "\x1b[\x00A", "\x1bOX", "\x1b]0;t\x1b\\", "\x1b",
	"\x1b[1;5D", "\x1b[999~", "\x1b[<0;12;4X", "\x1b[@", "\x1bO@",
	"\x1b[31;42;1m hello \x1b[0m", "\x1b]8;;http://x\x07link\x1b]8;;\x07",
}

func FuzzSequenceLenMatchesRegex(f *testing.F) {
	for _, s := range oracleCorpus {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		if got, want := SequenceLen(buf), regexLenAt(buf); got != want {
			t.Errorf("SequenceLen(%q) = %d, regex says %d", buf, got, want)
		}
	})
}

// Strip must equal ReplaceAll for the two wrapcmd call sites that used it
// (wrap.go:812 stripTerminalControls, wrap.go:1151 the capture-early path).
func FuzzStripMatchesRegexReplaceAll(f *testing.F) {
	for _, s := range oracleCorpus {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		// Both halves matter, and only the first one existed when Strip shipped a
		// fast path that ALIASED its input: comparing by value cannot see storage,
		// so a 20M-execution session stayed green while callers were corrupting
		// their own buffers. The mutation check is the generic form of that bug.
		before := append([]byte(nil), buf...)
		got := Strip(buf)
		want := otherEscRe.ReplaceAll(buf, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("Strip(%q) = %q, ReplaceAll says %q", buf, got, want)
		}
		if !bytes.Equal(buf, before) {
			t.Fatalf("Strip mutated its input: %q -> %q", before, buf)
		}
		// Writing through the result must not reach the input either.
		for i := range got {
			got[i] = 0
		}
		if !bytes.Equal(buf, before) {
			t.Fatalf("Strip returned storage aliasing its input: %q -> %q", before, buf)
		}
	})
}

// The seed corpus as a plain table too, so `go test` (no -fuzz) still checks the
// interesting cases on every run rather than only under an explicit fuzz session.
func TestOracleAgreesOnCorpus(t *testing.T) {
	for _, s := range oracleCorpus {
		buf := []byte(s)
		if got, want := SequenceLen(buf), regexLenAt(buf); got != want {
			t.Errorf("SequenceLen(%q) = %d, regex says %d", buf, got, want)
		}
		if got, want := Strip(buf), otherEscRe.ReplaceAll(buf, nil); !bytes.Equal(got, want) {
			t.Errorf("Strip(%q) = %q, ReplaceAll says %q", buf, got, want)
		}
	}
}
