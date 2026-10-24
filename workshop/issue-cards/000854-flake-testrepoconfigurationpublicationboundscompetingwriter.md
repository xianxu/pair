---
id: 000854
status: codecomplete
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestRepoConfigurationPublicationBoundsCompetingWriter]
started: 2026-10-24T05:56:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-3efa649111e1
        repository: github.com/xianxu/pair
        reviewed_head: c4d4ed487292cadb4bb629d42381ae25406a8468
        evidence_commit: e86c55d8882cbe5718a9d1362526424c02f60efc
---

# flake: TestRepoConfigurationPublicationBoundsCompetingWriter

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestRepoConfigurationPublicationBoundsCompetingWriter: Same-exact-head fail then focused pass on 954e3caea7078371c91c7523e86a70bdad751658. Recorded affected attempt01392f0a6d0a73ce07338d4fa6c68104 failed only this whole test/store at repopreferences_adoption_test.go417: expected deadline cause but warning was thread store busy;retry (store5.38s). Parser proven ancestor is whole top-level; no subtest-only registration. Exact detached HEAD verified by git, unchanged tree, standard runtimebundle preparation; go test -json ./cmd/internal/couchcore -run ^TestRepoConfigurationPublicationBoundsCompetingWriter$ -count=1 -timeout=90s then PASS10.562s, store5.07s and coordinator5.22s. JSON log /Volumes/TestScratch/pair841-leave-repro-gub_n4rh/841-repoprefs-repro.jsonl. Original failed log SHA prefix37b0a76723f5. #841 is bell display only and does not change repopreferences/store code. Focused diagnostic proves intermittency only, not a passing affected status; script groups did not run after original Go failure. Root cause needs independent repair, no timeout inflation or retry fix.
