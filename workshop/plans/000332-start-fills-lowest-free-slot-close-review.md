# Boundary Review — pair#332 (whole-issue close)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..f658095d6d167e81288fa65dec1d87626ddfeb6b |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T13:49:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The lowest-free selector and archived-slot reuse work in focused tests, including preservation of leftover files and the menu’s commit round trip. Reuse nevertheless bypasses ownership safeguards and can replace a newly parked occupant. Notice ordering and README also diverge from the stated contract.

1. **Strengths**
   - `SelectStartSlot` is pure, treats `:0` uniformly, and rejects invalid candidate inventories.
   - The reuse integration test checks a fresh conversation, preserved files, and no unnecessary directory allocation.
   - `ReuseSlot` participates in the fingerprint; existing drift tests passed.

2. **Critical findings**
   - `slotstart.go:53,198`: numbered-slot read failures are omitted from admission, and reuse bypasses `checkSlotCreation`.
   - `slotstart.go:198`: calling the replacement-oriented `StartFreshSlot` does not enforce that the selected hole remains empty.

3. **Important findings**
   - `menu_render.go:414`: separate parked/lost loops discard combined numeric ordering.
   - `README.md:410`: unchanged instructions still describe the superseded allocation/admission behavior.

4. **Minor findings**
   - The issue’s Plan still describes classifier-based occupancy, action `StartFresh`, and the old notice format; implementation differs without a `## Revisions` entry.

5. **Test coverage notes**
   - Focused tests passed in both packages using `-count=1 -timeout=60s`.
   - The full package run produced no result before I interrupted it; full-suite success is unverified.
   - Add regressions for unreadable sibling slots, controlled occupancy changes before reuse, and interleaved parked/lost notices. The checked-off `:1` hole with `:2` parked integration case is also absent from `slotstart_test.go`.

6. **Architectural notes**
   - **ARCH-DRY — pass:** one shared allocation selector.
   - **ARCH-PURE — pass:** allocation remains deterministic and directly tested.
   - **ARCH-PURPOSE — flag:** admission and notice ordering miss explicit requirements.
   - **ARCH-MOCK — pass for this delta:** existing injected process/artifact/workspace seams are reused.
   - **ARCH-CONSTRAINTS — pass:** bounded sequential allocation; no new concurrency.
   - **ARCH-SECURE — flag:** unreadable persisted slot state can be treated as available capacity.
   - **ARCH-ORDER — flag:** reuse does not atomically enforce the accepted empty-slot condition.
   - **ARCH-FUNERAL — pass:** existing storage lifecycles are reused; no new durable artifact family.

7. **Plan revision recommendations**
   - Append a dated `## Revisions` entry explaining snapshot-based occupancy, `StartCreate + ReuseSlot`, and actionable parked/lost notices. Reconcile the active Plan and coverage checkboxes with delivered behavior.

