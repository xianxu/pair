---
id: 000578
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Profile and reduce couchcore test runtime

## Problem

The 2026-10-15 `sdlc test-stats --since 7d` sample shows `cmd/internal/couchcore` at 481.3 seconds p90 over 21 runs, a major component of pair's 556/786-second suite p50/p90. #390's operator review deferred numeric budgets until weekly trends exist, but a cause-level performance investigation can begin from the measured bottleneck.
