package continuationcmd

import "strings"

// StripStickyComments drops the draft's sticky-comment lines (those matching
// `===` after optional leading whitespace — the `=== label ===` stickies) and
// trims leading/trailing blank lines. The WIP that remains is what gets folded
// into a continuation's NEXT ACTION on compaction.
//
// This is a minimal Go mirror of the Lua strip_comments in nvim/init.lua (~L995,
// the `^%s*===` rule + edge-blank trim). It is PINNED to that source: if the
// sticky-comment convention there changes, change it here too. No cross-language
// drift test is built — pair has no Lua unit harness and `^\s*===` is a trivial,
// stable one-liner (see the issue's ## Revisions).
func StripStickyComments(s string) string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), "===") {
			continue
		}
		out = append(out, ln)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// FoldDraftIntoNextAction inserts wip into body's `## NEXT ACTION` section —
// after the section's existing content, before the next `## ` heading (or EOF) —
// under a one-line label, so the operator's parked draft survives the compaction
// restart (which otherwise overwrites the draft with a seed line). It is a no-op
// when wip is blank or body has no `## NEXT ACTION` heading.
//
// Inserting *after* existing content keeps NextActionPreview (the `pair continue`
// list summary, which reads the first content line) unaffected.
func FoldDraftIntoNextAction(body, wip string) string {
	wip = strings.TrimRight(wip, "\n")
	if strings.TrimSpace(wip) == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	start := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "## NEXT ACTION" {
			start = i
			break
		}
	}
	if start == -1 {
		return body
	}
	end := len(lines) // section runs to EOF unless a later heading closes it
	for i := start + 1; i < len(lines); i++ {
		if isATXHeading(strings.TrimSpace(lines[i])) {
			end = i
			break
		}
	}
	// Trim trailing blank lines inside the section so the insert reads cleanly.
	insert := end
	for insert > start+1 && strings.TrimSpace(lines[insert-1]) == "" {
		insert--
	}
	hasExisting := false
	for _, l := range lines[start+1 : insert] {
		if strings.TrimSpace(l) != "" {
			hasExisting = true
			break
		}
	}
	// A label distinguishes folded WIP from the agent's own NEXT ACTION content.
	// But when the section was empty, the WIP *is* the next action — emitting a
	// label there would make it the first content line, so NextActionPreview (the
	// `pair continue` list summary) would show the label instead of the WIP.
	var block []string
	if hasExisting {
		block = append([]string{"", "_Parked draft at compaction:_", ""}, strings.Split(wip, "\n")...)
	} else {
		block = append([]string{""}, strings.Split(wip, "\n")...)
	}
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:insert]...)
	out = append(out, block...)
	out = append(out, lines[insert:]...)
	return strings.Join(out, "\n")
}

// InCompactionContext reports whether the writer is running inside its own live
// pair pane — the gate for folding the draft + triggering the restart. It mirrors
// the tag-match half of the launcher's compactionDecision (compaction.go), which
// guards against a sibling pane's leaked ZELLIJ_SESSION_NAME.
//
// This is a proxy for "compaction was requested": the compaction prompt is the
// sole in-pane invoker of `pair continuation`, so in practice in-pane + tag-match
// means compaction. A deliberate manual in-pane write uses --no-restart to opt
// out of both the restart and the fold.
func InCompactionContext(pairTag, zellijSession string) bool {
	return pairTag != "" && zellijSession == "pair-"+pairTag
}
