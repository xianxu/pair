#!/usr/bin/env bash
# A plan must not keep asserting a fact its own Revisions says was superseded.
#
# WHY THIS IS A TEST AND NOT A RULE IN THE PLAN. pair#199 wrote the rule
# ("correct the class, not the site"), then failed its own acceptance grep in
# three consecutive review rounds — each time the Revisions said "All swept."
# while superseded prose sat live in the same file. A sweep that depends on
# remembering to sweep is the same defect as a consumer set that depends on
# remembering to update it. This runs the grep.
#
# The list is per-plan and deliberately hand-written: what counts as superseded
# is a judgement the plan's author makes when a finding lands. What is NOT
# hand-maintained is the CHECKING.
#
# Adding a pair: when a review supersedes a fact, add the dead token here in the
# same commit that corrects the prose. A `# note:` line says what replaced it,
# so a reader hitting a failure knows the correction rather than just the ban.
#
# Run: bash tests/plan-superseded-facts-test.sh   (wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fails=0
bad() { echo "  FAIL $*"; fails=$((fails + 1)); }

# check FILE TOKEN REPLACEMENT [MAX_LINE]
#   MAX_LINE bounds the check to the body, so a Revisions entry may quote the
#   dead token while narrating its own correction — which is the one place it
#   legitimately appears.
check() {
	file="$1" token="$2" replacement="$3" max="${4:-0}"
	path="$ROOT/$file"
	[ -f "$path" ] || { bad "$file does not exist"; return; }
	hits=$(grep -n -- "$token" "$path" || true)
	[ -n "$hits" ] || return
	while IFS= read -r hit; do
		line=${hit%%:*}
		if [ "$max" -gt 0 ] && [ "$line" -ge "$max" ]; then continue; fi
		bad "$file:$line still asserts superseded '$token' — use: $replacement"
	done <<< "$hits"
}

echo "plan-superseded-facts-test:"

PLAN="workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md"
# The Revisions section narrates these corrections and may quote them.
REV=$(grep -n "^## Revisions" "$ROOT/$PLAN" | cut -d: -f1)
REV=${REV:-0}

# pair#199 BR-16: the probe is tracked, not in a scratchpad.
check "$PLAN" 'scratchpad/199-probe' 'probes/zellijscrollregion' "$REV"
# pair#199 BR-21: probes live at probes/, which make test-smoke runs wholesale.
check "$PLAN" 'cmd/probes/zellijscrollregion' 'probes/zellijscrollregion' "$REV"
# pair#199 PQ-7: couchtty's sanitize/truncate are unexported; rowtext is the home.
check "$PLAN" 'same helpers `couchtty` uses' 'rowtext.Sanitize / rowtext.Fit' "$REV"
# pair#199 finding 7: Child.TakeRowDirty is drained by the Sink; read batch.RowDirty.
check "$PLAN" 'repaint on tab change, resize, and `TakeRowDirty`' 'batch.RowDirty read in the Sink' "$REV"
# pair#199 finding 9: the rename-pane consumers are derived, not run.go:229.
check "$PLAN" 'consumers at$' 'the derived set in finding 9' "$REV"

# pair#199 BR-28: M2 fixed the subprocess writers at the RUNTIME, not by routing
# termcmd's call sites through Quiet -- the first form of the step, which the
# plan kept describing after the code stopped doing it.
#
# NB: a token that never appeared in any revision of the file is worse than no
# entry -- it reads as coverage while being unable to fire. One such entry
# ('route only stdout through') was removed after `git log -S` found it in no
# revision. Add a pair only for prose the plan ACTUALLY carried.
check "$PLAN" 'Route every `RunZellijAction` call in `termcmd`' 'make the Runtime incapable' "$REV"

# pair#199: rowtext was extracted in M2 (the diagnostic path needed it), not M3,
# and couchtty's unexported originals are gone rather than merely unreachable.
check "$PLAN" 'the shared package M3 extracts' 'extracted in M2' "$REV"

check "probes/zellijscrollregion/main.go" 'cmd/probes/zellijscrollregion' 'probes/zellijscrollregion'

if [ "$fails" -ne 0 ]; then
	echo "plan-superseded-facts-test: $fails failure(s)"
	exit 1
fi
echo "plan-superseded-facts-test: all passed"
