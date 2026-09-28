---
id: 000264
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# Fix Codex changelog refresh failure

## Problem

The change-log view does not work reliably for Codex sessions. On refresh, the
viewer displayed the visible error:

`change log refresh failed: model: codex exec failed: exit status 1: OpenAI Codex v0.154.0`

The viewer should remain useful even when the model command fails, but the
current path surfaces only the failure and does not produce a changelog.
