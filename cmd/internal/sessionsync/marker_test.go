package sessionsync

import (
	"strings"
	"testing"
	"time"
)

func TestReadyMarkerIn(t *testing.T) {
	for text, want := range map[string]bool{
		"see 🤖[rewrite this section]":          true,
		"🤖<quoted text>[tighten]":              true,
		"🤖[a [nested] ask]":                    true,
		"🤖{agent asked}[operator answered]":    true,
		"🤖[asked]{agent answered}":             false, // the agent spoke last
		"🤖~struck~[note]":                      false, // a strike is never ready
		"🤖[]":                                  false,
		"🤖[unclosed":                           false,
		"🤖\\[escaped\\]":                       false,
		"no marker [here]":                     false,
		"🤖{only agent} then 🤖[fold in] → link": true,
	} {
		if got := readyMarkerIn(text); got != want {
			t.Errorf("readyMarkerIn(%q)=%v, want %v", text, got, want)
		}
	}
}

const proposeParent = "---\npresentation: propose\ndiscussions:\n  - doc: doc.side/old.md\n---\n\nIntro.\n🤖[rewrite this section]\nLine one.\nLine two.\nLine three.\nLine four.\n"

var t0Propose = time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)

// Under propose, the record answering the operator's ready marker with the
// side-doc link applies directly; the same change without the marker is a
// proposal, and a large rewrite without it is refused.
func TestProposeAnsweringTheOperatorAppliesDirectly(t *testing.T) {
	link := "→ [proposal](doc.side/rewrite.md)"
	e, err := BuildEdit(proposeParent, []EditRecord{{Old: "🤖[rewrite this section]", Occurrence: 1, New: link, Explain: "side doc"}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "\n"+link+"\n") || strings.Contains(e.Next, "🤖~") {
		t.Fatalf("answer not applied directly: %q %v", e.Next, err)
	}
	if len(e.Round.Spans) != 1 || e.Round.Spans[0].Explain != "side doc" {
		t.Fatalf("round %+v", e.Round)
	}
	// No marker in old: the exception does not apply. A one-line change is a
	// proposal; a rewrite over three lines is refused.
	e, err = BuildEdit(proposeParent, []EditRecord{{Old: "Intro.", Occurrence: 1, New: "Preface."}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "🤖~Intro~{Preface}") {
		t.Fatalf("plain edit not proposed: %q %v", e.Next, err)
	}
	_, err = BuildEdit(proposeParent, []EditRecord{{Old: "Line one.\nLine two.\nLine three.\nLine four.\n", Occurrence: 1, New: "A.\nB.\nC.\nD.\n"}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-too-large")
	// The fold-in: old holds the link and its 🤖[fold in] marker, so the large
	// folded text applies directly.
	folded := strings.Replace(proposeParent, "🤖[rewrite this section]", link+" 🤖[fold in]", 1)
	e, err = BuildEdit(folded, []EditRecord{{Old: link + " 🤖[fold in]\nLine one.\nLine two.\nLine three.\nLine four.\n", Occurrence: 1, New: "A.\nB.\nC.\nD.\nE.\n"}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "\nA.\nB.\nC.\nD.\nE.\n") || strings.Contains(e.Next, "🤖") {
		t.Fatalf("fold-in not applied directly: %q %v", e.Next, err)
	}
}

// The side-doc index rides with the operator-approved link (S3): the matching
// discussions: entry may change in the same apply; any other frontmatter
// change is still refused.
func TestProposeAllowsOnlyTheMatchingDiscussionsChange(t *testing.T) {
	link := "→ [proposal](doc.side/rewrite.md)"
	answer := EditRecord{Old: "🤖[rewrite this section]", Occurrence: 1, New: link}
	index := EditRecord{Old: "  - doc: doc.side/old.md\n", Occurrence: 1, New: "  - doc: doc.side/old.md\n  - doc: doc.side/rewrite.md\n"}
	e, err := BuildEdit(proposeParent, []EditRecord{answer, index}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "  - doc: doc.side/rewrite.md\n") || strings.Contains(e.Next, "🤖") {
		t.Fatalf("link plus index not applied: %q %v", e.Next, err)
	}
	// An index entry for another doc, or any other key, is not the answer's.
	other := EditRecord{Old: "  - doc: doc.side/old.md\n", Occurrence: 1, New: "  - doc: doc.side/old.md\n  - doc: doc.side/unrelated.md\n"}
	_, err = BuildEdit(proposeParent, []EditRecord{answer, other}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-unencodable")
	key := EditRecord{Old: "presentation: propose", Occurrence: 1, New: "presentation: direct"}
	_, err = BuildEdit(proposeParent, []EditRecord{answer, key}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-unencodable")
	// Without a qualifying answer the index alone can't change either.
	_, err = BuildEdit(proposeParent, []EditRecord{index}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-unencodable")
}
