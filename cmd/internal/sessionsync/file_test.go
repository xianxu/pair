package sessionsync

import (
	"strings"
	"testing"
	"time"
)

const doc = "---\ntype: session-sync\nowner: ops:0\n---\n\n# TL status\n\n_Updated 2026-10-10 23:43._\n\n" +
	"## Needs you\n\nold needs, see `## Needs you` above\n\n" +
	"```\n## Needs you\n```\n\n" +
	"## Trains\n\nold trains\n"

func TestReplaceSectionMatchesWholeHeadingLinesOnly(t *testing.T) {
	// A heading quoted in prose or indented in a fence never matches; the
	// fenced line here is unindented, so it does count, which is why the
	// fixture refuses below and the indented variant succeeds.
	quoted := strings.Replace(doc, "```\n## Needs you\n```", "```\n  ## Needs you\n```", 1)
	got, err := ReplaceSection(quoted, "Needs you", "new needs\n")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(quoted, "old needs, see `## Needs you` above\n\n```\n  ## Needs you\n```\n", "new needs\n", 1)
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := ReplaceSection(doc, "Needs you", "x"); err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Fatalf("two whole-line headings must refuse, got %v", err)
	}
	if _, err := ReplaceSection(doc, "Needs", "x"); err == nil || !strings.Contains(err.Error(), "found 0") {
		t.Fatalf("a heading prefix must not match, got %v", err)
	}
}

func TestReplaceSectionLastSectionRunsToEOF(t *testing.T) {
	got, err := ReplaceSection("## A\n\na\n\n## B\n\nb\nmore\n", "B", "\nnew b\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if want := "## A\n\na\n\n## B\n\nnew b\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got, err = ReplaceSection("## A  \na\n## B\nb\n", "A", "x")
	if err != nil {
		t.Fatal(err)
	}
	if want := "## A  \n\nx\n\n## B\nb\n"; got != want {
		t.Fatalf("trailing space on a heading still matches: got %q", got)
	}
}

func TestRestampReplacesOnlyTheFirstStamp(t *testing.T) {
	at := time.Date(2026, 10, 11, 9, 5, 0, 0, time.Local)
	got := Restamp("Updated 2026-10-10 23:43. Updated 2026-01-01 00:00", at)
	if want := "Updated 2026-10-11 09:05. Updated 2026-01-01 00:00"; got != want {
		t.Fatalf("got %q", got)
	}
	if got := Restamp("no stamp", at); got != "no stamp" {
		t.Fatalf("got %q", got)
	}
}

func TestParseOwnerReadsFrontmatterOnly(t *testing.T) {
	if got := ParseOwner(doc); got != "ops:0" {
		t.Fatalf("got %q", got)
	}
	if got := ParseOwner("# no frontmatter\nowner: ops:1\n"); got != "" {
		t.Fatalf("an owner line in the body must not count, got %q", got)
	}
	if got := ParseOwner("---\ntype: session-sync\n---\nowner: ops:1\n"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := ParseOwner("---\nowner: \"ops:2\"  # the TL\n---\n"); got != "ops:2" {
		t.Fatalf("quotes and comments are stripped, got %q", got)
	}
}

func TestParseHolder(t *testing.T) {
	for _, c := range []struct {
		lock string
		ok   bool
		want Holder
	}{
		{"", false, HolderFree},
		{"holder: agent\nat: x\n", true, HolderAgent},
		{"holder: operator\n", true, HolderOperator},
		{"garbage\n", true, HolderOperator},
		{"", true, HolderOperator},
	} {
		if got := ParseHolder(c.lock, c.ok); got != c.want {
			t.Fatalf("ParseHolder(%q, %v) = %q, want %q", c.lock, c.ok, got, c.want)
		}
	}
}

func TestPathsNameTheSidecars(t *testing.T) {
	p := PathsOf("/a/b/tl-status-x.md")
	if p.Lock != "/a/b/tl-status-x.md.lock" || p.Shadow != "/a/b/.tl-status-x.md.shadow" || p.State != "/a/b/.tl-status-x.md.sync" {
		t.Fatalf("got %+v", p)
	}
}
