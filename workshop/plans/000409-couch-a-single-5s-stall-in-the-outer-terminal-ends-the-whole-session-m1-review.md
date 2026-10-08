# Boundary Review — pair#409 (milestone M1)

| field | value |
|-------|-------|
| issue | 409 — couch: full-screen repaint per frame overruns a backgrounded terminal, and the resulting exit is silent |
| repo | pair |
| issue file | workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4ba09dce8b10baf3b0351075dc69bbc5d1d5cc43..0f43f229f69bec717bff140b4739c627fd64ffbd |
| command | sdlc milestone-close --issue 409 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T22:03:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 does what it set out to do. A parent-output `WriteFailure` becomes one wording (`terminal.ExitReason`). couchcmd writes it into the run's existing crash file through `crashreport.RecordExit`, and the next `Install` reports it once as a new `Exited` kind, separately from panics. Tests pass when run outside the sandbox: `crashreport`, `couchcmd` and the terminal `ExitReason`/stall tests; `go vet` is clean. The atlas is updated, and the README lists no startup notices, so it has nothing to update.

One problem should be fixed first. `Console.TerminalFailure()` hands back the presenter's latched failure without the shutdown-cancellation filter that `teardown` applies to the same value. A parent-output write cancelled by an ordinary shutdown can then be recorded as "previous couch exited: … context canceled". The couchcmd test passes a hand-built error to `recordConsoleExit`, so `TerminalFailure()` itself is never exercised, which is how this got through.

**1. Strengths**
- **One source for the wording.** `ExitReason` (`cmd/internal/terminal/transport.go:27-40`) is the only place it lives, and it takes the duration from `WriteTimeout` rather than a literal "5s".
- **No new storage.** It reuses #397's per-run file, its rename-on-report rule (`crashreport.go:203`) and the `pair gc` sweep. Nothing new is created that would need cleaning up (ARCH-FUNERAL passes).
- **Layering holds.** couchtty doesn't import crashreport; couchcmd glues the two together (`run.go:733-745`). `Classify` stays pure, with the IO kept to a bounded `readHead`.
- **Report wording.** `Summary` folds several exits into one sentence with a "(+N earlier)" count, and the table tests cover mixing exits with crashes.
- **Real stall evidence.** `TestAStalledParentReadsAsTheTerminalStopping` connects a real stalled parent to the `ExitReason` wording.

**2. Critical:** none.

**3. Important**
- **A cancelled write at shutdown can be reported as an exit** (`cmd/internal/couchtty/terminal.go:74-82`, used by `run.go:734`).
  - `Presenter.write` returns `WriteFailure{Op:"parent output", Err: context.Canceled}` when the caller's context (the console lifetime) is cancelled. `writeComplete` checks `ctx.Err()` first.
  - `Presenter.fail` latches that failure whenever `p.life` hasn't been cancelled yet.
  - `teardown` (`console.go:973-981`) deliberately drops a latched shutdown cancellation (`c.lifetime.Err() != nil && shutdownCancellation(...)`). `TerminalFailure()` joins the presenter's failure without that filter.
  - `ExitReason` accepts any parent-output failure that isn't a deadline. On SIGTERM or a quit that lands during a paint, it records "terminal output failed after 0 of N bytes: context canceled", and the next start shows that false notice.
  - **Fix:** classify the failure in one place (ARCH-DRY). Have `teardown` save its already-filtered `failure + presenterFailure` on the console and have `TerminalFailure()` return that. Or apply the same `shutdownCancellation` filter there, and also have `ExitReason` refuse `context.Canceled`.
  - **Test:** build a console, latch a parent-output `WriteFailure{Err: context.Canceled}` after the lifetime is cancelled, and assert `ExitReason(console.TerminalFailure())` is not ok. Add a case where a real stall *is* reported through `TerminalFailure()`, so the accessor itself is covered.

**4. Minor**
- **Deadline wording.** `ExitReason` words every `DeadlineExceeded` as "for 5s", even when a shorter caller deadline caused it. Production paints don't appear to use one, but the wording claims more than the error proves.
- **A later panic would be hidden.** If the process panics after `RecordExit` (late in `main`), the panic text lands after the marker line. `Classify` then calls the file `Exited`, the panic is hidden, and the summary doesn't name the file. This is unlikely; a later fix could prefer `Crashed` when the head contains `panic:` / `goroutine `.
- **Sweep reads file heads it doesn't use.** `list` now reads up to 256 bytes of every non-empty `.log` file during `Sweep` as well. That is bounded by `MaxEntries`, so it is harmless.
- **Test isolation.** `TestRecordedExitIsReportedOnceWithItsReason` ends with a bare `Finish()`, so if an assertion fails partway, the global capture state leaks into later tests.

