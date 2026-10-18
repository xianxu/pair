---
gate: boundary-review
issue: 656
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c51cfd62-3f66-4aa1-b876-95659cbbf09a
            repository: github.com/xianxu/pair
            issue_id: "000656"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5e1e913eea8bea39
            reviewed_head: 5afa91e728b992b16d112cc49d2fb8aba39c6efc
            slot: pair:6
            requested_at: "2026-10-18T12:55:45.872106Z"
          phase: finished
          pid: 21603
          started_at: "2026-10-18T12:55:45.969015Z"
          finished_at: "2026-10-18T12:57:38.600546Z"
          session_id: 01a14f15-1d17-72f3-b357-a0ce9aa44cce
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 5afa91e728b992b16d112cc49d2fb8aba39c6efc
        tracker_head: 272ba504c15217f30aab4022d0263fc18c5ffa61
        branch_point: 479b91376e5055f8106fe95b34bc64c71c6d2bc6
        since: "2026-10-18T05:12:41-07:00"
        until: "2026-10-18T05:55:28-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T05:12:41-07:00"
              end: "2026-10-18T05:55:28-07:00"
              began: "2026-10-18T05:12:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: f3c217a27bf35da62e169db66d1b6a60d94e0a5585f4bd2e227e8602b7a270a2
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 10750027
              windows:
                - start: "2026-10-18T05:12:41-07:00"
                  end: "2026-10-18T05:55:28-07:00"
        queue_waits:
            - attempt_id: 67113e66e35b559d7a5501b6ac24a65a
              slot: pair:6
              record_digest: f907181eb97a9740df5e50e01a68467f3a5efe49aaabead2ed4511b3d0f4e5b9
              start: "2026-10-18T12:24:51.517211Z"
              end: "2026-10-18T12:42:03.74225Z"
              queue_seconds: 1032.225039
              raw_seconds: 1032.225039
              exact: true
      "n": 1
      timestamp: "2026-10-18T05:57:48-07:00"
      dispatched: "2026-10-18T05:55:44-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Database growth bypasses terminal mutation checks during reads
          detail: 'cmd/internal/sessioninventory/title_projection.go:399 compares the transcript only against p.transcript.Size. After an append, a same-size rewrite during the boundary or suffix read escapes this comparison, while errTitleReadmit skips the paired fingerprint checks at lines 446 and 461. Both interleavings stop without database growth but return nil with growth, as confirmed by temporary overlay regressions. The other instances in this family are transcript and database replacement during the new retained-boundary read at line 403: neither is re-statted before line 411 returns transient recovery, and both overlay regressions fail. Preserve the paired fingerprint checks before transient classification, fence the added boundary read with artifact metadata checks, and add regressions for all four cases. ARCH-PURPOSE: growth must not hide the mutation class the Spec explicitly requires to remain terminal.'
          family: mutation-precedence-before-readmission
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 6755cb05-1760-4581-8b3d-c3a9e2ad5ab6
            repository: github.com/xianxu/pair
            issue_id: "000656"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: codex
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/0f947e6fa4e07ace
            reviewed_head: 35c214922cd6b66920d0042a4165e816e8ad5fa3
            slot: pair:6
            requested_at: "2026-10-18T13:32:12.485503Z"
          phase: finished
          pid: 57186
          started_at: "2026-10-18T13:32:12.640328Z"
          finished_at: "2026-10-18T13:33:20.836141Z"
          session_id: 01a14f36-7ae0-7c61-97d5-0fede5d5c6e7
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 35c214922cd6b66920d0042a4165e816e8ad5fa3
        tracker_head: 0a70ed19b500a1c01a54a506e8d3189d40484de0
        branch_point: 479b91376e5055f8106fe95b34bc64c71c6d2bc6
        since: "2026-10-18T05:12:41-07:00"
        until: "2026-10-18T06:31:37-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T05:12:41-07:00"
              end: "2026-10-18T06:31:37-07:00"
              began: "2026-10-18T05:12:41-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot6/pair
        sources:
            - harness: codex
              id: 01a14b2c-9890-74f3-a738-58879659bef5
              slot: pair:6
              cwd: /Users/xianxu/workspace/worktree/pair-slot6/pair
              digest: a60f884f11b2315f5829e481188171270ef2a581cd001a3af3b1b867ed44afce
              started: "2026-10-17T18:42:56.865Z"
              spawned: false
              bytes: 11574099
              windows:
                - start: "2026-10-18T05:12:41-07:00"
                  end: "2026-10-18T06:31:37-07:00"
        queue_waits:
            - attempt_id: 67113e66e35b559d7a5501b6ac24a65a
              slot: pair:6
              record_digest: f907181eb97a9740df5e50e01a68467f3a5efe49aaabead2ed4511b3d0f4e5b9
              start: "2026-10-18T12:24:51.517211Z"
              end: "2026-10-18T12:42:03.74225Z"
              queue_seconds: 1032.225039
              raw_seconds: 1032.225039
              exact: true
            - attempt_id: 15941241da8dd25043c701d0914fb006
              slot: pair:6
              record_digest: 86b0b8aecc8ed702cb7f6641f05387beececa3dc73bd41ebcee8dc1e36335cfd
              start: "2026-10-18T13:04:28.401107Z"
              end: "2026-10-18T13:20:04.907964Z"
              queue_seconds: 936.506857
              raw_seconds: 936.506857
              exact: true
        cuts:
            - dispatched: "2026-10-18T05:55:44-07:00"
              finalized: "2026-10-18T05:57:48-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-18T06:33:36-07:00"
      dispatched: "2026-10-18T06:32:10-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Paired fingerprint checks now precede readmission at title_projection.go:399; the recovery boundary read is fenced by fresh artifact metadata at lines 419–424. Both rewrite regressions and both replacement regressions pass at head and fail against pre-fix production via a temporary Go overlay. Expanded mutation controls also fail without the fix.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 35c214922cd6b66920d0042a4165e816e8ad5fa3
      blocked: false
---

# Gate ledger — pair#656 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-18T05:57:48-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `mutation-precedence-before-readmission` (kind: defect) Database growth bypasses terminal mutation checks during reads
  cmd/internal/sessioninventory/title_projection.go:399 compares the transcript only against p.transcript.Size. After an append, a same-size rewrite during the boundary or suffix read escapes this comparison, while errTitleReadmit skips the paired fingerprint checks at lines 446 and 461. Both interleavings stop without database growth but return nil with growth, as confirmed by temporary overlay regressions. The other instances in this family are transcript and database replacement during the new retained-boundary read at line 403: neither is re-statted before line 411 returns transient recovery, and both overlay regressions fail. Preserve the paired fingerprint checks before transient classification, fence the added boundary read with artifact metadata checks, and add regressions for all four cases. ARCH-PURPOSE: growth must not hide the mutation class the Spec explicitly requires to remain terminal.

## Round 2 — 2026-10-18T06:33:36-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Paired fingerprint checks now precede readmission at title_projection.go:399; the recovery boundary read is fenced by fresh artifact metadata at lines 419–424. Both rewrite regressions and both replacement regressions pass at head and fail against pre-fix production via a temporary Go overlay. Expanded mutation controls also fail without the fix.

## Open findings

(none — every finding has been disposed)
