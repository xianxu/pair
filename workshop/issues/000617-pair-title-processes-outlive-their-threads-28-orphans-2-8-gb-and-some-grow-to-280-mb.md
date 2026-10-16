---
id: 000617
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '1e87f45857cd25b22870a22dd9341e3ed40e267e' # card fields mirrored from issue-cards; edit via sdlc
---

# pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB

## Problem

`pair title` processes (one per agent thread) outlive their threads: on 2026-10-16 there were 28 of them, all reparented
to launchd (ppid 1), **2.8 GB resident** in total. Some belong to threads long gone (`1-parli-12`, `1-ducks-11` from
Oct 6; `1-xianxu-dev-22` from Oct 9; `couch-9d6fdf5e…` from Oct 13), and several of the live ones have grown to
~280 MB (`1-ops-33`, `1-pair-8`, `couch-9d6fdf5e…`), so there may also be unbounded growth inside one. Couch's resource
monitor now flags them (orphan-rss alerts for ops:0 and pair:2).

## Spec

- **Lifetime:** a title process lives exactly as long as its thread's wrapper. Prefer the lock/lease shape: it holds
  or watches something its owner holds for life (the wrapper's pid with its start stamp, or a pipe/flock the wrapper
  owns) and exits when that goes, rather than a reaper deciding later by observation.
- **Growth:** measure why one grows to ~280 MB (heap profile on a live one); fix the cause (bounded buffers/caches).
- **Existing leaks:** a one-time cleanup for title processes whose thread has no live wrapper, identity-fenced (pid +
  start stamp), consistent with pair#583's rules; never by name alone.

## Done when

- A test: kill a thread's wrapper; its title process exits within seconds.
- A title process's memory stays bounded over a long-running thread (a soak test or a measured bound).
- After the fix and the cleanup, live: one title process per live thread, none with ppid 1 for a dead thread.

## Plan

- [ ]

## Log

### 2026-10-16
