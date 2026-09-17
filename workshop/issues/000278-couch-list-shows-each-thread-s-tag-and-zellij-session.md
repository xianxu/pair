---
id: 000278
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T15:54:17-07:00
---

# couch --list shows each thread's tag and zellij session

## Problem

## Spec

## Done when

-

## Plan

- [ ]

## Log

### 2026-09-17

## Problem

Operator friction, hit live on 2026-09-17: wanting to `zellij kill-session` the
ariadne thread, there was no way to get from what `couch --list` prints to the
session name zellij knows. It took reading two on-disk files by hand:

    ~/.local/share/pair/couch/threadstore/records/<scope>/<tag>.json   # tag -> path
    ~/.local/share/pair/repos/<scope>/session-names.jsonl              # tag -> session name

`couch --show <ref>` resolves a tag, a path or an operator-assigned NAME — and
these threads have no name set, so `couch --show ariadne` answers "thread
reference not found". The label column shows a disambiguated repo name, which is
not something any other tool accepts as input.

## Spec

`couch --list` prints, per thread, the pair tag and the bound zellij session
name.

- The tag is already carried on `ThreadSummary` and already rendered — `--list`
  simply passes `includeAddress: false` (`couchcmd/run.go:706`) where `--show`
  passes `true`.
- The session name is NOT carried. `SessionObservation` lost its `Name` field in
  **#256 M1 round 6**, on a boundary-review finding that it was "written in four
  places, read by nothing in production". That finding was correct then; this
  issue is the consumer that makes the field earn its place again. Re-add it
  deliberately, and say in the code that it is a reversal with a reader — not a
  field that drifted back.
- The name must come from the SAME binding index the classifier uses
  (`ProjectSessionPresence`'s `SessionNameBinding`), not a second read, or the
  displayed name and the name couch acts on can disagree.
- A thread with no binding prints nothing for the session rather than an empty
  pair of quotes — absence is the common, correct case for a parked thread.

## Done when

- `couch --list` shows tag and zellij session for every thread that has them.
- The session name shown is the one `ProjectSessionPresence` resolved, proven by
  a test that changes the binding and sees the output change.
- A thread with no session binding renders without a session line and without
  claiming one is absent, since an unreadable scope is not an absent binding.
