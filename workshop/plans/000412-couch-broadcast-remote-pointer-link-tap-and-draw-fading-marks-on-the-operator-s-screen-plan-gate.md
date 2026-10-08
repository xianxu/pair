---
gate: plan-quality
issue: 412
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-08T09:27:06-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Overlay closure's mutex is unnamed; it runs on the Presenter goroutine, so it must be a leaf lock never held across presenter/session calls
          detail: 'If Marks shares c.mu (or any lock held while calling p.call synchronously), paintPublication deadlocks the operator''s terminal. State: Marks has its own leaf mutex; OnPoints is invoked from the HTTP handler outside the session mutex (runTerminalCommand blocks the caller until the Console loop runs it, and the loop calls DisablePointer).'
          family: unstated-lock-contract
          round: 1
        - id: PQ-2
          severity: Minor
          title: Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
          detail: Compress to the one-line-per-risky-function strategy already in the Test strategy section.
          family: enumerated-test-prose
          round: 1
        - id: PQ-3
          severity: Minor
          title: Whether the overlay draws pre-existing marks on a FramePrivate panel is undecided
          detail: Points are dropped while the switcher is open, but older marks still overlay the private panel on the operator's screen. Decide whether the overlay skips FramePrivate, and test that decision.
          family: undecided-edge-state
          round: 1
        - id: PQ-4
          severity: Minor
          title: Server has one immutable opts.Token and Session has no mutex; name the seam carrying pointer state into the server
          detail: server.go:57-87 and session.go:44-56. Name the interface (e.g. a PointerGate) the routes query for token, pointing, Current() and OnPoints.
          family: unnamed-seam
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-08T09:27:54-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Lock discipline section: marksMu is a leaf separate from c.mu, PointerState.mu is a leaf, callbacks run after release, and a -race stress test has a deadline.'
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: 'Tasks 1.3/2.2/2.3/3.2 still list cases in prose. Rule: a task''s test step names the functions under test and points to its Test strategy line; it does not list cases. Minor, carried to the close review.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: The overlay returns a FramePrivate frame unchanged and marks keep ageing; Task 1.3 tests that the overlay receives FramePrivate.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: PointerState (mu, token, pointing) is the server's only seam, through Match/On; the view token stays immutable in ServerOptions.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-08T09:28:35-07:00"
      agent: claude
      dispose:
        - id: PQ-2
          disposition: not-addressed
          note: Tasks 1.3, 2.2, 2.3, 3.2 still enumerate cases; Minor, carried to close review.
          round: 3
      blocked: false
content_hash: a0a033552e09a8d66b508881b992c39e8ed3c360475eb5a74dd0f620a7a4e848
---

# Gate ledger — pair#412 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T09:27:06-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unstated-lock-contract` Overlay closure's mutex is unnamed; it runs on the Presenter goroutine, so it must be a leaf lock never held across presenter/session calls
  If Marks shares c.mu (or any lock held while calling p.call synchronously), paintPublication deadlocks the operator's terminal. State: Marks has its own leaf mutex; OnPoints is invoked from the HTTP handler outside the session mutex (runTerminalCommand blocks the caller until the Console loop runs it, and the loop calls DisablePointer).
- **PQ-2** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
  Compress to the one-line-per-risky-function strategy already in the Test strategy section.
- **PQ-3** [Minor] `undecided-edge-state` Whether the overlay draws pre-existing marks on a FramePrivate panel is undecided
  Points are dropped while the switcher is open, but older marks still overlay the private panel on the operator's screen. Decide whether the overlay skips FramePrivate, and test that decision.
- **PQ-4** [Minor] `unnamed-seam` Server has one immutable opts.Token and Session has no mutex; name the seam carrying pointer state into the server
  server.go:57-87 and session.go:44-56. Name the interface (e.g. a PointerGate) the routes query for token, pointing, Current() and OnPoints.

## Round 2 — 2026-10-08T09:27:54-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Lock discipline section: marksMu is a leaf separate from c.mu, PointerState.mu is a leaf, callbacks run after release, and a -race stress test has a deadline.
- PQ-2 — not-addressed — Tasks 1.3/2.2/2.3/3.2 still list cases in prose. Rule: a task's test step names the functions under test and points to its Test strategy line; it does not list cases. Minor, carried to the close review.
- PQ-3 — addressed — The overlay returns a FramePrivate frame unchanged and marks keep ageing; Task 1.3 tests that the overlay receives FramePrivate.
- PQ-4 — addressed — PointerState (mu, token, pointing) is the server's only seam, through Match/On; the view token stays immutable in ServerOptions.

## Round 3 — 2026-10-08T09:28:35-07:00 (claude) — passed

### Disposed

- PQ-2 — not-addressed — Tasks 1.3, 2.2, 2.3, 3.2 still enumerate cases; Minor, carried to close review.

## Open findings

- **PQ-2** [Minor] `enumerated-test-prose` Tasks 1.3, 2.2, 2.3, 3.2 enumerate test cases in prose, restating Done-when
