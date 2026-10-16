# Boundary Review — pair#595 (whole-issue close)

| field | value |
|-------|-------|
| issue | 595 — Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store |
| repo | pair |
| issue file | workshop/issues/000595-replace-the-native-owner-forensics-556-584-with-a-wrapper-held-flock-per-conversation-one-time-cutover-delete-the-scan-and-reservation-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e93c3d9e0309d35f9993e3b52f5339480074081..71b211cc49811f3df0676c657e181578765cd7f6 |
| command | sdlc close --issue 595 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T04:07:38-07:00 |
| verdict | REWORK |

## Review

Reviewing the wrapper exit path in `wrap.go` now, then the watcher notice, selector, and recovery changes.

Still reviewing: the wrapper's lock/exit path is done (found a 1s exit delay and a new synchronous `sdlc` call at startup); now checking the watcher notice pipe, selector, and recovery changes.

Tests pass on all affected packages. Now checking whether the wrapper's late-collision and stop-during-notice paths have test coverage.

```verdict
verdict: REWORK
confidence: medium
```

The core of the change is sound and the deletion is real. The new `nativelock` package uses one non-blocking flock on a stable, owner-only file. Its lock/stop lifecycle logic (`Advance`) is pure and tested without IO. The bounded watcher pipe only reports bindings already committed to the ledger. The stop-set helpers moved over almost unchanged. Build and tests pass: `-race` on nativelock, sessionwatch, artifactpath and the wrapcmd lock/group/selector tests; plain runs of launcher, couchcore, couchcmd, termcmd, procutil and sessioninventory. The claimed net change matches the diff: production code −1,156 lines, tests −2,541.

Three things block the close:
- **Stale contracts.** The Spec, Done-when and plan still require the "relaunch older slots first" cutover gate and a readiness capability. The approved re-plan dropped both, and the code doesn't have them.
- **A dropped restart.** If a binding notice arrives after a restart has begun, the wrapper treats it as a lock failure and exits instead of restarting.
- **Slower exits and starts.** Every wrapper exit or restart now waits a full second, and every wrapper start first runs `sdlc workspace --json`.

### 1. Strengths
- `cmd/internal/nativelock/lock.go:25-90`: the directory and file are opened with `O_NOFOLLOW` through a dirfd, and checked for owner, mode and link count. The lock is taken before any write, a losing contender never truncates the file, and the file is never unlinked. This matches plan steps 2–4.
- `cmd/internal/nativelock/state.go`: the lifecycle (`Advance`) is pure and its state is immutable to callers. Late lock handles are released correctly after cleanup has finished. The sequence tests cover stopping, rejecting late discoveries and invalid events.
- `cmd/internal/sessionwatch/lifecycle.go:229`: the notice is sent only after the ledger append commits. A failed notice becomes a diagnostic and doesn't interrupt config publication (`TestBindingNotificationFailurePreservesConfig`).
- `recovery_terminal_confirmed_owner.go`: the four moved helpers are identical to the originals apart from the error text, and nothing used the removed error types.
- `TestSubprocessRelease` shows a real subprocess releasing the lock on both normal close and SIGKILL. `TestKnownNewAndForkDestinationsLockBeforeSpawn` pins the new/fork destination fix.

