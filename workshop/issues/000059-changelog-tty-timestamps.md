---
id: 000059
status: done
deps: []
github_issue:
created: 2026-06-14
updated: 2026-06-14
estimate_hours: 3
actual_hours: 1.02
---

# timestamp TTY scrollback for real change-time change-log dates

## Problem

#58 **removed** the change log's `## YYYY-MM-DD` headers because they were the
**distill-time** date (the `Alt+l`-press date, injected via `--today`), not the
date the change happened — so a session worked across several days collapsed
under one "today" header on any bulk/first distill. The rendered TTY the
distiller reads carries no per-change timestamps (`events.jsonl` held only
`resize` events), so real change-dates weren't recoverable. This issue captures
a timestamp at the source so the change log can date entries by **real
change-time**.

## Spec

Design agreed in brainstorm (operator: day-level granularity, minute cadence, no
backfill). The timestamp lives in the **events sidecar**, NOT injected into the
raw TTY — the raw is byte-faithful agent output (`resume` replays it; scrollback
+ change log derive from it), and there's already a generic offset-anchored
sidecar for exactly this kind of out-of-band metadata.

1. **Capture (`cmd/pair-wrap`).** Emit a `time` event via the existing generic
   `logScrollbackEvent(typ, fields)` (`main.go:1413`, already writes
   `{"type","offset",...}` keyed by `scrollbackBytes`) — one call:
   `logScrollbackEvent("time", {"ts": <RFC3339>})`. **Debounced to ≤1/minute**:
   emit only when wall-clock has advanced ≥60s since the last `time` event AND
   there's new output (keeps `events.jsonl` small). Raw stream untouched.
   pair-wrap sees bytes not turns, so periodic-by-minute beats per-turn.

2. **Scrollback — no change (free skip).** The render's `parseEvents`
   (`cmd/pair-scrollback-render/main.go:81`) already filters to `type=="resize"`,
   so `time` events are ignored by the scrollback path with zero changes. This is
   the operator's "skipped when we want to" — already free.

