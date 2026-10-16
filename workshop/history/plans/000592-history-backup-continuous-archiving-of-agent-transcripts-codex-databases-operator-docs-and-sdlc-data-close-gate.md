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
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 954c8538-4de5-4968-8da0-e2f123554b56
            repository: github.com/xianxu/pair
            issue_id: "000592"
            recipe: milestone-review
            round: 2
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a4461b8e13b9ea58
            reviewed_head: 1cdc2240281c07a0173ce1ae732f38e8b5465992
            slot: pair:2
            requested_at: "2026-10-16T17:03:13.234534Z"
          phase: finished
          pid: 94943
          started_at: "2026-10-16T17:03:13.620214Z"
          finished_at: "2026-10-16T17:07:33.215624Z"
          session_id: f468242f-fb5f-43fc-bfa1-bf966c674f0d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 1cdc2240281c07a0173ce1ae732f38e8b5465992
        tracker_head: f7e79397438562fb21f4be6a4d2e69773588615a
        branch_point: da3d07a086a7db606877165aa5259d3093b8fa71
        since: "2026-10-16T08:50:30-07:00"
        until: "2026-10-16T10:01:45-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T08:50:30-07:00"
              end: "2026-10-16T10:01:45-07:00"
              began: "2026-10-16T08:50:30-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 15f514290997f510c96426817b6c37034d59a2f509f58e061b0371876a986516
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 11131860
              windows:
                - start: "2026-10-16T08:50:30-07:00"
                  end: "2026-10-16T10:01:45-07:00"
        cuts:
            - dispatched: "2026-10-16T09:56:55-07:00"
              finalized: "2026-10-16T10:00:09-07:00"
              finalizing: false
      "n": 2
      timestamp: "2026-10-16T10:07:36-07:00"
      dispatched: "2026-10-16T10:03:08-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1734 now documents pair archive continuous, continuous.json, the 3-day quiet rule, coherent snapshots and the Couch daily run; matches continuous.go/archive_schedule.go.
          round: 2
        - id: BR-2
          disposition: withdrawn
          note: 'Re-measured on host: ops teams/devinfra 106 files, 0 older than 4320 min; workspace .git/sdlc 398 files, 0 older; Discover descends nested dirs, so the 72 h rule rather than a coverage gap explains 0 entries.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: sqlite_snapshot.go:80 bound check precedes MkdirTemp/copy; reverting it turns TestSQLiteSnapshotSkipsOverLimitDatabaseWithoutCopying red; sameContent streams 1 MiB chunks (TestSameContentStreams).
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: Sweep helper is tested, but the Run hook at continuous.go:67 is reached only via a duck-typed assertion that no test fixture satisfies; disabling it leaves all tests green. Put Sweep in the Snapshotter interface and assert the pass sweeps a planted sqlite-* dir.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: Unchanged in this window.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: Unchanged in this window.
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: Unchanged in this window.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: Third pass covers a newly-settled transcript and a changed DB, not a changed quiet transcript re-archived as a new version.
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 26a73bf1-bae9-4f03-93bf-2d9406d4a2e9
            repository: github.com/xianxu/pair
            issue_id: "000592"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/5d8881827cf71658
            reviewed_head: 37776a0518eeb935dd23684bc02f352ca0047b76
            slot: pair:2
            requested_at: "2026-10-16T17:10:45.166664Z"
          phase: finished
          pid: 85941
          started_at: "2026-10-16T17:10:45.425132Z"
          finished_at: "2026-10-16T17:14:00.103692Z"
          session_id: c42b5987-136c-4388-b7b9-0906075dfac2
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 37776a0518eeb935dd23684bc02f352ca0047b76
        tracker_head: f7e79397438562fb21f4be6a4d2e69773588615a
        branch_point: da3d07a086a7db606877165aa5259d3093b8fa71
        since: "2026-10-16T08:50:30-07:00"
        until: "2026-10-16T10:09:31-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-16T08:50:30-07:00"
              end: "2026-10-16T10:09:31-07:00"
              began: "2026-10-16T08:50:30-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: claude
              id: 9073df29-d226-4140-b23c-4f03f2b4c472
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 5401eee6e60b4ea9c077f4f6a772278914f338b5cb16deb74ac5080aad0788bc
              started: "2026-10-16T03:47:36.231Z"
              spawned: false
              bytes: 11238087
              windows:
                - start: "2026-10-16T08:50:30-07:00"
                  end: "2026-10-16T10:09:31-07:00"
        cuts:
            - dispatched: "2026-10-16T09:56:55-07:00"
              finalized: "2026-10-16T10:00:09-07:00"
              finalizing: false
            - dispatched: "2026-10-16T10:03:08-07:00"
              finalized: "2026-10-16T10:07:36-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-16T10:14:05-07:00"
      dispatched: "2026-10-16T10:10:41-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: Sweep is on the Snapshotter interface and runs under continuous.lock (continuous.go:69-75); the production Sweep is tested (sqlite_snapshot_test.go:121) and the integration test checks a planted sqlite-killed dir is removed by Run.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: Backoff is fixed and tested (archive_schedule_test.go:39-49). The errors.Join keep-going change has no test making one source fail and checking the others still archive with no last-run marker.
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: O_TRUNC is correct by inspection (archive_schedule.go:43), but the launch closure is never run by a test, so nothing fails without the change; make the log opener injectable and pin it.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: 'Dropping Worktree is correct, but continuous_discover_test uses two separate repos and never checks identity fields; restoring Worktree: dir passes every test. Add a git-worktree sibling case that asserts one root with an empty Worktree.'
          round: 3
        - id: BR-8
          disposition: addressed
          note: Fourth pass in continuous_external_test.go:104-110 edits an already-archived quiet transcript and checks it is archived again exactly once.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 37776a0518eeb935dd23684bc02f352ca0047b76
      blocked: false
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

