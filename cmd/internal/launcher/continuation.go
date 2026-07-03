package launcher

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// `pair continue` continuation-doc resolution (#99 M5b, shell 611-648). Bare
// `continue` lists the docs; `continue <slug>` resolves the newest matching doc
// (its path seeds the draft, its `agent:` frontmatter picks the port). The scan +
// glob are Runtime effects (ContinuationOps); the list formatting + frontmatter
// field extraction are pure.

// ContinuationRow is one bare-`continue` list entry.
type ContinuationRow struct {
	Slug    string
	Issues  string // the `issues:` frontmatter value (e.g. "[#99]")
	Preview string // raw first NEXT ACTION line (truncated at format time)
}

// frontmatterField extracts a `key: value` line's value from a doc body (shell's
// awk `-F': '` reads of `agent:` / `issues:`). Pure; the single reader both
// ResolveContinuationDoc (agent) and ScanContinuations (issues) share.
func frontmatterField(body, key string) string {
	for _, ln := range strings.Split(body, "\n") {
		if strings.HasPrefix(ln, key+":") {
			return strings.TrimSpace(strings.TrimPrefix(ln, key+":"))
		}
	}
	return ""
}

// truncatePreview caps a NEXT ACTION preview at 80 runes (shell 622: >80 → first
// 79 + …) so a long line doesn't flood the list row.
func truncatePreview(s string) string {
	if utf8.RuneCountInString(s) <= 80 {
		return s
	}
	return string([]rune(s)[:79]) + "…"
}

// formatContinuations renders the bare-`continue` list (shell 617-624). Pure.
func formatContinuations(rows []ContinuationRow, dir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "continuations in %s:\n", dir)
	for _, r := range rows {
		issues := r.Issues
		if issues == "" {
			issues = "[]"
		}
		fmt.Fprintf(&b, "  %-22s %-18s %s\n", r.Slug, issues, truncatePreview(r.Preview))
	}
	return b.String()
}

// runContinueList drives bare `pair continue` (shell 615-629): list the docs to
// stdout, or "no continuations" to stderr. Always exits 0.
func runContinueList(rt Runtime, stdout, stderr io.Writer) int {
	rows, dir := rt.ScanContinuations()
	if len(rows) == 0 {
		fmt.Fprintf(stderr, "pair: no continuations in %s\n", dir)
		return 0
	}
	fmt.Fprint(stdout, formatContinuations(rows, dir))
	return 0
}
