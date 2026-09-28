---
id: '000059'
status: done
created: 2026-06-14
updated: 2026-06-14
estimate_hours: 3
actual_hours: 1.02
---

# timestamp TTY scrollback for real change-time change-log dates

## Problem

#58 **removed** the change log's `## YYYY-MM-DD` headers because they were the
**distill-time** date (the `Alt+l`-press date, injected via `--today`), not the
date the change happened — so a session worked across several days collapsed
under one "today" header on any bulk/first distill. The rendered TTY the
distiller reads carries no per-change timestamps (`events.jsonl` held only
`resize` events), so real change-dates weren't recoverable. This issue captures
a timestamp at the source so the change log can date entries by **real
change-time**.
