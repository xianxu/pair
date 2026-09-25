---
gate: boundary-review
issue: 332
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-25T13:49:07-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Slot reuse bypasses unreadable repository ownership
          detail: cmd/internal/couchcore/slotstart.go:53 checks only snapshot.Unreadable, but threadstore_snapshot.go:26-40 stores numbered-slot read failures in snapshot.Slots[].Err. With an archived :1 and unreadable :2, allocation selects :1; notice collection ignores the unreadable row, and slotstart.go:198 bypasses checkSlotCreation. Enforce repository-wide uncertainty admission for every allocation branch and test a corrupt sibling slot through PrepareStart/SpawnPrepared. ARCH-SECURE, ARCH-PURPOSE.
          family: uncertain-inventory-admission
          round: 1
        - id: BR-2
          severity: Critical
          title: Filling a hole can replace a newly parked occupant
          detail: cmd/internal/couchcore/slotstart.go:198 delegates reuse to StartFreshSlot. If another actor installs a parked conversation after SpawnPrepared's fingerprint comparison but before slotrecovery.go:268 reads current state, StartFreshSlot accepts its absent process ownership and archives/replaces that conversation. Its byte comparison protects only the later observation, not the accepted empty-slot condition. Require an atomic empty-current precondition for allocation reuse, preserving explicit fresh-slot replacement behavior, and test this controlled interleaving. ARCH-ORDER.
          family: allocation-precondition-atomicity
          round: 1
        - id: BR-3
          severity: Important
          title: Mixed reuse notices are not rendered in slot-number order
          detail: cmd/internal/couchtty/menu_render.go:414-422 renders all parked notices before all lost notices. A lost :1 and parked :2 therefore render :2 before :1 despite the Spec's combined-order requirement; sorting rows before splitting them does not preserve that order. Carry an ordered typed notice list and test interleaved kinds and multi-digit numbers. ARCH-PURPOSE.
          family: combined-notice-ordering
          round: 1
        - id: BR-4
          severity: Important
          title: README update is missing for changed add-slot behavior
          detail: README.md:410-417 still says subsequent starts create numbered slots and instructs users to resume parked threads before adding another slot. Update it in this range to explain lowest-free reuse, inherited checkout contents, nonblocking reuse notices, and the lowercase add slot label.
          family: user-facing-documentation-parity
          round: 1
        - id: BR-5
          severity: Minor
          title: Active Plan retains superseded design claims
          detail: workshop/issues/000332-start-fills-lowest-free-slot.md:68-86 describes classifier-based occupancy, StartFresh action mapping, and the old parked notice text. The implementation uses snapshot records, StartCreate plus ReuseSlot, and actionable notices. Add a dated Revisions entry and reconcile the active Plan and claimed test coverage.
          family: plan-implementation-traceability
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-25T14:09:29-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: slotstart.go:58 checks snapshot.Slots errors before selecting any allocation branch. TestManagedCreateRefusesUnreadableSiblingSlot exercises an archived :1 beside corrupt :2 through PrepareStart and asserts the slot-specific refusal.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'slotrecovery.go:279 rejects decoded occupants but does not require absence: corrupt current bytes produce Exists=true, Record=nil and pass this guard. If those bytes appear after fingerprint validation, the recovery path can preserve them as recovery evidence and replace them. Require an absent current file, enforced by the existing locked comparison. The added test directly calls startFreshSlot; it does not exercise the requested controlled interleaving through SpawnPrepared.'
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The ordered typed list fixes the inspected implementation, but menu_async_test.go:217 supplies parked :1, parked :2, lost :3. Rendering all parked notices before lost notices still passes. Add lost :1, parked :2, lost :10, including production notice collection, and verify restoring kind grouping or removing numeric sorting fails.
          round: 2
        - id: BR-4
          disposition: addressed
          note: README.md:410-423 now documents lowest-free reuse, inherited branch/files, nonblocking parked/lost guidance, and lowercase add slot. These match SelectStartSlot, reuse routing, menuItemLabel, and notice rendering.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: The dated Revisions entry and active design now reflect snapshot occupancy, StartCreate plus ReuseSlot, and typed notices. However, issue lines 90-92 still mark the :1-hole/:2-parked integration case complete without a corresponding test; mixed-order regression coverage is also overstated.
          round: 2
      findings:
        - id: BR-6
          severity: Critical
          title: Reused-slot creation bypasses accepted launch-profile revalidation
          detail: slotstart.go:216-217 passes only path and RequestedAgent into startFreshSlot. That function independently resolves a profile at slotrecovery.go:289 and launches after workspace readiness at lines 347-354 without comparing against the accepted resolution. Newly provisioned slots perform that comparison through couch.go:559-563 and revalidateCreatedSlot. A default/preference change after fingerprint comparison can therefore launch an unaccepted profile, and changes during readiness escape the existing drift refusal. Carry the accepted resolution through reuse and share post-readiness authority validation; extend TestManagedCreateRefusesProfileDriftDuringSetup to an archived-slot reuse fixture. ARCH-ORDER, ARCH-PURPOSE.
          family: accepted-launch-authority
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-25T14:27:39-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Unreadable slot-current admission remains enforced in resolveManagedStart, with TestManagedCreateRefusesUnreadableSiblingSlot.
          round: 3
        - id: BR-2
          disposition: addressed
          note: startFreshSlot rejects old.Exists for allocation reuse; replaceSlotCurrent compares existence and bytes under the store lock. TestStartCreateReuseRefusesAnOccupiedCurrent exercises the new guard; existing concurrent-mutation coverage exercises the locked comparison.
          round: 3
        - id: BR-3
          disposition: addressed
          note: ReuseNotices carries a combined typed sequence. Sorting and rendering tests cover lost :1, parked :2, and lost :10; focused tests pass.
          round: 3
        - id: BR-4
          disposition: addressed
          note: README.md now documents lowest-free allocation, checkout inheritance, and actionable parked/lost reuse guidance, matching the changed allocation and rendering paths.
          round: 3
        - id: BR-5
          disposition: addressed
          note: The active Plan now describes snapshot occupancy, StartCreate plus ReuseSlot, and typed actionable notices; dated Revisions explain the changes.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: slotrecovery.go:289 independently resolves profile B, while lines 351-352 validate current configuration against accepted profile A. If configuration returns to A during readiness, validation passes, but lines 340 and 357 still launch B. The added regression covers persistent drift during readiness, not this A-to-B-to-A sequence. ARCH-ORDER, ARCH-PURPOSE.
          round: 3
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-25T14:36:56-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Allocation rejects unreadable slot-current observations; TestManagedCreateRefusesUnreadableSiblingSlot covers an unreadable sibling beside a reusable hole.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Reuse requires physical current-file absence and retains compare-and-replace protection; occupied-current and concurrent-change tests cover the guards.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Typed notices are sorted numerically across kinds; selector and rendering tests cover slots 1, 2, and 10.
          round: 4
        - id: BR-4
          disposition: addressed
          note: README.md documents lowest-free allocation, checkout inheritance, parked-work admission, and reuse actions, consistent with the implementation.
          round: 4
        - id: BR-5
          disposition: addressed
          note: The active Plan now describes StartCreate plus ReuseSlot, physical absence, and typed ordered notices; revisions record the superseded design.
          round: 4
        - id: BR-6
          disposition: not-addressed
          note: Pinned slotrecovery.go:289 independently resolves the payload used at lines 340 and 357. Revalidation at lines 351–352 cannot detect a transient profile change that returns to its accepted value. A scratch regression selecting an agent without saved argv launched [--model unaccepted] after three profile reads. Sourcing the payload from accepted.Profile and its provenance made the regression pass; that correction is uncommitted and the regression is absent from the pinned range. ARCH-ORDER, ARCH-PURPOSE.
          round: 4
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-09-25T14:42:35-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: resolveManagedStart rejects unreadable slot-current inventory; TestManagedCreateRefusesUnreadableSiblingSlot covers that refusal.
          round: 5
        - id: BR-2
          disposition: addressed
          note: Reuse requires physical current-record absence, and replaceSlotCurrent retains the observed-state comparison before replacement.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Typed reuse notices are sorted numerically across kinds; selector and rendering tests cover mixed ordering.
          round: 5
        - id: BR-4
          disposition: addressed
          note: README now describes lowest-free reuse, inherited checkout contents, and parked/lost reuse guidance, matching the implementation.
          round: 5
        - id: BR-5
          disposition: addressed
          note: The active Plan describes StartCreate plus ReuseSlot, ordered typed notices, and accepted-profile validation; revisions record these changes.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: slotrecovery.go:290-301 now binds the launch payload to the accepted profile, but slotstart_test.go:162 only tests persistent drift during readiness. Restoring the parent commit's profile-selection code through a temporary Go overlay leaves both the drift and archived-reuse tests passing. Add a deterministic transient-profile-change test asserting the actual launched profile and provenance, and demonstrate failure without the binding fix. ARCH-ORDER, ARCH-PURPOSE; existing family accepted-launch-authority.
          round: 5
      recipe: milestone-review
      blocked: true
    - "n": 6
      timestamp: "2026-09-25T14:49:40-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: Reuse now carries the accepted launch profile through readiness, revalidates drift before launch, and launches from the accepted profile. TestManagedCreateReuseRefusesProfileDriftDuringSetup and TestManagedCreateReuseLaunchesAcceptedTransientProfile provide regression evidence.
          round: 6
      recipe: milestone-review
      blocked: false
    - "n": 7
      timestamp: "2026-09-25T14:55:44-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Unreadable repository and slot inventory refuses allocation; focused unreadable-sibling regression passes.
          round: 7
        - id: BR-2
          disposition: addressed
          note: Reuse requires physical current-record absence and preserves compare-and-replace protection.
          round: 7
        - id: BR-3
          disposition: addressed
          note: Typed mixed notices are numerically ordered, including multi-digit slots.
          round: 7
        - id: BR-4
          disposition: addressed
          note: README documents lowest-free reuse, inherited checkout state, and parked/lost guidance.
          round: 7
        - id: BR-5
          disposition: addressed
          note: The active issue plan and revisions match the implemented selector, reuse, notices, and authority behavior.
          round: 7
        - id: BR-6
          disposition: addressed
          note: Accepted profile authority is preserved through readiness and launch; the transient A-to-B-to-A regression passes.
          round: 7
      findings:
        - id: BR-7
          severity: Important
          title: Atlas retains stale SelectNewSlot allocation semantics
          detail: atlas/workspace-provisioning.md:69-71 still describes lowest-unused-positive allocation, contradicting SelectStartSlot, :0 reuse, and the current lowest-free rule. This is the 2nd finding in family user-facing-documentation-parity; sweep stale allocation terminology across the atlas.
          family: user-facing-documentation-parity
          round: 7
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#332 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-25T13:49:07-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `uncertain-inventory-admission` Slot reuse bypasses unreadable repository ownership
  cmd/internal/couchcore/slotstart.go:53 checks only snapshot.Unreadable, but threadstore_snapshot.go:26-40 stores numbered-slot read failures in snapshot.Slots[].Err. With an archived :1 and unreadable :2, allocation selects :1; notice collection ignores the unreadable row, and slotstart.go:198 bypasses checkSlotCreation. Enforce repository-wide uncertainty admission for every allocation branch and test a corrupt sibling slot through PrepareStart/SpawnPrepared. ARCH-SECURE, ARCH-PURPOSE.
