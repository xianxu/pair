---
id: 000869
status: codecomplete
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
started: 2026-10-24T08:28:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 0.27
tracker:
    version: 1
    completion:
        token: close-7c30b1e6943b
        repository: github.com/xianxu/pair
        reviewed_head: be270eeb7be0d806d7aaa76fc18bf331e9ac69e7
        evidence_commit: a55eb8db8abd9d0761137603cb7d390852912836
---

# Publish cancellation fixture PID readiness atomically

## Problem

TestCLILiveLeaderCancellationJoinsBeforeCompletion intermittently parses an empty PID marker. First observed in #863 recorded suite attempt0c7a9818a957d392fc43e9fb44889545 on d00c828e28d41b317b924f3bde4f2f0a622076f7 (2026-10-24; suite exit2 in18m38s). cli_group_test.go:84: strconv.Atoi: parsing "": invalid syntax. Package duration20.261s. Six other failures in that attempt are #863 stale fixtures, not this bug.

The shell uses printf redirected directly to the final marker path. Its open publishes an empty file before printf writes the PID. waitFixture waits only for os.Stat success, then the test reads once and parses it, so a valid scheduling gap reaches this exact empty-string error. This is a fixture readiness race, not evidence of a cancellation-owner defect. #863 changes none of this package; Git diff from the failed head to the repeated head is empty for cmd/internal/testcouch/agentfixture.

One unchanged focused repeat passed (package0.130s, exit0). That is a targeted repeat, not a passing whole suite on the same commit; no known-flake registration is claimed.
