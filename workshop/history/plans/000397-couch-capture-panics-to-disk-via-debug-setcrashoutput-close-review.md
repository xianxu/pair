# Boundary Review — pair#397 (whole-issue close)

| field | value |
|-------|-------|
| issue | 397 — couch: capture panics to disk via debug.SetCrashOutput |
| repo | pair |
| issue file | workshop/issues/000397-couch-capture-panics-to-disk-via-debug-setcrashoutput.md |
| boundary | whole-issue close |
| milestone | — |
| window | 01502719f947082e290e5c19d05fb3ea53dcbfd8..c7e71f12167eb7a94ec3baed57089fd60590cc36 |
| command | sdlc close --issue 397 |
| reviewer | claude |
| timestamp | 2026-10-06T16:30:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The design is sound: a strict name grammar, pure `Classify`, rename-before-report, the gc sweep reusing `DecideSegment`, the lease-ordered install, and a re-exec test that proves `SetCrashOutput` reaches the file. But the couch wiring defeats the feature in the case most likely to happen. `run.go:409` registers `defer installCrashReport(...).Close()` in the function that goes on to call `console.Run()` on the same goroutine (`run.go:683`), and `Console.Run`'s event loop (input, layout, render, switchTo) runs on that goroutine (`console.go:592-680`). When a goroutine panics, Go runs its deferred calls *before* it prints the panic. So `Capture.Close` runs first: it calls `SetCrashOutput(nil)`, sees an empty file, and deletes it. The runtime then prints the panic to stderr only. I checked this with a scratch re-exec test: install, `defer c.Close()`, then panic on the same goroutine leaves the crash dir **empty**. Because the file is gone, the next start doesn't even show the "ended abruptly" notice. A panic in the console loop is exactly as silent as before #397. Only panics in other goroutines and runtime `fatal error`s get captured. No test covers this, because the forced-panic test (`crashreport_test.go` TestMain) panics without the deferred `Close` that production uses.

1. **Strengths**
   - `crashreport.go:37-52` `parseName` is strict: it round-trips the pid with `Itoa` and parses the timestamp, so foreign files are never touched (tested at `TestClassify`, and by `notes.txt` in `TestInstallReportsPreviousEndingsOnce`).
   - Rename-then-report (`crashreport.go:131-138`) makes "reported once" hold even if the notice is lost.
   - The defers run in the right order: crash `Close` is registered after `lease.Close`, so on a clean exit it runs first and the next incarnation never sees a live `.log`.
   - `Sweep` uses Lstat on regular files only, skips `.log` files whose owner is alive, and reuses `diagnosticlog.DecideSegment` and `RetentionPeriod` (ARCH-DRY). The symlink case is tested.
   - The `Notify` notice stands until displaced (`console_notify_test.go`), so a slow first paint can't expire it.

2. **Critical**
   - `cmd/internal/couchcmd/run.go:408-410` together with `crashreport.go:163-177`: a panic on the console goroutine deletes its own crash file during unwinding. **Fix sketch:** only run `Close` on a non-panicking exit. One way: `capture := installCrashReport(...); defer func(){ if p := recover(); p != nil { panic(p) }; capture.Close() }()` — on Go ≥1.23 the re-panic prints `[recovered, repanicked]` with the original frames. Another: a `clean` flag set on the normal return path. Add a regression test that re-execs a child, which goes through the production wiring (`installCrashReport`, or a helper it shares, plus the same deferred close) and panics on the deferring goroutine. Assert the file holds `panic:` and a goroutine stack. Done-when #1 claims "a test-only forced panic in couch", but the current test bypasses couch's deferral, so that box is not truly met.

3. **Important**
   - `crashreport.go:128-141`: any failure to rename or remove an older file returns before this run's capture is opened. A single stuck stale file (bad permissions or similar) then turns off capture on *every* later start. Report the error, carry on, and still install this run's capture.

