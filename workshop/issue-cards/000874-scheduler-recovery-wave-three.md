---
id: 000874
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
---

# Scheduler recovery wave-three gate

## Problem

The operator approved ariadne#564 with three recovery waves after a full shutdown: pending-work asleep slots, remaining asleep slots, then the scheduled-task system. Scheduled tasks belong to the one Couch (pair#737), while each hub can report only its own worker-wave completion. A hub enabling automation on the first operator message would resume scheduled work before the machine-wide recovery barrier.
