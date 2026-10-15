---
id: 000575
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Claude slots: couch --send-to fails inside Claude's sandbox (broker socket not allowed); fix the slot's sandbox settings

## Problem

On Claude Code slots, `couch --send-to …` run inside Claude's default sandbox times out ("outcome uncertain; no
recipient retains it"), although inbound delivery to the same slot works; it succeeds only when the agent runs it with
the sandbox disabled (reported by ariadne:2 on 2026-10-15; the TL on ops:0 also runs every couch call with the sandbox
off). So the per-call sandbox exemption pair installs for Claude slots doesn't match `couch` sends, and every worker
report from a Claude slot either fails or needs a sandbox override, which the agent must justify each time.
