---
id: 000585
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Preserve cloudflared stderr on early process exit

## Problem

The full recorded suite intermittently loses the fake cloudflared process's stderr failure before reporting its exit. This is independent of #571's Codex adapter; no real tunnel or network is used by this test.
