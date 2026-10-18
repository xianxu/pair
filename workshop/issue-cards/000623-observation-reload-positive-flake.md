---
id: 000623
status: working
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T18:13:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Review observation positive control intermittently loses external edit reload

## Problem

tests/review-observation-diagnostics-test.py intermittently fails its positive control, ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The nested real editor/Git fixture asserts: matching branch external edit did not reload; snapshot lost no-final-newline state.

Observed during #604's recorded rerun on fe4eb6d6e83bd53b9d42fcf27811a383c348b222, first seen here 2026-10-16 at about 16:01 PDT. Eight diagnostic cases ran in 63.905s; one failed. The full local-check stopped before Go tests, exit 2 in 6m59s after 1239s in the machine queue. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/ffdc28f65f4aad6e6fdf5f09c7feeb01.log, SHA256 prefix 7c703e6585f2.

The preceding complete make -k suite passed this diagnostic command and all other shell/editor/Go checks, with no production/test changes between attempts. Passing log: /tmp/pair604-keepgoing-suite.log, SHA256 79a1e5cb433c82e9af9207831d7deb750f9b8280a826238059423d03fc5b8624. One pass and one failure do not establish a calibrated rate.
