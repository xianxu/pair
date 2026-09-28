---
id: '000180'
status: done
created: 2026-09-03
updated: 2026-09-04
actual_hours: N/A
---

# Retire finished threads out of the couch working set into an archive

## Problem

The operator inspects couch's store by hand to change per-repo configs, and
asks that the inventory hold only threads in a usable state -- live, parked or
detached -- with finished ones moved to an archive that is inspectable but out
of the way.

Two corrections from measuring the live store first (2026-09-03):

**`threadstore/path-preferences/` is already clean.** Six files, one per
physical repo path, each `{repo_identity, physical_path, last_agent,
argv_by_agent}`. No thread data, nothing stale. That is the directory named in
the request, and it needs no work.

**The clutter is in two other places, and one of them is not couch's.**

| location | holds | scale |
| --- | --- | --- |
| `couch/threadstore/records/` | one file per thread record | 13 records |
| `pair/repos/<scope>/` | per-TAG artifacts: `agent-*`, `ledger-*`, `config-*`, `workbench-layout-*`, `adapt-*` | 728 files for 31 couch tags in the pair scope alone, against 2 surviving records |

The per-tag artifacts are Pair's, not couch's, and they are never retired --
every thread ever launched leaves five or so files behind forever. Couch minted
66 thread names across five repos; 13 records survive.

**The blocker: today's "dead" threads are mostly NOT finished, they are
LOST.** Measured resumability of all 13 surviving records:

```
state      binding             count
live       established           3
detached   established           1     tools-couch-2   (blocked by pair#179)
detached   provisional/no-id     1     pair-couch-24   (hidden by pair#179)
parked     established           1     parley          (the only resumable park)
parked     provisional/no-id     8     brain x5, kbench x2, ... (hidden AND unresumable)
```

Nine of thirteen are invisible to the switcher and refused by resume. The cause
is pair#168 in every case -- a trailing `launch` ledger row with no `binding`
row after it, which shadows the earlier established binding:

```
couch-1539ce935d4238b7  legacy launch binding legacy launch binding legacy launch binding legacy launch binding   <- resumable
couch-e78b962be29c4d9a  legacy launch binding legacy launch binding legacy launch binding launch                  <- trailing launch, LOST
couch-05156384da12af64  legacy launch                                                                             <- never bound, LOST
```

So archiving "dead" threads now would bury nine recoverable sessions and hide
the bug that broke them. **This issue is blocked on pair#168**, and its
retirement rule must be written against an inventory whose states are honest.
