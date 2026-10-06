# Boundary Review — pair#387 (whole-issue close)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | whole-issue close |
| milestone | — |
| window | 57ae1bede1146c1cd360d265e17d78611687e5ca..c3394e404ca9d3337d01a4e243243caeee07c1b5 |
| command | sdlc close --issue 387 |
| reviewer | claude |
| timestamp | 2026-10-05T21:40:37-07:00 |
| verdict | unknown |

## Review

The targeted test run passed: `go test ./cmd/internal/couchcore/ -run 'Slot|SetAside|Show|Reconcile|Saved'` printed `ok` after 133s.

The verdict stays FIX-THEN-SHIP. The only open item is BR-21 (Minor): the code fix is correct, but no test fails without it. A test would need to make the manifest write fail after the rename succeeds, then check that the set-aside was still reported. I only ran this subset of the suite, so run the full `make -k test` before `sdlc close`.

---

## Re-review — 2026-10-06T10:55:47-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | whole-issue close |
| milestone | — |
| window | 57ae1bede1146c1cd360d265e17d78611687e5ca..28486df7b4d639e5fcbfd70b4a2391220347951a |
| command | sdlc close --issue 387 |
| reviewer | claude |
| timestamp | 2026-10-06T10:55:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

This round covers the four commits since the last scored round: c3394e40, d439ada4, 14ea591d and 28486df7. I also re-checked every open finding against the code at HEAD. BR-21 is fixed: `setAsideDone` now fires right after the rename (`slotsave.go:146-152`). `TestASetAsideIsReportedWhenTheMoveHappens` covers it with a store that fails only the "complete" manifest write; the run must report the entry and leave a pending manifest naming the restore. BR-12 and BR-9 are fixed as rules, not just at the sites first reported. The weave-identity commit (28486df7) is small and correct, has tests, and updates the atlas. Its tests and the earlier findings' tests pass when run as a targeted set (`go test -run …` in couchcore). Four Minor findings stay open: BR-1, BR-8, BR-19 and BR-20. In each, the code is right or the problem is plan prose, but there is no test that would fail if the fix were removed. None of them blocks.

1. **Strengths**
   - `setAsideHoldError` / `SetAsideHold` (`slotsave.go:46-56`) is one pure reading used by both `PlanSlot` and the converge step. `ClassifyConvergeError` (`slotfailure.go:50-58`) maps every typed step error to a class, and `errCheckoutRecovered` is retryable. This closes the stop-reason-precision family as a rule.
   - `--show` slot errors are handled in one place (`operationdispatch.go:166-203`). When no thread matched, the slot error fails the command. When a thread matched, it becomes `SlotReport.PlanError`. A slot error is never reduced to "not a slot". `TestShowSurfacesSlotResolutionErrors` covers Discover with no thread, Discover with a thread, and the path probe.
   - The weave identity is an optional `ProgramIdentifier` capability (`provision_io.go:32-54`). It resolves the program the same way `exec` does: `LookPath`, then `EvalSymlinks`, size and mtime. A test against the real OS checks that the identity changes when the file is replaced, and the fixture test checks that a remembered failure re-runs after an upgrade.
   - BR-21's test injects the failure behind the real storage seam, a wrapped `ProvisionStorage`, rather than mocking the step.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-1: Tasks 1.4, 2.5 and 3.1 in the plan still list test cases in prose (still open).
   - BR-8: no test drives `show` with `repo:N` for a missing slot and checks for the "does not exist (existing: …)" text. This is the third error source in BR-9's rule (`WorkspaceReferencePath`), and the only one without a test (still open).
   - BR-19: the fix is in place (`recoverplan.go:667`). But the reconcilable branch of `TestRecoverReconcileReadingIsMetamorphic` checks only the class and the steps, not that `d.Notes` is kept, so removing the fix stays green (still open).
   - BR-20: `collectSavedWork` ignores a `saved_at` dated in the future (`slotsave.go:199-201`). `TestSavedWorkIsCollectedPastRetention` has no future-dated entry, so removing that check stays green (still open).
   - New: `setupInputsDigest` reads the weave identity through a type assertion (`slotmemo.go:33`). If a decorating `ProvisionIO` is added later, the assertion will quietly drop weave from the digest. There is no wrapper today, so I'm noting it only, not raising it.

5. **Test coverage**
   - The four open Minors share one gap: the code is right, but no test fails without it. Each needs a one-assertion test:
     - BR-19: check that the notes survive.
     - BR-20: add a future-dated entry and check it is kept.
     - BR-8: drive `show` with `repo:N` for a missing slot.
   - The weave tests cover the digest input both through the fixture and on the real OS.

6. **Architecture**
   - **ARCH-DRY:** pass. There is one source for holds, and the restore command is built in one place.
   - **ARCH-PURE:** pass. The digest and the classification are pure, and IO goes through `ProvisionIO`.
   - **ARCH-PURPOSE:** pass. The class rules from BR-9 and BR-12 are applied to every site.
   - **ARCH-MOCK:** pass. `ProvisionFixture` implements `ProgramIdentity` behind the same seam, and the real OS path is tested.
   - **ARCH-CONSTRAINTS:** pass. The weave identity costs one `LookPath` and one `Stat` per digest.
   - **ARCH-SECURE:** pass, apart from BR-20's missing test. A future-dated `saved_at` is not trusted, and the manifest is parsed into a typed struct.
   - **ARCH-ORDER:** pass. The set-aside is reported as the effect happens (BR-21), and the run's entries are identified by the step, not by a time window.
   - **ARCH-FUNERAL:** pass. The new commits create nothing durable, and saved work is collected by its owner after one year, with a bound of 16 entries per slot.

7. **Plan revisions**
   - Revision (aa) correctly records the weave-identity change.
   - Revision (z) says "Idle-to-reconcilable keeps earlier notes", but no test enforces it (BR-19). It should cite the regression test once one is added.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan Tasks 1.4/2.5/3.1 still list test cases in prose; Minor, non-blocking.
  - id: BR-8
    disposition: not-addressed
    note: |
      Code returns the WorkspaceReferencePath error, but TestShowSurfacesSlotResolutionErrors has no subtest for show repo:N of a missing slot asserting "does not exist (existing: ...)".
  - id: BR-9
    disposition: addressed
    note: |
      operationdispatch.go:166-203 surfaces slotErr either as the command error or as SlotReport.PlanError; slotreport.go:73-76 returns the probe error; TestShowSurfacesSlotResolutionErrors covers discover with no thread, discover with a thread, and the path probe.
  - id: BR-12
    disposition: addressed
    note: |
      setAsideHoldError derives the error from SetAsideHold (slotsave.go:46-56); ClassifyConvergeError maps agent-appeared, agent-unobserved and saved-work-full to holds and checkout-recovered to retryable; TestSetAsideHoldsMatchThePlan sweeps every agent state.
  - id: BR-19
    disposition: not-addressed
    note: |
      Fix present at recoverplan.go:667, but the reconcilable branch of TestRecoverReconcileReadingIsMetamorphic checks only class and steps, never Notes, so reverting it stays green.
  - id: BR-20
    disposition: not-addressed
    note: |
      Identity reporting is done; the future-saved_at guard (slotsave.go:199-201) still has no fixture in TestSavedWorkIsCollectedPastRetention.
  - id: BR-21
    disposition: addressed
    note: |
      setAsideDone fires right after the rename (c3394e40); TestASetAsideIsReportedWhenTheMoveHappens fails the complete-manifest write and asserts the entry is reported with a pending restore manifest.
```
