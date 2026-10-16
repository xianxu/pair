---
id: 000615
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Review lanes for couchcore remaining test budget

## Problem

After #611 R2 landed (PR371, merge24fe0959f4eaa6577c0d28e207953c899c2944e3), the requested main control still takes263.916s, above the hub's approximately200s threshold for bringing the lane proposal back for review. The final landing suite passed with couchcore357.969s. #610's temporary20m allowance remains unchanged. This is a separate capacity decision after the cost reductions, not approval to add a runner.
