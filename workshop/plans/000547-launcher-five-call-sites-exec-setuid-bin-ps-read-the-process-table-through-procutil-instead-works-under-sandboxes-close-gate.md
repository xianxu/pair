---
gate: boundary-review
issue: 547
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-14T01:00:28-07:00"
      dispatched: "2026-10-14T00:57:23-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Reparented test helper leaves descendants running after timeout
          detail: tests/without-ancestors.py:36 kills only the immediate subprocess on timeout; the worker reports failure and the supervisor cancels its group-cleanup alarm at line 63. A controlled reproduction returned after approximately 0.37 seconds while its sleep grandchild remained alive. ARCH-ORDER and ARCH-FUNERAL require cleanup of the owned process group on timeout and supervisor cancellation, with regression tests proving descendants terminate.
          family: owned-process-lifetime
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-14T01:10:41-07:00"
      dispatched: "2026-10-14T01:06:58-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: tests/without-ancestors.py:30-106 establishes group ownership and a startup handshake, then kills the group on failure or cancellation. All five lifecycle tests pass at head. Against the pre-fix helper, the timeout regression fails with a surviving grandchild; cancellation regressions also fail. The disposable regression fixture was removed.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: false
---

# Gate ledger — pair#547 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T01:00:28-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `owned-process-lifetime` Reparented test helper leaves descendants running after timeout
  tests/without-ancestors.py:36 kills only the immediate subprocess on timeout; the worker reports failure and the supervisor cancels its group-cleanup alarm at line 63. A controlled reproduction returned after approximately 0.37 seconds while its sleep grandchild remained alive. ARCH-ORDER and ARCH-FUNERAL require cleanup of the owned process group on timeout and supervisor cancellation, with regression tests proving descendants terminate.

## Round 2 — 2026-10-14T01:10:41-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — tests/without-ancestors.py:30-106 establishes group ownership and a startup handshake, then kills the group on failure or cancellation. All five lifecycle tests pass at head. Against the pre-fix helper, the timeout regression fails with a surviving grandchild; cancellation regressions also fail. The disposable regression fixture was removed.

## Open findings

(none — every finding has been disposed)
