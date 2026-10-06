# Boundary Review — pair#387 (milestone M3)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | fd7da05196f34ccac2b6c58653fed802a268d565..548f95467784780105e80535e4eb6d67a9479be7 |
| command | sdlc milestone-close --issue 387 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-05T21:06:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M3 does most of what it set out to do. The saved-work lifecycle has an owner now. Removing the `RegisterStore` hook took out a real hazard, and Revision (w) explains why. The recovery report has a closed `EvidenceReconcile` dimension, tested by a metamorphic sweep. The two real-git acceptance tests check the right things: the dependency's local-only commit survives, the host's dirty file is left exactly as it was, and a second run does nothing. The focused tests pass (`go test ./cmd/internal/couchcore -run 'TestRecover|TestSavedWork|TestReconcileResult|TestAcceptanceDirty|TestAcceptanceForeign|TestDeriveRecoverPlan'`, ok in 50 s).

Four Important problems block SHIP. All are cheap to fix:
1. **Wrong restore command.** The restore command the result now prints does not restore. I checked this: `mv tree dep` on a re-cloned `dep` creates `dep/tree`.
2. **"no rule matched".** Every new report class gets that reason text, so the `:0` hand-off arrives with no cause attached.
3. **Hold on usable slots.** `RecoverSlotClass` puts a slot whose checkout works on hold, which disagrees with `SlotOutcome`, the function it says it shares a source with.
4. **Skill not updated.** Task 3.6 is ticked, but the couch skill still says "Nothing is ever deleted" and doesn't cover the new classes.

**1. Strengths**
- `collectSavedWork` (`slotsave.go:150-183`) never follows or removes symlinks. It falls back to the entry's own age when there is no manifest. The test covers old, young, manifest-less and symlink entries, plus collection during a real reconcile run. The ARCH-FUNERAL entry in Revision (w) gives the full lifecycle: creator, last reader, remover and bound.
- Removing `RegisterStore` (Revision w) was the right call. The reasoning that a slot backend would have been read as a pseudo-namespace is sound.
- `TestRecoverReconcileReadingIsMetamorphic` checks rules stated independently of the code (relative to the converged reading). It does not restate `classifyRecover`, and it is stride-sampled with a floor on the number of points.
- `recoverSlotPlans` only plans slots that `Discover` knows about. This avoids advising someone to create a slot that is known only from a dangling claim (`recoverplan_source.go:179-184`).
- `TestAcceptanceDirtySlot` checks preservation through git itself (`log -1` in the set-aside tree), not by checking that files exist.

**2. Critical**
None.

**3. Important**
- **Wrong restore command** (`slotsave.go:134`, surfaced at `provision.go:110-114`). The command is `mv "<entry>/tree" "<path>"`, but it is printed after `<path>` has been re-cloned or re-added. `mv` into an existing directory nests the tree, so the restore doesn't happen. Two more problems in the same area:
  - When a later step fails after a set-aside, the blocking path returns before `SetAside` is filled in, so the operator never sees the restore pointer.
  - `%q` uses Go quoting, not shell quoting. `ShellQuote` already exists.
  - **This is the 2nd finding in family `advice-names-reachable-action`.** The rule (already in `lessons.md` this window): every printed command is run, in a test, from the state that prints it. Here that means:
    - the restore command first moves the recreated checkout aside, then moves the tree back, using `ShellQuote`;
    - `TestAcceptanceDirtySlot` runs `r.SetAside[0].Restore` through `sh -c` and asserts `untracked.txt` is back at `dep`;
    - the failure path also carries the set-aside list.
- **"no rule matched" for the new classes** (`recoverplan.go:1534`). `recoverReason` has no case for `RecoverSlotNeedsZero` or `RecoverReconcilable`, so both fall into the default text. Plan Task 3.4 promised needs-zero "with `ReconcileAdvice`", but the reconciler's resource and cause never reach the row. The `RecoverDirectoryMissing` text ("repairing it is pair#387") is also stale now.
  - **This is the 4th finding in family `error-surface-preserved`.** The rule: any consumer of a `SlotReport` carries the cause the report already knows (`stopFailure`/`ReconcileAdvice`) into its own text, and `recoverReason` must cover every value in `AllRecoverClasses()`. Enforce both with a test that sweeps every class and rejects the default text.
