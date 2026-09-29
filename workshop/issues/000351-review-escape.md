---
id: 000351
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '650b9adb606abc0a552e7cb570934f1551e42199' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-29T14:53:39-07:00
flow: {kind: quick, provenance: inferred, spec: "ddfb2da7", done: "a3dc7c9c"}
---

# Preserve Escape in review pane

## Problem

Normal-mode Escape hides the review pane during ordinary Vim editing.

## Spec

Escape may dismiss review floats but must never return to draft. Alt+c remains the explicit return shortcut. Update statusline, help, and atlas together. ARCH-DRY: reuse the existing Alt+c route; only remove Escape’s fallback.

## Done when

- Repeated Escape in normal, insert, and visual modes keeps review visible; float dismissal still works.
- Alt+c returns to draft and hints advertise only that return key.

## Plan

- [x] Update regression, remove Escape return fallback, reconcile help and atlas, and run focused tests.

## Log

### 2026-09-29

- Root cause: escape_review returns to draft when no float exists. Existing headless test explicitly required this behavior.

- Verified red/green with real headless Neovim and the stateful host: repeated Escape retains review, floats dismiss, Alt+c returns. Review-window suite and keyhelp tests pass after make build refreshes embedded runtime.
