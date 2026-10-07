---
gate: plan-quality
issue: 205
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-06T23:00:38-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Task 8 assumes Leave is the only park-worker submitter; other parks at the same time overload a capacity-4 worker
          detail: Parks from continuations (continuation.go:296, :246; continuation_recovery.go:261), operator/remote park jobs (operationdispatch.go:368-374) and RecoverActiveParks share parkWorker with Leave. Leave's semaphore of 4 plus any one of them exceeds capacity 4, and parkworker.go:74 refuses with ErrParkWorkerOverloaded, so quit fails a thread. Replace "Then decide" with a committed policy (size for all submitters, wait on capacity, or leaveOne retries) and a test with a non-Leave park running at the same time.
          family: shared-bound-assumed-exclusive
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 3 Step 1 spells out test cases as numbered prose scripts
          detail: Compress to test names plus one strategy line per risky function. The code will state the cases better.
          family: test-prose-enumeration
          round: 1
        - id: PQ-3
          severity: Minor
          title: 'ThreadGate held by value in Couch: any copy of Couch splits the gate'
          detail: Check that go vet copylocks is clean after Task 1, or hold a pointer created lazily.
          family: lock-by-value-copy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-06T23:01:26-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Task 8 states one rule for all submitters: capacity makes callers wait (ctx-bounded) and only a duplicate address is refused. It applies to every bounded resource and is tested with a non-Leave park at the same time.'
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: 'Task 3 Step 1 still writes three tests as numbered prose scripts. Minor, carried to close. Rule: one line per test giving its name and invariant.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The gate is a pointer created lazily via gateOnce, and Task 1 requires a clean go vet copylocks run.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-06T23:02:34-07:00"
      agent: claude
      dispose:
        - id: PQ-2
          disposition: addressed
          note: Task 3 Step 1 now gives test names, each with its asserted invariant; the remaining setup clauses are one line each, not numbered scripts.
          round: 3
      blocked: false
content_hash: f5a1f64121c144dac2e9ef2b0036968e719a58379eb38551c52731786f3d5695
---

# Gate ledger — pair#205 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T23:00:38-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `shared-bound-assumed-exclusive` Task 8 assumes Leave is the only park-worker submitter; other parks at the same time overload a capacity-4 worker
  Parks from continuations (continuation.go:296, :246; continuation_recovery.go:261), operator/remote park jobs (operationdispatch.go:368-374) and RecoverActiveParks share parkWorker with Leave. Leave's semaphore of 4 plus any one of them exceeds capacity 4, and parkworker.go:74 refuses with ErrParkWorkerOverloaded, so quit fails a thread. Replace "Then decide" with a committed policy (size for all submitters, wait on capacity, or leaveOne retries) and a test with a non-Leave park running at the same time.
- **PQ-2** [Minor] `test-prose-enumeration` Task 3 Step 1 spells out test cases as numbered prose scripts
  Compress to test names plus one strategy line per risky function. The code will state the cases better.
- **PQ-3** [Minor] `lock-by-value-copy` ThreadGate held by value in Couch: any copy of Couch splits the gate
  Check that go vet copylocks is clean after Task 1, or hold a pointer created lazily.

## Round 2 — 2026-10-06T23:01:26-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 8 states one rule for all submitters: capacity makes callers wait (ctx-bounded) and only a duplicate address is refused. It applies to every bounded resource and is tested with a non-Leave park at the same time.
- PQ-2 — not-addressed — Task 3 Step 1 still writes three tests as numbered prose scripts. Minor, carried to close. Rule: one line per test giving its name and invariant.
- PQ-3 — addressed — The gate is a pointer created lazily via gateOnce, and Task 1 requires a clean go vet copylocks run.

## Round 3 — 2026-10-06T23:02:34-07:00 (claude) — passed

### Disposed

- PQ-2 — addressed — Task 3 Step 1 now gives test names, each with its asserted invariant; the remaining setup clauses are one line each, not numbered scripts.

## Open findings

(none — every finding has been disposed)
