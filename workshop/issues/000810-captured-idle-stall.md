---
id: 000810
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd7f96db618cdea2cd2770c66f859d20e43262bb4' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch recognition: fix first captured idle-admission stall

## Problem

Recurring Couch deliveries stall until operator input, before/after pair#735, perhaps twice daily. Static Claude auto/banner and bypass/banner frames and the Codex blank-line/one-warning frame replay as empty on current main; banner alone cannot explain the intermittent fault.

## Spec

Wait for the first real failed-admission capture from pair#791 diagnostics. Replay that exact screen with cursor/style/completion/resize and refusal evidence, isolate the responsible state/recognition transition, fix narrowly preserving unknown/draft/picker refusal and uncertain delivery handling. Do not invent cursor visibility or relax guards based on footer text.

## Done when

- The first real failing capture is retained as a sanitized regression with provenance.
- The exact refusing condition is reproduced; a bounded fix passes that regression and input-safety controls for Claude and Codex.

## Plan

- Wait for capture evidence; no implementation before a demonstrated failing transition.

## Log

2026-10-22: Filed by pair:1 under ops:0 approval of diagnostic-only pair#791. Unclaimed because it waits for a future real capture; no active implementation assigned.
