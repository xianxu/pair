---
id: 000800
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: low
---

# Bind runtime jobs to invocations for proven idle attribution

## Problem

ariadne#547 found that a retained landing ticket proves a command was blocked, but cannot prove its transcript source was idle. pair#778's retained car waiter was counted as focused work while it polled another conductor's train; its historical adopted actual must not be rewritten from guesses. Ticket start/bind/end facts will expose counted unverified coverage in ariadne#547, without subtracting time.

Claude tool IDs and Codex call IDs pair invocations/results, while queue/ticket records carry operation ID and PID incarnation. There is no structural binding between these identities and the harness runtime job. An outer Codex exec call can contain arbitrary mixed work; parsing shell/JavaScript command text or guessing from tool output cannot establish pure observation.
