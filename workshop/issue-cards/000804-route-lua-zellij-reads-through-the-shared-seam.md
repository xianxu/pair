---
id: 000804
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
---

# Route Lua Zellij reads through the shared seam

## Problem

#803 centralizes bounded Zellij read recovery for Go consumers, but the Neovim
Lua consumers still execute Zellij directly and can receive the same empty or
crossed reply. The remaining exclusion must not become another copied retry loop.
