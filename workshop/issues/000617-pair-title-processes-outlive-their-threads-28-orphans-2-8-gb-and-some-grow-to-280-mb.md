---
id: 000617
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 3.33
card_mirror: 'b625859c279abf3fe676c7bda8b9f1864aa0c763' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T12:54:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A primitives plus Method B on memory attribution. Calibration marked stale by estimate-source; numbers provisional. Existing os.Pipe/exec.CommandContext/procutil remove any novel library implementation. Owner lease design uses the resolved spec discount (1.0h x0.2); legacy cleanup retains mid-density design (0.8h x0.5) until #583 answers. Memory attribution has three unresolved decisions at 0.15h each (retention versus spike, which scan dominates, and the resulting work bound), with code-heavy implementation 0.45 x1.5 x0.4=0.27h. Other implementation values below use v3.1's 40% scaling once. Familiar stack, 15% design buffer for this durable plan.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: real-api-discovery design=0.00 impl=0.24
item: method-b-decisions design=0.45 impl=0.27
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.05 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.33
```

Design subtotal 1.30h plus15%=1.495h; implementation subtotal1.83h; total3.325h, rounded3.33h. Respectively: lease, fenced cleanup, live-profile discovery, memory attribution/fix, cancellation integration, launcher ownership sweep, docs, single close review.

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

### 2026-10-16 — plan handoff unavailable

Plan committed at be96916a6. Couch socket disappeared before handoff. The send to hub returned uncertain outcome with receipt 15b2288f-f3ba-415e-bc7b-192ccf543f9c; status lookup also could not connect. No automatic resend, no alternative channel, no implementation. TL has not been confirmed to receive this plan; query that receipt before any later send. Earlier pair:4 coordination was queued, not confirmed read. Await transport recovery and TL review.

### 2026-10-16 — TL approved plan

After restart the plan report was resent (09bd705d-e7a6-48a7-b2a2-e9f6b1804a90). Verified TL approval receipt 72aed150-a96e-46b8-8e68-8bb2dcbaa16f: proceed to change-code, implement wrapper pipe lifetime with red tests first, choose controlled production-path replica profiling using real ledger/catalog, no diagnostic rollover of live titles. Cleanup coordination with #583 remains pending. Full flow requested for this cross-process ownership and memory change.

### 2026-10-16 — plan gate round 1

Plan-quality REWORK: PQ-1 requires cancellation to cover in-process inventory/usage, not only sleep and commands; PQ-2 requires named function-level testing strategies. Revised plan compresses test inventories and specifies cooperative context throughout query/range IO plus a proposed bounded CLI-process exit fallback for indivisible decoding. Requested TL approval of that fallback (submitted receipt d4707560-510f-42d6-a747-eb62e6e2dcd0). Profiling must copy the real ledger/catalog first: QuerySession can advance catalog state, so a live PAIR_DATA_DIR would not be read-only. Real ops ledger is 10,323,719 bytes, catalog 37,403 bytes. No implementation changes.
