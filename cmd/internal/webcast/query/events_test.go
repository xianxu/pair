package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestEventLinesWriteShortRunsOut(t *testing.T) {
	recs, _ := ContractRecording()
	lines, err := EventLines(recs, 1500, 2100)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`1500 input @1 <input#name> value "Ada"`,
		`2000 click @3 <button#save> "Save" at 100,40`,
		`2100 attr @3 <button#save> "Save" disabled=""`,
	}
	if got := strings.Join(lines, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

// A busy re-render is one line: counts by kind and the most-touched
// elements, never thousands of lines.
func TestEventLinesFoldBusyRuns(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("rec-stress"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := EventLines(r.Records, 0, 99999)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines)*10 > len(r.Records) {
		t.Errorf("%d lines for %d records: runs aren't folding", len(lines), len(r.Records))
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "dom changes until") && strings.Contains(l, " kids") && strings.Contains(l, "<tbody#tb>") {
			found = true
		}
	}
	if !found {
		t.Errorf("no folded run names the re-rendered table")
	}
}

// A field typed a key at a time is one line with its final value, the
// framework's value attribute folded in.
func TestEventLinesFoldTyping(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("rec-app"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := EventLines(r.Records, 2830, 3600)
	if err != nil {
		t.Fatal(err)
	}
	// The recorder's caret after each key (a selection) folds in too.
	want := `2834-3514 input @69 <input#draft.flex-1.border.…> value "Review PR 42" (24 changes)`
	if !strings.Contains(strings.Join(lines, "\n"), want) {
		t.Errorf("want %q in\n%s", want, strings.Join(lines, "\n"))
	}
}

// A collapsed caret folds into the typing it follows, and only the same
// person's: another's caret, or a selection that isn't a caret, splits it.
func TestEventLinesFoldCaret(t *testing.T) {
	b := recording.NewBuilder("caret")
	field := b.El("input", []string{"id", "q"})
	b.Page(recording.SegLive, "http://app/", field)
	ana := recording.Author{ID: "v1", Name: "Ana"}
	var c recording.Control
	for _, op := range []string{recording.CtlRequest, recording.CtlGrant} {
		next, err := recording.NextControl(c, op, ana)
		if err != nil {
			t.Fatal(err)
		}
		c = next
		b.At(5).Add(&recording.Ctl{Op: op, Who: ana, Control: next})
	}
	at := func(off int64) *recording.Pos { return &recording.Pos{int64(field.ID), off} }
	typed := func(t int64, v, by string) {
		b.At(t).Add(&recording.Input{ID: field.ID, V: &v, Source: recording.Source{By: by}})
	}
	caret := func(t int64, from, to int64, by string) {
		b.At(t).Add(&recording.Selection{Anchor: at(from), Focus: at(to), Source: recording.Source{By: by}})
	}
	typed(10, "a", "")
	caret(11, 0, 0, "")
	typed(20, "ab", "")
	caret(21, 0, 0, "v1") // not the typist's
	typed(30, "abc", "")
	caret(31, 0, 2, "") // a range, not a caret
	typed(40, "abcd", "")
	typed(50, "abcde", "v1")
	caret(51, 0, 0, "v1") // the driver's own, after their key
	typed(60, "abcdef", "v1")
	lines, err := EventLines(b.Recs, 10, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 6 || !strings.HasPrefix(lines[0], `10-20 input`) || !strings.Contains(lines[0], `value "ab" (2 changes)`) ||
		!strings.HasPrefix(lines[5], `50-60 input`) || !strings.Contains(lines[5], `value "abcdef" by Ana (v1) (2 changes)`) {
		t.Errorf("%q", lines)
	}
}

// Who did what: a driver's acts say so, by name; the operator's don't. The
// moments come from the recorded fixture's own records (lesson #474).
func TestEventLinesAttribute(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("rec-driven"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := EventLines(r.Records, 0, r.Records[len(r.Records)-1].Time())
	if err != nil {
		t.Fatal(err)
	}
	at := func(t0 int64, kind string) []string {
		var out []string
		for _, l := range lines {
			if strings.HasPrefix(l, fmt.Sprintf("%d ", t0)) && strings.Contains(l, kind) {
				out = append(out, l)
			}
		}
		return out
	}
	var driverInput, driverClick, operatorClick, grant bool
	driving := ""
	for _, rec := range r.Records {
		switch x := rec.(type) {
		case *recording.Ctl:
			driving = x.Driving()
			if x.Op == recording.CtlGrant {
				for _, l := range at(x.T, "control grant") {
					grant = grant || strings.Contains(l, fmt.Sprintf("control grant %s (%s) → %s (%s) drives", x.Who.Name, x.Who.ID, x.Who.Name, x.Who.ID))
				}
			}
		case *recording.Input:
			if x.By != "" {
				for _, l := range at(x.T, " input @") {
					driverInput = driverInput || strings.Contains(l, " by Ana (v1)")
				}
			}
		case *recording.Click:
			for _, l := range at(x.T, fmt.Sprintf("click @%d ", x.ID)) {
				switch {
				case x.By != "":
					driverClick = driverClick || strings.HasSuffix(l, " by Ana (v1)")
				case driving != "":
					operatorClick = operatorClick || !strings.Contains(l, " by ")
				}
			}
		}
	}
	if !driverInput || !driverClick || !operatorClick || !grant {
		t.Errorf("attribution: driver input %v, driver click %v, operator click while driving %v, grant %v\n%s",
			driverInput, driverClick, operatorClick, grant, strings.Join(lines, "\n"))
	}
}

// A removed element is named as it was, before it went.
func TestEventLinesNameRemovedNodes(t *testing.T) {
	b := recording.NewBuilder("rm")
	gone := b.El("li", []string{"id", "old"}, b.Txt("Old item"))
	b.Page(recording.SegLive, "http://app/", b.El("ul", nil, gone))
	b.At(10).Add(&recording.Rm{ID: gone.ID})
	lines, err := EventLines(b.Recs, 10, 10)
	if err != nil || len(lines) != 1 || lines[0] != fmt.Sprintf(`10 remove @%d <li#old> "Old item"`, gone.ID) {
		t.Errorf("%v %q", err, lines)
	}
}
