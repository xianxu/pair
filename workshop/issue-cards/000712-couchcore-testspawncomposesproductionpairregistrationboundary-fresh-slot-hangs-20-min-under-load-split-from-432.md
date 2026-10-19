---
id: 000712
status: working
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.9
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestSpawnComposesProductionPairRegistrationBoundary]
started: 2026-10-19T14:08:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432)

## Problem

`couchcore.TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot` hung in pair#689's integrated suite (2026-10-19,
f2333e34): 1172.59 s, LaunchNative reported "terminal not listed live", the cleanup spawn goroutine failed to exit, then the
package hit its 20-minute timeout. The whole suite took 26 min (median 12.5). pair#432 records the same test "failing
identically on origin/main" (10-10) and asks to triage it separately; most pair suites since pass it, so it's intermittent,
probably load-sensitive. It now blocks a landing as an unattributable failure.
