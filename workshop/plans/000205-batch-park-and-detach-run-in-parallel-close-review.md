# Boundary Review — pair#205 (whole-issue close)

| field | value |
|-------|-------|
| issue | 205 — batch park and detach run in parallel |
| repo | pair |
| issue file | workshop/issues/000205-batch-park-and-detach-run-in-parallel.md |
| boundary | whole-issue close |
| milestone | — |
| window | b933b5a5bfef7fb681c5dffa2fd39c40d8765e7d..d3019e4331687d53afd7145ba79e27b921f493b1 |
| command | sdlc close --issue 205 |
| reviewer | claude |
| timestamp | 2026-10-07T13:25:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All four open findings are fixed at HEAD `d3019e43`, and each fix has evidence. BR-5 has a test that hits the zero-incarnation branch. BR-7 is fixed by reference-counted marks plus a test that runs both orderings. BR-8 and BR-13 are prose and naming fixes, confirmed by reading the code. The issue's own tests pass under `-race -count=3`: `couchcore` with the `Leave|Gate|Busy|Abort|ParkWorker|Registry|Retired|Recover` filter, and `couchtty` with the `Parallel|Refusal|Continuation|Reattach|Queue|Stop` filter. The full package runs failed only in pty and zellij conformance tests. Those failed because the sandbox blocks starting the pty child (`ptychild: start …: operation not permitted`), and the full `couchcore` run then hit the 10-minute test timeout. A known environment limit, not a regression. I found nothing new to raise. The window was examined with the stat and name-status recipes plus targeted reads.

1. **Strengths**
   - `threadgate.go:66-76`: admission is one pure rule, `gateDecision`. Holds are matched by token, not by address, so a context that outlived its release cannot get back into a later holder's hold.
   - `console_continuation.go:37-59`: `exitMarks` is reference-counted. Undo removes only the caller's own mark, while one real exit satisfies every owner. This is the shared rule for the `cleanup-scope-wider-than-own-effect` family, not a fix to a single site.
   - `park.go:211-228`: the `Leave` dispatcher checks `ctx.Err()` before and after the `select`. Go's `select` picks at random between a done context and a free unit, so the second check is needed. A test with parallelism 2 (`leave_test.go:177`) pins it.
   - `couch.go:878-889`: `startStillOwnsThread` requires at least one incarnation and every incarnation to match before it counts as ownership. `threadgate_couch_test.go:408` covers this.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
5. **Test coverage:** the tests synchronize on hooks rather than sleeps, use parallelism above 1 (`withParallelism`), and check both mark orderings. One sleep remains: a 30ms negative check at `console_continuation_test.go:520`. It was already in scope for BR-11, so I'm not raising it again.
6. **Architecture:**
   - ARCH-DRY: pass. One gate and one marks type.
   - ARCH-PURE: pass. `gateDecision` is pure.
   - ARCH-PURPOSE: pass. `Leave`, the park worker and the queue workers all draw on the single `LifecycleParallelism` bound.
   - ARCH-MOCK: pass. The fake thread store and process seams are reused.
   - ARCH-CONSTRAINTS: pass. The bound is half the cores, callers wait rather than being refused, and all of it is documented.
   - ARCH-SECURE: N/A. The gate holds only in-memory state and reads no untrusted input.
   - ARCH-ORDER: pass. Gate verdicts form an explicit enum, and the interleaving cells are tested.
   - ARCH-FUNERAL: pass. Holds end at release; marks are consumed on exit or undone on refusal.
7. **Plan revisions:** none. The M2 close already recorded the corrected cancellation policy.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      couch.go:880 returns false on zero incarnations; TestAnAbortOnARetiredThreadLeavesTheSessionAlone (threadgate_couch_test.go:408) reaches that branch.
  - id: BR-7
    disposition: addressed
    note: |
      exitMarks is now a refcount map (console_continuation.go:37); park's re-mark at console.go:1898 increments; TestARefusalNeverRemovesAnotherOperationsExitMark covers both orderings.
  - id: BR-8
    disposition: addressed
    note: |
      withoutDead's doc comment sits directly above it at couch.go:1316-1321; registry() has its own comment.
  - id: BR-13
    disposition: addressed
    note: |
      Renamed TestLeaveCancelledMidFanOutStartsNoFurtherThread (leave_test.go:177); its comment says started threads stop at their own safe points, matching park.go:196-200.
```
