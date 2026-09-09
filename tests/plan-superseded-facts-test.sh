#!/usr/bin/env bash
# A plan OR ITS ISSUE must not keep asserting a fact its own Revisions superseded.
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
# BOTH ARTIFACTS, not just the plan. The close gate checks `--verified` against
# the ISSUE's `## Done when`, so a retired deliverable left standing there is the
# more expensive half — and this script was bounded to the plan for three rounds
# while exactly that drifted (pair#199 BR-67).
#
# Run: bash tests/plan-superseded-facts-test.sh   (wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fails=0
bad() { echo "  FAIL $*"; fails=$((fails + 1)); }

# resolve_plan NAME -> a repo-relative path, ACTIVE OR ARCHIVED.
#
# `sdlc close` MOVES plans to workshop/history/plans/, so a hardcoded
# workshop/plans path asserts the issue will never close -- and this script,
# wired into `make test`, would then fail the whole repo at that close
# (pair#199 BR-61, third instance of the class: the round that stated the rule
# swept the two Go guards and never ran `grep -rn workshop/plans`).
resolve_plan() {
	name="$1"
	if [ -f "$ROOT/workshop/plans/$name" ]; then
		echo "workshop/plans/$name"
		return
	fi
	found=$(cd "$ROOT" && find workshop/history -name "$name" -type f 2>/dev/null | head -1)
	echo "$found"
}

# resolve_issue is the same rule for the other artifact this script reads.
resolve_issue() {
	name="$1"
	if [ -f "$ROOT/workshop/issues/$name" ]; then
		echo "workshop/issues/$name"
		return
	fi
	found=$(cd "$ROOT" && find workshop/history -name "$name" -type f 2>/dev/null | head -1)
	echo "$found"
}

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

PLAN="$(resolve_plan 000199-pair-term-own-the-right-pane-tab-bar-plan.md)"
if [ -z "$PLAN" ]; then
	bad "000199 plan is in neither workshop/plans nor workshop/history"
	PLAN="workshop/plans/000199-pair-term-own-the-right-pane-tab-bar-plan.md"
fi
# The Revisions section narrates these corrections and may quote them.
REV=$(grep -n "^## Revisions" "$ROOT/$PLAN" 2>/dev/null | cut -d: -f1)
REV=${REV:-0}

ISSUE="$(resolve_issue 000199-pair-term-own-the-right-pane-tab-bar.md)"
if [ -z "$ISSUE" ]; then
	bad "000199 issue is in neither workshop/issues nor workshop/history"
	ISSUE="workshop/issues/000199-pair-term-own-the-right-pane-tab-bar.md"
fi
IREV=$(grep -n "^## Revisions" "$ROOT/$ISSUE" 2>/dev/null | cut -d: -f1)
IREV=${IREV:-0}

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

