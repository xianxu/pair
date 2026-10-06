---
id: 000398
status: open
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '9e776f5c0afda209f8638287da4e849fa7f7aa68' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: startup reattach pass reruns inventory and slot-git sweeps after every thread

## Problem

The startup reattach pass (#206) re-runs a full inventory refresh and the
slot quick-status `git` sweep (#317, every slot checkout) after **each**
reattached thread. A `COUCH_TRACE` of a 19-thread startup on 2026-10-06 (AC
power) recorded:

| Event | Count | Each |
|---|---|---|
| `reattach-start`/`-done` | 19, strictly serial | 0.89 s → 1.32 s, creeping up |
| `inventory` (rows=26) | 20 | — |
| `slot-git` (ok=26) | 24 | ~0.6 s |

Pattern: `reattach-done` → `inventory` → `slot-git` → next `reattach-done`,
repeated for the whole pass (first-frame 1.65 s, last reattach 25.0 s). The
sweeps overlap the next reattach rather than blocking it, but they compete
for CPU. That likely explains the creep in per-thread cost, and it matters
most when the machine is throttled. An earlier startup that day on a 2%
battery took ~1m45s, with 5–13 s per reattach. About 40 of these sweeps are
redundant: nothing about slot git state changes because a thread reattached.

## Spec

- While the startup reattach pass is in flight, coalesce the inventory
  refresh and slot-git sweeps that a reattach completion triggers. Run **one**
  inventory + slot-git pass when the reattach pass drains. Keep the existing
  periodic cadence; only the per-reattach triggers are coalesced.
- Rows still update as each thread reattaches. Per-thread row state must
  not wait for the end-of-pass sweep, only the global inventory/git refresh.
- Parallelizing the reattach pass itself is out of scope (tracked separately).

## Done when

- [ ] A test drives an N-thread reattach pass through the fake and asserts
      O(1) inventory and slot-git passes during the pass (not O(N)), with one
      after it drains.
- [ ] Re-running `COUCH_TRACE=… couch` on the same fleet shows ≤2 `inventory`
      and ≤2 `slot-git` events between `pass-seeded` and the last
      `reattach-done`, and per-thread reattach time no longer creeps up.
- [ ] `atlas/couch.md` notes the coalescing next to the reattach pass.

## Plan

- [ ]

## Log

### 2026-10-06

- Found while investigating the 2026-10-06 couch crash (#397). The trace was
  `~/.cache/pair/couch-trace-20261006-141202.tsv` (addresses/timings only).
  Repro: `COUCH_TRACE=<path> couch`, then pair each `reattach-start` with its
  `reattach-done` and count the `inventory`/`slot-git` lines in between.
