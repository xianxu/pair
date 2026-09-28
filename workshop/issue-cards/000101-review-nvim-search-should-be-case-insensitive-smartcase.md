---
id: '000101'
status: done
started: 2026-07-06T14:46:52-07:00
created: 2026-07-05
updated: 2026-07-06
estimate_hours: 0.25
actual_hours: 0.20
---

# review nvim search should be case-insensitive (smartcase)

## Problem

Search in the review nvim pane is case-sensitive. It should be case-insensitive
by default but case-sensitive when the query contains an uppercase letter — the
"smart" option. Split out of #89 to keep that issue focused on concurrent-edit
reconciliation.
