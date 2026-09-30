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
