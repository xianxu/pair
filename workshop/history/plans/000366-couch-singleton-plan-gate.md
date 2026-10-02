---
gate: plan-quality
issue: 366
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-01T22:40:06-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define side-effect-free preview and ownership-aware adoption revalidation.
          detail: Existing owner inspection can create a lock file, and acquiring the selected-store lease publishes owner metadata. Specify a non-mutating preview probe, digest evidence, and how apply distinguishes its own acquired lease from conflicting ownership so unchanged evidence remains applicable without hiding external changes (ARCH-PURE, ARCH-ORDER).
          family: observation-effect-separation
          round: 1
        - id: PQ-2
          severity: Important
          title: Compress test inventories into named function strategies with controllable ordering.
          detail: Tasks 1–3 enumerate test cases rather than naming each risky function's adversarial input class and mechanical guard. Replace those inventories with concise strategies, including a deterministic ordering/failure seam for Manager.Adopt and Manager.Acquire that exercises revalidation, interrupted publication, and retry through production logic (ARCH-ORDER).
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-01T22:42:05-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Revision defines non-mutating preview, preservation digest boundaries, separate contention checks and acquired-lease-aware source revalidation.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Revision supersedes case inventories with named function strategies and deterministic production-path ordering, publication-failure and retry seams.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-01T22:44:16-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The revision specifies non-mutating observation, separates contention from preservation evidence, and supplies owned authority during final revalidation.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Named function strategies supersede case inventories and provide controllable ordering, publication failures, and forbidden-effect assertions.
          round: 3
      blocked: false
content_hash: fce169c0d826e8b12a5a570fe6dd2c8c3e738dabdad7cddb77b6bd4488feb51d
---

# Gate ledger — pair#366 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T22:40:06-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `observation-effect-separation` Define side-effect-free preview and ownership-aware adoption revalidation.
  Existing owner inspection can create a lock file, and acquiring the selected-store lease publishes owner metadata. Specify a non-mutating preview probe, digest evidence, and how apply distinguishes its own acquired lease from conflicting ownership so unchanged evidence remains applicable without hiding external changes (ARCH-PURE, ARCH-ORDER).
- **PQ-2** [Important] `function-level-test-strategy` Compress test inventories into named function strategies with controllable ordering.
  Tasks 1–3 enumerate test cases rather than naming each risky function's adversarial input class and mechanical guard. Replace those inventories with concise strategies, including a deterministic ordering/failure seam for Manager.Adopt and Manager.Acquire that exercises revalidation, interrupted publication, and retry through production logic (ARCH-ORDER).

## Round 2 — 2026-10-01T22:42:05-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Revision defines non-mutating preview, preservation digest boundaries, separate contention checks and acquired-lease-aware source revalidation.
- PQ-2 — addressed — Revision supersedes case inventories with named function strategies and deterministic production-path ordering, publication-failure and retry seams.

## Round 3 — 2026-10-01T22:44:16-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The revision specifies non-mutating observation, separates contention from preservation evidence, and supplies owned authority during final revalidation.
- PQ-2 — addressed — Named function strategies supersede case inventories and provide controllable ordering, publication failures, and forbidden-effect assertions.

## Open findings

(none — every finding has been disposed)