- **Usable slots put on hold** (`recoverplan.go:518`). `RecoverSlotClass` maps any `StopHandoff` to needs-zero. `SlotOutcome` (`slotfailure.go:197-211`) only blocks when `OutcomeSeverity` is blocking; otherwise a hand-off stop is just a warning.
  - The fixture "workspace needs :0" (branch checked out elsewhere, host usable) therefore gets `workspace-handoff` and loses its `resume` step, even though resume would succeed.
  - This is exactly the case the new lesson ("Classify an outcome by what it means to the caller") describes. It also breaks ARCH-DRY: the same decision is made in two places, which is the lesson's BR-12 shape again.
  - Fix: derive needs-zero from `SlotOutcome(...)` returning blocking, and turn a hand-off stop on a usable workspace into a note.
  - Also, `len(r.Plan.Retried) > 0` can never be true in the report, because `PlanInput` has no `Attempted`.
- **Skill not updated, though Task 3.6 is ticked** (`cmd/internal/couchcmd/skills/couch/SKILL.md:125-127`). It still says "Nothing is ever deleted", which retention collection now contradicts. It also doesn't tell a recovering agent what to do for `slot-needs-:0`/`workspace-handoff`, the `reconcile` step, or the `workspace-held`/`workspace-unknown` notes.
  - **This is the 2nd finding in family `readme-tracks-user-surface`.** The rule: when a diff adds a class, hold or note, or changes a lifecycle statement, sweep every document that lists that vocabulary in the same window (README, atlas and the couch skill). A test can derive the skill's vocabulary mentions from `AllRecoverClasses`/`AllRecoverHolds`.

**4. Minor**
- `classifyRecover` replaces `d` for idle + reconcilable (`recoverplan.go:661`), so notes computed earlier (such as `claims-stale`) are dropped.
- `SavedWorkSince` finds "this run's" entries by a wall-clock window, not by identity (ARCH-ORDER). `setAside` could return its entry path in `Executed` instead.
- A `saved_at` set in the future (hand-edited) is never collected (ARCH-SECURE). Clamping to the entry's age would fix it.
- The report's per-slot observe budget (≤150 ms per slot, measured with 10 slots, Task 3.4) was not measured, and no revision records that (ARCH-CONSTRAINTS).

**5. Test coverage notes**
- No test runs a printed restore command.
- No test checks report reason text for the new classes.
- The "workspace needs :0" fixture asserts the misclassification above. Its expected result should become resume plus a note.
- The `index.lock` → unknown row is covered through the IO seam instead (Revision y explains why). That's acceptable.

**6. Architecture**
- ARCH-DRY: flag. `RecoverSlotClass` duplicates `SlotOutcome`; restore quoting duplicates `ShellQuote`.
- ARCH-PURE: pass. `RecoverSlotClass` and `classifyRecover` are pure; the shell is `recoverSlotPlans`.
- ARCH-PURPOSE: flag. "The result names how to restore it" is printed but doesn't work, and is missing on failure.
- ARCH-MOCK: pass (real-git fixtures plus the IO seam).
- ARCH-CONSTRAINTS: minor flag (budget not measured).
- ARCH-SECURE: minor flag (Go quoting in a shell command; trusting `saved_at`).
- ARCH-ORDER: pass with a minor flag (identifying entries by time window).
- ARCH-FUNERAL: pass. The collector is the writer and the bound is stated.

