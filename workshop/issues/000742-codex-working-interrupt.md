---
id: 000742
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '789da3925b54577e4f5a6a1044181df64fccc908' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch interrupt refuses a working Codex turn

## Problem

Follow-up to #717 (landed in PR #483): the morning live check reported couch --interrupt refusing a Codex turn that was working. The existing admission proof combines reducer-owned working state, recognized composer, current viewport interrupt hint, and automatic-input ownership. Diagnose which proof is missing from the real Codex path; do not weaken idle/panel/picker or second-interrupt protections to make the working case pass.

## Spec

## Done when

- Reproduce the reported Codex working-turn refusal with captured evidence and an automated regression that fails before the fix.
- A current working Codex turn admits exactly one Ctrl+C, then confirms only the observed turn end. A second interrupt after completion refuses and sends nothing.
- Idle, stop grace, panels, pickers, stale ownership and competing input continue to refuse without a write.
- Targeted tests, the recorded local check, boundary review and a hub-coordinated live Codex check pass before landing.

## Plan

- [ ]

## Log

- 2026-10-20 — Dispatched by ops:0 after #726 landed, before #733. #717 is already done and its old branch must not be reused. Current archived #717 Log describes BR-4/BR-6 fixes but lacks the reported failed live-check detail; asked ops:0 for target/time/refusal while tracing the production path.
