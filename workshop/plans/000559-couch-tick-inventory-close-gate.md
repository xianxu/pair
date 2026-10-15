---
gate: boundary-review
issue: 559
id_prefix: BR
rounds:
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: e541a6d669708ef6743504511c9d9868c53541e3
        tracker_head: 08277fd20427f621f7fe30b6c2f18b80d66813e2
        branch_point: c6af746c94f18ef4a58cb092a739cc9fe0887124
        since: "2026-10-14T16:33:51-07:00"
        until: "2026-10-14T17:20:19-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-14T16:33:51-07:00"
              end: "2026-10-14T17:20:19-07:00"
              began: "2026-10-14T16:33:51-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 756e35027e46383290be4d0b12353e55eff10964a63bb04327b16b32b3ba7370
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 49135133
              windows:
                - start: "2026-10-14T16:33:51-07:00"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13ccd-4f29-74a1-b8c1-73b75b866b28
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 4d1e8f6a0660d639e1260ed86a7d79e5e49395573aa6416c3491e510ed42f646
              started: "2026-10-14T23:44:11.166Z"
              spawned: false
              bytes: 632590
              windows:
                - start: "2026-10-14T23:44:11.166Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13ccf-10b7-7c42-b28b-c7de21ec6d8d
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8868a0dbad5449404deccf5b8a0e47cdbd6797027f692e4bfd0286d082e9cbdc
              started: "2026-10-14T23:46:06.691Z"
              spawned: false
              bytes: 639655
              windows:
                - start: "2026-10-14T23:46:06.691Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13cd1-8b39-7833-b2f7-5964c6983bf3
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 955cd9486d9c8bfd68d25d851bbd462571f0c1a9e1b13fe9edaf47f33d9a59e7
              started: "2026-10-14T23:48:48.938Z"
              spawned: false
              bytes: 440291
              windows:
                - start: "2026-10-14T23:48:48.938Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13cd2-007c-7a03-832d-da65ea158cd8
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 2a5bb9afd68a2df4295e19cd45e1a5109f782104e99eca7312ede7cc4b60be20
              started: "2026-10-14T23:49:18.749Z"
              spawned: false
              bytes: 415395
              windows:
                - start: "2026-10-14T23:49:18.749Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13cd2-ae96-7553-ba44-343c0c764581
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: bfc91c29ddadc760b1d8b5345c6b063bf1c70451a6702974f800575355dd0cfc
              started: "2026-10-14T23:50:02.754Z"
              spawned: true
              bytes: 1013243
              windows:
                - start: "2026-10-14T23:50:02.754Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13cd2-d32d-7f23-bc38-c9858e67d9cc
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 501784e68ee22bd5be94cac32e18455a18e5490554e89b1e7d360f21cbb51202
              started: "2026-10-14T23:50:12.104Z"
              spawned: true
              bytes: 1189380
              windows:
                - start: "2026-10-14T23:50:12.104Z"
                  end: "2026-10-14T17:20:19-07:00"
            - harness: codex
              id: 01a13cd9-cfc0-7b83-a804-01c055ef39c6
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 5b926acffc61cd60902891e24a0d64507c61346f0f378acf203338fe01bcd99a
              started: "2026-10-14T23:57:50.135Z"
              spawned: true
              bytes: 557587
              windows:
                - start: "2026-10-14T23:57:50.135Z"
                  end: "2026-10-14T17:20:19-07:00"
      "n": 1
      timestamp: "2026-10-14T17:28:05-07:00"
      dispatched: "2026-10-14T17:20:55-07:00"
      agent: codex
      findings:
        - kind: defect
          id: BR-1
          severity: Critical
          title: Failed observations can preserve or publish reusable evidence
          detail: cmd/internal/sessioninventory/query.go:370-389 returns on failed listing/stat or missing ledger without invalidating the existing owner entry; after recovery with unchanged metadata, QueryResumeTarget reuses the old parse. Separately, activity_cache.go:378-379 does not record catalog-load failure, which QuerySessionContext intentionally tolerates; successful native fallback can then publish a ready activity entry at lines 212-216 despite incomplete catalog observation. This contradicts the Spec's explicit invalidation rule (ARCH-SECURE, ARCH-ORDER). Invalidate owner evidence on incomplete source observations and prevent activity publication after failed catalog observation; retain uncached fallback behavior. Add failure→recovery regressions proving fresh observation occurs.
          family: incomplete-observation-invalidates-evidence
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Previous-launch reuse retains full historical boundary and proof payloads
          detail: cmd/internal/sessioninventory/query.go:476-478 retains the complete previous sessionledger.Current in ownerLaunchEntry.previous (launch_cache.go:34), including launch boundary arrays and binding scanner proofs. ResumeTargetForLaunch consumes only identity/status fields, so these potentially large payloads remain resident unnecessarily across up to 128 entries (ARCH-CONSTRAINTS, ARCH-FUNERAL). Store a compact fallback projection and audit both retained launch projections for unused payloads. Add a substantive large-boundary fixture and retained-memory evidence; the current scale fixture expands files with whitespace and cannot detect this retention.
          family: compact-cache-projections
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: Activity publication lacks the promised controlled ordering regressions
          detail: cmd/internal/sessioninventory/activity_cache_test.go:89-106 cancels the purported waiter before Query starts, so it never exercises waiting behind the loader. The activity tests also do not replace dependencies during a blocked load or force repeated drift before publication. Owner/index race tests do not exercise the activity tracker and commit path (ARCH-ORDER). Add barrier-controlled activity mutation, active-loader cancellation, and genuinely queued-waiter tests that fail when the corresponding publication guards are removed.
          family: temporal-cache-regression-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: dbf3acf30224bbaad8ae1c0c1f625d7a538063ae
        tracker_head: d77d1fcb79ac8913c853877c9dd948a304811a41
        branch_point: 4780a55286f5a171d64907cf03d9c12c7a2ce656
        since: "2026-10-14T16:33:51-07:00"
        until: "2026-10-14T17:41:38-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-14T16:33:51-07:00"
              end: "2026-10-14T17:41:38-07:00"
              began: "2026-10-14T16:33:51-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot7/pair
        sources:
            - harness: codex
              id: 01a13398-d940-72d3-8d80-c203fc8bd296
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 28fe9ba057e8a2fc56a0bdbdf187dd48601322b3152a1cbf0442233abf6e7b82
              started: "2026-10-13T04:50:49.58Z"
              spawned: false
              bytes: 50869522
              windows:
                - start: "2026-10-14T16:33:51-07:00"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13ccd-4f29-74a1-b8c1-73b75b866b28
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 4d1e8f6a0660d639e1260ed86a7d79e5e49395573aa6416c3491e510ed42f646
              started: "2026-10-14T23:44:11.166Z"
              spawned: false
              bytes: 632590
              windows:
                - start: "2026-10-14T23:44:11.166Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13ccf-10b7-7c42-b28b-c7de21ec6d8d
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 8868a0dbad5449404deccf5b8a0e47cdbd6797027f692e4bfd0286d082e9cbdc
              started: "2026-10-14T23:46:06.691Z"
              spawned: false
              bytes: 639655
              windows:
                - start: "2026-10-14T23:46:06.691Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13cd1-8b39-7833-b2f7-5964c6983bf3
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 955cd9486d9c8bfd68d25d851bbd462571f0c1a9e1b13fe9edaf47f33d9a59e7
              started: "2026-10-14T23:48:48.938Z"
              spawned: false
              bytes: 440291
              windows:
                - start: "2026-10-14T23:48:48.938Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13cd2-007c-7a03-832d-da65ea158cd8
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              digest: 2a5bb9afd68a2df4295e19cd45e1a5109f782104e99eca7312ede7cc4b60be20
              started: "2026-10-14T23:49:18.749Z"
              spawned: false
              bytes: 415395
              windows:
                - start: "2026-10-14T23:49:18.749Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13cd2-ae96-7553-ba44-343c0c764581
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: a21fe1b0ca181987f67d1a1776e64c2d5d37bc45e5badafbda6e583fac741171
              started: "2026-10-14T23:50:02.754Z"
              spawned: true
              bytes: 1353026
              windows:
                - start: "2026-10-14T23:50:02.754Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13cd2-d32d-7f23-bc38-c9858e67d9cc
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: f768cf0f648172138f74e5c98032b3ea55bd20fe9b0d7868e45ca1a7d4349f75
              started: "2026-10-14T23:50:12.104Z"
              spawned: true
              bytes: 1570964
              windows:
                - start: "2026-10-14T23:50:12.104Z"
                  end: "2026-10-14T17:41:38-07:00"
            - harness: codex
              id: 01a13cd9-cfc0-7b83-a804-01c055ef39c6
              cwd: /Users/xianxu/workspace/worktree/pair-slot7/pair
              parent_id: 01a13398-d940-72d3-8d80-c203fc8bd296
              digest: 5b926acffc61cd60902891e24a0d64507c61346f0f378acf203338fe01bcd99a
              started: "2026-10-14T23:57:50.135Z"
              spawned: true
              bytes: 557587
              windows:
                - start: "2026-10-14T23:57:50.135Z"
                  end: "2026-10-14T17:41:38-07:00"
        cuts:
            - dispatched: "2026-10-14T17:20:55-07:00"
              finalized: "2026-10-14T17:28:05-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-14T17:46:26-07:00"
      dispatched: "2026-10-14T17:42:17-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'Owner invalidation is verified: removing it fails failure→recovery and active-loader regressions. Catalog-load invalidation still lacks independent regression evidence. Removing activity_cache.go:402-404 leaves TestActivityCacheFailedCatalogObservationNeverPublishesFallback green because activity_cache_failure_test.go:30-34 also fails publication, which independently prevents caching. A disposable overlay allowing publication success passes with the load guard and fails without it with “incomplete catalog observation published reusable fallback.” Commit that independent read-failure regression.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: Previous evidence is now a ResumeTarget; compactOwnerLaunch removes unused current boundaries, watermarks and binding histories while retaining required authorization proof. The valid substantive fixture measured 906502 source bytes and 615 retained encoded bytes. Removing compaction fails TestCompactOwnerLaunchKeepsCurrentProofDropsUnusedPayloads.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Activity tests now establish an actually queued waiter, cancel an active loader, replace a ledger during blocked loading, and force drift across both bounded attempts. Targeted race checks pass. Bypassing stability validation fails both replacement and repeated-drift regressions.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#559 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-14T17:28:05-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `incomplete-observation-invalidates-evidence` (kind: defect) Failed observations can preserve or publish reusable evidence
  cmd/internal/sessioninventory/query.go:370-389 returns on failed listing/stat or missing ledger without invalidating the existing owner entry; after recovery with unchanged metadata, QueryResumeTarget reuses the old parse. Separately, activity_cache.go:378-379 does not record catalog-load failure, which QuerySessionContext intentionally tolerates; successful native fallback can then publish a ready activity entry at lines 212-216 despite incomplete catalog observation. This contradicts the Spec's explicit invalidation rule (ARCH-SECURE, ARCH-ORDER). Invalidate owner evidence on incomplete source observations and prevent activity publication after failed catalog observation; retain uncached fallback behavior. Add failure→recovery regressions proving fresh observation occurs.
