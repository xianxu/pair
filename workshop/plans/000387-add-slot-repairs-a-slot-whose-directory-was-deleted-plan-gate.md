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
    - "n": 2
      timestamp: "2026-10-05T17:05:32-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: One SetAside(checkout) step for the host and every dep:* (rule 2, I3, I4, R1, ARCH-ORDER, Tasks 1.4, 3.1, 3.2, 3.5), branch recorded before rename; agent-live is degraded by reason.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Task 1.5's deleted-slot case now lists RemoveRegistration.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: The author declined; the lists in Tasks 1.4 and 3.1 are hand-written cases, not derived perturbations. Carried to the close review as Minor.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: A single Non-goals section now exists.
          round: 2
      findings:
        - id: PQ-5
          severity: Minor
          title: The agent row of the resource table and the OutcomeSeverity signature still describe the dependency-only hold
          detail: '2nd finding in this family. The agent row says a live agent forbids any dep:* removal, but rule 3 and I4 now hold the host too. OutcomeSeverity(resource, state, marker) takes no stop reason, although rule 3 derives severity from the reason, so as written a host agent-live stop is classed blocking. Rule: a revision that changes a step''s or guard''s subject must search every occurrence of the old subject in the same revision. Fix: update the agent row to cover the host and every dep:*, and make OutcomeSeverity take the stop reason.'
          family: revision-not-propagated-to-invariants
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-05T17:06:23-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Agent row names the host and every dep:*; OutcomeSeverity takes stopReason (table and outcome section), reason decides first.
          round: 3
        - id: PQ-3
          disposition: not-addressed
          note: Revision h deliberately kept the prose case lists in Tasks 1.4/2.5/3.1; Minor, carried to the close review.
          round: 3
      blocked: false
content_hash: 182604b7711801e94e62febf94d655f4f88f0b393fa522e13c6611d52facf9a0
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

## Round 2 — 2026-10-05T17:05:32-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — One SetAside(checkout) step for the host and every dep:* (rule 2, I3, I4, R1, ARCH-ORDER, Tasks 1.4, 3.1, 3.2, 3.5), branch recorded before rename; agent-live is degraded by reason.
- PQ-2 — addressed — Task 1.5's deleted-slot case now lists RemoveRegistration.
- PQ-3 — not-addressed — The author declined; the lists in Tasks 1.4 and 3.1 are hand-written cases, not derived perturbations. Carried to the close review as Minor.
- PQ-4 — addressed — A single Non-goals section now exists.

### Raised

- **PQ-5** [Minor] `revision-not-propagated-to-invariants` The agent row of the resource table and the OutcomeSeverity signature still describe the dependency-only hold
  2nd finding in this family. The agent row says a live agent forbids any dep:* removal, but rule 3 and I4 now hold the host too. OutcomeSeverity(resource, state, marker) takes no stop reason, although rule 3 derives severity from the reason, so as written a host agent-live stop is classed blocking. Rule: a revision that changes a step's or guard's subject must search every occurrence of the old subject in the same revision. Fix: update the agent row to cover the host and every dep:*, and make OutcomeSeverity take the stop reason.

## Round 3 — 2026-10-05T17:06:23-07:00 (claude) — passed

### Disposed

- PQ-5 — addressed — Agent row names the host and every dep:*; OutcomeSeverity takes stopReason (table and outcome section), reason decides first.
- PQ-3 — not-addressed — Revision h deliberately kept the prose case lists in Tasks 1.4/2.5/3.1; Minor, carried to the close review.

## Open findings

- **PQ-3** [Minor] `plan-enumerates-test-cases` Tasks 1.4, 2.5 and 3.1 enumerate test cases in prose; compress to functions plus one strategy line per risky function
