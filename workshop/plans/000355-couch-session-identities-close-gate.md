---
gate: boundary-review
issue: 355
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-30T10:37:08-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Cold launch, registration, and cleanup do not consistently use the terminal incarnation
          detail: launch_existing.go:463 allocates a new M for an attached-live conversation, while createflow.go:427 filters out its existing terminal, permitting a duplicate agent. Registration and cleanup still resolve the old address index; require proven absence before creation and use the proposed binding for registration and teardown, with composed regressions (ARCH-ORDER, ARCH-PURPOSE).
          family: terminal-binding-authority
          round: 1
        - id: BR-2
          severity: Critical
          title: Warm attach revalidates ownership before blocking preparation rather than at handoff
          detail: createflow.go:397 checks the generation before lifecycle.go:43-110 performs retention and cmux preparation. Replacement during that interval reaches AttachSession unchecked; retain the original proof and revalidate immediately before attachment, testing replacement during preparation (ARCH-ORDER, ARCH-SECURE).
          family: owner-proof-at-effect-boundary
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-30T10:59:00-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Cold admission requires absence; registration and cleanup select the pending/current terminal binding. Passing terminal_incarnation_test.go regressions cover attached-terminal refusal, stale registration, wrong-terminal cleanup, and interrupted recovery; launcher tests cover late terminal appearance.
          round: 2
        - id: BR-2
          disposition: addressed
          note: lifecycle.go:134-145 revalidates the original ownership proof immediately before AttachSession, after blocking preparation. TestCouchSessionWarmGenerationRevalidatedAtAttachEffect injects replacement during retention/cmux through both entrypoints and asserts refusal and poller cleanup.
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: Unicode-only repository names now prevent conversation creation
          detail: cmd/internal/couchidentity/identity.go:41-60 discards every non-ASCII character and rejects the resulting empty token. Both couch.go:488 and slotrecovery.go:320 pass the repository basename, so supported names such as 项目 now fail allocation. Use a deterministic safe fallback when normalization yields nothing; C/N already provide uniqueness. Add pure formatter tests and composed new/fresh launch regressions for Unicode-only and punctuation-only names. ARCH-PURPOSE.
          family: descriptive-label-must-not-gate-identity
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-30T11:06:56-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: The shared formatter supplies a safe fallback and bounds descriptive tokens. Pure formatter and composed new/fresh launch tests pass at HEAD and fail with the previous formatter substituted through a temporary Go overlay. Unicode-only, punctuation-only, and long repository names are exercised.
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 4
      timestamp: "2026-09-30T11:25:49-07:00"
      agent: codex
      findings:
        - id: BR-4
          severity: Critical
          title: Nested independent repositories are routed into the enclosing slot's storage
          detail: slotmigration.go:72 selects destinations by containment alone; threadstore_location.go:131, local-origin validation, and threadstore_snapshot.go:14 repeat that assumption. A nested independent repository's conversation is migrated into the outer slot's single-current store, blocks enrollment when both have current records, or is filtered from global inventory. Enforce one scope/common-Git-identity membership rule across migration, preferences, routing, and snapshots; test outer and nested conversations together. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
          family: checkout-membership-requires-repository-identity
          round: 4
        - id: BR-5
          severity: Important
          title: Permanent repository-family descriptors have no removal path or admission bound
          detail: repository_family_store.go:95 appends permanent descriptors without a capacity check, and no consumer removes them. The plan defers removal without bounding retained growth. Add and test an explicit admission bound with actionable refusal, or implement a deliberate removal lifecycle that preserves parked reservations; document the policy. ARCH-FUNERAL.
          family: durable-family-reservations-need-bounds
          round: 4
      boundary: M2
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-09-30T11:42:12-07:00"
      agent: codex
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: Storage consumers now share repository-aware membership, but cmd/internal/couchcore/slotsessions.go:307-313 still admits actors by containment alone. A nested independent actor reaches add(), which rejects its foreign scope and aborts outer-slot open/fresh. A scratch regression fails on HEAD and passes when foreign-scope actors are excluded. Complete the checkout-membership-requires-repository-identity sweep through hosted-session observation. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
          round: 5
        - id: BR-5
          disposition: addressed
          note: repository_family_store.go:67 enforces the 4096-family admission bound under the existing lock; existing families remain usable. Capacity, concurrent-final-entry, and persisted-overflow tests cover the policy. Removing the admission guard makes TestFamilyCapacityExistingReuseAndRefusal fail. atlas/couch.md:59 documents permanent bounded reservations.
          round: 5
      boundary: M2
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#355 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-30T10:37:08-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `terminal-binding-authority` Cold launch, registration, and cleanup do not consistently use the terminal incarnation
  launch_existing.go:463 allocates a new M for an attached-live conversation, while createflow.go:427 filters out its existing terminal, permitting a duplicate agent. Registration and cleanup still resolve the old address index; require proven absence before creation and use the proposed binding for registration and teardown, with composed regressions (ARCH-ORDER, ARCH-PURPOSE).