- **BR-2** [Important] `compact-cache-projections` (kind: defect) Previous-launch reuse retains full historical boundary and proof payloads
  cmd/internal/sessioninventory/query.go:476-478 retains the complete previous sessionledger.Current in ownerLaunchEntry.previous (launch_cache.go:34), including launch boundary arrays and binding scanner proofs. ResumeTargetForLaunch consumes only identity/status fields, so these potentially large payloads remain resident unnecessarily across up to 128 entries (ARCH-CONSTRAINTS, ARCH-FUNERAL). Store a compact fallback projection and audit both retained launch projections for unused payloads. Add a substantive large-boundary fixture and retained-memory evidence; the current scale fixture expands files with whitespace and cannot detect this retention.
- **BR-3** [Important] `temporal-cache-regression-coverage` (kind: defect) Activity publication lacks the promised controlled ordering regressions
  cmd/internal/sessioninventory/activity_cache_test.go:89-106 cancels the purported waiter before Query starts, so it never exercises waiting behind the loader. The activity tests also do not replace dependencies during a blocked load or force repeated drift before publication. Owner/index race tests do not exercise the activity tracker and commit path (ARCH-ORDER). Add barrier-controlled activity mutation, active-loader cancellation, and genuinely queued-waiter tests that fail when the corresponding publication guards are removed.

