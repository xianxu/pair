---
id: 000148
status: open
deps: [ariadne#200]
github_issue:
created: 2026-08-21
updated: 2026-08-21
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
before anything fires. This matters most for destructive operations: "stop it"
hitting the wrong session is the failure that would make the whole thing
untrusted.

**Silence detection** narrows to actors that are not running at all — a running
actor's shell reports its own mailbox depth and fires its own timers, so most of
what an external observer used to be needed for is now intrinsic.

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

## Plan

- [ ] Advisor reads registry + inventory + mailbox facts; no stored charter.
- [ ] Fuzzy reference resolution returning a shown canonical id.
- [ ] Notification surfacing as an escalation queue.
- [ ] Silence/at-risk detection for not-running actors.
- [ ] Router-not-relay switching from the advisor.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project. Depends on
`ariadne#200` for measured fleet inventory.
