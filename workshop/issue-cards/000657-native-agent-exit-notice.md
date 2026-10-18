---
id: 000657
status: open
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
---

# Couch reports native agent exit without mistaking attach disconnect

## Problem

#574 cannot report a native agent exit reliably from the existing Couch observations. Console.onExit observes the Pair attachment client, and registry EOF may be a reconnect or wrapper replacement. Neither proves that the native agent stopped.
