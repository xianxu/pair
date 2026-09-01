---
id: 000162
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Enter sends when the composer line is addressed to the harness

## Problem

In the **agent pane** — the Claude/Codex tty itself, not the nvim draft —
sending `/login` costs Alt+Enter. So does `/clear`, `/model opus`, `/context`,
`/resume`. Natively, every one of these agents submits on plain Enter; Pair is
what took that away.

That was deliberate. `cmd/internal/wrapcmd/wrap.go:127-143` documents it: the
draft uses Enter = newline / Alt+Enter = send, the agent's native TUI uses
Enter = send, "that mismatch is jarring when the user moves between panes", so
pair-wrap rewrites stdin to give the agent the inverted mapping too
(`sendKeymap`, `PAIR_WRAP_REMAP_RETURN`). For prose the inversion is right and
should stay. For a slash command it is pure cost: it charges a modifier for the
shortest, most frequent input, and it is *subtracting* behavior the agent
shipped with.

The path is already traced end to end:

- `decidePlainReturn` (`cmd/internal/wrapcmd/harness_tty.go:90`) is the single
  place a plain Enter is resolved.
- Its first branch is the overlay bypass, which does **not** cover this. Claude's
  detector keys on OSC 777 with body `notify;Claude Code;Claude needs your
  permission` (`wrap.go:650-688`) — a permission prompt. The slash-command
  dropdown is not that, so `overlayActive` stays false.
- So the `composerGatePositive` branch runs, `claudeComposerActive` correctly
  reports the cursor is inside the composer's ruled box, and the decision is
  `remap()` → `plainCR` = `\<CR>` → **a newline in the composer.**

Nothing is malfunctioning. The gate answers the only question it is currently
able to ask — *is a composer active?* — and for a slash command the right
question is a different one.

## Spec

### The rule

**Enter sends when the line is addressed to the harness. Enter inserts a
newline when the text is addressed to the model.**

`/` is only its first instance. Prose is addressed to the model and is
inherently *growable* — a second line is always a meaningful thing to want, so
Enter must stay a newline there. A sigil-led line is addressed to the
*harness*, whose grammar is line-terminated by construction: there is no
two-line `/login`. The binding follows the addressee, not a hardcoded
character. That framing also settles what does **not** qualify — `@file.go` is
a reference *inside* prose, addressed to the model, so it keeps today's
mapping.

Stated against the existing code, the rule is one more branch in
`decidePlainReturn`'s `composerGatePositive` arm: composer active **and** its
content harness-addressed → emit bare `\r` (submit), `adapt.Bypass`, its own
`reason` string so `pair-doctor` can see it fire.

### The predicate

`harnessAddressed(line, harness)` is true when all three hold:

1. The composer holds exactly one line.
2. Its first character is a **line-committed sigil** for this harness.
3. The line matches that sigil's token grammar — for `/`:
   `^/[A-Za-z][A-Za-z0-9_-]*(\s.*)?$`.

Rule 3 is why the predicate is not just "starts with `/`", and it is the part
that has to be right. `/Users/xianxu/workspace/brain — look at this repo` is an
ordinary opening line of a prompt in this workflow and it begins with a slash;
an inner `/` in the first token disqualifies it, as does a space straight after
the sigil.

Sigils are per-harness data, not a constant: `!` (bash) and `#` (memory) are
Claude-specific, Codex has only `/`. They belong in `harnessTTYProfiles`
alongside `keymap`, `overlay` and `recognize` — the table that already exists
for exactly this kind of per-harness fact — so bringing up a new harness
declares its sigils where it declares everything else
(`atlas/how-to-bring-up-a-new-harness-cli.md`).

### The structural change

`composerRecognizer` is `func(terminalSnapshot) bool`. To ask the second
question the gate needs the composer's *text*, not just its existence.

The geometry is already computed and thrown away.
`ruledBoxComposerActive` (`composer_recognizers.go:124`) walks up to find
`promptY`, calls `ruledBoxBottomRule` for the closing rule, and carries
`spec.minCursorX` as the first text column — the exact region of the composer —
then returns `true`. `codexComposerActive` and `agyComposerActive` do the same.
So the change is to return that located region instead of discarding it, and
let the caller read cells from it. No new terminal parsing, no second source of
composer geometry (ARCH-DRY).

### Why this is not the failure mode the recognizer was hardened against

`claudeComposerActive`'s own comment is explicit that false negatives are the
expensive direction: a decline means the next Return submits a half-written
draft, and `claudeComposerMaxRows` is unbounded precisely so a tall draft can
never be mis-submitted. This issue makes Return submit *on purpose*, so it has
to say why that is a different thing.

