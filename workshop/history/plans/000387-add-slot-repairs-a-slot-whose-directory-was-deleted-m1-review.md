# Boundary Review — pair#387 (milestone M1)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 57ae1bede1146c1cd360d265e17d78611687e5ca..84467da9dcac9487527739027c4b46ab6acb8b46 |
| command | sdlc milestone-close --issue 387 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-05T18:32:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers what the plan says it would. The resource table, the `SlotLayout` path authority with its AST coverage audit, the `layergraph` import, the real-git observation, the pure planner over a generated perturbation domain, and the `--show` slot section are all in the diff, and the slot test subset passes (`go test ./cmd/internal/couchcore -run 'Slot|Observe|PlanSlot|DeclaredDeps|ParseSlotPath|Show|Provision|Ensure'` → ok, 132 s). The wider package failures I saw are environmental or pre-existing. The pty and helper tests fail with "operation not permitted" (sandbox). `artifactpath`'s `TestProductionArtifactReferencesAreExactlyClassified` fails at the base commit `57ae1bed` as well, on files this diff does not touch.

Two things should be fixed before M2 builds on them:
- **`--show` guesses the repository's git directory instead of asking git.** Observation compares that guess against git's resolved paths, so a primary checkout reached through a symlink, or one whose `.git` is a file, reads as broken on every slot. M2 will drive repairs from the same layout.
- **A working dependency clone can be classified as unreadable.** Unreadable dependencies are what get set aside, so this undercuts the plan's "broken only on positive evidence" rule.

The README also needs a line for the new `--show` behavior.

**1. Strengths**
- `slotlayout.go` is now the one place slot paths are spelled. The token and path-root audit in `slotresource_coverage_test.go` enforces that, so the table really is the single source rather than documentation (ARCH-DRY, ARCH-PURPOSE).
- `slotobserve.go` treats a failed probe as unknown, never absent: a failed `worktree list` makes the branch and registration unknown, and a failed dependency `rev-parse` makes that dependency unknown. `TestObserveFailedProbesAreUnknownNeverAbsent` pins both.
- `TestPlanSlotDomain` builds its domain from the table itself (`BrokenSubs`, `SubStates`, `AllEvidenceAgents`), over single and paired perturbations. Its invariants I3 and I4 are written independently of the planner's rules.
- `PlanSlot` sorts its steps by `SlotResourceOrder()`, so the step order holds by construction.
- Re-adding a stale registration with `--force` on the branch it records keeps a deleted slot on its issue branch rather than silently moving it to `main-slotN`. That is a good catch from implementation, recorded in Revision (l).

**2. Critical findings**
None.

**3. Important findings**
- **`slotreport.go:63` — `--show` builds the slot layout from a guessed git directory.** `slotOfShowReference` uses `conventionalSlot(primary, n)`, which assumes the common git directory is `<primary>/.git`. It never uses the real one that `Discover` / `sdlc` already return as `RepoIdentity`. The raw path argument is also not resolved through symlinks.
  - Git reports resolved paths: I checked that `worktree list`, `--git-common-dir` and `--absolute-git-dir` all resolve symlinks. So when the primary is reached through a symlink, or its `.git` is a file, three readings go wrong. The registration reads absent, the branch reads "checked out elsewhere" (its own host path differs as a string), and the host reads mismatched (`slotobserve.go:385`).
  - The fix: resolve the reference with `EvalSymlinks`, as `slotcontext.go:23` does, and take the slot identity from `Discover`, which carries git's own common directory. Add a fixture whose primary is reached through a symlink. (ARCH-DRY: one authority for the repository's identity.)
- **`slotobserve.go:444` — a valid repository can be marked unreadable.** The check `filepath.Dir(out) != d.Path` marks a dependency broken/unreadable, the one reading that leads to a set-aside, even though git just read it successfully. That happens for a clone whose `.git` is a file pointing elsewhere (a linked worktree, or `--separate-git-dir`). R2 limits broken to "no `.git`, or git says not a git repository".
  - The fix: compare `rev-parse --show-toplevel` with `d.Path` instead, and add a test with a gitfile-backed dependency.
- **README update appears missing for the new `--show` behavior.** `README.md:383` and `:528` still describe `--show` as showing one thread by tag or path. It now also accepts `repo:N`, prints a slot's resources and plan, and works when the slot has no thread.

