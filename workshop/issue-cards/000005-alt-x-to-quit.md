---
id: '000005'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# Alt-X to quit

## Problem

Now that we have Alt+d to detach, add Alt+x to quit. "Quit" means kill all subprocesses *and* delete the session entry on the way out — vs. zellij's default Ctrl+q which kills the session but keeps it in the resurrect list.
