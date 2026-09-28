---
id: '000168'
status: punt
created: 2026-09-01
updated: 2026-09-03
---

# Preserve parked binding after failed resume

## Problem

Resuming a verified parked Couch thread can fail after Pair appends a new
launch row but before it appends the corresponding binding. The session ledger
then treats that newest, incomplete launch as current and no longer projects
the earlier established binding. The thread disappears from Couch's resumable
inventory even though its verified park record and native agent transcript are
still intact and resumable.

Observed on `couch-71d653a6fc8f615d`: two attempts rebound Codex session
`01a05f9d-76cf-7ca3-8133-f21eeb3e0798`; a later attempt left only a launch
row. The exact owner query became provisional/empty and hid the parked thread.
