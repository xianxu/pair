---
id: 000217
status: open
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
github_issue:
---

# dim the right pane's tab strip when the pane loses focus

## Problem

The tab strip (`#199`) renders identically whether the right pane has focus or
not, so it competes for attention while the operator is typing in the draft
pane. The operator wants it muted when unfocused, the way the draft's statusline
reads as secondary information rather than a loud bar.
