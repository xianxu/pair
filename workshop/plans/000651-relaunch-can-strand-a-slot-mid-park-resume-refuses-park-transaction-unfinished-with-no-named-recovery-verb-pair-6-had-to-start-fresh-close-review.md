# Boundary Review — pair#651 (whole-issue close)

| field | value |
|-------|-------|
| issue | 651 — Relaunch can strand a slot mid-park: resume refuses 'park transaction unfinished' with no named recovery verb (pair:6 had to start fresh) |
| repo | pair |
| issue file | workshop/issues/000651-relaunch-can-strand-a-slot-mid-park-resume-refuses-park-transaction-unfinished-with-no-named-recovery-verb-pair-6-had-to-start-fresh.md |
| boundary | whole-issue close |
| milestone | — |
| window | c8f3f03ef1cbc095d5909dac8f8da656cc7f38a0..59c8d4f0d0d42f03488638a6643476d7c9c2a869 |
| command | sdlc close --issue 651 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T17:14:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation preserves exact ownership, avoids manufacturing cleanup success, and exposes confirmed recovery through CLI and menu. However, recovery can still leave a worker blocked beyond cancellation, and the promised guidance sweep misses existing callers. Focused tests passed; the complete package run had failures described below.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: bounded-lifecycle-recovery
    title: |
      Recovery can remain blocked after its caller cancels
    detail: |
      cmd/internal/couchcore/park_recovery.go:201 calls reconcileAttempts without a context-aware completion seam. Production ObserveCompletion reaches Store.Reconcile and blocking LOCK_EX (park.go:37; pairlifecycle/store_unix.go:29), before any completion timeout starts. With a present terminal and cleanup holding that lock, cancellation returns from future.Await while the worker retains its address reservation indefinitely. The drive branch at park_recovery.go:222 also enters legacy blocking store operations. ARCH-CONSTRAINTS and ARCH-ORDER require bounded worker settlement, not merely a bounded caller wait. Carry cancellation through every reachable lock/read/write and add gated contention tests asserting worker settlement and reservation release.
  - id: new
    severity: Important
    kind: defect
    family: actionable-park-refusals
    title: |
      Existing unfinished-park paths still omit the executable recovery action
    detail: |
      The guidance guard at cmd/internal/couchcore/resume_route.go:158 comes after continuation routing at lines 152–156. Those routes can reach recovery.go:34, which still says “let lifecycle recovery finish”, or continuation.go:318, which invokes the old controller Recover and returns its error without ParkRecoveryAdvice. switchagent.go:364 likewise still advertises inaccessible park retry/recover/abandon modes. This leaves the Spec's “every unfinished-park refusal” and Plan's controller-error sweep incomplete (ARCH-PURPOSE, ARCH-DRY). Enumerate these callers, attach the shared exact-target advice, and test their production entry points.
```

1. **Strengths**

   - Retirement atomically tombstones the exact park and removes its matching incarnation without creating `VerifiedPark` (`park_recovery_store.go:34`).
   - Admission and menu confirmation retain the exact address and nonce, protecting replacement transactions.
   - The interruption matrix independently checks retained conversation authority, publication uncertainty, and all six cleanup stages.
   - README, atlas, help, and skill documentation cover the new command.

2. **Critical findings**

   - **Bounded recovery:** `park_recovery.go:201,222`. Introduce cancellation-aware lifecycle observation and store access throughout recovery. Test contention after initial admission, including cancellation while cleanup owns the lifecycle lock.

3. **Important findings**

   - **Incomplete guidance sweep:** `resume_route.go:152`, `recovery.go:34`, `continuation.go:318`, and `switchagent.go:364`. Route their unfinished-park errors through shared advice without weakening continuation or ownership guards.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused recovery/relaunch tests passed across `couchcore`, `couchcmd`, `couchmessage`, `couchtty`, and `storagegc`.
   - Complete five-package run: four packages passed; `couchcore` failed:
     - `TestSpawnComposesProductionPairRegistrationBoundary` failed both variants with “couch session repository scope changed”; reproduced in isolation. The tracker already records this test's pre-existing failure in pair#432.
     - `TestParkCoordinatorCoalescesExplicitRecoveryAndRetry` failed with “thread has no active park transaction”; passed in the isolated follow-up. Its 50 ms timing assumption does not establish that the retry joined before release.
   - Native opt-in tests were not enabled. Deferred operator acceptance remains explicitly documented.
   - Missing regressions are the contended-worker cancellation sequence and the remaining refusal entry points.

6. **Architectural notes**

   - **ARCH-DRY — flag:** remaining guidance callers bypass the shared formatter.
   - **ARCH-PURE — pass:** decision and outcome classification have direct tests without IO.
   - **ARCH-PURPOSE — flag:** actionable recovery is not yet present throughout the promised refusal class.
   - **ARCH-MOCK — pass:** existing stateful lifecycle/process/terminal seams and private native fixtures are reused.
   - **ARCH-CONSTRAINTS — flag:** blocking locks escape recovery's declared bounds.
   - **ARCH-SECURE — pass:** exact identity, nonce, revision, and positive absence checks protect retirement.
   - **ARCH-ORDER — flag:** cancellation can return while an indefinitely blocked worker retains authority.
   - **ARCH-FUNERAL — pass:** existing artifact retention is reused and tested; no new durable family appears.

7. **Plan revision recommendations**

   Add `## Revisions` entries enumerating all recovery IO requiring cancellation bounds and all unfinished-park advice callers. Record the new contention/entry-point regressions and distinguish the complete-run failures from the passing focused evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"df753325-bd15-4342-8c4b-292fb973ba1c","repository":"github.com/xianxu/pair","issue_id":"000651","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e08862185ea33ff0","reviewed_head":"59c8d4f0d0d42f03488638a6643476d7c9c2a869","slot":"pair:5","requested_at":"2026-10-18T00:06:07.484096Z"},"phase":"finished","pid":83985,"started_at":"2026-10-18T00:06:07.650432Z","finished_at":"2026-10-18T00:14:43.32567Z","session_id":"01a14c54-7d8b-7833-9aaa-f000812b1a73","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.563115541s
head: "59c8d4f0d0d42f03488638a6643476d7c9c2a869"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