4. **Minor**
   - `lease.Close` also runs during panic unwinding, before the runtime writes the trace. A couch relaunched in that window would classify the still-empty `.log` as Abrupt and unlink it. The window is microseconds, but it is worth a comment.
   - Each Abrupt or Crashed file becomes its own standing control notice (`Kind: "crash "+name`). After several power-loss ends you get N standing notices; consider folding them into one count.
   - `gcruntime.Preview` lists crash rows even when migration is incomplete, but `Apply` skips them in that case (it returns early at `runtime.go:198`), so a preview can promise removals that Apply won't make.
   - `Sweep`'s `limit` counts every row, retained ones included, and applies per store, so the effective budget is limit × number of stores.
   - `maxCrashRows` (gcruntime) restates `crashreport.maxEntries` (4096).
   - In `Sweep`, the eligibility decision sits inline in the IO loop. It could be a pure `decide(name, mtime, alive, now)` next to `Classify`.
   - If a directory holds more than `maxEntries` entries, both `Install` and `Sweep` refuse, so gc can't drain it. This can't happen in practice from couch's own writes.

5. **Test coverage**
   - Good: grammar, classification, the report-once lifecycle, clean close, and gc preview/apply through a registered store.
   - Missing: a panic through the real deferred-close path (the Critical above), and Install continuing after a failed report.
   - The `TestProductionArtifactReferencesAreExactlyClassified` failure is the same as on base. The extra lines at head come from untracked `runtimebundle/assets` build output, and nothing involves `crashreport`.

6. **Architecture**
   - ARCH-DRY: pass (the 4096 restatement is Minor).
   - ARCH-PURE: pass (`Classify` is pure; the inline decision in `Sweep` is Minor).
   - ARCH-PURPOSE: **flag**. The Critical above means capture misses the main-goroutine panics it exists for.
   - ARCH-MOCK: pass. No external dependency; liveness is injected into `Sweep`.
   - ARCH-CONSTRAINTS: pass. One bounded readdir at startup, and gc honours its limit.
   - ARCH-SECURE: pass. Files are 0600, names are strictly grammared, Lstat on regular files only.
   - ARCH-ORDER: **flag**. Panic unwinding is an interrupting event the lifecycle never lists. "Clean exit" and "panicking exit" both reach `Close`, and that conflation is the bug.
   - ARCH-FUNERAL: pass. Empty files go at close or next start, `.crash` and dead `.log` files are aged out by `pair gc`, and growth is one file per crash.

7. **Plan revisions**
   - Lifecycle step 4 should read "On a *non-panicking* exit: disable crash output, close, delete if empty; a panicking exit leaves the file to the runtime." Add a Revisions entry recording that panic unwinding runs deferred `Close` and how the wiring now avoids it.

```findings
findings:
  - id: new
    severity: Critical
    family: defer-runs-before-panic-print
    title: |
      Deferred Capture.Close runs during panic unwinding on the console goroutine and deletes the crash file before the runtime writes the panic
    detail: |
      run.go:409 defers installCrashReport(...).Close() in the same goroutine that runs console.Run's event loop (run.go:683). Go runs defers before printing a panic, so Close disables crash output and removes the still-empty file; verified with a re-exec scratch test (crash dir empty). Gate Close on a non-panicking exit (recover-and-repanic, or a clean flag) and add a re-exec regression test through the production wiring.
  - id: new
    severity: Important
    family: report-failure-blocks-capture
    title: |
      A failure reporting an old crash file aborts Install before this run's capture is opened
    detail: |
      crashreport.go:128-141 returns on any rename/remove error, so one stuck stale file disables crash capture on every subsequent start. Record the error as a notice and continue to open and install the current file.
  - id: new
    severity: Minor
    family: lease-released-before-panic-print
    title: |
      lease.Close also runs during panic unwinding, opening a tiny window where a relaunch misclassifies the dying run's empty .log as Abrupt
  - id: new
    severity: Minor
    family: notice-per-file-unbounded
    title: |
      Each previous crash or abrupt file posts its own standing control notice; several stale endings stack N notices
  - id: new
    severity: Minor
    family: preview-apply-parity
    title: |
      gc Preview lists crash rows when migration is incomplete, but Apply skips them in that case
  - id: new
    severity: Minor
    family: restated-constant
    title: |
      maxCrashRows in gcruntime restates crashreport.maxEntries (4096)
```

---

