# Boundary Review — pair#355 (milestone M2)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c6f3419a64e970d948ffe8bc9da8b1015b9ce052..044c19358307034850429c9d594c08c6654b09eb |
| command | sdlc milestone-close --issue 355 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-30T11:25:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 implements journaled family reservation and propagates subdirectory CWDs through launch and menu paths. Focused tests pass, including race tests. Shipping is blocked by storage routing that confuses nested independent repositories with their enclosing slot, plus an unbounded permanent family registry.

1. **Strengths**

   - Reservation reuses the existing lock and journal; tests cover concurrent admission and interrupted publication.
   - Launch tests verify added-slot CWDs, conflicts after parking/restart, and missing-directory refusal.
   - Inventory preserves recorded starting/working paths, and add-slot projects the selected directory into the primary checkout.
   - README and atlas updates describe the new behavior.

2. **Critical findings**

   **Checkout containment is treated as repository membership.** At [slotmigration.go:72](/Users/xianxu/workspace/pair/cmd/internal/couchcore/slotmigration.go:72), migration selects a destination using only path containment. A conversation in an independent Git repository nested inside a numbered checkout therefore moves into the outer slot’s single `thread.json`. If both repositories have conversations, enrollment instead fails with “multiple legacy current records.”

   The same assumption appears in [threadstore_location.go:131](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadstore_location.go:131), local-origin validation, and [snapshot filtering](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadstore_snapshot.go:14). Family inference correctly distinguishes nested repositories, but these consumers do not.

   **Fix:** establish one membership rule using checkout scope/common-Git identity alongside containment. Apply it across current/archive migration, preferences, routing, and snapshot filtering. Add a regression with both outer-slot and nested-repository conversations. **ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.**

3. **Important findings**

   **Permanent family reservations have neither removal nor a growth bound.** [repository_family_store.go:95](/Users/xianxu/workspace/pair/cmd/internal/couchcore/repository_family_store.go:95) appends descriptors permanently. No consumer removes them or limits admission. The plan defers removal without specifying a bounded alternative.

   **Fix:** enforce and test an explicit capacity limit with actionable refusal, or provide a deliberate removal lifecycle preserving parked-family reservations. Document the chosen policy. **ARCH-FUNERAL.**

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused family, migration, storage, inventory, and add-slot tests; family reservation/launch tests also passed under `-race`. The pinned diff passes `git diff --check`; the worktree remains clean.

   The nested-repository test covers inference only. It does not exercise storage routing, migration, or snapshot visibility. Full suites and live Zellij conformance were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — flag:** consolidate membership decisions across storage consumers.
   - **ARCH-PURE — pass:** deterministic resolution/projection helpers remain directly testable.
   - **ARCH-PURPOSE — flag:** repository identity separation stops at inference.
   - **ARCH-MOCK — pass:** changed integration tests use isolated stores, real temporary Git, and existing runtime doubles.
   - **ARCH-CONSTRAINTS — pass:** inference has record/byte limits and cancellation checks; no new unbounded fan-out found.
   - **ARCH-SECURE — flag:** directory containment cannot establish repository ownership.
   - **ARCH-ORDER — pass:** reservation serializes competing choices; journal recovery is tested.
   - **ARCH-FUNERAL — flag:** permanent family rows lack a bound or removal mechanism.

7. **Plan revision recommendations**

   Append timestamped `## Revisions` entries defining the shared repository-membership rule and enumerating its consumers, then specifying the family registry’s capacity/removal policy and regression evidence.

```findings
findings:
  - id: new
    severity: Critical
    family: checkout-membership-requires-repository-identity
    title: |
      Nested independent repositories are routed into the enclosing slot's storage
    detail: |
      slotmigration.go:72 selects destinations by containment alone; threadstore_location.go:131, local-origin validation, and threadstore_snapshot.go:14 repeat that assumption. A nested independent repository's conversation is migrated into the outer slot's single-current store, blocks enrollment when both have current records, or is filtered from global inventory. Enforce one scope/common-Git-identity membership rule across migration, preferences, routing, and snapshots; test outer and nested conversations together. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
  - id: new
    severity: Important
    family: durable-family-reservations-need-bounds
    title: |
      Permanent repository-family descriptors have no removal path or admission bound
    detail: |
      repository_family_store.go:95 appends permanent descriptors without a capacity check, and no consumer removes them. The plan defers removal without bounding retained growth. Add and test an explicit admission bound with actionable refusal, or implement a deliberate removal lifecycle that preserves parked reservations; document the policy. ARCH-FUNERAL.
```

---

