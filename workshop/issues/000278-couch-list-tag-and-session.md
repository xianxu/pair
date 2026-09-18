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

## Plan

- [ ] Carry the session name from `ProjectSessionPresence` to `ThreadSummary`.
- [ ] Render tag + session in `--list`; print couch's own identity in the header.
- [ ] Tests per Done when.

## Log

### 2026-09-17

- Filed from live operator friction (see Problem).
- Structure repaired 2026-09-17: the file carried a duplicate empty skeleton
  above the real body, leaving `## Plan` holding a bare unchecked `- [ ]` that
  would have tripped the close gate's plan-check. Content preserved verbatim;
  only the section order was fixed and the Plan filled in.

## Revisions

### 2026-09-17 — also print couch's own pid

**Reason:** operator request, prompted by a live incident the same day. The
`astro` thread investigation (`pair#273`) turned on a fact `--list` cannot show:
the **running couch was older than the binary on disk**. The supervisor had
started at 15:49; `#256` M3 landed 16:28–16:54 and the binary was rebuilt at
16:51. Every conclusion drawn about M3's behavior from that process would have
been wrong, and finding it out took `ps` plus commit timestamps.

**Delta to `## Spec`:** `couch --list` also identifies **the couch supervisor
itself** — its pid, alongside the per-thread rows it already prints.

- The pid is the operator's handle for the supervisor: it is what `ps` needs to
  answer "how long has this been running, and does it predate the binary I just
  built?", and what a deliberate restart targets.
- Print it as couch's own line — a header, not a thread row. It is the process
  that owns all of them, and formatting it like a thread invites confusion with
  the launcher pids already shown as `recorded: live pid N`, which are a
  different thing entirely.
- `--list` runs as a separate short-lived process, so this is the **supervisor's**
  pid read from the singleton lease, not `os.Getpid()`. Worth stating explicitly
  because the wrong one is easy to reach for and silently useless.
- No live supervisor is a normal state, not an error: say so plainly rather than
  printing an empty field.

**Delta to `## Done when`:** add —

- `couch --list` names the running supervisor's pid, read from the lease, and
  says plainly when no supervisor is running.
