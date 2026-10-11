package couchcmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// A waiting notice (pair#451) tells the hub that a slot has sat waiting for
// someone past a threshold. Pair renders and never classifies (tail.go); the
// judgement is Couch's, as the tail's reader.

// waitFacts is what Couch already holds in memory about one slot.
type waitFacts struct {
	Quiet   time.Duration // since the wrapper's last output or keystroke
	Settled *bool         // the wrapper's #421 self-report; nil is unknown
	Resting bool          // on its resting branch
	Pending bool          // a message to it is in flight
}

type waitReason string

const (
	waitPrompt   waitReason = "on a prompt"
	waitQuestion waitReason = "after asking"
	waitIdle     waitReason = "idle off its resting branch"
)

// needsTail reports whether the slot can be waiting at all, before any IO:
// quiet past the threshold, with a Settled claim to judge by.
func needsTail(f waitFacts, threshold time.Duration) bool {
	return f.Quiet >= threshold && f.Settled != nil
}

// classifyWait is the one definition of "waiting for someone". An unsettled
// slot that has been byte-quiet that long has something on screen asking for
// input (a dialog, a draft, a stuck orientation); a settled one waits if its
// turn ended with a question, or if it stopped off its resting branch with
// nothing in flight to it.
func classifyWait(f waitFacts, threshold time.Duration, tail []string) (waitReason, string, bool) {
	if !needsTail(f, threshold) {
		return "", "", false
	}
	prose := lastProseLine(tail)
	switch {
	case !*f.Settled:
		if title := dialogTitle(tail); title != "" {
			return waitPrompt, title, true
		}
		return waitPrompt, prose, true
	case isQuestion(prose):
		return waitQuestion, prose, true
	case !f.Resting && !f.Pending:
		return waitIdle, prose, true
	}
	return "", "", false
}

var (
	tailMarkup  = regexp.MustCompile(`‹(/?dim|/?rev|cursor)›`)
	dimRun      = regexp.MustCompile(`‹dim›.*?(‹/dim›|$)`)
	optionLine  = regexp.MustCompile(`^[❯›>]?\s*1\.\s`)
	separatorRe = regexp.MustCompile(`^[─━\-═]+$`)
)

// stripTailMarkup renders one tail line (tail.go's grammar) as plain text,
// without a dialog box's side borders.
func stripTailMarkup(line string) string {
	const escaped = "\x00"
	line = strings.ReplaceAll(line, "‹‹", escaped)
	line = tailMarkup.ReplaceAllString(line, "")
	return strings.Trim(strings.ReplaceAll(line, escaped, "‹"), " \t│┃")
}

// dimOnly reports a line whose only text is faint: agent chrome.
func dimOnly(line string) bool {
	return stripTailMarkup(dimRun.ReplaceAllString(line, "")) == ""
}

// dialogTitle is the line above a dialog's first option, or "" without one.
func dialogTitle(tail []string) string {
	for i, raw := range tail {
		if !optionLine.MatchString(stripTailMarkup(raw)) {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if t := stripTailMarkup(tail[j]); t != "" {
				return t
			}
		}
	}
	return ""
}

// lastProseLine is the agent's last line of prose above its composer: chrome
// paragraphs (a turn summary ✻, a recap ※, all-faint runs, separators) are
// skipped.
func lastProseLine(tail []string) string {
	end := len(tail)
	for i := len(tail) - 1; i >= 0; i-- {
		if t := stripTailMarkup(tail[i]); strings.HasPrefix(t, "❯") || strings.HasPrefix(t, "›") {
			end = i
			break
		}
	}
	var last string
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		first := stripTailMarkup(para[0])
		chrome := strings.HasPrefix(first, "✻") || strings.HasPrefix(first, "※")
		allDim := true
		for _, l := range para {
			allDim = allDim && dimOnly(l)
		}
		if !chrome && !allDim {
			last = stripTailMarkup(para[len(para)-1])
		}
		para = nil
	}
	for _, raw := range tail[:end] {
		t := stripTailMarkup(raw)
		if t == "" || separatorRe.MatchString(t) {
			flush()
			continue
		}
		para = append(para, raw)
	}
	flush()
	return last
}

func isQuestion(line string) bool {
	return strings.HasSuffix(strings.TrimRight(line, " *_`)\"'"), "?")
}

// maxWaitSummary bounds the quoted screen line in a notice.
const maxWaitSummary = 200

// waitNoticeBody puts the slot and reason first: a truncated send keeps its
// head (pair#211).
func waitNoticeBody(slot string, quiet time.Duration, reason waitReason, summary, optOutPath string) string {
	if len(summary) > maxWaitSummary {
		cut := maxWaitSummary
		for cut > 0 && !utf8RuneStart(summary[cut]) {
			cut--
		}
		summary = summary[:cut] + "…"
	}
	if summary == "" {
		summary = "(no text on screen)"
	}
	return fmt.Sprintf("slot %s waiting %d min %s: %s (couch --peek %s; mute: add %s to %s)",
		slot, int(quiet/time.Minute), reason, summary, slot, slot, optOutPath)
}

func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

// parseWaitOptOut reads the operator's mute list: one slot per line, # comments.
func parseWaitOptOut(text string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if line = strings.TrimSpace(line); line != "" {
			out[line] = true
		}
	}
	return out
}
