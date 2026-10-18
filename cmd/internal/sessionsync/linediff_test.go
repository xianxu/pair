package sessionsync

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// applyHunks rebuilds b from a and the hunks, taking new lines from b.
func applyHunks(a, b []string, hunks []Hunk) []string {
	var out []string
	at := 0
	for _, h := range hunks {
		out = append(out, a[at:h.OldLo]...)
		out = append(out, b[h.NewLo:h.NewHi]...)
		at = h.OldHi
	}
	return append(out, a[at:]...)
}

func TestLineDiffCases(t *testing.T) {
	cases := []struct {
		name, a, b string
		want       []Hunk
	}{
		{"identity", "x\ny\n", "x\ny\n", nil},
		{"insert", "x\nz\n", "x\ny\nz\n", []Hunk{{1, 1, 1, 2}}},
		{"delete", "x\ny\nz\n", "x\nz\n", []Hunk{{1, 2, 1, 1}}},
		{"replace", "x\ny\nz\n", "x\nY\nz\n", []Hunk{{1, 2, 1, 2}}},
		{"trailing newline", "x\ny", "x\ny\n", []Hunk{{1, 2, 1, 2}}},
		{"two hunks", "a\nb\nc\nd\ne\n", "a\nB\nc\nd\nE\n", []Hunk{{1, 2, 1, 2}, {4, 5, 4, 5}}},
	}
	for _, c := range cases {
		if got := LineDiff(splitLines(c.a), splitLines(c.b)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

// Adversarial strategy: random line sequences over a tiny alphabet maximize
// repeated lines, where a wrong backtrack misplaces hunks. Applying the hunks
// must rebuild b exactly, and the hunks must be ordered and disjoint.
func TestLineDiffRebuildsB(t *testing.T) {
	rng := rand.New(rand.NewSource(661))
	gen := func() []string {
		n := rng.Intn(12)
		lines := make([]string, n)
		for i := range lines {
			lines[i] = string(rune('a'+rng.Intn(3))) + "\n"
		}
		return lines
	}
	for i := 0; i < 2000; i++ {
		a, b := gen(), gen()
		hunks := LineDiff(a, b)
		if got := applyHunks(a, b, hunks); strings.Join(got, "") != strings.Join(b, "") {
			t.Fatalf("a=%q b=%q hunks=%+v rebuilt %q", a, b, hunks, got)
		}
		for j := 1; j < len(hunks); j++ {
			if hunks[j].OldLo < hunks[j-1].OldHi || hunks[j].NewLo < hunks[j-1].NewHi {
				t.Fatalf("unordered hunks %+v", hunks)
			}
		}
	}
}

func TestLineDiffEditCapFallsBackToOneHunk(t *testing.T) {
	saved := maxDiffEdits
	maxDiffEdits = 2
	defer func() { maxDiffEdits = saved }()
	a := splitLines("k\na\nb\nc\nk\n")
	b := splitLines("k\nx\ny\nz\nk\n")
	if got := LineDiff(a, b); !reflect.DeepEqual(got, []Hunk{{1, 4, 1, 4}}) {
		t.Fatalf("capped diff %+v, want the whole middle", got)
	}
}
