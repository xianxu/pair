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

**The root actor is home** — the session couch launched in, reachable by one
keystroke from every child, always. That is usually brain, by convention rather
than mechanism: couch can be started anywhere, and nothing in the design knows
about brain specifically. If home is ever flaky the operator reverts to tabs and
the project fails.

**Navigation is one key.** `ctrl-space` moves *up one level* — from a child to the
root actor, from the root actor to couch's control panel. No prefix keymap and no
timing window: one key to memorize, then read a screen. All richer navigation
lives inside couch's TUI where there is typeahead and the operator can see what
they are picking. That also suits notifications, where knowing *what* happened
should precede landing in it.

**couch does not composite.** A child gets the terminal one row shorter and couch
owns the last row — the child never knows, so this is a resize, not compositing.
That row carries rolling notifications, identically in the root actor and while
attached to any child, so there is one place to look. Notification detail is not
injected into the transcript as system messages: that would burn the LLM's
context on every turn. The operator asks the advisor "what was that?" and it
answers via a tool call against the same query surface.

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

- [x] spawn + registry [pair#145]
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

### 2026-08-21 — UI model settled

Operator-facing shape, folded into `#145`/`#146`:

- Root session loads like `pair --layout2` — two panes, chat straight away.
  Session selection simplifies away from pair's tag: load the last active
  non-subagent thread in this tree, else start fresh; clearing context is
  `alt+shift+n`.
- `ctrl-space` = up one level (child → root actor → couch panel). Rejected
  double-ESC: ESC is already interrupt/cancel in Claude Code and mode-switch in
  nvim, and a double-tap needs a timing window that either delays every
  legitimate ESC or cannot be retracted.
- Actor ids are generated and opaque (`couch-ah8d`). Operator-assigned names and
  agent-supplied descriptions **attach to the tree, not the id**, so naming
  survives revival — otherwise every restart re-imposes the memory load the
  naming layer exists to remove.
- Notifications render on a reserved status row (child pty shrunk by one), not
  as transcript system messages and not as a composited overlay. Out-of-terminal
  paging can use OSC 777/9, with the known Ghostty caveat that it deletes its own
  notification when the posting window regains focus.
- Everything in couch's TUI is the same operation surface the advisor's tools
  call. `/start ../pair` and the LLM's `start` are two clients, never two
  implementations.

**Open fork for `#146`, to answer empirically first:** does couch host `pair` as
it exists (couch → pair → zellij → claude+nvim, three layers of terminal
management) or host what pair spawns, taking over zellij's role? Step 1 of the
smoke sequence answers it — if hosting pair directly is too complex, couch
absorbs the zellij layer.

### 2026-08-21 — layering fork settled: host `pair` whole

`./bin/couch start ../pair` brings pair up correctly as a couch child, so the
fork opened when the UI model was settled is closed in favour of hosting `pair`
as it exists. couch never needs to reimplement zellij's role, which keeps
`pair#146` to its narrow shape -- one tty, one child at a time -- and unblocks
estimating it.

The wider point for the project: couch supervises *sessions*, not terminals.
Whatever a session runs inside itself (zellij, nvim, an agent) stays that
session's business, which is the encapsulation the actor model asked for,
arriving here as a practical result rather than a design assertion.

### 2026-08-21 — durability decided: the repo is the agent state, no checkpoint clock

couch will **not** run a periodic checkpoint. A timer-driven "externalise now"
control message was proposed and dropped, for three reasons: it is a mechanism
for a problem nothing has measured (the cold-revival experiment says repo-only
recovery works); it creates a second cadence alongside the SDLC's existing
externalisation points (claim, commits, milestone-close, close), which is
mechanism proliferation; and a timer fires mid-thought, while the agent knows
where its boundaries are.

The bet instead: **an ariadne-style repo already is the agent's state.** Two
substrates back it — the repo (commits, issue Log, plan ticks) and pair's
continuously tee'd `scrollback-<tag>-<agent>.raw`, so a crashed agent's whole
transcript is on disk. What the repo does not carry -- the reasoning that never
landed -- is reconstructed by spending tokens against that transcript, which is
the cheap currency.

Two things this makes explicit rather than assumed:

- **`couch stop` is a kill, not a park.** It sends SIGTERM; nothing instructs
  the agent to write out first, and no harness produces a continuation from a
  signal. Parking before shutdown is an operator step today. Having `stop`
  invoke pair's existing park/continue flow is a later issue, not v1 -- it needs
  the agent responsive and it takes time.
- **Silence detection stays in `pair#148`**, as a signal rather than a forcing
  function: a space dirty and uncommitted for hours is one whose reconstruction
  will be expensive, and knowing that while it is still live beats discovering
  it cold.

Net effect on v1: spawn, registry, tty switching, transport. No cadence policy.

### 2026-08-21 — tag and space collapse; naming becomes an attribute layer

Opened `pair#149`. pair's tag does two jobs -- durable storage key and human
handle -- which is why naming is demanded upfront, renaming is not offered, and
tags accumulate uncleaned. The fix is the split couch already made for its own
identity, applied one layer down: the generated id becomes the durable **tag**
(reversing `#145`'s per-incarnation framing), and human names become a mutable
attribute layer over it.

Consequences recorded there: `Spawn` resolves the id rather than minting one;
no separate incarnation id is needed since `{PID, Identity}` already names a
run; names live in pair's session index because pair must resolve them with
couch not running; opaque tags and the picker ship together or not at all; an
unnamed space shows its hex string; `pair claude` standalone is unchanged; and
naming doubles as the retention signal that makes cleanup decidable.
