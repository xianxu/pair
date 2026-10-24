---
id: 000854
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# flake: TestRepoConfigurationPublicationBoundsCompetingWriter

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestRepoConfigurationPublicationBoundsCompetingWriter: Same-exact-head fail then focused pass on 954e3caea7078371c91c7523e86a70bdad751658. Recorded affected attempt01392f0a6d0a73ce07338d4fa6c68104 failed only this whole test/store at repopreferences_adoption_test.go417: expected deadline cause but warning was thread store busy;retry (store5.38s). Parser proven ancestor is whole top-level; no subtest-only registration. Exact detached HEAD verified by git, unchanged tree, standard runtimebundle preparation; go test -json ./cmd/internal/couchcore -run ^TestRepoConfigurationPublicationBoundsCompetingWriter$ -count=1 -timeout=90s then PASS10.562s, store5.07s and coordinator5.22s. JSON log /Volumes/TestScratch/pair841-leave-repro-gub_n4rh/841-repoprefs-repro.jsonl. Original failed log SHA prefix37b0a76723f5. #841 is bell display only and does not change repopreferences/store code. Focused diagnostic proves intermittency only, not a passing affected status; script groups did not run after original Go failure. Root cause needs independent repair, no timeout inflation or retry fix.
