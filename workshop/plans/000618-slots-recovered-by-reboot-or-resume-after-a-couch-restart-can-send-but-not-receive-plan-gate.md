---
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

## Open findings

- **PQ-1** [Important] `operating-envelope-arithmetic` (kind: defect) A cold cache miss still runs two serial listings, about 2.1 s at the measured rate, which exceeds the 2 s AdmissionTimeout
- **PQ-2** [Important] `new-record-lifecycle-unstated` (kind: defect) Refused receipts are stored like other receipts: same-ID retries are pinned to the refusal for 1 h and refusals fill MaxReceipts
- **PQ-3** [Important] `done-when-tracks-spec` (kind: defect) Spec and Done-when still describe a reboot/resume registration bug the Log has disproved
- **PQ-4** [Important] `new-record-lifecycle-unstated` (kind: defect) M2 Task 6 adds an unnamed wrapper-written file of delivered-body hashes with no path, format, parse-failure behaviour or removal
- **PQ-5** [Minor] `vague-plan-step` (kind: defect) Task 7's relaunch "longer bound" is unnamed, as is the behaviour when it is exceeded
- **PQ-6** [Minor] `deterministic-listing-path` (kind: defect) The --list receive state should be one supervisor request for all slots, and should name the slot when recipient resolution fails
- **PQ-7** [Minor] `test-prose-enumeration` (kind: defect) Task 1's tests are an enumerated case list; compress to one strategy line (injected clock, blocking runner, -race)
