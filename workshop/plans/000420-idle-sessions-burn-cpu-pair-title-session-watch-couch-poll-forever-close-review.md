# Boundary Review — pair#420 (whole-issue close)

| field | value |
|-------|-------|
| issue | 420 — idle sessions burn CPU: pair title / session-watch / couch poll forever |
| repo | pair |
| issue file | workshop/issues/000420-idle-sessions-burn-cpu-pair-title-session-watch-couch-poll-forever.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7afffc58f430c44b8b705e25dd2846c3df1bc3bd..d90164010fd6bd8c99fbb6535d3885624f829640 |
| command | sdlc close --issue 420 |
| reviewer | claude |
| timestamp | 2026-10-10T18:42:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read in parts.

Running the tests for the touched packages. The launcher failures come from leaked session environment variables, as noted in memory, so I'm re-running with those variables scrubbed.

Diff and tests reviewed. Now spot-checking the regression tests by mutating the fixes in a scratch copy.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Verdict: ship after small fixes. Nothing blocks.** I checked the work against the issue's Spec and Plan on 25 files. It does what the Spec says:

- **Title pollers:** liveness now ignores EXITED sessions. The shared row parser moved into a pure `zellijsession` package.
- **Bound-session reads:** they stat only the proof's artifacts instead of walking the store. `TargetEstablished` keeps only the proof's artifacts, so the results are unchanged.
- **Unbound watcher:** it is gated on a pending Pair turn and only validates transcripts newer than the last quiet moment. It exits after 24h with no send.

Test results:
- **Pass:** `sessionwatch`, `sessioninventory/...`, `sessioninventorytest`, `zellijsession` and `titlepoller`.
- **`launcher` fails:** this is the known `PAIR_*` session-environment leak (scoped-data-dir conflict), not this diff.
- **`artifactpath` fails:** `TestProductionArtifactReferencesAreExactlyClassified` flags files this diff doesn't touch (couchcore, reviewcmd, review lua). They arrived with the merge from `origin/main`. This diff's only change in that package, the new `manifest.go` row, is not among them.

I mutation-checked two fixes in a scratch copy. Disabling the agent-output gate turns `TestRunIdleChosenWatcherWalksOnlyAfterAgentOutput` red, and removing the mtime floor turns `TestRunPendingTurnSkipsCandidatesOlderThanQuietFloor` red.

What keeps this from a clean SHIP:
- The real `OSRuntime.StatFile` has no test of its own.
- Idle expiry ignores agent output, even though a chosen launch can bind without any Pair send.

1. **Strengths**
   - `zellijsession.go:14` pulls the row scan out of `launcher`, so `launcher` and `titlepoller` can share it without an import cycle. Its error behaviour is unchanged (ARCH-DRY).
   - `ListFiles` and `StatFile` build each entry with the same `fileEntryFor` (`runtime_os.go:154`), so a stat can't drift from a walk.
   - `ValidateBindingProof` (`query.go:157`) keeps the `len(Eligible) != len(proof.Artifacts)` check. A proof artifact that is missing or unrecognized still fails closed with `ErrArtifactChanged`.
   - The idle tests count operations: walks and reads of another session's transcript. They were mutation-checked, and `workshop/lessons.md` records the vacuous `OperationCountForRoot` trap.
   - The gate keeps the risky cases on the full path: an unreadable Pair log, a bound follower (`boundRootNodeID != ""`), and the epoch-acquisition walk.

2. **Critical:** none.

3. **Important**
   - **`OSRuntime.StatFile` has no test (`runtime_os.go:139`, ARCH-MOCK).** The fake returns `os.ErrNotExist` for a missing file, and `ActivityForSession` and `ObserveNamedArtifacts` both rely on that. Nothing checks that the real runtime matches. Suggested test on a temp dir:
     - a regular file gives the same `FileEntry` as `ListFiles`;
     - a missing file gives an error where `errors.Is(err, os.ErrNotExist)` holds;
     - a symlink or non-regular file is rejected.

     The live probe isn't a repeatable check.

