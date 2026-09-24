---
id: 000311
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T13:07:35-07:00
flow: {kind: quick, provenance: inferred, spec: "73a4f0a4", done: "de0c83b9"}
---

# Click right-pane tab to switch tabs

## Problem

The right pane renders a tab strip, but clicking a tab does not switch the
right-pane terminal to that tab. Operators must use the keyboard tab controls
instead, even when the desired tab is visible.

This is the focused successor to the punted `#200`: implement the interaction
without reintroducing a second mouse-mode arbitration path.

## Spec

Clicking a visible tab chip in the right pane's tab strip selects that tab.
Clicks outside a visible chip remain pass-through or no-op according to the
existing mouse ownership rules. Hit testing must use the display-column spans
emitted by the same render pass, including clipped names and wide characters;
it must not reconstruct geometry from tab names or rune counts.

Reuse the existing right-pane tab-switch operation and mouse arbitration. Do
not change keyboard tab shortcuts, child mouse tracking, rename-field behavior,
or the handling of clicks outside the strip.

## Done when

- Clicking each visible inactive tab switches the right pane to that tab,
  including plain shell tabs: the parent uses the shared AnyMotion policy,
  while child-facing tracking remains governed by the existing presenter.
- Clicking the active tab is harmless and does not start rename mode or alter
  the tab order.
- Clicks on clipped-away tabs, separators, the rename field, and empty strip
  space do not select a different tab or leak an unintended event to the
  child.
- Hit testing remains correct for wide-glyph names and narrow/clipped strips.
- Tests exercise the production mouse-routing boundary and verify the existing
  child mouse-tracking behavior is preserved.
- The issue's implementation does not introduce a second mouse-mode tracker;
  the relevant reuse/ownership rule is recorded in the implementation log.

## Plan

Mirror couch's status row exactly (ARCH-DRY: one mouse-ownership model, two
consumers):

- [x] Parent mode: `newTerminalMux` builds its presenter with the same
      any-motion policy couch uses (`terminal.CouchAnyMotion`), so clicks
      reach `pair term` in a plain shell tab. The child still receives only
      what its own tracking mode requests (`Presenter.mouseInput`, unchanged).
- [x] Pure hit test: `RenderedStrip.ColumnToTab(col) (int, bool)` over the
      spans the clipping pass emitted — the twin of couch's
      `RenderedStatusRow.ColumnToActor` (ARCH-PURE). Unit tests: each chip,
      separators, empty tail, wide glyphs, clipped/dropped tabs.
- [x] Mux keeps the last drawn strip's spans (`chromeForGeometryLocked`
      records them; a notice replaces the row, so it clears them).
      `clickStrip(x, y) bool` selects via the existing `selectLocked` +
      `renamePane` path that `switchRelative` uses; active tab is a no-op.
- [x] Route: in `pumpStdinContext`, a left-button press on the strip row is
      offered to `clickStrip` before `writeEvents` — couch's
      `routeMouseEvent` shape. Anything it does not consume falls through to
      the presenter, which already owns strip-row presses as a parent gesture
      (no second tracker, no child leak). Rename sessions already drop mouse.
- [x] Production-boundary tests through `pumpStdinWithTimer` with a fake mux;
      presenter tests unchanged (child tracking preserved).
- [x] Live smoke in a pair session: click tabs in a shell tab and while nvim
      holds `?1002`; wheel scroll in a shell; note what drag-select does.

## Log

### 2026-09-23

Filed from the operator request to switch right-pane tabs by clicking their
visible tab labels. Existing context: `#199` owns the strip, historical `#200`
was punted while resolving mouse arbitration, and `#258` covers keyboard/global
tab chords rather than pointer interaction.

### 2026-09-23

