---
id: '000291'
status: done
started: 2026-09-20T14:05:57-07:00
created: 2026-09-19
updated: 2026-09-20
actual_hours: 1.02
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
