---
id: 000597
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: critical
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).