**7. Plan revisions**
- Task 3.4: add a Revision saying needs-zero now follows `SlotOutcome` blocking, not raw hand-off stops. Record that the budget measurement and the `TestRowAdviceNamesOnlyReachableActions` sweep item were not delivered, or deliver them. **This is the 2nd finding in family `plan-tracks-implementation`.** The rule: a ticked item that shipped differently gets a Revision line in the same commit that ticks it.
- Task 3.6: untick the skill item until the skill is updated.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan prose for Tasks 1.4/2.5/3.1 unchanged in this window; still Minor, carried to close as Revision (u) states.
findings:
  - id: new
    severity: Important
    family: advice-names-reachable-action
    title: |
      Printed restore command does not restore: mv into the recreated checkout nests the tree; missing on failure; Go-quoted not shell-quoted
    detail: |
      2nd in family. slotsave.go:134 builds mv "<entry>/tree" "<path>" and provision.go:110 prints it after path was re-cloned/re-added, so mv yields path/tree (verified). The blocking path returns before SetAside is reported, and %q is Go quoting (ShellQuote exists). Rule: every printed command is executed in a test from the state that prints it. Make Restore move the recreated checkout aside then move the tree back via ShellQuote, run r.SetAside[0].Restore with sh -c in TestAcceptanceDirtySlot, and carry the set-aside list on failure too.
  - id: new
    severity: Important
    family: error-surface-preserved
    title: |
      recoverReason has no case for slot-needs-:0 or reconcilable, so those rows read "no rule matched" and drop the reconciler's cause
    detail: |
      4th in family. Task 3.4 promised needs-zero with ReconcileAdvice. The DirectoryMissing text "repairing it is pair#387" is also stale. Rule: a SlotReport consumer carries the report's known cause (stopFailure/ReconcileAdvice) into its text, and recoverReason is total over AllRecoverClasses, enforced by a test sweeping every class and rejecting the default text.
  - id: new
    severity: Important
    family: report-agrees-with-caller-outcome
    title: |
      RecoverSlotClass holds a usable slot on any handoff stop, while SlotOutcome treats it as a warning
    detail: |
      recoverplan.go:518 maps StopHandoff to needs-zero, but SlotOutcome blocks only when OutcomeSeverity is blocking. A resting branch checked out elsewhere with a working host loses its resume step (fixture "workspace needs :0" asserts this). It is the same decision made twice (ARCH-DRY, lessons BR-12). Derive needs-zero from SlotOutcome returning blocking and turn non-blocking handoff stops into a note. The Retried branch is unreachable in the report.
  - id: new
    severity: Important
    family: readme-tracks-user-surface
    title: |
      Couch skill not updated despite Task 3.6 tick: still says "Nothing is ever deleted" and omits the new classes, hold and notes
    detail: |
      2nd in family. cmd/internal/couchcmd/skills/couch/SKILL.md:125-127 contradicts retention collection and gives the recovering agent no procedure for slot-needs-:0/workspace-handoff, the reconcile step, or workspace-held/unknown. Rule: a diff adding a class/hold/note or changing a lifecycle statement sweeps every doc listing that vocabulary (README, atlas, skill) in the same window.
  - id: new
    severity: Minor
    family: plan-tracks-implementation
    title: |
      Task 3.4 ticked items without delivery or Revision: 150 ms/slot budget with 10 slots, row-advice sweep including reconcile
    detail: |
      2nd in family. Rule: a ticked item that shipped differently gets a Revision line in the same commit that ticks it. The unmeasured budget is also an ARCH-CONSTRAINTS gap.
  - id: new
    severity: Minor
    family: decision-preserves-notes
    title: |
      Idle plus reconcilable replaces the whole decision, dropping notes computed earlier such as claims-stale
    detail: |
      recoverplan.go:661 rebuilds recoverDecision; it should keep d.Notes.
  - id: new
    severity: Minor
    family: identity-not-time-window
    title: |
      SavedWorkSince picks this run's entries by wall-clock window, not by the step's own entry
    detail: |
      ARCH-ORDER: have setAside return its entry path in Executed. Separately, a hand-edited future saved_at is never collected (ARCH-SECURE).
