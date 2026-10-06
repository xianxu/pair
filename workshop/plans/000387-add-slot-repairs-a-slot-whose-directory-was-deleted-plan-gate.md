---
gate: plan-quality
issue: 387
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-05T17:04:11-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Host set-aside (Revision g) is in the table and Done-when but absent from PlanSlot rules, invariants I3/I4, ARCH-ORDER, R1 and any task
          detail: 'Rule 2, I3, I4 and ARCH-ORDER name SetAsideDep as the only removing step and R1 says only a dependency is removed. No task implements a host set-aside, captures the registration''s recorded branch before git worktree remove, or observes a host broken on positive evidence after repair. Host-under-live-agent severity is also contradictory: agent-live is degraded, a non-converged host is blocking. Generalize to one SetAside(checkout) step, or add a host step, and extend I3/I4 plus the crash test to cover it.'
          family: revision-not-propagated-to-invariants
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 1.5 named case still lists a Prune step, which round 1 replaced with targeted RemoveRegistration
          family: stale-vocabulary-after-revision
          round: 1
        - id: PQ-3
          severity: Minor
          title: Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
          family: plan-enumerates-test-cases
          round: 1
        - id: PQ-4
          severity: Minor
          title: Non-goals are scattered across sections rather than stated in one place
          family: missing-non-goals
          round: 1
      blocked: true
---

# Gate ledger — pair#387 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-05T17:04:11-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `revision-not-propagated-to-invariants` Host set-aside (Revision g) is in the table and Done-when but absent from PlanSlot rules, invariants I3/I4, ARCH-ORDER, R1 and any task
  Rule 2, I3, I4 and ARCH-ORDER name SetAsideDep as the only removing step and R1 says only a dependency is removed. No task implements a host set-aside, captures the registration's recorded branch before git worktree remove, or observes a host broken on positive evidence after repair. Host-under-live-agent severity is also contradictory: agent-live is degraded, a non-converged host is blocking. Generalize to one SetAside(checkout) step, or add a host step, and extend I3/I4 plus the crash test to cover it.
- **PQ-2** [Minor] `stale-vocabulary-after-revision` Task 1.5 named case still lists a Prune step, which round 1 replaced with targeted RemoveRegistration
- **PQ-3** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
- **PQ-4** [Minor] `missing-non-goals` Non-goals are scattered across sections rather than stated in one place

## Open findings

- **PQ-1** [Important] `revision-not-propagated-to-invariants` Host set-aside (Revision g) is in the table and Done-when but absent from PlanSlot rules, invariants I3/I4, ARCH-ORDER, R1 and any task
- **PQ-2** [Minor] `stale-vocabulary-after-revision` Task 1.5 named case still lists a Prune step, which round 1 replaced with targeted RemoveRegistration
- **PQ-3** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
- **PQ-4** [Minor] `missing-non-goals` Non-goals are scattered across sections rather than stated in one place