4. **Minor**
   - **Idle expiry only counts Pair sends (`run.go:126`).** A chosen-id launch can bind through the filename handshake with no Pair send. The new test itself models the operator typing straight into the agent pane. Since the idle chosen watcher now costs about one stat per SlowPoll, counting `ScrollbackRaw` movement as activity in `idleExpired` would keep bindings for threads used directly after more than 24h idle. The Spec accepted this trade-off, so it's only noted here.
   - **Symlink handling differs between `StatFile` and the old walk.** For a final-component symlink, `resolveArtifact` returns `ErrPathEscape`. `ActivityForSession` now returns that as an error, where the old walk returned `(false, nil)`. Small, but it's a difference in failure behaviour.
   - **Extra walk on one pass (`run.go:195` and `:236`).** The pass that acquires the epoch now walks twice when a turn is pending: once for the epoch and once for the snapshot. It happens once per watcher.
   - **Possible missed handshake.** `outputMoved` records the recording's mtime before the walk. If the agent's last output lands before the chosen transcript file is created and nothing is written after it, the handshake waits for the next output. This is unlikely, because Claude keeps rendering after it writes the transcript.
   - **The counting helper is still broken.** `OperationCountForRoot` stays vacuous for `ListFiles`; it's recorded as a lesson but not fixed. Fixing the helper would cover the whole class of `== 0` assertions, not just the tests that avoided it (ARCH-PURPOSE).
   - **Import grouping.** `titlepoller/runtime.go:6` and `run_test.go:6` add a pair import inside the stdlib block. gofmt accepts this, but goimports would group them differently.

5. **Test coverage notes**
   - Covered: the send gate, the floor, idle expiry, the idle chosen-id walk count, the `zellijsession` table, and named-artifact observation through a counting fake.
   - Missing: the real `StatFile` conformance test above, and a test that expiry is reset by output, if that change is adopted.

6. **Architectural notes**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. The row parser and `idleExpired` are pure; `SessionAlive` is a thin exec wrapper.
   - **ARCH-PURPOSE:** pass. All three idle-cost causes are fixed. Couch-side cost was explicitly moved to #430/#431 in Revisions.
   - **ARCH-MOCK:** flag, the missing `StatFile` conformance test above.
   - **ARCH-CONSTRAINTS:** pass. Costs were measured live and pinned by counts.
   - **ARCH-SECURE:** pass. `StatFile` goes through `resolveArtifact`, which rejects symlinks and paths that escape the root.
   - **ARCH-ORDER:** flag, Minor. The watcher loop now carries more independent state (`quietAt`, `quietLogMod`, `outputSeen`, `turnPending`, `chosen`, `observationEpoch`, `boundRootNodeID`). The issue Log already defers an enum of phases (epoch, quiet, pending, bound) to the next change to the watcher; it should go first in that change.
   - **ARCH-FUNERAL:** pass. Title pollers end on EXITED, unbound watchers end on idle expiry, and the tracked-candidate map is cleared while the watcher waits for a turn.

7. **Plan revision recommendations:** none. The Spec and Done-when already describe the chosen-id walk policy and the expiry trade-off.

```findings
findings:
  - id: new
    severity: Important
    family: real-fake-seam-conformance
    title: |
      OSRuntime.StatFile has no direct test; the os.ErrNotExist contract callers depend on is pinned only by the fake
    detail: |
      ActivityForSession and ObserveNamedArtifacts branch on errors.Is(err, os.ErrNotExist). Add a temp-dir test: regular file gives the same FileEntry as ListFiles, a missing file is ErrNotExist, and symlink or non-regular entries are rejected.
  - id: new
    severity: Minor
    family: idle-expiry-activity-signal
    title: |
      IdleExpiry counts only Pair sends, but a chosen-id launch binds via the filename handshake with no send
    detail: |
      An operator typing directly into a chosen Claude pane after more than 24h idle never binds. Counting ScrollbackRaw movement as activity would keep the funeral and those bindings at about one stat of cost.
  - id: new
    severity: Minor
    family: walk-to-stat-behavior-parity
    title: |
      A final-component symlink in StatFile now errors in ActivityForSession, where the walk returned (false, nil)
  - id: new
    severity: Minor
    family: loop-state-flag-constellation
    title: |
      The sessionwatch loop adds quietAt, quietLogMod, outputSeen and turnPending flags without an explicit phase enum (ARCH-ORDER)
    detail: |
      Already deferred in the issue Log; the epoch, quiet, pending and bound enum should lead the next change to the watcher.
  - id: new
    severity: Minor
    family: vacuous-count-helper
    title: |
      OperationCountForRoot stays vacuous for ListFiles; only a lesson was recorded, not a fix to the helper
```

---

## Re-review — 2026-10-10T18:47:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 420 — idle sessions burn CPU: pair title / session-watch / couch poll forever |
| repo | pair |
| issue file | workshop/issues/000420-idle-sessions-burn-cpu-pair-title-session-watch-couch-poll-forever.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7afffc58f430c44b8b705e25dd2846c3df1bc3bd..53f20aded2b8d5cd83cc0e440389cfd3bee59b1d |
| command | sdlc close --issue 420 |
| reviewer | claude |
| timestamp | 2026-10-10T18:47:01-07:00 |
| verdict | SHIP |

## Review

Prior-round fixes check out so far. Next I'm checking `resolveArtifact` for symlinked directories in the middle of a path, then running the tests.

```verdict
verdict: SHIP
confidence: high
```

The round-2 fixes hold up against the code. I'm closing four of the five prior findings: three as addressed and one (BR-3) as withdrawn. BR-4 stays open as an accepted Minor deferral. I found one new Minor issue. Nothing blocks.

