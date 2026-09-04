---
id: 000182
status: working
deps: []
github_issue:
created: 2026-09-03
updated: 2026-09-04
estimate_hours: 3.58
started: 2026-09-04T09:16:38-07:00
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec                 design=0.90 impl=0.08
item: smaller-go-module          design=0.06 impl=0.20
item: greenfield-go-module       design=0.20 impl=0.28
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.16
item: smaller-go-module          design=0.04 impl=0.14
item: tui-screen                 design=0.16 impl=0.20
item: real-api-discovery         design=0.00 impl=0.18
item: atlas-docs                 design=0.03 impl=0.08
item: milestone-review           design=0.00 impl=0.20
design-buffer: 0.15
total: 3.58
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

| Slug | Instances |
| --- | --- |
| `issue-spec` | this issue + plan authoring, two plan-quality rounds spent |
| `smaller-go-module` | the `CheckResumePreconditions` extraction; the refuse-before-park test; the park-failure branch; the success test |
| `greenfield-go-module` | `Couch.Relaunch` + `RelaunchResult`/`RelaunchOutcome` |
| `tui-screen` | the switcher action, its confirmation, and the two-phase progress notice |
| `real-api-discovery` | the rebuilt-binary verification on the real stack |
| `atlas-docs` | atlas + README |
| `milestone-review` | one boundary |

**Where judgment entered:**

- **`issue-spec` at full design (×1.0), no spec-quality discount**, on pair#170's
  precedent: the spec-authoring primitive cannot be pre-resolved by its own
  output, and the hard question here — the failure model — was open until the
  plan-quality gate closed it. Two rounds are already spent, one of them on a
  Critical (the park-failure branch was missing entirely).
- **Every other design line ×0.2.** The plan fixes files, signatures and test
  strategies per task, so the remaining design cost is reading.
- **One `real-api-discovery` line, not zero.** Verifying that the agent
  conversation *continued* rather than restarted needs a rebuilt binary between
  two observations on the live stack, and pair#181 M2 showed that kind of check
  finds things a fake cannot (the ledger row count is what proved reattach).
- **No `scope-pivot` line, unlike pair#170 and pair#181.** Both halves already
  exist as tested operations, and the composition's semantics were settled by
  the gate rather than left to be discovered. This is the "shape known" column
  of pair#170's own ratio table.
- **Design buffer +15%** (v2.1 Step 6): thorough plan doc, and +30% on top of a
  ×0.2 discount double-counts the same thoroughness.
- **`impl=` values are already v3.1-scaled** to 40% of the v2/v2.1 table.

**Known risks:**

- **The park-failure branch is the least testable part.** It needs a fake that
  fails at specific park exits, and `pairlifecycletest` supports that — but if
  this estimate misses, it is because reaching three distinct exits took longer
  than one `smaller-go-module` line.
- **Direction: more likely high than low.** Unlike pair#181, nothing here is a
  measurement-driven redesign; the work is composition over two surfaces the
  session has spent all day in.


## Plan

Design landed at `workshop/plans/000182-relaunch-an-actor-plan.md`. One review
boundary — the work is one action composed from two existing operations, so
splitting it would force a redundant milestone-close (AGENTS.md §3).

- [ ] M1 — relaunch as one declared operation: the resume preconditions split
      from the occupancy rule they share with resume, `Couch.Relaunch` checking
      those preconditions BEFORE the park, a recoverable-and-stated outcome when
      the park succeeds and the resume does not, the switcher action on live
      rows only, and real-stack verification with a rebuilt binary.

Superseded first draft, kept for its reasoning:

- [-] Decide where the composition lives: a `relaunch` operation in
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
