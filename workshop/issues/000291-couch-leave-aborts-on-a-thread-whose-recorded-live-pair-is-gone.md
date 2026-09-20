---
id: 000291
status: codecomplete
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-20
estimate_hours:
started: 2026-09-20T14:05:57-07:00
flow: {kind: full, provenance: inferred}
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
- That thread afterwards has no live incarnation; it reads as `parked` when
  its saved conversation is resumable, otherwise `session gone`.
- A live thread whose session is absent is neither signalled nor retired.
- An error after a detach signal still stops leave and names the thread.
- A test drives leave across a stale record placed before live ones.
- Unknown process identity is preserved and reported skipped; observation
  errors and cancellation stop leave without inventing successful detaches.

## Plan

- [x] Reuse `clearLifecycleDebris` in `Leave`; drive the stale-record ordering
      in a test; confirm `reportLeave` stays silent for a retired thread.

## Log

### 2026-09-19

- Filed from the operator's screenshot. Workaround offered meanwhile: Tab →
  archive on the `kaggle` row (reversible), then Alt+d again.
- Operator settled the direction the same day (see Spec): mark it lost, carry
  on, stay silent. Superseded the "skip and report" reading this issue was
  filed with.

### 2026-09-20 — implementation on main
- Review SHIP; corrected its sole minor finding: two present-tense atlas
  descriptions of semantic submit/compose now reflect submit-only delivery.
  The wider review window includes intervening main commits; local verification
  above covers the full Couch package suites despite the reviewer stopping its
  duplicate broad run. No implementation changes were required by review.
- 2026-09-20: closed — Six stale-record preflight regressions fail before and pass after; couchcore/couchtty/couchcmd full suites pass; Leave and Detach race tests pass; vet passes; pair and couch rebuilt. Dead entries emit no leave result while running and uncertain processes remain untouched.; review verdict: SHIP
- 2026-09-20: flow upgraded quick → full — 824 added lines in code files (limit 100)

- Operator reproduced the failure on tools (`couch-2e662a595ae09564`), then
  requested fixing this on main before resuming #292. The recorded PID 3779
  is absent; the switcher correctly offers the saved conversation as parked.
- Reuse exact-process observation and `clearLifecycleDebris` for confirmed
  dead incarnations (ARCH-DRY). Preserve unknown processes and live processes
  without a session; do not signal them. Keep errors after signalling fatal.
- Six preflight cases failed before implementation and pass after it. All
  couchcore, couchtty and couchcmd tests pass; targeted Leave/Detach race
  tests and vet pass. Repeated new regressions pass ten runs. Built both
  `bin/pair` and `bin/couch`; the current shell resolves couch to this checkout.
- Dead retirement contributes no Detached/Parked/Skipped entry, so the existing
  `Console.reportLeave` emits no line for it. Existing partial-failure tests
  still prove that a failed signalled detach stops the sweep.

## Revisions

### 2026-09-20 — current classifier and implementation scope

- The earlier Done-when wording "session gone" predates ledger-backed cold
  resume: a retired thread with a resolvable saved conversation is **parked**;
  only one without that proof is **session gone**. Acceptance is that it no
  longer carries a live incarnation; classification remains evidence-derived.
- Before detaching each eligible recorded-live thread, check its exact process.
  Dead means retire through existing cleanup and continue silently, including
  when its session survives. Unknown means preserve and report skipped. Live
  with an absent session also means preserve and skip. Observation/write errors
  stop leave, as do errors from the existing detach operation.
- Regression coverage includes stale-first ordering, PID reuse, surviving
  sessions, live/unknown preservation, cancellation and observation errors.