```

---

## Re-review — 2026-10-05T21:36:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | fd7da05196f34ccac2b6c58653fed802a268d565..a7acac1828f8a047987f6b261ebb9a1f3c33cf3e |
| command | sdlc milestone-close --issue 387 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-05T21:36:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All four Important findings from round 5 (BR-14 to BR-17) are fixed, and each fix has a test that would fail without it. I ran the restore path and the report/caller agreement code myself. The minor findings BR-18 and BR-20's identity half are also done. Nothing blocks SHIP. Two small fixes have no test: the BR-19 notes fix and the BR-20 future-`saved_at` guard. When I reverted each one in a scratch copy, the suite stayed green. The PTY test failures in my full `couchcore` run come from the sandbox ("operation not permitted"), not from this change. The targeted run (`Recover|SavedWork|SetAside|Acceptance|Reconcile|Slot`) passes.

1. **Strengths**
   - **The printed restore command is now actually run in a test.** `SavedWorkRestoreCommand` (slotsave.go:157) builds a shell-quoted command. `TestAcceptanceDirtySlot` runs it with `sh -c` from the exact state that printed it, and checks two things: the user's work is back in place, and the recreated clone is kept in the entry.
   - **A failed run still names what it set aside.** `SlotReconcileError.SetAside` carries the list, and `TestAFailedRunStillNamesWhatItSetAside` covers it.
   - **The report and the callers make one decision, not two.** `RecoverSlotClass` (recoverplan.go:513) decides needs-zero through `SlotOutcome`, the same function the callers use. `TestRecoverSlotClassAgreesWithTheCallers` sweeps the planner's whole state domain against every agent state, so the report can't hold a slot the callers would open.
   - **Set-aside entries are reported by identity.** The `setAsideDone` hook replaces the old time-window lookup (`SavedWorkSince`), which is gone from the tree.
   - **The budget is measured.** `TestAcceptanceReportBudgetTenSlots` times observe+plan over ten real slots.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-19 has no test.** I reverted `Notes: d.Notes` at recoverplan.go:667 and the Recover tests still passed. The metamorphic test needs to check that the idle→reconcilable change keeps the notes the row had before.
   - **BR-20's future-`saved_at` guard has no test.** Removing `!m.SavedAt.After(now)` at slotsave.go:202 leaves the saved-work tests green.
   - **One gap in set-aside reporting (new).** In `setAside`, the tree is renamed first and the "complete" manifest is written afterwards. If that manifest write fails, `setAsideDone` never runs, so the tree has moved but the run doesn't report it. The fix is to report the entry as soon as the rename succeeds.
   - `recoverWorkspaceAdvice` repeats the `SlotOutcome` call and the `strings.Cut` parse of the address that `RecoverSlotClass` already does. A small shared helper would remove the copy.

5. **Test coverage notes**
   - I confirmed every fixture class has reason text other than "no rule matched", and that reconcilable, slot-needs-zero and directory-missing each have a fixture.
   - The two untested guards above are the only fixes that pass my scratch mutation check.

6. **Architectural notes**
   - **ARCH-DRY:** pass. The decision goes through `SlotOutcome`; the only leftover is the duplicated call noted above.
   - **ARCH-PURE:** pass. `RecoverSlotClass` and `classifyRecover` stay pure.
   - **ARCH-PURPOSE:** pass. Every class has reason text.
   - **ARCH-MOCK:** pass. The acceptance tests use real git.
   - **ARCH-CONSTRAINTS:** pass. The 150 ms per-slot budget is measured.
   - **ARCH-SECURE:** pass on structure (the restore command uses `ShellQuote`), but the future-`saved_at` guard is untested.
   - **ARCH-ORDER:** minor flag for the rename-then-manifest-write gap above.
   - **ARCH-FUNERAL:** pass. `recreated/` lives inside the saved-work entry and is collected with it under the one-year retention.

7. **Plan revisions:** none needed. Task 3.4's step text still says `slot-needs-:0`, but Revision (z) records the rename, which is how the convention says to handle it.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Still carried to close by the plan's own note at line 1389; Minor, does not block.
  - id: BR-14
    disposition: addressed
    note: |
      SavedWorkRestoreCommand is ShellQuoted and moves the path aside first; executed via sh -c in TestAcceptanceDirtySlot; failure path covered by TestAFailedRunStillNamesWhatItSetAside.
  - id: BR-15
    disposition: addressed
    note: |
      Reconcilable and slot-needs-zero reason cases added, stale text fixed; coverage test rejects the default text per fixture class.
  - id: BR-16
    disposition: addressed
    note: |
      RecoverSlotClass derives needs-zero via SlotOutcome; non-blocking handoff becomes the workspace-degraded note; TestRecoverSlotClassAgreesWithTheCallers sweeps the planner domain x agents.
  - id: BR-17
    disposition: addressed
    note: |
      SKILL.md step 10 documents slot-needs-zero, workspace-handoff, reconcile and the three notes; deletion statement corrected; README and atlas also list workspace-degraded.
  - id: BR-18
    disposition: addressed
    note: |
      TestAcceptanceReportBudgetTenSlots measures the budget; Revision (z) records that the reconcile row-advice item does not apply.
  - id: BR-19
    disposition: not-addressed
    note: |
      Fix present at recoverplan.go:667, but reverting it leaves the Recover tests green; add a notes-preserved check for idle to reconcilable.
  - id: BR-20
    disposition: not-addressed
    note: |
      Identity reporting addressed (setAsideDone; SavedWorkSince removed); the future saved_at guard has no test, as removing it keeps the suite green.
findings:
  - id: new
    severity: Minor
    family: error-surface-preserved
    title: |
      A set-aside whose final manifest write fails after the rename is not reported
    detail: |
      This is the 5th finding in family error-surface-preserved. Rule: report an effect when it happens, not when its bookkeeping completes. setAsideDone fires only after the complete-manifest write, so a tree already moved by rename goes unreported if that write fails. Fix: call setAsideDone right after the rename succeeds.
```