## Re-review — 2026-09-30T11:42:12-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c6f3419a64e970d948ffe8bc9da8b1015b9ce052..dbf7899e8f5af6498c269ae427918b92fd7439e0 |
| command | sdlc milestone-close --issue 355 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-30T11:42:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 implements journaled family reservations, propagates starting directories, and fixes the reported storage-routing cases. BR-5 is addressed, with mutation-tested admission enforcement. BR-4 remains incomplete: the slot-session observer still treats containment as membership, allowing a nested repository’s actor to block outer-slot operations.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Storage consumers now share repository-aware membership, but cmd/internal/couchcore/slotsessions.go:307-313 still admits actors by containment alone. A nested independent actor reaches add(), which rejects its foreign scope and aborts outer-slot open/fresh. A scratch regression fails on HEAD and passes when foreign-scope actors are excluded. Complete the checkout-membership-requires-repository-identity sweep through hosted-session observation. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
  - id: BR-5
    disposition: addressed
    note: |
      repository_family_store.go:67 enforces the 4096-family admission bound under the existing lock; existing families remain usable. Capacity, concurrent-final-entry, and persisted-overflow tests cover the policy. Removing the admission guard makes TestFamilyCapacityExistingReuseAndRefusal fail. atlas/couch.md:59 documents permanent bounded reservations.
```

**Strengths**

- Storage membership now consistently uses scope/common-Git identity across migration, preferences, local-origin validation, and inventory filtering.
- The nested-repository regression exercises current records, archives, preferences, inventory, and subsequent creation together.
- Family reservation reuses the existing journal and lock; capacity refusal preserves manifest bytes and permits existing-family reuse.
- README and atlas explain starting-directory inheritance, conflicts, missing directories, and reservation lifetime.

**Critical findings**

- **BR-4 remains open — [slotsessions.go:307](/Users/xianxu/workspace/pair/cmd/internal/couchcore/slotsessions.go:307).** A hosted actor under `<outer-slot>/independent` has a different repository scope, but its contained working directory passes the selection condition. `add()` then returns `slot native owner belongs to another scope`. This propagates through fresh admission, open/recovery, and other-owner verification in `slotrecovery.go`.

  Complete the existing family `checkout-membership-requires-repository-identity` rather than patching this predicate alone: enumerate membership decisions across storage and hosted-session observation, and apply the shared authority rule. Preserve conservative handling of genuinely same-scope owners. Add production-boundary regressions covering both in-memory and durable actor registries while outer and nested conversations coexist. **ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.**

**Important findings:** None additional.

**Minor findings:** None.

**Test coverage notes**

- Passed focused core family, membership, migration, observer, nested-storage, and launch tests; full `couchtty` and `couchcmd` suites; and pinned-range `git diff --check`.
- Scratch overlay test reproduced BR-4; an isolated predicate correction made it pass.
- Removing BR-5’s admission guard made its existing regression fail.
- The complete core suite and build were not rerun during this review. Repository files remain unchanged.

**Architectural notes**

- **ARCH-DRY — flag:** observer membership bypasses the shared rule.
- **ARCH-PURE — pass:** family resolution/projection remain deterministic, with physical validation and persistence separated.
- **ARCH-PURPOSE — flag:** independent nested repositories still interfere with outer-slot actions.
- **ARCH-MOCK — pass:** reviewed tests use isolated Git fixtures and existing stateful runtime/storage seams.
- **ARCH-CONSTRAINTS — pass:** inference and admission have explicit bounds; no new external probing on keystroke paths.
- **ARCH-SECURE — flag:** containment still substitutes for repository provenance in actor selection.
- **ARCH-ORDER — pass:** reservation uses existing serialized journal publication and concurrent-admission tests.
- **ARCH-FUNERAL — pass:** permanent descriptors now have a tested writer-side admission bound.

**Plan revision recommendation**

Append a `## Revisions` entry extending BR-4’s membership sweep to hosted actor observation and its open/fresh/recovery callers. Record the nested-actor regression and update the affected task’s verification evidence before re-running this boundary.

---

## Re-review — 2026-09-30T11:51:04-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c6f3419a64e970d948ffe8bc9da8b1015b9ce052..972675c789af92f2a159a32b21c5dd92bb5ba7d5 |
| command | sdlc milestone-close --issue 355 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-30T11:51:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The nested-repository actor correction works, but BR-4’s membership rule still receives fabricated repository identity from storage discovery. A real repository using `git init --separate-git-dir` can successfully migrate a conversation into slot storage and then fail to read it. The checkout remained unchanged.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Nested storage and hosted-actor regressions pass; removing actor membership checking makes all four open/fresh registry cases fail. However, threadstore_location.go:37 and :63 reconstruct slots through conventionalSlot, which assumes RepoIdentity is primary/.git. The new RecordCheckoutMembership check rejects records carrying the actual separate Git directory. A temporary real-Git regression successfully enrolled a record, then GetThread failed with “slot record identity does not match its host checkout”. This repeats family checkout-membership-requires-repository-identity: derive identity from verified or retained authority throughout discovery, routing, validation, and inventory instead of treating conventional paths as identity.
  - id: BR-5
    disposition: addressed
    note: |
      The 4096-family admission bound remains enforced; focused capacity, concurrent admission, existing-family reuse, and persisted-overflow tests passed.
