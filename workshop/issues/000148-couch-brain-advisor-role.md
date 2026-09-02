---
id: 000148
status: punt
deps: [ariadne#200]
github_issue:
created: 2026-08-21
updated: 2026-09-02
estimate_hours:
---

# couch: brain advisor role

Project: `workshop/projects/couch.md` — architecture and non-goals live there;
this issue is task 4.

## Problem

The failure this project exists to fix is **forgetting a thread exists** — not
mis-ranking threads. A registry, a switcher and a transport give the operator the
machinery but not the answer to "what is going on, and what needs me." That
synthesis is judgment, and it is the one part a deterministic shell cannot do.

## Spec

**brain is a role over files, not a long-lived process.** Portfolio state lives
in the registry, the transport and fleet inventory — never in a session's
context, which compaction and crashes lose. Any brain session picks the role up
by reading them, which makes "always available" free.

**Derived on read, never stored.** The moment the advisor maintains a document of
issues/milestones/progress it has rebuilt the artifact that rots first — measured
evidence: two metis project files froze on 2026-07-22, before the phase that
mattered.

**Enumerate on measured facts, never self-declared status.** Consumes
`ariadne#200`. A sibling repo's issue frontmatter read `status: working` for a
month while the real work happened elsewhere; git said `0 ahead, last touched
2026-07-23` and was correct.

**Two staleness signals, both surfaced and distinct:** mailbox depth says someone
is waiting on this agent; git staleness says the thread has gone cold. The thread
nobody has messaged and nobody has touched is the one that dies.

**Router, not relay.** brain says *which* session and switches the operator
there; the operator types into the coding session directly. Relaying a
paraphrased work instruction through an LLM with its own context is a telephone
game. Control-plane traffic (notifications, park, summarize) relays fine — it is
structured and lossless.

**Fuzzy in, exact out.** "Switch to the one where we're refactoring pair"
resolves to a canonical id, and resolution **returns what it resolved to**, shown
before anything fires. What it matches against is the runtime naming layer —
operator-assigned short names plus the agent's own one-line description — not an
issue title, so it reflects what the agent is actually doing rather than what a
ticket claimed. Duplicate or stale labels are expected; resolution asks which. This matters most for destructive operations: "stop it"
hitting the wrong session is the failure that would make the whole thing
untrusted.

**"What was that notification about?" is answered by a tool call**, not by
reading system messages out of the transcript. The status row (`#146`) carries
the signal; the advisor queries the same operation surface everything else uses
to explain it. This keeps notification volume off the LLM's context budget while
leaving the detail one question away.

**Silence detection** narrows to actors that are not running at all — a running
actor's shell reports its own mailbox depth and fires its own timers, so most of
what an external observer used to be needed for is now intrinsic.

## Revisions

### 2026-08-26 — distribute a thin advisor skill, keep authority in Pair

**Reason:** the operator requires every operation available through the human
panel to be available to the root agent as well, including thread lookup,
naming, rename, description, start, attach/switch, and later lifecycle actions.
The root role is conventional rather than brain-specific, so a brain-only skill
would put the adapter at the wrong boundary.

**Delta:** Ariadne distributes a shared couch-advisor skill to agent repos. The
skill teaches when to enumerate, how to resolve fuzzy human references, when to
show the exact composite target and request confirmation, and how to invoke the
local couch client. It contains no authoritative verb list or concurrency
policy: Pair's discoverable `couchcore.Operations()` schema remains the source
for names, typed arguments, effect class, owner requirement, fallback safety,
and confirmation requirement. Adding a human panel action without declaring it
there remains a conformance failure, so human and root-agent capabilities cannot
drift (ARCH-DRY, ARCH-PURPOSE).

The skill operates only in the root actor's inherited couch namespace and sends
owner-required calls through #147's local endpoint. It never discovers another
`COUCH_STORE_DIR`, executes a peer actor's repo commands itself, or turns fuzzy
text directly into a destructive call. Pair owns behavior and exact results;
Ariadne owns portable skill distribution and adaptation to supported harnesses.

Every effectful human interaction is a declared operation, including implicit
key behavior such as Enter on a live row. Pair declares explicit owner-required
`switch` and `attach` operations rather than letting the console call a private
`forceSwitch` path. A conformance test enumerates keyed actions, implicit row
actions, CLI dispatch, and the generic advisor client and requires all four to
resolve through the same schema. Rendering/filter/navigation may remain local
pure UI behavior; creating, attaching, switching, naming, describing, and later
lifecycle transitions may not (ARCH-DRY, ARCH-PURPOSE).

## Done when

- A brain session answers "what is going on" from precomputed state fast enough
  to beat looking at tabs, reviving an actor only when interpretation is needed.
- A fuzzy natural-language reference resolves to a canonical id that is shown
  before any switch or stop takes effect.
- A thread nobody has messaged and nobody has touched is surfaced as at-risk
  without the operator asking.
- Notifications surface to the operator while attached elsewhere, as a queue —
  not a dashboard.
- No portfolio state is stored as a maintained document.
- Every effectful operation available to the human, including implicit
  Enter-to-switch/attach, is callable through the root advisor's generic client.

## Plan

- [ ] Advisor reads registry + inventory + mailbox facts; no stored charter.
- [ ] Fuzzy reference resolution returning a shown canonical id.
- [ ] Ariadne-distributed couch-advisor skill derived from Pair's operation
      schema, with target display and confirmation semantics.
- [ ] Notification surfacing as an escalation queue.
- [ ] Silence/at-risk detection for not-running actors.
- [ ] Router-not-relay switching from the advisor.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project. Depends on
`ariadne#200` for measured fleet inventory.

### 2026-09-02

Punted by the couch-lite rescope (`pair#170`). couch narrows to a switcher over
a group of live coding sessions — one LLM stack, one path, no cluster transport,
no advisor, no managed worktrees. This is deferral, not rejection: the spec above
stands and the scope event in `workshop/projects/couch.md` records why it stopped
being the next thing to build.
