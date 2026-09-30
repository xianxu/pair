---
id: 000352
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '87ac516c44826d5ae327a1cb76eb9ab6ecf2bbf1' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --notify: stateful operator notifications

## Problem

Agents signal the operator ("I need input", "ready for you to look") through
OSC terminal notifications. Each agent CLI emits them differently, they are
fire-and-forget, and nothing remembers which slot is waiting on the operator.
With several Couch slots running, the operator's verification attention is
the serial resource (ariadne#272), and there is no view of which slots are
waiting for it.

## Spec

- `couch --notify "<message>"` (callable from any shell inside a Couch slot)
  records a notification against the calling slot: kind (`needs-input`,
  `smoke-ready`), optional issue ref (e.g. `#333`), message, timestamp.
- Notifications are stateful: they stay pending until the operator visits the
  slot or answers, then are acknowledged. Pending notifications across slots
  form the operator's verification inbox; any inbox UI is secondary.
- The Couch skill instructs agents to use `couch --notify` instead of OSC when
  they need operator input or a smoke test is ready (e.g. "smoke test of #333
  ready"). Agents learn they run inside Couch from an environment marker plus
  the skill.
- Lifecycle (ARCH-FUNERAL): a notification ends at acknowledgment or when its
  slot's session ends; nothing durable outlives Couch unless the design says so.

## Done when

- An agent in a Couch slot can run `couch --notify` and the notification is
  visible to the operator from Couch, tied to its slot and issue.
- Visiting/answering the slot acknowledges it; a test covers raise → ack.
- The Couch skill documents when to notify, replacing OSC guidance.

## Plan

- [ ] Design the notification record, ack trigger, and where Couch surfaces it.

## Log

### 2026-09-29

Filed from ariadne#272's brainstorm (ban stacked development; slots, not
stacks, buffer work in front of the operator's smoke test). Companion to
pair#353 (live cross-slot dispatch), which can land independently.
