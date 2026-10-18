package sessionsync

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	fold := "A.\nB.\nC.\nD.\nE. " + link + " (folded 2026-10-18)\n"
	e, err = BuildEdit(folded, []EditRecord{{Old: link + " 🤖[fold in]\nLine one.\nLine two.\nLine three.\nLine four.\n", Occurrence: 1, New: fold}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "\n"+fold) || strings.Contains(e.Next, "🤖") {
		t.Fatalf("fold-in not applied directly: %q %v", e.Next, err)
	}
	// S3: a fold keeps its link. One that drops the link, or keeps the
	// fold-in marker, is not the fold the operator asked for.
	for _, bad := range []string{"A.\nB.\nC.\nD.\nE.\n", fold + "🤖[fold in]\n"} {
		_, err = BuildEdit(folded, []EditRecord{{Old: link + " 🤖[fold in]\nLine one.\nLine two.\nLine three.\nLine four.\n", Occurrence: 1, New: bad}}, t0Propose, false)
		if err == nil {
			t.Fatalf("a fold without its folded link was applied: %q", bad)
		}
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

type conformanceCase struct {
	Name, Base, After, Refuse string
	Records                   []struct {
		Old, New, Explain string
		Occurrence        int
	}
}

// The shared conformance fixtures (TL, 2026-10-18): the exact before/after
// bytes of ariadne#437's universal-review examples under presentation:
// propose, so the engine and the skill test the same documents.
func TestProposeConformanceFixtures(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(editFixturesDir, "conformance", "*.json"))
	if len(paths) == 0 {
		t.Fatal("no conformance fixtures")
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var c conformanceCase
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		t.Run(c.Name, func(t *testing.T) {
			var edits []EditRecord
			for _, r := range c.Records {
				edits = append(edits, EditRecord{Old: r.Old, New: r.New, Explain: r.Explain, Occurrence: max(r.Occurrence, 1)})
			}
			e, err := BuildEdit(c.Base, edits, t0Propose, false)
			if c.Refuse != "" {
				wantRefusal(t, err, ExitRefused, c.Refuse)
				return
			}
			if err != nil || e.Next != c.After {
				t.Fatalf("got %q err=%v\nwant %q", e.Next, err, c.After)
			}
		})
	}
}

// Rule (a) is a new link replacing the marker: an answer to a marker that
// already sits beside a link is not a side-doc answer, so it can't slip a
// rewrite through by keeping the link.
func TestProposeExistingLinkIsNotANewAnswer(t *testing.T) {
	link := "→ [proposal](doc.side/x.md)"
	doc := "---\npresentation: propose\n---\n\nOld text. " + link + " 🤖[shorten]\n"
	e, err := BuildEdit(doc, []EditRecord{{Old: "Old text. " + link + " 🤖[shorten]", Occurrence: 1, New: "New. " + link}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "🤖~") {
		t.Fatalf("a rewrite beside an existing link applied directly: %q %v", e.Next, err)
	}
}

// Rule (a) is the marker swapped for the link and nothing else (close review
// BR-9): a rewrite that ends in a link is still a proposal, or refused when
// it is large.
func TestProposeLinkAnswerCannotCarryARewrite(t *testing.T) {
	doc := "---\npresentation: propose\n---\n\nLine one.\nLine two.\nLine three.\nLine four. 🤖[shorten]\n"
	_, err := BuildEdit(doc, []EditRecord{{Old: "Line one.\nLine two.\nLine three.\nLine four. 🤖[shorten]", Occurrence: 1,
		New: "Totally rewritten.\nX\nY\nZ\nW → [proposal](doc.side/x.md)"}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-too-large")
	e, err := BuildEdit(doc, []EditRecord{{Old: "Line four. 🤖[shorten]", Occurrence: 1, New: "Line 4. → [proposal](doc.side/x.md)"}}, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "🤖~") {
		t.Fatalf("a text change beside the link applied directly: %q %v", e.Next, err)
	}
	e, err = BuildEdit(doc, []EditRecord{{Old: "Line four. 🤖[shorten]", Occurrence: 1, New: "Line four. → [proposal](doc.side/x.md)"}}, t0Propose, false)
	if err != nil || strings.Contains(e.Next, "🤖") {
		t.Fatalf("the exact swap was not applied directly: %q %v", e.Next, err)
	}
}
