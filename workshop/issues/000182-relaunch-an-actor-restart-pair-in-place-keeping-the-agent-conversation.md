---
id: 000182
status: open
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
---

# Relaunch an actor: restart Pair in place, keeping the agent conversation

## Problem

Developing Pair while working inside it has no clean restart. Rebuild the
binary and the running actor keeps the old code; the only way to pick up a new
Pair is to lose the session, and losing the session means losing the agent
conversation with it.

couch already has the symmetric move one level up: `alt+d` detaches everything
and leaves, so rebuilding and re-running `couch` picks up new couch code with
every agent still running behind its zellij session. The actor level has no
equivalent -- which is the level where Pair development actually happens.

Pair's own `Alt+n` (`ActionRestartPair`, "reload pair -- kill and re-launch the
workbench in place") is close but not the same thing: it restarts the workbench
*inside* the session couch's child already handed off to. The process couch
spawned is not replaced, so a rebuilt Pair binary is not what comes back.

## Spec

A **relaunch** action on an actor, alongside detach and park:

1. Park the thread -- tear the zellij session down (`kill-session`,
   `delete-session`) and stop the Pair instance, exactly as park does today,
   producing a verified park.
2. Resume it immediately from that verified park: a fresh Pair process, from
   the CURRENT binary, with the saved launch profile, resuming the agent's
   native session so the conversation continues where it was.

Net effect: new Pair code, same conversation, same thread address, same row in
the switcher.

**It is park-then-resume as one action**, and both halves already exist as
declared operations. What this adds is the composition, its failure semantics,
and one operator gesture for it.

**Failure semantics are the whole design question.** Park is destructive and
resume can refuse -- and a relaunch that parks successfully then fails to
resume has destroyed a working session and left the operator with a cold
thread. That is strictly worse than not offering the action. So:

- The resume's preconditions are checked BEFORE the park, not after: an
  established native binding with a non-empty root id, a saved launch profile
  with a supported agent, and a working path that still resolves. If resume
  could not run, relaunch refuses and parks nothing.
- A resume that fails anyway leaves a verified park, which is recoverable --
  the thread stays in the switcher as `parked` and Enter resumes it. The
  refusal must say that plainly rather than reading as data loss.
- pair#181's warm/cold split matters here: relaunch is deliberately COLD. It
  exists to replace the process, so reattaching is not what is wanted.

**Out of scope:** restarting the agent conversation itself (that is Pair's
`Alt+Shift+N`), and any notion of a rolling or zero-downtime restart. The
session goes away and comes back.

## Done when

- An actor action `relaunch` appears alongside detach and park, confirmed like
  park, and reachable from the same declared-operation surface (no private
  switcher verb).
- Relaunching an actor running an OLD Pair binary yields a session running the
  CURRENT one, with the agent conversation continued rather than restarted --
  verified on the real stack by rebuilding Pair between the two observations.
- Relaunch refuses BEFORE parking when its resume could not succeed, proved by
  a test that makes the binding unresumable and asserts the thread is still
  live afterwards.
- A resume failure after a successful park leaves a resumable parked thread and
  says so, proved by test.
- The thread address, its row, and its ledger identity are unchanged across a
  relaunch.

## Plan

- [ ] Decide where the composition lives: a `relaunch` operation in
      `couchcore.Operations()` that sequences park and resume, versus the
      console driving two existing operations. Prefer the operation -- the
      switcher must not grow a private verb, and the precondition check has to
      sit with the durable state, not the terminal.
- [ ] Precondition check extracted from resume's own eligibility rules so the
      two cannot drift (ARCH-DRY): relaunch asks "would a cold resume of this
      thread be accepted right now?" and refuses if not.
- [ ] Sequence park -> resume with the failure semantics above, tested at both
      failure points.
- [ ] Switcher action + confirmation naming what it does ("stops and restarts
      Pair; the conversation continues").
- [ ] Real-stack verification with a rebuilt binary between observations.
- [ ] Atlas: relaunch beside detach/park, and why its preconditions run first.

## Log

### 2026-09-03

- Filed from the couch-lite usability work (pair#170, pair#181). The operator's
  framing: "with alt+d we can quit couch and restart couch; with relaunch we can
  relaunch pair code, but keep working state." Part of making couch-lite usable,
  deliberately kept out of pair#181 rather than appended to it.
