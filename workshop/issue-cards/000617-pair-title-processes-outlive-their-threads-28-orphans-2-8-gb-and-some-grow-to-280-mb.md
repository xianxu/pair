---
id: 000617
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB

## Problem

`pair title` processes (one per agent thread) outlive their threads: on 2026-10-16 there were 28 of them, all reparented
to launchd (ppid 1), **2.8 GB resident** in total. Some belong to threads long gone (`1-parli-12`, `1-ducks-11` from
Oct 6; `1-xianxu-dev-22` from Oct 9; `couch-9d6fdf5e…` from Oct 13), and several of the live ones have grown to
~280 MB (`1-ops-33`, `1-pair-8`, `couch-9d6fdf5e…`), so there may also be unbounded growth inside one. Couch's resource
monitor now flags them (orphan-rss alerts for ops:0 and pair:2).
