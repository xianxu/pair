---
id: 000276
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
---

# Surface couch-tagged agents that have no thread record

## Problem

`ReasonUnrecordedChild` ("running but unrecorded") exists in the classifier's
vocabulary, but it is reached **per record** — so an agent running with *no*
record at all is invisible rather than reported. `pair#272` named the case and
its `## Done when` carries the bullet; split out of `pair#256` on 2026-09-16 when
that plan was re-cut, because this is an **additive discovery feature**, not a
lifecycle-authority fix, and it was the largest remaining piece of scope that did
not repair a broken state.

Live fixtures, running since 2026-08-30 and still present after the operator's
2026-09-16 cleanup:

| Session | `pair wrap` | Conversation | Tag |
|---|---|---|---|
| `📁parley-couch` | 84488 | `claude --session-id 44e6ec1b…` | `couch-797c45e8e649a9bb` |
| `📁parley-couch-2` | 1130 | `claude --session-id 9a0a197f…` | `couch-2583ed61c0ab6ebe` |

Both hold intact conversations couch cannot see. Also present: `87464`
`📁brain-couch-2`, a zellij server whose `pair wrap` child is gone — the
agent-died-but-shell-survived shape, which any listing here must not present as
reattachable work.
