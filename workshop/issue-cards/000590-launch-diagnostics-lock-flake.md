---
id: 000590
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Diagnose concurrent launch diagnostics lock timeout under load

## Problem

`TestLaunchFailureStoreReopenConcurrent` intermittently fails under full-suite load when one of 17 concurrent diagnostics Save calls times out acquiring the store lock. The later retention assertion also fails because that failure record was not saved. This is independent of #571's Codex adapter; the diagnostics store and its test are unchanged by that branch.
