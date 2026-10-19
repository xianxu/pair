---
id: 000709
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# landing: pilot landing trains in pair (60-minute interval)

## Problem

ariadne#449 (landed, PR 324) adds landing trains: ready merges in one repo are batched and tested with one recorded suite and
merged together, instead of one suite per change. Trains are off unless a repo has `.sdlc/landing.json`. The operator approved
a pilot in pair (2026-10-19): pair's suites are the long pole in the landing queue.
