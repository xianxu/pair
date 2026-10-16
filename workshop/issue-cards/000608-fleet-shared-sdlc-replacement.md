---
id: 000608
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Isolate fleet conformance from shared installed SDLC replacement

## Problem

TestFleetInventoryLiveConformance executes the machine's shared installed sdlc from PATH. Another slot's dev alias can replace that binary while the suite is running. The conformance test can therefore fail due to unrelated tool publication rather than the pinned pair source.
