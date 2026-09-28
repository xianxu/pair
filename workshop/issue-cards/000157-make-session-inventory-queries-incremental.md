---
id: 000157
status: open
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
github_issue:
---

# Make session inventory queries incremental

## Problem

`pair session-inventory` reconstructs the complete native-session forest by
calling the full scanners on every invocation. It therefore rereads transcript
bodies, including hundreds of megabytes of unchanged history, even though Pair
already maintains a durable session-inventory catalog with artifact
fingerprints, parser offsets, scanner state, and extracted facts.

The watcher and targeted binding queries use that catalog incrementally, but
the public tree query bypasses it. This violates the expected operating
envelope for an inventory that may back interactive session selection
(`ARCH-CONSTRAINTS`).
