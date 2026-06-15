package main

import (
	"regexp"
	"strconv"
	"strings"
)

// promptGlyphChar is the per-agent prompt glyph — the SINGLE source for both the
// line-start regex (promptGlyphByAgent, derived below) and the empty-input-box
// detection (trimLiveTail). Sync-commented to nvim/scrollback.lua
// PROMPT_PATTERN_BY_AGENT. claude/codex are faithful; agy is a DELIBERATE
// SIMPLIFICATION of scrollback's box-aware variant (`\(──.*\n\)\zs>` — a `>` only
// after a `──` line): a bare `>` can over-match agy output, which now feeds the
// no-op gate as well as the lookback, so a false boundary can delay/add one
// distill — graceful (self-heals within ~1 press), never corrupts the log.
var promptGlyphChar = map[string]string{
	"claude": "❯",
	"codex":  "›",
	"agy":    ">",
}

// promptGlyphByAgent — line-start regex per agent, derived from promptGlyphChar
// so the two maps can't drift.
var promptGlyphByAgent = func() map[string]*regexp.Regexp {
	m := make(map[string]*regexp.Regexp, len(promptGlyphChar))
	for agent, ch := range promptGlyphChar {
		m[agent] = regexp.MustCompile("^" + regexp.QuoteMeta(ch))
	}
	return m
}()

func glyphFor(agent string) *regexp.Regexp {
	if re, ok := promptGlyphByAgent[agent]; ok {
		return re
	}
	return promptGlyphByAgent["claude"]
}

var (
	// ruleRe matches a horizontal-rule line (box-drawing / dashes only).
	ruleRe = regexp.MustCompile(`^[\s─━—=_·.\-]+$`)
	// thinkingRe matches claude's working spinner, e.g.
	// "* Cerebrating… (3s · thinking with xhigh effort)".
	thinkingRe = regexp.MustCompile(`^\* .*(…|\(\d+s)`)
	// contextMeterRe matches claude's context-usage footer line, e.g.
	// "100% context used" / "85% context left" — shown (right-aligned) when the
	// context window fills. A new footer variant: sitting as the LAST line it
	// stopped trimLiveTail dead, leaking the whole volatile footer into the
	// anchor → locate misses → FullRedistill / stale turn count (#58).
	contextMeterRe = regexp.MustCompile(`^\d+% context\b`)
)

// isFooterChrome reports whether line belongs to the live UI footer — none of
// which is committed scrollback (#58). The footer is multi-block when the agent
// is working: a thinking spinner + rule ABOVE the input box, then the box + rule
// + status below. Claude-shaped; other agents still get the generic blank / box
// / rule cases.
func isFooterChrome(line, glyph string) bool {
	t := strings.TrimSpace(line)
	switch {
	case t == "", t == glyph: // blank or empty input box
		return true
	case ruleRe.MatchString(line): // horizontal rule
		return true
	case thinkingRe.MatchString(t): // "* Cerebrating… (3s …)"
		return true
	case strings.HasPrefix(t, "⏵"): // "⏵⏵ bypass permissions …" status bar
		return true
	case strings.Contains(t, "esc to interrupt"):
		return true
	case contextMeterRe.MatchString(t): // "100% context used" context meter
		return true
	}
	return false
}

// trimLiveTail drops the live UI footer from the end of the cleaned text so the
// anchor/slice/turn-count work on stable committed scrollback — anchoring on the
// volatile footer is what made `locate` miss → FullRedistill every press (#58).
// It strips trailing chrome lines iteratively (handling the multi-block thinking
// footer), stopping at the first committed-content line. Pure.
func trimLiveTail(lines []string, agent string) []string {
	glyph := promptGlyphChar[agent]
	if glyph == "" {
		glyph = promptGlyphChar["claude"]
	}
	end := len(lines)
	for end > 0 && isFooterChrome(lines[end-1], glyph) {
		end--
	}
	return lines[:end]
}

// looksLikeChangelog reports whether s is a plausible change log: at least one
// `- ` bullet, and NO bare prose lines (every non-blank line must be a bullet, an
// indented bullet continuation, or a `#` header). This rejects a hijacked model
// that returned a conversation continuation — even one sprinkled with bullets,
// which a mere "has a bullet" check would wave through (#58).
func looksLikeChangelog(s string) bool {
	sawBullet := false
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "- "):
			sawBullet = true
		case strings.HasPrefix(line, "#"): // date header
		case strings.HasPrefix(line, " "), strings.HasPrefix(line, "\t"): // bullet continuation
		default:
			return false // a bare prose line → not a change log
		}
	}
	return sawBullet
}

