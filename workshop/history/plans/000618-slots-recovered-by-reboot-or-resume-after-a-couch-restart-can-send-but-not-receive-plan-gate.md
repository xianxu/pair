---
reviewer_runs:
    - version: 1
      intent:
        id: bb54b26f-dbc8-4a49-a4b7-bcbf74ce8074
        repository: github.com/xianxu/pair
        issue_id: "000618"
        recipe: estimate-quality
        round: 1
        attempt: 1
        agent: claude
        root: /Users/xianxu/workspace/worktree/pair-slot2/pair
        reviewed_head: 4739bc4f6e89394471b104ac6b70f4a54253bb6a
        slot: pair:2
        requested_at: "2026-10-16T22:51:58.363631Z"
      phase: finished
      pid: 83602
      started_at: "2026-10-16T22:51:58.974926Z"
      finished_at: "2026-10-16T22:52:32.260956Z"
      session_id: 07f507a1-f545-4353-94e5-fdb696746544
      outcome: returned
gate: plan-quality
issue: 618
id_prefix: PQ
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            id: 9e7ee92f-97ca-4e0a-ab73-49f288459272
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: plan-quality
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 48b8c1eb837015270f2d31d692323a2896103660
            slot: pair:2
            requested_at: "2026-10-16T22:45:49.028744Z"
          phase: finished
          pid: 42462
          started_at: "2026-10-16T22:45:49.400416Z"
          finished_at: "2026-10-16T22:47:35.45719Z"
          session_id: ef855715-99b9-41fa-8900-e650e2074c61
          outcome: returned
      "n": 1
      timestamp: "2026-10-16T15:47:36-07:00"
      agent: claude
      findings:
        - kind: defect
          id: PQ-1
          severity: Important
          title: A cold cache miss still runs two serial listings, about 2.1 s at the measured rate, which exceeds the 2 s AdmissionTimeout
          detail: snapshot runs list-sessions --short then --no-formatting serially (launcher/zellij.go:76-80); at 0.13-1.07 s each a single miss blows the budget, and Task 2's 1 s-per-listing test cannot pass cold. State the fix (concurrent listings, a background warm refresh, or a separate budget) and the single-flight context semantics (a leader's cancel must not fail followers).
          family: operating-envelope-arithmetic
          round: 1
        - kind: defect
          id: PQ-2
          severity: Important
          title: 'Refused receipts are stored like other receipts: same-ID retries are pinned to the refusal for 1 h and refusals fill MaxReceipts'
          detail: Send returns a retained receipt for a known ID (broker.go:532-539) and clients keep IDs through retries (protocol.go:73), so a stored refusal blocks re-admission for ReceiptLifetime (1 h). Refusals also count toward MaxReceipts=256 (broker.go:553), which becomes a capacity cliff under the load being fixed. State the lifetime and bound, the retry semantics, and whether Refused is excluded from Status.Terminal() for wrapper-reported outcomes.
          family: new-record-lifecycle-unstated
          round: 1
        - kind: defect
          id: PQ-3
          severity: Important
          title: Spec and Done-when still describe a reboot/resume registration bug the Log has disproved
          detail: Done-when requires rebooted and resumed slots to receive sends, but the plan's tests target slow-listing admission, refusal reasons and peer-message confirmation. Add a Revisions entry and restate Done-when to match the measured root cause.
          family: done-when-tracks-spec
          round: 1
        - kind: defect
          id: PQ-4
          severity: Important
          title: M2 Task 6 adds an unnamed wrapper-written file of delivered-body hashes with no path, format, parse-failure behaviour or removal
          detail: '"Where the watcher reads them" leaves the cross-process artifact undefined. Name the owner, path and format, the behaviour on truncated or older-version input (ARCH-SECURE), and its removal and growth bound (ARCH-FUNERAL).'
          family: new-record-lifecycle-unstated
          round: 1
        - kind: defect
          id: PQ-5
          severity: Minor
          title: Task 7's relaunch "longer bound" is unnamed, as is the behaviour when it is exceeded
          family: vague-plan-step
          round: 1
        - kind: defect
          id: PQ-6
          severity: Minor
          title: The --list receive state should be one supervisor request for all slots, and should name the slot when recipient resolution fails
          family: deterministic-listing-path
          round: 1
        - kind: defect
          id: PQ-7
          severity: Minor
          title: Task 1's tests are an enumerated case list; compress to one strategy line (injected clock, blocking runner, -race)
          family: test-prose-enumeration
          round: 1
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            id: 397ba842-cd6d-4dd5-b8cc-54d50f7b0249
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: plan-quality
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 4739bc4f6e89394471b104ac6b70f4a54253bb6a
            slot: pair:2
            requested_at: "2026-10-16T22:48:33.538097Z"
          phase: finished
          pid: 19848
          started_at: "2026-10-16T22:48:33.730061Z"
          finished_at: "2026-10-16T22:49:06.258538Z"
          session_id: 160fb503-f0f1-4a8f-8326-9e0cfe725840
          outcome: returned
      "n": 2
      timestamp: "2026-10-16T15:49:06-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: A background poller runs both listings concurrently; admissions read the in-memory snapshot and never run zellij.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 10-minute lifetime, a cap of 64 refusals kept outside MaxReceipts, and a resend with the refused ID is admitted fresh.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: A Revisions entry was added and the Spec and Done-when were restated to the measured load cause.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: peer_delivery records go in the existing ledger through its locked append, with its malformed-line handling and lifecycle.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: The wait is bounded at 60 s; ParkIncomplete is returned at 60 s.
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
      findings:
        - kind: defect
          id: PQ-8
          severity: Minor
          title: The plan doesn't say what an admission does before the poller's first snapshot exists
          detail: Right after the supervisor starts, no listing has completed yet. Say whether an admission then waits for the first listing within AdmissionTimeout or refuses with a distinct reason, and cover it in the Task 1 or Task 2 tests.
          family: operating-envelope-arithmetic
          round: 2
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: f6aa2adc-a087-420c-9f68-eecdde42b0a0
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: plan-quality
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 4739bc4f6e89394471b104ac6b70f4a54253bb6a
            slot: pair:2
            requested_at: "2026-10-16T22:49:26.11582Z"
          phase: finished
          pid: 47181
          started_at: "2026-10-16T22:49:26.267298Z"
          finished_at: "2026-10-16T22:49:36.919722Z"
          session_id: f05e1f4c-b0b9-47e7-a879-88650252a387
          outcome: returned
      "n": 3
      timestamp: "2026-10-16T15:49:37-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: not-addressed
          note: ListingPoller still defines only fresh (<=5 s) and stale (age named); the no-snapshot-yet read at supervisor start is unspecified and untested in Task 1/2. Minor, carried to close review.
          round: 3
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            id: 41a5697e-e0e8-4d13-a483-6a6aaced7b27
            repository: github.com/xianxu/pair
            issue_id: "000618"
            recipe: plan-quality
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/workspace/worktree/pair-slot2/pair
            reviewed_head: 4739bc4f6e89394471b104ac6b70f4a54253bb6a
            slot: pair:2
            requested_at: "2026-10-16T22:51:17.013355Z"
          phase: finished
          pid: 76579
          started_at: "2026-10-16T22:51:18.378203Z"
          finished_at: "2026-10-16T22:51:45.113585Z"
          session_id: 5c968657-789a-43c4-ac98-9785c4f1a7d8
          outcome: returned
      "n": 4
      timestamp: "2026-10-16T15:51:48-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: not-addressed
          note: No text yet on what an admission does before the poller's first listing; still Minor, carried to close.
          round: 4
      blocked: false
