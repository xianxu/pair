---
gate: plan-quality
issue: 526
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-13T08:52:40-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace test-case inventories with named functions and adversarial strategies.
          detail: Tasks 1–3 enumerate cases instead of providing the required concise strategy per risky function. Compress them into function → adversarial input class → mechanical guard entries covering ParseOriginPolicy, Decide, manifest validation, the policy reader, PrepareSubmission, CaptureCrops, and the box-text collector. Name proposed functions where necessary; specify generated malformed-input checks, controlled event ordering, or sentinel invariants as appropriate, retaining the invalid-policy successful-delivery and notice requirement (ARCH-PURE, ARCH-SECURE, ARCH-ORDER).
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Minor
          title: Describe actual retention and capacity behavior rather than implying existing cleanup.
          detail: Decision 8 and ARCH-FUNERAL claim an existing bounded lifecycle and store cleanup for recordings and crops. Recordings are retained until operator deletion (cmd/internal/couchcmd/webcast.go:24); submission admission refuses at capacity with a cleanup instruction (cmd/internal/submission/store.go:436), while RemoveAll at :444 removes staging data. State these existing limits and removal owners accurately; this issue need not introduce a new retention system.
          family: existing-lifecycle-claims
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T08:54:09-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Function-level strategies cover the requested risky surfaces and retain invalid-policy successful delivery and notice requirements.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Decision 8 and ARCH-FUNERAL correctly distinguish operator-owned retention, capacity refusal, and staging cleanup.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-13T08:55:48-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Named risky functions now have adversarial input strategies and mechanical guards.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Retention descriptions now distinguish operator deletion, capacity refusal, and staging cleanup.
          round: 3
      blocked: false
content_hash: 16e1cd4b04747d1633fbfda7dc7f79b52556a18ba14769e1ae2214b47182db21
---

# Gate ledger — pair#526 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T08:52:40-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace test-case inventories with named functions and adversarial strategies.
  Tasks 1–3 enumerate cases instead of providing the required concise strategy per risky function. Compress them into function → adversarial input class → mechanical guard entries covering ParseOriginPolicy, Decide, manifest validation, the policy reader, PrepareSubmission, CaptureCrops, and the box-text collector. Name proposed functions where necessary; specify generated malformed-input checks, controlled event ordering, or sentinel invariants as appropriate, retaining the invalid-policy successful-delivery and notice requirement (ARCH-PURE, ARCH-SECURE, ARCH-ORDER).
- **PQ-2** [Minor] `existing-lifecycle-claims` Describe actual retention and capacity behavior rather than implying existing cleanup.
  Decision 8 and ARCH-FUNERAL claim an existing bounded lifecycle and store cleanup for recordings and crops. Recordings are retained until operator deletion (cmd/internal/couchcmd/webcast.go:24); submission admission refuses at capacity with a cleanup instruction (cmd/internal/submission/store.go:436), while RemoveAll at :444 removes staging data. State these existing limits and removal owners accurately; this issue need not introduce a new retention system.

## Round 2 — 2026-10-13T08:54:09-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Function-level strategies cover the requested risky surfaces and retain invalid-policy successful delivery and notice requirements.
- PQ-2 — addressed — Decision 8 and ARCH-FUNERAL correctly distinguish operator-owned retention, capacity refusal, and staging cleanup.

## Round 3 — 2026-10-13T08:55:48-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Named risky functions now have adversarial input strategies and mechanical guards.
- PQ-2 — addressed — Retention descriptions now distinguish operator deletion, capacity refusal, and staging cleanup.

## Open findings

(none — every finding has been disposed)
