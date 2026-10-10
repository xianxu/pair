---
gate: plan-quality
issue: 426
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-10T12:19:51-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Define how the shared parser exposes malformed markers.
          detail: 'ARCH-PURPOSE / ARCH-SECURE: workshop/plans/000426-compact-review-threads-plan.md:67 requires parser output to supply warnings and shared eligibility, but the existing parser drops wholly malformed markers and accepts valid prefixes of broken chains. Specify a shared diagnostic/completeness result, including malformed extent and code exclusions, so conceal, cursor protection, and float targeting cannot treat a broken prefix as an eligible complete marker while legacy consumers retain their contract.'
          family: shared-parser-validity-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace test-case inventories with named functions and strategy lines.
          detail: Tasks 1–3 enumerate test cases at plan lines 139, 149, and 160; only encode_turn/decode_turn are named in the design. Compress these inventories into the planned function names for parsing, layout/snapping, thread conversion, resolution, and lifecycle transitions, with one adversarial input class and mechanical guard per risky function; retain independent wire fixtures and real-render/handoff checks.
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: Define the limit and fallback for synchronous document reparsing.
          detail: 'ARCH-CONSTRAINTS: plan lines 101–106 specify a representative workload and recording timings, but no acceptance budget or behavior beyond that workload. State a justified initial size or responsiveness bound and the visible fallback when exceeded, so a successful timing measurement alone does not establish an unlimited operating envelope.'
          family: explicit-operating-envelope
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T12:20:46-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Shared scan diagnostics and completeness govern conceal, snapping, and targeting while preserving legacy parser consumers.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Named functions now carry property and invariant test strategies, with independent wire fixtures and real-render/handoff verification retained.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Compact rendering is bounded to 1,000 lines and 128 KiB, with visible raw fallback and a 50ms refresh target.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-10T12:21:27-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Shared scan results define completeness, malformed diagnostics and eligibility.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Named risky functions have adversarial strategies and mechanical invariants.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Compact rendering has explicit input limits, a refresh target and visible raw fallback.
          round: 3
      blocked: false
content_hash: fe4e6370b93d6a1b7d784632b99d5a1fc0b410bcadecd45fe87b908e44652864
---

# Gate ledger — pair#426 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T12:19:51-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `shared-parser-validity-contract` Define how the shared parser exposes malformed markers.
  ARCH-PURPOSE / ARCH-SECURE: workshop/plans/000426-compact-review-threads-plan.md:67 requires parser output to supply warnings and shared eligibility, but the existing parser drops wholly malformed markers and accepts valid prefixes of broken chains. Specify a shared diagnostic/completeness result, including malformed extent and code exclusions, so conceal, cursor protection, and float targeting cannot treat a broken prefix as an eligible complete marker while legacy consumers retain their contract.
- **PQ-2** [Important] `function-level-test-strategy` Replace test-case inventories with named functions and strategy lines.
  Tasks 1–3 enumerate test cases at plan lines 139, 149, and 160; only encode_turn/decode_turn are named in the design. Compress these inventories into the planned function names for parsing, layout/snapping, thread conversion, resolution, and lifecycle transitions, with one adversarial input class and mechanical guard per risky function; retain independent wire fixtures and real-render/handoff checks.
- **PQ-3** [Minor] `explicit-operating-envelope` Define the limit and fallback for synchronous document reparsing.
  ARCH-CONSTRAINTS: plan lines 101–106 specify a representative workload and recording timings, but no acceptance budget or behavior beyond that workload. State a justified initial size or responsiveness bound and the visible fallback when exceeded, so a successful timing measurement alone does not establish an unlimited operating envelope.

## Round 2 — 2026-10-10T12:20:46-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Shared scan diagnostics and completeness govern conceal, snapping, and targeting while preserving legacy parser consumers.
- PQ-2 — addressed — Named functions now carry property and invariant test strategies, with independent wire fixtures and real-render/handoff verification retained.
- PQ-3 — addressed — Compact rendering is bounded to 1,000 lines and 128 KiB, with visible raw fallback and a 50ms refresh target.

## Round 3 — 2026-10-10T12:21:27-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Shared scan results define completeness, malformed diagnostics and eligibility.
- PQ-2 — addressed — Named risky functions have adversarial strategies and mechanical invariants.
- PQ-3 — addressed — Compact rendering has explicit input limits, a refresh target and visible raw fallback.

## Open findings

(none — every finding has been disposed)
