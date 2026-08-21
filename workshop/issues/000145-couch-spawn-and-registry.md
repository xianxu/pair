---
id: 000145
status: open
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours:
---

# couch: spawn and registry

Project: `workshop/projects/couch.md` — architecture, non-goals and execution
order live there; this issue is task 1.

## Problem

Nothing knows what agent sessions exist. Pair drives one session; the outer
shell is cmux or ghostty tabs, which have no notion of what an agent session
*is*, so there is no way to ask what threads are open, bring a dormant one back,
or stop two sessions from silently sharing one working tree.

Before any switching UI is worth building, couch needs the thing underneath it:
a registry of named actors and a deterministic way to bring one up.

## Spec

**Actors are goroutines inside couch**; the agent harness (claude/codex, or a
directly-driven loop) is a child process the actor spawns, feeds, and lets die.
One OS process stays up, mailboxes are structs, and the name registry is
authoritative because it lives in one address space.

**Identity is the issue name** — `pair#10`, `ariadne#111`. `sdlc` already speaks
`repo#id`, and the issue file carries branch, status, plan and Log, so start-args
are largely derivable. The record stays **structured, not a bare string**: given
a name, couch must know how to bring up a *properly equipped* agent (agent stack,
repo root, prompt/skill configuration). Prototype scope is ariadne's issue
subsystem only.

**Name registration IS the collision guard.** Refusing a second in-place spawn in
a repo that already holds one is `register(pair#10)` failing because the name is
taken — not a separate feature. On refusal, offer worktree-or-switch, which turns
today's silent collision into a decision at the moment the operator has context
to answer it.

**Per-repo concurrency policy is recorded, not inferred per spawn** —
`in-place-serial` where the checkout is the installation (pair, ariadne, parley),
`worktree-parallel` otherwise, plus a third case for workspaces with heavy local
state where worktrees are expensive for unrelated reasons. Read from fleet
metadata (`ariadne#200`) once that exists; a local stub until then.

**Queryable state and callable operations from the first commit.** Every
operation carries structured args and every state is readable — the LLM-drivable
surface the project depends on. Includes a one-line human descriptor per thread
(the issue title serves) so fuzzy resolution has something to match.

**Bounded mailbox with collapse-by-kind.** Ephemeral, not durable. Two channels —
`control` (stop, deadline) and `normal` — with a priority select. A send is a
non-blocking attempt that may fail; a full mailbox is a loud bug signal, not flow
control.

No switching in this issue: spawned sessions land in the current terminal, the
way they do today.

## Done when

- couch registers, lists and spawns actors by `repo#id`, bringing up a properly
  equipped agent for at least the claude stack.
- A second in-place spawn in a repo that already holds one is refused by name
  registration, with worktree-or-switch offered.
- Per-repo concurrency policy is read from a recorded source, not inferred.
- Every operation is invocable with structured args and every state is
  queryable — audited over the operation set, not spot-checked.
- `pair-go` standalone is unaffected; couch failing does not block work.

## Plan

- [ ] Registry: named actors, structured start-args, durable across couch runs.
- [ ] Actor goroutine: bounded mailbox, control/normal channels, priority select.
- [ ] Spawn: bring up a claude-stack child from start-args in the right root.
- [ ] Name registration as the collision guard; worktree-or-switch on refusal.
- [ ] Per-repo concurrency policy source (stub until `ariadne#200`).
- [ ] Queryable-state + callable-operation surface, with the operation-set audit.

## Log

### 2026-08-21

Split out of the former root ticket when couch was promoted to a project. The
architecture narrative, v1 non-goals and cross-repo enablers moved to
`workshop/projects/couch.md`; this issue keeps only task 1.
