---
gate: boundary-review
issue: 636
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 8f8d6bdd-c48a-4a55-a32f-f98c30469118
            repository: github.com/xianxu/pair
            issue_id: "000636"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/1939dabf624321ab
            reviewed_head: 9ca9253b21b9a5efc7a6f6402b82dcf348ea1ea4
            slot: pair:6
            requested_at: "2026-10-18T15:45:27.178775Z"
          phase: finished
          pid: 39810
          started_at: "2026-10-18T15:45:27.380778Z"
          finished_at: "2026-10-18T15:47:05.17191Z"
          session_id: 01a14fb0-790b-73a1-8508-ed46bd2695f5
          outcome: returned
      "n": 1
      timestamp: "2026-10-18T08:47:06-07:00"
      dispatched: "2026-10-18T08:45:24-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Query capture records the installation deadline instead of the query deadline
          detail: cmd/internal/couchcore/recovery_query_capture_test.go:234 closes over the context supplied to Context; lines 250–251 then report that context's deadline for every query. Recovery creates shorter observation contexts (recovery_execute.go:105,133), and the native adapter also bounds queries to five seconds. The affected consumers are warm/checkpoint recovery (recovery_conformance_live_test.go:70), normal park (park_lifecycle_live_test.go:227), and intent-only park (:249), whose later 750ms context is likewise invisible to this closure. This violates plan task 3's call-context measurement contract and ARCH-PURPOSE. Carry the actual query caller/effective deadline in the observer event and consume that value. Add a regression installing capture on a longer-lived parent, querying through a shorter-lived child, and asserting the recorded deadline; it must fail with the current closure.
          family: observation-context-fidelity
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c4d2f279-28d6-4f52-ae92-8d6fb1f9de1a
            repository: github.com/xianxu/pair
            issue_id: "000636"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/8b67ddf6bac0d95a
            reviewed_head: 823bafe3b328613b8016edd9baebe32be5b23a38
            slot: pair:6
            requested_at: "2026-10-18T16:18:59.092567Z"
          phase: finished
          pid: 12027
          started_at: "2026-10-18T16:18:59.275562Z"
          finished_at: "2026-10-18T16:22:11.024289Z"
          session_id: 01a14fcf-2c46-7623-9d57-a65ce8e2dbde
          outcome: returned
      "n": 2
      timestamp: "2026-10-18T09:22:12-07:00"
      dispatched: "2026-10-18T09:18:55-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: zellij.go:199–217 supplies actual caller and effective deadlines; recovery_query_capture_test.go:266–277 consumes them for warm/checkpoint and both park modes. TestRecoveryQueryCaptureUsesQueryChildDeadline passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child. Adapter tests also cover absent, shorter and longer caller deadlines.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 823bafe3b328613b8016edd9baebe32be5b23a38
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 70983eb1-f791-42b1-a2e1-b9202c4ff552
            repository: github.com/xianxu/pair
            issue_id: "000636"
            recipe: fix-review
            round: 3
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/fbaf4c690bbbb1c2
            reviewed_head: bf4af0d3b6ae83ca87b249e00148f7a903d28227
            slot: pair:6
            requested_at: "2026-10-18T16:50:51.056979Z"
          phase: finished
          pid: 61396
          started_at: "2026-10-18T16:50:51.152981Z"
          finished_at: "2026-10-18T16:52:15.722521Z"
          session_id: 01a14fec-5779-7fd2-9cf6-cee75bba9f64
          outcome: returned
      "n": 3
      timestamp: "2026-10-18T09:52:17-07:00"
      dispatched: "2026-10-18T09:50:49-07:00"
      agent: codex
      reviewer: codex (openai; author openai)
      dispose:
        - id: BR-1
          disposition: addressed
          note: launcher/zellij.go:199–217 supplies query caller/effective deadlines; recovery_query_capture_test.go:266–277 consumes them. The shorter-child regression passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child.
          round: 3
      recipe: fix-review
      isolation: fenced
      reviewed: bf4af0d3b6ae83ca87b249e00148f7a903d28227
      blocked: false
---

# Gate ledger — pair#636 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T08:47:06-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `observation-context-fidelity` (kind: defect) Query capture records the installation deadline instead of the query deadline
  cmd/internal/couchcore/recovery_query_capture_test.go:234 closes over the context supplied to Context; lines 250–251 then report that context's deadline for every query. Recovery creates shorter observation contexts (recovery_execute.go:105,133), and the native adapter also bounds queries to five seconds. The affected consumers are warm/checkpoint recovery (recovery_conformance_live_test.go:70), normal park (park_lifecycle_live_test.go:227), and intent-only park (:249), whose later 750ms context is likewise invisible to this closure. This violates plan task 3's call-context measurement contract and ARCH-PURPOSE. Carry the actual query caller/effective deadline in the observer event and consume that value. Add a regression installing capture on a longer-lived parent, querying through a shorter-lived child, and asserting the recorded deadline; it must fail with the current closure.

## Round 2 — 2026-10-18T09:22:12-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — zellij.go:199–217 supplies actual caller and effective deadlines; recovery_query_capture_test.go:266–277 consumes them for warm/checkpoint and both park modes. TestRecoveryQueryCaptureUsesQueryChildDeadline passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child. Adapter tests also cover absent, shorter and longer caller deadlines.

## Round 3 — 2026-10-18T09:52:17-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — launcher/zellij.go:199–217 supplies query caller/effective deadlines; recovery_query_capture_test.go:266–277 consumes them. The shorter-child regression passes at head and fails under a temporary overlay restoring installation-context capture, reporting the one-minute parent instead of the four-second child.

## Open findings

(none — every finding has been disposed)
