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

## Open findings

- **BR-1** [Critical] `notice-owner-loss-fallback` (kind: defect) Losing the owning hub can silently retire undelivered diagnostics
- **BR-2** [Important] `user-docs-behavior-parity` (kind: defect) README still promises the previous bell contents
- **BR-3** [Minor] `shared-analysis-single-source` (kind: defect) Producer inventory analysis is duplicated across packages
