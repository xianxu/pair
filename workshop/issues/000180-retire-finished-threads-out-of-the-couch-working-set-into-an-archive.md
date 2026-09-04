---
id: 000180
status: open
deps: [pair#168]
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
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

## Spec

After pair#168 restores what it can, a thread is RETIRED when it is finished,
not merely when it is unusable:

- no live incarnation, no live zellij session, and
- either its park is verified and its native transcript is gone (nothing left
  to resume), or it never started (a rolled-back or abandoned reservation).

A verified park whose transcript still exists is NOT retired, however old --
that is exactly the "I left something here" case couch exists to hold.

Retired records move to `threadstore/archive/` keeping their layout, so the
same reader can inspect them and a mistake is reversible by moving the file
back. The switcher, `couch --list` and the startup resume selector read only
the working set.

Open, to settle in design:

1. When retirement runs -- at startup, on a `couch prune` verb the operator
   invokes, or continuously. Prefer an explicit verb first: automatic deletion
   of state the operator has not seen is how a recoverable session becomes an
   unrecoverable one.
2. Whether couch may retire Pair's per-tag artifacts (`pair/repos/<scope>/`).
   They are the bulk of the clutter but they are pair's surface, and pair must
   keep working standalone -- so this may belong in a pair-side issue with its
   own rules about what a live pair session still needs.
3. Whether the thread-health view this issue was measured with (state x binding
   x last-active per record) becomes a real `couch doctor` / `couch --show`
   column. It answered "why is this thread invisible" in one line and there is
   no other way to ask.

## Done when

- The switcher, `couch --list` and startup selection see only live, parked and
  detached threads plus verified parks that can still be resumed.
- A retired record is inspectable under `threadstore/archive/` and restoring it
  is a file move.
- A verified park whose transcript survives is never retired, proved by test.
- Retirement is measured against the operator's real store before and after,
  and the count of records it removes is reported rather than assumed.

## Plan

- [ ] Blocked: land pair#168 first, then re-measure the real store. The
      retirement rule is written against the post-fix population, not this one.
- [ ] Pure `RetireThread` predicate over a record + its session/transcript
      evidence, unit-tested across the state matrix (ARCH-PURE).
- [ ] Archive move + reader that can list the archive without loading it into
      the working set.
- [ ] Operator-invoked verb first; decide automatic retirement separately.
- [ ] Decide the Pair per-tag artifact question -- likely a separate pair-side
      issue rather than couch reaching into pair's directory.

## Log

### 2026-09-03 — shipped as pair#181 M3, as an action rather than a predicate

Absorbed rather than worked separately, and deliberately narrowed.

This issue specified a `DecideRetirement` predicate over the reason vocabulary,
plus a `couch prune` verb. Neither was built. The operator's instruction was to
add a delete action instead -- "remove it from the system, so that we can start
anew" -- and they were right: an operator action beats a rule guessing what is
finished on their behalf, and the guessing was the retirement matrix.

What shipped is `archive`: a declared operation, confirmed, offered on every row
couch is not hosting, which stops the thread's session and then moves its record
to `threadstore/archive/<scope>/<tag>.json` in one journal entry. `couch
--archived` lists what was retired.

Two things the Spec did not anticipate. Park cannot do the stopping -- it needs
a live incarnation, which the debris this exists for does not have -- so the
mechanism is `Artifacts.Quiesce` a layer down. And the record move alone leaves
a running agent nothing tracks; the one-time cleanup produced exactly one such
orphan before that was fixed.

The Spec's premise was also overturned by measurement: it assumed the working
set held finished threads to retire. It held recoverable ones. The operator
judged them corrupted data and the one-time cleanup archived 10, leaving three
live threads, one per repo.

### 2026-09-03

- Filed from the operator's request to keep the hand-inspected store clean.
  Measurements above are from the live store, and they inverted the request:
  the population that looks dead is mostly recoverable, so this waits on
  pair#168.
