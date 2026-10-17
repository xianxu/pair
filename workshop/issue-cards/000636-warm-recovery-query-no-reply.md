---
id: 000636
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Warm recovery live conformance intermittently receives no Zellij query reply

## Problem

Native Zellij 0.45.1 occasionally returns no reply to a recovery fixture query while the exact owned session remains alive. The pre-existing warm live recovery test fails before fixture cleanup. This is separate from #627's daemon/cache cleanup join.
