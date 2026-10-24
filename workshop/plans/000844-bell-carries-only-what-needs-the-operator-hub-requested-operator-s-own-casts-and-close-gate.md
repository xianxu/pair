---
gate: boundary-review
issue: 844
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4b2c1112-b94e-4723-84b7-9a8d3e6e5170
            repository: github.com/xianxu/pair
            issue_id: "000844"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/159f9e002bf86c09
            reviewed_head: 93261df696424b545731f1161d5a25f124850155
            slot: pair:1
            requested_at: "2026-10-24T18:50:14.665211Z"
          phase: finished
          pid: 48342
          started_at: "2026-10-24T18:50:14.824197Z"
          finished_at: "2026-10-24T18:52:51.293993Z"
          session_id: 01a16f3f-cf73-7621-9284-a57dc355bada
          outcome: returned
      "n": 1
      timestamp: "2026-10-24T11:52:52-07:00"
      dispatched: "2026-10-24T11:49:42-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Losing the owning hub can silently retire undelivered diagnostics
          detail: 'cmd/internal/couchcmd/notice_delivery.go:239 treats every epoch difference as condition clearance. With four admissions occupied and a fifth notice pending under an enabled owner with a nonzero epoch, disconnecting that owner makes resolveNoticeOwner return Unavailable with epoch zero (notice_delivery_backend.go:106); reconcile deletes the fifth notice before RouteNotice can emit its required bell fallback. Notices already marked Sending have the same loss when DeliverNotice returns Refused for an unavailable owner and the reducer removes them at couchcore/notice_delivery.go:179. ARCH-ORDER / ARCH-PURPOSE: distinguish proven owner absence from automation invalidation, and route confirmed unadmitted occurrences through the unavailable-owner policy without resending uncertain admissions. Add deterministic tests for both orderings.'
          family: notice-owner-loss-fallback
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: README still promises the previous bell contents
          detail: README.md:1265 and README.md:1533 say actionable asynchronous warnings, blocked prompts, and judged questions appear in the bell without the new owning-hub qualification. README.md is unchanged in this range, although the implementation and atlas now route these events to connected owners and suppress fallback for automation-OFF owners. Update these user-facing descriptions to explain operator-owned exceptions, owning-hub routing, and unavailable-owner fallback.
          family: user-docs-behavior-parity
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Producer inventory analysis is duplicated across packages
          detail: 'cmd/internal/couchcmd/notice_class_inventory_test.go:44 and cmd/internal/couchtty/notice_class_inventory_test.go:44 contain near-identical AST walkers, receiver recognition, and rejection fixtures. ARCH-DRY: extract the common analysis into a shared test helper, passing package-specific audited projections and forwarders as configuration.'
          family: shared-analysis-single-source
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 3146359b-b465-48ce-a838-aba546717993
            repository: github.com/xianxu/pair
            issue_id: "000844"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/091905e0529f6c0e
            reviewed_head: 0d4186bad3b010c4985057fd932cc61f2e056f56
            slot: pair:1
            requested_at: "2026-10-24T19:02:06.902029Z"
          phase: finished
          pid: 72536
          started_at: "2026-10-24T19:02:07.00413Z"
          finished_at: "2026-10-24T19:05:57.039641Z"
          session_id: 01a16f4a-acce-77b2-8017-7f7d68907ce3
          outcome: returned
      "n": 2
      timestamp: "2026-10-24T12:06:02-07:00"
      dispatched: "2026-10-24T12:02:04-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Original disconnect orderings are repaired; restoring pre-fix production code through a temporary Go overlay makes their regressions fail. The same clearance rule remains incomplete at cmd/internal/couchcmd/notice_delivery.go:247: epochs are scoped per hub root, but reconciliation compares counters across different owners. A deterministic real-service fixture holding four admissions, queuing a fifth notice under /repo1 at epoch 2, then moving its source to /repo0 at epoch 0 deletes the notice. Disconnecting /repo0 after the move also produces zero fallback projections. Preserve scoped generation provenance and distinguish owner handoff from automation revocation across pending and confirmed-refused paths.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md:1264-1268 and :1536-1543 now describe owning-hub routing, operator-owned exceptions, unavailable-owner fallback and connected-OFF suppression/no replay. These passages match couchcore.RouteNotice and the condition projections in attention.go and message_service.go.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Both package inventories now delegate AST walking, receiver recognition and rejection fixtures to cmd/internal/testnotice/inventory.go, passing package-specific Projection and Forward configuration. The shared helper tests pass.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e3613c05-525b-451c-b562-ab804901230c
            repository: github.com/xianxu/pair
            issue_id: "000844"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: codex
            root: /Volumes/TestScratch/review-checkouts/474ca958dbc663fe
            reviewed_head: d7a07ecba6348d1d1a9d9de3c3704c45926c19c6
            slot: pair:1
            requested_at: "2026-10-24T19:20:34.7575Z"
          phase: finished
          pid: 87997
          started_at: "2026-10-24T19:20:34.930069Z"
          finished_at: "2026-10-24T19:26:30.075655Z"
          session_id: 01a16f5b-968f-7643-ad9e-c04f0a48b495
          outcome: returned
      "n": 3
      timestamp: "2026-10-24T12:26:31-07:00"
      dispatched: "2026-10-24T12:20:31-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Owner generation is validated within its retained root. Pending owner loss and confirmed refusal receive unavailable-owner fallback; unknown readiness preserves the obligation, while admitted/uncertain sends never resend. Production-path regressions pass, and mutations removing scoped revocation or witness retention fail.
          round: 3
        - id: BR-2
          disposition: addressed
          note: README.md:1264 and :1536 replace unconditional warning/prompt eligibility with owning-hub routing, operator exceptions, unavailable-owner fallback, and connected-OFF suppression, consistent with couchcore/notice_route.go:73.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Both package inventories use cmd/internal/testnotice/inventory.go:21 for analysis and its shared rejection fixtures, retaining package-specific projection and forwarding declarations.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: d7a07ecba6348d1d1a9d9de3c3704c45926c19c6
      blocked: false
