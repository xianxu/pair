---
id: 000786
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# Expose picker state and prompt in Couch actors JSON

## Problem

ariadne#533 needs to distinguish a slot blocked behind a picker from an idle agent. Couch already receives the wrapper's picker detection as registry SlotLiveness.Picker, but couch --actors --json does not expose it. Its Resting field means resting Git branch, not agent prompt readiness.
