# Boundary Review — pair#421 (milestone M3)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 675ff48f07b8d1dd92d43e53ee1b6d8824b61e83..03fe25ac32fe5ee03868ef06b9ad9df2486a14f1 |
| command | sdlc milestone-close --issue 421 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-10T00:50:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm checking M3's plan rows against the code.

I'm checking whether a SIGUSR2 restart actually yields a new broker session.

I'm waiting on a mutation test: it removes the reload-context dispatch case to see whether any test notices.

The mutation went unnoticed by couchcore; the failures shown come from the sandbox. Next I'm checking couchcmd.

The couchcmd failures are sandbox socket denials, so that run settles nothing. I'll grep the tests for reload-context dispatch coverage.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** M3 adds `couch --reload-context repo:N`. The core of it is sound and matches the plan's revisions:
- the verb is declared like the other socket slot verbs;
- admission reuses relaunch's rule minus freshness;
- `RestartConversation` checks that the PID still belongs to the wrapper the broker holds before it sends SIGUSR2;
- success is reported only when a new broker session appears, and every other outcome is a typed `unconfirmed` failure.

I confirmed the restart path in the wrapper: a SIGUSR2 makes it close its session (close-on-exec on re-exec) and send a new hello, so a new session token is valid evidence. Nothing blocks the gate, but three gaps should be fixed while they're cheap:
1. **Dispatch is untested.** I deleted the `case OpReloadContext` from `CouchLiveOwnerExecutor` in a scratch copy and ran couchcore's tests. None of the failures had anything to do with reload-context. The only failures were the sandbox's pty denials and a git-less scratch tree. The couchcmd run was not conclusive, because the sandbox refuses its message sockets. But a grep shows no couchcmd test that dispatches reload-context either.
2. **Busy is checked only at admission.** The settled (busy) guard is not re-checked when the restart runs, although the comment admits the queue runs in between.
3. **The plan body still describes the old design.** It names a file and a shared helper that no longer exist.

### 1. Strengths
- `cmd/internal/couchcmd/live_restart_probe.go:92-124`: the signal target is the PID the broker holds, checked against `Binding.Start` through `ProcOps.Identity` (PQ-1). The wait is bounded and polled, and a cancelled or timed-out wait becomes `ReloadUnconfirmed` rather than success or absence. That is the right ARCH-ORDER handling of an uncertain outcome.
- `couchmessage/registry.go:342-364`: using the session token as restart evidence is correct. The plan's original "new Start or Nonce" test could never have changed after a re-exec, and the delta says so.
- `withAdmissionNote` was extracted, and `TestWithAdmissionNoteOnSuccessAndFailure` covers both paths. This closes the M2 advisory properly.
- The declaration tests (`ops_declarations_test.go`, the arity map in `run_test.go`) and the menu refresh list were all updated, so the new verb is fully declared.

### 2. Critical findings
None.

### 3. Important findings
- **reload-context's dispatch and its not-live check are untested** (`couchcore/operationdispatch.go:412-417`, `live_restart.go:194-196`). The test comment "Dispatch reaches it through the declared operation" (`slot_operation_test.go:306`) only checks `OperationConfirms`. The not-live refusal in `ReloadContext` is never exercised either.
  - This is the **3rd finding in family `safety-guard-wiring-untested`**, so the fix should be a rule, not this instance.
  - **Rule:** every operation declared `ExecuteLiveOwner` must be reachable through `CouchLiveOwnerExecutor`. Add one table test that walks `Operations()`, dispatches each live-owner op through `DispatchOperation`, and asserts the error is never the executor's unknown-operation error.
  - Separately, add a not-live case for `ReloadContext`: the thread has no occupied incarnation, so the result is `LiveRestartNotLive` and the probe records no restart.
