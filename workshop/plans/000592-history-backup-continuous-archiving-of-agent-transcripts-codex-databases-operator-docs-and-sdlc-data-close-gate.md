---
gate: boundary-review
issue: 592
id_prefix: BR
rounds:
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 43870216-4b9a-4345-aa1b-652a37d8d2b6
            repository: github.com/xianxu/pair
            issue_id: "000592"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/cffd7eee317b1527
            reviewed_head: 7c73505199bdb15d0a4f1091c7f0fc03c9b0c227
            slot: pair:2
            requested_at: "2026-10-16T16:56:59.956655Z"
          phase: finished
          pid: 30716
          started_at: "2026-10-16T16:57:00.215627Z"
          finished_at: "2026-10-16T16:59:52.854594Z"
          session_id: 006a5383-263d-4b31-a41e-b4d98b71b637
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 7c73505199bdb15d0a4f1091c7f0fc03c9b0c227
        tracker_head: 9a7d5cc46488cd9a0f88a00332315381e7481ad6
        branch_point: da3d07a086a7db606877165aa5259d3093b8fa71
        since: "2026-10-16T08:50:30-07:00"
        until: "2026-10-16T09:27:56-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T08:50:30-07:00"
              end: "2026-10-16T09:27:56-07:00"
              began: "2026-10-16T08:50:30-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: eedfaefa2e95870794a1a8197bc3939b77cee7d85f8b192d1cc2ca66cd5c108a
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 11024432
              windows:
                - start: "2026-10-16T08:50:30-07:00"
                  end: "2026-10-16T09:27:56-07:00"
      "n": 1
      timestamp: "2026-10-16T10:00:09-07:00"
      dispatched: "2026-10-16T09:56:55-07:00"
      agent: claude
      findings:
        - kind: defect
          id: BR-1
          severity: Important
          title: README not updated for pair archive continuous; README.md:1734 still says no schedule or SQLite snapshot
          detail: New subcommand, config file continuous.json and Couch daily schedule are user-facing surface; README still disclaims scheduling and live SQLite snapshots. Update the README archive paragraph.
          family: docs-gate-new-surface
          round: 1
        - kind: defect
          id: BR-2
          severity: Important
          title: Live pass shows no catalog entries for ops-documents and sdlc-runtime, which Done-when requires
          detail: The Log records 0 archived for both kinds (files newer than 72h). Re-run after files settle and record the entries, or add a Revisions entry amending Done-when.
          family: done-when-live-evidence
          round: 1
        - kind: defect
          id: BR-3
          severity: Important
          title: Over-limit databases are snapshotted every pass, and sameContent loads both files into memory
          detail: continuous.go archiveRoot calls Snapshot on every sqlite file before any MaxBundleBytes check, so the 1.5 GB thread_history database is copied, backed up and integrity-checked daily, then refused by Describe. sqlite_snapshot.go sameContent uses os.ReadFile on both files (heap up to 2x DB size). Check DBSize+WALSize against the limit after observeDatabase, and compare by streaming or size first.
          family: bound-check-before-expensive-work
          round: 1
        - kind: defect
          id: BR-4
          severity: Important
          title: Snapshot scratch work dirs have no sweep; a killed pass leaves multi-GB private copies
          detail: sqlite-* dirs under archive-snapshots/.scratch are removed only by defer. SIGKILL or a crash during a copy leaves them forever. Clear Scratch at pass start while holding continuous.lock.
          family: durable-residue-removal-path
          round: 1
        - kind: defect
          id: BR-5
          severity: Minor
          title: First archive error aborts all remaining kinds; scheduler retries the full pass hourly without backoff
          family: retry-backoff-policy
          round: 1
        - kind: defect
          id: BR-6
          severity: Minor
          title: continuous.log is appended forever with no rotation or size bound
          family: durable-residue-removal-path
          round: 1
        - kind: defect
          id: BR-7
          severity: Minor
          title: sdlc-runtime identity Worktree is whichever workspace child is found first; a new sibling re-keys everything
          family: stable-source-identity
          round: 1
        - kind: defect
          id: BR-8
          severity: Minor
          title: No integration test for a changed quiet transcript archived as a new version (Plan Task 5)
          family: test-plan-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
