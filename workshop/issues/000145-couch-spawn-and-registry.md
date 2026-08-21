---
id: 000145
status: working
deps: []
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours:
started: 2026-08-21T12:40:25-07:00
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

**Spawn takes a peer repo, not an issue.** What the agent works on is decided
inside the session — an issue crystallizes mid-thread, it is not a precondition.
Start-args stay **structured, not a bare string**: repo, tree (checkout or
worktree path), agent stack, prompt/skill configuration, and optionally an issue
ref as metadata. couch must know how to bring up a *properly equipped* agent from
that record.

**The durable key is the working tree** (repo + checkout/worktree path). It
survives restarts, exists whether or not there is an issue, and is what two
agents actually collide over. The system-level id need not be legible.

**Actor ids are generated and opaque** — `couch-ah8d`. They identify an actor
*instance* and are not meant to be legible or typed.

**Human addressing is a mutable mapping above them** — short names the operator
assigns at runtime, plus a one-line description of what the agent is doing,
supplied by the agent and free to change mid-session. Nothing structural depends
on either: a label may be wrong, duplicated or stale without corrupting anything.
Resolution stays fuzzy-in/exact-out, so a duplicate label asks which.

**Names and descriptions attach to the tree, not to the actor id.** The id is
per-instance and dies with it; the tree is the durable key. If naming hung off
the id, every revival would re-impose exactly the memory load the naming layer
exists to remove.

**Session selection in the root actor** simplifies away from pair's tag: load the
last active non-subagent thread in this tree, else start fresh. Clearing context
is `alt+shift+n`.

**Name registration IS the collision guard.** Refusing a second agent on a
working tree is `register(repo, tree)` failing because the tree is taken — not a
separate feature. The invariant is one agent per tree, which is more accurate
than one issue per repo: the issue was only ever a proxy for the tree. On
refusal, offer worktree-or-switch, which turns today's silent collision into a
decision at the moment the operator has context to answer it.

**Per-repo concurrency policy is recorded, not inferred per spawn** —
`in-place-serial` where the checkout is the installation (pair, ariadne, parley),
`worktree-parallel` otherwise, plus a third case for workspaces with heavy local
state where worktrees are expensive for unrelated reasons. Read from fleet
metadata (`ariadne#200`) once that exists; a local stub until then.

**Queryable state and callable operations from the first commit.** Every
operation carries structured args and every state is readable — the LLM-drivable
surface the project depends on.

**Descriptors come from the agent, with a cached fallback.** couch asks a live
agent for its one-line description and caches the last answer, because a cold
agent cannot answer and cold is exactly where forgetting lives. This is not the
published-status artifact returning: that failed because it described *state* and
went stale confidently. A stale *label* still finds the right tree, and then you
ask it. Labels tolerate staleness; state does not.

**Bounded mailbox with collapse-by-kind.** Ephemeral, not durable. Two channels —
`control` (stop, deadline) and `normal` — with a priority select. A send is a
non-blocking attempt that may fail; a full mailbox is a loud bug signal, not flow
control.

No switching in this issue: spawned sessions land in the current terminal, the
way they do today.

## Done when

- couch registers, lists and spawns actors against a peer repo, bringing up a
  properly equipped agent for at least the claude stack, with no issue required.
- An agent started on a tree that already holds one is refused by name
  registration, with worktree-or-switch offered.
- An operator-assigned short name and an agent-supplied description both resolve
  to the right actor, and both can change mid-session without breaking anything.
- A name assigned before an actor dies still resolves after it is revived, since
  naming hangs off the tree rather than the instance id.
- Per-repo concurrency policy is read from a recorded source, not inferred.
- Every operation is invocable with structured args and every state is
  queryable — audited over the operation set, not spot-checked.
- `pair-go` standalone is unaffected; couch failing does not block work.

## Plan

- [ ] Registry keyed on the working tree; structured start-args; durable across
      couch runs.
- [ ] Actor goroutine: bounded mailbox, control/normal channels, priority select.
- [ ] Spawn: bring up a claude-stack child from start-args in the right tree.
- [ ] Name registration on (repo, tree) as the collision guard; worktree-or-switch
      on refusal.
- [ ] Runtime naming layer: operator short names + agent-supplied description,
      cached; neither load-bearing.
- [ ] Per-repo concurrency policy source (stub until `ariadne#200`).
- [ ] Queryable-state + callable-operation surface, with the operation-set audit.

## Log

### 2026-08-21

Split out of the former root ticket when couch was promoted to a project. The
architecture narrative, v1 non-goals and cross-repo enablers moved to
`workshop/projects/couch.md`; this issue keeps only task 1.

Rekeyed from issue-as-identity to tree-as-identity the same day (see the project
Log scope event). Spawn now takes a repo; the issue is metadata, not the key.