- **BR-2** [Critical] `allocation-precondition-atomicity` Filling a hole can replace a newly parked occupant
  cmd/internal/couchcore/slotstart.go:198 delegates reuse to StartFreshSlot. If another actor installs a parked conversation after SpawnPrepared's fingerprint comparison but before slotrecovery.go:268 reads current state, StartFreshSlot accepts its absent process ownership and archives/replaces that conversation. Its byte comparison protects only the later observation, not the accepted empty-slot condition. Require an atomic empty-current precondition for allocation reuse, preserving explicit fresh-slot replacement behavior, and test this controlled interleaving. ARCH-ORDER.
- **BR-3** [Important] `combined-notice-ordering` Mixed reuse notices are not rendered in slot-number order
  cmd/internal/couchtty/menu_render.go:414-422 renders all parked notices before all lost notices. A lost :1 and parked :2 therefore render :2 before :1 despite the Spec's combined-order requirement; sorting rows before splitting them does not preserve that order. Carry an ordered typed notice list and test interleaved kinds and multi-digit numbers. ARCH-PURPOSE.
- **BR-4** [Important] `user-facing-documentation-parity` README update is missing for changed add-slot behavior
  README.md:410-417 still says subsequent starts create numbered slots and instructs users to resume parked threads before adding another slot. Update it in this range to explain lowest-free reuse, inherited checkout contents, nonblocking reuse notices, and the lowercase add slot label.
