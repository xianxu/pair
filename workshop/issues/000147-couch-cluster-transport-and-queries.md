---
id: 000147
status: open
deps: [ariadne#199]
github_issue:
created: 2026-08-21
updated: 2026-08-21
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

## Done when

- A codex-stack actor and a claude-stack actor exchange control-plane messages
  over the same channel, with no agent-specific code path.
- An agent-emitted notification reaches brain while the operator is attached to a
  different child.
- A scheduled `deadline` notification fires unattended and is delivered.
- A query routes to the owning actor's shell, runs the binary in that repo root,
  and returns vocabulary-shaped JSON — with no LLM turn and no approval prompt.
- A verb absent from an actor's manifest cannot be invoked through the transport.
- Daemon-count question against #121 answered and recorded.

## Plan

- [ ] Thread-addressed message routing through the pair layer; eager spawn.
- [ ] Deadlines on messages; timers owned by the shell; non-blocking send.
- [ ] Query routing to the owning actor's shell; manifest fetch + cache.
- [ ] Notification contract as an agent-invocable tool call.
- [ ] Scheduled `deadline` notifications.
- [ ] Codex-stack participation proven end to end.
- [ ] Resolve daemon count vs #121.

## Log

### 2026-08-21

Split out of the former root ticket on promotion to a project. Depends on
`ariadne#199` for the exposed query surface.