It is strictly narrower, and gated the opposite way. The dangerous case is a
composer that is **not** recognized — an unknown state where arbitrary
multi-line prose gets committed. This rule fires only when the composer **is**
positively recognized, holds exactly one line, and that line matches a command
grammar. Every direction of failure degrades to today's behavior, and the worst
misfire sends one command-shaped line, which the agent answers with "unknown
command". The lost-draft risk is not reachable from here: a multi-line draft
fails predicate rule 1 before anything else is consulted.

### Invariants

- **Alt+Enter always submits, in every mode.** The rule is strictly *additive*:
  Enter gains a meaning, nothing loses one. Muscle memory never breaks.
- **Predicate false → today's bytes, exactly.** Overlay bypass keeps first
  claim; `composerGateLegacy`, `composerGateUnknown`, an absent profile and an
  empty `plainCR` all keep their current fail-closed paths untouched.
- **A targeted escape hatch.** `PAIR_WRAP_REMAP_RETURN=0` already turns the
  whole inversion off; add `PAIR_WRAP_COMMAND_ENTER=0` for this rule alone, so
  switching it off does not cost the inversion.
- **The new branch is instrumented** like every other branch in
  `decidePlainReturn` — its own `adapt` outcome and reason, visible to
  `pair-doctor` (`doctor/SKILL.md`, bring-up guide §3).

### The draft pane

Secondary, and it follows for free. `nvim/init.lua:3565` routes insert-mode
`<CR>` through `cr_keys`, whose contract is that `<CR>` always breaks the line
when no completion is selected; the same predicate becomes a prior branch
there, with the completion popup keeping first claim. Worth doing in the same
pass — the two panes are supposed to agree, and that agreement is the entire
justification for the inversion existing.

### Later, same predicate

Named so the rule reads as a rule, not scoped in: a numbered menu answer
(`1`, `2`) or `y`/`n` while a picker is up is the same shape — a closed
utterance addressed to the harness. That one keys on agent *state* rather than
composer text, which is what `overlayDetector` already computes. Same
predicate, different input.

## Done when

- `/login` typed in the **agent pane** submits on plain Enter, no Alt. Same for
  `/clear`, `/context`.
- `/model opus` — sigil plus arguments — submits on Enter.
- `/Users/xianxu/workspace/brain is the repo` gets a newline, not a submit. So
  does `/ ` followed by prose, and so does any composer already holding a
  second line.
- Alt+Enter still submits in every case above, including the ones where Enter
  now also submits.
- A permission-prompt overlay still takes the bare-CR bypass first, unchanged.
- An agent absent from `harnessTTYProfiles`, and one with no sigils declared,
  behave exactly as they do today.
- `PAIR_WRAP_COMMAND_ENTER=0` restores today's behavior with the inversion
  still on.
- `pair-doctor` shows a distinct signal when the branch fires.
- The existing `decidePlainReturn` / composer-recognizer tests pass unchanged,
  and the recognizers' returned region is asserted against the committed
  terminal fixtures in `cmd/internal/wrapcmd/testdata`.
- Verified against the real binaries, not just unit tests — this is a
  screen-reading change on a live tty, which is what `probes/` is for.

## Plan

- [ ]

## Log

### 2026-09-01

- Filed from a brain session, then **corrected the same session**: first draft
  scoped this to the nvim draft pane and explicitly ruled the agent pane out of
  scope. Wrong surface — the operator meant the Claude/Codex tty directly, and
  the agent pane is the whole point. Body rewritten against
  `decidePlainReturn`; the draft pane demoted to a follow-on paragraph. The
  rule itself survived the correction unchanged, which is some evidence it is
  the right rule.
- The out-of-scope argument in the first draft was wrong on the facts, not just
  on scope: I claimed byte-level stdin rewriting has no view of the composer.
  It has a full one — `terminal_model.go` keeps a snapshot and
  `composer_recognizers.go` already locates the composer's ruled box in it.
- Confirmed mechanically rather than assumed: `detectClaudeOverlayOpen` keys on
  OSC 777 body `notify;Claude Code;Claude needs your permission`, a permission
  prompt — so the slash-command dropdown does not set `overlayActive`, and the
  composer branch is what runs on `/login`. Worth a probe to confirm the
  dropdown emits no OSC of its own.
- `/Users/...` as a legitimate slash-leading prose opener is the case that
  forced the token grammar in predicate rule 3; "first char is `/`" would
  mis-submit it, and in this operator's workflow that line is common.
- The recognizers computing and discarding the composer region is the reason
  this is a small change rather than a new subsystem.
