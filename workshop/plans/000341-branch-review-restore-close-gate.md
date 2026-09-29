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
    - "n": 2
      timestamp: "2026-09-28T22:00:39-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Handoff consumption follows explicit apply/defer acceptance. The passing production-function regression fails with the old watcher at the final-refusal preservation assertion.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Edit observation is asynchronous and coalesced. Restoring synchronous observation fails the responsiveness test at approximately 1155 ms with six scans. The distinct late-completion safety regression is reported below.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:99 documents branch restoration and blocking conditions; README.md:108 documents recovery/discard commands implemented in nvim/review.lua.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Activation owns a clearable rendering group. The passing observation test fails with callback counts increasing from one to three when cleanup is removed.
          round: 2
      findings:
        - id: BR-5
          severity: Critical
          title: Late asynchronous observation reloads another branch into the active review
          detail: 'nvim/review/recovery_observer.lua:36 accepts a captured matching identity, then nvim/review.lua:843 invokes checktime against the current checkout. A controlled production-pane probe captured review/a, switched to review/b before delivery, and loaded B''s bytes while the pane remained bound to A. This is the 2nd finding in this family: enforce identity-bound observation effects across refresh, preservation, and coalescing decisions rather than patching only this callback. Reload verified snapshot bytes and add a controlled late-completion regression without restoring synchronous editor observation. ARCH-ORDER, ARCH-PURPOSE.'
          family: nonblocking-editor-observation
          round: 2
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

## Round 2 — 2026-09-28T22:00:39-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Handoff consumption follows explicit apply/defer acceptance. The passing production-function regression fails with the old watcher at the final-refusal preservation assertion.
- BR-2 — addressed — Edit observation is asynchronous and coalesced. Restoring synchronous observation fails the responsiveness test at approximately 1155 ms with six scans. The distinct late-completion safety regression is reported below.
- BR-3 — addressed — README.md:99 documents branch restoration and blocking conditions; README.md:108 documents recovery/discard commands implemented in nvim/review.lua.
- BR-4 — addressed — Activation owns a clearable rendering group. The passing observation test fails with callback counts increasing from one to three when cleanup is removed.

### Raised

- **BR-5** [Critical] `nonblocking-editor-observation` Late asynchronous observation reloads another branch into the active review
  nvim/review/recovery_observer.lua:36 accepts a captured matching identity, then nvim/review.lua:843 invokes checktime against the current checkout. A controlled production-pane probe captured review/a, switched to review/b before delivery, and loaded B's bytes while the pane remained bound to A. This is the 2nd finding in this family: enforce identity-bound observation effects across refresh, preservation, and coalescing decisions rather than patching only this callback. Reload verified snapshot bytes and add a controlled late-completion regression without restoring synchronous editor observation. ARCH-ORDER, ARCH-PURPOSE.

## Open findings

- **BR-5** [Critical] `nonblocking-editor-observation` Late asynchronous observation reloads another branch into the active review
