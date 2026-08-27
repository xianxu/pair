---
type: project
name: couch
goal: Stop losing track of concurrent work across peer repos by turning agent sessions into addressable actors the operator can enumerate, switch between, and be paged by.
done_when: The operator works inside a single terminal window, managing a fleet of agents across peer repos, and it works better than today's manual tracking across many tabs.
status: defined
mvp_scope: [pair#145, pair#146, pair#147, pair#148, pair#149, pair#151, pair#152, pair#153, ariadne#199, ariadne#200]
created: 2026-08-21
updated: 2026-08-26
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
- [x] shared pty-child core [pair#146 M1]
- [x] console over one child, with the reserved row [pair#146 M2]
- [x] many children and the panel [pair#146 M3]
- [x] exits, detach, and what the row says [pair#146 M4]
- [ ] durable work-thread identity, naming, and launch profiles [pair#149]
- [x] singleton namespace and normalized admission [pair#149 M1]
- [x] recoverable pre-exec start transaction [pair#149 M2]
- [x] shared thread metadata, operations, and standalone lookup [pair#149 M3]
- [x] remembered per-path agent and argument profiles [pair#149 M4]
- [ ] legacy migration and composite artifact proof [pair#149 M5]
- [ ] hierarchical thread menu [pair#151]
- [ ] verified park and activity age [pair#152]
- [ ] managed-worktree lifecycle [pair#153]
- [ ] expose query API to peer actors [ariadne#199]
- [ ] fleet thread inventory [ariadne#200]
- [ ] cluster transport and queries [pair#147]
- [ ] brain advisor role [pair#148]

<a id="pair-149-m1"></a>
### pair#149 M1 — singleton namespace and normalized admission

**est:** 17.80
**closed:** 2026-08-26
**actual:** 35.47h

Couch now resolves one physical store namespace and protects it with one
non-inherited supervisor lease. Its locked/revisioned ThreadStore atomically
claims final composite opaque tags before a child can fork and conservatively
reconciles creating/live/unknown incarnations against Ariadne #200's normalized
policy evidence. The surprise worth preserving is that removing the old local
policy model also required removing registry admission and validating unknown
CLI flags: deleting one enum without sweeping every decision consumer would
have left a functioning bypass. Pair owns a scheduled live conformance check
against the real provider so the stateful fake and strict decoder cannot drift
silently (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

<a id="pair-149-m2"></a>
### pair#149 M2 — recoverable pre-exec start transaction

**est:** 17.80 (whole issue)
**closed:** 2026-08-26
**actual:** 4.44h

Couch now forks an internal helper that cannot exec Pair until the exact
nonce/supervisor/helper tuple is durable. Pair's composite address claim is the
registration oracle, so a successful pipe write or live PID cannot prematurely
promote the thread. Restart reconciliation preserves the occupied-or-proven-free
invariant: dead and unregistered rolls back by nonce and revision; established
survivors promote; unknown evidence stays occupied. The surprising integration
fact is that the PTY runner had to inherit the same close-on-exec descriptor as
the stdio runner—otherwise console starts and `--no-console` starts would have
different crash safety. A committed real-process probe exercises both restart
outcomes against kernel process identities (ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

<a id="pair-149-m3"></a>
### pair#149 M3 — shared thread metadata, operations, and standalone lookup

**est:** 17.80 (whole issue)
**closed:** 2026-08-26
**actual:** 2.40h

Human names, operator descriptions, and agent summaries now live as independent
revisioned fields on the composite ThreadStore record. CLI and panel render one
shared inventory; selection and hosted-target joins use `{repo scope, tag}` so
same-path Brain threads remain separate. A closure-free operation schema routes
human and future advisor effects through explicit direct-store/live-owner
executors, with console-local exact switch/attach and typed #147 refusal when no
owner is available. Pair's portable read-only ThreadIndex makes names and parked
threads available to standalone resume/picker flows without mutating opaque
artifacts or zellij session-name bindings (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

<a id="pair-149-m4"></a>
### pair#149 M4 — remembered per-path agent and argument profiles

**est:** 17.80 (whole issue)
**closed:** 2026-08-26
**actual:** 1.52h

Couch resolves agent and argv provenance independently, using Pair's shared
harness inventory and repo-scoped defaults. A successful registration journals
the exact incarnation profile together with revisioned history keyed by
normalized repository identity and physical path; failed starts leave history
unchanged. A restart-level scenario proves the next thread restores the last
successful agent and that agent's exact arguments without leaking another
harness's flags or reopening Pair's config picker (ARCH-DRY, ARCH-PURE,
ARCH-PURPOSE, ARCH-MOCK).

<a id="pair-149-m5"></a>
### pair#149 M5 — legacy migration and composite artifact proof

**est:** 17.80 (whole issue)

Legacy registry metadata is journal-enriched into the existing ThreadStore
without moving source files or weakening conservative admission. One validated
`artifactpath` leaf now constructs every tag-bearing Pair path and exports exact
bindings to Go, shell, Neovim, and both layouts; a checked source/family manifest
includes the generated runtime mirror. A real two-scope strategy proves the
same legacy tag cannot observe or mutate the other repository's artifacts.
Standalone Pair retains its ordinary tag prompt while upserting its incarnation
through the same locked/revisioned ThreadStore authority (ARCH-DRY, ARCH-PURE,
ARCH-PURPOSE, ARCH-MOCK).

<a id="pair-146-m3"></a>
### pair#146 M3 — many children and the panel

**est:** 10.32 (whole issue)
**actual:** 9.20h
**closed:** 2026-08-24

Couch now hosts multiple warm Pair children and switches the operator among
them through a deterministic panel: `ctrl-space` climbs child → root → panel;
arrows/Enter, digits and typeahead select a destination; panel actions reuse the
same operation table the CLI and future advisor consume. Panel rows keep
worktree identity for human resolution separate from console-local child
identity for routing.

The real smoke was the milestone's design review. It found that key decoding
worked only in legacy encoding, actions were declared but initially inert, a
started actor never joined the live console, and the panel displayed repo-name
fallbacks its resolver could not search. The fixes addressed those classes at
their shared boundaries; the operator confirmed the final two-actor smoke.

**Scope event — 2026-08-24:** the first M3 boundary review returned REWORK, so
the provisional `actual`/`closed` metadata above is not the final milestone
record and the portfolio row remains open. Follow-up operator evidence confirmed
that killing the couch console and starting it again reattaches the same zellij
session. The same cold-start pass removed Pair's saved-config picker from the
couch path: couch now requests the repo default through a consumed one-shot
entry policy, while direct Pair retains its picker and manual default override.
Final measured time and close date will replace the provisional record only
when `sdlc milestone-close` accepts the boundary.

<a id="pair-146-m2"></a>
### pair#146 M2 — console over one child, with the reserved row

**est:** 10.32 (whole issue)
**actual:** 9.35h
**closed:** 2026-08-23

`couch start` became the console: a pty per child, the operator's terminal in
raw mode, and a status row reserved by pinning the scrolling region. `PtyRunner`
sits behind the existing `Runner` seam as a capability on the handle, so
`--no-console` keeps the stdio path alive rather than leaving it as dead code.

**The milestone's value was in what the verification found, not in the code
being hard.** Four real bugs, each invisible to the layer above it:

- The reserved row is destroyed by an ERASE, not just by scrolling. DECSTBM
  covers scrolling only, and every full-screen app clears on startup. Found by
  operator smoke; the emulator tests were green because a scrolling child never
  clears.
- The console spliced its row paint into the middle of the child's escape
  sequences, corrupting output. Found by putting a REAL pty child under the
  console; no fake-child test could produce it, because a fake emits only what
  the test hands it whole.
- `ctrl-space` was never intercepted: zellij enables the Kitty keyboard
  protocol, so the terminal sends CSI-u rather than NUL. The evidence was
  already in the tree — pair's own chord table carries both encodings.
- A production data race in the pty handle id, caught by the whole-tree `-race`
  target that had no runnable directory until M1's boundary review fixed it.

**The transferable lesson is about test ladders.** Fake child → emulator →
real pty child → real stack: each rung caught something the rung below reported
as green. Two of the four came from the operator's keyboard. Worth carrying into
M3, where the panel and multi-child switching have the same shape of risk.

**Measurement caveat.** The 9.35h increment is `sdlc actual`'s cumulative 12.91h
minus M1's 3.56h, over a window spanning 2026-08-22 19:34 to 2026-08-23 08:01 —
which contains an ~8h overnight gap and several operator-wait gaps. Idle removal
is the engine's, not mine; read it as "measured, window not clean" rather than
as focused hours.

<a id="pair-146-m1"></a>
### pair#146 M1 — shared pty-child core

**est:** 10.32 (whole issue; M1 is roughly its first quarter)
**actual:** 3.56h
**closed:** 2026-08-22

Extracted the terminal plumbing out of `pair term` into two packages both it and
`couch` drive -- `cmd/internal/ptychild` (a child on a pty, its bounded replay
ring, the #127 query deny-list, and one scanner over its output) and
`cmd/internal/hostty` (the operator's terminal: size, raw mode, coalesced
resizes, and the control constants). `pair term` migrated onto both in the same
milestone, deliberately: extracted code with no second consumer is unvalidated
new code, and termcmd's existing suite is the only net that could prove the
extraction faithful.

**The surprise was that extracting found bugs rather than just moving code.**
The ring's trim re-sliced instead of copying, so its bound depended on `append`
happening to reallocate -- invisible from outside, because `Snapshot` reports
the window not the allocation. `updateMouseMode` scanned each pty read
independently and could not see a sequence split across a read boundary. And BEL
was about to be grepped rather than framed, which would have fired the status
row's one activity signal on every title change.

**Worth preserving for the rest of the project:** the structure/policy split is
what makes one mechanism serve two switchers -- `termcmd` keeps numbered tabs,
rename and the zellij pane title; `couch` will keep named actors and a panel.
The same split `cmd/internal/ansi` already documents, and the reason
`wrapcmd`'s opposed capability table correctly stayed where it was.

**Measurement caveat for calibration.** The 3.56h is engine-measured
(active-time-v3, 15-min idle threshold) over `d0a3b251 → fedf3853`, but that
window spans 12:53–18:05 wall clock and contains a **3h48m gap** where the
session was waiting on an operator smoke test, plus roughly 0.65h of pre-window
planning after the claim. Read the number as "measured, window not clean"
rather than as a tight figure.

## Log

### 2026-08-26 — pair#149 M5 implementation ready for boundary

M5 preserves every legacy source while completing composite addressing all the
way through runtime consumers. Artifact construction has one checked leaf;
exact bindings cross process and layout boundaries; direct Pair and Couch now
publish into one ThreadStore inventory. The repeated-tag integration test uses
Go, shell, and Neovim mutations plus both layouts as the oracle. The portfolio
row remains open until the M5 review boundary and final issue close record their
verdict and measured actual. Full, race, live-process, policy-provider,
terminal/Zellij, crash-recovery, layout, and bundle-determinism checks are green
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

The first boundary review's upgrade and constructor-closure findings are now
covered by strict old/new index overlap reads, extensionless-source discovery,
explicit legacy-path authority, exact non-Go bindings, a mechanically truthful
entity inventory, and an atlas-wide obsolete-formula sweep. Full, race, live,
policy, terminal, recovery, bundle, and layout checks pass on the disposition.

### 2026-08-26 — pair#149 M3 implementation ready for boundary

The durable naming layer is now usable from Couch and standalone Pair: one
composite record, one matcher, one inventory, and one declared operation
surface. The boundary test writes a real Couch ThreadStore record, updates its
human name, resolves it through launcher with Couch absent, and verifies the
scoped Pair draft remains under the opaque tag. Duplicate names refuse or gain
picker-only tag disambiguators; direct Pair tags retain exact precedence. The
portfolio row stays open until `sdlc milestone-close` records review verdict,
measured actual, and closure (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — pair#149 M1 implementation ready for boundary

Couch now owns one leased physical namespace and a locked/revisioned
ThreadStore. New starts claim final composite opaque tags and consume
Ariadne #200's normalized policy result; Pair no longer has a repo-name policy
table or same-path admission bypass. Live provider conformance covers bounded,
unbounded, epoch-change, and typed-refusal behavior. The portfolio row remains
open because #149 still has M2-M5; measured M1 actual/close evidence is added
only after `sdlc milestone-close` accepts the boundary (ARCH-DRY,
ARCH-PURPOSE, ARCH-MOCK).

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

**Scope event — 2026-08-26:** the canonical couch store is the durable namespace
and has one live supervisor lease. Couch restarts adopt that namespace; a second
supervisor refuses rather than creating console-invisible actors. Ordinary
read/metadata clients may share the locked store, while owner-required actions
route through #147. #148 exposes the same declared operation surface to the root
agent through a thin Ariadne-distributed skill; Pair remains the authority for
operations and Ariadne only distributes the adapter.

The namespace is the once-resolved absolute physical store path, not a process
incarnation or caller spelling. Its lifetime supervisor lease is non-inheritable
by children. Transport, mailboxes, manifests, and notifications address exact
composite work threads inside that namespace; a path is only a discovery and
admission input. Even implicit human actions such as Enter-to-switch dispatch a
declared owner-required operation available through the same generic advisor
client.

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

- **`couch stop` is a PARK, not a kill -- corrected 2026-08-22 from a
  measurement.** This entry originally asserted the opposite and was never
  tested. `probes/zellijpark` creates a throwaway zellij session, kills its
  CLIENT, and looks: the session survives **both SIGTERM and SIGKILL**. pair
  installs no SIGTERM handler, and `DeleteSession` is reached only from explicit
  quit/restart/layout paths -- so signalling pair does not take its session with
  it. Measured at the zellij layer; the full `couch stop` path is confirmed by
  operator smoke in `pair#146` M2.

  What that changes: `stop` frees the tree and ends the *view*, while the work
  keeps running detached and is resumable. What it does NOT change is the
  original entry's real point -- **nothing tells the agent to write out first**.
  A parked session still holds un-externalised reasoning, so "the repo is the
  agent's state" is unaffected: the bet is that a revival reconstructs from
  commits, issue Logs and the tee'd scrollback, not that a park is a graceful
  shutdown. Having `stop` invoke pair's park/continue flow remains a later
  issue.
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

### 2026-08-25 — work-thread identity, park, and worktree lifecycle split

Fresh review of `pair#149` showed that three lifecycles had been folded into one
issue. The durable Pair tag and its human metadata are a storage/launcher
identity; proving a zellij-hosted session fully quiescent is a process protocol;
creating and garbage-collecting linked worktrees is repository lifecycle. They
now sequence independently:

- `ariadne#200` supplies the normalized prospective-path policy query;
- `pair#149` owns the durable work-thread record, start/resume, human metadata,
  migration, and policy consumption;
- `pair#152` owns verified park, flush/quiescence evidence, and observed
  `last_active_at`, sharing its proof boundary with `pair#135`;
- `pair#153` owns generated-worktree provisioning and conservative garbage
  collection; and
- `pair#151` renders the hierarchical thread menu only after both identity and
  verified park exist.

This also corrects two earlier project claims. Human names do not overload the
zellij `SessionName` binding in Pair's existing index; they live on Pair's own
durable thread record. And killing couch's zellij client is a detach, not a
verified park: it must never free an admission slot while the server-side
session can still write to the workspace (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-25 — actor launch remembers the path's LLM profile

Starting a thread is path-oriented, so couch remembers the last successfully
used agent for each canonical path and the last arguments for each agent at that
path. Ctrl-Space defaults to that agent; a path with no history inherits the
root actor's agent. Parameters never cross agents and fall back to Pair's
repository default when that path has no history for the selected agent. The
durable thread separately records its latest incarnation profile, and neither a
cancelled selection nor a failed start changes the remembered preference.

[pair#146 M1]: #pair-146-m1
[pair#146 M2]: #pair-146-m2
[pair#146 M3]: #pair-146-m3
[pair#149 M1]: #pair-149-m1
[pair#149 M2]: #pair-149-m2
[pair#149 M3]: #pair-149-m3
[pair#149 M4]: #pair-149-m4
[pair#149 M5]: #pair-149-m5
