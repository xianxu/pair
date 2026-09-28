---
id: '000062'
status: done
created: 2026-06-16
updated: 2026-06-16
estimate_hours: 0.5
actual_hours: 0.04
---

# move alt+delete back to always trigger deletion of future queue item

## Problem

at some point, we moved alt+delete in future queue item to condition on mode: normal mode it would behave as deletion to beginning of line, normal mode to delete that future queued item. this, in practice, is more confusing that helping. let's move alt+delete to always delete "future queued item" if we are in the +N portion of draft buffer.
