package sessionsync

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Round is the change an agent write made (pair#661): the spans that differ
// between the file before and after, derived on every agent write. It is the
// round sidecar's content and `round --json`'s Round.
type Round struct {
	BaseHash     string    `json:"base_hash"`
	ResultHash   string    `json:"result_hash"`
	At           time.Time `json:"at"`
	Presentation string    `json:"presentation"`
	Spans        []Span    `json:"spans"`
}

// Span is one changed region. Start and End are 0-based [line, byte column]
// positions in the result; a deletion has Start == End. Explain is set only
// on spans an `apply` record produced.
type Span struct {
	Start   [2]int `json:"start"`
	End     [2]int `json:"end"`
	Old     string `json:"old"`
	New     string `json:"new"`
	Explain string `json:"explain,omitempty"`
}

// Presentation names how parley shows a round, read from the document's
// `presentation:`; missing or unknown means highlight.
func Presentation(content string) string {
	switch p := frontmatter(content, "presentation"); p {
	case "direct", "propose":
		return p
	default:
		return "highlight"
	}
}

// changedRun is one diff hunk narrowed to the bytes that differ: [OldStart,
// OldEnd) in the baseline became [NewStart, NewEnd) in the result.
type changedRun struct {
	OldStart, OldEnd, NewStart, NewEnd int
}

// changedRuns diffs baseline against after by line, then narrows each hunk to
// its changed bytes, never splitting a UTF-8 rune.
func changedRuns(baseline, after string) []changedRun {
	a, b := splitLines(baseline), splitLines(after)
	offsets := func(lines []string) []int {
		off := make([]int, len(lines)+1)
		for i, l := range lines {
			off[i+1] = off[i] + len(l)
		}
		return off
	}
	ao, bo := offsets(a), offsets(b)
	var runs []changedRun
	for _, h := range LineDiff(a, b) {
		oldText := baseline[ao[h.OldLo]:ao[h.OldHi]]
		newText := after[bo[h.NewLo]:bo[h.NewHi]]
		p, s := narrow(oldText, newText)
		runs = append(runs, changedRun{
			OldStart: ao[h.OldLo] + p, OldEnd: ao[h.OldHi] - s,
			NewStart: bo[h.NewLo] + p, NewEnd: bo[h.NewHi] - s,
		})
	}
	return runs
}

// narrow returns the lengths of the common prefix and suffix of x and y,
// not overlapping and each ending on a rune boundary.
func narrow(x, y string) (prefix, suffix int) {
	for prefix < len(x) && prefix < len(y) && x[prefix] == y[prefix] {
		prefix++
	}
	for prefix > 0 && ((prefix < len(x) && !utf8.RuneStart(x[prefix])) || (prefix < len(y) && !utf8.RuneStart(y[prefix]))) {
		prefix--
	}
	for suffix < len(x)-prefix && suffix < len(y)-prefix && x[len(x)-1-suffix] == y[len(y)-1-suffix] {
		suffix++
	}
	for suffix > 0 && !utf8.RuneStart(x[len(x)-suffix]) {
		suffix--
	}
	return prefix, suffix
}

// DeriveRound returns the spans that turn baseline into after, with each
// span's reason taken from the resolved records it overlaps (attachReasons).
// baseline is the file before the write, restamped exactly when the write
// restamps, so the stamp line never forms a span. The stamp has a fixed width
// (`stamp`), so resolved ranges, taken before the restamp, hold in the
// baseline unchanged. No span means no round.
func DeriveRound(baseline, after string, resolved []ResolvedEdit) ([]Span, bool) {
	runs := changedRuns(baseline, after)
	if len(runs) == 0 {
		return nil, false
	}
	starts := lineStarts(after)
	spans := make([]Span, len(runs))
	for i, r := range runs {
		spans[i] = Span{
			Start: position(starts, r.NewStart), End: position(starts, r.NewEnd),
			Old: baseline[r.OldStart:r.OldEnd], New: after[r.NewStart:r.NewEnd],
		}
	}
	attachReasons(spans, runs, resolved)
	return spans, true
}

// attachReasons is the one rule for reasons, in base coordinates: a span
// takes the Explain of every edit whose base range overlaps its old range (an
// empty old range, an insertion, takes the edits it touches). Distinct
// reasons join with "; ".
func attachReasons(spans []Span, runs []changedRun, edits []ResolvedEdit) {
	for i, r := range runs {
		var reasons []string
		for _, e := range edits {
			hit := r.OldStart < e.BaseEnd && e.BaseStart < r.OldEnd
			if r.OldStart == r.OldEnd {
				hit = e.BaseStart <= r.OldStart && r.OldStart <= e.BaseEnd
			}
			if hit && e.Explain != "" && !contains(reasons, e.Explain) {
				reasons = append(reasons, e.Explain)
			}
		}
		spans[i].Explain = strings.Join(reasons, "; ")
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// lineStarts is the byte offset of each line's first byte.
func lineStarts(s string) []int {
	starts := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// position is the 0-based [line, byte column] of a byte offset.
func position(starts []int, off int) [2]int {
	line := sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
	return [2]int{line, off - starts[line]}
}
