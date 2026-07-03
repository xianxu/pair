// Command pair-continuation is the deterministic writer for the `continuation`
// datatype (ariadne#91): given the gathered frontmatter fields + an approved
// body, it renders a conformant continuation file, allocates a collision-safe
// timestamped name under workshop/continuation/, writes it, and commits +
// pushes it to main — the disaster-recovery invariants that must not depend on
// the distilling LLM remembering. The xx-datatype dispatcher does the
// distillation (judgment); this binary does the mechanics.
package continuationcmd

import (
	"fmt"
	"strings"
	"time"
)

// Fields are the frontmatter inputs for a continuation instance.
// Keep the field set + order in sync with construct/datatype/continuation.md
// (ariadne#91 Frontmatter-shape table) — the golden-string test pins them.
type Fields struct {
	Slug       string
	Agent      string
	SessionID  string
	Created    time.Time
	Supersedes string
	Branch     string
	Worktree   string
	Issues     []string
}

// RenderFrontmatter emits the continuation frontmatter body (without the
// surrounding `---` fences). Field order tracks continuation.md's
// Frontmatter-shape table; empty optionals are omitted.
func RenderFrontmatter(f Fields) string {
	var b strings.Builder
	b.WriteString("type: continuation\n")
	fmt.Fprintf(&b, "slug: %s\n", f.Slug)
	fmt.Fprintf(&b, "agent: %s\n", f.Agent)
	if f.SessionID != "" {
		fmt.Fprintf(&b, "session_id: %s\n", f.SessionID)
	}
	fmt.Fprintf(&b, "created: %s\n", f.Created.Format("2006-01-02T15:04:05"))
	if f.Supersedes != "" {
		fmt.Fprintf(&b, "supersedes: %s\n", f.Supersedes)
	}
	if f.Branch != "" {
		fmt.Fprintf(&b, "branch: %s\n", f.Branch)
	}
	if f.Worktree != "" {
		fmt.Fprintf(&b, "worktree: %s\n", f.Worktree)
	}
	fmt.Fprintf(&b, "issues: [%s]\n", strings.Join(f.Issues, ", "))
	return b.String()
}

// AllocName returns the continuation filename for slug at ts —
// `<YYYYMMDDTHHMMSS>-<slug>.md` — collision-safe against existing names
// (an exact clash appends -1, -2, …; same-second parks usually differ by slug).
func AllocName(slug string, ts time.Time, existing []string) string {
	base := ts.Format("20060102T150405") + "-" + slug
	seen := make(map[string]bool, len(existing))
	for _, e := range existing {
		seen[e] = true
	}
	name := base + ".md"
	for n := 1; seen[name]; n++ {
		name = fmt.Sprintf("%s-%d.md", base, n)
	}
	return name
}

// Assemble joins frontmatter + body into a full continuation file:
// fenced frontmatter, a blank line, then the body with one trailing newline.
func Assemble(frontmatter, body string) string {
	return "---\n" + frontmatter + "---\n\n" + strings.TrimRight(body, "\n") + "\n"
}

// firstNextActionLine finds the first non-blank content line under the '## NEXT
// ACTION' heading. found=false if the heading is absent or its section is empty;
// isHeading=true if that first non-blank line is itself a markdown heading (an
// empty section). The single scan HasNextAction + NextActionPreview share.
func firstNextActionLine(body string) (line string, isHeading, found bool) {
	lines := strings.Split(body, "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) != "## NEXT ACTION" {
			continue
		}
		for _, rest := range lines[i+1:] {
			t := strings.TrimSpace(rest)
			if t == "" {
				continue // skip blank lines under the heading
			}
			return t, isATXHeading(t), true
		}
		return "", false, false // heading was the last non-blank line
	}
	return "", false, false
}

// HasNextAction reports whether body has a '## NEXT ACTION' heading followed by
// at least one non-blank content line (and not immediately another heading).
// The writer's structural guard: continuation.md makes NEXT ACTION mandatory,
// and a bare heading with no content is as useless as a missing one (#52).
func HasNextAction(body string) bool {
	_, isHeading, found := firstNextActionLine(body)
	return found && !isHeading
}

// NextActionPreview returns the first non-blank content line under the '## NEXT
// ACTION' heading (or "" if absent/empty/a nested heading) — the one-line summary
// the `pair continue` bare list shows per doc (#99 M5b). Shares HasNextAction's
// scan (firstNextActionLine) so the "where NEXT ACTION content lives" rule has one
// source (ARCH-DRY).
func NextActionPreview(body string) string {
	line, isHeading, found := firstNextActionLine(body)
	if !found || isHeading {
		return ""
	}
	return line
}

// isATXHeading reports whether a trimmed line is a markdown ATX heading: one or
// more '#' followed by a space. It deliberately does NOT match a bare '#NN'
// issue reference (no space after the hashes) — this repo writes those
// constantly, and "#52: do the thing" is valid NEXT ACTION content, not an empty
// section. (#52 review)
func isATXHeading(t string) bool {
	i := 0
	for i < len(t) && t[i] == '#' {
		i++
	}
	return i > 0 && i < len(t) && t[i] == ' '
}

// ValidateFields rejects a continuation missing its required fields.
func ValidateFields(f Fields) error {
	if strings.TrimSpace(f.Slug) == "" {
		return fmt.Errorf("slug is required")
	}
	if strings.TrimSpace(f.Agent) == "" {
		return fmt.Errorf("agent is required")
	}
	if len(f.Issues) == 0 {
		return fmt.Errorf("at least one issue is required")
	}
	return nil
}
