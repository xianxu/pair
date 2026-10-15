---
id: 000586
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Diagnose intermittent review restoration observation mismatch

## Problem

`tests/review-observation-test.sh` intermittently refuses its repeated branch restoration during a full suite, with insufficient evidence to distinguish a stale observation, a timed-out observation, and an actual branch mismatch. This blocks unrelated #571 verification; the adapter changes none of the review observation/controller/test code.
