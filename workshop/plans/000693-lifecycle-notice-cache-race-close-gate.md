---
gate: boundary-review
issue: 693
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: c6955202-c283-4ada-97c4-1ea80abc866f
            repository: github.com/xianxu/pair
            issue_id: "000693"
            recipe: small-diff-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/c2c736c6873fb820
            reviewed_head: fdac39f3a0dfc74ccc828ba25211e04faa2a4e0a
            slot: pair:2
            requested_at: "2026-10-19T11:20:58.569114Z"
          phase: finished
          pid: 80464
          started_at: "2026-10-19T11:20:58.785992Z"
          finished_at: "2026-10-19T11:26:30.180634Z"
          session_id: 1e081480-d245-4832-a613-87f512cdcad2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: fdac39f3a0dfc74ccc828ba25211e04faa2a4e0a
        tracker_head: e903a0f4e11c6aafb2ad062109986970e006f7a6
        branch_point: f7be53a7fc8f3749d369eb821fbb21579b433712
        since: "2026-10-19T03:57:11-07:00"
        until: "2026-10-19T04:03:57-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T03:57:11-07:00"
              end: "2026-10-19T04:03:57-07:00"
              began: "2026-10-19T03:57:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: e94902dd625281886b8015b597604584493e196e7297fb3cd7e3a6fffcd4d1f9
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 43326643
              windows:
                - start: "2026-10-19T03:57:11-07:00"
                  end: "2026-10-19T04:03:57-07:00"
      "n": 1
      timestamp: "2026-10-19T04:26:31-07:00"
      dispatched: "2026-10-19T04:20:54-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: Restart-drain diagnostic now reports ordinary non-slot threads as lacking slot metadata
          detail: 'Ordinary facts lacking identity now enter p.unresolved, and the restart-drained branch (lifecycle_notices.go:153-158) counts every unresolved entry into a user-visible Console status notice. Probe confirmed: one ordinary-target recovered fact for a non-slot path between restart-armed and restart-drained emits "Couch lifecycle: 1 observations lacked verified slot metadata at restart drain" at head and nothing at base. Fix: count only untargeted (Target.Kind empty) entries at drain and clear ordinary ones silently, or word the diagnostic by target kind; add a test that an ordinary non-slot fact produces no drain diagnostic and an untargeted one still does. Same family in this window: the drain branch is the only consumer of unresolved that reports to the user; also update atlas/couch.md 883-888 to state what drain reports.'
          family: widened-state-consumer-unswept
          round: 1
        - kind: defect
          id: BR-2
          severity: Minor
          title: Only two lifecycleNextMessage call sites pass the publisher for the timeout diagnostic
          detail: The variadic publisher argument is passed only in the primary-slot and ordinary-wait tests; the other roughly ten call sites still fail with the bare "no lifecycle message".
          family: bounded-wait-diagnostic-coverage
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: Unresolved ordinary facts have no age bound and can publish arbitrarily late
          detail: An ordinary fact waits until drain, ownership change or eviction; if its wrapper verifies hours later, a stale notice is published. Newest-per-path softens this.
          family: unbounded-age-retention
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 89c16974-7097-45c9-ba89-c85581467dd6
            repository: github.com/xianxu/pair
            issue_id: "000693"
            recipe: small-diff-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/eb8257215319e8a8
            reviewed_head: 877401b241a651bd16c3dc6c14def7447e0917e5
            slot: pair:2
            requested_at: "2026-10-19T11:30:32.356393Z"
          phase: finished
          pid: 25141
          started_at: "2026-10-19T11:30:32.454742Z"
          finished_at: "2026-10-19T11:31:29.478162Z"
          session_id: 87b99c43-0b7a-461a-829f-2db6ec5a1505
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 877401b241a651bd16c3dc6c14def7447e0917e5
        tracker_head: 553413985febf3c1dd4d5178614a157c3748778e
        branch_point: f7be53a7fc8f3749d369eb821fbb21579b433712
        since: "2026-10-19T03:57:11-07:00"
        until: "2026-10-19T04:28:44-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-19T03:57:11-07:00"
              end: "2026-10-19T04:28:44-07:00"
              began: "2026-10-19T03:57:11-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              slot: pair:2
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: bcc172e46f59c06b6218b5920a40f58394b608cf1cbba9ae636b5282aee05c8b
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 43580627
              windows:
                - start: "2026-10-19T03:57:11-07:00"
                  end: "2026-10-19T04:28:44-07:00"
        queue_waits:
            - attempt_id: ed29c0db8dca8225a67b7fd4ae9d0cbc
              slot: pair:2
              record_digest: 4bd92929838b7fb562e413a3068436421c7addbc9db54bed13c45dcd113238fc
              start: "2026-10-19T11:04:06.932955Z"
              end: "2026-10-19T11:04:47.611061Z"
              queue_seconds: 40.678106
              raw_seconds: 40.678106
              exact: true
        cuts:
            - dispatched: "2026-10-19T04:20:54-07:00"
              finalized: "2026-10-19T04:26:31-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-19T04:31:30-07:00"
      dispatched: "2026-10-19T04:30:30-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: Drain counts only Target.Kind=="" and clears all (lifecycle_notices.go:170-180); TestLifecycleDrainReportsOnlyUntargetedFacts expects 1 (would be 2 without the fix); atlas/couch.md updated.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The publisher parameter is now required, so the compiler forces every call site, including native_exit_test.go:173, to pass it; the whole family is closed.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Ordinary facts expire after lifecycleOrdinaryWait, checked on the 1s refresh ticker; TestLifecycleOrdinaryFactExpires checks exactly-at-limit and one second past it.
          round: 2
      findings:
        - kind: defect
          id: BR-4
          severity: Minor
          title: retainUnresolvedLocked doc comment omits the new ordinary-fact expiry
          detail: lifecycle_notices.go:241-244 says drain and ownership change clear the set; reducePendingLocked now also evicts ordinary facts after lifecycleOrdinaryWait. This is the only stale comment in the window; atlas/couch.md already states the expiry.
          family: widened-state-consumer-unswept
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 877401b241a651bd16c3dc6c14def7447e0917e5
      blocked: false
