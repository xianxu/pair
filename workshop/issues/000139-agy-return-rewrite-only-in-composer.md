---
id: 000139
status: open
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
---

# Agy Return rewrite only in composer

## Problem

Agy currently participates in the pair Return remap convention, while overlay
handling depends on known visible prompt markers. Codex now uses a safer rule:
rewrite plain Return only when Pair positively identifies the live
composer/input box. Agy should follow the same contract so permission pickers
and future UI variants keep plain Return as confirm instead of receiving an
accidental newline.

## Spec

- For agy, plain Return should rewrite to LF only when Pair positively
  identifies agy's live composer/input box.
- Known agy permission/overlay markers still take precedence and force the next
  plain Return to pass through as bare CR.
- If agy composer state is unknown, hidden, or not active, plain Return should
  pass through as bare CR.
- Composer detection should use stable terminal or agent-native signals, not
  absence of an overlay marker.
- Update `atlas/how-to-bring-up-a-new-harness-cli.md` or `atlas/architecture.md`
  if agy needs agent-specific detection notes.

## Done when

- Agy plain Return rewrites only inside a positively detected composer.
- Agy permission prompts/pickers still accept plain Return.
- Unknown/non-composer agy UI state receives bare CR.
- Tests cover active composer, inactive composer, and overlay precedence.
- Any agy-specific harness guidance is documented.

## Plan

- [ ] Identify a stable agy composer/input-box signal from raw Pair logs or
      agent output.
- [ ] Add an agy composer tracker/gate or shared composer-gating abstraction
      following the Codex model.
- [ ] Add focused Return-routing tests for active composer, inactive composer,
      and overlay precedence.
- [ ] Update atlas/docs if the agy signal is agent-specific.

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; agy should adopt the same safety model.
