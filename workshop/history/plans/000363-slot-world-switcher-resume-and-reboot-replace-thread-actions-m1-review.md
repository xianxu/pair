# Boundary Review — pair#363 (milestone M1)

| field | value |
|-------|-------|
| issue | 363 — Slot-world switcher: resume and reboot replace thread actions |
| repo | pair |
| issue file | workshop/issues/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 11adc0740d0fb9b18d788bd0ace1a1986760f122..8f3d00bb50f7beb39be7dc80630ff1b0672c860c |
| command | sdlc milestone-close --issue 363 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-02T23:43:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

M1 delivers what the plan claims. The `reboot` and routed `resume` operations are in couchcore. They are built from extracted shared pieces: `prepareRetirement`, `claimFreshRecord`, `launchClaimedThread`, and the `archiveJournalEntries`/`createJournalEntries` builders. A `:0` reboot now commits in one store journal (`ReplaceThreadExpected`), the counterpart of the slot store's `replaceSlotCurrent`. I inspected the pinned range (stat, name-status, targeted diffs for every source file) and ran the suites. All targeted tests pass (`Reboot|Resume|ReplaceThread|FreshSlot|Archive|OpenSlot|ContinuationRefuses|ChooseResume`: ok, 74s). The failures in the wider `couchcore`/`couchcmd`/`couchtty` runs are all sandbox "operation not permitted" (PTY, sockets, `/tmp`). `TestProductionArtifactReferencesAreExactlyClassified` fails only on the couchmessage/reviewcmd files that also fail on main; none of the new files are listed. Nothing blocks the boundary. The findings below are about wording and plan accuracy, which M2 will touch anyway.

**1. Strengths**
- **Ordering in reboot is correct.** In `reboot.go:110-130` the profile is resolved, its arguments validated and the family enrolled before `prepareRetirement` stops anything. `TestRebootProfileFailureStopsNothing` checks this: no quiesce happens, the row stays detached and nothing is archived.
- **The journal builders are extracted, not copied** (`threadstore.go:272-283, 1362-1384`). `ReplaceThreadExpected` refuses when the old and new records would land in different stores (`threadstore_replace.go:50`), and the crash, stale-revision, local-layout and slot-routing tests cover it.
- **Reboot admission is archive's rule by construction.** `RebootableState` delegates to `ArchivableState`, so the two cannot drift apart.
- **The warm-adoption guard is well reasoned** (`slotrecovery.go:433-437`). With a guessed agent, a record-less detached survivor is refused with a typed `ResumeSurvivorUnproven` rather than adopted. The plan Revision states this honestly.
- **`ResumeRebootAdvice` is checked against the source.** The test enumerates the declared `ResumeDiagnosticCode` constants with `go/parser`, so a new code cannot ship unclassified (the ARCH-PURPOSE "derived enumeration" lesson applied).

**2. Critical**
None.

**3. Important**
None.

**4. Minor**
- **Reboot advice names an action the switcher doesn't offer yet.** `resume_route.go:89`: `withRebootAdvice` says "Tab → reboot", but M1 declares `reboot` with `RowAction: false`. If M1 landed on its own, operators would be pointed at a missing action. That contradicts the plan's claim that every milestone leaves main releasable.
- **OpenSlot's refusals lost their next step.** At `slotrecovery.go:494-500` and `:530`, the old "choose Start fresh" exit is gone. The switcher's `open-slot` calls OpenSlot directly, not through `withRebootAdvice`, so in M1 those refusals name no next action even though `fresh-slot` is still offered.
- **Console continuation watch misses `resume`.** `console_continuation.go:230` only registers the watch when the operation is `recover-thread` or `recover-checkpoint`. A `resume` that routes to `RecoverThread`/`RetryContinuation` returns a `ContinuationResult` the console will not track. In M1 this needs a stale row to happen. In M2, where resume replaces recover-thread, it is the main path.
- **`SessionNotStopped` is dropped in `rebootSlot`.** At `reboot.go:241`, `r.SessionNotStopped` is never copied into the result. It is always false for a readable slot record today, but the field's contract is silently narrower for slots.
- **Resume's extra classification round isn't in the plan.** On ordinary records, `resumeRouted` now runs `classifyForAction` (one host-wide `list-sessions`) on every keypress before `ResumeContextWith`. That's acceptable off the UI path, but the plan's ARCH-CONSTRAINTS note counts the extra round only for reboot.
- **`--agent` is silently ignored on archive-only and refuse plans** in `Couch.Reboot`.

