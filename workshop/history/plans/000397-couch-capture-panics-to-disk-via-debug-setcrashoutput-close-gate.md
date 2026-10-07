---
gate: boundary-review
issue: 397
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-06T16:30:56-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Deferred Capture.Close runs during panic unwinding on the console goroutine and deletes the crash file before the runtime writes the panic
          detail: run.go:409 defers installCrashReport(...).Close() in the same goroutine that runs console.Run's event loop (run.go:683). Go runs defers before printing a panic, so Close disables crash output and removes the still-empty file; verified with a re-exec scratch test (crash dir empty). Gate Close on a non-panicking exit (recover-and-repanic, or a clean flag) and add a re-exec regression test through the production wiring.
          family: defer-runs-before-panic-print
          round: 1
        - id: BR-2
          severity: Important
          title: A failure reporting an old crash file aborts Install before this run's capture is opened
          detail: crashreport.go:128-141 returns on any rename/remove error, so one stuck stale file disables crash capture on every subsequent start. Record the error as a notice and continue to open and install the current file.
          family: report-failure-blocks-capture
          round: 1
        - id: BR-3
          severity: Minor
          title: lease.Close also runs during panic unwinding, opening a tiny window where a relaunch misclassifies the dying run's empty .log as Abrupt
          family: lease-released-before-panic-print
          round: 1
        - id: BR-4
          severity: Minor
          title: Each previous crash or abrupt file posts its own standing control notice; several stale endings stack N notices
          family: notice-per-file-unbounded
          round: 1
        - id: BR-5
          severity: Minor
          title: gc Preview lists crash rows when migration is incomplete, but Apply skips them in that case
          family: preview-apply-parity
          round: 1
        - id: BR-6
          severity: Minor
          title: maxCrashRows in gcruntime restates crashreport.maxEntries (4096)
          family: restated-constant
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-06T16:46:55-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Code fixed (no defer in run.go; Finish after Run in main), but reintroducing defer crashreport.Finish() in run.go in a scratch copy leaves every crash test green; the regression test bypasses the production wiring.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Install collects per-file problems and still opens and installs the capture; TestUnreportableFileStillInstallsCapture fails against the old early return.
          round: 2
        - id: BR-3
          disposition: addressed
          note: A .log whose pid is alive is skipped; covered by TestInstallLeavesALiveOwnersFileAlone.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Summary folds all endings into one notice; TestSummary plus the combined assertion in TestInstallReportsPreviousEndingsOnce.
          round: 2
        - id: BR-5
          disposition: withdrawn
          note: gcruntime Apply returns before any diagnostics rows when migration is incomplete; crash rows follow the existing rows exactly.
          round: 2
        - id: BR-6
          disposition: addressed
          note: crashreport.MaxEntries is exported and gcruntime uses it; no restated constant remains.
          round: 2
      findings:
        - id: BR-7
          severity: Minor
          title: Sweep refuses a crash dir with more than MaxEntries entries, so pair gc can never drain it
          detail: list errors past MaxEntries (foreign files count) and Sweep returns that error with no rows, so neither startup nor gc makes progress. Sweep the first MaxEntries entries and still report the overflow.
          family: reader-cap-without-writer-bound
          round: 2
        - id: BR-8
          severity: Minor
          title: Install overwrites the global active capture without closing the previous one; existing console tests now leak installs
          detail: run_test and continuation_acceptance_test call runTypedOperationWithConsole with live-owning operations, which installs process-wide crash output into temp dirs and never calls Finish. Production installs once per process; this affects only tests.
          family: process-global-install-unpaired
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-06T17:00:35-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Finish now runs only after a normal return in main. Re-adding the defer in run.go makes TestConsoleOwnerPanicLeavesItsCrashFile fail (checked in a scratch copy).
          round: 3
        - id: BR-7
          disposition: addressed
          note: list now keeps the first MaxEntries and reports the overflow. TestSweepDrainsAnOverfullDirectory would fail under the old refuse-the-directory code.
          round: 3
        - id: BR-8
          disposition: addressed
          note: A second Install now ends the first (TestSecondInstallEndsTheFirst). Each test process can still leave one install pointing at a deleted temp file; harmless and test-only.
          round: 3
      findings:
        - id: BR-9
          severity: Minor
          title: installCrashReport shows "crash capture unavailable" even when Install returned a working capture
          detail: Since BR-2, Install can return an error from a failed rename or remove, or from the overflow check, while still returning a live capture. run.go:180 puts "unavailable" in front of every error. The notice should depend on whether a capture came back.
          family: notice-misstates-capture-state
          round: 3
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#397 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-06T16:30:56-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `defer-runs-before-panic-print` Deferred Capture.Close runs during panic unwinding on the console goroutine and deletes the crash file before the runtime writes the panic
  run.go:409 defers installCrashReport(...).Close() in the same goroutine that runs console.Run's event loop (run.go:683). Go runs defers before printing a panic, so Close disables crash output and removes the still-empty file; verified with a re-exec scratch test (crash dir empty). Gate Close on a non-panicking exit (recover-and-repanic, or a clean flag) and add a re-exec regression test through the production wiring.
