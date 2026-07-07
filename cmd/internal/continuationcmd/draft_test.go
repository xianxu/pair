package continuationcmd

import (
	"strings"
	"testing"
)

func TestStripStickyComments(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"drops === lines", "=== label ===\nreal WIP\n=== end ===", "real WIP"},
		{"trims blank edges", "\n\n  hi\n\n", "  hi"},
		{"keeps interior blanks", "a\n\nb", "a\n\nb"},
		{"indented === also a comment", "   === x ===\nkeep", "keep"},
		{"all comments -> empty", "=== a ===\n=== b ===", ""},
		{"empty -> empty", "", ""},
		{"no comments passthrough", "line1\nline2", "line1\nline2"},
	}
	for _, c := range cases {
		if got := StripStickyComments(c.in); got != c.want {
			t.Errorf("%s: StripStickyComments(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestFoldDraftIntoNextAction(t *testing.T) {
	body := "## NEXT ACTION\n\nrun the thing\n\n## State of play\n\ndone"
	got := FoldDraftIntoNextAction(body, "half-typed idea")
	if !strings.Contains(got, "run the thing") || !strings.Contains(got, "half-typed idea") {
		t.Fatalf("WIP not folded under NEXT ACTION:\n%s", got)
	}
	// folded WIP must sit inside NEXT ACTION, before the next section
	na := strings.Index(got, "## NEXT ACTION")
	sop := strings.Index(got, "## State of play")
	wip := strings.Index(got, "half-typed idea")
	if !(na < wip && wip < sop) {
		t.Fatalf("WIP not positioned inside NEXT ACTION section:\n%s", got)
	}
	// existing NEXT ACTION content stays before the folded WIP
	if strings.Index(got, "run the thing") > wip {
		t.Errorf("folded WIP should come after existing NEXT ACTION content:\n%s", got)
	}

	// no-ops
	if FoldDraftIntoNextAction(body, "") != body {
		t.Error("empty WIP should be a no-op")
	}
	if FoldDraftIntoNextAction(body, "   \n\n ") != body {
		t.Error("blank-only WIP should be a no-op")
	}
	if got := FoldDraftIntoNextAction("## Other\n\nx", "wip"); strings.Contains(got, "wip") {
		t.Error("no NEXT ACTION -> no fold")
	}

	// section at EOF (no trailing heading)
	eof := FoldDraftIntoNextAction("## NEXT ACTION\n\ndo it", "tail wip")
	if !strings.Contains(eof, "tail wip") {
		t.Errorf("EOF section: WIP not folded:\n%s", eof)
	}

	// empty NEXT ACTION heading + WIP: fold still lands the WIP under the heading
	// (this runs before the writer's HasNextAction guard — judge nit).
	empty := FoldDraftIntoNextAction("## NEXT ACTION\n\n## State of play\n\nx", "rescued wip")
	if !strings.Contains(empty, "rescued wip") {
		t.Errorf("empty-heading fold lost the WIP:\n%s", empty)
	}
	if ena, ewip, esop := strings.Index(empty, "## NEXT ACTION"), strings.Index(empty, "rescued wip"), strings.Index(empty, "## State of play"); !(ena < ewip && ewip < esop) {
		t.Errorf("empty-heading fold mispositioned:\n%s", empty)
	}
	// empty section: WIP is the next action → NextActionPreview must surface the
	// WIP, not the fold label (close-review minor: no label leak in the preview).
	if empty := FoldDraftIntoNextAction("## NEXT ACTION\n", "rescued wip"); NextActionPreview(empty) != "rescued wip" {
		t.Errorf("empty-section rescue: preview = %q, want the WIP (no label leak)\n%s", NextActionPreview(empty), empty)
	}
	// non-empty section: the label IS present (distinguishes folded WIP).
	if got := FoldDraftIntoNextAction(body, "extra"); !strings.Contains(got, "_Parked draft at compaction:_") {
		t.Errorf("non-empty section should label the folded WIP:\n%s", got)
	}
}

func TestInCompactionContext(t *testing.T) {
	if !InCompactionContext("mytag", "pair-mytag") {
		t.Error("matching tag+session should be compaction context")
	}
	if !InCompactionContext("mytag", "pair-work-mytag") {
		t.Error("scoped public session should be compaction context")
	}
	if InCompactionContext("", "pair-") {
		t.Error("empty tag is never compaction")
	}
	if InCompactionContext("mytag", "pair-other") {
		t.Error("sibling session (leaked env) must not match")
	}
	if InCompactionContext("mytag", "") {
		t.Error("no zellij session -> not in a pane")
	}
}