**4. Minor findings**
- `slotobserve.go:405-411`: when a dependency is not a layer, every other declared dependency disappears from the observation. That includes siblings of the host, not only what lies beyond the bad one. If the non-layer is outside the env, in-env dependencies that are missing go unseen, and the setup marker can read present ("nothing to do"). Plan step 1.3 said "the others are still reported"; Revision (j) should say they are not.
- `slotplan.go:340`: a held weave lock makes a converged slot report a retryable stop, so `--show` of a healthy slot says "stops at setup" while an agent is compiling. Decide whether that is intended before M2 maps it to a severity.
- `slotplan.go:255`: an unknown agent is reported as the `live-agent` hold reason. Consider a distinct reason for the unknown case.
- `slotreport.go:51`: an error from `WorkspaceReferencePath` is dropped, so `couch --show repo:9` (slot does not exist) shows the generic "thread not found" instead of the better "slot does not exist (existing: …)".
- The plan's Task 1.7 names `atlas/couch.md`, but the section landed in `atlas/workspace-provisioning.md`. The content is fine; record where it went.

**5. Test coverage notes**
- The `--show` render tests (`slotreport_render_test.go`) use made-up observations. Only `TestShowReportsASlotWithoutAThread` exercises real git, and it uses an already-resolved fixture path, so neither Important path issue above is covered.
- Task 1.6's named fixtures (the `tools:1`-shaped one printing `plan: compile`, and the unknown-resource line) are covered at the planner and renderer level, not end to end. That is acceptable for M1.
- The I2 invariant reuses `SlotResourceDependents`, the same helper the planner uses, so it partly restates the implementation. A hand-written expectation for one case would make it independent.