disputes:
    - id: BR-2
      rationale: 'The live pass covered both kinds; it archived 0 because no file in them has yet been unmodified for 72 h, which is the approved rule (TL 82ac1242: quiet for 3 days). Forcing entries would need a quiet override on the CLI, new surface the design does not want. The daily Couch pass will archive them as files settle.'
      evidence: 'Measured at the live pass (2026-10-16 ~09:30 PDT): ops teams/*/qa, teams/*/assets, teams/*/tl-status-*.side: 104 files, 0 older than 72 h (find -mmin +4320); every workspace repo git-common sdlc/: 388 files, 0 older than 72 h. Discovery and roots for both kinds are covered by TestContinuousRootsOpsTeamsAndSharedGitCommonDirs (ops per team; one sdlc root per git common dir) and the pass test.'
      after: 1
      timestamp: "2026-10-16T17:01:40Z"
      by: Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair)
---

# Gate ledger — pair#592 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-16T10:00:09-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-gate-new-surface` (kind: defect) README not updated for pair archive continuous; README.md:1734 still says no schedule or SQLite snapshot
  New subcommand, config file continuous.json and Couch daily schedule are user-facing surface; README still disclaims scheduling and live SQLite snapshots. Update the README archive paragraph.
- **BR-2** [Important] `done-when-live-evidence` (kind: defect) Live pass shows no catalog entries for ops-documents and sdlc-runtime, which Done-when requires
  The Log records 0 archived for both kinds (files newer than 72h). Re-run after files settle and record the entries, or add a Revisions entry amending Done-when.
- **BR-3** [Important] `bound-check-before-expensive-work` (kind: defect) Over-limit databases are snapshotted every pass, and sameContent loads both files into memory
  continuous.go archiveRoot calls Snapshot on every sqlite file before any MaxBundleBytes check, so the 1.5 GB thread_history database is copied, backed up and integrity-checked daily, then refused by Describe. sqlite_snapshot.go sameContent uses os.ReadFile on both files (heap up to 2x DB size). Check DBSize+WALSize against the limit after observeDatabase, and compare by streaming or size first.
- **BR-4** [Important] `durable-residue-removal-path` (kind: defect) Snapshot scratch work dirs have no sweep; a killed pass leaves multi-GB private copies
  sqlite-* dirs under archive-snapshots/.scratch are removed only by defer. SIGKILL or a crash during a copy leaves them forever. Clear Scratch at pass start while holding continuous.lock.
- **BR-5** [Minor] `retry-backoff-policy` (kind: defect) First archive error aborts all remaining kinds; scheduler retries the full pass hourly without backoff
- **BR-6** [Minor] `durable-residue-removal-path` (kind: defect) continuous.log is appended forever with no rotation or size bound
- **BR-7** [Minor] `stable-source-identity` (kind: defect) sdlc-runtime identity Worktree is whichever workspace child is found first; a new sibling re-keys everything
- **BR-8** [Minor] `test-plan-coverage` (kind: defect) No integration test for a changed quiet transcript archived as a new version (Plan Task 5)

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-2** — after round 1 (2026-10-16T17:01:40Z, Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair))
  rationale: The live pass covered both kinds; it archived 0 because no file in them has yet been unmodified for 72 h, which is the approved rule (TL 82ac1242: quiet for 3 days). Forcing entries would need a quiet override on the CLI, new surface the design does not want. The daily Couch pass will archive them as files settle.
  evidence: Measured at the live pass (2026-10-16 ~09:30 PDT): ops teams/*/qa, teams/*/assets, teams/*/tl-status-*.side: 104 files, 0 older than 72 h (find -mmin +4320); every workspace repo git-common sdlc/: 388 files, 0 older than 72 h. Discovery and roots for both kinds are covered by TestContinuousRootsOpsTeamsAndSharedGitCommonDirs (ops per team; one sdlc root per git common dir) and the pass test.

## Open findings

- **BR-1** [Important] `docs-gate-new-surface` (kind: defect) README not updated for pair archive continuous; README.md:1734 still says no schedule or SQLite snapshot
- **BR-3** [Important] `bound-check-before-expensive-work` (kind: defect) Over-limit databases are snapshotted every pass, and sameContent loads both files into memory
- **BR-4** [Important] `durable-residue-removal-path` (kind: defect) Snapshot scratch work dirs have no sweep; a killed pass leaves multi-GB private copies
- **BR-5** [Minor] `retry-backoff-policy` (kind: defect) First archive error aborts all remaining kinds; scheduler retries the full pass hourly without backoff
- **BR-6** [Minor] `durable-residue-removal-path` (kind: defect) continuous.log is appended forever with no rotation or size bound
- **BR-7** [Minor] `stable-source-identity` (kind: defect) sdlc-runtime identity Worktree is whichever workspace child is found first; a new sibling re-keys everything
- **BR-8** [Minor] `test-plan-coverage` (kind: defect) No integration test for a changed quiet transcript archived as a new version (Plan Task 5)