- **BR-5** [Minor] `plan-implementation-traceability` Active Plan retains superseded design claims
  workshop/issues/000332-start-fills-lowest-free-slot.md:68-86 describes classifier-based occupancy, StartFresh action mapping, and the old parked notice text. The implementation uses snapshot records, StartCreate plus ReuseSlot, and actionable notices. Add a dated Revisions entry and reconcile the active Plan and claimed test coverage.

## Round 2 — 2026-09-25T14:09:29-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — slotstart.go:58 checks snapshot.Slots errors before selecting any allocation branch. TestManagedCreateRefusesUnreadableSiblingSlot exercises an archived :1 beside corrupt :2 through PrepareStart and asserts the slot-specific refusal.
- BR-2 — not-addressed — slotrecovery.go:279 rejects decoded occupants but does not require absence: corrupt current bytes produce Exists=true, Record=nil and pass this guard. If those bytes appear after fingerprint validation, the recovery path can preserve them as recovery evidence and replace them. Require an absent current file, enforced by the existing locked comparison. The added test directly calls startFreshSlot; it does not exercise the requested controlled interleaving through SpawnPrepared.
- BR-3 — not-addressed — The ordered typed list fixes the inspected implementation, but menu_async_test.go:217 supplies parked :1, parked :2, lost :3. Rendering all parked notices before lost notices still passes. Add lost :1, parked :2, lost :10, including production notice collection, and verify restoring kind grouping or removing numeric sorting fails.
- BR-4 — addressed — README.md:410-423 now documents lowest-free reuse, inherited branch/files, nonblocking parked/lost guidance, and lowercase add slot. These match SelectStartSlot, reuse routing, menuItemLabel, and notice rendering.
- BR-5 — not-addressed — The dated Revisions entry and active design now reflect snapshot occupancy, StartCreate plus ReuseSlot, and typed notices. However, issue lines 90-92 still mark the :1-hole/:2-parked integration case complete without a corresponding test; mixed-order regression coverage is also overstated.

