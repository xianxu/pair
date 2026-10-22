---
id: 000806
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: high
---

# Replacement Zellij disconnects before first wrapper startup

## Problem

A disposable public reboot completed exact old-session retirement, but its replacement Zellij session disconnected before the first wrapper startup/agent-ready publication. Pair exited1 before Couch registration. This is distinct from #797's ownership refusal and quiescence ordering correction. Cause is unclassified; no flake is registered. It may relate to #773's missed-ready startup evidence, but no shared root cause is established.
