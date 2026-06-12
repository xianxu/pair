package main

import "strings"

// entryGuidance is the shared definition of what belongs in the change log —
// the category vocabulary lives here (in the prompt), not in the output format.
const entryGuidance = `You maintain a concise CHANGE LOG of a pair-programming session: the notable
milestones and decisions a human operator would want to glance at, distilled
from the terminal transcript (what the operator actually saw).

A change-log entry is one of: a milestone started or finished ("M1 started",
"M1 done"); a design or product decision ("decided to distill the TTY, not the
transcript"); a significant change (a feature shipped, a bug root-caused or
fixed); a blocker hit; or a scope shift. Routine back-and-forth, tool noise,
and incidental steps are NOT entries.

Output rules:
- Each entry is a Markdown bullet starting with "- ", 1 to 2 sentences, concise
  enough to glance at.
- Separate entries with a single blank line.
- Reference tickets (#NN), milestones (Mx), branch names, and file paths inline
  where relevant.
- Output ONLY the bullets. No date headers, no timestamps, no preamble, no
  trailing commentary.`

// buildSystemPrompt returns the instructions for a first-ever run (summarize the
// whole transcript) or an incremental run (revise the last entry + append).
func buildSystemPrompt(firstRun bool) string {
	if firstRun {
		return entryGuidance + "\n\n" +
			`You are given the full terminal transcript of the session so far.
Summarize it into change-log bullets in chronological order (most recent last).`
	}
	return entryGuidance + "\n\n" +
		`You are given: (1) the entries ALREADY LOGGED (read-only — do not repeat
them), (2) the current tentative LAST entry, and (3) the NEW terminal activity
since.

Produce, in order:
1. The LAST entry — possibly revised if the new activity clarifies what it was
   (e.g. "investigating X" → "fixed X"). Emit it as exactly ONE bullet. Never
   drop it, even if unchanged.
2. Then zero or more NEW bullets for any further notable milestones/decisions in
   the new activity. If nothing new is notable, output only the last entry.`
}

// buildInput assembles the model's user content. On a first run it is just the
// cleaned transcript; otherwise it sections the read-only memory, the revisable
// last entry, and the new activity. Pure.
func buildInput(frozen, ek, slice string, firstRun bool) string {
	if firstRun {
		return slice
	}
	var b strings.Builder
	b.WriteString("=== ALREADY LOGGED (read-only, do not repeat) ===\n")
	b.WriteString(strings.TrimRight(frozen, "\n"))
	b.WriteString("\n\n=== CURRENT LAST ENTRY (you may revise this one) ===\n")
	b.WriteString(strings.TrimRight(ek, "\n"))
	b.WriteString("\n\n=== NEW TERMINAL ACTIVITY ===\n")
	b.WriteString(slice)
	return b.String()
}
