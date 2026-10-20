---
id: 000742
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '693e630e0e85f0b4c7dbc5b23e1e1968a0757b32' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T12:53:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

- 2026-10-20 12:56 PDT — ops:0 supplied the morning evidence: target parley.nvim:4, Codex freshly switch-agent launched around 09:00 on f3c597f8 wrapper; requested sleep 60. UI showed Working (3s, esc to interrupt), then Waiting for background terminal (15s, esc to interrupt), 1 background terminal running. couch --interrupt refused with the nothing-sent/no-working-turn message. Source audit: Codex already has a visual observer; its recognizer only accepts Working in fixed physical bottom rows, so Waiting is a confirmed unsupported status. A top-anchored composer plus work status is also missed by fixed rows. No evidence yet assigns the first Working refusal to one guard.
- Existing Codex working.raw is a 154-byte status delta without a composer or complete synchronized frame; it cannot establish a composer-relative recognition contract. Capturing a fresh owned native PTY run before defining geometry. ARCH-DRY: reuse one Codex work recognizer for lifecycle and interrupt evidence, and preserve Claude working/stop-grace guards.
