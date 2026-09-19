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

**Direction (operator, 2026-09-19).** Global detach means: disconnect Couch,
leave every running instance running. Threads that are not live are fine —
parked ones already are. A thread recorded live that turns out to be dead is
also fine: Couch marks that state (session lost) and carries on with the
disconnect. **No error, and no report.** Couch and Pair wrap a coding agent, so
an operator who wants to know why a thread died can ask the agent to
investigate after the fact; the leave path does not owe them a diagnosis.

So the fix is not "skip and report". It is:

- Leave stops choosing what to detach from the record alone. When the thread's
  Pair session is absent, or its recorded `{PID, identity}` is proved Dead,
  Leave retires that incarnation and moves on to the next thread. The record
  then carries no incarnation, which is the state the classifier already
  renders as `session gone` (`ReasonSessionGone`) — no new vocabulary.
- `clearLifecycleDebris` (`couchcore/lifecycledebris.go`) is that proof and that
  write, already used by detach, resume and switch-agent: it screens the exact
  identity with `observeExactProcess` and retires through
  `RetireIncarnation` / `RetireUnprovenIncarnation` (ARCH-DRY). Leave should
  reuse it, not re-derive the rule.
- A live process whose session is gone is left alone — running instances are
  preserved, which is the point of detach.
- An error AFTER a detach signal still stops leave and names the thread: that
  state is uncertain and must not collapse into "handled" (ARCH-ORDER).
- `LeaveResult.Skipped` keeps its meaning for threads Couch really could not
  act on (two incarnations, an open park). A retired-dead thread is not
  skipped; it is disconnected, so it needs no line in `reportLeave`.

## Done when

- Leave with a thread whose recorded Pair process and session are gone detaches
  every other live thread, retires the dead one's incarnation, exits Couch, and
  says nothing about it.
- That thread afterwards reads as `session gone`, not as live.
- A live thread whose session is absent is neither signalled nor retired.
- An error after a detach signal still stops leave and names the thread.
- A test drives leave across a stale record placed before live ones.

## Plan

- [ ] Reuse `clearLifecycleDebris` in `Leave`; drive the stale-record ordering
      in a test; confirm `reportLeave` stays silent for a retired thread.

## Log

### 2026-09-19

- Filed from the operator's screenshot. Workaround offered meanwhile: Tab →
  archive on the `kaggle` row (reversible), then Alt+d again.
- Operator settled the direction the same day (see Spec): mark it lost, carry
  on, stay silent. Superseded the "skip and report" reading this issue was
  filed with.
