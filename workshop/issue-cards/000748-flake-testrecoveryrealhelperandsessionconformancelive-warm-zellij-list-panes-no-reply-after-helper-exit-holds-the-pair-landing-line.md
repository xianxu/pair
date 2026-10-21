---
id: 000748
status: codecomplete
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T17:33:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 1.33
tracker:
    version: 1
    completion:
        token: close-dbf437090452
        repository: github.com/xianxu/pair
        reviewed_head: 8ebb557f6406d40da5512e16a1e320daae225433
        evidence_commit: bb1a052e8580727712755f71c3a21c2b73e8fdb4
---

# flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line

## Problem

`couchcore.TestRecoveryRealHelperAndSessionConformanceLive/warm` in the required `couch-zellij-conformance` job fails
intermittently on GitHub macOS: "recovery session state could not be checked; retry: observe live session owner:
zellij action list-panes --json --command returned no reply" (helper exited before recovery; the final query got no
reply). It failed on pair#706's heads twice on 10-20 (run 39501347923 attempt 1; run 39521409698 job 118965862227,
log SHA256 cd823c3894b2ef291a8c4ae4be3b0c70c062aa9ada5fa3d8b436538e393c4f69) and passed on rerun once; #706 touches
no couchcore, launcher or zellij code. Registered in pair#636. Because conformance is required, each occurrence
holds the pair landing line.
