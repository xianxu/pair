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