## Round 2 — 2026-10-14T17:46:26-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — Owner invalidation is verified: removing it fails failure→recovery and active-loader regressions. Catalog-load invalidation still lacks independent regression evidence. Removing activity_cache.go:402-404 leaves TestActivityCacheFailedCatalogObservationNeverPublishesFallback green because activity_cache_failure_test.go:30-34 also fails publication, which independently prevents caching. A disposable overlay allowing publication success passes with the load guard and fails without it with “incomplete catalog observation published reusable fallback.” Commit that independent read-failure regression.
- BR-2 — addressed — Previous evidence is now a ResumeTarget; compactOwnerLaunch removes unused current boundaries, watermarks and binding histories while retaining required authorization proof. The valid substantive fixture measured 906502 source bytes and 615 retained encoded bytes. Removing compaction fails TestCompactOwnerLaunchKeepsCurrentProofDropsUnusedPayloads.
- BR-3 — addressed — Activity tests now establish an actually queued waiter, cancel an active loader, replace a ledger during blocked loading, and force drift across both bounded attempts. Targeted race checks pass. Bypassing stability validation fails both replacement and repeated-drift regressions.

## Open findings

- **BR-1** [Critical] `incomplete-observation-invalidates-evidence` (kind: defect) Failed observations can preserve or publish reusable evidence