---

# Gate ledger — pair#693 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T04:26:31-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `widened-state-consumer-unswept` (kind: defect) Restart-drain diagnostic now reports ordinary non-slot threads as lacking slot metadata
  Ordinary facts lacking identity now enter p.unresolved, and the restart-drained branch (lifecycle_notices.go:153-158) counts every unresolved entry into a user-visible Console status notice. Probe confirmed: one ordinary-target recovered fact for a non-slot path between restart-armed and restart-drained emits "Couch lifecycle: 1 observations lacked verified slot metadata at restart drain" at head and nothing at base. Fix: count only untargeted (Target.Kind empty) entries at drain and clear ordinary ones silently, or word the diagnostic by target kind; add a test that an ordinary non-slot fact produces no drain diagnostic and an untargeted one still does. Same family in this window: the drain branch is the only consumer of unresolved that reports to the user; also update atlas/couch.md 883-888 to state what drain reports.
- **BR-2** [Minor] `bounded-wait-diagnostic-coverage` (kind: defect) Only two lifecycleNextMessage call sites pass the publisher for the timeout diagnostic
  The variadic publisher argument is passed only in the primary-slot and ordinary-wait tests; the other roughly ten call sites still fail with the bare "no lifecycle message".
- **BR-3** [Minor] `unbounded-age-retention` (kind: defect) Unresolved ordinary facts have no age bound and can publish arbitrarily late
  An ordinary fact waits until drain, ownership change or eviction; if its wrapper verifies hours later, a stale notice is published. Newest-per-path softens this.

## Round 2 — 2026-10-19T04:31:30-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Drain counts only Target.Kind=="" and clears all (lifecycle_notices.go:170-180); TestLifecycleDrainReportsOnlyUntargetedFacts expects 1 (would be 2 without the fix); atlas/couch.md updated.
- BR-2 — addressed — The publisher parameter is now required, so the compiler forces every call site, including native_exit_test.go:173, to pass it; the whole family is closed.
- BR-3 — addressed — Ordinary facts expire after lifecycleOrdinaryWait, checked on the 1s refresh ticker; TestLifecycleOrdinaryFactExpires checks exactly-at-limit and one second past it.

### Raised

- **BR-4** [Minor] `widened-state-consumer-unswept` (kind: defect) retainUnresolvedLocked doc comment omits the new ordinary-fact expiry
  lifecycle_notices.go:241-244 says drain and ownership change clear the set; reducePendingLocked now also evicts ordinary facts after lifecycleOrdinaryWait. This is the only stale comment in the window; atlas/couch.md already states the expiry.

## Open findings

- **BR-4** [Minor] `widened-state-consumer-unswept` (kind: defect) retainUnresolvedLocked doc comment omits the new ordinary-fact expiry
