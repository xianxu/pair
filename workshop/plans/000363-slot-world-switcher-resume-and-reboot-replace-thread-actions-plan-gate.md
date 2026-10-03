---
gate: plan-quality
issue: 363
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-02T22:04:48-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
          detail: Keep the derived-domain tables (DecideReboot, everyMenuRowShape, ResumeRebootAdvice parse); compress the named per-case bullets to the adversarial class + guard.
          family: test-prose-enumeration
          round: 1
        - id: PQ-2
          severity: Minor
          title: The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
          detail: Add an assertion that a later start/reboot in the same path succeeds with a leaked start claim present, so "same window as AllocateThreadTag" stays an invariant.
          family: unconfirmed-outcome-untested
          round: 1
        - id: PQ-3
          severity: Minor
          title: Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
          detail: Have the root status and Enter notice read the menuPhaseBusy fact so the per-row authority stays single-sourced.
          family: single-action-authority
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-10-02T22:06:03-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Tasks 1.3/1.5/1.6/2.2/2.4 still enumerate per-case bullets; carried to close review as Minor.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Task 1.6 step 4b still asserts the leaked-claim window in prose with no test pinning it.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Task 2.1 still patches menu.go case "" instead of deriving busy text from menuPhaseBusy.
          round: 2
      blocked: false
content_hash: aca5fbc312dcf59133f3132d9a302cc8a11a924b11d348a3476fb0c590bc4510
---

# Gate ledger — pair#363 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-02T22:04:48-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `test-prose-enumeration` Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
  Keep the derived-domain tables (DecideReboot, everyMenuRowShape, ResumeRebootAdvice parse); compress the named per-case bullets to the adversarial class + guard.
- **PQ-2** [Minor] `unconfirmed-outcome-untested` The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
  Add an assertion that a later start/reboot in the same path succeeds with a leaked start claim present, so "same window as AllocateThreadTag" stays an invariant.
- **PQ-3** [Minor] `single-action-authority` Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
  Have the root status and Enter notice read the menuPhaseBusy fact so the per-row authority stays single-sourced.

## Round 2 — 2026-10-02T22:06:03-07:00 (claude) — passed

### Disposed

- PQ-1 — not-addressed — Tasks 1.3/1.5/1.6/2.2/2.4 still enumerate per-case bullets; carried to close review as Minor.
- PQ-2 — not-addressed — Task 1.6 step 4b still asserts the leaked-claim window in prose with no test pinning it.
- PQ-3 — not-addressed — Task 2.1 still patches menu.go case "" instead of deriving busy text from menuPhaseBusy.

## Open findings

- **PQ-1** [Minor] `test-prose-enumeration` Tasks 1.3/1.5/1.6/2.2/2.4 enumerate test cases in prose instead of one strategy line per risky function
- **PQ-2** [Minor] `unconfirmed-outcome-untested` The :0 reboot claim leak (death between Claim and journal) is asserted in prose but not pinned by a test
- **PQ-3** [Minor] `single-action-authority` Busy-row "starting elsewhere" is patched at menu.go case "" rather than derived from the action-table busy phase
