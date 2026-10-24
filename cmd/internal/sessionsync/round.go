package sessionsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/editorial"
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
	unknown      map[string]json.RawMessage
}

// Span is one changed region. Start and End are 0-based [line, byte column]
// positions in the result; a deletion has Start == End. Explain is set only
// on spans an `apply` record produced.
type Span struct {
	Start        [2]int `json:"start"`
	End          [2]int `json:"end"`
	Old          string `json:"old"`
	New          string `json:"new"`
	Explain      string `json:"explain,omitempty"`
	Presentation string `json:"presentation,omitempty"`
	unknown      map[string]json.RawMessage
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

// roundMaxBytes caps the round sidecar on write and read (ARCH-CONSTRAINTS):
// a 4 MiB document's old and new text plus JSON escaping fits.
const roundMaxBytes = 16 << 20

// encodeRound is the sidecar's bytes, or false when they would pass the cap.
func encodeRound(r Round) ([]byte, bool) {
	raw, err := r.MarshalJSON()
	if err != nil || len(raw) > roundMaxBytes {
		return nil, false
	}
	return raw, true
}

// readRound reads and validates a round sidecar: nil, nil when there is none;
// a round-invalid refusal when it is oversized or not a round.
func readRound(path string) ([]byte, error) {
	raw, err := readRegular(path, roundMaxBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, roundInvalid(err.Error())
	}
	if _, err := decodeRound(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// decodeRound accepts only a well-formed round: hashes, a time, a known
// presentation, and spans whose positions are ordered and non-negative.
func decodeRound(raw []byte) (*Round, error) {
	var r Round
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, roundInvalid(err.Error())
	}
	if r.BaseHash == "" || r.ResultHash == "" || r.At.IsZero() {
		return nil, roundInvalid("missing hash or time")
	}
	switch r.Presentation {
	case "direct", "highlight", "propose":
	default:
		return nil, roundInvalid("unknown presentation " + r.Presentation)
	}
	for _, s := range r.Spans {
		if s.Start[0] < 0 || s.Start[1] < 0 || s.End[1] < 0 || s.End[0] < s.Start[0] || (s.End[0] == s.Start[0] && s.End[1] < s.Start[1]) {
			return nil, roundInvalid("a span's positions are out of order")
		}
	}
	return &r, nil
}

func roundInvalid(reason string) error {
	return &Refusal{Exit: ExitRefused, Code: "round-invalid", Reason: "the round sidecar is unreadable: " + reason}
}

// Edit is one agent change, built purely (pair#661): the bytes to write and
// the round they make. Changed is false for a write that changes nothing,
// which keeps the previous round.
type Edit struct {
	Next     string
	Round    Round
	Changed  bool
	Resolved []ResolvedEdit
}

// BuildEdit is the shared engine every edit path writes through (couch
// --session-sync apply and pair review apply): resolve the records all or
// nothing, then contentStep. restamp says whether this document carries an
// `Updated` stamp the write refreshes.
func BuildEdit(base string, edits []EditRecord, now time.Time, restamp bool) (Edit, error) {
	policy, err := compileEditorial(base)
	if err != nil {
		return Edit{}, err
	}
	return buildEditWithPolicy(base, edits, now, restamp, policy)
}

func buildEditWithPolicy(base string, edits []EditRecord, now time.Time, restamp bool, policy *editorial.DocumentPolicy) (Edit, error) {
	next, resolved, err := resolveBatch(base, edits)
	if err != nil {
		return Edit{}, err
	}
	return contentStep(base, next, resolved, now, restamp, policy)
}

// resolveBatch is ResolveEdits as a refusal: `edits-invalid` lists every bad
// record and nothing is written.
func resolveBatch(base string, edits []EditRecord) (string, []ResolvedEdit, error) {
	if len(edits) == 0 {
		return "", nil, refuse("apply needs at least one record")
	}
	next, resolved, bad := ResolveEdits(base, edits)
	if len(bad) != 0 {
		return "", nil, &Refusal{Exit: ExitRefused, Code: "edits-invalid", Invalid: bad, Reason: fmt.Sprintf(
			"%d of %d records don't resolve against the file; nothing was written. Fix them and apply again", len(bad), len(edits))}
	}
	return next, resolved, nil
}

// contentStep validates intended edits against the transaction's base source
// authority, then derives a highlight round from the clean written bytes.
// Restamping follows validation and never forms a span.
func contentStep(base, next string, resolved []ResolvedEdit, now time.Time, restamp bool, policy *editorial.DocumentPolicy) (Edit, error) {
	runs := changedRuns(base, next)
	changes := make([]editorial.Change, len(runs))
	for i, r := range runs {
		changes[i] = editorial.Change{Start: r.OldStart, End: r.OldEnd, New: next[r.NewStart:r.NewEnd]}
	}
	if _, err := editorial.ValidateEdits([]byte(base), policy, changes); err != nil {
		return Edit{}, editorialRefusal(err)
	}
	baseline := base
	if restamp {
		baseline, next = Restamp(base, now), Restamp(next, now)
	}
	e := Edit{Next: next, Resolved: resolved}
	if spans, ok := DeriveRound(baseline, next, resolved); ok {
		for i := range spans {
			spans[i].Presentation = "highlight"
		}
		e.Changed = true
		e.Round = Round{BaseHash: Hash(base), ResultHash: Hash(next), At: now.UTC(), Presentation: "highlight", Spans: spans}
	}
	return e, nil
}

func editorialRefusal(err error) error {
	var p *editorial.PolicyError
	if errors.As(err, &p) {
		return &Refusal{Exit: ExitRefused, Code: p.Code, Reason: p.Error()}
	}
	return &Refusal{Exit: ExitRefused, Code: "editorial-invalid", Reason: err.Error()}
}