### Raised

- **BR-6** [Critical] `accepted-launch-authority` Reused-slot creation bypasses accepted launch-profile revalidation
  slotstart.go:216-217 passes only path and RequestedAgent into startFreshSlot. That function independently resolves a profile at slotrecovery.go:289 and launches after workspace readiness at lines 347-354 without comparing against the accepted resolution. Newly provisioned slots perform that comparison through couch.go:559-563 and revalidateCreatedSlot. A default/preference change after fingerprint comparison can therefore launch an unaccepted profile, and changes during readiness escape the existing drift refusal. Carry the accepted resolution through reuse and share post-readiness authority validation; extend TestManagedCreateRefusesProfileDriftDuringSetup to an archived-slot reuse fixture. ARCH-ORDER, ARCH-PURPOSE.

## Round 3 — 2026-09-25T14:27:39-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Unreadable slot-current admission remains enforced in resolveManagedStart, with TestManagedCreateRefusesUnreadableSiblingSlot.
- BR-2 — addressed — startFreshSlot rejects old.Exists for allocation reuse; replaceSlotCurrent compares existence and bytes under the store lock. TestStartCreateReuseRefusesAnOccupiedCurrent exercises the new guard; existing concurrent-mutation coverage exercises the locked comparison.
- BR-3 — addressed — ReuseNotices carries a combined typed sequence. Sorting and rendering tests cover lost :1, parked :2, and lost :10; focused tests pass.
- BR-4 — addressed — README.md now documents lowest-free allocation, checkout inheritance, and actionable parked/lost reuse guidance, matching the changed allocation and rendering paths.
- BR-5 — addressed — The active Plan now describes snapshot occupancy, StartCreate plus ReuseSlot, and typed actionable notices; dated Revisions explain the changes.
- BR-6 — not-addressed — slotrecovery.go:289 independently resolves profile B, while lines 351-352 validate current configuration against accepted profile A. If configuration returns to A during readiness, validation passes, but lines 340 and 357 still launch B. The added regression covers persistent drift during readiness, not this A-to-B-to-A sequence. ARCH-ORDER, ARCH-PURPOSE.