## Re-review — 2026-10-06T16:46:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 397 — couch: capture panics to disk via debug.SetCrashOutput |
| repo | pair |
| issue file | workshop/issues/000397-couch-capture-panics-to-disk-via-debug-setcrashoutput.md |
| boundary | whole-issue close |
| milestone | — |
| window | 01502719f947082e290e5c19d05fb3ea53dcbfd8..158fc0474a8176ca323d7d89dc65be353601c343 |
| command | sdlc close --issue 397 |
| reviewer | claude |
| timestamp | 2026-10-06T16:46:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The BR-1 code defect is fixed. `run.go:409-413` no longer defers anything. `cmd/couch/main.go:16-19` calls `crashreport.Finish` only after `Run` returns normally, so a panic now skips it. BR-2, BR-3, BR-4 and BR-6 are fixed and have tests. BR-5 is the same behavior the existing diagnostics rows already have. What's left is BR-1's regression test. I copied the tree to a scratch directory and put the original bug back at the production site, adding `defer crashreport.Finish()` right after `installCrashReport(...)` in `run.go`. Every crash test in `couchcmd`, `crashreport` and `gcruntime` still passed. The new test calls `installCrashReport` directly, below the place the bug lived, so it can't catch it. That's a cheap fix and doesn't block shipping the behavior.

(Process note: early on I ran `git checkout -q 158fc047` by mistake, which detached HEAD at the same commit. I put the branch back right away. HEAD and the working tree are unchanged: the issue file is still modified and the two plan files are still untracked.)

1. **Strengths**
   - `crashreport.Install` (`crashreport.go:134-189`) reports each stale-file problem but still installs this run's capture, and `TestUnreportableFileStillInstallsCapture` would fail against the old early return.
   - A `.log` whose pid is still alive is left alone. That handles the dying owner whose lease has already been released.
   - `Summary` folds every previous ending into one standing notice. It is pure and tested by a table test.
   - Names must match an exact pattern, only regular files count (`Lstat`), files are 0600, and symlinks are skipped in `Sweep` and tested.
   - The lessons entry states the rule for this whole class ("never end anything a panic must outlive from a defer"), not just this instance.

2. **Critical**: none new. BR-1 stays open below, for its regression test only.

3. **Important**
   - **BR-1 regression test** (`couchcmd/crashreport_test.go:25-49`): it does not run through `runTypedOperationWithConsole` or `main`. To fix, have the re-exec child call `runTypedOperationWithConsole` with a `finish` callback that panics. The existing pty/testRT setup in `run_test.go:350-380` can do this. Then a `defer crashreport.Finish()` reintroduced in `run.go` turns the test red.

4. **Minor**
   - `crashreport.go` `list`/`Sweep`: with more than `MaxEntries` entries (foreign files count too), `list` returns an error. `Sweep` then refuses the whole directory, so `pair gc` can never shrink it. The limit is on the reader, and nothing bounds the writer side. `Sweep` should process the first `MaxEntries` entries and still report that the directory is over the limit.
   - `Install` overwrites the global `active` without closing an earlier capture. The `run_test.go` and `continuation_acceptance_test.go` tests that call `runTypedOperationWithConsole` with a live-owning operation now install process-wide crash output into temp directories and never call `Finish`. Production installs once per process, so only test hygiene is affected.
   - `procutil.Alive(strconv.Itoa(pid))` is wrapped the same way in `run.go` and `gcruntime/runtime.go`. Trivial.

5. **Test coverage**
   - The re-exec child in `crashreport` proves the file plus stderr. Classify, Summary, Sweep, the live-owner skip, the unrenamable stale file and the gc preview/apply flow are all covered.
   - The gap is the production wiring: both the `run.go` site and the `Finish` call in `main` are unpinned.

