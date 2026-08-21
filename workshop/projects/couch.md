---
type: project
name: couch
goal: Stop losing track of concurrent work across peer repos by turning agent sessions into addressable actors the operator can enumerate, switch between, and be paged by.
done_when: The operator works inside a single terminal window, managing a fleet of agents across peer repos, and it works better than today's manual tracking across many tabs.
status: defined
mvp_scope: [pair#145, pair#146, pair#147, pair#148, ariadne#199, ariadne#200]
created: 2026-08-21
updated: 2026-08-21
sources: [brain/workshop/pensive/2026-08-20-01-pensive-couch-agent-switcher.md]
---

# couch

A prototype tty switcher that turns pair sessions into an Erlang-style actor
cluster on a single host, with brain as the always-home advisor. **The headline
omission: one host, one operator, a read-only star.** Agents answer the advisor
and never write to each other; there is no mesh and nothing clusters across
machines. If a second person or a second machine needs in, the model does not
stretch — it gets redesigned.

Also out: LLM suspension and budgets, and durable mailboxes.

## PRD

**The problem is forgetting, not ranking.** Running several workstreams at once
across peer repos, the operator loses count of what is open — worst across
multiple days with time off in the middle — and two sessions in one repo can
collide silently on a working tree. Ranking wants an advisor; forgetting wants a
ledger with cheap writes. This project builds the ledger and puts an advisor on
top of it.

**Agents are intelligent actors.** An agent is LLM + prompts + binaries (`sdlc`
and friends) + harness + an async message-passing API. The keystone: **the
deterministic shell is always up and the LLM is bolted on.** The actor is cheap
and always running, holding identity, mailbox and timers; the LLM loop is the
expensive, suspendable part. Implementation is Erlang's design principles in Go,
not Erlang the runtime — **actors are goroutines inside couch**, and the agent
harness is a child process the actor spawns, feeds, and lets die.

**Four states, not three:** nothing running · running with no child · running
with a **warm** child (context loaded, resume instant, costs RAM) · running with
the child gone (**cold**, context rebuilt from durable artifacts). Warm-vs-cold
decides what waking an actor costs, and is the scheduler's whole job.

**Identity is the working tree; naming is a runtime mapping.** An agent is
spawned against a *peer repo* — what it works on is decided inside the session,
which is how the SDLC already works: an issue crystallizes mid-thread, it is not
a precondition. So the durable key is the **working tree** (repo plus checkout or
worktree path). It survives restarts, exists whether or not there is an issue,
and is precisely what two agents collide over.

The system-level id does not need to be legible. Human addressing is a mutable
layer above it: short names the operator assigns at runtime, plus a one-line
description of what the agent is doing, sourced from the agent itself and free to
evolve during a session. Nothing structural depends on either, so a label may be
wrong, duplicated or stale without corrupting anything. Today's pair *tag*
collapses into this — it was a name the operator had to hold in their head *and*
map to work; the mapping becomes the system's job.

An issue becomes **metadata on a tree** ("this tree is working `pair#10`") —
useful for the advisor's synthesis and for `sdlc` integration, never required at
spawn, never load-bearing for identity.

**brain is the home actor** — the only actor with no issue address, reachable by
one keystroke from every child, always. If that is ever flaky the operator
reverts to tabs and the project fails.

**A switcher, not a multiplexer.** One tty attached to one child at a time, a key
interceptor, a per-child buffer replayed on attach. No splits, layouts, floating
panes, or simultaneous rendering — that is where tmux's real complexity lives,
and reimplementing it badly is the failure mode.

**Every function is LLM-drivable, from day 1.** State is queryable rather than
only rendered; every operation is a callable with structured args. The design
test: *there is no operation the operator can perform that an LLM cannot, and no
state the operator can see that an LLM cannot read.* The terminal UI and the LLM
are peer clients of one surface. But the LLM sits in the **naming** path, never
the **switching** path — a model turn in the critical path reintroduces the
latency that sends the operator back to tabs.

**Queries, not commands.** The dialect is shared
(`construct/vocabulary/*.cue`); the authority is not — actor B invoking `sdlc
close` in A's repo is permission laundering and breaks "only the actor
interprets its own state." couch routes `{query, verb, args}` to the *owning*
actor's shell, which runs the binary in its own root under its own permissions.
The caller never runs the binary, so the deterministic query API is literally
the CLI, filtered.

**Anything with a real-time obligation lives in the deterministic layer.** The
LLM is suspendable by construction, so it never owns a deadline. Timers belong
to the shell; switching is LLM-free; name registration is exact — and **name
registration IS the collision guard**, since refusing a second agent on a
working tree is just `register(repo, tree)` failing. The real invariant was never
one issue per repo; it is one agent per tree, which is what actually collides.

**Two staleness signals, both needed:** mailbox depth says someone is waiting on
this agent; git staleness says the thread has gone cold. The thread nobody has
messaged and nobody has touched is the one that dies.

**Enumerate on measured facts, never self-declared status.** A cold-revival
experiment on 2026-08-20 established this: `kbench#24` read `status: working,
estimate_hours: 4.98` for a month while 256 commits of the real work happened
elsewhere, while git said `0 ahead, last touched 2026-07-23` and was correct.
That same experiment **falsified** an earlier design in which agents published a
cached status document — the published status went stale while the repo carried
current truth, and a cold reconstruction beat the cached artifact. Hence: no
cached status layer; revival plus durable repo state is the recovery path.

## Estimate

Not yet derived. Required before the `defined` → `committed` transition, along
with `deadline` and `planned_finish`. Per-issue `estimate_hours` are derived
after each plan clears `change-code`'s plan-quality gate, so the project-level
figure assembles from those rather than being typed up front.

Sizing basis when it is done: `pair#145` and `pair#146` are the substantive
build (registry, actor loop, tty routing); much of the terminal layer already
exists in `cmd/internal` (`wrapcmd` over `x/vt` + `creack/pty`, `scrollbackcmd`,
`launcher`/`dispatcher`/`runtimebundle`), so shared code is reuse rather than
extraction. `ariadne#199`/`#200` are additive surface on an existing binary.

## Breakdown

**Sequencing is deliberately not framework-first.** The actor runtime is the
target architecture, but building it first would deliver nothing observable for
a long time — the classic trap. Instead `#145` and `#146` ship a working
switcher whose *shapes* are actor-compatible (every operation a message, every
agent a registered name, all state queryable), and the runtime arrives as a
refactor of something that already runs.

**Each of the first two tasks is a real stopping point.** After `#145` the
project has already killed silent collisions; after `#146` the operator has a
switcher they would use daily. If it stalls there, most of the original value is
banked. Value front-loaded, risk back-loaded — the terminal work in `#146` and
the cross-stack work in `#147` are where risk concentrates.

`ariadne#199` and `#200` are enablers, not couch work: `sdlc` owns the inventory
(what work exists, measured git facts, per-repo concurrency policy), couch owns
the runtime (bringing actors up, tty routing, transport, live registry). They
gate `#147` and `#148` respectively; `#145` and `#146` do not depend on them.

- [ ] spawn + registry [pair#145]
- [ ] tty switching and attach [pair#146]
- [ ] expose query API to peer actors [ariadne#199]
- [ ] fleet thread inventory [ariadne#200]
- [ ] cluster transport and queries [pair#147]
- [ ] brain advisor role [pair#148]

## Log

### 2026-08-21 — project opened

Promoted from a single root ticket (the former `pair#145`, "couch: agent
switcher and pair cluster") after the milestone set grew cross-repo and the
milestones were better served as separate issues with their own close gates. The
architecture narrative moved here; `pair#145` was rewritten as task 1.

`done_when` set by the operator: one terminal window, a fleet of agents across
peer repos, beating today's manual tab tracking. Note the single-window clause is
a stronger claim than "switch between agents" — see the open scope question
below.

Status is `defined` rather than `committed`: the PRD exists but no `deadline` or
`planned_finish` has been set, and neither was invented. Moving to `committed`
needs both, via `sdlc project set-status`.

**Scope decided 2026-08-21:** couch hosts **agent children only**. "A single
terminal window" means one window for *agent* work; plain shells, log tails and
test runs stay outside it and the operator leaves the window for those. `#146`
therefore stays narrow — a switcher over agent actors, not a general child host —
which keeps the no-multiplexer line intact.

Two open questions carried in from the design discussion that nothing currently
answers: **when does couch kill a warm child** (warm costs RAM and buys instant
resume; cold costs a reconstruction — that policy is the scheduler's entire job),
and whether revival quality is contingent on ledger discipline. The 2026-08-20
experiment revived an unusually well-kept thread; a messy one is untested, and
the productive form of the worry is whether the observer can cheaply *measure*
revivability and warn while a thread is still fixable.

### 2026-08-21 — scope event: identity rekeyed from issue to working tree

Walked back the issue-as-identity decision. Original framing required an issue
ref at spawn, which made *how untracked work gets named* the project's headline
omission.

Why it was wrong: requiring an issue at spawn contradicts the SDLC's own flow —
claim happens after an idea crystallizes, and pre-issue exploration is explicitly
normal. The spawn parameter is the **peer repo**; what to work on is decided
inside the session.

What changes: the durable key becomes the working tree (repo + checkout/worktree
path). System ids need not be legible; human addressing is a runtime mapping of
operator-assigned short names plus a live one-line description supplied by the
agent, both mutable mid-session and neither load-bearing. Name registration
rekeys from `register(pair#10)` to `register(repo, tree)`, which is a more
accurate statement of the invariant. An issue becomes metadata on a tree.

Net effect on scope: the headline omission is *closed*, not deferred — a thread
like rogii-v2 (11 days, 301 commits, no issue file, missed deadline) becomes
addressable on day one. `ariadne#200` gets simpler, enumerating working trees
rather than issue records. The new headline omission is the honest remaining one:
single host, single operator, read-only star.
