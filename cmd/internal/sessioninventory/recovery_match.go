package sessioninventory

import (
	"sort"
	"strings"
	"time"
)

type RecoveryTailScore struct {
	Matched int
	Total   int
	Percent float64
	Excerpt string
}
type RankedRecoveryCandidate struct {
	Candidate     RecoveryCandidate
	Score         RecoveryTailScore
	ActivityDelta *time.Duration
	// No score can grant launch authority; selection always requires an operator.
	RequiresExplicitSelection bool
}

// ScoreRecoveryTail reports the best distinct-eight-word-window coverage among
// the last three complete assistant messages of at least sixteen words. It
// supplies evidence for display, not a calibrated probability or threshold.
func ScoreRecoveryTail(rendered []string, messages []RecoveryMessage) RecoveryTailScore {
	tty := recoveryWindows(recoveryWords(strings.Join(rendered, "\n")))
	var best RecoveryTailScore
	considered := 0
	for i := len(messages) - 1; i >= 0 && considered < 3; i-- {
		m := messages[i]
		if m.Role != EventAssistant || !m.Complete {
			continue
		}
		words := recoveryWords(m.Text)
		if len(words) < 16 {
			continue
		}
		considered++
		windows := recoveryWindows(words)
		score := RecoveryTailScore{Total: len(windows), Excerpt: strings.Join(words, " ")}
		// Excerpt is deliberately bounded even when the authored message is large.
		runes := []rune(score.Excerpt)
		if len(runes) > 160 {
			score.Excerpt = string(runes[:160]) + "…"
		}
		for word := range windows {
			if tty[word] {
				score.Matched++
			}
		}
		if score.Total > 0 {
			score.Percent = 100 * float64(score.Matched) / float64(score.Total)
		}
		if best.Total == 0 || scoreBetter(score, best) {
			best = score
		}
	}
	return best
}
func recoveryWords(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"• ", "● ", "- ", "* ", "> "} {
			line = strings.TrimPrefix(line, prefix)
		}
		// Preserve case and punctuation: only measured display decorations fold.
		line = strings.NewReplacer("**", "", "__", "", "`", "").Replace(line)
		lines = append(lines, line)
	}
	return strings.Fields(strings.Join(lines, " "))
}
func recoveryWindows(words []string) map[string]bool {
	out := make(map[string]bool)
	for i := 0; i+8 <= len(words); i++ {
		out[strings.Join(words[i:i+8], " ")] = true
	}
	return out
}
func scoreBetter(a, b RecoveryTailScore) bool {
	if a.Total == 0 {
		return false
	}
	if b.Total == 0 {
		return true
	}
	return int64(a.Matched)*int64(b.Total) > int64(b.Matched)*int64(a.Total)
}

// RankRecoveryCandidates filters native/source/cwd eligibility, coalesces roots,
// and orders evidence deterministically. Callers own physical cwd resolution,
// source chronology and capture/native generation revalidation before effects.
func RankRecoveryCandidates(candidates []RecoveryCandidate, rendered []string, targetCWD string, ttyActivity *time.Time) []RankedRecoveryCandidate {
	return rankRecoveryCandidates(candidates, rendered, targetCWD, ttyActivity, false)
}

func rankRecoveryCandidates(candidates []RecoveryCandidate, rendered []string, targetCWD string, ttyActivity *time.Time, nativeOnly bool) []RankedRecoveryCandidate {
	var out []RankedRecoveryCandidate
	for _, c := range candidates {
		if !c.RootEligible || c.Interactive.State != InteractiveYes || targetCWD == "" || c.CWD != targetCWD {
			continue
		}
		c.Messages = append([]RecoveryMessage(nil), c.Messages...)
		r := RankedRecoveryCandidate{Candidate: c, RequiresExplicitSelection: true}
		if !nativeOnly {
			r.Score = ScoreRecoveryTail(rendered, c.Messages)
		}
		if !nativeOnly && ttyActivity != nil && !c.SemanticLast.IsZero() {
			delta := c.SemanticLast.Sub(*ttyActivity)
			if delta < 0 {
				delta = -delta
			}
			r.ActivityDelta = &delta
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if nativeOnly && !a.Candidate.SemanticLast.Equal(b.Candidate.SemanticLast) {
			return a.Candidate.SemanticLast.After(b.Candidate.SemanticLast)
		}
		if scoreBetter(a.Score, b.Score) {
			return true
		}
		if scoreBetter(b.Score, a.Score) {
			return false
		}
		if a.ActivityDelta != nil && b.ActivityDelta == nil {
			return true
		}
		if a.ActivityDelta == nil && b.ActivityDelta != nil {
			return false
		}
		if a.ActivityDelta != nil && *a.ActivityDelta != *b.ActivityDelta {
			return *a.ActivityDelta < *b.ActivityDelta
		}
		if a.Candidate.Agent != b.Candidate.Agent {
			return a.Candidate.Agent < b.Candidate.Agent
		}
		if a.Candidate.NativeID != b.Candidate.NativeID {
			return a.Candidate.NativeID < b.Candidate.NativeID
		}
		return a.Candidate.Entry.Artifact.RelativePath < b.Candidate.Entry.Artifact.RelativePath
	})
	seen := map[string]bool{}
	result := out[:0]
	for _, r := range out {
		key := string(r.Candidate.Agent) + ":" + r.Candidate.NativeID
		if !seen[key] {
			seen[key] = true
			result = append(result, r)
		}
	}
	return result
}
