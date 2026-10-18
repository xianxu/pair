---
id: 000670
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# couch --notify never reaches the desktop; pair review --help fails on the bare group

## Problem

Two operator-facing gaps, both seen 2026-10-18:

1. `couch --notify` never reaches the operator as a desktop notification. The hub has sent about ten today (each new Needs-you item); the operator saw none, including a test titled "Couch test". Couch reports "Notification written to the operator terminal."
2. `pair review --help` fails with `pair-go: unknown command "review"`, though `pair review apply --help` and `pair review round --help` work: review is registered only as two-word subcommands.