```findings
findings:
  - id: new
    severity: Critical
    family: uncertain-inventory-admission
    title: |
      Slot reuse bypasses unreadable repository ownership
    detail: |
      cmd/internal/couchcore/slotstart.go:53 checks only snapshot.Unreadable, but threadstore_snapshot.go:26-40 stores numbered-slot read failures in snapshot.Slots[].Err. With an archived :1 and unreadable :2, allocation selects :1; notice collection ignores the unreadable row, and slotstart.go:198 bypasses checkSlotCreation. Enforce repository-wide uncertainty admission for every allocation branch and test a corrupt sibling slot through PrepareStart/SpawnPrepared. ARCH-SECURE, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: allocation-precondition-atomicity
    title: |
      Filling a hole can replace a newly parked occupant
    detail: |
      cmd/internal/couchcore/slotstart.go:198 delegates reuse to StartFreshSlot. If another actor installs a parked conversation after SpawnPrepared's fingerprint comparison but before slotrecovery.go:268 reads current state, StartFreshSlot accepts its absent process ownership and archives/replaces that conversation. Its byte comparison protects only the later observation, not the accepted empty-slot condition. Require an atomic empty-current precondition for allocation reuse, preserving explicit fresh-slot replacement behavior, and test this controlled interleaving. ARCH-ORDER.
  - id: new
    severity: Important
    family: combined-notice-ordering
    title: |
      Mixed reuse notices are not rendered in slot-number order
    detail: |
      cmd/internal/couchtty/menu_render.go:414-422 renders all parked notices before all lost notices. A lost :1 and parked :2 therefore render :2 before :1 despite the Spec's combined-order requirement; sorting rows before splitting them does not preserve that order. Carry an ordered typed notice list and test interleaved kinds and multi-digit numbers. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: user-facing-documentation-parity
    title: |
      README update is missing for changed add-slot behavior
    detail: |
      README.md:410-417 still says subsequent starts create numbered slots and instructs users to resume parked threads before adding another slot. Update it in this range to explain lowest-free reuse, inherited checkout contents, nonblocking reuse notices, and the lowercase add slot label.
  - id: new
    severity: Minor
    family: plan-implementation-traceability
    title: |
      Active Plan retains superseded design claims
    detail: |
      workshop/issues/000332-start-fills-lowest-free-slot.md:68-86 describes classifier-based occupancy, StartFresh action mapping, and the old parked notice text. The implementation uses snapshot records, StartCreate plus ReuseSlot, and actionable notices. Add a dated Revisions entry and reconcile the active Plan and claimed test coverage.
```

---

## Re-review — 2026-09-25T14:09:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..0439c2c74633cbe9ee13f29a7ac4d8c900c40fae |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T14:09:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Lowest-free allocation and ordinary archived-slot reuse are implemented, and the README now describes the behavior. However, the empty-slot precondition remains incomplete, reuse bypasses launch-profile revalidation, and two claimed regression scenarios lack effective coverage.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      slotstart.go:58 checks snapshot.Slots errors before selecting any allocation branch. TestManagedCreateRefusesUnreadableSiblingSlot exercises an archived :1 beside corrupt :2 through PrepareStart and asserts the slot-specific refusal.
  - id: BR-2
    disposition: not-addressed
    note: |
      slotrecovery.go:279 rejects decoded occupants but does not require absence: corrupt current bytes produce Exists=true, Record=nil and pass this guard. If those bytes appear after fingerprint validation, the recovery path can preserve them as recovery evidence and replace them. Require an absent current file, enforced by the existing locked comparison. The added test directly calls startFreshSlot; it does not exercise the requested controlled interleaving through SpawnPrepared.
  - id: BR-3
    disposition: not-addressed
    note: |
      The ordered typed list fixes the inspected implementation, but menu_async_test.go:217 supplies parked :1, parked :2, lost :3. Rendering all parked notices before lost notices still passes. Add lost :1, parked :2, lost :10, including production notice collection, and verify restoring kind grouping or removing numeric sorting fails.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:410-423 now documents lowest-free reuse, inherited branch/files, nonblocking parked/lost guidance, and lowercase add slot. These match SelectStartSlot, reuse routing, menuItemLabel, and notice rendering.
  - id: BR-5
    disposition: not-addressed
    note: |
      The dated Revisions entry and active design now reflect snapshot occupancy, StartCreate plus ReuseSlot, and typed notices. However, issue lines 90-92 still mark the :1-hole/:2-parked integration case complete without a corresponding test; mixed-order regression coverage is also overstated.
findings:
  - id: new
    severity: Critical
    family: accepted-launch-authority
    title: |
      Reused-slot creation bypasses accepted launch-profile revalidation
    detail: |
      slotstart.go:216-217 passes only path and RequestedAgent into startFreshSlot. That function independently resolves a profile at slotrecovery.go:289 and launches after workspace readiness at lines 347-354 without comparing against the accepted resolution. Newly provisioned slots perform that comparison through couch.go:559-563 and revalidateCreatedSlot. A default/preference change after fingerprint comparison can therefore launch an unaccepted profile, and changes during readiness escape the existing drift refusal. Carry the accepted resolution through reuse and share post-readiness authority validation; extend TestManagedCreateRefusesProfileDriftDuringSetup to an archived-slot reuse fixture. ARCH-ORDER, ARCH-PURPOSE.
