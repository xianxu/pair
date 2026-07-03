package launcher

import (
	"bytes"
	"strings"
	"testing"
)

// parseContinue mirrors the shell's `continue [slug] [agent] [-- args]` contract
// (the cases pair-continue-test.sh pins via PAIR_DEBUG_ARGS against the shell).
func TestParseContinue(t *testing.T) {
	bare, err := ParseArgs([]string{"continue"})
	if err != nil || bare.Command != "continue" || bare.ContinueSlug != "" {
		t.Fatalf("bare continue = %+v err=%v", bare, err)
	}
	slug, _ := ParseArgs([]string{"continue", "demo"})
	if slug.ContinueSlug != "demo" || slug.Agent != "" || len(slug.AgentArgs) != 0 {
		t.Fatalf("continue demo = %+v (agent from doc at resolve, not here)", slug)
	}
	port, _ := ParseArgs([]string{"continue", "demo", "codex"})
	if port.Agent != "codex" {
		t.Fatalf("explicit port = %+v", port)
	}
	fwd, _ := ParseArgs([]string{"continue", "demo", "--", "--dangerously-skip-permissions"})
	if fwd.Agent != "" || len(fwd.AgentArgs) != 1 || fwd.AgentArgs[0] != "--dangerously-skip-permissions" {
		t.Fatalf("-- forwarding (no port) = %+v", fwd)
	}
	fwd2, _ := ParseArgs([]string{"continue", "demo", "codex", "--", "--foo", "bar"})
	if fwd2.Agent != "codex" || strings.Join(fwd2.AgentArgs, " ") != "--foo bar" {
		t.Fatalf("-- forwarding after port = %+v", fwd2)
	}
	if _, err := ParseArgs([]string{"continue", "demo", "codex", "stray"}); err == nil {
		t.Fatal("a stray non-`--` arg after the port must error")
	}
}

func TestFrontmatterField(t *testing.T) {
	body := "---\nagent: claude\nissues: [#99, #93]\n---\n## NEXT ACTION\ngo\n"
	if got := frontmatterField(body, "agent"); got != "claude" {
		t.Fatalf("agent = %q", got)
	}
	if got := frontmatterField(body, "issues"); got != "[#99, #93]" {
		t.Fatalf("issues = %q", got)
	}
	if got := frontmatterField(body, "missing"); got != "" {
		t.Fatalf("missing = %q, want empty", got)
	}
}

func TestTruncatePreview(t *testing.T) {
	if got := truncatePreview("short"); got != "short" {
		t.Fatalf("short = %q", got)
	}
	long := strings.Repeat("X", 90) + "TAIL"
	got := truncatePreview(long)
	if !strings.HasSuffix(got, "…") || strings.Contains(got, "TAIL") {
		t.Fatalf("long truncation = %q", got)
	}
	if r := []rune(got); len(r) != 80 { // 79 kept + the ellipsis
		t.Fatalf("truncated to %d runes, want 80", len(r))
	}
}

func TestContinuationSlug(t *testing.T) {
	if got := continuationSlug("20260101T000000-demo.md"); got != "demo" {
		t.Fatalf("slug = %q", got)
	}
	if got := continuationSlug("20260702T215853-launcher-m5.md"); got != "launcher-m5" {
		t.Fatalf("multi-dash slug = %q", got)
	}
}

func TestRunContinueList(t *testing.T) {
	rt := newFakeRuntime()
	rt.continuationDir = "/repo/workshop/continuation"
	rt.continuationRows = []ContinuationRow{
		{Slug: "demo", Issues: "[#99]", Preview: strings.Repeat("X", 90) + "TAILMARKER"},
	}
	var out, errBuf bytes.Buffer
	if code := runContinueList(rt, &out, &errBuf); code != 0 {
		t.Fatalf("code=%d", code)
	}
	s := out.String()
	if !strings.Contains(s, "demo") || !strings.Contains(s, "[#99]") {
		t.Fatalf("list output = %q", s)
	}
	if !strings.Contains(s, "…") || strings.Contains(s, "TAILMARKER") {
		t.Fatalf("list must truncate a long NEXT ACTION line: %q", s)
	}
}

func TestRunContinueListEmpty(t *testing.T) {
	rt := newFakeRuntime()
	rt.continuationDir = "/repo/workshop/continuation"
	var out, errBuf bytes.Buffer
	if code := runContinueList(rt, &out, &errBuf); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if out.Len() != 0 || !strings.Contains(errBuf.String(), "no continuations") {
		t.Fatalf("empty: stdout=%q stderr=%q", out.String(), errBuf.String())
	}
}
