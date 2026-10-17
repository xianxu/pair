---
id: 000619
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354)

## Problem

parley.nvim#354 (the attention finder, part of ops project universal-agent-input-interface) binds **Alt+u** (M-u) in
parley to open the list of documents waiting on or answered by an agent. Couch's key help
(`cmd/internal/keyhelp/catalog.go`) doesn't list it, so the operator can't discover it from Couch.