## Round 2 — 2026-10-16T10:07:36-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — README.md:1734 now documents pair archive continuous, continuous.json, the 3-day quiet rule, coherent snapshots and the Couch daily run; matches continuous.go/archive_schedule.go.
- BR-2 — withdrawn — Re-measured on host: ops teams/devinfra 106 files, 0 older than 4320 min; workspace .git/sdlc 398 files, 0 older; Discover descends nested dirs, so the 72 h rule rather than a coverage gap explains 0 entries.
- BR-3 — addressed — sqlite_snapshot.go:80 bound check precedes MkdirTemp/copy; reverting it turns TestSQLiteSnapshotSkipsOverLimitDatabaseWithoutCopying red; sameContent streams 1 MiB chunks (TestSameContentStreams).
- BR-4 — not-addressed — Sweep helper is tested, but the Run hook at continuous.go:67 is reached only via a duck-typed assertion that no test fixture satisfies; disabling it leaves all tests green. Put Sweep in the Snapshotter interface and assert the pass sweeps a planted sqlite-* dir.
- BR-5 — not-addressed — Unchanged in this window.
- BR-6 — not-addressed — Unchanged in this window.
- BR-7 — not-addressed — Unchanged in this window.
- BR-8 — not-addressed — Third pass covers a newly-settled transcript and a changed DB, not a changed quiet transcript re-archived as a new version.

## Round 3 — 2026-10-16T10:14:05-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — Sweep is on the Snapshotter interface and runs under continuous.lock (continuous.go:69-75); the production Sweep is tested (sqlite_snapshot_test.go:121) and the integration test checks a planted sqlite-killed dir is removed by Run.
- BR-5 — not-addressed — Backoff is fixed and tested (archive_schedule_test.go:39-49). The errors.Join keep-going change has no test making one source fail and checking the others still archive with no last-run marker.
- BR-6 — not-addressed — O_TRUNC is correct by inspection (archive_schedule.go:43), but the launch closure is never run by a test, so nothing fails without the change; make the log opener injectable and pin it.
- BR-7 — not-addressed — Dropping Worktree is correct, but continuous_discover_test uses two separate repos and never checks identity fields; restoring Worktree: dir passes every test. Add a git-worktree sibling case that asserts one root with an empty Worktree.
- BR-8 — addressed — Fourth pass in continuous_external_test.go:104-110 edits an already-archived quiet transcript and checks it is archived again exactly once.

## Disputes

Findings the implementer declined (`sdlc finding dispute`), each answered by the
next review round.

- **BR-2** — after round 1 (2026-10-16T17:01:40Z, Xian Xu on Xian’s MacBook Pro at pair:2 (/Users/xianxu/workspace/worktree/pair-slot2/pair))
  rationale: The live pass covered both kinds; it archived 0 because no file in them has yet been unmodified for 72 h, which is the approved rule (TL 82ac1242: quiet for 3 days). Forcing entries would need a quiet override on the CLI, new surface the design does not want. The daily Couch pass will archive them as files settle.
  evidence: Measured at the live pass (2026-10-16 ~09:30 PDT): ops teams/*/qa, teams/*/assets, teams/*/tl-status-*.side: 104 files, 0 older than 72 h (find -mmin +4320); every workspace repo git-common sdlc/: 388 files, 0 older than 72 h. Discovery and roots for both kinds are covered by TestContinuousRootsOpsTeamsAndSharedGitCommonDirs (ops per team; one sdlc root per git common dir) and the pass test.

## Open findings

- **BR-5** [Minor] `retry-backoff-policy` (kind: defect) First archive error aborts all remaining kinds; scheduler retries the full pass hourly without backoff
- **BR-6** [Minor] `durable-residue-removal-path` (kind: defect) continuous.log is appended forever with no rotation or size bound
- **BR-7** [Minor] `stable-source-identity` (kind: defect) sdlc-runtime identity Worktree is whichever workspace child is found first; a new sibling re-keys everything
