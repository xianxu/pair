---
id: 000138
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-19
estimate_hours: 3.32
started: 2026-08-19T19:25:36-07:00
---

# Claude Return rewrite only in composer

## Problem

Claude currently gets pair-style multiline input by rewriting plain Return in
the agent pane by default, with overlay detection as the escape hatch. Codex
now uses the safer rule: rewrite only when Pair positively identifies the live
composer/input box. Claude should move to the same integration contract so
permission prompts, pickers, and future Claude UI variants do not depend on
enumerating every non-composer menu.

## Spec

- For Claude, plain Return should rewrite to Claude's multiline input sequence
  only when Pair positively identifies Claude's live composer/input box.
- Known Claude permission/overlay signals still take precedence and force the
  next plain Return to pass through as bare CR.
- If Claude composer state is unknown, hidden, or not active, plain Return
  should pass through as bare CR.
- Composer detection should use stable terminal or agent-native signals, not
  "no overlay marker matched" as proof of composer focus.
- Update `atlas/how-to-bring-up-a-new-harness-cli.md` or `atlas/architecture.md`
  if Claude needs agent-specific detection notes.

## Done when

- Claude plain Return rewrites only inside a positively detected composer.
- Claude permission prompts/pickers still accept plain Return.
- Unknown/non-composer Claude UI state receives bare CR.
- Tests cover active composer, inactive composer, and overlay precedence.
- Any Claude-specific harness guidance is documented.

## Plan

- [ ] Identify a stable Claude composer/input-box signal from raw Pair logs or
      agent output.
- [ ] Add a Claude composer tracker/gate or shared composer-gating abstraction
      based on Claude's own native composer-availability signal, not Codex's
      cursor/paint heuristic unless Claude logs prove the same signal is stable.
- [ ] Add focused Return-routing tests for active composer, inactive composer,
      and overlay precedence.
- [ ] Update atlas/docs if the Claude signal is agent-specific.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
as stale (newer ledger rows than the doc), so the number is provisional but uses
the required method: design hours unchanged, `impl=` written at 40% of the
v2/v2.1 primitive table, +15% design buffer for a thorough plan doc.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.10
item: cross-cutting-refactor design=0.15 impl=0.20
item: smaller-go-module design=0.15 impl=0.20
item: smaller-go-module design=0.05 impl=0.10
item: smaller-go-module design=0.10 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
item: tui-screen design=0.20 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: milestone-review design=0.04 impl=0.20
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.32
```

Sizing notes, revised after estimate-quality (all four findings were bookkeeping
errors in the first draft and are corrected here rather than argued with).

`issue-spec` covers plan authoring and the plan-quality gate rounds — the first
round blocked on seven findings — not the live captures; those are
`real-api-discovery`, which prices the Claude CLI as one external surface across
three capture campaigns (composer, bash mode, permission prompt). The first draft
described the same work under both labels.

`cross-cutting-refactor` is Task 1. The three `smaller-go-module` rows are Tasks
2, 3, and **4's non-capture half** — the profile flip, the legacy
characterization update, the driven-scenario registration, and the two possible
gap-ledger entries. The first draft's prose claimed those three rows were Tasks
1–3, leaving Task 4 — the largest task, five files — carried entirely by
`real-api-discovery`, which is not API discovery.

Two `impl=` values were above the v3.1 40% band and are pulled to their ceilings:
the recognizer row 0.25 → 0.20, and `atlas-docs` 0.12 → 0.08. `atlas-docs` is
also split in two, matching Task 6's actual shape: the atlas pair
(`architecture.md`, the bring-up guide) and the user-facing pair (`README.md`,
`doctor/README.md`).

`milestone-review` now appears three times rather than two. The slug is a stretch
— this plan carries no `Mx` tags, so there is one close boundary and one
mandatory review — but the v2 table has no primitive for *rework rounds*, and
rework is the single largest historical cost driver for this work. pair#139 drew
REWORK then two FIX-THEN-SHIP rounds; budgeting two rounds while citing three was
the first draft's own inconsistency. All three rows sit at the primitive's
ceiling, so pricing a fourth round would need a fourth row.

**Known under-estimation risk.** The immediately preceding and closely related
pair#139 estimated 5.83 and measured 23.40 — a 0.25x ratio, and the lone low
outlier in a pair family that otherwise runs *over* (#137 4.05x, #136 3.64x,
#131 3.35x, #130 2.42x). The dominant cause was review rounds discovering that
each recognizer rejected ordinary composing states. This plan front-loads those
exact cases (multi-line, blank-line, empty-continuation, height ceiling, mode
coverage) specifically to avoid repeating that, which is the argument for a
smaller number here — but the model is derived, not adjusted, and if this issue
also overruns then the recognizer family needs its own primitive rather than
another one-off correction.

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; Claude should adopt the same safety model.
- Updated after pair#142 design review: Codex's cursor/paint detector is
  Codex-specific; Claude should reuse the positive-detection contract, not the
  exact terminal heuristic.
