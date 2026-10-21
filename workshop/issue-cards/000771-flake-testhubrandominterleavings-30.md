---
id: 000771
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
flake_tests: 
---

# flake: TestHubRandomInterleavings/30

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/broadcast.TestHubRandomInterleavings/30: Same exact integrated train234 head d6efc430bb4fbd8b1e943981688eec3914f8c1ca: recorded suite attempt88fb48b8719d5a57732691bd0fe8cb2f raw inflight run.log failed only broadcast.TestHubRandomInterleavings/30 at hub_test.go405 step4, pointer callback0 with armed=true shown=false. On that unchanged detached checkout, exact anchored seed30 count100 passed in1.126s; broader seeds30/130/230/330/430 count100 also passed5.203s. Underlying manual-clock/callback ordering is unproven; register only the evidenced subtest, not sibling seeds. Full suite continues without cancellation; no production source or timeout change.