```

1. **Strengths**

   - Shared membership checks now distinguish nested repositories across migration, preferences, snapshots, and hosted actors.
   - The actor regression exercises both in-memory and durable registries through actual open/fresh operations. The mutation check confirms it detects the removed fix.
   - Family reservation uses the existing journal; tests cover interrupted publication, conflicting admissions, and retention after archive.
   - README and atlas explain starting-directory inheritance, conflicts, and missing-directory recovery.

2. **Critical findings**

   **BR-4 remains open — reconstructed storage identity rejects valid conversations.** At `threadstore_location.go:37`, discovered backends inherit `conventionalSlot`’s assumed `<primary>/.git` identity; line 63 makes the same assumption for routing. `repository_family.go:119` now compares this against persisted incarnation identity, and `threadstore_layout.go:65` rejects mismatches.

   Reproduction used a real separate Git directory and numbered worktree, supplied the correct catalog identity, created a conversation, and successfully enrolled it. Subsequent `GetThread` failed with a metadata-recovery error.

   **This is a repeat in family `checkout-membership-requires-repository-identity`.** Fix the rule across all reconstructed-backend consumers: conventional paths establish location, while verified or retained common-directory identity establishes membership. Add regression coverage for readback, preferences, and inventory after enrollment and restart.

3. **Important findings**

   None separate from BR-4.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused family, migration, nested-repository, actor, and local-storage tests passed: **94.768s**.
   - Focused menu tests passed.
   - Containment-only actor mutation failed all four intended regression cases.
   - Additional separate-Git-directory regression failed against HEAD.
   - Pinned-range whitespace check passed. Full-suite and live conformance checks were not rerun.

6. **Architectural notes**

   - **ARCH-DRY — flag:** shared membership logic still consumes independently reconstructed identity.
   - **ARCH-PURE — pass:** family conflict resolution and lexical projection remain directly testable without IO.
   - **ARCH-PURPOSE — flag:** repository identity propagation remains incomplete.
   - **ARCH-MOCK — pass:** integration tests use temporary Git repositories and existing stateful runtime seams; extend their layout coverage.
   - **ARCH-CONSTRAINTS — pass:** inference and family admission have explicit limits.
   - **ARCH-SECURE — flag:** a conventional filesystem location is substituted for authoritative common-directory identity.
   - **ARCH-ORDER — pass:** reservation precedes provisioning; journal interruption and concurrent admission are exercised.
   - **ARCH-FUNERAL — pass:** permanent family reservations have an enforced admission bound.

7. **Plan revision recommendations**

   Append a `## Revisions` entry requiring authoritative common-directory identity across backend reconstruction, routing, and inventory, with separate-Git-directory enrollment/restart regressions. Preserve the existing nested-repository isolation tests.

---

## Re-review — 2026-09-30T12:09:30-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | c6f3419a64e970d948ffe8bc9da8b1015b9ce052..879a8573786e843e25a500e58e1b345982c5d7c9 |
| command | sdlc milestone-close --issue 355 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-30T12:09:30-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M2 implements persistent family admission and relative-directory propagation across launch, storage, inventory, and menu paths. BR-4 is addressed, including nested hosted actors and separate-Git-directory storage reconstruction. No blocking findings remain.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      Shared checkout membership covers migration, routing, preferences, local-origin validation, inventory, and hosted actors. Enrollment retains verified common-Git identity across reconstruction and restart. Focused regressions pass; restoring the primary/.git assumption through a temporary Go overlay makes TestSeparateGitDirectoryEnrollmentPreservesStorageAuthority fail at record readback.
  - id: BR-5
    disposition: addressed
    note: |
      Prior disposition retained. The 4096-family admission bound, existing-family reuse, and persisted-overflow checks remain covered.
```

1. **Strengths**
   - One membership rule serves storage and runtime observers: `repository_family.go:96`, `slotsessions.go:314`.
   - Enrollment journals identity backfill independently of family-directory reservation: `slotmigration.go:92`.
   - Real-Git regressions cover nested repositories, separate Git directories, missing directories, and symlink escapes.
   - README and atlas describe the new behavior and compatibility policy.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Focused family/membership/storage suite passed: 39.668s.
   - Hosted-actor/enrollment/launch suite passed: 31.075s.
   - Menu tests passed: 0.627s.
   - Targeted race tests passed: 4.600s.
   - Mutation check failed as expected; pinned-range whitespace check passed.
   - Full suites and live Zellij conformance were not rerun during this review. Checkout remains unchanged.

6. **Architecture**
   - **ARCH-DRY — pass:** shared membership and projection helpers.
   - **ARCH-PURE — pass:** deterministic family resolution separated from persistence and physical-path validation.
   - **ARCH-PURPOSE — pass:** consumer sweep includes persisted records, preferences, hosted actors, and reconstructed backends.
   - **ARCH-MOCK — pass:** existing injected boundaries, stateful doubles, and temporary Git repositories exercise integration.
   - **ARCH-CONSTRAINTS — pass:** bounded inference and admission; no new passive inventory subprocess probes.
   - **ARCH-SECURE — pass:** identity provenance, malformed metadata, and physical containment are checked.
   - **ARCH-ORDER — pass:** serialized reservation, journal recovery, stale-preview revalidation, and interrupted backfill have coverage.
   - **ARCH-FUNERAL — pass:** permanent family reservations have an explicit admission bound.

7. **Plan revisions:** None required; appended revisions describe the implemented corrections.
