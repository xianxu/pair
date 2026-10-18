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

// Frontmatter is never proposed (TL, 2026-10-18): under propose, records
// wholly inside it apply directly in any batch, the side-doc index included;
// the operator-owned keys refuse, and so does a record crossing its end.
func TestProposeFrontmatterRecordsApplyDirectly(t *testing.T) {
	link := "→ [proposal](doc.side/rewrite.md)"
	answer := EditRecord{Old: "🤖[rewrite this section]", Occurrence: 1, New: link}
	index := EditRecord{Old: "  - doc: doc.side/old.md\n", Occurrence: 1, New: "  - doc: doc.side/old.md\n  - doc: doc.side/rewrite.md\n"}
	e, err := BuildEdit(proposeParent, []EditRecord{answer, index}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "  - doc: doc.side/rewrite.md\n") || strings.Contains(e.Next, "🤖") {
		t.Fatalf("link plus index not applied: %q %v", e.Next, err)
	}
	// Any frontmatter record, alone or beside a body proposal, is direct.
	title := EditRecord{Old: "discussions:", Occurrence: 1, New: "title: S4\ndiscussions:"}
	body := EditRecord{Old: "Intro.", Occurrence: 1, New: "Preface."}
	e, err = BuildEdit(proposeParent, []EditRecord{title, body}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "---\npresentation: propose\ntitle: S4\n") || !strings.Contains(e.Next, "🤖~Intro~{Preface}") {
		t.Fatalf("frontmatter direct, body proposed: %q %v", e.Next, err)
	}
	// The operator-owned keys refuse, however the change arrives.
	for _, r := range []EditRecord{
		{Old: "presentation: propose", Occurrence: 1, New: "presentation: direct"},
		{Old: "presentation: propose", Occurrence: 1, New: "presentation: propose\ncollaboration-mode: discussion"},
	} {
		_, err = BuildEdit(proposeParent, []EditRecord{r}, t0Propose, false)
		wantRefusal(t, err, ExitRefused, "operator-owned")
	}
	// A record crossing the end of the frontmatter refuses.
	_, err = BuildEdit(proposeParent, []EditRecord{{Old: "doc.side/old.md\n---\n\nIntro.", Occurrence: 1, New: "x\n---\n\nY."}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-unencodable")
}

// An ordinary answer removes its marker too: a 🤖[tighten this] rewrite under
// propose is still a proposal. Only the side-doc link that answers the marker,
// or the fold the link's 🤖[fold in] asks for, applies directly.
func TestProposeOrdinaryAnswerStaysAProposal(t *testing.T) {
	doc := "---\npresentation: propose\n---\n\nThis sentence is far too long. 🤖[tighten this]\n"
	e, err := BuildEdit(doc, []EditRecord{{Old: "This sentence is far too long. 🤖[tighten this]", Occurrence: 1, New: "This is too long."}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "🤖~") {
		t.Fatalf("an ordinary answer applied directly: %q %v", e.Next, err)
	}
	// A link label that isn't a proposal or discussion is not a side-doc answer.
	e, err = BuildEdit(doc, []EditRecord{{Old: "🤖[tighten this]", Occurrence: 1, New: "→ [elsewhere](doc.side/x.md)"}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "🤖~") {
		t.Fatalf("a non-side-doc link applied directly: %q %v", e.Next, err)
	}
	// The discussion link is a side-doc answer.
	e, err = BuildEdit(doc, []EditRecord{{Old: "🤖[tighten this]", Occurrence: 1, New: "→ [discussion](doc.side/tighten.md)"}}, t0Propose, false)
	if err != nil || strings.Contains(e.Next, "🤖") {
		t.Fatalf("the discussion link was not applied directly: %q %v", e.Next, err)
	}
	// 🤖[fold in] not attached to a side-doc link does not qualify a fold.
	loose := "---\npresentation: propose\n---\n\n🤖[fold in]\na\nb\nc\nd\n"
	_, err = BuildEdit(loose, []EditRecord{{Old: "🤖[fold in]\na\nb\nc\nd\n", Occurrence: 1, New: "w\nx\ny\nz\nv\n"}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-too-large")
}
