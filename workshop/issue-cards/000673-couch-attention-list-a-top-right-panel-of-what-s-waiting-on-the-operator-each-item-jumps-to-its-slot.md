---
id: 000673
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot

## Problem

Ghostty skips OSC desktop notifications sent from the focused tab (confirmed by the operator 2026-10-18), and Couch's hub is nearly always the focused tab, so `couch --notify` never reaches the operator in practice. The operator wants Couch to manage attention itself.
