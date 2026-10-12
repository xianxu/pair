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
	r, err := recording.ReadDir(recording.Fixture("stress"))
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
	r, err := recording.ReadDir(recording.Fixture("app"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := EventLines(r.Records, 4400, 5200)
	if err != nil {
		t.Fatal(err)
	}
	want := `4414-5116 input @73 <input#draft.flex-1.border.…> value "Review PR 42" (19 changes)`
	if !strings.Contains(strings.Join(lines, "\n"), want) {
		t.Errorf("want %q in\n%s", want, strings.Join(lines, "\n"))
	}
}

// Who did what: a driver's acts say so, by name; the operator's don't.
func TestEventLinesAttribute(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("driven"))
	if err != nil {
		t.Fatal(err)
	}
	lines, err := EventLines(r.Records, 0, 99999)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(lines, "\n")
	for _, want := range []string{
		`2400 input @1 <input#q> value "invoice" by Ana (v1)`,
		`3000 click @3 <button#save> "Save" at 370,30` + "\n",
		`7000 control grant Bo (v2) → Bo (v2) drives`,
		`7200 click @14 <p> "Page two" at 20,20 by Bo (v2)`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("want %q in\n%s", want, all)
		}
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