What I checked:
- **BR-1:** `TestOSRuntimeStatFileMatchesListFiles` tests the real `OSRuntime` against a temp dir. It covers a regular file (compared field by field with `ListFiles`), a missing file returning `os.ErrNotExist`, a directory returning `ErrNotRegularArtifact`, a symlink refused without being reported as missing, and an unknown root.
- **BR-2:** `wait()` now treats movement of the `ScrollbackRaw` recording as activity, measured against a baseline taken at launch. `TestRunAgentOutputDefersIdleExpiry` requires the watcher to exit at about 44h, so it would fail without the fix, which exits at 24h.
- **BR-3 was mistaken.** The old `ActivityForSession` also returned the walk's `listErr` when the transcript was a symlink, so stat and walk agree. The new test pins both refusals.
- **BR-5:** `OperationCountForRoot` now counts bare-root keys, and `TestOperationCountForRootIncludesRootLevelOperations` fails without that change.

Test runs:
- `sessionwatch`, `sessioninventory`, `sessioninventorytest`, `zellijsession` and `titlepoller` all pass.
- `launcher` fails only because of the known sandbox restriction and the leaked `PAIR_DATA_DIR`. Its zellij parse tests pass with `env -u PAIR_DATA_DIR`.

1. **Strengths**
   - `StatFile` reuses `fileEntryFor` and `resolveArtifact`, the same code `ListFiles` and `ReadFile` use. It keeps the existing checks against path escape and relative symlinks, so it opens no new trust gap (ARCH-DRY, ARCH-SECURE).
   - The gated idle pass is cheap: it skips the Pair-log read when the log's mtime hasn't changed (`run.go` `quietLogMod`), and it skips the catalog read along with the walk (`newIncremental` is lazy).
   - The idle-expiry baseline correctly ignores output written before launch (`outputActivitySeen` is initialised before the loop).
   - The issue Log records which mutation each test catches. I re-derived these from the tests rather than trusting the Log.
2. **Critical:** none.
3. **Important:** none.
4. **Minor**
   - **New, family `funeral-covers-every-wait-path`.** The v3 epoch-acquisition branch (`run.go`, `observationEpoch == nil`) calls `Observe` on the whole native store every `SlowPoll` and then `continue`s without going through `wait()`. If `launchBoundaries` stays incomplete, idle expiry never runs there. That happens when any file in the store is unreadable or has an invalid path. The watcher then walks the store for the agent's whole life, which contradicts the Done-when's unconditional idle-cost and expiry claims.
     - **Rule:** every unbound pass that sleeps must go through `wait()`.
     - **Fix:** in that branch, replace `rt.Sleep` plus the deadline check with `if wait() { return nil }`.
   - The `ScrollbackRaw` mtime is read twice per idle pass, once by `outputMoved` and once by `wait`. It's cheap; one read per pass could feed both.
5. **Test coverage:** All of the behaviour changed this round now has a test that would fail without the fix. The epoch-acquisition branch has no idle-cost test (see the new Minor).
6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. `idleExpired` and `zellijsession` are pure.
   - **ARCH-PURPOSE:** pass, with the epoch-branch gap noted.
   - **ARCH-MOCK:** pass. The fake now models `StatFile`, and the real conformance test closes BR-1.
   - **ARCH-CONSTRAINTS:** pass. The live probe numbers are logged.
   - **ARCH-SECURE:** pass.
   - **ARCH-ORDER:** flag, same as BR-4. The loop now carries six cross-pass variables: `quietAt`, `quietLogMod`, `outputSeen`, `outputActivitySeen`, `lastOutput` and `lastSend`. The deferred phase enum is getting more urgent.
   - **ARCH-FUNERAL:** pass, except the epoch-branch gap.
7. **Plan revisions:** none required. If the epoch-branch fix is deferred, add a Log line naming it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestOSRuntimeStatFileMatchesListFiles (runtime_os_test.go) exercises real OSRuntime on a temp dir: entry parity, ErrNotExist, non-regular, symlink, unknown root.
  - id: BR-2
    disposition: addressed
    note: |
      wait() counts ScrollbackRaw mtime movement after launch; TestRunAgentOutputDefersIdleExpiry requires about 44h, which fails at 24h without the fix.
  - id: BR-3
    disposition: withdrawn
    note: |
      The old walk also returned listErr when the transcript was a symlink (not false,nil); parity holds and the new test pins both refusals.
  - id: BR-4
    disposition: not-addressed
    note: |
      Still deferred, as logged in the issue; the loop now carries 6 cross-pass vars, so the phase enum should lead the next watcher change.
  - id: BR-5
    disposition: addressed
    note: |
      OperationCountForRoot adds the bare-root key; TestOperationCountForRootIncludesRootLevelOperations fails without it.