**5. Test coverage**
- **Covered:** every row of the plan's reboot sequence table: parked, detached ordering via the `AfterJournal` hook, refusals for live, busy and unknown rows, directory missing, unreadable `:0`, rolled back, profile failure, crash recovery, and the double-archive retry for both `:0` and `:1+`.
- **Done-when:** the end-to-end resume→reboot case is pinned by `TestRebootAfterResumeFailedIsUsable`.
- **Gap:** no test drives `resume` through the console to check how a `ContinuationResult` completion is handled. Add one in M2 alongside the `console_continuation.go` change.

**6. Architecture**
- **ARCH-DRY: pass.** Retirement, claim, launch and the journal builders are each shared, and `releaseRefusedSlotClaim` is generalized into `releaseRefusedClaim` rather than duplicated.
- **ARCH-PURE: pass.** `DecideReboot`, `ChooseResumeRoute` and `ResumeRebootAdvice` are pure and table-tested. `Couch.Reboot` and `ResumeTarget` are the IO shells.
- **ARCH-PURPOSE: pass for M1's scope.** See the plan revision below for the M2 consumer of `resume`'s `ContinuationResult`.
- **ARCH-MOCK: pass.** No new external dependency; the tests use the artifact fake, `FakeRunner` and the real store under `TempDir`.
- **ARCH-CONSTRAINTS: pass with a note.** Resume's extra classify round (Minor above) is not documented.
- **ARCH-SECURE: pass.** An unreadable record is never grounds to stop a session, and the revision check plus `archivableRecord` run under the lock.
- **ARCH-ORDER: pass.** The sequence table is implemented as written, and the interleavings that matter are pinned by tests with hooks (quiesce before the journal, a crash after the journal, a CAS refusal before the journal releasing the claim).
- **ARCH-FUNERAL: pass.** Archive records and grace files fall under the existing 60-day GC. The claim leak between `Claim` and the journal is documented as matching `AllocateThreadTag`'s existing window.

**7. Plan revision recommendations**
- **Task 2.6 Step 1 needs correcting.** It says the `console.go:1891` / `console_continuation.go:230` handling for `recover-thread`/`recover-checkpoint` should "become `reboot` or are deleted". It should become `resume`, because resume now routes to `RecoverThread`/`RetryContinuation` and returns their `ContinuationResult`. Reboot returns a `RebootResult`.
- **Task 2.6 should reword two more message sites**, before the switcher stops offering `fresh-slot`: the `withRebootAdvice` exit text and OpenSlot's refusals.
- **Extend the ARCH-CONSTRAINTS Revision** to cover resume's added classification round on ordinary records.

```findings
findings:
  - id: new
    severity: Minor
    family: refusal-names-unoffered-action
    title: |
      withRebootAdvice tells the operator "Tab → reboot" while M1 declares reboot RowAction false
    detail: |
      resume_route.go:89. Contradicts "each milestone leaves main releasable"; M2 Task 2.6 rewords it, or the M1 text should not name a missing switcher action.
  - id: new
    severity: Minor
    family: refusal-names-unoffered-action
    title: |
      OpenSlot refusals dropped the "choose Start fresh" exit while fresh-slot is still offered in M1
    detail: |
      slotrecovery.go:494-500,530. The switcher's open-slot calls OpenSlot directly (no withRebootAdvice), so the M1 refusal names no next action.
  - id: new
    severity: Minor
    family: result-consumer-keyed-by-op-name
    title: |
      Console continuation watch registers only for recover-thread/recover-checkpoint, not routed resume
    detail: |
      console_continuation.go:230. resume can now return ContinuationResult from RecoverThread/RetryContinuation. The plan's Task 2.6 says these sites "become reboot"; they must become resume.
  - id: new
    severity: Minor
    family: result-field-dropped
    title: |
      rebootSlot ignores prepareRetirement's SessionNotStopped
    detail: |
      reboot.go:241. Always false for a readable slot record today, but the RebootResult contract is silently narrower for slots.
  - id: new
    severity: Minor
    family: undeclared-runtime-cost
    title: |
      Routed resume adds a classifyForAction round per ordinary resume, not noted in ARCH-CONSTRAINTS
    detail: |
      resume_route.go:162. One host-wide list-sessions per keypress, off the UI path; the plan Revision should state it, as it does for reboot.
```