### 2. Critical
- **The plan and issue contradict the code (Core concepts, Spec, Done-when).**
  - Plan Core-concepts rows still list `launcher/native_lock_capability.go` (doesn't exist) and a modified `readiness/record.go` (unchanged in this window).
  - Spec (issue line 186) still promises Resume refusing with "relaunch older slots first: X, Y".
  - Done-when bullet 4 still requires legacy-capability tests.
  - The issue Log calls the re-plan "awaiting OK", while the atlas cites TL e93b109d as approving it.
  - The plan has no Revisions entry for the re-plan.
  - Fix: add the Revisions entry and rewrite those rows, the Spec paragraph and Done-when bullet 4 to match the re-plan: no gate, and pre-#595 wrappers hold no lock until relaunched.

### 3. Important
- **Late binding notice during a restart turns the restart into a failure** (`wrapcmd/wrap.go` masterPump notice case; `wrapcmd/native_lock.go:31-55`).
  - SIGUSR2 calls `locks.Stop()`. A notice that arrives before the PTY closes then calls `Discover`, and `Advance` rejects it with "discovery after stop".
  - That error is stored in `nativeLockErr`, which is checked before `restartFresh`. The wrapper exits 1 instead of re-executing.
  - The same applies to a watcher ID that passes `validBindingNotice` but fails `NewKey` (spaces, `/`, format characters): a diagnostic-only mismatch kills the agent group.
  - Fix: while stopping, or when a key is invalid, ignore and log the notice instead of treating it as a lock failure. Add a test that drives the notice channel during a restart.
- **Every exit and restart now takes a fixed one second** (`wrapcmd/owned_group.go:28-36`, ARCH-CONSTRAINTS).
  - `StartStop` always waits the full grace period unless the kill returns ESRCH.
  - When only the unreaped leader is left, Darwin returns EPERM (I measured this on this host), and Linux signals zombies successfully. So normal exit and restart always wait one second before the SIGKILL and reap.
  - The test uses a 1 ms grace, so it can't see this.
  - Fix: treat the grace as a maximum. Poll the group, or `waitid(WNOWAIT)` on the leader plus a group-empty probe, and finish early.
- **Every wrapper start now runs `sdlc workspace --json` synchronously** (`wrapcmd/native_lock.go` `nativeLockHolder`, ARCH-CONSTRAINTS / ARCH-MOCK).
  - It is capped at 2 s; I measured about 0.16 s here.
  - It runs before spawn, even for non-native wraps and Fresh launches with no known ID, and there is no seam or fake for it.
  - It only produces a display label. Fix: resolve the label lazily when a key is first acquired, from launch environment Pair already provides, behind an injectable seam.
- **No wrapper-level tests for the late-collision and controlled-exit paths** (Done-when bullets 2–3, ARCH-ORDER).
  - Nothing exercises the masterPump notice → `Discover` → `StartStop` → `nativeLockErr` path, nor the SIGTERM/HUP/SIGUSR2 handling in `run`.
  - Only the primitives are tested. The restart bug above is exactly what such tests would catch.

### 4. Minor
- `atlas/couch.md` "Owners" still lists the deleted `launcher/native_lock_capability.go`.
- The watcher-start-failure notice in `wrap.go` fires even when an exact ID is already held, contradicting "never claim… an already-held exact ID remains protected". It should be gated on `Unbound()`. Also, when there is no watcher at all, an unknown-ID Fresh launch gets no notice.
- `Acquire` fails outright on an invalid holder label, so a diagnostic-only value can block locking. Fall back to the generic label instead.
- `wrapperNativeLocks` discards the `Effects` that `Advance` returns, and its callers hard-code the stop (ARCH-ORDER).
- Three slightly different ID/label validators (`NativeLockFilename`, `validHolder`, `validBindingNotice`) should share one rule (ARCH-DRY / ARCH-SECURE). The mismatch is the root of the invalid-key kill above.
- The issue Plan item "Run full verification and SHIP review" is ticked before this review returned SHIP.
- The fd-3 notice writer is inherited by the watcher without close-on-exec, so any long-lived child the watcher spawns keeps the pipe open.

### 5. Test coverage notes
Primitive coverage is good: kernel contention, SIGKILL release, unsafe files, a fuzz test, state sequences, notice decoding, and committed-only notification. The gaps are at the wrapper level (Important above), and the owned-group test can't observe grace timing. I did not revert-and-rerun individual fixes, because this is the first round and there are no claimed fixes to check.

### 6. Architectural notes
- **ARCH-DRY:** flag (validators, Minor).
- **ARCH-PURE:** pass. Key and state are pure; the IO is in `lock.go` and the wrapper.
- **ARCH-PURPOSE:** pass. The scan and reservation store really are gone and the re-plan scope is delivered; only the documents lag.
- **ARCH-MOCK:** flag (the `sdlc` probe has no seam). Real flock and process tests run against temp directories, which is acceptable.
- **ARCH-CONSTRAINTS:** flag (fixed one-second exit, startup probe).
- **ARCH-SECURE:** pass. Checks are no-follow and owner-only, metadata is display-only, and filenames are hashed.
- **ARCH-ORDER:** flag (late notice while stopping is mishandled; no ordering test at wrapper level).
- **ARCH-FUNERAL:** pass with a note. One permanent lock file (about 4 KiB) is left per distinct conversation, with cleanup only by the operator. This is documented and measured; a quiescent sweep should exist eventually.

### 7. Plan revision recommendations
- Add "2026-10-16 re-plan (TL e93b109d)" to the plan's Revisions: the capability gate and the RecoveryIntent.Relaunch marker are dropped, and the transitional gap is accepted.
- Strike the Core-concepts rows for `native_lock_capability.go` and readiness, and the `launcher.LegacyLockSlots` seam row.
- Rewrite the issue Spec's cutover paragraph and Done-when bullet 4 to match.
- Record the TL approval in the issue Log.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: plan-claims-undelivered-surface
    title: |
      Spec, Done-when and plan Core concepts still require the dropped capability cutover gate
    detail: |
      Plan rows name launcher/native_lock_capability.go and readiness/record.go (absent/unchanged); issue Spec line 186 and Done-when bullet 4 still promise "relaunch older slots first" refusal and legacy-capability tests; the re-plan has no plan Revisions entry and the issue Log says "awaiting OK" while atlas cites TL e93b109d. Update plan/issue to the approved re-plan.
  - id: new
    severity: Important
    kind: defect
    family: stopping-rejection-treated-as-lock-failure
    title: |
      Late binding notice after a restart has started (SIGUSR2) turns the agent restart into a wrapper failure
    detail: |
      masterPump calls Discover after locks.Stop(); Advance rejects "discovery after stop", the error is stored in nativeLockErr and returned before the restartFresh check, so the wrapper exits 1 instead of re-execing. An ID valid for validBindingNotice but rejected by NewKey likewise kills the agent group. Ignore/log rejected notices while stopping and add a wrapper-level test.
  - id: new
    severity: Important
    kind: defect
    family: fixed-grace-instead-of-bound
    title: |
      ownedGroup always waits the full 1s grace, adding a second to every wrapper exit and restart
    detail: |
      StartStop only skips the timer on ESRCH; with only the unreaped leader left, Darwin returns EPERM (measured on this host) and Linux succeeds against zombies, so group.Wait always blocks 1s before SIGKILL and reap. The test uses a 1ms grace and cannot see it. Finish early once the group is empty.
  - id: new
    severity: Important
    kind: defect
    family: external-probe-on-startup-path
    title: |
      nativeLockHolder synchronously execs sdlc workspace --json (up to 2s) before every agent spawn
    detail: |
      It runs unconditionally, even for non-native wraps and unknown-ID Fresh launches, only to produce a diagnostic label, with no seam or fake (ARCH-CONSTRAINTS/ARCH-MOCK). Resolve lazily when a key is first acquired, from launch environment, behind an injectable seam.
  - id: new
    severity: Important
    kind: defect
    family: wrapper-ordering-untested
    title: |
      No wrapper-level tests for late collision, notice-channel handling or controlled signal/restart exits
    detail: |
      Done-when bullets 2-3 are covered only by the primitives; nothing drives masterPump notice to Discover to StartStop to nativeLockErr, or the SIGTERM/HUP/SIGUSR2 paths in run (ARCH-ORDER), which is how the restart bug above shipped.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-undelivered-surface
    title: |
      atlas/couch.md Owners list still names the deleted launcher/native_lock_capability.go
  - id: new
    severity: Minor
    kind: defect
    family: truthful-lock-status-notice
    title: |
      Watcher-start-failure notice fires even when an exact ID is already locked; a missing watcher gives no notice
  - id: new
    severity: Minor
    kind: defect
    family: diagnostic-label-blocks-lock
    title: |
      Acquire fails outright on an invalid holder label instead of falling back to the generic label
  - id: new
    severity: Minor
    kind: defect
    family: shared-identifier-validation
    title: |
      Three divergent ID/label validators (NativeLockFilename, validHolder, validBindingNotice) should share one rule
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0312b5fe-d8a9-46d1-868f-608248ac6572","repository":"github.com/xianxu/pair","issue_id":"000595","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/db47110353867fed","reviewed_head":"71b211cc49811f3df0676c657e181578765cd7f6","slot":"pair:2","requested_at":"2026-10-16T10:51:45.547552Z"},"phase":"finished","pid":22006,"started_at":"2026-10-16T10:51:46.012238Z","finished_at":"2026-10-16T11:07:20.651513Z","session_id":"9963600e-ec74-4ba4-9605-b9280c0825ca","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.693117s
head: "71b211cc49811f3df0676c657e181578765cd7f6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-16T04:34:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 595 — Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store |
| repo | pair |
| issue file | workshop/issues/000595-replace-the-native-owner-forensics-556-584-with-a-wrapper-held-flock-per-conversation-one-time-cutover-delete-the-scan-and-reservation-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e93c3d9e0309d35f9993e3b52f5339480074081..7a7b60867800e9c409f91bbd5c126082255655e6 |
| command | sdlc close --issue 595 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T04:34:35-07:00 |
| verdict | REWORK |

## Review

Checking the fixes now: running the wrapcmd/nativelock tests, then reverting a fix to confirm its regression test goes red.

Both BR-2 regression tests go red when the fix is reverted. Next: checking whether the holder label could come from the launch environment (BR-4).

```verdict
verdict: REWORK
confidence: medium
```

Most of round 1 is fixed, and I checked the fixes directly rather than relying on the commit messages. One Important finding is still open. **BR-3 (cleanup waits the full grace) is fixed on Darwin only.** `groupHasLiveMember` (`cmd/internal/wrapcmd/owned_group.go:49`) treats `kill(-pgid, 0) == nil` as "a live member remains". On Darwin, a group holding only zombies returns EPERM, so cleanup ends early. On Linux, signal 0 to a group whose only remaining member is our unreaped zombie leader still succeeds, so every wrapper exit and restart there still waits the full 1s grace and then sends SIGKILL. The fix's own comment covers only Darwin, and the atlas says these suites run on Darwin and Linux (`atlas/couch.md:4113`). I could not run Linux here: the Docker daemon is not running. Nothing else blocks.

1. **Strengths**
   - **BR-2 is properly fixed and has a real regression test.** I removed the stopping-phase guard in `Discover` and `TestRestartThenLateBindingNoticeStillRestarts` failed ("discovery after stop"). I changed the unusable-ID path back to `return err` and `TestUnusableBindingNoticeLeavesAgentRunning` failed. Both pass at the pinned head.
   - **The notice and signal logic moved into `handleBindingNotice` / `stopForSignal`** (`wrapcmd/native_lock.go:203-239`). Production (`wrap.go:3007`, `wrap.go:3221`) and the tests now call the same functions, so the wrapper-level tests exercise the real decision code (BR-5).
   - **The pure `nativelock.Advance` state machine is unchanged and still decides every transition.** The wrapper only adds a pre-filter that classifies events to drop; it does not change state outside the model.
   - **An unusable holder label now falls back to `"unlabelled"`** (`nativelock/lock.go:29`). The test shows the fallback label appears in the refusal message.
   - Build, vet and the targeted suites pass: `nativelock`, `sessionwatch`, `artifactpath`, `launcher`, and the lifecycle tests in `wrapcmd`.

2. **Critical:** none.

3. **Important**
   - **BR-3 is not addressed on Linux.**
     - **Rule:** deciding whether a process group is empty must not depend on how a platform answers signal 0 for zombies.
     - **Fix:** give `groupHasLiveMember` a Linux implementation that scans `/proc/*/stat` for members of the pgid whose state is not `Z`. Keep the kill-probe for Darwin. Run `TestOwnedGroupFinishesEarlyWhenTheGroupExits` on Linux; it should fail there today.

4. **Minor**
   - **The first late lock blocks the terminal output loop.** When a late binding notice arrives, `Discover` runs the label lookup (`sdlc workspace --json`) on `masterPump` while holding the lock mutex. That lookup took about 0.2s here and is capped at 2s. Known-ID launches also pay it before spawn.
     - This is the 2nd finding in family `external-probe-on-startup-path`.
     - **Rule:** a probe that only produces a diagnostic label never runs on the terminal I/O loop or before spawn. Start it once in the background when the wrapper starts, and have `Acquire` use the result if it is ready, otherwise the fallback label.
   - **Parts of the plan still describe the dropped cutover gate as current work.** The Architecture line (plan line 7), the capability sections (lines 64-70), test row 94 and unchecked checklist items 120 and 123 still say so. Item 123 is an operator acceptance step that would check for "Resume lists legacy blockers", a behaviour this build does not have.
     - This is the 2nd finding in family `plan-claims-undelivered-surface`.
     - **Rule:** when a revision drops scope, every active statement (summary, unchecked checklist items, test tables) is edited or struck where it stands. Only dated log and revision entries stay as history.
   - BR-7 and BR-9 are deferred, as the issue log records. They stay open as Minor.

5. **Test coverage**
   - BR-2 and BR-5 have regression tests that I confirmed fail without the fix.
   - The tests for BR-3 and BR-4 are real, but the BR-3 timing test has only run on Darwin.
   - The wrapper tests drive notice and signal handling through the production helper functions. They still see one interleaving only: a SIGUSR2 racing a notice that is mid-acquisition has no seam to control ordering. That is acceptable at this size.

6. **Architecture**
   - **ARCH-DRY:** pass. The validators are deferred under BR-9.
   - **ARCH-PURE:** pass. `Advance` is pure; the wrapper is the thin IO layer.
   - **ARCH-PURPOSE:** pass. Spec, Done-when and code now match the approved re-plan; see the plan Minor above.
   - **ARCH-MOCK:** pass. The label lookup is injected as a function and tests use fakes.
   - **ARCH-CONSTRAINTS:** flag. BR-3 on Linux, plus the minor stall on the output loop.
   - **ARCH-SECURE:** pass. Notice IDs are parsed through `NewKey`, and invalid ones are dropped with a log line.
   - **ARCH-ORDER:** pass with a note. Dropping notices after stop is explicit and tested; a collision that races a restart still reports a failure, which is defensible.
   - **ARCH-FUNERAL:** pass. Locks are released after the group is reaped; the lock files are bounded per conversation as the plan records.

7. **Plan revisions**
   - Add a "## Revisions" note, and strike or edit plan lines 7, 64-70, 94, 120 and 123 so the active plan and the operator acceptance checklist no longer describe the capability gate or legacy-blocker listing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue Spec line 40 and Done-when rewritten; plan rows struck and Revisions entry 147 added; leftover active plan prose raised as a new Minor.
  - id: BR-2
    disposition: addressed
    note: |
      Reverted the stopping guard and the errNoticeIgnored wrap; TestRestartThenLateBindingNoticeStillRestarts and TestUnusableBindingNoticeLeavesAgentRunning both go red, green at head.
  - id: BR-3
    disposition: not-addressed
    note: |
      Darwin only. On Linux kill(-pgid,0) succeeds while the unreaped zombie leader is in the group, so the poll still spends the full grace and SIGKILLs; the comment at owned_group.go:44 admits Darwin only, and atlas claims Darwin/Linux.
  - id: BR-4
    disposition: addressed
    note: |
      Label resolved lazily once behind an injected func (TestHolderLabelResolvesOnlyOnFirstLock); probe measured ~0.2s; remaining placement on the pump raised as a Minor.
  - id: BR-5
    disposition: addressed
    note: |
      handleBindingNotice/stopForSignal are the production call sites (wrap.go:3007,3221) and the new wrapper tests drive them against a real owned group.
  - id: BR-6
    disposition: addressed
    note: |
      atlas/couch.md:4108 Owners no longer lists native_lock_capability.go.
  - id: BR-7
    disposition: not-addressed
    note: |
      Deferred by the implementer; a missing watcher still gives no notice.
  - id: BR-8
    disposition: addressed
    note: |
      Acquire falls back to "unlabelled"; lock_test asserts the refusal names it.
  - id: BR-9
    disposition: not-addressed
    note: |
      Deferred by the implementer; three validators remain.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: external-probe-on-startup-path
    title: |
      First late lock runs the sdlc workspace label lookup on masterPump while holding the lock mutex
    detail: |
      2nd finding in this family. Rule: an external probe that only produces a diagnostic label never runs on the PTY I/O loop or before spawn. Start it once in the background when the wrapper starts and use its result if ready, else the fallback label. Measured about 0.2s per lookup, capped at 2s.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-undelivered-surface
    title: |
      Plan's active Architecture line, capability sections, test row and unchecked items 120/123 still describe the dropped cutover gate
    detail: |
      Rule: when a revision drops scope, edit or strike every active statement where it stands (summary, unchecked checklist, test tables); only dated log entries stay as history. Item 123 would have the operator check for "legacy blockers" that this build does not produce.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"623c738e-ae9f-42c0-8a4f-2af648ce93b5","repository":"github.com/xianxu/pair","issue_id":"000595","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6826bb3b7d0f3a67","reviewed_head":"7a7b60867800e9c409f91bbd5c126082255655e6","slot":"pair:2","requested_at":"2026-10-16T11:29:59.843353Z"},"phase":"finished","pid":44528,"started_at":"2026-10-16T11:29:59.967681Z","finished_at":"2026-10-16T11:34:12.076894Z","session_id":"f8ab1b46-694a-4e05-bc60-8af702feb6c4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.903280667s