**6. Architecture**
- **ARCH-DRY:** flag (the slot identity is rebuilt from a guess at `slotreport.go:63` instead of using `Discover`'s). Otherwise pass: `SlotLayout` and `ParseSlotPath` replaced the scattered path parsers.
- **ARCH-PURE:** pass. `PlanSlot` is pure; observation is the IO shell behind the `ProvisionIO` seam.
- **ARCH-PURPOSE:** pass for M1. The reconciler, `--show` and the audit all read the table; the recovery report and allocation are planned for M2/M3.
- **ARCH-MOCK:** pass. Observation runs against the real-git fixture through the same seam as production, and the weave fake moving to M2 is recorded in Revision (k).
- **ARCH-CONSTRAINTS:** pass for M1. `--show` adds a handful of git calls; the observe budget is measured in Task 2.6.
- **ARCH-SECURE:** flag (minor). The user-supplied path goes into the layout without symlink resolution (same root as the first Important finding). Dependencies outside the env are correctly treated as not slot state.
- **ARCH-ORDER:** pass. M1 keeps no state between events; the planner and observation are level-triggered.
- **ARCH-FUNERAL:** pass. M1 creates nothing durable.

For M2: the reconcile loop's no-progress guard compares full observations, including the `Reason` strings. Git error text that varies from run to run (paths, PIDs) would defeat the guard, so consider comparing only state and reading.

**7. Plan revision recommendations**
- Revision (o): `--show` resolves its slot through `Discover`'s identity, using git's own common directory rather than the `<primary>/.git` convention.
- Amend Revision (j): when a dependency is not a layer, the walk stops, and the other declared dependencies are not reported until it is repaired (this replaces Task 1.3's "others still reported").
- Task 1.7: the atlas section lives in `workspace-provisioning.md`.

```findings
findings:
  - id: new
    severity: Important
    family: slot-identity-from-git-not-convention
    title: |
      --show builds the slot layout from conventionalSlot's guessed common dir and an unresolved user path, while observation compares against git's resolved paths
    detail: |
      slotreport.go:63 uses conventionalSlot (common = primary/.git) and never runs EvalSymlinks on the ref. Git resolves symlinks in worktree list, --git-common-dir and --absolute-git-dir, so a primary reached through a symlink, or one with a gitfile .git, reads registration absent, branch broken (elsewhere) and host mismatched. Take the identity from Discover (RepoIdentity) and resolve the path; add a symlinked-primary fixture.
  - id: new
    severity: Important
    family: broken-only-on-positive-evidence
    title: |
      observeDep marks a git-readable clone unreadable when its git dir is not <path>/.git
    detail: |
      slotobserve.go:444 checks filepath.Dir(absolute-git-dir) != d.Path, so a gitfile-backed clone (linked worktree, separate-git-dir) becomes broken/unreadable, the reading that triggers set-aside. That violates R2. Use rev-parse --show-toplevel == d.Path and add a gitfile dependency test.
  - id: new
    severity: Important
    family: readme-tracks-user-surface
    title: |
      README not updated for couch --show repo:N, the slot resources and plan, and showing a slot with no thread
    detail: |
      README.md:383 and :528 still describe --show as one thread by tag or path.
  - id: new
    severity: Minor
    family: plan-tracks-implementation
    title: |
      A non-layer dependency hides every other declared dependency; plan step 1.3 still says the others are reported
    detail: |
      slotobserve.go:405 replaces the list with the NotLayer alone. When it is outside the env, in-env missing dependencies go unseen and setup can read present. Record this in Revision (j).
  - id: new
    severity: Minor
    family: stop-only-when-work-pending
    title: |
      A held weave lock makes a converged slot report a retryable stop
    detail: |
      slotplan.go:340 stops on SubLockHeld whatever the setup state, so --show of a healthy slot reports a stop while an agent compiles. Settle this before M2 assigns it a severity.
  - id: new
    severity: Minor
    family: stop-reason-precision
    title: |
      An unknown agent is reported with the live-agent hold reason
  - id: new
    severity: Minor
    family: error-surface-preserved
    title: |
      slotOfShowReference drops WorkspaceReferencePath's error, so --show repo:N of a missing slot shows the generic not-found message
```

---

## Re-review — 2026-10-05T18:44:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 387 — Slot reconciler: reconcile a Couch slot's dispersed state |
| repo | pair |
| issue file | workshop/issues/000387-add-slot-repairs-a-slot-whose-directory-was-deleted.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 57ae1bede1146c1cd360d265e17d78611687e5ca..82442380cab5d93ab73eecb84a36bfab0779cd8a |
| command | sdlc milestone-close --issue 387 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-05T18:44:55-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three Important findings from earlier rounds are fixed, and each fix has a regression test that I proved by mutation. I took HEAD with `git archive`, reverted the fix hunks of 82442380 in `slotobserve.go`, `slotreport.go` and `operationdispatch.go`, and ran the tests on that copy:
- `TestObserveGitfileDependencyIsReadable` fails on the reverted code: the dependency reads `broken/unreadable … has no repository of its own`.
- `TestShowThroughASymlinkedFleetReadsGitsIdentity` fails with exactly what BR-2 predicted: registration absent, host mismatched, branch elsewhere, and a planned `repair-host`.

The README now documents `--show repo:N`, the resource lines and the plan line. The BR-6 and BR-7 minors are fixed with tests that target the old behaviour. BR-5 is recorded in Revision (o) as an amendment to (j).

On HEAD, `go test ./cmd/internal/couchcore/` passes (144.8s), and the couchcmd slot-report render tests pass. The other couchcmd and artifactpath failures don't involve this window:
- **couchcmd:** the sandbox blocks `mkdir /tmp/pair-message-service-*`, and a leaked `PAIR_DATA_DIR` breaks the continuation writer tests.
- **artifactpath:** 33 unclassified-inventory files, none of them slot files and none in the diff.

Two minors are still open (BR-1, BR-8), plus one new minor about the error surface. None of them blocks.

**Strengths**
- `slotplan.go:154-157`: the set-aside guard now covers both agent states: an unknown agent gets its own hold reason (`unknown-agent`), separate from a running one (`live-agent`).
- `slotobserve.go:443-455`: `--show-toplevel == path` is the right positive-evidence test (R2). A clone whose `.git` is a gitfile reads present, and a directory inside another work tree still reads broken.
- `slotreport.go:91-108`: `slotIdentityFromGit` takes the identity from Discover first. It falls back to the conventional location only for a number known from leftovers, and even then it uses git's `RepoIdentity`.
- The new hand-written "unknown host blocks exactly its dependents" case (`slotplan_test.go:221`) checks I2 independently of `SlotResourceDependents`, so the test doesn't just restate the implementation.

**Critical**
- None.

**Important**
- None.

**Minor**
- **Error surface, same family as BR-8.** In `slotreport.go:72-89` and `operationdispatch.go:165-170`, three errors on the slot-resolution path of `--show` are still discarded:
  - a `retainedPhysicalPath` error other than not-exist returns `(…, false, nil)`;
  - a Discover error is returned as `slotErr`, but the dispatcher only reads it when thread resolution also failed;
  - so when the thread resolves and slot discovery fails, the slot section disappears with no word.
- **BR-8 (no regression test).** No test drives `show` with `repo:N` for a slot that doesn't exist and asserts the `slot … does not exist (existing: …)` message.
- **BR-1.** Tasks 2.5 and 3.1 in the plan still list test cases in prose.

**Test coverage**
- The BR-2, BR-3, BR-6 and BR-7 fixes all have tests that fail without the fix; I confirmed BR-2 and BR-3 by mutation, and BR-6 and BR-7 by reading the assertions.
- The BR-8 change has no test.

**Architecture**
- **ARCH-DRY:** pass. `retainedPhysicalPath` and Discover are reused rather than re-implemented.
- **ARCH-PURE:** pass. `PlanSlot` stays pure; the IO lives in the observer and Discover.
- **ARCH-PURPOSE:** pass for M1. `--show` works for a slot with no thread, which is the point of the issue.
- **ARCH-MOCK:** pass. The tests run against real git fixtures and the OS slot catalog.
- **ARCH-CONSTRAINTS:** pass. `--show` now calls Discover once per slot reference, which is acceptable for a diagnostic command.
- **ARCH-SECURE:** pass. Paths are resolved through symlinks before they become identity, and a path replaced during discovery is still re-inspected.
- **ARCH-ORDER:** pass. The new held-lock and unknown-agent conditions are encoded in `PlanSlot`'s switch, not added as flags.
- **ARCH-FUNERAL:** pass. Nothing durable is created in this window.

**Notes for M2**
- Revision (o) already records that M2's no-progress guard must compare observations on state and reading only, not the `Reason` text. Keep that when Task 2.2 lands.
- The `Sub` reading now changes meaning depending on `State` (a held lock on present setup is benign). OutcomeSeverity in Task 2.4 should rank from the plan's stops, not from raw readings.

**Plan revisions**
- When BR-8 gets its test, add one line to Revision (o) noting it.
- Otherwise none: Revision (o) matches the code.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 2.5 and 3.1 still list test cases in prose; Minor, can be compressed when M2/M3 start.
  - id: BR-2
    disposition: addressed
    note: |
      slotreport.go:68-108 resolves through retainedPhysicalPath and takes the identity from Discover; TestShowThroughASymlinkedFleetReadsGitsIdentity fails on the reverted code (registration absent, host mismatched, branch elsewhere).
  - id: BR-3
    disposition: addressed
    note: |
      slotobserve.go:447 uses rev-parse --show-toplevel == path; TestObserveGitfileDependencyIsReadable fails on the reverted code (broken/unreadable).
  - id: BR-4
    disposition: addressed
    note: |
      README.md:383 and :541-551 now document --show repo:N, the resource states and readings, the plan line, and a slot with no thread; this matches slotreport/couchcmd rendering.
  - id: BR-5
    disposition: addressed
    note: |
      Revision (o) amends (j): the walk stops at a non-layer dependency, the consequence is bounded, and Task 1.3's claim is retracted.
  - id: BR-6
    disposition: addressed
    note: |
      slotplan.go:235 stops on a held lock only when setup is not present; the test "a held weave lock stops only pending setup work" checks both sides.
  - id: BR-7
    disposition: addressed
    note: |
      StopReasonAgentUnknown is new and the test "an unknown agent holds with its own reason" pins it.
  - id: BR-8
    disposition: not-addressed
    note: |
      The code now returns slotErr, but no test drives show of repo:N for a missing slot and asserts the "does not exist (existing: …)" message.
findings:
  - id: new
    severity: Minor
    family: error-surface-preserved
    title: |
      --show still drops slot-resolution errors whenever thread resolution succeeds, or the physical path probe fails
    detail: |
      This is the 2nd finding in family error-surface-preserved; do not fix only this site. Rule: every error on --show's slot-resolution path (retainedPhysicalPath when the error is not not-exist, Discover, WorkspaceReferencePath) either fails the command or appears on the ShowResult as a slot-report error line. It never reduces the result to "not a slot". Today slotreport.go:73 returns (false, nil) on a path error, and operationdispatch.go:165-170 reads slotErr only when the thread lookup also failed. Make slotOfShowReference's error always reach the result (e.g. a SlotReport carrying PlanError) and add one test per error source.
```
