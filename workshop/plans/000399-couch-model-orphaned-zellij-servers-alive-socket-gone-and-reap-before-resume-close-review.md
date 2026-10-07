# Boundary Review — pair#399 (whole-issue close)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7e4694c32a477b99bdb0a39f5edd0604ce14e661..83acde0eaf436b93f55b66a2510fe0bbf04391a3 |
| command | sdlc close --issue 399 |
| reviewer | claude |
| timestamp | 2026-10-07T12:38:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Review of pair#399, round 7 (whole-issue close): this can ship once three small leftovers are fixed, and none of them blocks the close. The recover/reap change is complete. Round 6's two missing fixes landed in 17124a86: a plan Revisions entry saying reap has its own switcher entry, and a corrected `ActorActions` comment. The M3 work added after round 6 holds together:
- **Recover no longer asks for confirmation.** The operator decided this on 2026-10-07, and it is recorded as a plan revision.
- **Old confirmation code is fully removed.** `ConfirmByPlan`, `prepare-recover` and the `steps`/`confirmed` arguments have no references left in code, README or atlas.
- **The live orphan is handled.** A thread Couch still hosts, whose server nothing else can reach, now carries `Orphan` and is offered `[recover, reap]`.
- **Recover shows progress.** It reports which step it is on, through a context-carried progress reporter.

Three Minor findings from round 6 are still open: BR-15, BR-16 and BR-17.

1. **Strengths**
   - `ActorActions` (`cmd/internal/couchcore/actor_actions.go`) checks `f.Orphaned` before looking at the row's state, so a live orphan can never be offered resume or reboot. Busy and unknown rows never carry `Orphan` because of `orphanOf`'s guard, so that ordering cannot leak to them.
   - Before each step, `Recover` re-reads the row through the same admission table (`awaitRecoverStep`) and waits a bounded 5 seconds. That handles a hosted client exiting a moment after its server is reaped, and `TestRecoverOnALiveOrphanReapsWaitsThenResumes` and `TestRecoverStopsWhenTheRowNeverAdmitsTheNextStep` pin that behaviour.
   - Removing `ConfirmByPlan` takes `OperationConfirms` back to two answers. That deletes a three-way caller obligation which the CLI, the socket and the switcher each had to handle.
   - `deriveRecoverSteps` is pure and table-tested, and when there is no report row it falls back to the single admission table (one source of truth, ARCH-DRY).

2. **Critical:** none.

3. **Important:** none.

4. **Minor** (all carried over, see dispositions below)
   - BR-15: `cmd/internal/launcher/session_reap.go` `OSProcessTable.Snapshot` still reads the parent pid from `ps` and the start time from a later `procutil.Identity` sysctl. To fix it, read both from the same `kinfo_proc`.
   - BR-16: three descriptions of reap are still out of date.
     - The `atlas/couch.md` paragraph describing reap (around line 2197) says `OSOrphanReaper` goes straight from the tree kill to the quiescence step. It still lacks the `ReapTagHelpers` step in between, and still describes `pair title` as the server's child.
     - The `lifecycle.go` comment at line 516 says the poller is outside the tree because of `Setsid`. That is false for Couch-launched Pair: `sidecarProcessAttributes` returns nil there.
     - `workshop/lessons.md` line 591 repeats the `Setsid` claim.
     - The real reason is that the launcher, not the zellij server, is the poller's parent.
   - BR-17: nothing compares `launcherCleanupOps.editorPaths` (`lifecycle.go:196`) with `editorPathsOf(paths)`. The test at `session_reap_test.go:222` only checks `editorPathsOf` against itself.

5. **Test coverage**
   - The tests drive the real `Recover`, `ActorActions` and menu dispatch paths rather than mocks.
   - The menu test `TestRecoverDispatchesWithoutAConfirmation` and the CLI test case rejecting `--confirm` pin the "never asks" behaviour.
   - Gap: no test checks that a fallback `[reboot]`-only plan (an unusable row whose resume is not offered) reboots without asking. The plan revision accepts that consent model, but a fixture pinning it would protect the envelope.

6. **Architecture, principle by principle**
   - **ARCH-DRY:** pass, apart from BR-17.
   - **ARCH-PURE:** pass. `deriveRecoverSteps`, `recoverReportRowFor` and `PlanReap` are pure.
   - **ARCH-PURPOSE:** pass. Reap, recover and the live orphan cover the issue's Done-when.
   - **ARCH-MOCK:** pass. A process-table seam and a helper-reaper seam have fakes.
   - **ARCH-CONSTRAINTS:** pass. The waits are bounded: 3 s after SIGTERM, 2 s after SIGKILL, and 5 s for a step to become admissible.
   - **ARCH-SECURE:** pass. The server's start identity is re-read before every signal.
   - **ARCH-ORDER:** pass, apart from BR-15. A hold refuses before any step runs, and each step is admitted against a fresh read of the row.
   - **ARCH-FUNERAL:** pass. Progress reporters live only in memory, and the removed preview operation leaves no residue.

7. **Plan revisions needed:** none beyond those already recorded.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Plan revision "2026-10-07 — reap is its own switcher entry too (M2)" (17124a86) records [recover, reap].
  - id: BR-14
    disposition: addressed
    note: |
      actor_actions.go ActorActions comment now names resume, reboot and reap, and the live-orphan exception.
  - id: BR-15
    disposition: not-addressed
    note: |
      session_reap.go OSProcessTable.Snapshot still takes ppid from ps and start identity from a later per-pid sysctl.
  - id: BR-16
    disposition: not-addressed
    note: |
      atlas/couch.md about line 2197 still skips ReapTagHelpers and lists pair title as a child; lifecycle.go:516 and lessons.md:591 still credit Setsid.
  - id: BR-17
    disposition: not-addressed
    note: |
      The only test (session_reap_test.go:222) compares against editorPathsOf itself; nothing compares it with the quit path's inline literal at lifecycle.go:196.
```
