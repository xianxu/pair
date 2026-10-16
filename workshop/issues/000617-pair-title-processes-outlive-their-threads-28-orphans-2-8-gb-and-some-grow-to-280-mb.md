---
id: 000617
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '5f6c39b81a11407e9b069d75831ee207d393928a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T12:54:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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

See [implementation plan](../plans/000617-title-process-lifetime-plan.md). TL approval required before code.

- [ ] Attribute Go heap/allocations and reconcile #583 cleanup contract.
- [ ] Red tests, then wrapper-owned title lease and cancellation.
- [ ] Measured memory fix and bounded soak.
- [ ] Identity-fenced legacy cleanup, live acceptance, normal SDLC close/landing.

## Log

### 2026-10-16

### 2026-10-16 — planning investigation

Claimed in pair:6 and started planning from main; initial start-plan refreshed the ariadne substrate and refused until rerun, then created the issue branch. No implementation changes. Detached launcher ownership and 60s/five-miss session probes do not bind title lifetime to the wrapper. PPID 1 alone is not dead-thread proof: live ops wrapper 43773 still corresponds to title 25522. Native heap/vmmap succeeded: 24.5 MiB current physical footprint, 271.8 MiB peak, about 283 MiB ps RSS, only 65 KiB native malloc. Go heap attribution remains open; native heap is not a Go pprof capture. Repeated uncached inventory/usage scans are an allocation candidate, not an established retained leak. Private evidence paths/hashes and proposed red tests are in the durable plan. Sent #583 coordination to pair:4 (receipt 6a93aca7-49d2-4fff-8a4f-07677803c112), still queued at the last read. ARCH-DRY favors existing activity caches; ARCH-ORDER/SECURE require EOF lifetime and exact generation-fenced cleanup. Await TL plan/profile-method decision before code.

## Revisions

### 2026-10-16 — plan-first investigation

Added the durable implementation proposal and explicit evidence gates. Original Spec and Done when remain the contract; memory cause, Go-profile method, #583 cleanup seam and live rollout are recorded as unresolved rather than assumed.
