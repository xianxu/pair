---
id: 000579
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '17a6d03a4d8ff4f54f1ce510e58f5cdba6e8056b' # card fields mirrored from issue-cards; edit via sdlc
---

# pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation

## Problem

2026-10-15 (operator's system monitor): each `pair wrap` process (`--scrollback-log …/scrollback-1-<tag>-codex.raw`)
**writes 3.4–4.1 MB/s to disk**, about 29 MB/s total for the fleet (~2.5 TB/day of write wear). Yet the files it
writes grow only **~17 KB/s** (`wrap-events-1-<tag>.jsonl`) and **~2.3 KB/s** (`scrollback-…raw`) per active slot,
measured over 10 s. That's a **~200× write amplification**, most likely a sync (fsync/F_FULLFSYNC) or a re-open per
small event write, so every few-hundred-byte append costs whole blocks plus filesystem journal writes. It's also likely
behind the FSEvents daemon's ~200% CPU seen on 10-14.

Separately, these files are **never rotated**: wrap-events logs of **3.9 GB** (pair-8), **2.6 GB** (pair-43), 1.9 GB,
1.7 GB, 1.0 GB; the pair data dir was 57 GB on 10-14.

## Spec

1. **Measure first:** confirm the cause (count write/fsync syscalls per second for one wrapper, e.g. `fs_usage` or
   dtrace on a scratch slot), and record bytes-written vs. bytes-appended.
2. **Fix the amplification:** buffer event and scrollback writes and flush on an interval or size threshold; fsync only
   at real durability boundaries (e.g. before acknowledging a delivery, or at turn end), not per event. State what a
   crash may lose (at most the last interval's events) and why that's acceptable for each file.
3. **Retention:** rotate wrap-events and scrollback logs by size (e.g. 64 MB segments) and keep a bounded number or age
   (consistent with crash-only: what recovery and orientation actually read must remain).
4. A counted invariant (#204 style): bytes written per appended byte stays under a small bound for a synthetic event
   stream (no timing thresholds).

## Done when

- A wrapper under normal agent output writes on the order of what it appends (target: < 3× amplification), measured on a
  live slot before and after.
- Logs rotate; total size stays bounded; recovery and `couch --peek --previous` still work across a rotation.
- Tests: the amplification invariant; rotation and retention; crash-loss bound.

## Plan

- [ ]

## Log

### 2026-10-15
