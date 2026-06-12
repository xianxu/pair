package main

import (
	"regexp"
	"strconv"
	"strings"
)

// promptGlyphByAgent — sync-commented to nvim/scrollback.lua PROMPT_PATTERN_BY_AGENT.
// claude/codex are faithful single-glyph ports. agy is a DELIBERATE
// SIMPLIFICATION: scrollback uses a box-aware, multi-line variant
// (`\(──.*\n\)\zs>` — a `>` only when the prior line starts with `──`); we
// approximate with `^>`. Over-matching is safe because turns drive ONLY the
// lookback — a false boundary just shortens context (graceful degradation, per
// the #53 spec), never breaks the anchor. A faithful port (also require
// lines[i-1] to start with `──`) is a future refinement if agy quality suffers.
var promptGlyphByAgent = map[string]*regexp.Regexp{
	"claude": regexp.MustCompile(`^❯`),
	"codex":  regexp.MustCompile(`^›`),
	"agy":    regexp.MustCompile(`^>`),
}

func glyphFor(agent string) *regexp.Regexp {
	if re, ok := promptGlyphByAgent[agent]; ok {
		return re
	}
	return promptGlyphByAgent["claude"]
}

// scanTurnBoundaries returns the indices of lines that begin a user turn (the
// per-agent prompt glyph at line start). Pure.
func scanTurnBoundaries(lines []string, agent string) []int {
	re := glyphFor(agent)
	var out []int
	for i, l := range lines {
		if re.MatchString(l) {
			out = append(out, i)
		}
	}
	return out
}

// LocateKind is the outcome of an incremental-boundary decision.
type LocateKind int

const (
	// Found — the anchor was located; Start is the slice start.
	Found LocateKind = iota
	// NoOp — the anchor sits flush with the end; nothing new to distill.
	NoOp
	// FullRedistill — the anchor is absent (or empty); distill from Start=0.
	FullRedistill
)

// LocateResult carries the outcome. Start is the slice start index; the slice
// end is always len(lines). NoOp carries no slice.
type LocateResult struct {
	Kind  LocateKind
	Start int
}

// locate finds the newest exact occurrence of the anchor block (consecutive
// lines) and decides the slice start. Pure and total:
//   - empty anchor → FullRedistill from 0
//   - anchor not found → FullRedistill from 0
//   - anchor flush with the end (no lines after it) → NoOp
//   - else Found, with Start = walk back from the match past lookbackTurns turn
//     boundaries; if fewer than lookbackTurns boundaries exist before the match,
//     walk to the start (0); then clamp so (match - Start) ≤ lineCap.
func locate(lines, anchor []string, turnBoundaries []int, lookbackTurns, lineCap int) LocateResult {
	if len(anchor) == 0 {
		return LocateResult{Kind: FullRedistill, Start: 0}
	}
	matchAt := -1
	for i := len(lines) - len(anchor); i >= 0; i-- { // newest-first
		if linesEqual(lines[i:i+len(anchor)], anchor) {
			matchAt = i
			break
		}
	}
	if matchAt < 0 {
		return LocateResult{Kind: FullRedistill, Start: 0}
	}
	if matchAt+len(anchor) >= len(lines) {
		return LocateResult{Kind: NoOp}
	}
	start := matchAt
	count := 0
	for i := len(turnBoundaries) - 1; i >= 0 && count < lookbackTurns; i-- {
		if turnBoundaries[i] < matchAt {
			start = turnBoundaries[i]
			count++
		}
	}
	if count < lookbackTurns {
		start = 0 // fewer than lookbackTurns turns available → take all context
	}
	if matchAt-start > lineCap {
		start = matchAt - lineCap
	}
	return LocateResult{Kind: Found, Start: start}
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// lastBulletStart returns the byte offset of the last line beginning with "- ",
// or -1 if there is none.
func lastBulletStart(log string) int {
	best := -1
	if strings.HasPrefix(log, "- ") {
		best = 0
	}
	if i := strings.LastIndex(log, "\n- "); i >= 0 && i+1 > best {
		best = i + 1
	}
	return best
}

// splitFrozenTail splits the prior log into the byte-verbatim frozen prefix
// (everything up to, not including, the last bullet block) and the last bullet
// block Ek (incl. its indented continuations). frozen+ek == log. Pure.
func splitFrozenTail(log string) (frozen, lastEntry string) {
	idx := lastBulletStart(log)
	if idx < 0 {
		return log, ""
	}
	return log[:idx], log[idx:]
}

// splitFirstEntry splits model output into its first bullet block (Ek') and the
// remaining new entries. The first block is returned trimmed of trailing
// whitespace; the rest is returned as-is (assemble normalizes both). Pure.
func splitFirstEntry(s string) (first, rest string) {
	s = strings.TrimLeft(s, "\n")
	if !strings.HasPrefix(s, "- ") {
		return strings.TrimRight(s, "\n\t "), ""
	}
	if i := strings.Index(s[1:], "\n- "); i >= 0 {
		split := 1 + i + 1 // position of the next bullet's '-'
		return strings.TrimRight(s[:split], "\n\t "), s[split:]
	}
	return strings.TrimRight(s, "\n\t "), ""
}

var headerDateRe = regexp.MustCompile(`(?m)^## (\d{4}-\d{2}-\d{2})\s*$`)

// lastHeaderDate returns the date of the last "## YYYY-MM-DD" header, or "".
func lastHeaderDate(log string) string {
	m := headerDateRe.FindAllStringSubmatch(log, -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

// ensureBlock normalizes a block to end in exactly one newline.
func ensureBlock(s string) string {
	return strings.TrimRight(s, "\n\t ") + "\n"
}

// assemble builds the new log. frozenPrefix is byte-verbatim. ekPrime is the
// revised last entry ("" on first-ever). newEntries is the model's new bullets
// ("" if none). A "## today" header is inserted before newEntries iff there are
// new entries AND today != lastDate. Invariant: a "## date" header is only ever
// emitted immediately before ≥1 bullet, so splitFrozenTail's "last bullet
// block" stays well-defined for every reachable state. Pure.
func assemble(frozenPrefix, ekPrime, newEntries, today, lastDate string) string {
	var b strings.Builder
	b.WriteString(frozenPrefix)
	if ekPrime != "" {
		b.WriteString(ensureBlock(ekPrime))
	}
	if newEntries != "" {
		if today != lastDate {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("## " + today + "\n\n")
		} else if ekPrime != "" {
			b.WriteString("\n")
		}
		b.WriteString(ensureBlock(newEntries))
	}
	return b.String()
}

// parseAnchor reads the anchor sidecar: an optional "turns:<N>" header line (the
// completed-turn count at the last distill, for the no-op check) followed by the
// verbatim K-line content snippet. Tolerates a header-less (legacy) file and a
// malformed count (falls back to turns=0, whole content as snippet). Pure.
func parseAnchor(content string) (turns int, snippet []string) {
	ls := splitLines(content)
	if len(ls) == 0 {
		return 0, nil
	}
	if rest, ok := strings.CutPrefix(ls[0], "turns:"); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil {
			return n, ls[1:]
		}
	}
	return 0, ls
}

// anchorSnippet returns the last k lines of the cleaned text (the next anchor).
func anchorSnippet(lines []string, k int) []string {
	if len(lines) <= k {
		return append([]string(nil), lines...)
	}
	return append([]string(nil), lines[len(lines)-k:]...)
}
