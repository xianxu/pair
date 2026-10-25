package sessionsync

import "strings"

type applyShadowStats struct {
	SpanVisits, EditVisits, CopiedBytes int
}

// applyShadow acknowledges only actual changes that map wholly through an
// unchanged source interval. Failure retains the entire previous baseline;
// coarse diff hunks and uncertain insertion boundaries grant no authority.
func applyShadow(shadow, base, next string) (string, bool, applyShadowStats) {
	var stats applyShadowStats
	edits := changedRuns(base, next)
	if len(edits) == 0 {
		return shadow, true, stats
	}
	unread := changedRuns(shadow, base)
	common := make([]changedRun, 0, len(unread)+1)
	shadowAt, baseAt := 0, 0
	for _, r := range unread {
		common = append(common, changedRun{shadowAt, r.OldStart, baseAt, r.NewStart})
		shadowAt, baseAt = r.OldEnd, r.NewEnd
	}
	common = append(common, changedRun{shadowAt, len(shadow), baseAt, len(base)})
	var out strings.Builder
	copyText := func(text string) { out.WriteString(text); stats.CopiedBytes += len(text) }
	index, copied := 0, 0
	for _, edit := range edits {
		stats.EditVisits++
		for index < len(common) && common[index].NewEnd <= edit.OldStart && common[index].NewEnd < len(base) {
			index++
			stats.SpanVisits++
		}
		if index == len(common) {
			return shadow, false, stats
		}
		span := common[index]
		stats.SpanVisits++
		if edit.OldStart < span.NewStart || edit.OldEnd > span.NewEnd {
			return shadow, false, stats
		}
		start := span.OldStart + edit.OldStart - span.NewStart
		end := start + edit.OldEnd - edit.OldStart
		if edit.OldStart == edit.OldEnd {
			// At a changed boundary left and right give different coordinates.
			left := edit.OldStart == span.NewStart && (span.NewStart != 0 || span.OldStart != 0)
			right := edit.OldEnd == span.NewEnd && (span.NewEnd != len(base) || span.OldEnd != len(shadow))
			if left || right {
				return shadow, false, stats
			}
		}
		if start < copied || end > len(shadow) || shadow[start:end] != base[edit.OldStart:edit.OldEnd] {
			return shadow, false, stats
		}
		copyText(shadow[copied:start])
		copyText(next[edit.NewStart:edit.NewEnd])
		copied = end
	}
	copyText(shadow[copied:])
	return out.String(), true, stats
}
