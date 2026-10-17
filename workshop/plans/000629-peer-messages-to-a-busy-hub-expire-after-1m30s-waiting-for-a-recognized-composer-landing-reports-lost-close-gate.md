---
gate: boundary-review
issue: 629
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 294c4d88-035e-4b39-9236-eb6f9c6ce0cd
            repository: github.com/xianxu/pair
            issue_id: "000629"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/7c096cf9632e86dd
            reviewed_head: 80be425bef48935ed3af1151c4eb49280aef1f54
            slot: pair:3
            requested_at: "2026-10-17T05:09:51.014808Z"
          phase: finished
          pid: 2165
          started_at: "2026-10-17T05:09:51.2458Z"
          finished_at: "2026-10-17T05:12:16.519808Z"
          session_id: 01a14844-3417-7460-8774-2dc1e678971f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 80be425bef48935ed3af1151c4eb49280aef1f54
        tracker_head: 6d629676cea52669c0d55425d764db1b8ef04894
        branch_point: 628ab0cadba9a7027130383d886fff443708c948
        since: "2026-10-16T18:58:38-07:00"
        until: "2026-10-16T22:09:11-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T18:58:38-07:00"
              end: "2026-10-16T22:09:11-07:00"
              began: "2026-10-16T18:58:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 9f42654d1a73cd05befdfb9717168c11a60ab5a3446cb8193043e3263b49dbd2
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 9721162
              windows:
                - start: "2026-10-16T18:58:38-07:00"
                  end: "2026-10-16T22:09:11-07:00"
            - harness: codex
              id: 01a14799-37a8-7ce3-8a22-12ab22fa6d3e
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: ce89c5022995f73a10bfd2fe3c7ee204d9f29094e36bf0b83dca8fb2067a076a
              started: "2026-10-17T02:03:06.129Z"
              spawned: true
              bytes: 518805
              windows:
                - start: "2026-10-17T02:03:06.129Z"
                  end: "2026-10-16T22:09:11-07:00"
      "n": 1
      timestamp: "2026-10-16T22:12:18-07:00"
      dispatched: "2026-10-16T22:09:48-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Sender disconnect classifies handed commits as cancelled
          detail: 'cmd/internal/couchmessage/model.go:409 now preserves Queued progress, but broker.go:269-273 decides cancellation safety from the displayed receipt status. After a handed commit reports Queued, sender disconnect records Cancelled even though the wrapper may have pasted since that observation; the terminal receipt cannot subsequently be corrected. Family enumeration: both Queued and Held interim receipts take this unsafe path. A disposable regression using newBusyReceiverFixture reproduced Cancelled for both. Classify uncertainty from delivery ownership rather than the last displayed status, and test both interim states plus genuinely uncommitted cancellation. This violates the plan''s uncertainty contract and ARCH-PURPOSE.'
          family: cancellation-preserves-input-uncertainty
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Regression tests do not detect restoration of either outer timeout
          detail: The affected owners are Broker.deliver at cmd/internal/couchmessage/broker.go:955 and RemoteEndpoint.Deliver at endpoint.go:217. Restoring both previous outer deadlines in a disposable copy still passed all TestBusyReceiver and TestRemoteEndpoint cases (7.407s). The broker fixture's clock jump does not advance an already-created real context timer, and the endpoint exchange completes far before its former horizon. Add deterministic coverage that fails when either owner independently regains a pre-paste timeout, while preserving parent cancellation and bounded commit/RPC behavior.
          family: expiry-owner-regression-coverage
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: f8951c81-eb97-43ba-9040-5a3da343aac9
            repository: github.com/xianxu/pair
            issue_id: "000629"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/2efff9e29ed44569
            reviewed_head: acce298254c4f151c1c355f0dccd8a6084eef0dd
            slot: pair:3
            requested_at: "2026-10-17T06:10:31.672616Z"
          phase: finished
          pid: 77655
          started_at: "2026-10-17T06:10:31.77058Z"
          finished_at: "2026-10-17T06:12:01.322797Z"
          session_id: 01a1487b-bfcc-7473-a0a0-8726fcca1273
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: acce298254c4f151c1c355f0dccd8a6084eef0dd
        tracker_head: 8e89665c5553549d94ecefff3b4344b55f647ebf
        branch_point: a991ba8fb2b1e30b199013aebdcdab9d237d7ed9
        since: "2026-10-16T18:58:38-07:00"
        until: "2026-10-16T23:10:14-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T18:58:38-07:00"
              end: "2026-10-16T23:10:14-07:00"
              began: "2026-10-16T18:58:38-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot3/pair
        sources:
            - harness: codex
              id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              digest: 5ae7eb18e26c84ac01e7be1373f47cc3f3d5dce67745c2ee66d326b40ab36f95
              started: "2026-10-16T22:02:33.467Z"
              spawned: false
              bytes: 11126136
              windows:
                - start: "2026-10-16T18:58:38-07:00"
                  end: "2026-10-16T23:10:14-07:00"
            - harness: codex
              id: 01a14799-37a8-7ce3-8a22-12ab22fa6d3e
              cwd: /Users/xianxu/workspace/worktree/pair-slot3/pair
              parent_id: 01a146bc-fcee-7450-93c9-096b6d3071a3
              digest: ce89c5022995f73a10bfd2fe3c7ee204d9f29094e36bf0b83dca8fb2067a076a
              started: "2026-10-17T02:03:06.129Z"
              spawned: true
              bytes: 518805
              windows:
                - start: "2026-10-17T02:03:06.129Z"
                  end: "2026-10-16T23:10:14-07:00"
        excluded_brain_sessions: 2
        excluded_brain_minutes: 0.7423500000000001
        cuts:
            - dispatched: "2026-10-16T22:09:48-07:00"
              finalized: "2026-10-16T22:12:18-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T23:12:02-07:00"
      dispatched: "2026-10-16T23:10:29-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: broker.go:266 classifies sender disconnect from active delivery ownership. TestBusyReceiverSenderDisconnectPreservesHandoffUncertainty covers handed queued/held receipts and uncommitted cancellation controls. Restoring receipt-based classification in a disposable pinned-head copy failed both handed cases.
          round: 2
        - id: BR-2
          disposition: addressed
          note: delivery_lifetime_test.go:31 and :70 exercise broker and endpoint context lifetimes at their IO boundaries, including receiver cancellation, parent cancellation and bounded commit admission. Independently restoring each former outer timeout failed its corresponding regression.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: acce298254c4f151c1c355f0dccd8a6084eef0dd
      blocked: false