```

1. **Strengths**
   - `SelectStartSlot` is pure and directly tested for primary, missing-directory, and existing-directory holes.
   - `TestManagedCreateReusesArchivedSlotBeforeAllocating` exercises the menu’s `CommitArgs`/`SpawnPrepared` path, preserves leftover files, and verifies a new conversation identity.
   - Existing locked byte comparisons provide useful protection against changes after observing slot state.
   - README and atlas updates explain the allocation change.

2. **Critical findings**
   - **BR-2:** Require `!old.Exists`, not merely `old.Record == nil`, for allocation reuse. Preserve explicit fresh-slot recovery behavior. Test valid and corrupt arrivals between fingerprint comparison and current observation, asserting no replacement or launch.
   - **New accepted-launch-authority finding:** Preserve and revalidate accepted settings across reuse and readiness, as detailed above.

3. **Important findings**
   - **BR-3:** The test does not distinguish correct combined ordering from the prior implementation. Exercise interleaved kinds and multi-digit slot numbers through collection and rendering.

4. **Minor findings**
   - **BR-5:** Reconcile checked coverage claims with actual tests.

5. **Test coverage notes**
   - Focused couchcore allocation/create/recovery tests passed.
   - Focused couchtty preview and lowercase-label tests passed.
   - `git diff --check` passed.
   - The full couchcore/couchtty run had not completed at reporting time; no full-suite success is claimed.
   - No mutation experiments were performed under this read-only review.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** Central selector and existing lifecycle machinery are reused.
   - **ARCH-PURE — Pass:** Allocation logic is IO-free and directly tested.
   - **ARCH-PURPOSE — Flag:** Accepted-start guarantees and claimed regression coverage remain incomplete.
   - **ARCH-MOCK — Pass:** Changed integration tests use existing provision/process/artifact seams; no new external dependency.
   - **ARCH-CONSTRAINTS — Pass:** No new unbounded concurrency; allocation remains inventory-bounded.
   - **ARCH-SECURE — Flag:** Corrupt current state can satisfy the supposed empty-slot condition.
   - **ARCH-ORDER — Flag:** Atomic absence and accepted-profile authority need enforcement across reuse.
   - **ARCH-FUNERAL — Pass:** No new durable artifact family; existing retention machinery remains responsible for cleanup.

7. **Plan revision recommendations**
   - Append a dated `## Revisions` entry defining reuse as **confirmed current-file absence**, and preserving accepted launch authority through readiness.
   - Reconcile the checked test claims after adding controlled-interleaving, interleaved-notice, and hole-with-parked-sibling regressions.

---

## Re-review — 2026-09-25T14:27:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..e247198f61ca301e8d71362e40072e9a0f0be986 |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T14:27:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Lowest-free allocation, checkout reuse, and ordered notices are implemented with focused regression coverage. BR-2, BR-3, and BR-5 are addressed. BR-6 remains blocking: reuse revalidates current configuration but can still launch a different, previously resolved profile.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Unreadable slot-current admission remains enforced in resolveManagedStart, with TestManagedCreateRefusesUnreadableSiblingSlot.
  - id: BR-2
    disposition: addressed
    note: |
      startFreshSlot rejects old.Exists for allocation reuse; replaceSlotCurrent compares existence and bytes under the store lock. TestStartCreateReuseRefusesAnOccupiedCurrent exercises the new guard; existing concurrent-mutation coverage exercises the locked comparison.
  - id: BR-3
    disposition: addressed
    note: |
      ReuseNotices carries a combined typed sequence. Sorting and rendering tests cover lost :1, parked :2, and lost :10; focused tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      README.md now documents lowest-free allocation, checkout inheritance, and actionable parked/lost reuse guidance, matching the changed allocation and rendering paths.
  - id: BR-5
    disposition: addressed
    note: |
      The active Plan now describes snapshot occupancy, StartCreate plus ReuseSlot, and typed actionable notices; dated Revisions explain the changes.
  - id: BR-6
    disposition: not-addressed
    note: |
      slotrecovery.go:289 independently resolves profile B, while lines 351-352 validate current configuration against accepted profile A. If configuration returns to A during readiness, validation passes, but lines 340 and 357 still launch B. The added regression covers persistent drift during readiness, not this A-to-B-to-A sequence. ARCH-ORDER, ARCH-PURPOSE.
