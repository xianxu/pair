---
id: '000014'
status: done
created: 2026-05-03
updated: 2026-05-03
actual_hours: N/A
---

# stop forwarding bare BELs by default

## Problem

`bin/pair-wrap` over-notifies cmux. Live data from `~/pair-wrap.log` (one session, ~2hr): 76 EMITs total, only 8 of them legitimate (OSC 777 `Claude is waiting for your input`). The other 68 are bare-BEL fallback firing on the trailing `\x07` of OSC 8 hyperlinks and OSC 0 title sets that the streaming regex couldn't reconstruct across read boundaries.

Modern TUI agents emit OSC 9 / OSC 777 explicitly when they want attention. The bare-BEL fallback was defensive coding for unfamiliar agents that "might use BEL" — a case we have no concrete example of in the wild. Today its only measurable effect is noise.

Fix: stop forwarding bare BELs by default. Keep the *detection and logging* live so `PAIR_WRAP_LOG` still shows every BEL the wrapper sees, and we can choose to whitelist specific contexts later. Provide an env flag to re-enable forwarding when discovering a new agent's protocol.