---

# Gate ledger — pair#629 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T22:12:18-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `cancellation-preserves-input-uncertainty` (kind: defect) Sender disconnect classifies handed commits as cancelled
  cmd/internal/couchmessage/model.go:409 now preserves Queued progress, but broker.go:269-273 decides cancellation safety from the displayed receipt status. After a handed commit reports Queued, sender disconnect records Cancelled even though the wrapper may have pasted since that observation; the terminal receipt cannot subsequently be corrected. Family enumeration: both Queued and Held interim receipts take this unsafe path. A disposable regression using newBusyReceiverFixture reproduced Cancelled for both. Classify uncertainty from delivery ownership rather than the last displayed status, and test both interim states plus genuinely uncommitted cancellation. This violates the plan's uncertainty contract and ARCH-PURPOSE.
- **BR-2** [Important] `expiry-owner-regression-coverage` (kind: defect) Regression tests do not detect restoration of either outer timeout
  The affected owners are Broker.deliver at cmd/internal/couchmessage/broker.go:955 and RemoteEndpoint.Deliver at endpoint.go:217. Restoring both previous outer deadlines in a disposable copy still passed all TestBusyReceiver and TestRemoteEndpoint cases (7.407s). The broker fixture's clock jump does not advance an already-created real context timer, and the endpoint exchange completes far before its former horizon. Add deterministic coverage that fails when either owner independently regains a pre-paste timeout, while preserving parent cancellation and bounded commit/RPC behavior.

## Round 2 — 2026-10-16T23:12:02-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — broker.go:266 classifies sender disconnect from active delivery ownership. TestBusyReceiverSenderDisconnectPreservesHandoffUncertainty covers handed queued/held receipts and uncommitted cancellation controls. Restoring receipt-based classification in a disposable pinned-head copy failed both handed cases.
- BR-2 — addressed — delivery_lifetime_test.go:31 and :70 exercise broker and endpoint context lifetimes at their IO boundaries, including receiver cancellation, parent cancellation and bounded commit admission. Independently restoring each former outer timeout failed its corresponding regression.

## Open findings

(none — every finding has been disposed)