# pair#199 M4 took the frame off the layout-3 terminal. Four statements
# asserting it is framed survived the commit that falsified them, two of them in
# files M4 itself edited -- the same class, one milestone later. These are the
# ATLAS's copies; config.kdl was corrected in M4.
# THE ENUMERATION (BR-33's rule): this file list IS the set of artifact classes
# that assert the design, and it must name EVERY class, not every instance.
# Five classes exist in this repo and all five are represented below:
#
#   plan            $PLAN
#   issue           $ISSUE
#   atlas           atlas/*.md     <- a GLOB: the class is every atlas file (BR-84)
#   config/layout   $CONFIG        <- added after M4 falsified it and nothing objected
#   code comments   run.go, probes/*/main.go
#
# A pair registered for one class does not defend the others; that is how M4's
# frame claims survived in the atlas while config.kdl was corrected in the same
# commit.
# check_atlas TOKEN REPLACEMENT [MAX_LINE] -- the atlas is a CLASS, not a file.
#
# BR-84: this sat as `ATLAS="atlas/architecture.md"` directly beneath a comment
# declaring "atlas" an artifact class, so a superseded fact in any other atlas
# file was invisible. It was measured: atlas/couch.md described couch's paint
# gate as deferring on mid-sequence alone, falsified by this window's own change
# to that gate, and no guard could see it. The scope is now the glob.
check_atlas() {
	found=0
	for atlas_file in "$ROOT"/atlas/*.md; do
		[ -f "$atlas_file" ] || continue
		found=$((found + 1))
		check "${atlas_file#$ROOT/}" "$1" "$2" "${3:-0}"
	done
	if [ "$found" -lt 2 ]; then
		bad "atlas glob matched $found file(s); the atlas class went blind"
	fi
}
CONFIG="zellij/config.kdl"
LAYOUT="zellij/layouts/main-3.kdl"
# The SIXTH class, and the one that bites hardest: a TEST can encode the design
# too, and a guard asserting the old truth outranks prose because `make test`
# enforces it. tests/term-pane-shortcuts-test.sh carried "split panes keep
# zellij default frames" through M4.
GUARD="tests/term-pane-shortcuts-test.sh"
check_atlas 'agent pane and layout-3 terminal.*render frames' 'only the agent pane is framed'
# note: the gate asks TWO questions now -- mid-sequence AND the child holding the
# cursor-save slot outside the alt screen. Found only once the atlas class became
# a glob; atlas/couch.md had carried the one-question claim since this window
# changed the gate.
check_atlas 'defers while the CHILD.s stream is mid-sequence' 'SafeToPaint asks mid-sequence AND the cursor-save slot'
# note: HoldsCursorSave reports the raw bit; SafeToPaint is the decision. Gating
# on the raw bit freezes the row for a full-screen child's whole session (BR-79).
check_atlas 'HoldsCursorSave. is the$' 'SafeToPaint is the predicate'
check_atlas 'The draft pane opts out via `borderless=true` in both' 'TWO panes opt out'

# The config/layout class. This is the pre-M4 sentence, verbatim: it claimed the
# DRAFT pane was the only opt-out, which stopped being true when the terminal
# went borderless at all nine rungs.
check "$CONFIG" 'scroll offset to plugins or the CLI. The draft pane opts out via' 'TWO panes opt out'
# BR-72's residual: three more sites asserted the terminal is framed, and the
# token registered above covered a DIFFERENT sentence in the same file. One
# token per file is not one token per claim.
check "$CONFIG" 'keep their frames' 'they are borderless since M4'
check "$LAYOUT" 'while keeping frames' 'the terminal panes are borderless'
check_atlas 'drag-immune while keeping frames and full mouse support' 'keeping the agent pane s frame'
check_atlas 'frames stay by zellij default' 'borderless since M4'
check_atlas 'frame-title rename editor' 'the field renders in the strip'
check "$GUARD" 'keep zellij default frames' 'the layout declares borderless, not the call site'
# BR-33's own paragraph: diagnostics stopped sharing the coalescing slot when
# they got owedDiag, and the paragraph that described one slot for every
# console write outlived that by two commits.
check_atlas 'deferred into a single \*\*coalescing\*\* slot' 'a PAINT coalesces; diagnostics queue'

check "probes/zellijscrollregion/main.go" 'cmd/probes/zellijscrollregion' 'probes/zellijscrollregion'

# ---------------------------------------------------------------- the ISSUE
#
# The close gate reads `## Done when`, so a retired deliverable standing there is
# the expensive half of this family (pair#199 BR-67). These pairs are the
# 2026-09-06 Revisions entry, checked instead of trusted.

# The scroll-position deliverable was STRUCK: ptychild carries no scroll offset,
# and the frameless follow-on it was meant to justify is M4 in this issue.
check "$ISSUE" 'displays scroll position for its own pane' 'struck 2026-09-06 — see ## Revisions' "$IREV"
# Its replacement -- borderless -- moved INTO this issue as M4, so the Spec must
# not still call it a follow-on.
check "$ISSUE" 'going borderless is a \*\*follow-on\*\*' 'M4 of this issue' "$IREV"

# ------------------------------------------------------------------ the CODE
#
# Same family, same window: a comment describing a milestone in the future tense
# after that milestone has landed. BR-67 names this instance explicitly.
check "cmd/internal/termcmd/run.go" 'becomes a writer in M3' 'writes the row on every resize (M3, tested)'

if [ "$fails" -ne 0 ]; then
	echo "plan-superseded-facts-test: $fails failure(s)"
	exit 1
fi
echo "plan-superseded-facts-test: all passed"
