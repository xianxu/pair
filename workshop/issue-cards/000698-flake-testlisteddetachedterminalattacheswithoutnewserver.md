---
id: 000698
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# flake: TestListedDetachedTerminalAttachesWithoutNewServer

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer: failed in attempt adbcdbbf22e41d689a246f916f9dd2d2, passed on the same head in attempt b8cfedebf5403d1c880f0aa2a5e89ccd