head: "7a7b60867800e9c409f91bbd5c126082255655e6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T05:28:35-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 595 — Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store |
| repo | pair |
| issue file | workshop/issues/000595-replace-the-native-owner-forensics-556-584-with-a-wrapper-held-flock-per-conversation-one-time-cutover-delete-the-scan-and-reservation-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e93c3d9e0309d35f9993e3b52f5339480074081..c1a5e8b9c93b76608cd6507ae8618e38760d5458 |
| command | sdlc close --issue 595 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T05:28:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on open findings first. Let me look at the owned group code (BR-3).

Now revert the BR-3 fix (make groupHasLiveMember always true) and confirm the regression test fails.

I've finished checking the open findings and am writing up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The one Important finding still open, BR-3, is fixed and I confirmed the fix with a revert. Restoring the old darwin behaviour (`groupHasLiveMember` always true) makes `TestOwnedGroupFinishesEarlyWhenTheGroupExits` fail: it took 5.009s against a 5s grace. With the fix in place, the wrapcmd, nativelock, sessionwatch, launcher, artifactpath, couchcore and termcmd test packages all pass at head. BR-10 is also fixed: the slot-label lookup now runs once in a background goroutine at wrapper start (`wrap.go:2896`), and the locks are relabelled when it returns (`native_lock.go:37`). Nothing runs on `masterPump` or under the lock mutex except an in-place file write. Three Minor findings are still open: BR-7 (the watcher notice isn't truthful), BR-9 (three separate ID validators) and BR-11 (plan statements still describe the dropped design). None of them blocks the gate, and all three are cheap to fix before landing.