**5. Test coverage**
- The pure parts are well covered: `Classify`, `Summary`, `ExitReason` (including joined and wrapped errors, broken pipe and child input), and `RecordExit` with no capture installed.
- The link from a stall to a recorded exit is proven in two halves, as Revision (a) PQ-1 set out. But the console half never calls `Console.TerminalFailure()`, which is where the Important bug sits.

**6. Architecture**
- **ARCH-DRY: flagged.** Two places decide whether a terminal failure counts (`teardown` and `TerminalFailure`); see Important.
- **ARCH-PURE: pass.** `Classify`, `Summary` and `ExitReason` are pure; the IO is confined to `readHead` and `RecordExit`.
- **ARCH-PURPOSE: pass.** The exit is no longer silent, and couchcmd is the only producer of a terminal exit.
- **ARCH-MOCK: pass.** The stall test uses the existing stalled-parent harness.
- **ARCH-CONSTRAINTS: pass.** Reads are capped at 256 B per file and 4096 files.
- **ARCH-SECURE: pass.** The file is self-written but untrusted when read back. The read is bounded, the reason is cut at the first newline, and a malformed head degrades to `Crashed`.
- **ARCH-ORDER: flagged**, as part of the Important finding. The shutdown/failure ordering that `teardown` handles is ignored by the new accessor.
- **ARCH-FUNERAL: pass.**

**7. Plan revisions:** none required. Optionally, add a note to task 1.3 that `TerminalFailure()` returns teardown's filtered classification.

```findings
findings:
  - id: new
    severity: Important
    family: terminal-failure-classified-twice
    title: |
      Console.TerminalFailure skips teardown's shutdown-cancellation filter, so a cancelled shutdown write is recorded as an exit
    detail: |
      Presenter.write wraps a lifetime-cancelled write as WriteFailure{parent output, context.Canceled}, and Presenter.fail latches it while p.life is live. teardown (console.go:973-981) filters that case, but TerminalFailure (terminal.go:74-82) does not, and ExitReason accepts non-deadline errors. On SIGTERM or a quit during a paint the next start shows a false "previous couch exited: ... context canceled". Fix: have TerminalFailure return teardown's filtered failure (one classification), and/or have ExitReason refuse context.Canceled. Add a test that goes through Console.TerminalFailure, which no current test calls.
  - id: new
    severity: Minor
    family: exit-marker-masks-later-panic
    title: |
      A panic written after RecordExit is classified Exited, and its path is not named in the summary
  - id: new
    severity: Minor
    family: exit-reason-overclaims-cause
    title: |
      ExitReason words every DeadlineExceeded as stopped for WriteTimeout, including shorter caller deadlines
```

---

## Re-review — 2026-10-07T22:07:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 409 — couch: full-screen repaint per frame overruns a backgrounded terminal, and the resulting exit is silent |
| repo | pair |
| issue file | workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4ba09dce8b10baf3b0351075dc69bbc5d1d5cc43..dfd184efa522e295834a08b259ca4938a6ba1dff |
| command | sdlc milestone-close --issue 409 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-07T22:07:16-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 does what the plan says: there is a recorded-exit kind (`Exited`) in `crashreport`, `RecordExit` writes into the per-run file from #397, `terminal.ExitReason` holds the one wording, and couchcmd's hook in `runConsole` records the reason after `console.Run()` returns. The round-1 blocker (BR-1) is fixed at its root. `Console.TerminalFailure()` now returns `c.exitFailure`. Teardown sets it from the same `errors.Join(failure, presenterFailure, cleanupErr)` it reports, after filtering out shutdown cancellations, so the console has only one classification. The regression test sits in `TestConsoleStopDuringPaintClassifiesJoinedFailure`. Before the fix, `TerminalFailure` joined `presenter.Failure()` without filtering. In the `host-error=false` case that returned `WriteFailure{parent output, Canceled}`, `ExitReason` accepted it and the test assertion would fail. So the test proves the fix. Tests I ran at `dfd184ef`: `crashreport`, `couchtty` (inside the sandbox), and `terminal` plus `TestConsoleTerminalFailureIsRecordedAsTheExitReason` (outside the sandbox, because they need a pty and `/tmp`) all pass. `TestMessageWrapperExitDisconnectsAndForgetsWorkspace` fails because the agent shell leaks `PAIR_*` variables into the test environment (63 are set). That file is untouched by this diff. Nothing blocks SHIP. Only Minor notes remain.

