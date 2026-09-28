---
id: 000135
status: open
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
github_issue:
---

# Live cross-agent handoff

## Problem

The revived #115 path lets a user switch an exited/recent tag to another agent
through continuation-backed recovery, but it deliberately avoids taking over a
currently live session owned by a different agent. The abandoned live handoff
coordinator proved that the shape is useful, but its production quiescence proof
was unsound: the acceptance fake modeled cleanup behavior real zellij did not
provide, so the source could be destroyed and then time out.

Pair still needs a safe live handoff for the case where a provider is degraded,
quota is exhausted, or the current agent can no longer produce the continuation
document itself. The tag should remain the work identity while the source agent
is quiesced, its Pair scrollback/state is parked, and the target agent starts as
the exclusive driver under the same tag.
