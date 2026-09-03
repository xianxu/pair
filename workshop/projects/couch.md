---
type: project
name: couch
goal: Let the operator run a group of live coding sessions from one terminal — enumerate them, switch between them, and be paged by them — instead of tracking them across tabs.
done_when: The operator runs a group of live coding sessions in one terminal window — starting, resuming, detaching, reattaching, and following notifications between them — and prefers it to tabs.
status: executing
mvp_scope: [pair#145, pair#146, pair#149, pair#151, pair#152, pair#155, pair#170, ariadne#200]
created: 2026-08-21
updated: 2026-09-02
sources: [brain/workshop/pensive/2026-08-20-01-pensive-couch-agent-switcher.md]
---

# couch

> **Current scope: couch-lite (2026-09-02).** couch is a switcher over a group
> of live coding sessions, not an actor cluster. The brain advisor, cluster
> transport, cross-actor queries and managed worktrees are punted — see the
> scope event at the top of the Log, and `pair#170`. The PRD below is the
> original intent, kept for its reasoning rather than as a description of what
> is being built.

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
- [x] durable work-thread identity, naming, and launch profiles [pair#149]
- [x] singleton namespace and normalized admission [pair#149 M1]
- [x] recoverable pre-exec start transaction [pair#149 M2]
- [x] shared thread metadata, operations, and standalone lookup [pair#149 M3]
- [x] remembered per-path agent and argument profiles [pair#149 M4]
- [x] legacy migration and composite artifact proof [pair#149 M5]
- [x] deterministic agent session-tree inventory [pair#155]
- [x] deterministic native forests [pair#155 M1]
- [x] round-gated native bindings and public inventory [pair#155 M2]
- [x] verified park and activity age [pair#152]
- [x] actionable inventory and token-bound start authority [pair#151 M1]
- [x] pure hierarchical menu and scheduler [pair#151 M2]
- [x] Console integration and performance evidence [pair#151 M3]
- [-] managed-worktree lifecycle [pair#153]
- [-] expose query API to peer actors [ariadne#199]
- [x] fleet thread inventory [ariadne#200]
- [-] cluster transport and queries [pair#147]
- [-] brain advisor role [pair#148]
- [ ] rescope to couch-lite [pair#170]
- [x] switch rule and key layer [pair#170 M1]
- [x] detach, and detached threads that reattach [pair#170 M2]
- [x] start or resume in a folder [pair#170 M3]
- [x] delete the machinery the rescope orphans [pair#170 M4]

<a id="pair-170"></a>
### pair#170 — rescope to couch-lite
**est:** 10.69
**status:** in progress — M1–M4 closed; plan at `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md`
**started:** 2026-09-02

Narrows couch to a switcher over a group of live coding sessions whose unit is a
**pair session**, not a terminal. Adds resume of a *live* session, `alt+d` detach
with detached sessions listed and reattachable, notification-focused switching,
and a single `previous` slot governed by one rule — `entered_via_notification`,
which makes a notification hop ephemeral so following a page never costs the
operator their place. Retires the actor-cluster scope and makes the machinery
that only defended it a deletion candidate.

Spec, the measured ctrl+backspace encodings, and the two `panelkeys.go` sites
that currently swallow that chord are in the issue.

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
**closed:** 2026-08-27
**actual:** 12.36h

Legacy registry metadata is journal-enriched into the existing ThreadStore
without moving source files or weakening conservative admission. One validated
`artifactpath` leaf now constructs every tag-bearing Pair path and exports exact
bindings to Go, shell, Neovim, and both layouts; a checked source/family manifest
classifies the ignored, deterministically regenerated runtime mirror without
tracking its duplicate bytes. A real two-scope strategy proves the
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

<a id="pair-155-m1"></a>
### pair#155 M1 — deterministic native forests

**est:** 7.85 (whole issue)
**closed:** 2026-08-28
**actual:** 3.83h

Pair now reconstructs complete, deterministic Claude, Codex, Agy, and Muse
native-session forests through one bounded runtime, retaining validated
descendants and explicit disputed or malformed orphans without treating native
parentage as Pair ownership. Sanitized fixtures, a stateful fake, fuzz seeds,
and a redacted live conformance scan pin the installed schemas. The surprising
part was that type-erasing shape inspection initially made Codex's `subagent`
object key look like a string source; a second type-preserving pass corrected
the allowlist before close. M2 can therefore focus on the actual preservation
boundary: establish a root only after one completed operator round, then let
native parent edges propagate that already-proven binding (ARCH-DRY,
ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

<a id="pair-155-m2"></a>
### pair#155 M2 — round-gated native bindings and public inventory

**est:** 7.85
**actual:** 4.17h
**closed:** 2026-08-28

Pair now captures a content-free launch baseline before input and establishes a
native root only after one unique completed operator-to-agent round. The same
pure matcher owns live watching and launch-delimited crash recovery; competing
roots remain ambiguous, parent edges only propagate an established binding,
and stale launch ordinals cannot persist. `pair session-inventory` exposes the
complete forests, bindings, ambiguities, and diagnostics as stable human or
schema-v1 JSON, with redacted live conformance. Worth preserving: a minted ID
or open transcript is invocation/corroboration state, not recovery authority;
there is almost nothing to preserve until native progress completes the round
(`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`).

<a id="pair-151-m1"></a>
### pair#151 M1 — actionable inventory and token-bound start authority

**est:** 7.60 (whole issue)
**actual:** 5.15h
**closed:** 2026-08-30

M1 introduces the proof-bearing live/verified-parked projection while retaining
undecodable lifecycle state in raw diagnostics. The transitional flat panel remains wired to raw `ThreadInventory` until M3 migrates that consumer. Public
start resolves policy, preference, and repository defaults into one
fingerprinted owner-local grant, revalidates it once, and admits and launches
only the accepted values. The surprise worth preserving is that an incumbent
from another policy epoch must fail immediately after candidate acceptance:
retrying cannot silently revise already-authorized authority.

<a id="pair-151-m2"></a>
### pair#151 M2 — pure hierarchical menu and scheduler

**est:** 7.60 (whole issue)
**actual:** 3.59h
**closed:** 2026-08-31

M2 supplies the inert pure hierarchy behind the still-flat Console: one shared
store-free matcher, identity-bound reducer/reconciliation stack, contained
wide/narrow renderer, semantic legacy/Kitty Tab key, and one-running/one-latest
preview schedule. The staged boundary remains explicit—the current Console
does not consume these components until M3. Worth preserving: cancellation is
a request, not a completion; only the matching terminal outcome frees the
running preview slot (ARCH-PURE, ARCH-PURPOSE, ARCH-CONSTRAINTS).

**Scope event 2026-08-30:** the first M2 boundary review expanded the pure-core
acceptance surface to cover operation presentation versus dispatch identity,
optional start-agent resolution and provenance, accepted-generation grant
reuse, exact completion origins, zero-match list frames, confirmation
filtering, parent-anchored geometry, and hidden-target label/location
diagnostics. The boundary remains M2; these are corrections to its promised
reducer/renderer semantics, not M3 Console integration.

<a id="pair-151-m3"></a>
### pair#151 M3 — Console integration and performance evidence

**est:** 7.60 (whole issue)
**closed:** 2026-08-31
**actual:** 11.18h

M3 makes the hierarchical switcher the reachable Console UI over the
proof-bearing actionable projection. Inventory and preference preview are
bounded asynchronous work, operation completions are correlated to exact frame
and attempt identities, and start/resume land only after transactional terminal
attach. A committed 100-row harness passes the M2 Max baseline plus two
four-worker co-tenancy trials; clean-store operator smoke covers park, exact
resume, Leave Couch, and terminal restoration. Raw lifecycle detail remains in
`couch --list` / `couch --show`, not in the two-state switcher (ARCH-CONSTRAINTS, ARCH-DRY,
ARCH-PURE, ARCH-PURPOSE).

## Log

### 2026-09-02 — scope event: rescoped to couch-lite

**Demoted from MVP**, punted rather than rejected: `pair#147` cluster transport
and queries, `pair#148` brain advisor role, `pair#153` managed-worktree
lifecycle, `ariadne#199` expose query API to peer actors. **Added:** `pair#170`
rescope to couch-lite.

**Why.** ~172 measured hours across twelve closed issues bought a switcher that
replaced the substrate — tabs became a switcher — without yet adding a
capability, and ~139 of those went into the switcher and identity layers. The
root cause is not ephemerality but the missing razor: without a clear view of
what couch was, the gap filled with generality — admission control, supervisor
leases, start grants, park transactions and fail-closed projections defending
one operator on one host across ~22k lines. The estimate ratios separate cleanly
along that seam: shape-unknown work ran #146 0.28x, #149 0.32x, #154 0.27x, #151
0.38x, while shape-known work ran #156 2.27x, #158 1.72x, #159 1.19x, #167
1.95x. pair avoided the same ambiguity by exposing CLI options and letting usage
reveal the right configuration; couch decided in advance and encoded the
decision in types, so every later discovery meant changing the ontology instead
of adding a flag.

**What the narrowed project is.** A switcher over a group of live coding
sessions, whose unit is a pair session rather than a terminal — that being the
one thing that cannot be bought, and the boundary that keeps the scope from
creeping back. One LLM stack, one path. No mesh, no relay, no cross-actor
queries.

**What this does not solve, stated plainly.** The project was opened for
*forgetting a thread exists*, with a dated cost: the rogii submission whose
2026-08-05 deadline passed unnoticed. A switcher does not catch that. Whether
couch-lite keeps a durable `{tree, what, when}` list plus a clock is open, and
is recorded in `pair#170`'s Log rather than assumed here.

Source: brain session 2026-09-02 working over
`brain/workshop/pensive/2026-08-20-01-pensive-couch-agent-switcher.md`.

### 2026-08-31 — pair#151 M3 architectural inventory completed

The fourth M3 review accepted the documentation behavior contracts and found
parked-resume proof missing from the Core-concept inventory. The inventory now
includes both proof inputs, native-binding/session-inventory resolution, and
exact repo-relative paths for every row. Production declaration markers derive
the delivered M3 set, while entity- and dependency-omission mutations enforce
it (`ARCH-PURPOSE`, `ARCH-MOCK`).

### 2026-08-31 — pair#151 M3 documentation boundary aligned

The third M3 review accepted refresh provenance and found the remaining
current-state contract gap. All Task 10–13 checkbox states are now parsed and
pinned; both Core-concept tables name the M3 boundary; and README's root Escape
sentence is executed against the reducer's no-live-actor behavior. Only the M3
and subsequent issue close remain open (`ARCH-PURPOSE`).

### 2026-08-31 — pair#151 M3 refresh provenance ready

The second M3 review accepted authority retirement and lifecycle timing, then
found a pre-mutation refresh could clear pending state before the dirty
post-mutation follow-up. Mutations now capture the latest admitted refresh
generation; only a strictly later successful snapshot authorizes their
projection, and failure preserves visible pending state. The authoritative M3
checklist now matches committed delivery, with only its boundary and issue
close open (`ARCH-PURE`, `ARCH-PURPOSE`).

### 2026-08-31 — pair#151 M3 first-review disposition ready

The first M3 boundary review's operation-consumer, superseded-authority, and
lifecycle-evidence findings are addressed across their full families. Mutating
operation success is visibly refresh-pending until actionable projection
converges; the flat compatibility authority is removed and absence-enforced;
and all six target latency paths now traverse the running Console to a
correlated emitted frame. Corrected M2 Max trials pass with exactly four joined
co-tenancy workers (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`,
`ARCH-CONSTRAINTS`).

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

<a id="pair-170-m1"></a>
### pair#170 M1 — switch rule and key layer

**est:** 10.69
**actual:** 0.36h
**closed:** 2026-09-02

`ctrl-space` now means one thing — open the switcher, focused on whoever paged —
and the child → root-actor → panel ladder is gone with the root-actor/home
concept. `ctrl+backspace` returns to `previous`, one slot governed by one boolean
carried on the *current* actor, so a notification hop never spends it.

Three things were more entangled than the plan expected, and each is the kind of
thing that ships silently broken. **`switchTo` owes two rules per landing, not
one:** record the landing, *and* acknowledge the landed actor's notifications.
Rule 2 had been living in the ctrl-space home-landing path this milestone
deleted, so `ctrl+backspace` home would have landed on a still-lit actor —
`NewestActor()` would then name the actor the operator is sitting in, and the
next `ctrl-space` would open on the wrong row. `previous` itself would have
worked perfectly; a test that only checked `previous` would have passed.
**Three landing sites, not one:** `installObservedThreadActor` lands the first
actor without passing through `switchTo` (so the tracker is seeded there, or the
starting actor is never recorded), and `onExit` must *drop* rather than record,
since the operator lands on the panel and a dead thread must never become the
return target. **`leave`'s confirmation was thread-bound by accident:** it rode
the root actor's live address, so five separate thread lookups passed — one of
them asynchronously, on the next inventory refresh. It became a global frame,
the shape the start form already used. That matters more once `leave` detaches
rather than parks: an all-detached couch is then the *normal* state to quit from.

Two smaller notes worth keeping. `alt+d` was deliberately deferred to M2 rather
than claimed here: intercepting it without wiring it would take Pair's own
detach chord from the child and give nothing back, leaving M1 less operable than
its predecessor. And #146's Core-concepts contract pinned `Focus` / `Up`; it was
revised at its source with a Revisions note rather than by loosening the test,
so the contract keeps defending the rows that are still true.

<a id="pair-170-m2"></a>
### pair#170 M2 — detach, and detached threads that reattach

**est:** 10.69
**actual:** 1.6h
**closed:** 2026-09-02

`alt+d` detaches: the Pair client and its sidecars go, the zellij server session
and the agent inside it stay, and the row remains in the switcher where `Enter`
reattaches it. `leave` detaches every thread instead of parking them, so quitting
couch no longer kills a running agent — which is what turns *detached* from an
edge case into the normal resting state.

**Detached is derived, not persisted**, and that was the design win: `launcher`
already classifies a zero-client live session as `SessionDetached` and
`pair resume` already reattaches onto one, so couch adds no `ThreadRecord` field
— it asks. The projector's detached branch requires **zero** incarnations, which
is the fail-closed property that keeps a crashed couch's stale `IncarnationLive`
from masquerading as a clean detach.

**What the plan got wrong and review caught, in order.** The first draft claimed
detach needed no durable transition at all; in fact `FinalizePark` is the only
path that removes a live incarnation, so detach would have left a dead-PID
incarnation forever — hiding the row *and* tripping `DecideResume`'s
occupied-incarnation gate. Hence `RetireIncarnation`. Then round two found that
reattach was blocked by a **second** verified-park refusal in
`ReconcileResumeAdmission`, reached after `DecideResume` returns, so widening one
gate would have listed a row whose Enter failed. And that a detached resume's
rollback would **delete the record** — the verified park had been the only
rollback authority, and an unnamed detached thread has none, so a post-claim
failure took the agent and argv needed to reattach while the session kept
running.

Two smaller things worth keeping. `reduceParkHotkey` returned no effects, so
`alt+d` decoded, reduced, and silently did nothing — caught only because the
test drove raw bytes through the production path rather than the reducer.
And the ordering decision (detach leads park in the action list: safe before
destructive) broke a dozen fixtures that navigated by counting `Down` presses;
they now select by name, because a fixture that silently retargets to a
different operation when a list is reordered is worse than the reorder.

The crashed-couch case is explicitly **not** solved here — `pair#171` owns it.
Conflating a crash with a clean exit is the exact fail-closed weakening the
projector refused to make.

<a id="pair-170-m4"></a>
### pair#170 M4 — delete the machinery the rescope orphans

**est:** 10.69
**actual:** measured at close
**closed:** 2026-09-02

Fleet admission and its cross-repo `sdlc fleet policy` provider, the start-grant
capability table, the legacy registry cutover, the never-instantiated actor
loop, and the registry-era dead surface. The razor was the issue's own:
machinery that exists only to defend multi-owner or multi-host cases. Applied
honestly it deleted less than the framing suggested — the supervisor lease, park
transaction, write-ahead journal and start transaction all defend *single-host*
failures and stayed, each with its reason written down.

The sweep's real content was not the deletions. Three things it surfaced:

**Two persisted schemas would have been bricked.** `threadrecord.Record` and the
store manifest ARE the on-disk format and are decoded with
`DisallowUnknownFields`, so removing a field makes every artifact still carrying
it undecodable. Measured against the operator's live store rather than reasoned
from the Go types: `claim_generation` in 17/17 records, `policy` in 5/5
incarnations, `legacy_cutover` and `legacy_migration_version` in the manifest.
The manifest is the worse half — a bad record loses one thread, a bad manifest
loses the whole store, because nothing can be listed, resumed or reattached at
all. All became decode-only tombstones with fixtures copied from real data, and
both guards were mutation-checked.

**A dead-code deletion exposed a live bug.** The start-grant token was genuinely
redundant, but removing it revealed that three call sites each rebuilt "the
arguments that reproduce this resolution" by hand. Getting it wrong is silent:
passing the resolved agent where the operator requested none changes
`AgentSource`, so the commit re-resolves to a different fingerprint and refuses a
drift that never happened. `StartResolution.CommitArgs` owns it now.

**Two tests were passing vacuously.** The post-acknowledgement cancellation test
asserted a hardcoded address that was never the one allocated, and asserted the
*opposite* of the correct behaviour — registration is established, so
occupied-or-proven-free keeps the record rather than deleting it. The entropy
shift from another deletion is what exposed it. A retargeted journal test had the
same defect and needed a second attempt before it reddened under mutation.

Tests were retargeted rather than deleted wherever they pinned surviving
behaviour; `IsLive` folded Unknown into false, so asserting `Liveness` directly
made the recycled-PID test stronger on its way out. One finding was recorded and
deliberately not acted on: the worktree `NamingTable` looks dead too, but cutting
it is wider than this sweep.

<a id="pair-170-m3"></a>
### pair#170 M3 — start or resume in a folder

**est:** 10.69
**actual:** 1.0h
**closed:** 2026-09-02

`couch` in a directory now reattaches a **detached** thread as readily as a
parked one. `SelectUniqueParkedRoot` becomes `SelectUniqueResumableRoot`, and the
rename is the deliverable rather than a side effect: a selector still called
`Parked` while selecting detached rows is a lie the next reader pays for.

The Spec asked for "live or parked", which needed interpreting rather than
implementing literally. couch holds its supervisor lease for the whole run, so at
*startup* there is no other couch hosting anything — a thread whose zellij
session is up with no client is exactly what M2 calls detached. "Live" therefore
means detached, and a genuinely live row is now never selected, because it would
be one this couch already hosts.

Exactness survived the widening, deliberately. A parked row and a detached row at
one path are TWO matches and start a new thread, exactly as two parked rows do.
Preferring warm over cold is a ranking policy, and #167 established that this
selector has none; #170 did not quietly add one. Recorded as a Revisions note on
#167's archived plan rather than by editing what it said it delivered.

The subtle part was proof, not code. The path physicalization M3's commit first
narrated as its own discovery had actually shipped at M2, where that review asked
for it; what M3 added is the alias-path test that pins it. Corrected here after
the M3 review caught the misattribution — a record that credits the wrong
milestone makes the next reader look for a change that is not in the window.

What M3 got genuinely wrong was the binding gate. Startup has no fallback by
design, so a row the inventory OFFERS must be one resume can take — and the
detached branch appended candidates before resolving the native binding that
parked rows were already gated on. A thread whose agent session data had been
pruned or raced was auto-selected and `couch` exited 1 with no way through, in
the tree the operator was standing in, for what M2 had just made the normal
resting state. Caught at the boundary as Critical, fixed as one gate for both
kinds, and pinned by the couchcore-level test whose absence was the reason
nothing caught it.

Both acceptance tests run through production interactive routing to initial
Console attach, and the reattach one is mutation-verified: narrowing the selector
back to parked-only reddens it.

[pair#170 M4]: #pair-170-m4
[pair#170 M3]: #pair-170-m3
[pair#170 M2]: #pair-170-m2
[pair#170 M1]: #pair-170-m1
[pair#146 M1]: #pair-146-m1
[pair#146 M2]: #pair-146-m2
[pair#146 M3]: #pair-146-m3
[pair#149 M1]: #pair-149-m1
[pair#149 M2]: #pair-149-m2
[pair#149 M3]: #pair-149-m3
[pair#149 M4]: #pair-149-m4
[pair#149 M5]: #pair-149-m5

### 2026-08-26 — pair#149 M5 second boundary disposition

The composite artifact sweep now covers complete companion sets and every
production source shape, while the session-binding relocation has one strict
legacy-plus-scoped reader across launch, claim, and quiescence. Verified
whole-incarnation capacity release remains deliberately in pair#152; #149 does
not reinterpret client exit or detach as session quiescence (ARCH-PURPOSE,
ARCH-MOCK).

### 2026-08-26 — pair#149 M5 constructor-closure disposition

Artifact enforcement now scans every production `cmd` package and refuses
constructor classification outside `artifactpath`; mutation fixtures pin both
the formerly omitted top-level-command case and an internal-package escape.
The atlas consistently presents exact companion bindings and the stable
changelog-ready marker rather than retired global derivation formulas
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-26 — pair#149 M5 generated-mirror correction

The runtime bundle remains an ignored generated mirror. Its deterministic
generator and drift tests preserve build confidence while keeping duplicate
generated bytes out of the milestone review window (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-26 — pair#149 M5 constructor and documentation closure

Selected-scope enumeration and parsing now remain inside `artifactpath` even
when the caller is already classified as a resolved consumer. Mutation tests
pin that label-bypass class, and the documentation inventory distinguishes
exact current bindings from descriptive or compatibility filename vocabulary
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 tracked-input constructor proof

Constructor closure is now based on the absence of unapproved family literals,
not a list of Go expression forms. Generated-path coverage builds its own
temporary mirror from tracked inputs, so a clean checkout and a developer tree
exercise the same proof (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 positive derivation and clean bootstrap

The source manifest now proves each resolved family through a named
`artifactpath` resolver/member witness; closed vocabulary allowances account
only for exact non-path protocol and CLI uses. The clean bootstrap is an
executable public contract: from an archive with no Git metadata and no
generated mirror, `make test` generates first and passes the complete suite.
Compatibility restart/quit markers use the same path authority with a safe
Unicode-basename contract for public Zellij session names (ARCH-DRY,
ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-27 — pair#149 M5 exhaustive production participation

Every production source now declares artifact participation or explicit
non-participation, so a new or split-token file cannot bypass discovery. Binding
witnesses follow the resolver object's lexical identity into a non-discarded
family-member use. The clean-bootstrap oracle also handles the clean-HEAD
empty-patch case, and the milestone entity table has been reswept across the
whole M5 diff (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-27 — pair#149 M5 exclusive derivation and concept contract

Constant concatenation is checked independently for every resolved consumer,
so a valid family witness cannot hide a second constructor in the same source.
The complete M5 Core concepts and integration inventory is now an executable
exact-row contract with deletion and field-mutation coverage (ARCH-DRY,
ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 runtime assembly and derived inventory

The constructor guard now assembles runtime literal fragments across calls and
function bodies, covering joins and builders beside otherwise valid witnesses.
The artifact Core concepts contract derives its entity set directly from the
M5-created package's exported type/catalog declarations, including `Families`
and `SourceClassifications`, then mutates every plan consumer field
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 order-independent proof and diff-wide dispositions

Artifact constructor enforcement now follows runtime string provenance through
variable use order and local helpers, with reversed-order and cross-helper
mutations. The M5 concept contract classifies every declaration in the complete
milestone Go diff and derives the plan-visible architectural subset from
source-local markers, including pure, seam, and integration entities outside
`artifactpath` (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 package provenance and fail-closed authorities

The constructor proof now includes package constants/variables and aliases in
the same fixed point as local helpers and runtime composition. Exported types
and catalog variables in the M5 artifact authority are concept-by-default, so a
new unmarked public authority cannot opt out of the plan inventory
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 package-wide provenance and closed declarations

Artifact provenance now crosses Go files within each package. The concept
inventory closes the complete M5 declaration population with a stable AST
signature, so any new declaration requires an explicit concept/detail
disposition and exported additions fail closed across all M5 sources
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 proof boundary correction

Operator review rejected the emergent requirement to prove artifact-string
provenance for arbitrary future Go programs. M5 retains exhaustive current-
source participation, constructor-location enforcement, positive per-family
bindings, bounded literal checks, and cross-scope consumer integration. The
custom package dataflow evaluator is removed; a typed filesystem capability or
SSA analyzer would be separate future work if ever justified (Simplicity
First, ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — pair#149 M5 finite importer closure

The bounded repository contract now requires every Go source importing
`artifactpath` to be positively classified; the twenty existing importers have
exact family and resolver witnesses. Legacy migration and current rename share
one authority-owned sidecar shape without confusing legacy and scoped roots,
and the atlas no longer advertises the deleted open-ended provenance analyzer
(ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-27 — pair#149 M5 documentation regression

The artifact documentation sweep now pins Couch's bounded-analysis statement
and rejects the retired package-dataflow claims, so the atlas cannot silently
drift back to behavior the implementation no longer provides (ARCH-PURPOSE,
ARCH-MOCK).

### 2026-08-27 — pair#154 Pair/Couch ownership scope correction

The #149 M3/M5 entries above are retained as the historical record of what
those boundaries claimed at close. #154 corrects that scope: standalone Pair
has no ThreadIndex reader or ThreadStore registrar/upsert path. Pair owns scoped
address claims, artifacts, ledgers, session bindings, and exact-tag resume;
Couch independently owns ThreadStore lifecycle, admission, metadata, and
human-name/path resolution. A hosted start coordinates the two only through the
reserved→established Pair marker: Pair does not mutate Couch records, and Couch
alone promotes the expected helper from creating to live (ARCH-DRY,
ARCH-PURPOSE, ARCH-PURE).

### 2026-08-28 — scope event: native sessions become complete trees

Pair's current launch watcher discovers one root transcript, then stops; its
fallback may select the newest candidate, while Codex and Muse subagents are
excluded rather than modeled. Added #155 to the MVP before #152: it inventories
each supported agent's complete root/subagent forest and correlates roots to
Pair tags through explicit config, ledger, and identity-authorized live-process
evidence. Ambiguity remains visible and unbound; chronology orders candidates
but never authorizes one. #152 now treats the durable repo plus this identified
native session tree as recovery state and no longer waits for an LLM-authored
flush acknowledgement. #152 and its consumer #151 are marked blocked until the
shared inventory lands (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-30 — pair#152 verified park/resume delivered

Pair Alt+x and Couch Park now share one nonce-bound full-quit lifecycle;
ThreadStore releases capacity only after the matching durable completion and
final CAS. Couch confirmation/`parking…` precedes bounded asynchronous work,
and Resume re-enters the exact composite address, saved profile, working path,
and #155 native root through a read-only established Pair marker. Alt+d remains
Pair-local detach and there is no Couch detach surface. The #151 menu can now
consume these operations without inventing lifecycle semantics
(ARCH-CONSTRAINTS, ARCH-DRY, ARCH-PURPOSE).

### 2026-09-01 — pair#159 makes the TUI Couch's public interface

Bare `couch` opens the current directory and `couch <path>` opens another path.
Only `--list` and `--show <ref>` remain as public diagnostics; lifecycle and
metadata actions live in the TUI's typed in-process dispatcher. The one hosted
agent process boundary uses hidden `--internal publish-description`, so adding
a typed operation cannot accidentally grow the public CLI (`ARCH-PURPOSE`,
`ARCH-DRY`, `ARCH-CONSTRAINTS`).

[pair#155 M1]: #pair-155-m1
[pair#155 M2]: #pair-155-m2
[pair#151 M1]: #pair-151-m1
[pair#151 M2]: #pair-151-m2