- **BR-2** [Critical] `owner-proof-at-effect-boundary` Warm attach revalidates ownership before blocking preparation rather than at handoff
  createflow.go:397 checks the generation before lifecycle.go:43-110 performs retention and cmux preparation. Replacement during that interval reaches AttachSession unchecked; retain the original proof and revalidate immediately before attachment, testing replacement during preparation (ARCH-ORDER, ARCH-SECURE).

## Round 2 — 2026-09-30T10:59:00-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Cold admission requires absence; registration and cleanup select the pending/current terminal binding. Passing terminal_incarnation_test.go regressions cover attached-terminal refusal, stale registration, wrong-terminal cleanup, and interrupted recovery; launcher tests cover late terminal appearance.
- BR-2 — addressed — lifecycle.go:134-145 revalidates the original ownership proof immediately before AttachSession, after blocking preparation. TestCouchSessionWarmGenerationRevalidatedAtAttachEffect injects replacement during retention/cmux through both entrypoints and asserts refusal and poller cleanup.

### Raised

- **BR-3** [Critical] `descriptive-label-must-not-gate-identity` Unicode-only repository names now prevent conversation creation
  cmd/internal/couchidentity/identity.go:41-60 discards every non-ASCII character and rejects the resulting empty token. Both couch.go:488 and slotrecovery.go:320 pass the repository basename, so supported names such as 项目 now fail allocation. Use a deterministic safe fallback when normalization yields nothing; C/N already provide uniqueness. Add pure formatter tests and composed new/fresh launch regressions for Unicode-only and punctuation-only names. ARCH-PURPOSE.

## Round 3 — 2026-09-30T11:06:56-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — The shared formatter supplies a safe fallback and bounds descriptive tokens. Pure formatter and composed new/fresh launch tests pass at HEAD and fail with the previous formatter substituted through a temporary Go overlay. Unicode-only, punctuation-only, and long repository names are exercised.

## Round 4 — 2026-09-30T11:25:49-07:00 (codex) — BLOCKED

### Raised

- **BR-4** [Critical] `checkout-membership-requires-repository-identity` Nested independent repositories are routed into the enclosing slot's storage
  slotmigration.go:72 selects destinations by containment alone; threadstore_location.go:131, local-origin validation, and threadstore_snapshot.go:14 repeat that assumption. A nested independent repository's conversation is migrated into the outer slot's single-current store, blocks enrollment when both have current records, or is filtered from global inventory. Enforce one scope/common-Git-identity membership rule across migration, preferences, routing, and snapshots; test outer and nested conversations together. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
- **BR-5** [Important] `durable-family-reservations-need-bounds` Permanent repository-family descriptors have no removal path or admission bound
  repository_family_store.go:95 appends permanent descriptors without a capacity check, and no consumer removes them. The plan defers removal without bounding retained growth. Add and test an explicit admission bound with actionable refusal, or implement a deliberate removal lifecycle that preserves parked reservations; document the policy. ARCH-FUNERAL.

## Round 5 — 2026-09-30T11:42:12-07:00 (codex) — BLOCKED

### Disposed

- BR-4 — not-addressed — Storage consumers now share repository-aware membership, but cmd/internal/couchcore/slotsessions.go:307-313 still admits actors by containment alone. A nested independent actor reaches add(), which rejects its foreign scope and aborts outer-slot open/fresh. A scratch regression fails on HEAD and passes when foreign-scope actors are excluded. Complete the checkout-membership-requires-repository-identity sweep through hosted-session observation. ARCH-DRY, ARCH-PURPOSE, ARCH-SECURE.
- BR-5 — addressed — repository_family_store.go:67 enforces the 4096-family admission bound under the existing lock; existing families remain usable. Capacity, concurrent-final-entry, and persisted-overflow tests cover the policy. Removing the admission guard makes TestFamilyCapacityExistingReuseAndRefusal fail. atlas/couch.md:59 documents permanent bounded reservations.

## Open findings

- **BR-4** [Critical] `checkout-membership-requires-repository-identity` Nested independent repositories are routed into the enclosing slot's storage
