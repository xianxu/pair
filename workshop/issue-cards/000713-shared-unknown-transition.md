---
id: 000713
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
severity: low
---

# Consolidate exact incarnation unknown transition

## Problem

#712 close review BR-3 (Minor, single-incarnation-transition-authority) found that Couch.persistUnconfirmedStart repeats the exact process-identity match, live-to-unknown mutation and revision-conflict loop in ThreadStore.MarkIncarnationUnknown. Both currently preserve ownership correctly, but two authorities can drift.