6. **Architecture**
   - ARCH-DRY: pass. `MaxEntries`, `DecideSegment` and `RetentionPeriod` are shared.
   - ARCH-PURE: pass. `Classify`, `Summary` and `parseName` are pure, and the IO in `Install`/`Sweep` is thin.
   - ARCH-PURPOSE: pass. Everything in Done-when is delivered, and the `pair` wrapper was optional in the Spec.
   - ARCH-MOCK: pass. Only the real filesystem is involved, via temp directories, and process liveness is injected as `alive`.
   - ARCH-CONSTRAINTS: pass. The startup scan is bounded, and gc honors `limit` per store.
   - ARCH-SECURE: pass. Startup reads only file sizes, never contents, and parsing is strict.
   - ARCH-ORDER: pass with a note. Install lives in `couchcmd` and Finish in `main`; this split exists because of the defer constraint and is documented. Only the test above enforces it, and that test is currently missing.
   - ARCH-FUNERAL: flagged Minor (the `MaxEntries` cliff above). Otherwise every file has an end: `.crash` files age out after 365 days, and a `.log` is removed on a clean exit or at the next start.

7. **Plan revisions**: none needed. The plan matches the code. The `## Log` claim "Mutating its defer into Finish fails it" is true of the test's own no-op defer, not of the production site, so the wording should say that.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Code fixed (no defer in run.go; Finish after Run in main), but reintroducing defer crashreport.Finish() in run.go in a scratch copy leaves every crash test green; the regression test bypasses the production wiring.
  - id: BR-2
    disposition: addressed
    note: |
      Install collects per-file problems and still opens and installs the capture; TestUnreportableFileStillInstallsCapture fails against the old early return.
  - id: BR-3
    disposition: addressed
    note: |
      A .log whose pid is alive is skipped; covered by TestInstallLeavesALiveOwnersFileAlone.
  - id: BR-4
    disposition: addressed
    note: |
      Summary folds all endings into one notice; TestSummary plus the combined assertion in TestInstallReportsPreviousEndingsOnce.
  - id: BR-5
    disposition: withdrawn
    note: |
      gcruntime Apply returns before any diagnostics rows when migration is incomplete; crash rows follow the existing rows exactly.
  - id: BR-6
    disposition: addressed
    note: |
      crashreport.MaxEntries is exported and gcruntime uses it; no restated constant remains.
findings:
  - id: new
    severity: Minor
    family: reader-cap-without-writer-bound
    title: |
      Sweep refuses a crash dir with more than MaxEntries entries, so pair gc can never drain it
    detail: |
      list errors past MaxEntries (foreign files count) and Sweep returns that error with no rows, so neither startup nor gc makes progress. Sweep the first MaxEntries entries and still report the overflow.
  - id: new
    severity: Minor
    family: process-global-install-unpaired
    title: |
      Install overwrites the global active capture without closing the previous one; existing console tests now leak installs
    detail: |
      run_test and continuation_acceptance_test call runTypedOperationWithConsole with live-owning operations, which installs process-wide crash output into temp dirs and never calls Finish. Production installs once per process; this affects only tests.
