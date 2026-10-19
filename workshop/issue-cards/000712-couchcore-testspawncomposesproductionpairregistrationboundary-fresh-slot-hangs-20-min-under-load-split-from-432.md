---
id: 000712
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestSpawnComposesProductionPairRegistrationBoundary]
---

# couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432)

## Problem

`couchcore.TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot` hung in pair#689's integrated suite (2026-10-19,
f2333e34): 1172.59 s, LaunchNative reported "terminal not listed live", the cleanup spawn goroutine failed to exit, then the
package hit its 20-minute timeout. The whole suite took 26 min (median 12.5). pair#432 records the same test "failing
identically on origin/main" (10-10) and asks to triage it separately; most pair suites since pass it, so it's intermittent,
probably load-sensitive. It now blocks a landing as an unattributable failure.