1. **Strengths**
   - `nativelock.Advance` is the only path for lock transitions. `wrapperNativeLocks.Discover/Stop/Close` (`cmd/internal/wrapcmd/native_lock.go:55-108`) go through it, and locks are released only after `CleanupComplete`, which runs after `group.Wait()`.
   - The live-member probe is per platform: kill(-pgid,0) on darwin, where EPERM means only zombies remain, and a /proc scan on Linux, where a zombie leader still answers kill. The Linux parser has a pure test, including command names that contain parentheses (`owned_group_test.go:94`).
   - `nativelock.Acquire` (`lock.go:25-84`) makes one nonblocking flock attempt with dirfd/O_NOFOLLOW checks for ownership, mode and nlink. A losing contender only reads, capped at a bounded length, and holder metadata cannot change the kernel's decision.
   - SIGUSR2 and the termination signals now share one controlled stop path (`stopForSignal`). The old delayed group-kill goroutine, which was never joined, is gone.

2. **Critical:** none.

3. **Important:** none new.

4. **Minor**
   - BR-7, still open: `wrap.go:2984` emits the "discovery unavailable" notice without checking `p.nativeLocks.Unbound()`. When `launchWatcherArgv` returns nil, or `os.Executable` fails (`wrap.go:2971`), there is no notice at all.
   - BR-9, still open: the three validators still disagree. `validBindingNotice` (`binding_notice.go:23`) accepts IDs containing spaces or Cf characters, and `NativeLockFilename` then rejects them, so the notice is dropped with only a debug line.
   - BR-11, partly fixed: three plan passages still describe the dropped capability/maintenance design.
     - Line 72: "replace admission with the small capability check".
     - Line 90: a test row for "Resume-only maintenance refusal and exact-source migration exemption".
     - Line 108: the estimate includes a "capability check".
   - New: `Handle.Relabel` writes and then truncates in place. A losing contender reading at that moment can see a torn label (e.g. "pair:2" written over "standalone" reads as "pair:2alone"). The impact is diagnostic only; truncating first, or writing a padded fixed-size record, would avoid it.

