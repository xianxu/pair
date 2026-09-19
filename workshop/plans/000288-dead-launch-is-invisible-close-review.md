# Boundary Review — pair#288 (whole-issue close)

| field | value |
|-------|-------|
| issue | 288 — A launch whose zellij server dies at birth hangs the launcher and leaves Couch's thread live |
| repo | pair |
| issue file | workshop/issues/000288-dead-launch-is-invisible.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23e37f9e06bc087fb80bb54c3c1cacb7182294fa..3e1aeb0a23bd9efc9c26df57a64404c6ed28e7ea |
| command | sdlc close --issue 288 |
| reviewer | claude |
| timestamp | 2026-09-18T22:09:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue's purpose is delivered and independently verified: `panebirth` genuinely consolidates the three birth waits, the watch's verdict is a tagged enum with a pure, table-tested `judgeUnborn`, the OS kill path is tested against real processes, and the live probe run (base 20/20 hung → fix 0/20, no client left behind) is the right kind of evidence. I re-ran the affected packages under `-race` (panebirth, titlepoller, launcher, artifactpath all green; the only failures in this environment are `operation not permitted` from pty/exec restrictions, unrelated to the diff) and ran the mutation checks myself. Two things hold it back from SHIP, both cheap: the fake models `LaunchSession`'s cancel return more optimistically than `os/exec` actually behaves — I verified with a standalone program that a child which exits **0** after `Cancel` makes `Run` return `context.Canceled`, so the `code != 0` guard never sees a clean exit in production and the create falls into the generic `failed to launch … : context canceled` path *above* the dead-birth message — and neither of the two cause guards is individually pinned: removing either one alone leaves test 6 green 20× under `-race`; only removing both turns it red.

**1. Strengths**

- `cmd/internal/panebirth/panebirth.go:66` — the "a failed observation is *not yet*, never birth" rule now has exactly one home, with a fake-clock test (`panebirth_test.go:96`) that pins both halves (error-then-birth → nil; errors-only → the error rides `ErrUnborn`). The grace→`DeadlineExceeded` mapping is the non-obvious part, and `coldresume_birth_test.go:123` was strengthened to assert the `(waited …)` diagnosis so losing it fails a test.
- `cmd/internal/launcher/birthwatch.go:36-46` — a 5-value tagged verdict instead of a `born/alive/unknown` boolean constellation, with `judgeUnborn` pure and table-tested including zellij's `(nil, nil)` empty inventory (ARCH-ORDER, ARCH-PURE).
- `cmd/internal/launcher/osruntime_launch_test.go` — tests the real reaping (`syscall.Kill(pid, 0) == ESRCH`) against real processes, including the TERM-deaf `exec sleep` case; this is where the Couch zombie/pty-hold risk actually lives, and it isn't mocked.
- Extent is lexically bounded (`launchWatched` starts and joins the watch), `birthAlive` stands down for good, and `birthUnknown` never tears down — the "no teardown without an answer" rule is tested (`pane_birth_test.go:206`).
- ARCH-MOCK is taken seriously for once: the fake models death B, the live `-hammer` conformance run exists, and the re-run-on-version-change cadence is recorded in both `SKILL.md` and `atlas/architecture.md`.

**2. Critical findings** — none.

**3. Important findings**

