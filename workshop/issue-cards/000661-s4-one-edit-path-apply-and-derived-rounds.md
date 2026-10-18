---
id: 000661
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply

## Problem

Slice 4 of the universal-agent-input-interface project (design: `ops:teams/devinfra/projects/universal-agent-input-interface/design.side/10-s4-edit-path.md`, sections 3, 5 and 8). Today an agent changes a document in two ways that don't meet: `couch --session-sync write` rewrites a body or section with no record of what changed, and the review pane's nvim applies `{old, occurrence, new, explain}` records itself, dropping the ones that fail. S4 puts every agent edit on one Couch path and keeps each round so parley can show it.
