---
id: 000871
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
---

# Land the opt-in launch-only Couch profiler independently

## Problem

The live Couch CPU investigation (#865) needs an opt-in profile on the operator's next separately approved restart. Land its already implemented diagnostic independently so scaled synthetic controls and optimization do not hold it up. TL ops:0 explicitly requested this extraction; pair:7 owns implementation/landing from main.
