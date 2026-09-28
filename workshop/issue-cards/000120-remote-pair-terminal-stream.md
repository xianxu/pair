---
id: 000120
status: open
created: 2026-07-26
updated: 2026-07-26
estimate_hours:
github_issue:
---

# Remote Pair terminal stream

## Problem

After remote lifecycle control works, Pair still cannot be operated remotely as a
live coding surface. The local terminal UI is visible only inside zellij, so a
tablet or browser cannot watch agent output stream, send prompts, or recover
orientation without attaching to the local terminal.

Pair already captures PTY output through `pair-wrap` and can replay scrollback
from raw bytes plus resize events. The next step is to expose that stream to a
browser without changing the local zellij workflow.
