---
gate: boundary-review
issue: 341
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T21:33:20-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Handoff deletion precedes final application authorization
          detail: nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
          family: consume-after-acceptance
          round: 1
        - id: BR-2
          severity: Important
          title: Routine typing synchronously resolves full Git history
          detail: nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
          family: nonblocking-editor-observation
          round: 1
        - id: BR-3
          severity: Important
          title: README omits branch restoration and recovery commands
          detail: README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
          family: user-surface-documentation
          round: 1
        - id: BR-4
          severity: Minor
          title: Repeated activation accumulates rendering autocmds
          detail: nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).
          family: activation-resource-ownership
          round: 1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#341 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T21:33:20-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `consume-after-acceptance` Handoff deletion precedes final application authorization
  nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
- **BR-2** [Important] `nonblocking-editor-observation` Routine typing synchronously resolves full Git history
  nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
- **BR-3** [Important] `user-surface-documentation` README omits branch restoration and recovery commands
  README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
- **BR-4** [Minor] `activation-resource-ownership` Repeated activation accumulates rendering autocmds
  nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).

## Open findings

- **BR-1** [Critical] `consume-after-acceptance` Handoff deletion precedes final application authorization
- **BR-2** [Important] `nonblocking-editor-observation` Routine typing synchronously resolves full Git history
- **BR-3** [Important] `user-surface-documentation` README omits branch restoration and recovery commands
- **BR-4** [Minor] `activation-resource-ownership` Repeated activation accumulates rendering autocmds
