---
gate: boundary-review
issue: 409
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T22:03:51-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Console.TerminalFailure skips teardown's shutdown-cancellation filter, so a cancelled shutdown write is recorded as an exit
          detail: 'Presenter.write wraps a lifetime-cancelled write as WriteFailure{parent output, context.Canceled}, and Presenter.fail latches it while p.life is live. teardown (console.go:973-981) filters that case, but TerminalFailure (terminal.go:74-82) does not, and ExitReason accepts non-deadline errors. On SIGTERM or a quit during a paint the next start shows a false "previous couch exited: ... context canceled". Fix: have TerminalFailure return teardown''s filtered failure (one classification), and/or have ExitReason refuse context.Canceled. Add a test that goes through Console.TerminalFailure, which no current test calls.'
          family: terminal-failure-classified-twice
          round: 1
        - id: BR-2
          severity: Minor
          title: A panic written after RecordExit is classified Exited, and its path is not named in the summary
          family: exit-marker-masks-later-panic
          round: 1
        - id: BR-3
          severity: Minor
          title: ExitReason words every DeadlineExceeded as stopped for WriteTimeout, including shorter caller deadlines
          family: exit-reason-overclaims-cause
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T22:07:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TerminalFailure returns teardown's stored filtered join (console.go:985); the stop-during-paint matrix now asserts ExitReason(c.TerminalFailure()) and would fail on the pre-fix unfiltered join.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Summary now appends the file path to the exited notice, and the atlas documents that a panic during the exit lands in the same file.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Wording changed to "a write waited up to <WriteTimeout>", which is true for shorter caller deadlines too; ExitReason's doc comment states it.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: Plan Core-concepts ExitReason wording and unticked M1 task boxes no longer match the code
          detail: 'Plan says "for 5s (wrote A of N bytes: <cause>)"; code says "a write waited up to 5s; wrote A of N bytes". Add a Revisions entry and tick 1.1-1.4.'
          family: plan-drifts-from-shipped-surface
          round: 2
        - id: BR-5
          severity: Minor
          title: atlas/couch.md recorded-exit bullet loses its continuation indent mid-item
          family: atlas-list-formatting
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 3
      timestamp: "2026-10-07T23:27:36-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: Revision (b) records the as-built ExitReason wording ("a write waited up to 5s") and that 1.1-1.4 are done; the issue Plan ticks M1/M2.
          round: 3
        - id: BR-5
          disposition: addressed
          note: atlas/couch.md recorded-exit bullet now keeps a two-space continuation indent on every line.
          round: 3
      findings:
        - id: BR-6
          severity: Minor
          title: Row-diff fast path also runs for frames that stay in the alt screen, but the generative test only builds primary-screen frames
          detail: history_render.go:556 gates on AltScreen equality, not on primary-only. Add alt-screen base frames to TestHistoryRowDiffEqualsFullRebuild.
          family: rowdiff-coverage-alt-screen
          round: 3
        - id: BR-7
          severity: Minor
          title: Row diff no longer repaints the whole visible screen on a dirty frame, so an outside clear of the host stays blank except for changed rows
          detail: Before, every non-reset dirty frame rebuilt all lower rows. If this is intended, note it in atlas/terminal.md next to changedPlainRows.
          family: rowdiff-no-self-heal
          round: 3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#409 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T22:03:51-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `terminal-failure-classified-twice` Console.TerminalFailure skips teardown's shutdown-cancellation filter, so a cancelled shutdown write is recorded as an exit
  Presenter.write wraps a lifetime-cancelled write as WriteFailure{parent output, context.Canceled}, and Presenter.fail latches it while p.life is live. teardown (console.go:973-981) filters that case, but TerminalFailure (terminal.go:74-82) does not, and ExitReason accepts non-deadline errors. On SIGTERM or a quit during a paint the next start shows a false "previous couch exited: ... context canceled". Fix: have TerminalFailure return teardown's filtered failure (one classification), and/or have ExitReason refuse context.Canceled. Add a test that goes through Console.TerminalFailure, which no current test calls.
- **BR-2** [Minor] `exit-marker-masks-later-panic` A panic written after RecordExit is classified Exited, and its path is not named in the summary
- **BR-3** [Minor] `exit-reason-overclaims-cause` ExitReason words every DeadlineExceeded as stopped for WriteTimeout, including shorter caller deadlines

## Round 2 — 2026-10-07T22:07:16-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TerminalFailure returns teardown's stored filtered join (console.go:985); the stop-during-paint matrix now asserts ExitReason(c.TerminalFailure()) and would fail on the pre-fix unfiltered join.
- BR-2 — addressed — Summary now appends the file path to the exited notice, and the atlas documents that a panic during the exit lands in the same file.
- BR-3 — addressed — Wording changed to "a write waited up to <WriteTimeout>", which is true for shorter caller deadlines too; ExitReason's doc comment states it.

### Raised

- **BR-4** [Minor] `plan-drifts-from-shipped-surface` Plan Core-concepts ExitReason wording and unticked M1 task boxes no longer match the code
  Plan says "for 5s (wrote A of N bytes: <cause>)"; code says "a write waited up to 5s; wrote A of N bytes". Add a Revisions entry and tick 1.1-1.4.
- **BR-5** [Minor] `atlas-list-formatting` atlas/couch.md recorded-exit bullet loses its continuation indent mid-item

## Round 3 — 2026-10-07T23:27:36-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — Revision (b) records the as-built ExitReason wording ("a write waited up to 5s") and that 1.1-1.4 are done; the issue Plan ticks M1/M2.
- BR-5 — addressed — atlas/couch.md recorded-exit bullet now keeps a two-space continuation indent on every line.

### Raised

- **BR-6** [Minor] `rowdiff-coverage-alt-screen` Row-diff fast path also runs for frames that stay in the alt screen, but the generative test only builds primary-screen frames
  history_render.go:556 gates on AltScreen equality, not on primary-only. Add alt-screen base frames to TestHistoryRowDiffEqualsFullRebuild.
- **BR-7** [Minor] `rowdiff-no-self-heal` Row diff no longer repaints the whole visible screen on a dirty frame, so an outside clear of the host stays blank except for changed rows
  Before, every non-reset dirty frame rebuilt all lower rows. If this is intended, note it in atlas/terminal.md next to changedPlainRows.

## Open findings

- **BR-6** [Minor] `rowdiff-coverage-alt-screen` Row-diff fast path also runs for frames that stay in the alt screen, but the generative test only builds primary-screen frames
- **BR-7** [Minor] `rowdiff-no-self-heal` Row diff no longer repaints the whole visible screen on a dirty frame, so an outside clear of the host stays blank except for changed rows
