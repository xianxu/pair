# Boundary Review — pair#402 (whole-issue close)

| field | value |
|-------|-------|
| issue | 402 — Switcher: add slot is offered only while the repository's :0 is live |
| repo | pair |
| issue file | workshop/issues/000402-switcher-add-slot-is-offered-only-while-the-repository-s-0-is-live.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1fbe166cbf5af891b99ab6a9f98fb85877e70947..ad3abfcd762ee65376aa0fc1ad037d86e63a1746 |
| command | sdlc close --issue 402 |
| reviewer | claude |
| timestamp | 2026-10-06T22:55:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The production change is correct and small. `menuRowActions` (`cmd/internal/couchtty/menu_actions.go:123-130`) now appends `add-slot` to the resumable and unusable phases when `AddSlotOffered && !Actor.DirectoryMissing`. `DirectoryMissing` is exactly "unusable with reason path-missing" (`couchcore/actor_actions.go:30`), so a `:0` whose checkout is gone still doesn't offer add slot, and unknown and busy rows still offer nothing. The add-slot execution path (`menu.go:815-823`) reads only `menuAddSlotPath(thread)`, which derives from the row's paths. It never reads `:0`'s actor, so Spec bullet 2 holds. The Spec table sweep (`expectedRowActions`) was extended to the parked, detached and unusable `:0` shapes. The add-slot integration test now drives the create commit from a parked `:0`. Two things block a clean SHIP. Several "live `:0`" claims next to the change are now false, including the README how-to at `README.md:657`. And Spec bullet 3 ("OnPrimary add-slot advice … reachable when `:0` is parked too") is not exercised: the advice sweep still checks OnPrimary reach only against the live `:0`.

**1. Strengths**
- `menu_actions.go:127` gates on `!f.Actor.DirectoryMissing`, so the new offer doesn't contradict the `:0` "checkout missing" advice (`RebootCheckoutMissing`).
- The Spec table is restated independently in `menu_actions_test.go:115-130` rather than calling production. It is swept over the derived shape domain, so every unusable reason and continuation phase is covered for both `:0` and `:1+`.
- `menu_add_slot_test.go:37-60` runs the whole preview → Enter → `start create` path for live and parked `:0`. The only commit effect is `start`, with no `resume`, which pins the "without resuming `:0`" clause.
- The README table and atlas were updated in the same range.

**2. Critical:** none.

**3. Important**
- **Stale "live `:0`" doc and comment claims.** The `doc-claims-track-offer-table` family; every instance in the tree:
  - `README.md:657`: "open the repository's live `:0` row's action menu and choose **add slot**". A parked `:0` now works too.
  - `menu_actions.go:84-87`, the `menuRowActions` doc comment: "rows that are not live get the two actor operations, resume and reboot". A not-live `:0` now also gets add slot. This comment sits right above the edited code.
  - `menu_actions_test.go:466`: "OnPrimary: add slot on the live :0 recreating a gone :1+".

  `README.md:869` (the live `:0` table row) and `atlas/couch.md:167` (about alias) are still correct.
- **Spec bullet 3 is untested.** `TestRowAdviceNamesOnlyReachableActions` (`menu_actions_test.go:471-498`) sets `reach = livePrimary` for every OnPrimary step. The claim that `RebootDirectoryMissing` on a `:1+` row is reachable when `:0` is parked therefore has no coverage. Fix: also collect the parked and detached `:0` offers (or every `:0` shape whose directory exists and whose state is known), and check OnPrimary reach against each of them.

**4. Minor**
- `menu_actions_test.go:135-151`: `expectedUnusableActions` wraps its body in a leftover bare `{ … }` block from the extraction. Remove it.
- `atlas/couch.md:750-751`: the inserted line runs far past the file's wrap width.
- `## Log` is empty. Spec bullet 2's "first verify the execution path" finding, and the pending live smoke test (Done-when clause 2), aren't recorded anywhere.
- ARCH-DRY (minor): the `AddSlotOffered` append appears in two branches (`menu_actions.go:110` and `:127`). A single post-switch append guarded by phase ∈ {live, live-failed, resumable, unusable} would keep it in one place. This is acceptable as it stands, because the ordering differs from the alias append.

**5. Test coverage notes**
- Done-when clause 1 (a parked or unusable `:0` offers add slot, one test per phase): the table sweep covers parked, detached and every unusable reason.
- Done-when clause 2: covered for parked. The operator's live smoke test still has to happen before close.

**6. Architectural notes**
- ARCH-DRY: pass (see the minor note above).
- ARCH-PURE: pass. The change stays in the pure `menuRowActions` table, and the tests need no IO.
- ARCH-PURPOSE: pass on the main behavior. The advice-reachability part of the Spec is the open coverage gap above.

**7. Plan revision recommendations**
- Add a `## Log` entry recording that the add-slot execution path (`menu.go:815`) reads only `menuAddSlotPath`, not the `:0` actor (Spec bullet 2 verified), and that the live smoke test is pending.

