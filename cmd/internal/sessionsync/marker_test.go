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

// A side-doc response, the index and an ordinary body edit all use clean
// highlight writes even when the document carries a retired propose key.
func TestCollaborativeMarkerResponseAndIndex(t *testing.T) {
	link := "→ [discussion](doc.side/rewrite.md)"
	edits := []EditRecord{
		{Old: "🤖[rewrite this section]", Occurrence: 1, New: link, Explain: "side doc"},
		{Old: "  - doc: doc.side/old.md\n", Occurrence: 1, New: "  - doc: doc.side/old.md\n  - doc: doc.side/rewrite.md\n"},
		{Old: "Intro.", Occurrence: 1, New: "Preface."},
	}
	e, err := BuildEdit(proposeParent, edits, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "\n"+link+"\n") || !strings.Contains(e.Next, "  - doc: doc.side/rewrite.md\n") || !strings.Contains(e.Next, "Preface.") || strings.Contains(e.Next, "🤖") {
		t.Fatalf("edit=%+v err=%v", e, err)
	}
	for _, span := range e.Round.Spans {
		if span.Presentation != "highlight" {
			t.Fatalf("span=%+v", span)
		}
	}
}

func TestCollaborativeMarkerCommandsWriteCleanEdits(t *testing.T) {
	for _, tc := range []struct{ old, next string }{
		{"This sentence is far too long. 🤖[tighten this]", "This is too long."},
		{"🤖[why?]", "→ [discussion](doc.side/why.md)"},
		{"Line one.\nLine two.\nLine three.\nLine four. 🤖[shorten]", "A.\nB.\nC.\nD.\nE. → [discussion](doc.side/rewrite.md)"},
		{"Old. → [discussion](doc.side/x.md) 🤖[shorten]", "New. → [discussion](doc.side/x.md)"},
		{"→ [discussion](doc.side/x.md) 🤖[fold in]\na\nb\nc\nd", "A\nB\nC\nD\nE → [discussion](doc.side/x.md) (folded 2026-10-23)"},
	} {
		t.Run(tc.old, func(t *testing.T) {
			base := "---\npresentation: propose\n---\n\n" + tc.old + "\n"
			e, err := BuildEdit(base, []EditRecord{{Old: tc.old, New: tc.next, Occurrence: 1}}, t0Propose, false)
			if err != nil || e.Next != strings.Replace(base, tc.old, tc.next, 1) {
				t.Fatalf("edit=%+v err=%v", e, err)
			}
		})
	}
}

func TestCollaborativeFrontmatterKeepsRetiredKeys(t *testing.T) {
	for _, edit := range []EditRecord{
		{Old: "presentation: propose", New: "presentation: direct", Occurrence: 1},
		{Old: "presentation: propose", New: "presentation: propose\ncollaboration-mode: discussion", Occurrence: 1},
	} {
		_, err := BuildEdit(proposeParent, []EditRecord{edit}, t0Propose, false)
		wantRefusal(t, err, ExitRefused, "operator-owned")
	}
	// With no proposal-encoding transform, a resolved record can span the
	// frontmatter boundary when it preserves every protected source byte.
	edits := []EditRecord{{Old: "doc.side/old.md\n---\n\nIntro.", New: "doc.side/new.md\n---\n\nPreface.", Occurrence: 1}}
	e, err := BuildEdit(proposeParent, edits, t0Propose, false)
	if err != nil || !strings.Contains(e.Next, "doc.side/new.md\n---\n\nPreface.") {
		t.Fatalf("edit=%+v err=%v", e, err)
	}
}
