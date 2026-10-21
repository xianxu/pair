---
id: 000767
status: codecomplete
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestProvisionFixtureIdentityUsesOneGitProcess]
started: 2026-10-21T05:33:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 2.28
tracker:
    version: 1
    completion:
        token: close-11461e019396
        repository: github.com/xianxu/pair
        reviewed_head: 458897c5274b5313c32017e02f4effb6b73024cf
        evidence_commit: 0c73fd04a3e02e01e108a0b72de0e042be3410fb
---

# flake: TestProvisionFixtureIdentityUsesOneGitProcess

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestProvisionFixtureIdentityUsesOneGitProcess: Same exact head ecef5f7af3515fead7c02d6464ae112b16498fe1: recorded full-suite attempt53424823b0e2ed3099f5dab5c5a415b2 failed only TestProvisionFixtureIdentityUsesOneGitProcess at provision_identity_fixture_test.go107 exit42 (raw log .git/sdlc/test-stats/failed/53424823b0e2ed3099f5dab5c5a415b2.log). After failure, go test ./cmd/internal/couchcore -run ^TestProvisionFixtureIdentityUsesOneGitProcess$ -count=1 -v passed; independent count10 passed10/10 in3.062s. Exit42 is exclusively the shim test -r /dev/fd/3 gate before realGit; Lease forwarding and test unchanged from origin/main. Underlying transient child-FD readability cause remains unproven; add child fstat/fcntl diagnostics before a fix.
