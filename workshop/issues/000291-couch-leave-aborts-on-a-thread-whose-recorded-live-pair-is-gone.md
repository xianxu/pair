---
id: 000291
status: open
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
---

# Couch leave aborts on a thread whose recorded-live Pair is gone

## Problem

Operator report, 2026-09-19, during `pair#284`'s smoke setup. Alt+d in the
switcher (leave Couch, detaching every live thread) stopped partway:

```
error: leave couch: detach couch-361d9f1b7e2eb70c: thread {RepoScope:5229d7c0c566d658
Tag:couch-361d9f1b7e2eb70c} has no live Pair session to detach from
```

brain, tools and xianxu.dev were detached; parley.nvim, ariadne, pair, 42shots,
kbench and astro stayed live, and Couch stayed up. Pressing Alt+d again would
fail the same way, because `kaggle` comes first in the snapshot.

The thread is `kaggle`. `couch --show` reports `recorded: live pid 90411` and
`unusable: session gone`, and pid 90411 does not exist (`ps -p`). The record
still claims a live incarnation whose process and session are both gone. The
switcher's classifier already knows this: the row renders dim as
`session gone` (`ReasonSessionGone`).

Mechanism (code reading):
- `Couch.Leave` (`couchcore/park.go`) chooses what to detach from the RECORD
  alone. One incarnation in state `IncarnationLive` qualifies.
- `Couch.Detach` (`couchcore/detach.go`) observes the session before sending
  any signal, and refuses with a plain `fmt.Errorf` when it is absent. Nothing
  has been mutated at that point.
- `Leave` returns on the first detach error, so one stale record strands every
  thread after it. Yet `LeaveResult.Skipped` and `Console.reportLeave` exist
  exactly to report threads Couch "could not prove detachable" and left
  untouched.

## Spec

To be designed. Direction to test first: `Detach` types its pre-effect
refusals. Examples are no session, an open park, and no live incarnation.
`Leave` files a typed refusal under `Skipped` and continues. An error after
the signal still aborts, because that state is uncertain and must not collapse
into "skipped" (ARCH-ORDER). Separately, it is still open whether the stale
live incarnation itself (dead pid, session gone) should be reconciled to a
non-live state, and where. `pair#288` (dead launch is invisible) is adjacent.

## Done when

- Leave with a thread whose recorded Pair process and session are gone detaches
  every other live thread, reports the stale one as skipped, and exits Couch.
- An error after a detach signal still stops leave and names the thread.
- A test drives leave across a stale record placed before live ones.

## Plan

- [ ] Design with the operator: skip-and-report alone, or also reconcile the
      stale incarnation.

## Log

### 2026-09-19

- Filed from the operator's screenshot. Workaround used meanwhile: Tab → archive
  on the `kaggle` row (reversible), then Alt+d again.