```findings
findings:
  - id: new
    severity: Important
    family: doc-claims-track-offer-table
    title: |
      "live :0" claims about add slot are now false (README how-to, menuRowActions doc, sweep comment)
    detail: |
      README.md:657 says to open the repository's live :0 row to add a slot. The menuRowActions doc comment (menu_actions.go:84-87) says not-live rows get only resume and reboot. menu_actions_test.go:466 says "add slot on the live :0". A parked or unusable :0 (directory present) now offers add slot. README.md:869 and atlas/couch.md:167 (alias) are still correct.
  - id: new
    severity: Important
    family: spec-clause-both-states-exercised
    title: |
      OnPrimary add-slot reachability with a parked :0 (Spec bullet 3) is not checked by the advice sweep
    detail: |
      TestRowAdviceNamesOnlyReachableActions sets reach = livePrimary for every OnPrimary step (menu_actions_test.go:471-498). Also check reach against the parked and detached :0 offers (or every :0 shape whose directory exists and whose state is known), so the Spec's "reachable when :0 is parked too" fails if it regresses.
  - id: new
    severity: Minor
    family: extraction-leftovers
    title: |
      expectedUnusableActions keeps a leftover bare { } block from the extraction
    detail: |
      menu_actions_test.go:135-151; atlas/couch.md:750-751 also runs past the file's wrap width.
  - id: new
    severity: Minor
    family: issue-log-records-verification
    title: |
      The issue Log does not record the Spec bullet 2 execution-path check or the pending live smoke test
    detail: |
      menu.go:815 reads only menuAddSlotPath(thread), not the :0 actor. Write that in the Log, and note that the operator's live smoke test (Done-when clause 2) is still pending.
```

---

## Re-review — 2026-10-06T22:57:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 402 — Switcher: add slot is offered only while the repository's :0 is live |
| repo | pair |
| issue file | workshop/issues/000402-switcher-add-slot-is-offered-only-while-the-repository-s-0-is-live.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1fbe166cbf5af891b99ab6a9f98fb85877e70947..e5c49b7a5f751e1f69fc82c7ffa17e7193883d89 |
| command | sdlc close --issue 402 |
| reviewer | claude |
| timestamp | 2026-10-06T22:57:55-07:00 |
| verdict | SHIP |

## Review

Reviewing the #402 close window. The targeted tests pass. The one couchtty failure is the sandboxed pty soak test, a known sandbox limit and not a regression. Writing the verdict now.

```verdict
verdict: SHIP
confidence: high
```

All four prior findings are addressed. The fix in `menuRowActions` (cmd/internal/couchtty/menu_actions.go:124-131) adds add slot to a resumable or unusable `:0` whenever `AddSlotOffered` holds and the directory is present. It runs only in the `:0` branch, because `AddSlotOffered` is set only for primary rows (`menu_actions.go:54`). The Spec table test now expects add slot on parked, detached and unusable `:0` rows. The advice sweep now requires an `OnPrimary` step to be offered by `:0` when live, parked and detached. `TestAddSlotPrefillsExactRepositoryAndUsesCreatePreview` runs the create commit from both a live and a parked `:0`. The Log records the Spec bullet 2 path check and the operator's waiver of the live smoke test. One Minor doc gap remains: README's how-to lists `:0` states and leaves out "unusable".

1. **Strengths**
   - `menu_actions.go:128`: the `DirectoryMissing` exclusion keeps add slot off a `:0` whose checkout is gone, and the reason is in the comment.
   - `menu_actions_test.go:471-520`: `reachableOnPrimary` takes the intersection over all three `:0` states, so an `OnPrimary` step that only a live `:0` can take fails the sweep.
   - `menu_add_slot_test.go:36-41`: the live and parked loop pins Done-when clause 2. Both states reach the same create commit, and nothing resumes `:0`.
   - Moving the unusable cases into `expectedUnusableActions` keeps the add-slot expectation in one place.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - README.md:657-658 says "`:0` may be live, parked or detached". An unusable `:0` with its directory present also offers add slot. Line 658 is also past the wrap width. This is the second finding in family `doc-claims-track-offer-table`. The rule for every instance: a doc describing when `:0` offers add slot states the condition ("its checkout is present") instead of listing states. README:884-885 and atlas/couch.md:750 already follow that rule. README:657 is the only place in this window that still lists states, so fixing it to the rule closes the family.

5. **Test coverage:** both phases in Done-when clause 1 are covered by `TestRowActionTableMatchesTheSpec` through `everyMenuRowShape`. Clause 2 is covered for live and parked. The targeted tests pass: `TestRowAction*`, `TestRowAdvice*` and `TestAddSlot*`. The full couchtty package fails only on the pty soak test, which the sandbox blocks ("operation not permitted").

6. **Architecture**
   - **ARCH-DRY: pass.** The rule is one condition in `menuRowActions`, and the test expectation is derived from `expectedUnusableActions`.
   - **ARCH-PURE: pass.** `menuRowActions` stays a pure function of `menuRowFacts`, and the tests use no IO.
   - **ARCH-PURPOSE: pass.** Every consumer of the old live-only claim was checked: the action table, the advice sweep, the add-slot flow test, README and atlas. Each now agrees with the fix, apart from the Minor wording above.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README:657, menuRowActions doc (menu_actions.go:84-88), sweep comment (test:464-466) and OnPrimary field comment no longer claim live-only; remaining enumeration gap raised as a Minor under the same family.
  - id: BR-2
    disposition: addressed
    note: |
      reachableOnPrimary requires the action in live, parked and detached :0 offers (menu_actions_test.go:471-520); len(primaryOffers)!=3 guards the derivation.
  - id: BR-3
    disposition: addressed
    note: |
      expectedUnusableActions is a clean function with no bare block; atlas/couch.md:750-752 rewrapped.
  - id: BR-4
    disposition: addressed
    note: |
      The Log records the Spec bullet 2 menuAddSlotPath-only check and the operator's 2026-10-06 waiver of the live smoke test.
findings:
  - id: new
    severity: Minor
    family: doc-claims-track-offer-table
    title: |
      README:657-658 lists live, parked or detached and omits an unusable :0 whose directory is present
    detail: |
      2nd in family. Rule: state the add-slot condition (the :0 checkout is present), not a list of states. README:884-885 and atlas/couch.md:750 already do; README:657 is the only listing instance in this window. Line 658 also exceeds the wrap width.
```