findings:
  - id: new
    severity: Minor
    family: funeral-covers-every-wait-path
    title: |
      The v3 epoch-acquisition branch walks the store each SlowPoll and bypasses wait(), so idle expiry never applies there
    detail: |
      When launchBoundaries stays incomplete (any unreadable or invalid-path artifact in the store), run.go's observationEpoch==nil branch Observes the whole store every SlowPoll for the agent's lifetime with no IdleExpiry. Rule: every unbound pass that sleeps goes through wait(); replace the rt.Sleep plus deadline check there with wait().
```

---

## Re-review — 2026-10-10T18:51:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 420 — idle sessions burn CPU: pair title / session-watch / couch poll forever |
| repo | pair |
| issue file | workshop/issues/000420-idle-sessions-burn-cpu-pair-title-session-watch-couch-poll-forever.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7afffc58f430c44b8b705e25dd2846c3df1bc3bd..109e22ba4f4987500d03d0268a123c72500e9724 |
| command | sdlc close --issue 420 |
| reviewer | claude |
| timestamp | 2026-10-10T18:51:52-07:00 |
| verdict | SHIP |

## Review

The tests pass and the BR-6 fix holds up, so I'm writing the verdict now.

```verdict
verdict: SHIP
confidence: high
```

This round's only new change is the post-close commit `99565c83`, which answers BR-6. It is ready to ship. The epoch-acquisition branch at `cmd/internal/sessionwatch/run.go:224` now goes through `waitFor(opts.SlowPoll)`, the same funeral (startup deadline, then idle expiry, then a send-aware sleep) that `wait()` uses. That makes the rule from the prior round hold structurally: every unbound pass that sleeps goes through `waitFor`. I checked every sleep site. The remaining direct `rt.Sleep` calls are at lines 189, 300 and 357, which are bound lifecycle-following passes, and in `waitForPairProcess`, which ends at `PIDWait`. None of them is an unbound pass. BR-4 is the only open item. It is Minor, was already deferred in the issue Log, and does not block. The targeted packages pass: `sessionwatch`, `sessioninventory/...`, `zellijsession` and `titlepoller`.

1. **Strengths**
   - The BR-6 fix covers the whole class, not just the one site. `waitFor(delay)` with `wait()` as its zero-delay alias leaves one funeral for every unbound sleep (`run.go:123-155`). The comment states the rule.
   - `TestRunIncompleteEpochStillExpires` (`cadence_test.go:278-302`) drives the real `Run` against the stateful fake runtime. It makes `ListFiles` fail with EIO so the baseline never completes, then checks that the watcher exits within `IdleExpiry + 2*SlowPoll`. Without the fix, `rootPID` is set, so the deadline never fires and the old loop runs forever. The test would hang to the go test timeout, so it really fails without the fix. I reasoned this from the code; I did not revert the fix to watch it fail.
   - As a side effect, the epoch-acquisition pass now also wakes early when a send lands, through `waitForScan`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Behaviour change worth noting in the Log: if the startup deadline passes during epoch acquisition, the watcher now logs `adapt.Fail` ("no completed native round within startup deadline"). The old path returned `nil` silently. This is consistent with the other unbound passes and is probably an improvement.
   - BR-4 is still open: there is no phase enum for the `quietAt`, `quietLogMod`, `outputSeen` and `turnPending` flags, and the new `waitFor` delay parameter adds one more input to that state.

5. **Test coverage:** the new test covers the incomplete-epoch expiry path. The earlier rounds' tests cover expiry deferred by agent output, deferred by sends, and the `StatFile` conformance.

6. **Architecture**
   - ARCH-DRY: pass. There is one funeral helper.
   - ARCH-PURE: pass. `idleExpired` and `scanDelay` stay pure.
   - ARCH-PURPOSE: pass. The class sweep was done.
   - ARCH-MOCK: pass. The test runs against the fake runtime through the production seam.
   - ARCH-CONSTRAINTS: pass. The epoch pass now runs at the `SlowPoll` cadence and is bounded by expiry.
   - ARCH-SECURE: N/A. The fix adds no new untrusted input.
   - ARCH-ORDER: flagged, but only through the existing deferral, BR-4.
   - ARCH-FUNERAL: pass. Every unbound pass now ends at `IdleExpiry`.

7. **Plan revisions:** none needed beyond the Log entry the commit already added.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Still deferred per the issue Log (Minor, non-blocking); waitFor's delay parameter adds another input to the same loop state, so the phase enum should lead the next watcher change.
  - id: BR-6
    disposition: addressed
    note: |
      run.go:224 now calls waitFor(opts.SlowPoll); every unbound sleep goes through waitFor, and TestRunIncompleteEpochStillExpires (cadence_test.go:278) would hang without the fix.
```