5. **Test coverage**
   - The BR-3 regression test is real and fails without the fix (verified above).
   - The Linux `groupHasLiveMember` path is covered only through the parser on this host; Linux CI exercises the rest.
   - `TestSetHolderRelabelsHeldLocks` covers relabelling. That the lookup stays off the I/O path is a structural property with no test that pins it, which is acceptable for a diagnostic label.

6. **Architecture**
   - ARCH-DRY: flagged, see BR-9.
   - ARCH-PURE: pass. Key, state and holder parsing are pure; the lock and group code are thin IO shells.
   - ARCH-PURPOSE: pass. The scan and the reservation store are fully deleted.
   - ARCH-MOCK: pass. Flock and process groups are tested against real temporary processes, with no external service.
   - ARCH-CONSTRAINTS: pass after BR-3. Cleanup ends early, the label lookup is capped at 2s and kept off hot paths, and the Linux /proc scan runs every 10ms only during cleanup.
   - ARCH-SECURE: pass. Lock files are owner-only with no-follow, holder text is validated, and the notice decoder is size-bounded with an ordinal fence.
   - ARCH-ORDER: pass. Transitions go through `Advance`; a late notice after stop is ignored and tested. Minor caveat: the label goroutine can outlive `locks.Close()`. That is harmless because `SetHolder` with no handles only sets a string, and the goroutine is bounded by the 2s timeout.
   - ARCH-FUNERAL: pass, with a note. Lock inodes are permanent while online, one small file per distinct conversation. The plan documents this, and reclamation requires quiescence.