```

1. **Strengths**

   - `SelectStartSlot` is pure and directly tests primary reuse, directory holes, and uncertain inventory.
   - Archived-slot integration coverage commits through `CommitArgs`/`SpawnPrepared` and verifies leftover files survive.
   - Empty-current admission composes with the existing locked replacement transaction.
   - Mixed notice ordering is tested numerically, including multi-digit slots.

2. **Critical findings**

   **BR-6 — accepted launch authority remains incomplete**, [slotrecovery.go:289](/Users/xianxu/workspace/pair/cmd/internal/couchcore/slotrecovery.go:289).

   The accepted resolution reaches reuse, but its profile does not supply the claim or launch payload. Revalidation compares accepted configuration with a later observation, leaving the intermediate launch profile unchecked.

   Apply the family-wide rule: **every launch payload and claimed profile must derive from the accepted resolution; later reads may invalidate that authority, never substitute it.** Preserve explicit fresh-slot behavior separately. Add a controlled A→B→A regression that inspects the actual launch profile and forbidden launch effects.

3. **Important findings**

   None additional.

4. **Minor findings**

   None additional.

5. **Test coverage notes**

   Focused selector, occupied-current, profile-drift, notice-sort, and renderer tests passed. The full `couchcore`/`couchtty` run had not completed at reporting time. No mutation experiment was performed under this read-only review. The remaining BR-6 sequence is established by source tracing and lacks regression coverage.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing replacement and revalidation machinery reused.
   - **ARCH-PURE: pass** — allocation remains independently testable.
   - **ARCH-PURPOSE: flag** — BR-6 violates accepted launch authority.
   - **ARCH-MOCK: pass** — existing injected dependency seams retained.
   - **ARCH-CONSTRAINTS: pass** — no new concurrency or unbounded allocation search.
   - **ARCH-SECURE: pass** — uncertain inventory is refused.
   - **ARCH-ORDER: flag** — validation does not bind the profile actually launched.
   - **ARCH-FUNERAL: pass** — no new durable artifact family.

7. **Plan revision recommendations**

   Append a dated `## Revisions` entry specifying accepted-profile derivation for claims and launches, plus the A→B→A regression. Reopen the checked profile-preservation step until that evidence passes.

---

## Re-review — 2026-09-25T14:36:56-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..08a298fc3eb20c23dfa549841150e3e9e2ea3c50 |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T14:36:56-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Lowest-free allocation, reuse notices, and documentation are delivered. **BR-6 remains open in the pinned commit:** post-readiness validation checks the accepted resolution, but the launch payload still comes from an independent profile read. A scratch regression reproduced an unaccepted launch. The uncommitted correction passes that regression, but is outside this review window.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Allocation rejects unreadable slot-current observations; TestManagedCreateRefusesUnreadableSiblingSlot covers an unreadable sibling beside a reusable hole.
  - id: BR-2
    disposition: addressed
    note: |
      Reuse requires physical current-file absence and retains compare-and-replace protection; occupied-current and concurrent-change tests cover the guards.
  - id: BR-3
    disposition: addressed
    note: |
      Typed notices are sorted numerically across kinds; selector and rendering tests cover slots 1, 2, and 10.
  - id: BR-4
    disposition: addressed
    note: |
      README.md documents lowest-free allocation, checkout inheritance, parked-work admission, and reuse actions, consistent with the implementation.
  - id: BR-5
    disposition: addressed
    note: |
      The active Plan now describes StartCreate plus ReuseSlot, physical absence, and typed ordered notices; revisions record the superseded design.
  - id: BR-6
    disposition: not-addressed
    note: |
      Pinned slotrecovery.go:289 independently resolves the payload used at lines 340 and 357. Revalidation at lines 351–352 cannot detect a transient profile change that returns to its accepted value. A scratch regression selecting an agent without saved argv launched [--model unaccepted] after three profile reads. Sourcing the payload from accepted.Profile and its provenance made the regression pass; that correction is uncommitted and the regression is absent from the pinned range. ARCH-ORDER, ARCH-PURPOSE.
