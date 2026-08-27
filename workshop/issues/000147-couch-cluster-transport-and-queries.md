---
id: 000147
status: open
deps: [ariadne#199]
github_issue:
created: 2026-08-21
updated: 2026-08-26
estimate_hours:
---

# couch: cluster transport and queries

Project: `workshop/projects/couch.md` — architecture and non-goals live there;
this issue is task 3.

## Problem

Actors that cannot talk are just supervised processes. The cluster needs message
passing that works when the recipient is usually offline, and a way for one actor
to ask another about its state without the caller reaching into the callee's repo
and without spending an LLM turn or a human approval.

The Claude-native peer roster is not that channel: it covers only live ∧ Claude ∧
reachable, missing codex sessions entirely and every parked thread, and a single
status query was held for human approval on both ends.

## Spec

**Transport lives in the pair layer**, so it is agent-stack-agnostic rather than
Claude-specific. A codex-backed actor participates over the same channel as a
Claude-backed one — no agent-specific path.

**Mail, not chat.** Messages are addressed to the **working tree** (durable), not
the session (ephemeral) and not an issue ref. v1 semantics, decided deliberately: mailboxes are ephemeral,
and **eager spawn** is the only mode — no actor for a message means start one.
Accepted cost: a low-value FYI can trigger a cold boot. If couch restarts,
in-flight messages are lost.

**Every message carries a deadline, and timers belong to the shell.** The LLM is
suspendable by construction so it can never own a real-time obligation.
Back-pressure is not a separate mechanism: an undeliverable message is the same
event to the sender as a reply that never arrived, handled by the sender's shell.

**Queries, not commands.** The dialect is shared
(`construct/vocabulary/*.cue`); the authority is not — actor B invoking `sdlc
close` in A's repo is permission laundering. couch hands `{query, verb, args}` to
the *owning* actor's shell, which runs the binary in its own repo root under its
own permissions and returns JSON. The caller never runs the binary. Consumes
`ariadne#199`'s annotation + `--expose-manifest` + vocabulary-derived shapes;
couch caches each actor's manifest.

**The notification contract**, emitted by agents as a tool call, never inferred
from screen contents: `{thread, kind: done | blocked | needs-decision | deadline,
one line, how-to-attach}`. Screen-scraped recognition is ruled out on evidence
from this repo — #139 estimated 5.83h and took 23.40h because recognizers built
from a startup capture rejected ordinary composing states, and #138 reintroduced
the same bug class. A missed page is silent, which makes it the wrong place to be
clever.

The `deadline` kind is separate because it is the one class an agent can identify
with total reliability, and it is the one that would have caught a real
2026-08-05 miss.

**This issue delivers notifications to the root actor; rendering them is
`#146`'s reserved status row.** They are deliberately *not* injected into the
transcript as system messages — that would put every notification into the LLM's
context window and distract the model on every turn. The row signals that
something happened; detail comes from a tool call when the operator asks.

**Liveness is a notification source too.** The root actor's shell monitors peer
actors and emits on transitions it observes — an actor exiting, or going
unreachable — so a child dying is visible without the operator being attached to
it.

**Check daemon count against `#121`** — two long-lived processes inside the pair
trust boundary both claiming authority over session lifecycle would be a smell,
even with disjoint command surfaces. #121 solves a different problem (reaching a
live session from a phone browser); the question is only whether they should
share a process.

**Read-only star.** brain asks, work sessions answer, nobody else initiates. No
agent→agent write channel until a message worth sending can be named. Single
host.

## Revisions

### 2026-08-26 — route control through the namespace owner

**Reason:** the work-thread design in #149 makes one canonical couch store a
durable namespace with exactly one live supervisor. A root agent invoking an
actor-creating operation in a separate CLI process must not create a child that
the supervising console cannot attach or route.

**Delta:** #147's local transport also exposes Pair's discoverable couch
operation schema through the live namespace owner. Structured clients send an
operation name, typed arguments, caller/namespace identity, and an optional
expected thread revision. The owner resolves and executes actor-creating,
attach, switch, and other console-local operations. Durable read/metadata
operations may use the same endpoint; if the owner is absent they may fall back
to a direct locked ThreadStore transaction only when the operation declaration
marks that behavior safe. An owner-required call never silently forks locally.

The endpoint is local to the canonical `COUCH_STORE_DIR`; it does not discover
or bridge alternate stores. Authorization remains the existing single-host,
single-operator trust boundary. Responses return the exact composite thread
address and revision so fuzzy resolution never becomes implicit execution.

### 2026-08-26 — address transport to work threads, never paths

**Reason:** #149 permits several durable work threads at one physical path, so
the original working-tree mailbox address cannot select an owning actor. It
would route a query or notification nondeterministically as soon as Brain hosts
two threads in place.

**Delta:** the earlier working-tree addressing contract is superseded in full.
Messages, mailboxes, queries, replies, notifications, deadlines, eager revival,
and cached actor manifests are keyed by the exact composite
`{repo_scope, tag}` work-thread address within the inherited couch namespace.
The current path is an attribute used for discovery and policy admission only;
it is never a delivery address. A client may request **create a new thread at
path**, which returns a new composite address, or **operate on/revive this exact
thread**, which requires an existing composite address. These are distinct
operations and no path-based fallback guesses between threads.

The owner routes a query to the live incarnation referenced by that thread. If
no incarnation exists, eager revival means reviving that exact durable thread,
never creating a replacement at its path. Verified revival remains gated on
#152; until then the result is a typed unavailable response rather than a fresh
thread. Manifest caches and pending deadlines are invalidated/rekeyed by thread
revision and cannot bleed between equal-path threads (ARCH-PURPOSE).

## Done when

- A codex-stack actor and a claude-stack actor exchange control-plane messages
  over the same channel, with no agent-specific code path.
- An agent-emitted notification reaches brain while the operator is attached to a
  different child.
- A scheduled `deadline` notification fires unattended and is delivered.
- A query routes to the owning actor's shell, runs the binary in that repo root,
  and returns vocabulary-shaped JSON — with no LLM turn and no approval prompt.
- A verb absent from an actor's manifest cannot be invoked through the transport.
- Two work threads sharing one physical path have distinct mailboxes, manifest
  caches, queries, and notifications, and routing reaches only the selected tag.
- Daemon-count question against #121 answered and recorded.

## Plan

- [ ] Composite-work-thread-addressed message routing through the pair layer;
      exact-thread eager revival after #152, never path-based replacement.
- [ ] Deadlines on messages; timers owned by the shell; non-blocking send.
- [ ] Query routing to the owning actor's shell; manifest fetch + cache.
- [ ] Root/client operation routing to the live couch namespace owner.
- [ ] Notification contract as an agent-invocable tool call.
- [ ] Scheduled `deadline` notifications.
- [ ] Codex-stack participation proven end to end.
- [ ] Resolve daemon count vs #121.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project. Depends on
`ariadne#199` for the exposed query surface.
