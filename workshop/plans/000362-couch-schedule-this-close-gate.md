---
gate: boundary-review
issue: 362
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-06T20:07:54-07:00"
      agent: claude
      recipe: milestone-review
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-10-06T20:21:14-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: peek's "console must not dispatch it" rule is a comment; the console dispatcher would run it
          detail: ops.go peek declaration vs run.go:747-750 console SetOperationDispatcher wiring DirectStoreExecutor; enforce via a declared field or console refusal plus a test (ARCH-CONSTRAINTS).
          family: declared-invariant-unenforced
          round: 2
        - id: BR-2
          severity: Minor
          title: Exercise 2 ends at expiry; the planned clear-then-submit or resend-after-expiry outcome is not logged
          detail: Revision (b) planned to show either a submit after clearing the draft or a safe resend after expiry; the Log stops at the expiry.
          family: done-when-evidence-partial
          round: 2
        - id: BR-3
          severity: Minor
          title: The generic --json branch in runTypedOperationWithConsole and the --lines refusal have no run-path tests
          detail: TestPeekJSONIsTheResult only marshals the struct; add a RunWithRuntime --peek --json test and a non-positive --lines dispatch test.
          family: cli-path-untested
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#362 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T20:07:54-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-10-06T20:21:14-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `declared-invariant-unenforced` peek's "console must not dispatch it" rule is a comment; the console dispatcher would run it
  ops.go peek declaration vs run.go:747-750 console SetOperationDispatcher wiring DirectStoreExecutor; enforce via a declared field or console refusal plus a test (ARCH-CONSTRAINTS).
- **BR-2** [Minor] `done-when-evidence-partial` Exercise 2 ends at expiry; the planned clear-then-submit or resend-after-expiry outcome is not logged
  Revision (b) planned to show either a submit after clearing the draft or a safe resend after expiry; the Log stops at the expiry.
- **BR-3** [Minor] `cli-path-untested` The generic --json branch in runTypedOperationWithConsole and the --lines refusal have no run-path tests
  TestPeekJSONIsTheResult only marshals the struct; add a RunWithRuntime --peek --json test and a non-positive --lines dispatch test.

## Open findings

- **BR-1** [Minor] `declared-invariant-unenforced` peek's "console must not dispatch it" rule is a comment; the console dispatcher would run it
- **BR-2** [Minor] `done-when-evidence-partial` Exercise 2 ends at expiry; the planned clear-then-submit or resend-after-expiry outcome is not logged
- **BR-3** [Minor] `cli-path-untested` The generic --json branch in runTypedOperationWithConsole and the --lines refusal have no run-path tests
