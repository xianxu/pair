---
gate: plan-quality
issue: 365
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-01T13:32:29-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Dormant sessions re-admit only on attach/submit/reconnect, not when a send targets them
          detail: After 5 failed admissions an idle-but-healthy wrapper stays unreachable for sends until the operator touches its pane; a send to a Dormant exact target should poke one bounded re-admission (still per-session, not global).
          family: recovery-trigger-misses-primary-use
          round: 1
        - id: PQ-2
          severity: Minor
          title: Family retry status(id) fan-out to every candidate has no latency/concurrency budget
          detail: N endpoint RPCs per family send with an unknown ID must fit the 2 s client deadline; state concurrency and per-call timeout, and say how a candidate timing out is treated (unknown, not absent).
          family: undeclared-request-path-budget
          round: 1
        - id: PQ-3
          severity: Minor
          title: Registry per-thread pane entries have no stated removal on PaneExited
          detail: In-memory and bounded by threads, but say that PaneExited without a successor deletes the thread entry.
          family: in-memory-structure-removal-unstated
          round: 1
        - id: PQ-4
          severity: Minor
          title: Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
          detail: Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.
          family: plan-enumerates-test-cases
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-10-01T13:34:09-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: SendTargeted re-admission added; also list SendTargeted in the RegistryEvent table and the ScheduleRetry Dormant sentence.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Concurrency 4, 300 ms per-call timeout, timeout counts as unknown and the send answers uncertain.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: PaneExited with no successor deletes the thread entry.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: 'Task 2.1 (13 cases) and Task 3.1 still enumerate cases. Rule: each test task names its functions plus one strategy line per risky function (permuted event sequences through Advance with invariants asserted); apply it to 2.1, 2.5 and 3.1 together.'
          round: 2
      blocked: false
content_hash: c20e09fb7adc82160b22a9c6831b07cf483e7e0d1e926f316b3f507daa8d81ce
---

# Gate ledger — pair#365 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-01T13:32:29-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `recovery-trigger-misses-primary-use` Dormant sessions re-admit only on attach/submit/reconnect, not when a send targets them
  After 5 failed admissions an idle-but-healthy wrapper stays unreachable for sends until the operator touches its pane; a send to a Dormant exact target should poke one bounded re-admission (still per-session, not global).
- **PQ-2** [Minor] `undeclared-request-path-budget` Family retry status(id) fan-out to every candidate has no latency/concurrency budget
  N endpoint RPCs per family send with an unknown ID must fit the 2 s client deadline; state concurrency and per-call timeout, and say how a candidate timing out is treated (unknown, not absent).
- **PQ-3** [Minor] `in-memory-structure-removal-unstated` Registry per-thread pane entries have no stated removal on PaneExited
  In-memory and bounded by threads, but say that PaneExited without a successor deletes the thread entry.
- **PQ-4** [Minor] `plan-enumerates-test-cases` Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
  Drive permuted event sequences through Advance and assert invariants (no stale token/pane connects; newest token wins; displaced never resurrects) — enumeration misses orderings the generator would cover.

## Round 2 — 2026-10-01T13:34:09-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — SendTargeted re-admission added; also list SendTargeted in the RegistryEvent table and the ScheduleRetry Dormant sentence.
- PQ-2 — addressed — Concurrency 4, 300 ms per-call timeout, timeout counts as unknown and the send answers uncertain.
- PQ-3 — addressed — PaneExited with no successor deletes the thread entry.
- PQ-4 — not-addressed — Task 2.1 (13 cases) and Task 3.1 still enumerate cases. Rule: each test task names its functions plus one strategy line per risky function (permuted event sequences through Advance with invariants asserted); apply it to 2.1, 2.5 and 3.1 together.

## Open findings

- **PQ-4** [Minor] `plan-enumerates-test-cases` Task 2.1 enumerates 13 test cases in prose; compress to a strategy line