- **`cmd/internal/launcher/osruntime.go:185` (+ `createflow.go:794`) — the fake's cancel return is one the real seam cannot produce.** `fakeRuntime.LaunchSession` returns `(-1, nil)` on `ctx.Done()` and `(0, nil)` only when released *before* the cancel. Real `exec.Cmd` with `Cancel`+`WaitDelay` returns `ctx.Err()` whenever `Cancel` succeeded and the child then exits **0** (verified: `trap 'exit 0' TERM` → `Run err = context canceled`, `ProcessState.ExitCode = 0`). `runBlockingHandoff` maps that to `(1, context.Canceled)`, and `runCreate` checks `err != nil` at line 794 *before* the verdict at line 803. Two consequences: (a) the contract stated in the plan, `atlas/architecture.md` and `lessons.md` — "a clean quit that races the probe keeps the normal path" — does not hold when the abort lands before the client is reaped; the quit is reported as `pair: failed to launch zellij session '…': context canceled` and the quit cleanup is skipped, which is exactly what the guard was written to prevent; (b) on any zellij whose client exits 0 on SIGTERM, the dead-birth diagnosis this issue exists to produce is replaced by that generic message. Fix sketch: in `runBlockingHandoff`, prefer the child's real status when it ran — `if cmd.ProcessState != nil { return cmd.ProcessState.ExitCode(), nil }` — before falling back to `1, err`; then teach the fake to model "released cleanly while cancelled" and pin it.
- **`cmd/internal/launcher/pane_birth_test.go:235` + `createflow.go:803` — neither cause guard has a regression test.** Measured in a scratch copy of the pinned tree: dropping `&& code != 0` alone → `TestCleanQuitRacingTheProbeKeepsTheNormalPath` passes 20/20 under `-race`; dropping the post-probe `ctx.Err()` recheck alone (`birthwatch.go:82`) → passes 20/20; dropping both → fails immediately. The test cannot control the `stopWatch()`-vs-verdict interleaving, so "the outcome holds under every interleaving" is a sample of whichever ordering the scheduler hands out (ARCH-ORDER's highest-leverage flag). Fix sketch: lift the decision out of the 300-line `runCreate` into a pure predicate (`func failedAtBirth(v birthVerdict, code int) bool`) with a table test — that pins `code != 0` — and give `launchWatched` a seam (an injected watch func, or a `watchDecided` hook on the fake) that lets a test force verdict-before-stopWatch, pinning the recheck.

**4. Minor findings**

- `birthwatch.go:70-90` — the `birthUnknown` loop has no cap: a launch whose sidecar never appears and whose probe keeps failing repeats a machine-wide `list-sessions` every ~10 s for the entire life of the launch, and that call is itself the birth-killing mechanism for *other* threads (ARCH-CONSTRAINTS). A cap (3 rounds → stand down) or backoff would bound it.
- The evidence *path* is single-sourced, but the *observation* isn't: the poller uses `rt.ModTime` (`titlepoller/run.go:127`), the watch uses `rt.FileSize` (`birthwatch.go:72`). Consider a `panebirth` predicate, or a line saying why two.
- `probes/zellijbirthrace/SKILL.md` — "The 2026-09-18 runs predate `-exit-wait`, so their Hung column is empty" sits under a table whose #288 rows are *also* 2026-09-18; say "the rows above #288" instead.

**5. Test coverage notes**

`panebirth` is well covered (already-born, poll-to-birth, grace-as-deadline with sleep counting, cancel with no further observation, error-is-not-birth both ways, `Evidence` path equality + invalid tag). The six end-to-end watch cases each fail their named mutation *except* the two cause guards above. Untested but harmless: `Await` with a non-positive grace (Couch's `time.Until(deadline)` when already past) and the new `no registration deadline` error — I checked every caller builds `registrationContext` with `context.WithTimeout`, so that branch is unreachable today. I could not run the full suite here (pty tests are blocked in this shell); the Log's `make test` claim is unverified from my side, and the operator Couch smoke exists only as an **uncommitted** working-tree edit to the issue file — make sure `sdlc issue sync` carries it before the close.

**6. Architectural notes**

ARCH-DRY pass (three waiters, one loop; `zellijLogPath` duplicating the probe's `zellijTmp` is forced by the `cmd/internal` boundary). ARCH-PURE mostly pass — flagged only where the dead-path decision stayed inline in `runCreate`. ARCH-PURPOSE pass: the shadow-sweep over `PaneChecked` consumers shows the create-path clear, the poller and the watch all deriving from `panebirth.Evidence`; the remaining call sites (quit path, gc, env export) are different purposes. ARCH-MOCK flagged (finding 1) — the fake and the real seam must agree on the cancel return, and a live conformance check that only exercises death B's signal exit won't catch the disagreement. ARCH-CONSTRAINTS mostly pass (bound measured, probes individually bounded), flagged for the uncapped unknown rounds. ARCH-SECURE: N/A — no new untrusted input or credential; the liveness parse already degrades visibly (`birthUnknown`, never absence). ARCH-ORDER flagged (finding 2). ARCH-FUNERAL pass — nothing durable created; the watch dies with `runCreate`, the client is reaped, the poller and session watcher exit at their own deadlines.

**7. Plan revision recommendations**

- A `## Revisions` entry noting that `LaunchSession`'s real return after a cancel is `context.Canceled` for a child that exits 0, that `runCreate` checks `err != nil` first, and how the code is now derived (whichever fix is chosen) — the current "Ordering (ARCH-ORDER)" section and the `lessons.md` rule both assert a guard the seam doesn't deliver.
- A revision recording that Task 4 Step 7's mutation check ("both cause guards removed → test 6 fails") does not establish either guard individually, and naming the pure predicate + ordering seam that will.

```findings
findings:
  - id: new
    severity: Important
    family: fake-diverges-from-seam
    title: |
      The fake's cancelled-launch return cannot occur in production, so the code != 0 guard is dead in the real seam
    detail: |
      fakeRuntime.LaunchSession returns (-1, nil) on ctx.Done and (0, nil) only
      when released before the cancel, but exec.Cmd with Cancel+WaitDelay returns
      ctx.Err() whenever Cancel succeeded and the child then exits 0 (verified:
      Run err = context canceled, ProcessState.ExitCode = 0). runBlockingHandoff
      (osruntime.go:185) turns that into (1, context.Canceled) and createflow.go:794
      handles err before the verdict at :803, so a clean quit racing the probe is
      reported as "failed to launch zellij session: context canceled" with the quit
      cleanup skipped, and a zellij client that exits 0 on SIGTERM loses the
      dead-birth diagnosis. Derive the code from cmd.ProcessState when the process
      ran, and let the fake model a clean release under cancel.
  - id: new
    severity: Important
    family: race-guard-untested-interleaving
    title: |
      Neither cause guard is pinned: removing either alone leaves test 6 green 20x under -race
    detail: |
      In a scratch copy of the pinned tree, dropping `&& code != 0`
      (createflow.go:803) alone, or the post-probe ctx recheck
      (birthwatch.go:82) alone, leaves TestCleanQuitRacingTheProbeKeepsTheNormalPath
      passing 20/20 under -race; only removing both fails it. The test cannot
      control the stopWatch-vs-verdict interleaving, so its "holds under every
      interleaving" claim samples one ordering. Extract the dead-path decision as a
      pure predicate with a table test, and add a seam that forces
      verdict-before-stopWatch.
  - id: new
    severity: Minor
    family: unbounded-probe-rounds
    title: |
      The birthUnknown loop has no cap, so a persistently failing probe repeats a machine-wide list-sessions every bound for the life of the launch
    detail: |
      birthwatch.go:70-90 loops bound-then-probe forever while the probe errors and
      the pane never appears. That probe is itself the birth-window risk for other
      threads (ARCH-CONSTRAINTS). Cap the unknown rounds or back off.
  - id: new
    severity: Minor
    family: single-source-partial
    title: |
      The evidence path is single-sourced but its observation is not: ModTime in the poller, FileSize in the watch
  - id: new
    severity: Minor
    family: doc-referent-ambiguous
    title: |
      SKILL.md's "the 2026-09-18 runs predate -exit-wait" sits above #288 rows that are also 2026-09-18
```

---

## Re-review — 2026-09-18T22:25:43-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 288 — A launch whose zellij server dies at birth hangs the launcher and leaves Couch's thread live |
| repo | pair |
| issue file | workshop/issues/000288-dead-launch-is-invisible.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23e37f9e06bc087fb80bb54c3c1cacb7182294fa..14e8478357a9cf8f3546c1dbaf40ca874a17025d |
| command | sdlc close --issue 288 |
| reviewer | claude |
| timestamp | 2026-09-18T22:25:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Important findings from round 1 are genuinely fixed, and I verified each by mutation in a scratch copy of the pinned tree rather than by reading the commit message: reverting `runBlockingHandoff` to the old `*exec.ExitError` form fails `TestCancelledHandoffReportsAChildsCleanExit` (BR-2), and dropping `&& code != 0` or the post-probe `ctx` recheck **each alone** now fails a named test (BR-3) — the exact defect the prior round identified. The shipped design is clean: one shared `panebirth.Await` with all three waiters derived from it, a pure verdict core (`judgeUnborn` / `failedAtBirth` / `nextProbeWait`) under a thin watch shell whose extent is lexically joined in `launchWatched`, and live conformance numbers (hung 20/20 → 0/20 under `-hammer 10ms`). Nothing blocks the boundary. What remains is Minor: BR-4's backoff is present and reachable but its *wiring* is unpinned — I removed `wait = nextProbeWait(wait)` from the loop head and the whole suite stayed green, so the plan's claim that "the backoff → its table fails" describes a mutation of the function, not of its use.

### 1. Strengths

- `cmd/internal/launcher/birthwatch.go:78` — `failedAtBirth` as a pure predicate with a table test is the right answer to BR-3: the dead-path decision is now readable off one line, and `TestFailedAtBirth` + `TestCleanQuitUnderTheWatchsCancelKeepsTheNormalPath` both go red when the guard is dropped (verified).
- `cmd/internal/launcher/osruntime.go:191` — deriving the code from `cmd.ProcessState` whenever the child ran is the correct fix for `os/exec`'s cancel contract, and `TestCancelledHandoffReportsAChildsCleanExit` drives the *real* seam (`sh -c 'trap "exit 0" TERM'`), not the fake. The blast radius is contained: the only other caller, `AttachSession` (`osruntime.go:773`), passes `exec.Command` with `*os.File` stdio, so no error can be swallowed there.
- `cmd/internal/panebirth/panebirth.go` — the ARCH-DRY win is real and complete. Shadow-sweep of the single source: `titlepoller/run.go:124`, `couchcore/launch_existing.go:392`, `launcher/birthwatch.go:92` all derive from `Await`, and `Evidence` is the one declaration of the path (`createflow.go:777` clears exactly what `birthwatch` stats). No hand-maintained restatement left.
- `launchWatched` (`birthwatch.go:112`) bounds the watch's extent lexically — started before the handoff, `stopWatch()` + `<-verdict` join after it, `defer abort()`. The one residual (a probe in flight at join) is bounded by two 5 s `zellijQueryTimeout` calls and is written down in the plan.
- `probes/zellijbirthrace` splitting `hung` from `hung-after-birth`, matching the leftover client on the ASCII tag, and recording the cadence rule ("re-run on every zellij version change") gives ARCH-MOCK a real live-conformance leg rather than a fake asserting itself.

### 2. Critical findings

None.

### 3. Important findings

None new. BR-2 and BR-3 are disposed `addressed` with mutation evidence (below).

### 4. Minor findings

- BR-4's backoff is reachable but unpinned — removing `wait = nextProbeWait(wait)` from `birthwatch.go:91` leaves every test green; `TestNextProbeWait` only restates the function.
- `zellijLogPath()` (`birthwatch.go:131`) re-declares zellij's tmp-dir layout already encoded in `probes/zellijbirthrace/main.go:100` (`zellijTmp`). Same module, two sources for one external fact; Go's `cmd/internal` boundary blocks direct reuse, so this is a note, not an ask.
- Plan drift that the Revisions don't record: Task 3 Step 1 still specifies `cancellableHandoff(ctx, name, args...)`, the code ships `killOnCancel(cmd)` (the `--config-dir` vocabulary gate, recorded in the issue Log only); the fake's probe signal is `launchProbed`, the plan calls it `livenessAfterLaunch`.

### 5. Test coverage notes

- `go test -race -count=1 ./cmd/internal/launcher/ ./cmd/internal/panebirth/ ./cmd/internal/titlepoller/` on a `git archive 14e84783` scratch copy: panebirth and titlepoller `ok`; launcher's only failure is `TestCreateLayoutWrapperPreservesAgentCommand`, which cannot build the wrapper because `cmd/internal/runtimebundle/assets/` is gitignored/generated — an artifact of the archive, not a regression.
- Mutation results (each reverted after): code guard removed → 2 failures; post-probe recheck removed → 1 failure; `ProcessState` derivation reverted → 1 failure; backoff call removed → **0 failures**.
- Both orderings of the watch/client race now have a seam: `TestWatchBirthDoesNotJudgeALaunchThatEndedDuringItsProbe` (client ends inside the probe, driven by `livenessHook`) and `TestCleanQuitUnderTheWatchsCancelKeepsTheNormalPath` (verdict lands first, deterministically, because the abort *is* what releases the client).
- Gap worth one line of test: nothing asserts the watch applies a growing wait — `TestUnansweredProbeIsAskedAgainNotActedOn` would catch it with an elapsed-time floor (`bound + 2·bound` ≈ 90 ms with `BirthBound = 30ms`).

### 6. Architectural notes for upcoming work

- **ARCH-DRY** pass (one `Await`, one `Evidence`, three consumers); the only duplicate fact is zellij's tmp path, noted above. **ARCH-PURE** pass — verdict logic is pure and tested with no IO; the fake is needed only for the shell. **ARCH-PURPOSE** pass — the single-source sweep is complete, and the deferrals (Couch surfacing the launcher's reason, `hung-after-birth`) are separable, not the point of the issue. **ARCH-MOCK** pass — stateful fake behind the same seam as production, plus the OS-side tests and the probe's conformance cadence; BR-2's lesson (read the wrapped API's documented returns before trusting a fake's mode) is recorded in `lessons.md`. **ARCH-CONSTRAINTS** pass — the 10 s bound is measured (3.4× loaded worst case), the probe backs off, the join is bounded. **ARCH-SECURE** pass — no new untrusted parse, no credential surface; the only external read is the pre-existing `SessionLiveness` seam. **ARCH-ORDER** pass — explicit `birthVerdict` enum, transitions pure, both interleavings injectable, extent joined. **ARCH-FUNERAL** pass — nothing durable created; watch goroutine joined, client reaped, layout record restored, title poller expires on its own grace.
- For follow-up: `LaunchOptions.BirthBound` is a test-only override with no operator escape hatch. The kill requires zellij to *also* list nothing, so a slow-but-socket-bound birth is safe, but if a report of a killed healthy launch ever arrives, an env override is the cheapest lever.
- The dead path returns before `<-defaultReady` (`createflow.go:802`), the same shape as the pre-existing launch-error path. The channel is buffered and the goroutine self-limits, so no leak — but if a third early return joins those two, extract a single `abandonLaunch(...)` helper rather than a third copy.

### 7. Plan revision recommendations

Add one `## Revisions` entry (2026-09-18, close review round 2) recording:
- `cancellableHandoff` → `killOnCancel(cmd)` (Task 3 Step 1), driven by the `--config-dir` vocabulary gate; and the fake's `livenessAfterLaunch` → `launchProbed`.
- `failedAtBirth` and `nextProbeWait` added to the **Pure entities** table (`cmd/internal/launcher/birthwatch.go`, new), and `launchWatched` to **Integration points** — both exist in code but not in the Core-concepts tables.
- The backoff's use in `watchBirth` has no failing-without-it test; either add the elapsed-time floor to test 4 or state in the plan that only the function, not its wiring, is pinned.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Declined with a rationale recorded in the plan's Revisions; the predicted
      drift did occur (cancellableHandoff vs killOnCancel, livenessAfterLaunch
      vs launchProbed, the watchBirth block missing the backoff), so the RULE
      the family needs is: every divergence from a plan code block must land in
      Revisions in the same round. Minor, non-blocking.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by mutation: reverting runBlockingHandoff to the ExitError form
      fails TestCancelledHandoffReportsAChildsCleanExit, which drives real exec.
  - id: BR-3
    disposition: addressed
    note: |
      Verified by mutation: dropping the code guard fails 2 tests and dropping
      the post-probe ctx recheck fails 1 — each alone, unlike round 1.
  - id: BR-4
    disposition: not-addressed
    note: |
      The backoff ships and is reachable, but removing `wait = nextProbeWait(wait)`
      from birthwatch.go:91 leaves the suite green; only the function is pinned.
  - id: BR-5
    disposition: addressed
    note: |
      panebirth.Evidence now states that birth is the file's existence and that
      each waiter stats it through its own seam; the referents (poller ModTime,
      watch FileSize) match the code.
  - id: BR-6
    disposition: addressed
    note: |
      SKILL.md now says the rows above the #288 ones predate -exit-wait, which
      disambiguates the empty Hung column on the same-date rows.
findings:
  - id: new
    severity: Minor
    family: duplicated-external-tool-knowledge
    title: |
      zellijLogPath re-declares zellij's tmp-dir layout that the probe already owns
    detail: |
      birthwatch.go:131 and probes/zellijbirthrace/main.go:100 both encode
      os.TempDir()/zellij-<uid>. Same module, one external fact, two sources; a
      zellij layout change drifts one of them. Go's cmd/internal boundary blocks
      direct reuse, so this is a note for the next time either is touched.
```
