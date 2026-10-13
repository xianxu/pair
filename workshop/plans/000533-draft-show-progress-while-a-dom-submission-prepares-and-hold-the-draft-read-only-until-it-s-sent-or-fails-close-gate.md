---
gate: boundary-review
issue: 533
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T14:47:41-07:00"
      dispatched: "2026-10-13T14:41:38-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Read-only DOM preparation permits partial navigation that overwrites queued text
          detail: 'nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.'
          family: operation-ownership-must-guard-mutators
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#533 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T14:47:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `operation-ownership-must-guard-mutators` Read-only DOM preparation permits partial navigation that overwrites queued text
  nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.

## Open findings

- **BR-1** [Critical] `operation-ownership-must-guard-mutators` Read-only DOM preparation permits partial navigation that overwrites queued text