## Round 4 — 2026-09-25T14:36:56-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Allocation rejects unreadable slot-current observations; TestManagedCreateRefusesUnreadableSiblingSlot covers an unreadable sibling beside a reusable hole.
- BR-2 — addressed — Reuse requires physical current-file absence and retains compare-and-replace protection; occupied-current and concurrent-change tests cover the guards.
- BR-3 — addressed — Typed notices are sorted numerically across kinds; selector and rendering tests cover slots 1, 2, and 10.
- BR-4 — addressed — README.md documents lowest-free allocation, checkout inheritance, parked-work admission, and reuse actions, consistent with the implementation.
- BR-5 — addressed — The active Plan now describes StartCreate plus ReuseSlot, physical absence, and typed ordered notices; revisions record the superseded design.
- BR-6 — not-addressed — Pinned slotrecovery.go:289 independently resolves the payload used at lines 340 and 357. Revalidation at lines 351–352 cannot detect a transient profile change that returns to its accepted value. A scratch regression selecting an agent without saved argv launched [--model unaccepted] after three profile reads. Sourcing the payload from accepted.Profile and its provenance made the regression pass; that correction is uncommitted and the regression is absent from the pinned range. ARCH-ORDER, ARCH-PURPOSE.

## Round 5 — 2026-09-25T14:42:35-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — resolveManagedStart rejects unreadable slot-current inventory; TestManagedCreateRefusesUnreadableSiblingSlot covers that refusal.
- BR-2 — addressed — Reuse requires physical current-record absence, and replaceSlotCurrent retains the observed-state comparison before replacement.
- BR-3 — addressed — Typed reuse notices are sorted numerically across kinds; selector and rendering tests cover mixed ordering.
- BR-4 — addressed — README now describes lowest-free reuse, inherited checkout contents, and parked/lost reuse guidance, matching the implementation.
- BR-5 — addressed — The active Plan describes StartCreate plus ReuseSlot, ordered typed notices, and accepted-profile validation; revisions record these changes.
- BR-6 — not-addressed — slotrecovery.go:290-301 now binds the launch payload to the accepted profile, but slotstart_test.go:162 only tests persistent drift during readiness. Restoring the parent commit's profile-selection code through a temporary Go overlay leaves both the drift and archived-reuse tests passing. Add a deterministic transient-profile-change test asserting the actual launched profile and provenance, and demonstrate failure without the binding fix. ARCH-ORDER, ARCH-PURPOSE; existing family accepted-launch-authority.

## Round 6 — 2026-09-25T14:49:40-07:00 (codex) — passed

### Disposed

- BR-6 — addressed — Reuse now carries the accepted launch profile through readiness, revalidates drift before launch, and launches from the accepted profile. TestManagedCreateReuseRefusesProfileDriftDuringSetup and TestManagedCreateReuseLaunchesAcceptedTransientProfile provide regression evidence.

## Round 7 — 2026-09-25T14:55:44-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Unreadable repository and slot inventory refuses allocation; focused unreadable-sibling regression passes.
- BR-2 — addressed — Reuse requires physical current-record absence and preserves compare-and-replace protection.
- BR-3 — addressed — Typed mixed notices are numerically ordered, including multi-digit slots.
- BR-4 — addressed — README documents lowest-free reuse, inherited checkout state, and parked/lost guidance.
- BR-5 — addressed — The active issue plan and revisions match the implemented selector, reuse, notices, and authority behavior.
- BR-6 — addressed — Accepted profile authority is preserved through readiness and launch; the transient A-to-B-to-A regression passes.

### Raised

- **BR-7** [Important] `user-facing-documentation-parity` Atlas retains stale SelectNewSlot allocation semantics
  atlas/workspace-provisioning.md:69-71 still describes lowest-unused-positive allocation, contradicting SelectStartSlot, :0 reuse, and the current lowest-free rule. This is the 2nd finding in family user-facing-documentation-parity; sweep stale allocation terminology across the atlas.

## Open findings

- **BR-7** [Important] `user-facing-documentation-parity` Atlas retains stale SelectNewSlot allocation semantics