// chunkLines splits lines into consecutive chunks of at most size, in order.
// Used to distill a long first-run transcript in batches (each per-call chunk
// bounded for the timeout) instead of truncating to the last `size` lines (#58).
func chunkLines(lines []string, size int) [][]string {
	if size <= 0 || len(lines) <= size {
		return [][]string{lines}
	}
	var out [][]string
	for i := 0; i < len(lines); i += size {
		end := i + size
		if end > len(lines) {
			end = len(lines)
		}
		out = append(out, lines[i:end])
	}
	return out
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

// headerDateRe matches a "## YYYY-MM-DD" day header. Change-log entries are dated
// by real change-time captured in the scrollback time events (#59).
var headerDateRe = regexp.MustCompile(`(?m)^## (\d{4}-\d{2}-\d{2})\s*$`)

// lastHeaderDate returns the date of the last "## YYYY-MM-DD" header in log, or
// "" if none — assemble's day-rollover check.
func lastHeaderDate(log string) string {
	m := headerDateRe.FindAllStringSubmatch(log, -1)
	if len(m) == 0 {
		return ""
	}
	return m[len(m)-1][1]
}

// tsMarkerRe matches a render-emitted day marker line (#59). MUST stay in sync
// with tsMarkerLine in cmd/pair-scrollback-render/main.go — the contract is
// pinned by the render→clean→distill e2e test e2e_test.go
// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
var tsMarkerRe = regexp.MustCompile(`^⟦pair:ts (\d{4}-\d{2}-\d{2})⟧$`)

// datedSegment is a run of consecutive content lines sharing one day ("" =
// undated, e.g. content captured before #59 was running). #59.
type datedSegment struct {
	date  string
	lines []string
}

// parseDatedLines strips the render's ⟦pair:ts DATE⟧ marker lines and returns the
// content lines plus a parallel `dates` slice: dates[i] is the day of the most
// recent preceding marker, "" before the first. The single point where markers
// leave the pipeline — everything downstream (anchor, turn-count, locate, slice,
// model input) sees clean content. len(content)==len(dates). Pure (#59).
func parseDatedLines(lines []string) (content, dates []string) {
	cur := ""
	for _, l := range lines {
		if m := tsMarkerRe.FindStringSubmatch(l); m != nil {
			cur = m[1]
			continue
		}
		content = append(content, l)
		dates = append(dates, cur)
	}
	return content, dates
}

// splitByDate groups content into consecutive same-date runs (oldest→newest), so
// a multi-day slice distills into multiple ## DATE sections. dates parallels
// content. Pure (#59).
func splitByDate(content, dates []string) []datedSegment {
	var segs []datedSegment
	for i, line := range content {
		d := ""
		if i < len(dates) {
			d = dates[i]
		}
		if len(segs) == 0 || segs[len(segs)-1].date != d {
			segs = append(segs, datedSegment{date: d})
		}
		segs[len(segs)-1].lines = append(segs[len(segs)-1].lines, line)
	}
	return segs
}

// ensureBlock normalizes a block to end in exactly one newline.
func ensureBlock(s string) string {
	return strings.TrimRight(s, "\n\t ") + "\n"
}

// assemble builds the new log: byte-verbatim frozen prefix + the revised last
// entry (ekPrime, "" on first-ever) + the model's new bullets (newEntries, "" if
// none). A "## <date>" header is inserted before newEntries iff there are new
// entries AND date != "" AND date != lastDate (the day rolled over vs the running
// log's last header). date=="" → no header (undated content, e.g. pre-#59
// capture). Dates are real change-time from the scrollback markers, not
// distill-time (the #58 fix, now sourced honestly). Pure.
func assemble(frozenPrefix, ekPrime, newEntries, date, lastDate string) string {
	var b strings.Builder
	b.WriteString(frozenPrefix)
	if ekPrime != "" {
		b.WriteString(ensureBlock(ekPrime))
	}
	if newEntries != "" {
		if date != "" && date != lastDate {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("## " + date + "\n\n")
		} else if b.Len() > 0 {
			b.WriteString("\n") // blank line between prior content and new bullets
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
