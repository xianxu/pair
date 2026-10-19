---
id: 000702
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# Compress large database snapshots faster with measured levels and threads

## Problem

#688's coherent 1,999,003,648-byte history snapshot acquired in 7.31s, but the following existing zstd -19 --single-thread plus gpg pipeline exhausted the acceptance pass's five-minute outer budget. Large daily database compression, rather than snapshot acquisition, now dominates latency. The snapshot reader had already exited and its WAL did not grow.