content_hash: 0da396674f475649432fd34abe1488ce942fdc3aa0b9e7dbf17055096ce0aaf6
---

# Gate ledger — pair#618 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T15:47:36-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `operating-envelope-arithmetic` (kind: defect) A cold cache miss still runs two serial listings, about 2.1 s at the measured rate, which exceeds the 2 s AdmissionTimeout
  snapshot runs list-sessions --short then --no-formatting serially (launcher/zellij.go:76-80); at 0.13-1.07 s each a single miss blows the budget, and Task 2's 1 s-per-listing test cannot pass cold. State the fix (concurrent listings, a background warm refresh, or a separate budget) and the single-flight context semantics (a leader's cancel must not fail followers).
- **PQ-2** [Important] `new-record-lifecycle-unstated` (kind: defect) Refused receipts are stored like other receipts: same-ID retries are pinned to the refusal for 1 h and refusals fill MaxReceipts
  Send returns a retained receipt for a known ID (broker.go:532-539) and clients keep IDs through retries (protocol.go:73), so a stored refusal blocks re-admission for ReceiptLifetime (1 h). Refusals also count toward MaxReceipts=256 (broker.go:553), which becomes a capacity cliff under the load being fixed. State the lifetime and bound, the retry semantics, and whether Refused is excluded from Status.Terminal() for wrapper-reported outcomes.
