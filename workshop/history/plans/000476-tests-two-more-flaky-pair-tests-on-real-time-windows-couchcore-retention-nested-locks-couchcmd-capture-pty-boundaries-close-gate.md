---
gate: boundary-review
issue: 476
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T05:35:04-07:00"
      dispatched: "2026-10-12T05:33:43-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Retention nested-lock test no longer pins the magnitude of the yield bound
          detail: retention_test.go:603 waits on done with no window. A regression that keeps polling for a long but bounded time before returning ErrCoordinatorBusy now passes; only an unbounded block is caught, at the binary timeout. If the bound matters, pin the poll-bound constant directly with no wall clock. Only instance in this window.
          family: test-subject-bound-unpinned
          round: 1
        - id: BR-2
          severity: Minor
          title: waitForDrain constant is defined in two test packages
          detail: couchcmd/capture_test.go:87 and terminalcapture/recorder_test.go:36 (ARCH-DRY). An exported terminalcapture sentinel (for example NoCloseTimeout) would give it one home. These are the only two instances.
          family: duplicated-test-helper
          round: 1
        - id: BR-3
          severity: Minor
          title: Unwrapped long line in atlas/couch-live-capture.md:109
          detail: The added parenthetical breaks the file's line-wrap convention. This is the only instance.
          family: doc-line-wrap
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: dafc05aada3cce72a18e5ca6b58de78685b1991e
      blocked: false
---

# Gate ledger — pair#476 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T05:35:04-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `test-subject-bound-unpinned` Retention nested-lock test no longer pins the magnitude of the yield bound
  retention_test.go:603 waits on done with no window. A regression that keeps polling for a long but bounded time before returning ErrCoordinatorBusy now passes; only an unbounded block is caught, at the binary timeout. If the bound matters, pin the poll-bound constant directly with no wall clock. Only instance in this window.
- **BR-2** [Minor] `duplicated-test-helper` waitForDrain constant is defined in two test packages
  couchcmd/capture_test.go:87 and terminalcapture/recorder_test.go:36 (ARCH-DRY). An exported terminalcapture sentinel (for example NoCloseTimeout) would give it one home. These are the only two instances.
- **BR-3** [Minor] `doc-line-wrap` Unwrapped long line in atlas/couch-live-capture.md:109
  The added parenthetical breaks the file's line-wrap convention. This is the only instance.

## Open findings

- **BR-1** [Minor] `test-subject-bound-unpinned` Retention nested-lock test no longer pins the magnitude of the yield bound
- **BR-2** [Minor] `duplicated-test-helper` waitForDrain constant is defined in two test packages
- **BR-3** [Minor] `doc-line-wrap` Unwrapped long line in atlas/couch-live-capture.md:109
