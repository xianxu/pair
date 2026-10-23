---
id: 000832
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Native VM readiness cancellation reports unconfirmed cleanup and blocks integration suites

## Problem

The #789 integration suite is blocked by vmcmd.TestNativeReadinessRunCancellationRetainsCleanupAuthority. It failed in the full suite and reproduced on the same unchanged head with an isolated targeted run. This is not established as a flake. #789 changes no vmcmd source relative to its integrated main.
