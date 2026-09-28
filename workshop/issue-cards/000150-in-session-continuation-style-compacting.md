---
id: 000150
status: open
created: 2026-08-22
updated: 2026-09-17
estimate_hours:
github_issue:
---

# in session continuation style compacting

## Problem

Token spend is dominated by context re-reads, and they scale with **session
depth**. Measured 2026-09-17 across ~52 days of local transcripts: 19.2B tokens,
**98% cache reads**; output — the actual work product — was 0.25%. Median
cache-read per assistant turn 324K, p99 960K, 98% of it main-thread. Splitting
one deep session into several fresh-context chunks doing identical work costs a
fraction.

Today the operator has two bad options at that moment:

- **Keep going** — every subsequent turn re-reads a context that only grows.
- **Shift+Alt+N** — fresh context, but the thread is simply dropped. Everything
  not already in a durable artifact is lost, so it's only safe at a boundary the
  operator has manually tidied.

Harness auto-compaction is the usual third answer and it's the wrong shape for
this repo: the carried summary is ephemeral, model-generated, and unreviewable —
you cannot read it, correct it, or diff it — and it fires on context *pressure*,
which is the most expensive possible moment (the turns that triggered it were
already paying the full window).

The gap is a **cheap, reviewable carry across a deliberate restart**.
