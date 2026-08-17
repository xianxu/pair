---
id: 000138
status: open
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
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

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; Claude should adopt the same safety model.
- Updated after pair#142 design review: Codex's cursor/paint detector is
  Codex-specific; Claude should reuse the positive-detection contract, not the
  exact terminal heuristic.
