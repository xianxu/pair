---
id: '000181'
status: done
started: 2026-09-03T16:38:51-07:00
created: 2026-09-03
updated: 2026-09-13
estimate_hours: 8.64
actual_hours: N/A
---

# One honest inventory: every thread gets a row and a reason

## Problem

The switcher shows 4 rows. The store holds 13 threads. `couch --list` shows all
13. Nothing reconciles those numbers, and nine threads are invisible with no
notice, no log line and no way to ask why.

Measured on the operator's live store, 2026-09-03:

```
state      resume proof        count
live       established           3     (only 2 are really hosted -- see stale, below)
detached   established           1     tools-couch-2   -- blocked from reattaching (pair#179)
detached   provisional/no-id     1     pair-couch-24   -- hidden (pair#168 lost its binding)
parked     established           1     parley          -- the ONLY resumable park
parked     provisional/no-id     8     brain x5, kbench x2, ariadne -- hidden AND unresumable
live(stale) --                   1     brain-couch-19  -- record says live, no console hosts it (pair#171)
```

**Root cause: "should this row exist" was fused with "can this row be acted
on", and the fused answer is computed half in an IO loop and half in a pure
projector, with `continue` as its only vocabulary for "no" (ARCH-PURE).**

The row set comes from one indexed directory:

```
threadstore/manifest.json  -> [{repo_scope, tag}, ...]
threadstore/records/<scope>/<tag>.json
```

`ThreadStore.Snapshot` (`threadstore.go:504-526`) walks the manifest. Three
proof sources are then joined on: live TTY observations (this console's own
children), the parked proof (pair's `repos/<scope>/ledger-<tag>.jsonl` via
`sessioninventory`), and the detached proof (zellij sessions + pair's
`session-names.jsonl`).

Rows then die in two places. `ActionableThreadInventoryContext:228-272` drops a
record BEFORE the pure function sees it -- reservation, park in flight, any
incarnation, no saved profile, unsupported agent, unphysicalizable path, no
resolver, and `bindingResumeDiagnostic != ""`, which alone accounts for 9 of the
13. Then `ProjectActionableThreads:101-142` / `actionableThreadState:144+` drops
more: invalid record, verified park without a matching parked proof, no
incarnation and no detached proof, an incarnation with no matching TTY
observation.

A pure function cannot be wrong about rows it never receives, and no test can
see the fused answer -- which is why this shipped through four milestones of
review.

Secondary, same root: **detached rows lie**. `menu_render.go:296-299` renders
anything not `Live()` as `parked · <age>`, so the operator's one visible
detached thread is labelled parked.
