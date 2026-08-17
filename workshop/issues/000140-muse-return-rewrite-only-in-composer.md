---
id: 000140
status: open
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
---

# Muse Return rewrite only in composer

## Problem

Muse currently participates in the pair Return remap convention, while
selection menus and permission prompts depend on known visible markers for
overlay bypass. Codex now uses a safer rule: rewrite plain Return only when
Pair positively identifies the live composer/input box. Muse should follow the
same contract so AskUserQuestion/request_user_input menus and future UI
variants can use plain Return to select without Pair having to enumerate every
menu footer.

## Spec

- For Muse, plain Return should rewrite to LF only when Pair positively
  identifies Muse's live composer/input box.
- Known Muse permission and user-selection menu markers still take precedence
  and force the next plain Return to pass through as bare CR.
- If Muse composer state is unknown, hidden, or not active, plain Return should
  pass through as bare CR.
- Composer detection should use stable terminal or agent-native signals, not
  absence of an overlay marker.
- Update `atlas/how-to-bring-up-a-new-harness-cli.md` or `atlas/architecture.md`
  if Muse needs agent-specific detection notes.

## Done when

- Muse plain Return rewrites only inside a positively detected composer.
- Muse permission prompts and request_user_input selection menus still accept
  plain Return.
- Unknown/non-composer Muse UI state receives bare CR.
- Tests cover active composer, inactive composer, and overlay precedence.
- Any Muse-specific harness guidance is documented.

## Plan

- [ ] Identify a stable Muse composer/input-box signal from raw Pair logs or
      agent output.
- [ ] Add a Muse composer tracker/gate or shared composer-gating abstraction
      following the Codex model.
- [ ] Add focused Return-routing tests for active composer, inactive composer,
      and overlay precedence.
- [ ] Update atlas/docs if the Muse signal is agent-specific.

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; Muse should adopt the same safety model.
