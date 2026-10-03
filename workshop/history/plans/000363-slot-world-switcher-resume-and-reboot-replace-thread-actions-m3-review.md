# Boundary Review — pair#363 (milestone M3)

| field | value |
|-------|-------|
| issue | 363 — Slot-world switcher: resume and reboot replace thread actions |
| repo | pair |
| issue file | workshop/issues/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 0e0e1f00425f325b9071a06e3654e2457c9333c3..13b20bd4d132daa8b3df2bc6f6801d1a9ad1e348 |
| command | sdlc milestone-close --issue 363 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-03T02:00:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M3 delivers what the plan says. `SelectResumableRoot` and `ScopeHoldsUsableThread` now match on the repository scope, through one shared filter, `primaryOfScope`. Startup in a subdirectory resumes the root `:0`. The start form refuses a second primary, and its fresh-start advice depends on whether the held row can be rebooted. The non-Git refusal is pinned to `Resolve`'s refusal, so the planned mutation (reverting `Resolve`'s error return) can turn the test red. I checked the two paths most likely to break under the wider guard. **Slot creation:** `launcher.ResolveRepoScope` keys a scope by its worktree root, and a slot has its own worktree root, so the guard never blocks add-slot. **Reboot:** its main path goes through `ReplaceThreadExpected`, which never reaches the guard. The M3 tests pass at head:
- `couchcore`: `TestSelectResumableRoot`, `TestScopeHolds`, `TestNarrowedStartup`, `TestStartInteractiveInSubdirectory`, `TestSpawnInSubdirectory`, `TestPrimaryOccupied`, `TestStartInANonGit`, `TestASecondThread`, `TestCoTenants`, `TestOccupancy`, `Reboot*`
- `couchtty`: `TestRowAdvice`, `TestPathMissing`, `TestRebootConfirmation`, `TestUnknownRow`, `Menu*`

In the full package runs, only the PTY-child and `/tmp` mkdir tests failed, all with "operation not permitted". That is the known sandbox limit, not this diff. Nothing blocks SHIP.

1. **Strengths**
   - `startup.go:93` `primaryOfScope` is the one filter both predicates read. The selector and the guard cannot drift apart (ARCH-DRY).
   - `couch.go:432` `primaryFreshStep` derives its advice from `RebootableState`. `TestPrimaryOccupiedFreshStepNamesOnlyWhatTheRowOffers` checks it over every state × reason the guard can hold. That fixes the rule, not just one instance (family `refusal-names-unoffered-action`).
   - `menu_actions.go` `menuRowAdviceOf` is now the single source for every next step a row shows. `TestRowAdviceNamesOnlyReachableActions` builds its vocabulary from the action table and checks the printed surfaces too, so advice that bypasses the source still gets caught.
   - `TestStartInANonGitDirectoryRefuses` asserts that the refusal comes from `Resolve` and that a `rev-parse --show-toplevel` call happened. Without that, a refusal from `resolveRepoIdentity` would have hidden the planned mutation.
   - `TestCoTenantsAreAddressableByActorID` creates its legacy second primary past the guard, by calling `spawnResolved` with no rows. Store compatibility stays covered without weakening the rule.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `menu_actions.go` (the `DirectoryMissing && Kind == menuRowSlot` case): `RebootCost.Text` is set, but a `:1+` row with no directory offers no actions, so that reboot confirmation can never appear. The text is dead. Drop it, or comment that it is unreachable.
   - `reboot.go:150` (rolled-back branch): with a store from before the rule holding two primaries in one scope, rebooting the one that rolled back now gets the "one primary slot" refusal from the other. That is consistent with the rule, and nothing is stopped. It is untested.

5. **Test coverage:** Strong. Every new test exercises the production predicates through `StartInteractive`, `SpawnPrepared` and `PrepareStart`, using the fakes. Task 3.2's red step was a mutation, and it is honestly recorded in the plan's Revisions.

6. **Architecture**
   - **ARCH-DRY:** pass. One shared filter (`primaryOfScope`) and one source for row advice.
   - **ARCH-PURE:** pass. The predicates and `menuRowAdviceOf` are pure; IO stays in `spawnResolved`.
   - **ARCH-PURPOSE:** pass. Every creation entry goes through the one guard site. The start form, startup and the rolled-back reboot all read the scope predicate.
   - **ARCH-MOCK:** pass. Git goes through `FakeGit`, processes through the fake runner.
   - **ARCH-CONSTRAINTS:** pass. Widening `startupAsks` adds cold proofs only for this scope's other records, and slots live in their own scopes, so the doc comment's "normally zero or one" holds.
   - **ARCH-SECURE:** N/A. No new untrusted input; unreadable records still block the scope.
   - **ARCH-ORDER:** pass. No new state is carried between events.
   - **ARCH-FUNERAL:** pass. Nothing new is persisted; refusals have no side effects, and the tests assert the record and op counts are unchanged.

7. **Plan revisions:** none needed. The 2026-10-03 M3 Revision already records the deviations. Task 3.5's remaining unticked items (full Chunk 4 verification, the live run from a subdirectory, the close) are boundary work, not missing code.

```findings
findings:
  - id: new
    severity: Minor
    family: dead-advice-text
    title: |
      The slot DirectoryMissing RebootCost text cannot appear, because that row offers no reboot
    detail: |
      menuRowActions returns nil for a :1+ row whose directory is missing, so no reboot confirmation ever shows it. Remove the field value or comment why it is kept.
  - id: new
    severity: Minor
    family: legacy-cotenant-guard-interaction
    title: |
      A rolled-back :0 reboot refuses when a pre-363 co-tenant primary exists in the scope
    detail: |
      reboot.go rolled-back branch calls spawnResolved, and the widened guard sees the other primary. This follows the rule and stops nothing, but it is untested and unmentioned.
```
