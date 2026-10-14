# Boundary Review — pair#502 (whole-issue close)

| field | value |
|-------|-------|
| issue | 502 — couch: keep the UI at normal priority and everything agents spawn at nice 10 (a periodic renicer over each slot's process tree) |
| repo | pair |
| issue file | workshop/issues/000502-couch-keep-the-ui-at-normal-priority-and-everything-agents-spawn-at-nice-10-a-periodic-renicer-over-each-slot-s-process-tree.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a0355cb8ec51e3720c550a0dcfd6b038b7bd009..f4f6dfbbad2161a965dc2169f60e9fc7fce7a0fd |
| command | sdlc close --issue 502 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T17:09:38-07:00 |
| verdict | SHIP |

## Review

Vet hit a missing embedded asset in this checkout; running the tests package by package instead.

```verdict
verdict: SHIP
confidence: medium
```

The renicer is ready to ship. `couchcore.PlanRenice` is pure. It trusts a slot's agent only when the same snapshot shows the agent's parent is the binding's wrapper and the wrapper's start identity matches. It never uses process names, and it never touches Couch, Couch's ancestors, any wrapper or any agent. The IO shell (`couchcmd/renicer.go`) re-checks each process's identity before every `setpriority`. It skips processes that have exited (ESRCH), records each refusal once and never asks again, and drops a slot's state when its binding goes. #498's launch nice is fully removed, and `COUCH_SLOT_NICE` is now the renicer's level. The README and atlas are updated, the scan cost is measured, and the Log records the live check passing. I found nothing blocking; the findings below are minor accuracy issues in the reported facts.

One inspection gap explains the medium confidence: `couchcmd` doesn't build in this review checkout. The generated `runtimebundle` assets (`assets/runtime/files`, `manifest.json`) are missing, which has nothing to do with this diff. So I reviewed `renicer_test.go` and `nice_test.go` by reading them but couldn't run them. Tests that did run and pass: `procutil` (including `TestTableReadsNice`), `couchmessage` (including `TestBrokerConnectedAndNiceView`), and the `couchcore` Renice, BlockedStart and Issue149 tests.

**1. Strengths**
- **Confirming the agent** (`couchcore/renice.go:58`): it requires the agent's parent to be the wrapper, the wrapper's identity to equal `Binding.Start`, and a non-empty start. A test case covers each way this can fail: unread file, stale pid now pointing at a decoy, pid gone, wrapper pid reused, agent pid that is a grandchild, agent pid that is the wrapper itself (`renice_test.go:140-159`).
- **The never-set** covers Couch and its ancestors (with a hop bound against cycles in a snapshot) plus every slot's wrapper and agent, so nested slots are safe. `TestPlanReniceNever` checks this, including the case of Couch running under an agent.
- **The stateful fake** (`renicer_test.go:473`) models processes exiting, pids being reused (via the `afterTable` hook) and refusals, and a test runs against each. The live conformance test uses only the test's own `sh › sleep` tree.
- **Linux nice field** (`table_linux.go`): index 16 after the last `)` is stat field 19. This is consistent with the existing starttime index 19 (field 22).
- **The agent-pid file** goes through the existing hardened reader: `O_NOFOLLOW`, regular file, at most 32 bytes, canonical integer. That matches the writer (`wrapcmd/wrap.go:2984`, which writes a bare `Itoa`).

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- **Level 0 still shows a count** (`renice.go:76`, `messages.go:153`). With `COUCH_SLOT_NICE=0`, every descendant counts as `AtLevel`, so `--actors` prints `lowered=N` even though nothing runs low. The fix is to skip counting when `level<=0` (or hide the column) and add a test.
- **Misleading notice under a niced Couch** (`renicer.go:423`). The notice fires when `Agent.Nice > 0`. If Couch itself was started from a niced shell, every agent inherits that nice and gets the notice, whose advice ("relaunch the slot") can't help. Comparing against Couch's own niceness would fix it.
- **Plan drift on the seam.** The plan's `procSource` has `Identity(pid) (string, error)` and its own `ReniceProc` type. The code has `Identity(pid) string` and reuses `procutil.Process`. This is harmless, but the plan has no Revisions entry for it.
- **Over-cautious never-set** (`renice.go:51`). It also includes the `AgentPID` of unconfirmed roots, so a stale pid file can protect an unrelated pid inside another slot's tree. It errs on the safe side; this is just a note.

**5. Test coverage notes**
- The pure core has thorough table tests.
- The shell tests cover: work lowered only under the agent, an unconfirmed tree left alone, identity re-check on exit and pid reuse, a refusal not retried, the low-agent notice shown once, a gone slot's state dropped, level 0 only reporting, and `niceColumn` and `reniceInterval` parsing.
- `TestBlockedStartRunsTheTargetAtCouchsOwnNiceness` pins the removal of #498's launch nice.
- Not tested: the level-0 display issue and the niced-Couch notice above.
- Not run here: the `couchcmd` tests, because of the missing bundle.

**6. Architecture**
- **ARCH-DRY: pass.** It reuses `readMessageWrapperPID`, `procutil.Table`/`Identity` and `slotNiceSettings`.
- **ARCH-PURE: pass.** The decision lives in `PlanRenice`; the IO stays in the shell.
- **ARCH-PURPOSE: pass.** All the Done-when items are delivered, and the shell-prefix hook was correctly split off into #504.
- **ARCH-MOCK: pass.** The `procSource` seam has a stateful fake and a live conformance test against a real process tree.
- **ARCH-CONSTRAINTS: pass.** A pass costs about 0.44 ms, roughly 0.009% of a core at 5 s. The interval is bounded to 1 s–1 h, and slow passes drop ticks rather than queueing.
- **ARCH-SECURE: pass.** The pid file is treated as untrusted and confirmed against the snapshot. The renicer can only lower priority, and only after the identity re-check.
- **ARCH-ORDER: pass.** One goroutine runs on the service's lifetime and is joined through `workers`. The exit and reuse races are handled and tested through the fake's `afterTable` hook.
- **ARCH-FUNERAL: pass.** Everything is in-memory and keyed by binding; a pass prunes the entries of bindings that are gone.

**7. Plan revision recommendations**
- Add a Revisions entry noting that `procSource` uses `procutil.Process` and `Identity(pid) string`, not `ReniceProc` with an error return.

```findings
findings:
  - id: new
    severity: Minor
    family: reported-facts-match-state
    title: |
      COUCH_SLOT_NICE=0 makes --actors print lowered=N though nothing was lowered
    detail: |
      PlanRenice counts every descendant as AtLevel when level<=0 (renice.go:76), and niceColumn shows Lowered+AtLevel as "lowered". Skip counting at level 0 or hide the column; add a test.
  - id: new
    severity: Minor
    family: reported-facts-match-state
    title: |
      Low-agent notice compares against 0, not Couch's own niceness
    detail: |
      renicer.go:423 fires when Agent.Nice>0. A Couch started from a niced shell gives every slot a notice whose advice (relaunch the slot) cannot help. Compare against Couch's own nice.
  - id: new
    severity: Minor
    family: plan-matches-code
    title: |
      procSource seam signature differs from the plan with no Revisions entry
    detail: |
      The plan has Identity(pid) (string, error) and a ReniceProc type; the code has Identity(pid) string and procutil.Process.
```
