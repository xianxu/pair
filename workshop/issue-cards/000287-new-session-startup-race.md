---
id: '000287'
status: done
started: 2026-09-18T13:38:02-07:00
created: 2026-09-18
updated: 2026-09-18
actual_hours: 1.54
---

# New sessions die at birth: title poller probes zellij during server startup

## Problem

Creating a new Pair session fails most of the time: the zellij server panics
~20 ms after starting, before the session exists. The operator sees a blank
pane that echoes typed characters (the zellij client never reached raw mode);
Couch marks the thread `live` while nothing runs. Reattaching existing sessions
is unaffected, which is why this surfaced as "can't start a thread for astro"
— astro was the only cold create being attempted. **It is not astro-specific:**
a standalone `pair resume <new-tag> --layout3` in a fresh scratch repo reproduces
it.

Zellij log (`$TMPDIR/zellij-501/zellij-log/zellij.log`), every failed launch:

```
Starting Zellij server!
Panic occurred: At zellij-server/src/lib.rs:1462:26
  called `Option::unwrap()` on a `None` value
```

Failures on 09-15 21:07, 22:10, 22:54 ×3; 09-16 ×6; 09-17 17:17; 09-18 12:24,
12:33, 12:41, 12:42.