- **The busy guard isn't re-checked when the restart runs** (`couchcmd/live_restart_probe.go:93-108`, `couchcore/live_restart.go:174-178`). Admission requires Settled. The queue then runs, and `ReloadContext` re-checks only liveness, even though its own comment says the queue ran between admission and now. A turn that starts in that window gets SIGUSR2 and is interrupted, which is exactly what the busy refusal exists to prevent.
  - The fix is cheap. `RestartConversation` already holds `before.Settled`; refuse with `LiveRestartBusy` or `LiveRestartBusyUnknown` unless it is known true.
  - Apply the same check to `Relaunch` (the whole class is these two effects).
  - Add a test where the snapshot flips to unsettled between admission and the effect.
- **The plan body still describes M3's old design** (plan lines 85, 89-93, 124-128).
  - The Core-concepts row says `ReloadContext` lives in `cmd/internal/couchcore/reload_context.go`. It actually lives in `live_restart.go`, and the signal is in `couchcmd/live_restart_probe.go`.
  - The bullet "It shares one helper with `agentcmd` … `SignalWrapper`" is not marked superseded.
  - M3's Tests line still asks for "helper extraction, keeping `agentcmd` tests green".
  - The M3 implementation deltas say no helper exists, but none of those body sites is marked.
  - This is the **3rd finding in family `revision-supersedes-body-unmarked`**. **Rule:** every Revisions delta must name each identifier or path it replaces, and that delta's commit must grep the plan body for those identifiers and mark every hit `(superseded: …)`, including table rows, prose and milestone rows. Apply that sweep for `reload_context.go`, `SignalWrapper`/`SignalVerifiedWrapper` and "helper extraction".

### 4. Minor findings
- `slotOperationOutcome` sets no `Tag` on a reload-context success, although its doc says "a success names the thread left running". Either add `ReloadContextResult` to the tag mapping or reword the doc.
- `--same-binary` is accepted silently with `--reload-context` (`SlotOperationTakesOverrides`). The README omits it there and the delta admits it has no effect. Refusing it would be clearer than ignoring it.
- `op == OpRelaunch || op == OpReloadContext` is repeated in `slot_operation.go:43` and `:153`. A single `isLiveRestartOp` would serve both (ARCH-DRY, small).
- `RestartConversation`'s cancelled-context path is untested; only the deadline path is covered.

### 5. Test coverage notes
- `RestartConversation` is covered well: a mismatched identity sends no signal; a matching one signals and is confirmed by a new session; an unchanged session gives unconfirmed; no session is refused. It uses `FakeProcOps` and an injected short deadline and poll interval.
- Missing: dispatch reachability, the not-live recheck, the busy-at-effect case, and the cancelled wait.
- The scratch-copy tests also failed on sandbox pty and socket denials, which are environment-only.

### 6. Architectural notes for upcoming work
- **ARCH-DRY:** pass apart from the minor `||` predicate repeat. Not sharing a helper with `agentcmd` is justified (it would have one caller).
- **ARCH-PURE:** pass. Admission stays in pure `DecideLiveRestart`, and the IO stays behind `LiveRestartProbe`.
- **ARCH-PURPOSE:** pass. The verb, the receipt, the README and atlas all ship together. The skill text belongs to M4.
- **ARCH-MOCK:** pass. `FakeProcOps` and the fake probe sit behind the production seams.
- **ARCH-CONSTRAINTS:** pass. The wait is bounded to 20s, polling every 100ms. It holds the thread for that time, which is acceptable.
- **ARCH-SECURE:** pass. The signal target comes from the broker-held binding, checked against the kernel start token, never from a pid file.
- **ARCH-ORDER:** flag. The settled guard is checked when the call is admitted but not when the restart runs (Important above). Otherwise the uncertain outcome is handled correctly.
- **ARCH-FUNERAL:** pass. Nothing durable is created, and the timer and ticker are stopped.

### 7. Plan revision recommendations
- Add a Revisions entry: "2026-10-10 M3 sweep: `ReloadContext` lives in `couchcore/live_restart.go`, and the verified signal is in `couchcmd/live_restart_probe.go` (`RestartConversation`). Mark the Core-concepts row, the `SignalWrapper` bullet and M3's 'helper extraction' Tests line as superseded."