```

1. **Strengths**
   - `SelectStartSlot` isolates allocation in a pure function with primary, directory-hole, archived-hole, and uncertain-inventory cases.
   - Reuse integration coverage exercises the menu’s `CommitArgs` round trip and preservation of leftover files.
   - Typed notices preserve numeric ordering through rendering; README and atlas changes explain the allocation change.

2. **Critical findings**
   - **BR-6 — accepted launch authority remains split**, `cmd/internal/couchcore/slotrecovery.go:289,340,351–357`. Sequence: acceptance reads A, payload construction reads B, final validation reads A, launch executes B. Enforce one rule across accepted-create paths: **payload and provenance derive exclusively from the accepted resolution; subsequent reads may only refuse launch**. Commit the correction and add a regression covering this sequence. This remains the existing `accepted-launch-authority` finding, not a new finding.

3. **Important findings**
   - None newly raised.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Pinned `couchtty` suite passed after generating runtime assets in the scratch copy.
   - Existing reuse drift test passed; it covers a change that persists through readiness.
   - Additional scratch regression failed against pinned code with `--model unaccepted`, then passed with the uncommitted payload-authority correction.
   - Full `couchcore` run was stopped before completion; no full-suite pass is claimed.

6. **Architectural notes**
   - **ARCH-DRY — flag:** independently resolving the accepted launch payload creates competing authority (BR-6).
   - **ARCH-PURE — pass:** allocation stays pure; orchestration owns IO.
   - **ARCH-PURPOSE — flag:** accepted-profile preservation remains incomplete in the pinned implementation.
   - **ARCH-MOCK — pass:** changed paths use existing injected provisioning and launch boundaries.
   - **ARCH-CONSTRAINTS — pass:** allocation is inventory-bounded; no new concurrent fan-out.
   - **ARCH-SECURE — pass:** uncertain inventory and occupied-current state refuse reuse.
   - **ARCH-ORDER — flag:** the change-and-return sequence escapes profile validation.
   - **ARCH-FUNERAL — pass:** no new durable artifact family; reuse retains existing lifecycle ownership.

7. **Plan revision recommendations**
   - Append a revision recording the change-and-return regression and its verification. The checked payload-authority item already states the correct requirement, but its implementation must land in the reviewed commit before closure.

---

## Re-review — 2026-09-25T14:42:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..9393d70bb1d967e30d063fe2aa9f65b5105bfc4e |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T14:42:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation now carries the accepted profile through reused-slot creation and revalidates after readiness. However, BR-6 lacks regression evidence for the final payload-binding correction: restoring the previous profile-selection code still passes the relevant tests. Under this gate’s test-fails-without-fix requirement, BR-6 remains open.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      resolveManagedStart rejects unreadable slot-current inventory; TestManagedCreateRefusesUnreadableSiblingSlot covers that refusal.
  - id: BR-2
    disposition: addressed
    note: |
      Reuse requires physical current-record absence, and replaceSlotCurrent retains the observed-state comparison before replacement.
  - id: BR-3
    disposition: addressed
    note: |
      Typed reuse notices are sorted numerically across kinds; selector and rendering tests cover mixed ordering.
  - id: BR-4
    disposition: addressed
    note: |
      README now describes lowest-free reuse, inherited checkout contents, and parked/lost reuse guidance, matching the implementation.
  - id: BR-5
    disposition: addressed
    note: |
      The active Plan describes StartCreate plus ReuseSlot, ordered typed notices, and accepted-profile validation; revisions record these changes.
  - id: BR-6
    disposition: not-addressed
    note: |
      slotrecovery.go:290-301 now binds the launch payload to the accepted profile, but slotstart_test.go:162 only tests persistent drift during readiness. Restoring the parent commit's profile-selection code through a temporary Go overlay leaves both the drift and archived-reuse tests passing. Add a deterministic transient-profile-change test asserting the actual launched profile and provenance, and demonstrate failure without the binding fix. ARCH-ORDER, ARCH-PURPOSE; existing family accepted-launch-authority.
```

