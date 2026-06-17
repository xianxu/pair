---
id: 000057
status: working
deps: []
github_issue:
created: 2026-06-12
updated: 2026-06-17
estimate_hours: 4.5
---

# Alt+q annotation in change-log viewer + shared nvim/annotate.lua

## Problem

The `Alt+l` change-log viewer (#53) is read-only with no way to react to an
entry. The scrollback viewer (`Alt+/`) already has the right affordance: `Alt+q`
drops a 🤖-marker (a question/comment) on a line or selection, and on quit the
markers ship to the draft pane (→ the agent) via a pending sidecar. We want the
**same Alt+q flow in the change-log viewer** — ask a question about a logged
milestone/decision and have the agent see it.

Both viewers are read-only nvim windows over pair state, so the marker machinery
should be **shared, not duplicated** — but they differ (scrollback renders SGR
via extmarks; changelog renders markdown + has an async background refresh), so
unification is partial.

(Deferred from #53; operator asked to split it out since it refactors the
working scrollback viewer.)

## Spec

### Extract `nvim/annotate.lua` — the shared 🤖-marker system

Lift the source-agnostic marker machinery out of `nvim/scrollback.lua` into a
new `nvim/annotate.lua`, `require`d by both viewers, parameterized by
`{ bufnr, pending_path, footer = bool }`:

- **Marker core:** `find_markers_in_line`, the `🤖[Y]` / `🤖<X>[Y]` escape /
  unescape, `strip_markers`, `marker_key`, `collect_markers_by_line` (the
  load-time baseline so only *newly-added* markers extract), `format_extraction`,
  `highlight_markers`.
- **Alt+q add/edit flow:** `add_marker_normal` / `add_marker_visual` /
  `edit_marker` / `open_marker_prompt` / `rewrite_line` (the read-only
  unlock→insert→relock dance — markers are inserted as buffer *text*, then
  highlighted).
- **Extraction:** the `VimLeavePre` → pending-file emit + the quit-confirm-if-
  markers gate.

**Stays per-viewer (NOT shared):** rendering (scrollback = SGR/ANSI extmarks;
changelog = markdown `syntax match`), `Alt+b` prompt-jump (scrollback only), the
async refresh + spinner (changelog only). The scrollback **footer overall-comment**
affordance (`footer_row_by_buf` etc.) is scrollback-specific → gate it behind the
`footer` option (changelog: `footer = false`).

`scrollback.lua` refactors onto `annotate.lua` keeping `scrollback_test.lua`
green (the extraction's safety net); `changelog.lua` wires `Alt+q` (normal +
visual) through it.

### Async-reload conflict (the changelog-specific wrinkle)

Markers are buffer **text**; the changelog's background distill calls `M.reload`,
which **replaces all lines** — wiping a marker added during the spinner.
Resolution: when the distill job finishes, **skip the destructive reload if the
user has added markers since open** (annotations win; the fresh log is on disk
and appears on the next `Alt+l`). After the spinner clears there are no more
reloads that press, so annotations are safe. One guard in `start_refresh`'s
`on_exit` (compare current marker count vs the load-time baseline before
reloading).

### Pending file + source differentiation

Reuse the **same** draft-pickup mechanism (the draft pane's `FocusGained` reads
`scrollback-pending-<tag>.md`) so changelog questions reach the agent with zero
new plumbing. Tag each emitted block with a source marker (e.g. a leading
`> [change log]`) so the agent knows the question is about a logged milestone vs
raw scrollback — this is the "differentiate source" ask.

### Out of scope

- Changing the scrollback viewer's *behavior* (pure refactor — same markers, same
  pending file, same UX).
- The forward-notes carried from #53's reviews (below) — track separately or fold
  in opportunistically, but they're not this issue's core.

## Done when

- `Alt+q` in the change-log viewer drops a 🤖-marker on a line (normal) or
  selection (visual), exactly like the scrollback viewer; on quit the markers
  ship to the draft pane via the pending sidecar, tagged with the change-log
  source.
- The marker machinery lives once in `nvim/annotate.lua`; `scrollback.lua` is
  refactored onto it with `scrollback_test.lua` still green; `changelog.lua`
  wires `Alt+q` through it.
- A marker added during the async spinner is **not** wiped by the background
  reload (the reload is skipped when annotations are present).
- A headless `annotate_test.lua` covers the shared marker parse/extract; the
  scrollback + changelog wirings each have a smoke check.

## Plan

- [ ]

## Log

### 2026-06-12

- Split out of #53 (operator). Design sketched during #53 (see its Log
  2026-06-12 + this Spec). Reviewed the scrollback marker code: the shared chunk
  is ~400 lines (`find_markers_in_line`, escape/unescape, `strip_markers`,
  `marker_key`, `collect_markers_by_line`, `format_extraction`, `highlight_markers`,
  the `add_marker_*`/`edit_marker`/`open_marker_prompt`/`rewrite_line` flow, the
  `VimLeavePre` emit). Differences: rendering, `Alt+b`, async refresh, the footer
  affordance.

### Forward-notes carried from #53 reviews (separate, opportunistic)

These were SHIP-level advisory notes on the shipped change-log feature — capture
here so they survive #53's archival; none are this issue's core:

- **Mid-turn staleness / force-refresh.** The turn-count no-op gates the model
  call on a *new user-prompt boundary*, so a long single-prompt autonomous turn
  that completes several milestones won't refresh on repeated `Alt+l` until the
  next prompt. Content is never lost (self-heals at the next boundary), but the
  glance view can lag long autonomous turns. A "force refresh" affordance, or a
  `max(turn-count-increase, cleaned-byte-delta-above-threshold)` gate, would fix
  it without reintroducing per-press churn.
- **Anchor is the volatile tail** → many later presses miss `locate` →
  `FullRedistill` (whole transcript re-distilled). Safe + dedup-protected, but the
  incremental-slice optimization is frequently bypassed in practice; measure real
  call sizes before relying on the anchor for cost savings.
- **Viewer silent model-failure.** `changelog.lua`'s `on_stderr` matches only
  `distilling N` / `up to date`; a distill error just clears the spinner and
  reloads the unchanged log — no user signal. Consider surfacing a one-line error
  in the virtual line.
- **`PAIR_SLUG_*` env names in shared `cmd/internal/model`** (`PAIR_SLUG_NESTED`,
  `PAIR_SLUG_OPENAI_BASE_URL`) now consumed by pair-changelog too — slightly
  misleading in the shared package. If a third caller appears, rename to
  `PAIR_MODEL_*` with a back-compat read of the old names.
- **agy `^>` over-match** now feeds the no-op gate (not just lookback): a false
  glyph can cause a spurious extra call, or — if agy's real prompt isn't `^>` — a
  one-shot log that never updates. Known best-effort simplification; a faithful
  multi-line port (`lines[i-1]` starts with `──`) would fix it.
- **Untested distiller branches:** the model-error (`fail` → exit 1, log
  preserved) and empty-model-output paths have no assertion. Cheap to add a
  fake-claude-empty / fake-claude-exit-1 integration case.
