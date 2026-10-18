---
id: 000666
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Guard native fixture root handoffs

## Problem

#636 fixed live recovery/park fixtures that accidentally lost disposable native socket and runtime roots: testpair.ApplyEnvironment clears inherited values, and nested helper environments need explicit propagation. The local source guard prevents recurrence only at its named entrypoints. Capture this Pair-specific mistake class in the repository lint path without making the shared catalog depend on Pair helper names.
