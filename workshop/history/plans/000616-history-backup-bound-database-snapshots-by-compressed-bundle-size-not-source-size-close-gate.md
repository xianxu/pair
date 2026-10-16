---
gate: boundary-review
issue: 616
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 4948d216-7bdc-4d38-adc4-a44786962dd1
            repository: github.com/xianxu/pair
            issue_id: "000616"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/b1fc78674ba95240
            reviewed_head: c9aa472665128fecbdaaf2294a9cb07e47e62a77
            slot: pair:2
            requested_at: "2026-10-16T18:38:55.777276Z"
          phase: finished
          pid: 52851
          started_at: "2026-10-16T18:38:56.124703Z"
          finished_at: "2026-10-16T18:41:11.399301Z"
          session_id: e39ea5da-edf2-4518-a85c-e5f3ebf75f82
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: c9aa472665128fecbdaaf2294a9cb07e47e62a77
        tracker_head: bb22dd672578e4f0fa618573e3f9f2062f94ec18
        branch_point: fb3e2e6debf0b009a94a358fa2b7f9d0bb6c1a77
        since: "2026-10-16T11:06:23-07:00"
        until: "2026-10-16T11:38:06-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T11:06:23-07:00"
              end: "2026-10-16T11:38:06-07:00"
              began: "2026-10-16T11:06:23-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 3a8ce1ae58fce810b9bac83ba7a88d499599a45e6dadc489e35cd6f64ac5774a
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 11772744
              windows:
                - start: "2026-10-16T11:06:23-07:00"
                  end: "2026-10-16T11:38:06-07:00"
      "n": 1
      timestamp: "2026-10-16T11:41:14-07:00"
      dispatched: "2026-10-16T11:38:51-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: ErrCiphertextLimit skip in the continuous pass drops every other snapshot still pending in the same request
          detail: 'Archiver.create returns at the first failing manifest (archive.go:85-88), and the new branch (continuous.go:177-183) then skips all n requested files with fresh = fresh[n:], so snapshots planned after the oversized bundle (sorted by month and path, so the same order every pass) are never archived, and the note does not name the file. Fix: archive snapshot roots one file per request, or retry the remaining files minus the failing manifest; put the path in the note; add a test with two snapshots where the oversized one sorts first.'
          family: partial-batch-failure-drops-siblings
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Daily archiving of thread_history_1.sqlite adds about 115 MB per pass to a never-deleting sink, and the atlas does not state the cost
          detail: Before this change the 1.6 GB database was skipped. Now each content change writes a new ~115 MB bundle, roughly 40 GB a year, and bundles are kept until a future retention decision (atlas/history-backup.md:148). Per ARCH-FUNERAL, add the per-pass cost and what bounds it to the history-backup storage section or storage-retention-io.md.
          family: growth-cost-unstated
          round: 1
        - kind: defect
          id: BR-3
          severity: Minor
          title: rawBundleLimit re-parses the built-in inventory on every call, including once per catalog record read
          detail: Parse it once (sync.OnceValue). Also, planBundles at plan.go:103 should size the label-overhead estimate with rawBundleLimit(r.Kind) instead of MaxBundleBytes; it is only correct today because 8 GiB and 1 GiB have the same number of digits.
          family: repeated-static-parse
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: bd40ca0d-d7bd-4b2a-ba77-5edd2e326588
            repository: github.com/xianxu/pair
            issue_id: "000616"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/9a4db2b4da2ca939
            reviewed_head: bedbf0ba13f30d25b364f1feafcc5173b1970af5
            slot: pair:2
            requested_at: "2026-10-16T18:43:24.392582Z"
          phase: finished
          pid: 8954
          started_at: "2026-10-16T18:43:24.602169Z"
          finished_at: "2026-10-16T18:45:03.751645Z"
          session_id: c5c4ff31-e891-460c-9d7b-be732664f221
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: bedbf0ba13f30d25b364f1feafcc5173b1970af5
        tracker_head: bb22dd672578e4f0fa618573e3f9f2062f94ec18
        branch_point: fb3e2e6debf0b009a94a358fa2b7f9d0bb6c1a77
        since: "2026-10-16T11:06:23-07:00"
        until: "2026-10-16T11:42:10-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T11:06:23-07:00"
              end: "2026-10-16T11:42:10-07:00"
              began: "2026-10-16T11:06:23-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 16b58a8d329d21683c35edfe08ad1012a3e6bee6a23dc00ed0f55c781f5599a1
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 11841721
              windows:
                - start: "2026-10-16T11:06:23-07:00"
                  end: "2026-10-16T11:42:10-07:00"
        cuts:
            - dispatched: "2026-10-16T11:38:51-07:00"
              finalized: "2026-10-16T11:41:14-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T11:45:06-07:00"
      dispatched: "2026-10-16T11:43:21-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'continuous.go:171-175 sends each snapshot as its own request (n = 1 when collected). Removing that line makes TestCiphertextOverflowIsTheSentinelAndASnapshotSkip fail, where big.sqlite sorts before small.sqlite. Leftover nit: the skip note names only the kind, not fresh[0].'
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/history-backup.md:219 now states that a database changing daily is a new version every pass, about 115 MB per bundle or roughly 40 GB a year, that the sink never deletes, and that pruning is a pending retention decision.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: rawBundleLimit still calls Defaults(), which re-decodes sourcesJSON on every call (inventory.go:49), and plan.go:103 still uses MaxBundleBytes for the label RawBytes prototype.
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: bedbf0ba13f30d25b364f1feafcc5173b1970af5
      blocked: false
