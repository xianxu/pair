---
id: 000852
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
severity: high
---

# Park rollback retries fail on main

## Problem

TestLeaveParkRetriesProvenRollbackWithinOneQuit fails in isolation on both the #841 executable head and pure origin/main. It also blocked #841's recorded affected check. This is a main regression requiring root-cause investigation; #841 changes bell display rendering and does not change park code.
