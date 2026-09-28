---
id: '000115'
status: done
started: 2026-07-16T12:17:57-07:00
created: 2026-07-16
updated: 2026-08-16
estimate_hours: 4.41
actual_hours: N/A
---

# Switch the agent driving existing work

## Problem

When an agent provider is degraded, the user cannot smoothly move live work to
another coding agent. Pair treats the live agent session as if it were the work:
its normal picker hides attached sessions, choosing a different agent tends to
allocate a sibling tag, and conversational state remains trapped in the source
agent's transcript. A sibling `*-resurrect` tag would fragment one body of work
across identities and would require copying state that already belongs to the
original tag.

The tag should identify the work; Claude, Codex, or another agent should be an
exclusive, replaceable driver. Switching drivers must retain the draft pane,
sent-prompt history, future queue, native per-agent conversations, and the
human-meaningful context distilled through a continuation.
