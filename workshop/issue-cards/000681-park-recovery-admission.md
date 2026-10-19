---
id: 000681
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: high
---

# Recover park inventory exceeds CLI admission deadline

## Problem

On the operator build containing #651, confirmed CLI recover-park can time out before receiving a receipt because it builds the full fleet inventory inside a two-second admission exchange. The same inventory during ordinary resume preparation took 3.055 and 5.233 seconds. Two requests returned context deadline exceeded / outcome uncertain, but caller-scoped operation-status returned unknown for both; neither recovery ran. This strands an otherwise recoverable interrupted park behind its named CLI recovery command.