7. **Plan revisions**
   - Strike or rewrite plan lines 72, 90 and 108 so the "capability check" and "maintenance refusal / migration exemption" are gone.
   - Reword item 118's "before the maintenance window" to match the rollout with no cutover.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Per-platform groupHasLiveMember ends cleanup early; reverting darwin probe makes TestOwnedGroupFinishesEarlyWhenTheGroupExits fail (5.009s), verified in checkout.
  - id: BR-7
    disposition: not-addressed
    note: |
      wrap.go:2984 still notifies unconditionally on start failure (no Unbound check); nil launchWatcherArgv and os.Executable failure at wrap.go:2971 still give no notice.
  - id: BR-9
    disposition: not-addressed
    note: |
      NativeLockFilename, validHolder and validBindingNotice remain separate rules; notice accepts space/Cf IDs that NewKey later rejects silently.
  - id: BR-10
    disposition: addressed
    note: |
      slotLockHolder now runs once in a background goroutine at wrap.go:2896; SetHolder only relabels under the mutex; TestSetHolderRelabelsHeldLocks covers relabel.
  - id: BR-11
    disposition: not-addressed
    note: |
      Architecture line, core concepts and checklist fixed, but plan line 72 (capability check), line 90 test row (maintenance refusal / migration exemption), line 108 estimate and item 118 (maintenance window) still describe the dropped gate.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: truthful-lock-status-notice
    title: |
      Relabel writes then truncates in place, so a contender can read a torn holder label
    detail: |
      This is the 2nd finding in family truthful-lock-status-notice. Rule: any holder text a refusal shows must be a complete label some owner actually wrote. nativelock/lock.go Relabel does WriteAt then Truncate; a loser reading between the two sees the new prefix plus the old suffix (e.g. pair:2alone). Diagnostic only; truncate first or write a fixed-size padded record.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4ab35a34-d877-454a-b82f-916d25983253","repository":"github.com/xianxu/pair","issue_id":"000595","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/884f93f09763acdf","reviewed_head":"c1a5e8b9c93b76608cd6507ae8618e38760d5458","slot":"pair:2","requested_at":"2026-10-16T12:18:59.372059Z"},"phase":"finished","pid":44568,"started_at":"2026-10-16T12:18:59.536766Z","finished_at":"2026-10-16T12:28:33.800339Z","session_id":"ea43f953-be3b-4c42-83fc-1654a928e401","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.952375666s
