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
omission: work that has no issue.** Identity is the issue name, so a thread
running off-spine — no issue file, commits straight to main — is unaddressable
by this MVP. That is not incidental: the concrete failure that motivated the
project (a Kaggle deadline of 2026-08-05 that passed unnoticed and cost a
submission) happened on exactly such a thread. So the MVP proves *no tracked
thread gets lost*, not *no thread gets lost*, and how untracked work gets named
is deferred rather than solved.

Also out: multi-host clustering, mesh topology (star only, read-only), LLM
suspension and budgets, durable mailboxes, and agent→agent write channels.

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

**Identity is the issue name** (`pair#10`, `ariadne#111`). Today's pair
identifier is a *tag*, which forces the operator to hold a tag→issue mapping in
their head — that indirection is itself part of the memory load being attacked.
Ad-hoc sessions with no issue behind them are expected to stay transient and
unregistered, the way brain is unaddressed; whether that holds is open.

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
registration IS the collision guard**, since refusing a second in-place spawn is
just `register(pair#10)` failing.

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