3. **Change log — additive consume.** Source the timestamp from the sidecar (NOT
   the raw): the render, behind a flag (e.g. `--with-timestamps`), emits a
   recognizable **marker line** into the *cleaned* output at each `time` event's
   offset (a sentinel the distiller recognizes, e.g. `⟦ts 2026-06-14⟧`). The
   distiller: maps each distilled entry to the **nearest preceding marker** →
   real change-date; **strips the markers** from the change-log text (same shape
   as `trimLiveTail`/`isFooterChrome` footer-chrome stripping); re-introduces
   `## YYYY-MM-DD` headers (day-level) sourced from those dates. (Bringing back
   the `assemble` date-header path #58 removed — but now fed real change-dates.)

4. **No backfill / graceful degradation.** Existing sessions have no `time`
   events; **if a region has no timestamp, emit no date header** for it (undated
   entries simply carry no `## date`). The change log must read correctly whether
   or not dates are present — a mix of dated (new) and undated (pre-#59) entries
   is fine.

Granularity: **day-level** display (`## YYYY-MM-DD`). Cadence: **minute**.

## Done when

- `pair-wrap` writes `{"type":"time","offset":N,"ts":...}` records to
  `events.jsonl`, at most one per minute of activity — unit/integration tested
  (a >1min activity window yields ≥2 time events; a <1min burst yields 1).
- The scrollback viewer (`Alt+/`) is visually unchanged — no timestamp markers
  leak into the rendered scroll (render still ignores non-`resize` events).
- The change log dates entries by **change-time**: an integration test feeds a
  cleaned stream spanning two days (via injected `time` events) and asserts the
  log carries the correct `## YYYY-MM-DD` headers for each day's entries — not
  the distill date.
- No-backfill: a cleaned stream with no `time` events produces a header-free
  change log (today's behavior preserved); markers are stripped, never shown.
- Full go + lua + smoke suites green; verified live (`Alt+l` over a real
  multi-day session shows real dates).

## Plan

- [x] `cmd/pair-wrap`: minute-debounced `time` event via `logScrollbackEvent`;
  tests (debounce window, offset anchoring).
- [x] `cmd/pair-scrollback-render`: `--with-timestamps` emits marker lines from
  `time` events (scrollback default path unchanged); tests.
- [x] `cmd/pair-changelog`: parse markers → per-entry change-date, strip markers,
  re-introduce day-level `## YYYY-MM-DD` headers from real dates, no-date →
  no-header; unit + integration tests (two-day stream, no-event stream).
- [x] `bin/pair-changelog-open`: pass `--with-timestamps` to the render step.
- [x] Atlas: update the Change-log section (date headers are back, sourced from
  `time` events) + the scrollback/events-sidecar description.

## Log

### 2026-06-14
- 2026-06-14: closed — Alt+l dates change-log entries by real change-time — live test: ## 2026-06-14 on new work, old content undated, no ts markers leaked. pair-wrap emits minute-debounced time events; render --with-timestamps interleaves day markers; distiller segments per-day (no markers → header-free, byte-identical to #58). Touched go suites + e2e (render→cleaned→distill) + orchestrator smoke green. Full `make test` aggregate hang tracked in #60 (test-infra, not feature code).; review verdict: FIX-THEN-SHIP
- Boundary review (#59) **FIX-THEN-SHIP**, no Critical. Fixed before merge:
  *Important* — added `TestIncrementalDatedAppend` (the cross-press dated path
  through `main.go`'s segment loop: dated prior log + a new-day marker → one new
  `## DATE`, no duplicate of the prior day, frozen prefix intact; was only covered
  at the pure level by `TestAssembleDated`). *Minors* — repointed the
  `tsMarkerLine`↔`tsMarkerRe` sync comments to the real pin (`e2e_test.go`
  `TestEndToEndMarkerSurvival`, not the no-marker `changelog-open-test.sh`); made
  `feedSegments` walk **all** events (not `events[1:]`) so a time event in any
  position is captured + empty events can't slice-panic (re-applying the offset-0
  resize is a harmless no-op). Left as accepted: the side-quest batch-size bump to
  2000 (#59 Log), and the negligible stale-header note on pre-#58 never-redistilled
  logs. Touched suites re-green.

- Filed from the post-#58 brainstorm. Root cause of the #58 date removal: no
  change-time source. Design settled (events-sidecar over raw-injection;
  free scrollback skip via the existing `resize`-only filter; day/minute/no-backfill
  per operator). Seams identified: `logScrollbackEvent` (pair-wrap:1413, generic),
  `parseEvents` resize-filter (scrollback-render:81), `assemble` date-header path
  (#58 removed — to be restored, fed real dates). See #58 history for context.
- Implemented (TDD, 5 commits): **pair-wrap** emits minute-debounced `time`
  events via the generic `logScrollbackEvent` (pure `dueForTimeEvent` + `p.now`
  clock seam); **render** `parseEvents` keeps `resize`+`time`, `feedSegments` is
  one offset-ordered walk (act on resize, snapshot `Scrollback().Len()` on time),
  pure `interleaveDateMarkers` inserts `⟦pair:ts DATE⟧` at day boundaries behind
  `--with-timestamps`; **distiller** `parseDatedLines` strips markers → per-line
  dates, `splitByDate` → per-day segments, `assemble` regains its date param
  (restored from #58, fed real dates), `stripDateHeaders` deleted; **orchestrator**
  passes `--with-timestamps`. No markers → header-free, byte-identical to #58
  (purely additive). Tests: pure units (`dueForTimeEvent`, `dateOf`,
  `interleaveDateMarkers`, `parseDatedLines`, `splitByDate`, `assemble`),
  `TestMaybeLogTimeDebounced`, `TestRenderWithTimestamps`, `TestTwoDayDating`,
  `TestNoMarkerHeaderFree`, and an e2e `TestEndToEndMarkerSurvival` (real
  render→cleaned→distill, plan-quality finding 2). go + render + pair-wrap suites
  green; orchestrator smoke green.
- Durable plan: `workshop/plans/000059-changelog-tty-timestamps-plan.md`. Plan
  quality judge (`sdlc change-code`) verdict **INFO** (start-ready). ARCH-DRY
  (reuse `logScrollbackEvent`; one `scrollbackEvent` struct+parser; restore the
  single `assemble` date authority) + ARCH-PURE (pure `dueForTimeEvent`/`dateOf`/
  `interleaveDateMarkers`/`parseDatedLines`/`splitByDate`/`assemble` vs thin
  clock/emulator/flag seams) both satisfied. Four non-blocking refinements folded
  into the plan: (1) **visible-buffer lag** — the snapshot reads `Scrollback().Len()`
  during feed, so up to ~one screenful of the prior day's not-yet-evicted tail can
  fall under the new day's marker; negligible at day granularity, noted as a risk.
  (2) **marker-survival e2e** — added an automated render(`--with-timestamps`)→
  cleaned→distill assertion (Task 9) so the seam isn't only live-verified. (3)
  `feedSegments` becomes the **single offset-ordered walk** over all events (act on
  `resize`, snapshot on `time`) — not a parallel feeder. (4) estimate 3h is
  optimistic (~3–5h for 10 TDD tasks across 4 pkgs); left as-is, actual measured at
  close. Single review boundary (capture+render are invisible without the distiller
  → one `sdlc close`, plain checkboxes).