head: "c1a5e8b9c93b76608cd6507ae8618e38760d5458"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T06:02:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 595 — Replace the native-owner forensics (#556/#584) with a wrapper-held flock per conversation; one-time cutover; delete the scan and reservation store |
| repo | pair |
| issue file | workshop/issues/000595-replace-the-native-owner-forensics-556-584-with-a-wrapper-held-flock-per-conversation-one-time-cutover-delete-the-scan-and-reservation-store.md |
| boundary | whole-issue close |
| milestone | — |
| window | 614cd4c77793dd43b8cba6dcd93a4d858025b932..6e30fdbf474c2b22924ade77271eb845590ee983 |
| command | sdlc close --issue 595 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T06:02:26-07:00 |
| verdict | SHIP |

## Review

Checking where the wrapper takes its exact-ID lock, to see whether the new `Unbound()` gate in the discovery-unavailable notice is accurate.

Running the nativelock and wrapcmd test packages at head.

```verdict
verdict: SHIP
confidence: high
```

This window contains the fixes made after the close review. It changes the `Relabel` write order, the watcher-unavailable notice in `wrap.go`, the plan text, and a native conformance test that now waits for its wrapper to exit before removing the fixture. None of the four open prior findings is blocking. BR-11 and BR-12 are fixed. BR-7's code is correct but has no regression test. BR-9 had no work in this window. The new code adds no Critical or Important defects. `go vet` passes on the three changed packages. `go test` passes for nativelock (0.29s) and wrapcmd (73s). I did not run the couchtty conformance test, because it needs a live Zellij/native fixture.

1. **Strengths**
   - `cmd/internal/nativelock/lock.go:89-96`: `Relabel` now truncates before it writes. A contender reading in between sees either an empty label or the start of the new one. An empty label fails `validHolder` and is shown as "another Pair session". The old label's tail can no longer mix with the new one.
   - `cmd/internal/wrapcmd/wrap.go:2970-2997`: one `discoveryUnavailable` helper now covers all three failure paths: `os.Executable` failure, no watcher for the launch, and a pipe/start failure. It only shows the notice when the agent is a supported native agent and no ID is locked yet (`locks.Unbound()`).
   - `terminal_native_test.go:66-80`: the cleanup now waits up to 5s for the wrapper to exit, using its pidfile. If the pidfile is missing (for example, the test failed before its path was set), it skips the wait and removes the fixture as before.
   - The plan now describes the current design in the Delete list, the test table row and the LOC paragraph, and the landing item no longer refers to the dropped maintenance window.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** none new. Two prior Minor findings stay open:
   - BR-9: nothing in this window touches the three validators.
   - BR-7: the code fix is in place, but no test covers it.

5. **Test coverage notes**
   - The BR-12 race cannot be tested deterministically: any single-threaded `Relabel` test passes with both orderings. I accepted the fix on inspection.
   - The BR-7 notice gating has no test. A wrapper test with an injected nil watcher, run once unbound and once with an exact ID locked, would cover it.

6. **Architectural notes**
   - ARCH-DRY: flag, carried as BR-9. The interdiff itself removes duplication: three copies of the notice logic became one helper.
   - ARCH-PURE: pass. The notice predicate is small glue over `locks.Unbound()`, and the lock state machine stays in the pure `nativelock.Advance`.
   - ARCH-PURPOSE: pass. The BR-7 fix covers every way discovery can fail to start, not just the one site the finding named.

7. **Plan revision recommendations:** none needed. One wording leftover: the checked item at plan line 115 still says "maintenance rollout". It is a completed item, so it reads as history.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Code fix present and covers all three discovery-failure paths (wrap.go:2970-2997, gated on locks.Unbound()), but no regression test pins it; add an injected-nil-watcher test for the unbound vs exact-ID-locked cases.
  - id: BR-9
    disposition: not-addressed
    note: |
      No change in this window to validHolder, validBindingNotice or NativeLockFilename; they still disagree on whitespace and Cf characters.
  - id: BR-11
    disposition: addressed
    note: |
      Plan lines 72, 90, 108 and the landing item 123 now describe the no-capability, no-maintenance design; the remaining mentions are struck rows or dated Revisions entries.
  - id: BR-12
    disposition: addressed
    note: |
      lock.go:89-96 truncates before WriteAt, so a reader sees an empty label (shown as the generic holder) or a prefix of the new one, never the old suffix; the race cannot be tested deterministically, so verified by inspection.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f2a6fe34-862d-418d-a939-4b0e462ea108","repository":"github.com/xianxu/pair","issue_id":"000595","recipe":"fix-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9c85b80c13d6e7c2","reviewed_head":"6e30fdbf474c2b22924ade77271eb845590ee983","slot":"pair:2","requested_at":"2026-10-16T13:00:00.86147Z"},"phase":"finished","pid":45543,"started_at":"2026-10-16T13:00:01.00873Z","finished_at":"2026-10-16T13:02:24.896156Z","session_id":"85ac8ce7-2578-479f-8872-cb3e56a0f848","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.73144s
head: "6e30fdbf474c2b22924ade77271eb845590ee983"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000408