- **BR-2** [Important] `report-failure-blocks-capture` A failure reporting an old crash file aborts Install before this run's capture is opened
  crashreport.go:128-141 returns on any rename/remove error, so one stuck stale file disables crash capture on every subsequent start. Record the error as a notice and continue to open and install the current file.
- **BR-3** [Minor] `lease-released-before-panic-print` lease.Close also runs during panic unwinding, opening a tiny window where a relaunch misclassifies the dying run's empty .log as Abrupt
- **BR-4** [Minor] `notice-per-file-unbounded` Each previous crash or abrupt file posts its own standing control notice; several stale endings stack N notices
- **BR-5** [Minor] `preview-apply-parity` gc Preview lists crash rows when migration is incomplete, but Apply skips them in that case
- **BR-6** [Minor] `restated-constant` maxCrashRows in gcruntime restates crashreport.maxEntries (4096)

## Round 2 — 2026-10-06T16:46:55-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Code fixed (no defer in run.go; Finish after Run in main), but reintroducing defer crashreport.Finish() in run.go in a scratch copy leaves every crash test green; the regression test bypasses the production wiring.
- BR-2 — addressed — Install collects per-file problems and still opens and installs the capture; TestUnreportableFileStillInstallsCapture fails against the old early return.
- BR-3 — addressed — A .log whose pid is alive is skipped; covered by TestInstallLeavesALiveOwnersFileAlone.
- BR-4 — addressed — Summary folds all endings into one notice; TestSummary plus the combined assertion in TestInstallReportsPreviousEndingsOnce.
- BR-5 — withdrawn — gcruntime Apply returns before any diagnostics rows when migration is incomplete; crash rows follow the existing rows exactly.
- BR-6 — addressed — crashreport.MaxEntries is exported and gcruntime uses it; no restated constant remains.

### Raised

- **BR-7** [Minor] `reader-cap-without-writer-bound` Sweep refuses a crash dir with more than MaxEntries entries, so pair gc can never drain it
  list errors past MaxEntries (foreign files count) and Sweep returns that error with no rows, so neither startup nor gc makes progress. Sweep the first MaxEntries entries and still report the overflow.
- **BR-8** [Minor] `process-global-install-unpaired` Install overwrites the global active capture without closing the previous one; existing console tests now leak installs
  run_test and continuation_acceptance_test call runTypedOperationWithConsole with live-owning operations, which installs process-wide crash output into temp dirs and never calls Finish. Production installs once per process; this affects only tests.

## Round 3 — 2026-10-06T17:00:35-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Finish now runs only after a normal return in main. Re-adding the defer in run.go makes TestConsoleOwnerPanicLeavesItsCrashFile fail (checked in a scratch copy).
- BR-7 — addressed — list now keeps the first MaxEntries and reports the overflow. TestSweepDrainsAnOverfullDirectory would fail under the old refuse-the-directory code.
- BR-8 — addressed — A second Install now ends the first (TestSecondInstallEndsTheFirst). Each test process can still leave one install pointing at a deleted temp file; harmless and test-only.

### Raised

- **BR-9** [Minor] `notice-misstates-capture-state` installCrashReport shows "crash capture unavailable" even when Install returned a working capture
  Since BR-2, Install can return an error from a failed rename or remove, or from the overflow check, while still returning a live capture. run.go:180 puts "unavailable" in front of every error. The notice should depend on whether a capture came back.

## Open findings

- **BR-9** [Minor] `notice-misstates-capture-state` installCrashReport shows "crash capture unavailable" even when Install returned a working capture
