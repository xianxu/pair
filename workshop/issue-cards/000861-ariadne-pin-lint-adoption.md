---
id: 000861
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
---

# Adopt Ariadne shared sidedoc pin and qualify newly active lint rules

## Problem

Pair#838 requires the landed shared sidedoc API from ariadne#575, #576 and #581. Main still pins ariadne 81c832d5; the required pin 5a17f789619d also activates newer golint rules and changes Rule.Match to accept *CheckContext. This blocks #838 pre-close and its joint operator stage.