Resumed in slot 2 (the original claim's session is unknown; no code existed).
Trace:

- Strip: `termcmd.RenderStrip` already emits display-column `TabSpan`s from the
  clipping pass; `chromeForGeometryLocked` discards them (`.Body` only).
- Routing: `pumpStdinContext` sends every non-wheel mouse event to
  `mux.writeEvents` → `Presenter.mouseInput`. A press on the strip row
  (`Y >= childRows`) is already a `ParentPressMouse` gesture — owned by the
  parent, never forwarded to the child, release swallowed. So the hook is
  "consume a left press on a span before the presenter" — the exact shape of
  couch's `routeMouseEvent` → `RenderedStatusRow.ColumnToActor` (ARCH-DRY).
- Switch op: `selectLocked(index)` (+ `renamePane`) is what `switchRelative` uses.
- Rename: the pump drops all non-key events while a rename session is open.

**Blocker:** `pair term` builds its presenter with `terminal.ChildRequested`, so
the parent enables mouse reporting ONLY while the active child holds tracking
(`desiredParentModes`). With a plain shell, zellij never forwards clicks to the
pane at all — the strip is unclickable exactly in the common case. Delivering
the Spec requires the parent to request tracking itself, which changes what
zellij does with drag-select in a shell tab. That is the #200 territory; needs
an operator decision before planning further.

Operator decision (2026-09-23): make the strip clickable in shell tabs and keep
it consistent with couch's tab bar. Couch's presenter always requests any-motion
(`CouchAnyMotion`, atlas "Mouse ownership (#255)") and intercepts status-row
presses via `ColumnToActor` spans; `pair term` adopts the same policy and shape.
Cost to verify live: zellij's own drag-select in a shell tab (same trade couch
already makes on the host terminal); wheel already falls back to zellij
`scroll-up/down` when the child holds no tracking.

Implemented: `AnyMotion` (renamed from `CouchAnyMotion` — it is no longer
couch-only) in `newTerminalMux`; `RenderedStrip.ColumnToTab`; mux records
`stripSpans` in `chromeForGeometryLocked` (nil under a notice); `clickStrip`
and `switchRelative` share one `switchTab(pick)` path whose pick runs under the
mux lock and bounds-checks, so a stale span can never `selectLocked` a missing
tab into `stopLocked` (ARCH-DRY, ARCH-PURE). Reuse/ownership rule: the strip
adds no mouse-mode tracker — the presenter's gesture state stays the only one;
the pump only intercepts a left press, exactly as couch's `routeMouseEvent`.
`ChildRequested` now has no production caller (tests only); left in place.
Tests: `TestColumnToTab*` (wide glyph, clipped), `TestPresentationClickingStrip
ChipSelectsThatTab` (real mux, fake children), `TestPresentationParentRequests
MouseForAPlainChild`, `TestPumpStdinOffersLeftPressToStrip`. termcmd, terminal,
couchtty, terminalqualify green (unsandboxed for pty tests).

Verification (2026-09-23): `go test ./...` in this checkout green (74 pkgs,
retention env scrubbed, TMPDIR=scratchpad). Full `make test` from a `git
archive` of a98a86f1 passes every shell/lua target; its only Go failures are
four #151/#155 contract tests that `git show` pinned objects (exit 128 — no
.git in an archive). In THIS worktree the nvim headless targets
(`test-lua`/`test-queue`/`test-submission-transaction`) fail rc=1 with empty
output, rotating per run; `submission-transaction` fails 5/5 here yet passes
3/3 from archives of both a98a86f1~1 and a98a86f1 and passes under `bash -x`
— a checkout-local, timing-sensitive environment issue, not this Go-only diff.
At that checkpoint, operator live smoke remained pending (confirmed below).

## Revisions

### 2026-09-23T22:30 — parent mouse policy
Reason: the Spec's "reuse existing mouse arbitration" cannot deliver clicks in a
shell tab, because `ChildRequested` never enables parent reporting there.
Delta: parent policy changes from `ChildRequested` to couch's any-motion
policy; child-facing tracking and the presenter's arbitration are unchanged.


### 2026-09-23 — operator acceptance

Operator confirmed the #311 smoke test passed after rebasing onto refreshed main, and requested close and publication. The implementation patch was unchanged by the rebase; duplicate published documentation commits were dropped.

### 2026-09-23 — acceptance wording aligned at close

Reason: close detected the previously approved parent-policy revision without a corresponding Done when update. Delta: explicitly require shell-tab clicks through shared AnyMotion while preserving child-facing tracking. The implementation and operator smoke already cover this behavior; no scope or code change.