---

# Gate ledger — pair#616 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T11:41:14-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `partial-batch-failure-drops-siblings` (kind: defect) ErrCiphertextLimit skip in the continuous pass drops every other snapshot still pending in the same request
  Archiver.create returns at the first failing manifest (archive.go:85-88), and the new branch (continuous.go:177-183) then skips all n requested files with fresh = fresh[n:], so snapshots planned after the oversized bundle (sorted by month and path, so the same order every pass) are never archived, and the note does not name the file. Fix: archive snapshot roots one file per request, or retry the remaining files minus the failing manifest; put the path in the note; add a test with two snapshots where the oversized one sorts first.
- **BR-2** [Important] `growth-cost-unstated` (kind: defect) Daily archiving of thread_history_1.sqlite adds about 115 MB per pass to a never-deleting sink, and the atlas does not state the cost
  Before this change the 1.6 GB database was skipped. Now each content change writes a new ~115 MB bundle, roughly 40 GB a year, and bundles are kept until a future retention decision (atlas/history-backup.md:148). Per ARCH-FUNERAL, add the per-pass cost and what bounds it to the history-backup storage section or storage-retention-io.md.
- **BR-3** [Minor] `repeated-static-parse` (kind: defect) rawBundleLimit re-parses the built-in inventory on every call, including once per catalog record read
  Parse it once (sync.OnceValue). Also, planBundles at plan.go:103 should size the label-overhead estimate with rawBundleLimit(r.Kind) instead of MaxBundleBytes; it is only correct today because 8 GiB and 1 GiB have the same number of digits.

## Round 2 — 2026-10-16T11:45:06-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — continuous.go:171-175 sends each snapshot as its own request (n = 1 when collected). Removing that line makes TestCiphertextOverflowIsTheSentinelAndASnapshotSkip fail, where big.sqlite sorts before small.sqlite. Leftover nit: the skip note names only the kind, not fresh[0].
- BR-2 — addressed — atlas/history-backup.md:219 now states that a database changing daily is a new version every pass, about 115 MB per bundle or roughly 40 GB a year, that the sink never deletes, and that pruning is a pending retention decision.
- BR-3 — not-addressed — rawBundleLimit still calls Defaults(), which re-decodes sourcesJSON on every call (inventory.go:49), and plan.go:103 still uses MaxBundleBytes for the label RawBytes prototype.

## Open findings

- **BR-3** [Minor] `repeated-static-parse` (kind: defect) rawBundleLimit re-parses the built-in inventory on every call, including once per catalog record read