1. **Strengths**
   - Allocation is isolated in the pure `SelectStartSlot`, with primary, occupied, and directory-hole cases.
   - Archived-slot integration coverage exercises the menu’s `CommitArgs` round trip and preserves leftover files.
   - Reuse shares `revalidateCreatedSlot` with newly provisioned slots.

2. **Critical findings**
   - **BR-6 remains open for regression evidence**, rather than an observed defect in the current implementation. Test the sequence accepted profile A → intervening profile B → validation sees A again. Assert that launch arguments and provenance remain A. The test must fail when `slotrecovery.go:290-301` is reverted.

3. **Important findings:** None newly raised.

4. **Minor findings:** None newly raised.

5. **Test coverage**
   - Focused drift and archived-reuse tests passed on HEAD.
   - Reverting accepted-profile binding: both tests still passed.
   - Removing post-readiness validation: drift test failed as expected.
   - Full `couchcore`/`couchtty` run was interrupted without a result; no full-suite pass claimed. Repository files were unchanged.

6. **Architecture**
   - **ARCH-DRY:** Pass—shared authority validator.
   - **ARCH-PURE:** Pass—pure allocation core.
   - **ARCH-PURPOSE:** Flag—BR-6’s required regression evidence remains incomplete.
   - **ARCH-MOCK:** Pass—existing provision/runtime seams support controlled tests.
   - **ARCH-CONSTRAINTS:** Pass—no new unbounded concurrency.
   - **ARCH-SECURE:** Pass—uncertain inventory refuses admission.
   - **ARCH-ORDER:** Flag—transient profile substitution lacks sequence coverage.
   - **ARCH-FUNERAL:** Pass—no new durable artifact family.

7. **Plan revisions:** Append a revision naming the transient-profile sequence and mutation evidence; retain the existing implementation design.

---

## Re-review — 2026-09-25T14:49:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 332 — Starting a thread fills the lowest free slot number, :0 included |
| repo | pair |
| issue file | workshop/issues/000332-start-fills-lowest-free-slot.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb007083dc98f052b93761bd7fe0f26c98274646..fd8f6d5afc5888630da0885bed6f8e55c79b7931 |
| command | sdlc close --issue 332 |
| reviewer | codex |
| timestamp | 2026-09-25T14:49:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range fulfills the issue’s spec and plan. Allocation, hole reuse, profile authority, unreadable inventory refusal, notice ordering, README, and atlas updates are covered. Focused regression tests pass; the broader package run was stopped after producing no output.

1. Strengths:

- Lowest-free allocation and `:0` handling are clearly separated and tested.
- Reuse preserves leftover files and rejects occupied current records.
- Accepted launch profiles remain authoritative through readiness (`slotstart.go`, `slotrecovery.go`).
- Mixed parked/lost notices are typed, ordered, and rendered consistently.
- README and atlas documentation were updated.

2. Critical findings:

None.

3. Important findings:

None.

4. Minor findings:

None.

5. Test coverage notes:

Focused `couchcore` regressions pass, including allocation, reuse, profile drift, transient profile authority, parked sibling notices, and unreadable slot refusal. Focused `couchtty` rendering tests pass. `git diff --check` passes.

6. Architectural notes for upcoming work:

- ARCH-DRY: pass.
- ARCH-PURE: pass.
- ARCH-PURPOSE: pass; allocation and all documented consumers are updated.
- ARCH-MOCK: pass; no new external dependency surface.
- ARCH-CONSTRAINTS: pass; no unbounded new work or fan-out.
- ARCH-SECURE: pass; unreadable persisted inventory fails visibly.
- ARCH-ORDER: pass; accepted resolution and reuse guards preserve ordering and authority.
- ARCH-FUNERAL: pass; no new durable artifact family is introduced.

7. Plan revision recommendations:

None.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Reuse now carries the accepted launch profile through readiness, revalidates drift before launch, and launches from the accepted profile. TestManagedCreateReuseRefusesProfileDriftDuringSetup and TestManagedCreateReuseLaunchesAcceptedTransientProfile provide regression evidence.
```
