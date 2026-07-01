package continuationcmd

import (
	"strings"
	"testing"
	"time"
)

func TestRenderFrontmatter(t *testing.T) {
	f := Fields{
		Slug: "robotics", Agent: "claude", SessionID: "7f3a",
		Issues: []string{"000071", "000073"}, Branch: "main",
		Created: time.Date(2026, 6, 11, 14, 20, 0, 0, time.UTC),
	}
	got := RenderFrontmatter(f)
	want := "type: continuation\nslug: robotics\nagent: claude\n" +
		"session_id: 7f3a\ncreated: 2026-06-11T14:20:00\nbranch: main\n" +
		"issues: [000071, 000073]\n"
	if got != want {
		t.Fatalf("frontmatter:\n got %q\nwant %q", got, want)
	}
}

func TestRenderFrontmatter_OmitsEmptyOptionals(t *testing.T) {
	f := Fields{
		Slug: "x", Agent: "claude", Issues: []string{"000001"},
		Created: time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC),
	}
	got := RenderFrontmatter(f)
	for _, k := range []string{"session_id:", "supersedes:", "branch:", "worktree:"} {
		if strings.Contains(got, k) {
			t.Fatalf("optional %q should be omitted when empty:\n%s", k, got)
		}
	}
	// required fields always present
	for _, k := range []string{"type: continuation", "slug: x", "agent: claude", "issues: [000001]"} {
		if !strings.Contains(got, k) {
			t.Fatalf("missing required %q:\n%s", k, got)
		}
	}
}

func TestAllocName_NoClash(t *testing.T) {
	ts := time.Date(2026, 6, 11, 14, 20, 5, 0, time.UTC)
	if got := AllocName("robotics", ts, nil); got != "20260611T142005-robotics.md" {
		t.Fatalf("got %q", got)
	}
}

func TestAllocName_Clash(t *testing.T) {
	ts := time.Date(2026, 6, 11, 14, 20, 5, 0, time.UTC)
	existing := []string{"20260611T142005-robotics.md"}
	if got := AllocName("robotics", ts, existing); got != "20260611T142005-robotics-1.md" {
		t.Fatalf("got %q", got)
	}
}

func TestAllocName_DoubleClash(t *testing.T) {
	ts := time.Date(2026, 6, 11, 14, 20, 5, 0, time.UTC)
	existing := []string{"20260611T142005-x.md", "20260611T142005-x-1.md"}
	if got := AllocName("x", ts, existing); got != "20260611T142005-x-2.md" {
		t.Fatalf("got %q", got)
	}
}

func TestAssemble(t *testing.T) {
	got := Assemble("type: continuation\nslug: x\n", "# Continuation: x\n\n## NEXT ACTION\nGo.")
	want := "---\ntype: continuation\nslug: x\n---\n\n# Continuation: x\n\n## NEXT ACTION\nGo.\n"
	if got != want {
		t.Fatalf("assemble:\n got %q\nwant %q", got, want)
	}
}

func TestHasNextAction(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"content on next line", "# C\n\n## NEXT ACTION\nDo the thing.\n", true},
		{"blank then content", "## NEXT ACTION\n\n\nDo the thing.\n", true},
		{"list content", "## NEXT ACTION\n- step one\n", true},
		{"heading then EOF", "# C\n\n## NEXT ACTION\n", false},
		{"heading then blanks then EOF", "## NEXT ACTION\n\n   \n", false},
		{"heading immediately followed by another heading", "## NEXT ACTION\n\n## Open questions\nstuff\n", false},
		{"first content line is a #NN issue ref (not a heading)", "## NEXT ACTION\n#52: drain the minors\n", true},
		{"first content line is a real h1 heading => empty", "## NEXT ACTION\n# Title\nbody\n", false},
		{"no heading at all", "# C\n\nsome prose\n", false},
		{"indented heading still counts", "  ## NEXT ACTION\n  go\n", true},
	}
	for _, c := range cases {
		if got := HasNextAction(c.body); got != c.want {
			t.Errorf("%s: HasNextAction = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestValidateFields(t *testing.T) {
	ok := Fields{Slug: "x", Agent: "claude", Issues: []string{"000001"}}
	if err := ValidateFields(ok); err != nil {
		t.Fatalf("valid fields rejected: %v", err)
	}
	bad := []Fields{
		{Agent: "claude", Issues: []string{"1"}}, // no slug
		{Slug: "x", Issues: []string{"1"}},       // no agent
		{Slug: "x", Agent: "claude"},             // no issues
	}
	for i, f := range bad {
		if ValidateFields(f) == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}
