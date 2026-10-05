---
id: 000392
status: open
deps: []
github_issue:
created: 2026-10-05
updated: 2026-10-05
estimate_hours:
card_mirror: '30eec0125306ab2aefcc8651b157888a67b3c6e9' # card fields mirrored from issue-cards; edit via sdlc
---

# Review pane: notify when an agent round comes back empty

## Problem

When the operator sends "finished my edits… please review" and the agent has
nothing to change, the agent replies with an empty handoff (`records: []`). The
pane lands it silently: the spinner stops, but nothing tells the operator that
the round was a no-op. In practice an empty round means the agent considers the
draft done, which is exactly the moment the operator wants to decide whether to
ship. Observed 2026-10-05 reviewing xianxu.dev `you-decide.md`: the final
review round returned empty and the only signal was the agent's chat reply.

## Spec

## Done when

- When a handoff with zero records lands for the active review, the pane shows
  an info notification (e.g. "No edits proposed — draft looks ready to ship")
  instead of landing silently.
- The notice appears only for an empty round, not when records were sent but
  all were dropped or reconciled (that case keeps its existing messaging).
- A test covers an empty handoff producing the notice, and a non-empty handoff
  not producing it.

## Plan

- [ ]

## Log

### 2026-10-05
