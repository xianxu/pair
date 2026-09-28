---
id: '000156'
status: done
started: 2026-08-29T06:33:52-07:00
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 8.96
actual_hours: 3.95
---

# incremental native session inventory

## Problem

#155 made the native session inventory authoritative for launch, watcher, UI,
and recovery decisions, but its production paths rebuild that inventory by
walking and parsing the complete native transcript corpus on every query. On
the operator's Claude corpus this means 1,573 files / 358 MiB: one inventory
query takes about 13.2 seconds (9.3 seconds user, 4.4 seconds system), including
5.4 seconds spent spawning one `stat` subprocess per file.

That full scan is synchronously repeated in latency-sensitive paths:

- `Alt+X` performs three inventory queries before it paints the confirmation;
- fresh Claude launch checks established ownership, checks a minted UUID for a
  collision, captures a full launch baseline, and then starts a watcher that
  repeats the same whole-corpus scan;
- launch baseline and watcher event projection reread every root transcript to
  normalize historical events even though only post-launch bytes can establish
  the new binding.

The typed-ledger compatibility projection also discards display metadata. A
typed launch wins authority over its compatibility row but supplies no
`repo_name`, so the session picker renders an otherwise known historical row as
`?/1 claude` instead of `pair/1 claude`.
