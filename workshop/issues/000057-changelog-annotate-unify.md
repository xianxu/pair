---
id: 000057
status: done
deps: []
github_issue:
created: 2026-06-12
updated: 2026-06-17
estimate_hours: 4.5
actual_hours: 0.55
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

Durable design: `workshop/plans/000057-changelog-annotate-unify-plan.md` (authored
via superpowers-writing-plans; fresh-eyes reviewed — all 4 critical design claims
verified, blocking Makefile-registration gap fixed). Two review boundaries:

- [x] M1 — Extract `nvim/annotate.lua` (pure marker core + IO/UI seam behind
  `attach{bufnr,pending_path,footer,source_label}`), refactor `scrollback.lua`
  onto it with **no behavior change**; new `annotate_test.lua` (registered in
  `Makefile.local` test-lua) covers the pure core; `scrollback_test.lua` stays
  green as the regression net. (ARCH-DRY: one marker subsystem; ARCH-PURE: pure
  core unit-tested without mocks.)
- [x] M2 — Wire `changelog.lua` through annotate: `Alt+q` normal+visual, emit
  tagged `> [change log] …` (per-quote prefix so `init.lua`'s `\n> ` pickup-count
  stays correct), and the async-reload guard (`has_new_markers`/`on_reloaded`) so a
  marker added during the spinner survives the distiller's reload; changelog
  smoke test; atlas update for the new shared surface.

## Log

- 2026-06-17: **dogfood bug + fix (post-close, pre-merge).** Operator's live pass: in the change-log viewer, Alt+q popped the prompt but it was un-typeable (modifiable=off). Root cause: the changelog autocmd fires on `BufWinEnter` with **no pattern**, so it also caught annotate's floating-prompt scratch buffer and ran `M.setup` on it → locked it read-only. (Scrollback dodged this — its autocmd is BufReadPost-only, which scratch buffers never fire; M2 wiring exposed the latent gap.) Fix: extracted `M.on_buf_enter(buf)` that **skips unnamed buffers** (the prompt has no file name; the real log buffer does) — only the named change-log buffer gets locked. Regression test added (unnamed scratch → skipped + stays modifiable; named → set up read-only). make test-lua green. This was exactly the interactive-path gap the integration review flagged — live dogfooding caught it.
- 2026-06-17: closed — Issue done-when met: (1) Alt+q in the change-log viewer drops a 🤖-marker (normal+visual) and on quit ships to the draft via the pending sidecar tagged "> [change log] …"; (2) marker machinery lives once in nvim/annotate.lua, scrollback.lua refactored onto it (SGR+header byte-identical to pre-refactor) with scrollback_test green; (3) a marker added during the async spinner survives the distiller reload (safe_reload/has_new_markers guard, asserted in changelog_test); (4) annotate_test covers the pure marker core, scrollback+changelog each have a wiring smoke. make test-lua all green (6 suites). Both milestones reviewed: M1 SHIP, M2 FIX-THEN-SHIP (minors applied).; review verdict: FIX-THEN-SHIP
- 2026-06-17: closed M2 — make test-lua all green (6 suites incl. annotate + changelog smoke). M2 wires changelog viewer to annotate.attach{footer=false, source_label="change log"}; smoke asserts footer=false adds no affordance line, emit ships a "> [change log] …" source-tagged block (per-quote prefix keeps init.lua \n> pickup-count faithful), and the reload guard skips the distiller reload while a marker is present so it survives. Esc/q route through shared confirm_quit. Both viewers now share one marker subsystem (ARCH-DRY); pure core tested without mocks (ARCH-PURE). Atlas updated. Headless limit documented: floating Alt+q prompt UI covered via pure-core + attach->emit data path, not a driven UI.
- 2026-06-17: M2 boundary review = **FIX-THEN-SHIP (high)** — no Critical/Important; reload-guard correctness + per-quote-prefix count + idempotent re-attach all verified by the reviewer. Two Minor doc-hygiene fixes applied before close: (1) changelog.lua header no longer claims "no marker system"; (2) atlas Change-log "View" bullet now documents the Alt+q affordance + confirm-if-markers quit.
- 2026-06-17: closed M1 — make test-lua all green (slug/scrollback/annotate/changelog/adapt/doctor). Extraction behavior-preserving: scrollback.lua SGR+header (lines 1-267) and final-opts tail byte-identical to pre-refactor (diff IDENTICAL); scrollback_test prompt-pattern green; new attach->emit wiring smoke asserts scrollback emit keeps legacy un-prefixed "> quote" format, no source label. annotate_test covers the pure marker core directly, no mocks (ARCH-PURE). Atlas maps the new annotate.lua surface.; review verdict: SHIP
- 2026-06-17: M1 boundary review = **SHIP (high)** — independently byte-diffed every moved fn vs base, all 10 pure fns byte-identical, others differ only by documented changes; no Critical/Important. Minors addressed/noted: (2) VimLeavePre multi-buffer behavior now commented in annotate.lua; (3) the live interactive Alt+q parity check (plan Task 1.3 Step 4) couldn't run in this headless env — superseded by the byte-level diff + the attach→emit wiring smoke; the floating-prompt UI remains the documented headless limit (verify live before final ship).
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