```findings
findings:
  - id: new
    severity: Important
    family: safety-guard-wiring-untested
    title: |
      reload-context dispatch case and its not-live check are untested (deleting the executor case fails no reload-context test)
    detail: |
      3rd in family. Rule: every ExecuteLiveOwner operation in Operations() must be reachable through CouchLiveOwnerExecutor. Add a table test that dispatches each one via DispatchOperation and asserts no unknown-operation error, plus a ReloadContext not-live case (LiveRestartNotLive, no probe restart). Mutation run: with the case removed, couchcore's only failures were sandbox pty denials and a git-less scratch tree.
  - id: new
    severity: Important
    family: guard-rechecked-at-effect
    title: |
      Settled (busy) guard is checked at admission but not re-checked under the thread hold before SIGUSR2 or relaunch
    detail: |
      The ReloadContext comment admits the queue runs between admission and effect, yet it re-checks only liveness. RestartConversation already holds before.Settled; refuse unless known true, apply the same check to Relaunch, and add a test where the snapshot flips to unsettled between admission and effect.
  - id: new
    severity: Important
    family: revision-supersedes-body-unmarked
    title: |
      Plan still places ReloadContext in reload_context.go and describes a shared SignalWrapper helper and helper-extraction tests
    detail: |
      3rd in family. Rule: each Revisions delta names the identifiers and paths it replaces, and its commit greps the plan body and marks every hit superseded (table rows, prose, milestone rows). Sweep reload_context.go, SignalWrapper/SignalVerifiedWrapper, and "helper extraction".
  - id: new
    severity: Minor
    family: typed-outcome-survives-boundary
    title: |
      A reload-context success receipt carries no Tag, unlike the documented "names the thread left running"
  - id: new
    severity: Minor
    family: override-accepted-without-effect
    title: |
      --same-binary is silently accepted with --reload-context; refuse it instead
  - id: new
    severity: Minor
    family: safety-guard-wiring-untested
    title: |
      RestartConversation's cancelled-context path has no test
```

---

## Re-review — 2026-10-10T00:56:29-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 675ff48f07b8d1dd92d43e53ee1b6d8824b61e83..e804157197c3868fa4a95604851f5ff4ef046afd |
| command | sdlc milestone-close --issue 421 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-10T00:56:29-07:00 |
| verdict | SHIP |

## Review

Checking the plan's Revisions section and running the touched packages' tests.

Sandbox /tmp denials only; rerunning with TMPDIR scoped.

Now a mutation check: remove the reload-context dispatch case and the relaunch re-check in a scratch copy to confirm the new tests fail.

```verdict
verdict: SHIP
confidence: high
```

All six open findings (BR-14 to BR-19) are fixed, and the fixes are backed by tests that fail without them. I checked the two behavior-changing fixes by mutation in a scratch copy:
- **BR-14:** with the `OpReloadContext` dispatch case removed, `TestEveryLiveOwnerOperationIsDispatched` fails with "reload-context has no live-owner dispatch case".
- **BR-15:** with relaunch's `ConfirmNotBusy` call removed, `TestLiveRestartGuardRecheckedAtEffect` fails.

`couchmessage` passes. In `couchcmd`, the only failure is the git-push failure in `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`, which has nothing to do with this diff. The `couchcore` tests I ran (`-run 'LiveOwner|LiveRestart|ReloadContext|SlotOperation|Relaunch'`) are green. The sandboxed runs showed only `/tmp` permission denials. Two Minor findings remain, and neither blocks the boundary.