```

---

## Re-review — 2026-10-06T17:00:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 397 — couch: capture panics to disk via debug.SetCrashOutput |
| repo | pair |
| issue file | workshop/issues/000397-couch-capture-panics-to-disk-via-debug-setcrashoutput.md |
| boundary | whole-issue close |
| milestone | — |
| window | 01502719f947082e290e5c19d05fb3ea53dcbfd8..889592b03d6563e4ff410c18034f27309dcbd837 |
| command | sdlc close --issue 397 |
| reviewer | claude |
| timestamp | 2026-10-06T17:00:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three open findings are fixed, and I found nothing new that blocks the close. I checked BR-1 by mutation. I copied the tree to a scratch directory and re-added `defer crashreport.Finish()` right after `installCrashReport` in `run.go`. With that change, `TestConsoleOwnerPanicLeavesItsCrashFile` fails ("crash dir = [] … want the crashing run's file"). On HEAD it passes. That test needs a pty, which the sandbox blocks, so I ran it unsandboxed. The crashreport, gcruntime and couchtty tests pass. One cosmetic issue remains, listed under Minor.

**Strengths**
- `cmd/couch/main.go:14-18`: capture ends only after a normal return from `Run`. A panic never reaches that line. The doc comments in `crashreport.go:453-488` make "never call `Finish`/`Close` from a defer" an explicit rule.
- `couchcmd/crashreport_test.go:68`: the regression test re-runs the test binary as a child, which panics inside the production function with its own defers still pending. The parent owns the store directory, because the test runner deletes `t.TempDir` while the panic unwinds. The test fails if the bug comes back.
- `crashreport.Install`: errors reporting old files now sit alongside a working capture instead of replacing it (BR-2). A `.log` whose pid is still alive is left alone, which covers the window where the lease is released before the panic is printed (BR-3).
- `Classify` and `Summary` are pure and tested from a table. The gc sweep uses the existing `diagnosticlog.DecideSegment` and `RetentionPeriod` rather than its own copy.

**Critical:** none.

**Important:** none.

**Minor**
- `run.go:180-182`: every `Install` error is shown as "crash capture unavailable: …". Since the BR-2 fix, `Install` also returns errors while capture is working: a failed rename, a failed remove, or the overflow error from an over-full directory. In those cases the notice says capture is off when it is on. The message should depend on whether a capture was returned.
- A SIGHUP from closing the terminal tab kills couch without running `Finish`. That leaves an empty `.log`, which the next start reports as "ended abruptly". That report is accurate enough. It's noted here only so it isn't later mistaken for a bug.

**Test coverage notes**
- BR-7 is covered by `TestSweepDrainsAnOverfullDirectory`. Under the old code `Sweep` returned an error and wrote no rows, so this test would have failed.
- BR-8 is partly covered by `TestSecondInstallEndsTheFirst`. Console tests that never call `Finish` still leave one install per test process. It points at a file under a deleted temp directory. This is harmless and limited to tests.
- `main.go`'s `Finish` call has no test of its own. It's two lines, and the mutation test above shows that a defer at the production site is what matters.

**Architecture (ARCH-\*)**
- **ARCH-DRY: pass.** One `MaxEntries`, one `ProcessAlive`, and the shared retention period.
- **ARCH-PURE: pass.** `Classify` and `Summary` are pure. `Install` and `Sweep` are the thin IO layer.
- **ARCH-PURPOSE: pass.** Every Done-when item is delivered, and the atlas names the crash path. `pair` itself was optional in the Spec ("Consider…") and was reasonably scoped out.
- **ARCH-MOCK: pass.** The only dependency is the Go runtime, and the test runs the real one in a child process.
- **ARCH-CONSTRAINTS: pass.** Directory scans are capped at `MaxEntries` and gc honours `limit`.
- **ARCH-SECURE: pass.** Only exact file names are accepted, `Lstat` limits the scan to regular files, files are created `O_EXCL` with mode 0600, and foreign files are ignored.
- **ARCH-ORDER: pass.** The `.log` → `.crash` / delete lifecycle is written down. Ordering against the lease and the panic is handled by the alive-pid check and by calling `Finish` only on a normal return, and the test pins that ordering.
- **ARCH-FUNERAL: pass.** gc ages out every crash file, reported or not. Empty files are removed on a clean exit or on the next start. The scan cap on the reader is matched on the writer side by one file per console run, and an over-full directory now drains.

**Architectural notes for upcoming work:** if `pair` gets the same capture, put the normal-return `Finish` in its `main` too. Never put it in a defer inside `Run`.

**Plan revisions:** none needed; the Log already records rounds 1 and 2.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Finish now runs only after a normal return in main. Re-adding the defer in run.go makes TestConsoleOwnerPanicLeavesItsCrashFile fail (checked in a scratch copy).
  - id: BR-7
    disposition: addressed
    note: |
      list now keeps the first MaxEntries and reports the overflow. TestSweepDrainsAnOverfullDirectory would fail under the old refuse-the-directory code.
  - id: BR-8
    disposition: addressed
    note: |
      A second Install now ends the first (TestSecondInstallEndsTheFirst). Each test process can still leave one install pointing at a deleted temp file; harmless and test-only.
findings:
  - id: new
    severity: Minor
    family: notice-misstates-capture-state
    title: |
      installCrashReport shows "crash capture unavailable" even when Install returned a working capture
    detail: |
      Since BR-2, Install can return an error from a failed rename or remove, or from the overflow check, while still returning a live capture. run.go:180 puts "unavailable" in front of every error. The notice should depend on whether a capture came back.
```
