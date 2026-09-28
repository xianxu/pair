---
id: '000040'
status: done
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 2.0
actual_hours: 1.5
---

# Remove gemini CLI support

## Problem

The old standalone `gemini` CLI is deprecated in favor of `agy` (Antigravity). We need to remove all deprecated `gemini` CLI integration logic, session watchers, keymap overrides, parsers, and tests from the codebase, while keeping `agy` fully intact. Because `agy` shares the `~/.gemini/` home namespace (under `~/.gemini/antigravity-cli/`), any deletion sweeps must be highly selective to prevent breaking the `agy` agent.