- **PQ-3** [Important] `done-when-tracks-spec` (kind: defect) Spec and Done-when still describe a reboot/resume registration bug the Log has disproved
  Done-when requires rebooted and resumed slots to receive sends, but the plan's tests target slow-listing admission, refusal reasons and peer-message confirmation. Add a Revisions entry and restate Done-when to match the measured root cause.
- **PQ-4** [Important] `new-record-lifecycle-unstated` (kind: defect) M2 Task 6 adds an unnamed wrapper-written file of delivered-body hashes with no path, format, parse-failure behaviour or removal
  "Where the watcher reads them" leaves the cross-process artifact undefined. Name the owner, path and format, the behaviour on truncated or older-version input (ARCH-SECURE), and its removal and growth bound (ARCH-FUNERAL).
- **PQ-5** [Minor] `vague-plan-step` (kind: defect) Task 7's relaunch "longer bound" is unnamed, as is the behaviour when it is exceeded
- **PQ-6** [Minor] `deterministic-listing-path` (kind: defect) The --list receive state should be one supervisor request for all slots, and should name the slot when recipient resolution fails
- **PQ-7** [Minor] `test-prose-enumeration` (kind: defect) Task 1's tests are an enumerated case list; compress to one strategy line (injected clock, blocking runner, -race)

## Round 2 — 2026-10-16T15:49:06-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — A background poller runs both listings concurrently; admissions read the in-memory snapshot and never run zellij.
- PQ-2 — addressed — 10-minute lifetime, a cap of 64 refusals kept outside MaxReceipts, and a resend with the refused ID is admitted fresh.
- PQ-3 — addressed — A Revisions entry was added and the Spec and Done-when were restated to the measured load cause.
- PQ-4 — addressed — peer_delivery records go in the existing ledger through its locked append, with its malformed-line handling and lifecycle.
- PQ-5 — addressed — The wait is bounded at 60 s; ParkIncomplete is returned at 60 s.
- PQ-6 — addressed
- PQ-7 — addressed

### Raised

- **PQ-8** [Minor] `operating-envelope-arithmetic` (kind: defect) The plan doesn't say what an admission does before the poller's first snapshot exists
  Right after the supervisor starts, no listing has completed yet. Say whether an admission then waits for the first listing within AdmissionTimeout or refuses with a distinct reason, and cover it in the Task 1 or Task 2 tests.

## Round 3 — 2026-10-16T15:49:37-07:00 (claude) — passed

### Disposed

- PQ-8 — not-addressed — ListingPoller still defines only fresh (<=5 s) and stale (age named); the no-snapshot-yet read at supervisor start is unspecified and untested in Task 1/2. Minor, carried to close review.

## Round 4 — 2026-10-16T15:51:48-07:00 (claude) — passed

### Disposed

- PQ-8 — not-addressed — No text yet on what an admission does before the poller's first listing; still Minor, carried to close.

## Open findings

- **PQ-8** [Minor] `operating-envelope-arithmetic` (kind: defect) The plan doesn't say what an admission does before the poller's first snapshot exists
