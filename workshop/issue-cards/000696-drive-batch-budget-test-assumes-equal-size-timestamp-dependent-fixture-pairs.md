---
id: 000696
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Drive batch budget test assumes equal-size timestamp-dependent fixture pairs

## Problem

TestDriveMirrorBatchBudgetDefersCompletePairs assumes the second fixture pair fits a byte budget measured only from the first pair. During #689 label compatibility qualification it failed at the second pass with Verified=0, AlreadyRecorded=1, Deferred=1, Pending=1. This is a fixture defect, independent of the new codec: a method-free legacy label serialization equals the new label bytes exactly for both pairs.
