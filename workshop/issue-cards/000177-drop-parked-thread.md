---
id: 000177
status: open
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Drop a parked thread from the switcher

## Problem

**Threads accumulate and nothing removes them.** The operation set is
`prepare-start, start, list, show, stop, name, describe, publish-description,
switch, attach, park, detach, leave, resume` — none of which deletes a durable
thread record. The two near-misses do something else:

- `stop` is `ExecuteLiveOwner` + "signal an actor's child and forget it": it
  needs a live child, and its `Forget` mutates `c.reg` (the transitional
  live-handle cache), not ThreadStore. It cannot remove a durable record.
- `park --mode=abandon` abandons an **in-flight park transaction**
  (`park.go:367-376`, keyed on `current.Park.Identity` and the record revision),
  not a completed park.

Measured on this host 2026-09-02: **17 thread records, 12 of them parked** — 5
on `pair`, 4 on `brain`, 2 on `tools`, 2 on `kbench/competition/arc-agi-3`, 1 on
`parley.nvim`. Every one of them is a switcher row the operator scrolls past.

This is the **drain** side of `pair#175`'s ratchet: that issue is about creating
duplicate threads on a tree; this one is about there being no way to remove the
duplicates once created. Fixing only one leaves the panel filling up either way.

Removing one today means hand-editing the store — delete the record file,
delete its `manifest.json` entry, bump `generation` — while the supervisor is
live. That is exactly the second-writer situation the store's lock, revision
checks and write-ahead journal exist to prevent.