1. **Strengths**
   - `refuseKnownBusy` (`couchcmd/live_restart_probe.go:153`) is one rule shared by both verbs at the effect (ARCH-DRY). Reload-context applies it after the thread hold, inside `RestartConversation`.
   - Relaunch's re-check runs before `c.Relaunch` takes its hold. That costs nothing: `hold` refuses rather than waits (`threadgate.go:163`), so no queued operation can slip in between.
   - The override flags are now split into `SlotOperationTakesSameBinary` and `SlotOperationTakesForceUnknown`. `ParseCLI`, `parseMessageCLI` and `ValidateRequest` all read from them, so a flag that has no effect is refused instead of silently ignored.
   - The dispatch test enumerates `Operations()` and needs no hand-kept list. That makes it the rule for this finding's whole class, not a patch for one site.
   - Lessons record the rule for each of the three families.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `couchcore/slot_operation.go:39` still carries the doc comment for the deleted `SlotOperationTakesOverrides`.
     - The plan's Revisions section still says "One verb list: `SlotOperationTakesOverrides`" and "`--same-binary` is accepted but has no effect".
     - The BR-14..16 delta replaced both without naming them.
   - `ConfirmNotBusy` lets "unknown" through on the assumption that admission already demanded `--force-unknown`. That assumption fails when admission saw a known-settled wrapper that then disconnected. ARCH-ORDER warns against exactly this: treating an unknown outcome as a pass without evidence.

5. **Test coverage**
   - BR-17, BR-18 and BR-19 are each pinned by an assertion: `out.Tag`, `TestReloadContextRefusesSameBinary` plus the protocol table, and the cancelled-context `ReloadUnconfirmed` check.
   - In the BR-15 test fixture, `PairLifecycle` is nil, so the "not parked" assertion is weak on its own. The error-code assertion is what catches the regression.

6. **Architecture**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. The busy rule is a pure helper behind the probe seam.
   - ARCH-PURPOSE: pass.
   - ARCH-MOCK: pass. The fake probe records its busy checks.
   - ARCH-CONSTRAINTS: pass.
   - ARCH-SECURE: pass. `require-settled` is an implicit argument that only admission sets.
   - ARCH-ORDER: flagged as Minor, see the second Minor finding above.
   - ARCH-FUNERAL: pass. Nothing durable is created.

7. **Plan revisions:** add one line to the BR-14..16 delta saying it supersedes `SlotOperationTakesOverrides` and the earlier "`--same-binary` accepted" bullet.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      TestEveryLiveOwnerOperationIsDispatched enumerates Operations(); mutation removing the OpReloadContext case goes red; TestReloadContextRefusesWhenNoLongerLive covers not-live.
  - id: BR-15
    disposition: addressed
    note: |
      refuseKnownBusy in RestartConversation (under hold) and ConfirmNotBusy via require-settled for relaunch; mutation removing ConfirmNotBusy fails TestLiveRestartGuardRecheckedAtEffect.
  - id: BR-16
    disposition: addressed
    note: |
      Plan lines 85, 94, 130 now mark reload_context.go, SignalWrapper and helper extraction superseded inline.
  - id: BR-17
    disposition: addressed
    note: |
      ReloadContextResult.ReceiptTag plus notedResult passthrough; asserted in TestRestartEffectRechecksBusyAndCancellation.
  - id: BR-18
    disposition: addressed
    note: |
      SlotOperationTakesSameBinary gates CLI and ValidateRequest; TestReloadContextRefusesSameBinary and protocol_test cover it.
  - id: BR-19
    disposition: addressed
    note: |
      Cancelled-context path asserts ReloadUnconfirmed containing "cancelled".
findings:
  - id: new
    severity: Minor
    family: revision-supersedes-body-unmarked
    title: |
      Deleted SlotOperationTakesOverrides still named in a code doc comment and in the M3 Revisions bullets
    detail: |
      4th in family. slot_operation.go:39 keeps the old function's doc comment, and the plan's M3 deltas still say SlotOperationTakesOverrides is the single list and --same-binary is accepted without effect. Rule: when a commit removes or renames an identifier, run git grep for it across code and plan in that commit, then delete or mark superseded every remaining hit.
  - id: new
    severity: Minor
    family: guard-rechecked-at-effect
    title: |
      ConfirmNotBusy passes "unknown" at the effect even when admission saw a known-settled wrapper
    detail: |
      2nd in family. The comment assumes admission required --force-unknown, but a wrapper that disconnects between admission and effect turns known into unknown without operator consent (ARCH-ORDER: uncertainty collapsed into a pass). Rule: the effect re-check reads the same decision admission made. Carry whether admission was forced in require-settled (for example "known" or "forced") and refuse unknown unless it was forced.
```
