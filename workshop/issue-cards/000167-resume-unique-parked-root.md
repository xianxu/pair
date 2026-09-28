---
id: '000167'
status: done
started: 2026-09-01T17:56:17-07:00
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 1.89
actual_hours: 0.97
---

# Resume unique parked root on Couch startup

## Problem

`Leave Couch` deliberately parks every active thread, including the home/root
actor. Starting `couch` again at the same repository currently creates a new
root, leaving the former root available only as a parked child in the panel.
The native conversation is recoverable, but its home role is not, and the
operator must create a disposable conversation just to reach it.