1. **Strengths**
   - `console.go:985-987`: teardown stores the joined, filtered error that it also prints and returns. The exit reason and the stderr line can no longer disagree, which is the rule-level fix for family `terminal-failure-classified-twice`, not just the one site.
   - `crashreport.go:105-108`: `Classify` stays pure over `File{Name, Size, Head}`. The only new IO is the bounded `readHead` (256 bytes) in `list`, and an unreadable file degrades to `Crashed`, which keeps the file visible.
   - `Install` renames `Exited` files to `.crash` exactly as it does crashes, so nothing new needs collecting. Report-once and the `pair gc` retention are reused (ARCH-FUNERAL, ARCH-DRY).
   - `RecordExit` replaces newlines and calls `Sync`, and `Finish` is still called only on the normal-return path (`cmd/couch/main.go:19`). A recorded reason survives the exit, and a panic still leaves its file.
   - `TestAStalledParentReadsAsTheTerminalStopping` connects the stall harness to `ExitReason` without changing the `WriteTimeout` const (plan revision PQ-1).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `atlas/couch.md:2016-2018`: the continuation lines of the new bullet lost their two-space indent, so the list item is inconsistent with its neighbours.
   - The plan's Core-concepts description of `ExitReason` ("terminal stopped accepting output for 5s (wrote A of N bytes: <cause>)") no longer matches the shipped wording ("a write waited up to 5s; wrote A of N bytes"). The M1 task boxes 1.1–1.4 are also still unticked.
   - ARCH-SECURE, note only: `Reason` is text read from a file under the store, and it reaches a notice body in the status row. The directory is the user's own and only this program writes the marker, so the risk is negligible. If notice bodies are ever written raw, strip control bytes in `Classify`.

5. **Test coverage notes**
   - These four paths are all covered:
     - classification, including the marker with a trailing panic text;
     - `Summary` composition when exits and crashes are both present;
     - report-once across three installs;
     - the no-capture no-op.
   - Two things are proved separately rather than end to end: a stall producing the typed failure, and the typed failure being recorded and reported. That is acceptable under PQ-1, since a single test would need an injectable `WriteTimeout`.
   - The BR-1 regression test sits in the existing stop-during-paint matrix, so it covers both `partial` modes.

6. **Architectural notes**
   - ARCH-DRY: pass. One wording site (`ExitReason`), one classification (teardown), one file family.
   - ARCH-PURE: pass. `Classify`, `Summary` and `ExitReason` are pure, and the IO is limited to `readHead` and `RecordExit`.
   - ARCH-PURPOSE: pass. The Done-when item for M1 is met by the two-proof split.
   - ARCH-MOCK: pass. The tests use the existing `FakeHost` and stalled-parent fakes, and no new external dependency is added.
   - ARCH-CONSTRAINTS: pass. One 256-byte read per non-empty `.log` at startup, bounded by `MaxEntries`.
   - ARCH-SECURE: pass, with the note above.
   - ARCH-ORDER: pass. `exitFailure` is written once, after workers join and the presenter releases, and it is nil before then (documented on the accessor).
   - ARCH-FUNERAL: pass. No new artifact; recorded exits follow the existing rename and gc sweep.

7. **Plan revision recommendations**
   - Add a short Revisions entry recording the final `ExitReason` wording ("a write waited up to WriteTimeout"), which answers BR-3. Also record that `TerminalFailure` reads teardown's stored classification rather than re-joining the presenter failure, which answers BR-1.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TerminalFailure returns teardown's stored filtered join (console.go:985); the stop-during-paint matrix now asserts ExitReason(c.TerminalFailure()) and would fail on the pre-fix unfiltered join.
  - id: BR-2
    disposition: addressed
    note: |
      Summary now appends the file path to the exited notice, and the atlas documents that a panic during the exit lands in the same file.
  - id: BR-3
    disposition: addressed
    note: |
      Wording changed to "a write waited up to <WriteTimeout>", which is true for shorter caller deadlines too; ExitReason's doc comment states it.
findings:
  - id: new
    severity: Minor
    family: plan-drifts-from-shipped-surface
    title: |
      Plan Core-concepts ExitReason wording and unticked M1 task boxes no longer match the code
    detail: |
      Plan says "for 5s (wrote A of N bytes: <cause>)"; code says "a write waited up to 5s; wrote A of N bytes". Add a Revisions entry and tick 1.1-1.4.
  - id: new
    severity: Minor
    family: atlas-list-formatting
    title: |
      atlas/couch.md recorded-exit bullet loses its continuation indent mid-item
```
