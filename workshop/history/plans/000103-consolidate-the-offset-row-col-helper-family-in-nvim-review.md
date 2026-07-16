---
id: 000103
status: done
deps: []
github_issue:
created: 2026-07-05
updated: 2026-07-07
estimate_hours: 0.75
started: 2026-07-07T22:18:55-07:00
actual_hours: 0.27
---

# consolidate the offset→(row,col) helper family in nvim/review

## Problem

The `nvim/review/` modules carry several near-duplicate helpers for turning a
byte offset into a line / (row, col) and for computing per-line start offsets —
flagged as an **ARCH-DRY** Minor across the #89 M1 and close-time reviews ("fold
this before a fourth copy appears" — it wasn't, and a fourth appeared). Current
copies:

- `nvim/review/markers.lua` — **two byte-identical** binary-search offset→(row,col)
  twins: `offset_to_pos` (inside `parse_markers`, ~L155) and `pos_of` (inside
  `spans_multiline`, ~L233). Both build a `line_starts` array then binary-search it.
- `nvim/review/reconstruct.lua` — `line_of` (offset→0-based line) and `pos_at`
  (offset→row,col), both **linear** scans; plus `nth_offset` (the occurrence
  locator, shared and correct — leave it).
- `nvim/review/reconcile.lua` — inline `v1_starts` line-start computation
  (`plan_conflicts`), `split_lines`, and `occurrence_at` (non-overlapping match
  counter — a sibling of `reconstruct.nth_offset`'s counting).

They are not lazy copy-paste (markers uses binary search over precomputed
line-starts; reconstruct scans linearly), so this is a **deliberate-divergence**
cleanup, not a bug — behavior is correct today. But four+ copies of "offset →
position" is exactly the drift ARCH-DRY guards against.

This is a **pure-module refactor** (no behavior change); it's low-risk but touches
the hot parse/render path, so it wasn't bundled into #89's SHIPّd close.

## Spec

Establish **one** home — `reconstruct.lua` is the pure offset/position module —
for the offset→position family, and have `markers`/`reconcile` derive from it:

- Add (or promote) a shared `line_starts(content_or_lines)` builder + a
  binary-search `pos_of(line_starts, offset) → (row0, col0)` in `reconstruct`
  (keep the efficient binary-search form; retire markers' two twins in favor of it).
- Point `markers.parse_markers` and `markers.spans_multiline` at the shared helper
  (they already precompute `line_starts` — pass it in).
- Point `reconcile.plan_conflicts` at the shared `line_starts` for `v1_starts`;
  fold `occurrence_at` and `reconstruct.new_occurrence_of`/`nth_offset`'s
  non-overlapping counting into one clearly-named counter if they can share (they
  have different signatures — check before merging; a note is fine if they can't).
- Keep `reconstruct.line_of`/`pos_at` as thin wrappers (or replace call sites),
  whichever keeps the diff smallest without leaving a linear+binary-search fork.

Guard the refactor with the existing pure tests (`markers_test`, `reconstruct_test`,
`reconcile_test`) — they already pin the byte-accurate positions; a pure move
should keep them green with no edits, which is the proof it's behavior-preserving.

## Done when

- One shared offset→(row,col) / line-starts helper in `reconstruct.lua`; the two
  `markers.lua` twins and `reconcile.lua`'s inline copy derive from it.
- `make test-lua` + `make test-review` green with **no test edits** (behavior
  unchanged) — or with only additive tests for the new shared helper.
- No remaining second implementation of the same offset→position math (grep clean).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.15 impl=0.60
design-buffer: 0.0
total: 0.75
```

## Plan

- [x] Add the shared `line_starts` + binary-search `pos_of` to `reconstruct.lua`
  (+ a colocated unit test for the new helper)
- [x] Repoint `markers.lua` (`parse_markers`, `spans_multiline`) — retire the twins
- [x] Repoint `reconcile.lua` (`plan_conflicts` `v1_starts`); reconcile the counters
- [x] `make test-lua` + `make test-review` green, unchanged assertions

## Log

### 2026-07-05

### 2026-07-07
- 2026-07-07: closed — RED: nvim -l nvim/review/reconstruct_test.lua failed on missing line_starts before implementation. GREEN: nvim -l nvim/review/reconstruct_test.lua, nvim -l nvim/review/markers_test.lua, and nvim -l nvim/review/reconcile_test.lua passed. Final verification: make test-lua passed; make test-review passed; git diff --check HEAD~1..HEAD passed. No atlas update: pure internal consolidation of existing nvim/review offset helper behavior, no new surface.; review verdict: SHIP

- Claimed and entered planning. Scope stays within the existing issue plan:
  consolidate duplicated offset helpers into pure `reconstruct.lua`
  (`ARCH-DRY`, `ARCH-PURE`) and repoint every listed consumer rather than
  landing only the easy subset (`ARCH-PURPOSE`).
- Added `reconstruct.line_starts` and `reconstruct.pos_of` as the shared
  offset→position source; `reconstruct.line_of`/`pos_at` now wrap the shared
  helper, while `markers.parse_markers`, `markers.spans_multiline`, and
  `reconcile.plan_conflicts` derive from it.
- Counter decision from the plan-quality INFO note: `occurrence_at` shares cleanly
  as `reconstruct.occurrence_at` with the same non-overlapping count semantics,
  so `reconcile` now derives from that helper too.
- RED/GREEN evidence: `nvim -l nvim/review/reconstruct_test.lua` first failed on
  missing `line_starts`; after implementation, `reconstruct_test`, `markers_test`,
  and `reconcile_test` passed.
- Verification: `make test-lua` and `make test-review` passed. Grep sweep shows
  the remaining line-start/binary-search implementation is in `reconstruct.lua`;
  consumers call it.