---

# Gate ledger — pair#844 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-24T11:52:52-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `notice-owner-loss-fallback` (kind: defect) Losing the owning hub can silently retire undelivered diagnostics
  cmd/internal/couchcmd/notice_delivery.go:239 treats every epoch difference as condition clearance. With four admissions occupied and a fifth notice pending under an enabled owner with a nonzero epoch, disconnecting that owner makes resolveNoticeOwner return Unavailable with epoch zero (notice_delivery_backend.go:106); reconcile deletes the fifth notice before RouteNotice can emit its required bell fallback. Notices already marked Sending have the same loss when DeliverNotice returns Refused for an unavailable owner and the reducer removes them at couchcore/notice_delivery.go:179. ARCH-ORDER / ARCH-PURPOSE: distinguish proven owner absence from automation invalidation, and route confirmed unadmitted occurrences through the unavailable-owner policy without resending uncertain admissions. Add deterministic tests for both orderings.
- **BR-2** [Important] `user-docs-behavior-parity` (kind: defect) README still promises the previous bell contents
  README.md:1265 and README.md:1533 say actionable asynchronous warnings, blocked prompts, and judged questions appear in the bell without the new owning-hub qualification. README.md is unchanged in this range, although the implementation and atlas now route these events to connected owners and suppress fallback for automation-OFF owners. Update these user-facing descriptions to explain operator-owned exceptions, owning-hub routing, and unavailable-owner fallback.
- **BR-3** [Minor] `shared-analysis-single-source` (kind: defect) Producer inventory analysis is duplicated across packages
  cmd/internal/couchcmd/notice_class_inventory_test.go:44 and cmd/internal/couchtty/notice_class_inventory_test.go:44 contain near-identical AST walkers, receiver recognition, and rejection fixtures. ARCH-DRY: extract the common analysis into a shared test helper, passing package-specific audited projections and forwarders as configuration.

## Round 2 — 2026-10-24T12:06:02-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — Original disconnect orderings are repaired; restoring pre-fix production code through a temporary Go overlay makes their regressions fail. The same clearance rule remains incomplete at cmd/internal/couchcmd/notice_delivery.go:247: epochs are scoped per hub root, but reconciliation compares counters across different owners. A deterministic real-service fixture holding four admissions, queuing a fifth notice under /repo1 at epoch 2, then moving its source to /repo0 at epoch 0 deletes the notice. Disconnecting /repo0 after the move also produces zero fallback projections. Preserve scoped generation provenance and distinguish owner handoff from automation revocation across pending and confirmed-refused paths.
- BR-2 — addressed — README.md:1264-1268 and :1536-1543 now describe owning-hub routing, operator-owned exceptions, unavailable-owner fallback and connected-OFF suppression/no replay. These passages match couchcore.RouteNotice and the condition projections in attention.go and message_service.go.
- BR-3 — addressed — Both package inventories now delegate AST walking, receiver recognition and rejection fixtures to cmd/internal/testnotice/inventory.go, passing package-specific Projection and Forward configuration. The shared helper tests pass.

## Round 3 — 2026-10-24T12:26:31-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Owner generation is validated within its retained root. Pending owner loss and confirmed refusal receive unavailable-owner fallback; unknown readiness preserves the obligation, while admitted/uncertain sends never resend. Production-path regressions pass, and mutations removing scoped revocation or witness retention fail.
- BR-2 — addressed — README.md:1264 and :1536 replace unconditional warning/prompt eligibility with owning-hub routing, operator exceptions, unavailable-owner fallback, and connected-OFF suppression, consistent with couchcore/notice_route.go:73.
- BR-3 — addressed — Both package inventories use cmd/internal/testnotice/inventory.go:21 for analysis and its shared rejection fixtures, retaining package-specific projection and forwarding declarations.

## Open findings

(none — every finding has been disposed)
