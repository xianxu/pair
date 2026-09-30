---
gate: plan-quality
issue: 355
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-30T09:59:40-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify pending-binding recovery through recovered-unknown transitions
          detail: 'ARCH-ORDER: Define the binding state and effects for registration, recovered-unknown, rollback, and retry through AdvanceStartTransaction/ReconcileStart. The current recovered-unknown transition clears incarnation.Start (cmd/internal/couchcore/starttransaction.go:116), so specify where the proposed binding survives, what evidence promotes or retires it, and when another M may be allocated; require controllable sequence tests through the production transition authority.'
          family: explicit-uncertain-state-transitions
          round: 1
        - id: PQ-2
          severity: Important
          title: Replace prose test-case inventories with named functions and adversarial strategies
          detail: Tasks 1–4 enumerate test cases rather than naming the functions under unit test and one mechanical strategy per risky function. Name the allocation, ownership-classification, parsing, binding-transition, and family-resolution functions, and compress the inventories into input classes plus guards such as fuzzing, invariant-based event sequences, and injected publication failures; preserve the required incident regression and production-boundary checks.
          family: function-level-test-strategy
          round: 1
        - id: PQ-3
          severity: Minor
          title: Define retention for compatibility index entries added per terminal incarnation
          detail: 'ARCH-FUNERAL: The plan appends a compatibility index association for every newly allocated M, while its retention discussion covers only thread/start bindings and host assignments. State when superseded index associations stop being needed and how they are removed or bounded; retaining them forever requires an explicit per-recreation cost.'
          family: durable-growth-lifecycle
          round: 1
        - id: PQ-4
          severity: Minor
          title: Name the recurring ownership-query conformance check cadence
          detail: 'ARCH-MOCK: The plan identifies the stateful fake and implementation-time live checks, but not when those checks recur to detect Zellij command-field or server-identity drift. Name the cadence or release trigger and the check that runs.'
          family: external-conformance-cadence
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-30T10:03:08-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Binding promotion precedes Start clearing; uncertainty preserves occupancy, and evidence gates rollback and retry through production transition tests.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Named functions now have adversarial strategies and mechanical guards, retaining incident and production-boundary regressions.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Atomic replacement bounds compatibility index associations to one per exact scope/tag.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Live conformance runs before releases supporting a changed Zellij version; parsing fixtures run in the ordinary suite.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-30T10:04:32-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Recovered-unknown promotes the pending binding before clearing Start; uncertainty retains occupancy and requires independent ownership evidence.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Tasks name tested functions with adversarial strategies and mechanical invariants.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Compatibility index publication replaces the exact scope/tag association while preserving unrelated addresses.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Live conformance runs before each release supporting a changed Zellij version; parser fixtures run in the ordinary suite.
          round: 3
      blocked: false
content_hash: 48efcb979f644acdcacd482854b24f4861058b53f103d4f2c75e2e4558b648ae
---

# Gate ledger — pair#355 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T09:59:40-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `explicit-uncertain-state-transitions` Specify pending-binding recovery through recovered-unknown transitions
  ARCH-ORDER: Define the binding state and effects for registration, recovered-unknown, rollback, and retry through AdvanceStartTransaction/ReconcileStart. The current recovered-unknown transition clears incarnation.Start (cmd/internal/couchcore/starttransaction.go:116), so specify where the proposed binding survives, what evidence promotes or retires it, and when another M may be allocated; require controllable sequence tests through the production transition authority.
- **PQ-2** [Important] `function-level-test-strategy` Replace prose test-case inventories with named functions and adversarial strategies
  Tasks 1–4 enumerate test cases rather than naming the functions under unit test and one mechanical strategy per risky function. Name the allocation, ownership-classification, parsing, binding-transition, and family-resolution functions, and compress the inventories into input classes plus guards such as fuzzing, invariant-based event sequences, and injected publication failures; preserve the required incident regression and production-boundary checks.
- **PQ-3** [Minor] `durable-growth-lifecycle` Define retention for compatibility index entries added per terminal incarnation
  ARCH-FUNERAL: The plan appends a compatibility index association for every newly allocated M, while its retention discussion covers only thread/start bindings and host assignments. State when superseded index associations stop being needed and how they are removed or bounded; retaining them forever requires an explicit per-recreation cost.
- **PQ-4** [Minor] `external-conformance-cadence` Name the recurring ownership-query conformance check cadence
  ARCH-MOCK: The plan identifies the stateful fake and implementation-time live checks, but not when those checks recur to detect Zellij command-field or server-identity drift. Name the cadence or release trigger and the check that runs.

## Round 2 — 2026-09-30T10:03:08-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Binding promotion precedes Start clearing; uncertainty preserves occupancy, and evidence gates rollback and retry through production transition tests.
- PQ-2 — addressed — Named functions now have adversarial strategies and mechanical guards, retaining incident and production-boundary regressions.
- PQ-3 — addressed — Atomic replacement bounds compatibility index associations to one per exact scope/tag.
- PQ-4 — addressed — Live conformance runs before releases supporting a changed Zellij version; parsing fixtures run in the ordinary suite.

## Round 3 — 2026-09-30T10:04:32-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Recovered-unknown promotes the pending binding before clearing Start; uncertainty retains occupancy and requires independent ownership evidence.
- PQ-2 — addressed — Tasks name tested functions with adversarial strategies and mechanical invariants.
- PQ-3 — addressed — Compatibility index publication replaces the exact scope/tag association while preserving unrelated addresses.
- PQ-4 — addressed — Live conformance runs before each release supporting a changed Zellij version; parser fixtures run in the ordinary suite.

## Open findings

(none — every finding has been disposed)
