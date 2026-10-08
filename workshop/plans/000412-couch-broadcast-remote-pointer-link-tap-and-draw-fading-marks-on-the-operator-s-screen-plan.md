# Remote Pointer Link Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** While broadcasting, a click on `👆` gives the operator a pointer link
whose holders can tap and draw fading marks on the operator's screen, through a
channel that carries only grid coordinates and never reaches a child program.

**Architecture:** The #395 session gains a second capability token and a
`pointing` switch it owns. Validated point batches from `POST
/<pointer-token>/point` reach the Console, which keeps a pure `Marks` model. A
new Presenter overlay hook paints the marks onto every composed frame, so the
operator and, through the existing tap, every viewer see them. A second hub
watch enforces the visible-capability invariant for `👆` the way #395 enforces
`LIVE ⏸`. Pointer-link pages turn Pointer Events into cell coordinates.

**Tech Stack:** Go (`net/http`, existing `cmd/internal/broadcast`,
`cmd/internal/terminal`, `cmd/internal/couchtty`), the vendored xterm.js 6.0.0
viewer, node tests for the page.

Issue: `workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md`
(the Spec there is the contract; this plan does not restate its decisions).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Marks` (Add, Clear, Overlay, Expired) | `cmd/internal/broadcast/marks.go` | new |
| `PointBatch`, `ParsePointBatch` | `cmd/internal/broadcast/point.go` | new |
| `RateLimit` (token bucket, injected clock) | `cmd/internal/broadcast/point.go` | new |
| `PointerLabel`, `PointerSGR`, `PointerShown` | `cmd/internal/broadcast/indicator.go` | modified |
| `Overlay` (func type) | `cmd/internal/terminal/tap.go` | new |
| `BroadcastCell` gains pointer/control cells; `RenderedStatusRow.Pointer`, `.Control` spans | `cmd/internal/couchtty/reserve.go` | modified |
| `cellAt`, `batchPoints` (viewer) | `cmd/internal/broadcast/web/viewer.js` | new |

- **Marks** — timestamped marked cells. `Add(batch, now)` fills lines between
  consecutive points *within one batch* (Bresenham), skips the chrome row, and
  keeps at most `cols*rows/8` cells, dropping the oldest. `Overlay(frame, now)`
  returns the frame with marked cells' backgrounds tinted, in one of three
  fade steps by age, and leaves text and foreground alone. `Expired(now)`
  reports when nothing is left to draw; `Clear` empties it. It has no clock or
  goroutine of its own.
  - **Relationships:** one per Console, created when pointing first turns on.
  - **DRY:** the one place mark geometry, fading and the coverage cap are
    decided; the overlay hook and the repaint timer only call it.
  - **Future:** per-helper colours (#412 chose one colour) widen `Add`'s
    input with a source id.
- **PointBatch / ParsePointBatch** — `{"cols","rows","down","points":[[c,r]...]}`.
  The parser takes the body bytes and limits and returns a batch or a typed
  rejection: size, too many points, unknown field, bad shape, out of range.
  Pure; the server maps rejections to 400.
- **RateLimit** — token bucket, about 30/s per link, `Allow(now) bool`.
- **PointerLabel / PointerSGR / PointerShown** — `👆` drawn in `PointerSGR`
  (amber background) at the fixed column right after `LiveLabel + " "`.
  `PointerShown(frame)` mirrors `IndicatorShown`. Drawer and checker share the
  constants (ARCH-DRY).
- **Overlay** — `func(Frame, FrameClass) Frame`, applied by the Presenter to
  every composed frame before painting. Couch's overlay returns a
  `FramePrivate` frame unchanged: marks are never drawn on the switcher
  (they keep ageing and show again, if still alive, when it closes).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Presenter.SetOverlay`, `Presenter.Refresh` | `cmd/internal/terminal/tap.go`, `presenter.go` | new | presenter goroutine |
| Hub pointer watch, `Hub.Current` | `cmd/internal/broadcast/hub.go` | modified | hub goroutine |
| Pointer routes (`/<ptoken>/…`, `POST point`), `caps` events | `cmd/internal/broadcast/server.go` | modified | `net/http` |
| `Session.EnablePointer/DisablePointer/PointerLink`, `Config.OnPoints/OnPointerOff` | `cmd/internal/broadcast/session.go` | modified | session state |
| Console pointer wiring (`console_pointer.go`) | `cmd/internal/couchtty/` | new | session, presenter, clipboard, timer |
| Pointer page mode | `cmd/internal/broadcast/web/viewer.js`, `viewer.css` | modified | browser Pointer Events, `fetch` |

- **Presenter.SetOverlay / Refresh** — `paintPublication` applies the overlay
  to the composed frame, keeps painting diffs against what it actually
  painted (`p.previous`), and hands the tap the overlaid frame, so viewers see
  marks. History rows come from the endpoint's `HistoryWindow`, never the
  frame, so marks can't reach scrollback. `Refresh` repaints the current
  selection (endpoint or panel) through the same path, for fade steps with no
  new child output. The panel's base frame is kept for that.
- **Hub pointer watch** — a second `off | shown | hidden` watch on
  `PointerShown`, armed by `ArmPointer()` and disarmed by `DisarmPointer()`.
  Past the grace it calls `OnPointerHidden` and disarms, without ending the
  hub. `Current()` reports the last accepted frame's geometry and class, so
  the session can drop stale-size and private-frame points.
- **Session** — owns a `PointerState` (`mu`, token, `pointing`), minted once
  per session on first enable. The `Server` holds a pointer to that
  `PointerState`, its only seam into pointer state, and reads it per request
  through methods (`Match(token) (pointer bool)`, `On()`), comparing in
  constant time. The view token stays immutable in `ServerOptions`. `POST point` is accepted only while pointing is on,
  the marker isn't hidden, the batch's cols×rows match `Current()`, and the
  frame is public. Accepted batches go to `Config.OnPoints`, never anywhere
  else. Disabling, by the operator or the watch, flips `caps` on pointer
  streams and calls `OnPointerOff`.
- **Console pointer wiring** — 👆 left-click toggles, right-click re-copies,
  👽 gives a notice. It owns `Marks` and the overlay closure (both under a
  mutex), plus a fade timer that calls `Presenter.Refresh` while marks live.
  Turning off clears the marks.
- **Pointer page** — on `caps` with pointer on, the page captures Pointer
  Events on the screen (`touch-action: none`), maps pixels to cells
  (`cellAt`), batches a stroke every ~50ms (`batchPoints`, each batch
  repeating the previous batch's last point), and POSTs to the relative URL
  `point`. On `caps` off it stops and says so.

### Operating envelope (ARCH-CONSTRAINTS)

- **Input:** at most 4 KB and 64 points per request, about 30 requests/s per
  link; beyond that 429 (rate) or 400 (shape). One helper is the usual case;
  several share the link's budget.
- **Screen:** at most 1/8 of the grid marked at once. A fade lasts about 3s
  in 3 steps, so at most about 3 extra repaints per stroke, and none when no
  marks live.
- **Paint path:** the overlay copies the frame only when marks exist; without
  marks it returns the frame unchanged.

### Trust boundaries (ARCH-SECURE)

- **Internet → `POST point` (untrusted):** constant-time token check,
  `MaxBytesReader`, JSON only, unknown fields rejected, nothing reflected,
  coordinates validated against the current grid. The only effect is cells
  tinted on Couch's own overlay; no byte reaches a child or the input path.
- **View link:** unchanged, GET-only (#395), and pinned by a test that POSTs
  to it.
- **Abuse by a pointer holder:** bounded by the rate limit and the coverage
  cap; marks are dropped on the status row (so the fail-safes can't be
  tripped) and while the switcher is open (the fleet list stays private). The
  operator's one-click off clears every mark at once.
- **Token handling:** pointer token 256-bit and distinct from the view token,
  never in notices or logs (the `*url.Error` scrub stays), `no-referrer`,
  `no-store`. CSRF is moot: the path token is the secret and the page sends no
  cookies.

### Ordering (ARCH-ORDER)

Console pointer phase (only while the broadcast is live):

| State | Event | Result |
|-------|-------|--------|
| none | click 👆 | mint token, pointing on, arm watch, copy link, notice |
| on | click 👆 | pointing off, clear marks, `caps` off, notice |
| off | click 👆 | pointing on (same token), arm watch, re-copy, notice |
| on/off | right-click 👆 | re-copy link |
| on | watch hidden past grace | pointing off, clear marks, notice |
| any | broadcast ends | token dies with the session; marks cleared |
| on | POST while marker hidden / stale size / private / chrome row | points dropped |

The session serializes `pointing` behind its mutex; `OnPoints` reaches the
Console loop through `runTerminalCommand`, after the session's own check, so a
batch already in flight when the operator turns pointing off is dropped by the
Console's own phase check too. Tests inject both orders: off-then-batch and
batch-then-off.

### Lock discipline

The overlay runs on the Presenter goroutine inside every paint, so any lock it
takes must be a **leaf**: never held while calling the Presenter, the session
or the hub.

- **`marksMu`** (Console): guards `Marks` and nothing else. It is a separate
  mutex, never `c.mu`. The overlay closure takes only `marksMu`. Console code
  takes it only around `Marks` calls and releases it before calling
  `Presenter.Refresh`, `SetOverlay`, the session or `setNotice`.
- **`PointerState.mu`** (session): guards the pointer token and `pointing`.
  It is a leaf too. The session calls `OnPoints`/`OnPointerOff` **after**
  releasing it, so a Console handler that calls back into the session (for
  example `DisablePointer`) can't deadlock.
- **Presenter goroutine** never calls the session or hub; the tap
  (`Offer`) and the overlay are its only outbound calls, and both are
  non-blocking or leaf-locked.
- **Test:** a stress test paints continuously while batches arrive and
  pointing toggles, under `-race` with a deadline; a deadlock fails it.

### Lifetimes (ARCH-FUNERAL)

- **Pointer token:** in memory; dies with the session.
- **Marks:** in memory; cleared on off, on broadcast end, and as they expire.
- **Fade timer:** runs only while marks live; stopped on Console teardown.
- Nothing is written to disk; #395's no-persistence test is extended to POST
  marks.

### Test strategy (one line per risky function)

- **`Marks`:** line fill (diagonals, single points), chrome-row drop, the
  1/8 cap with oldest-first eviction, fade steps by age, expiry, and that
  overlay touches only backgrounds. A table test with a fixed clock.
- **`ParsePointBatch` / `RateLimit`:** adversarial bodies (oversize, 65
  points, unknown field, negative or huge coordinates, wrong types, trailing
  data); a bucket under a manual clock.
- **Server pointer routes:** methods × routes × {view, pointer, wrong} tokens;
  POST only on the pointer link's `point`; 403 while off; 404 after the
  broadcast ends; `caps` on join and on flip; the view stream never gets
  `caps`.
- **Hub pointer watch:** the #395 interleaving property test, extended with
  arm/disarm and pointer-marker on/off frames. Invariant: the watch fires only
  after arm with the marker hidden past grace, and never ends the hub.
- **Presenter overlay:** marks present in the painted bytes and in the tapped
  frame; absent from history (oracle); `Refresh` repaints fading with no new
  output.
- **Console:** clicks and right-clicks; ordering of off against an in-flight
  batch (both orders); private-frame and chrome-row drops; the fail-safe when
  `👆` is clipped; nothing written to the child.
- **Viewer:** `cellAt` at fitted font sizes and edges; `batchPoints` overlap.
- **Emoji width:** `👆`/`👽` are 2 columns in `textwidth`, `ansi`, Couch's
  emulator and headless xterm.js.

---

## Chunk 1 — M1: marks and the Presenter overlay

### Task 1.1: Emoji width check
- [ ] Test (`broadcast/indicator_test.go`): `PointerLabel` and `👽` are 2
      columns in `textwidth.Width` and `ansi.StringWidth`, and advance the vt
      emulator's cursor by 2. Oracle test (`terminal/render_oracle_test.go`):
      headless xterm.js advances 2 for each. If any disagrees, stop and
      re-plan the markers.
- [ ] Commit `#412 M1: broadcast: pointer marker constants and emoji width`.

### Task 1.2: `Marks`
- [ ] Failing table tests (above), then the implementation in `marks.go`.
      Fade: steps at ages <1s, <2s, <3s, using three amber backgrounds from
      strong to faint; gone at ≥3s.
- [ ] Commit `#412 M1: broadcast: Marks model`.

### Task 1.3: Presenter overlay and `Refresh`
- [ ] Failing tests (`terminal/overlay_test.go`): with an overlay that tints
      cell (2,1), the painted bytes and the tapped frame carry the tint; a frame
      painted with no overlay set is byte-identical to today's; `Refresh`
      after the overlay changes repaints with no endpoint output; history
      oracle: overlay tint never appears in scrollback; a private panel's
      overlay call receives `FramePrivate`.
- [ ] Implement `SetOverlay(ctx, Overlay)` (ordered via `p.call`), apply it in
      `paintPublication`, keep the panel base for `Refresh`, and implement
      `Refresh(ctx)`.
- [ ] Atlas: terminal.md gets the overlay paragraph. `milestone-close M1`.

## Chunk 2 — M2: the pointer link in the broadcast

### Task 2.1: `PointBatch`, `RateLimit`
- [ ] Failing tests, then `point.go`.

### Task 2.2: Hub pointer watch and `Current`
- [ ] Failing tests: arm, then marker hidden past grace → `OnPointerHidden`
      once, hub still live, watch disarmed; marker returns before grace → no
      call; never fires unarmed; `Current` reports geometry and class of the
      last accepted frame. Extend `TestHubRandomInterleavings` with
      arm/disarm/pointer-marker ops and the invariant above (mutation-check
      it).
- [ ] Implement in `hub.go` (a second tagged watch, same shape as BR-1's).

### Task 2.3: Session and server
- [ ] Failing tests: `EnablePointer` mints once and returns the same link
      after off/on; `DisablePointer` flips `caps` and `POST` → 403;
      stale-size, private-frame and chrome-row batches are dropped (never
      reach `OnPoints`); accepted batches reach `OnPoints`; the method/route/
      token table; `caps` on join after `theme` for pointer streams only; the
      view link still 405 on POST; after `Stop`, both links refuse; the
      no-persistence walk also POSTs marks.
- [ ] Implement routes in `server.go` (pointer-token asset routes share the
      `assets` table), pointer state in `session.go`.
- [ ] Atlas: broadcast.md gets the pointer link section. `milestone-close M2`.

## Chunk 3 — M3: Couch controls and marks

### Task 3.1: Status row
- [ ] Failing tests: while live the row is `LIVE ⏸ 👆 👽` (👆 dim or
      `PointerSGR`); spans for 👆 and 👽 are exact; a drawn active row
      satisfies `PointerShown`, a dim one doesn't; narrow widths clip 👆
      before `LIVE ⏸`; off/starting/stopping rows are unchanged.
- [ ] Implement in `reserve.go`.

### Task 3.2: Console wiring
- [ ] Failing tests (`console_pointer_test.go`, fixture as in #395):
      left/right clicks per the ordering table, with clipboard and notices; a
      POST lands as tinted cells in the operator's frame and the broadcast; fade
      to nothing after ~3s (manual timer); off clears marks; both orders of
      off against an in-flight batch; switcher open → points dropped; clipped
      👆 → pointing off; 👽 click → notice only; no byte to the child.
- [ ] Implement `console_pointer.go` (route clicks from `routeMouseEvent`:
      button 0 and button 2 on the status row).
- [ ] Atlas update. `milestone-close M3`.

## Chunk 4 — M4: pointer page and smoke

### Task 4.1: Viewer pointer mode
- [ ] Failing node tests: `cellAt` maps pixel points to cells at several
      font sizes and edges; `batchPoints` splits strokes with overlap; `caps`
      on/off toggles capture and the notice. The page lint now allows exactly
      one `fetch` target, the relative `point`, and still bans storage APIs.
- [ ] Implement in `viewer.js` / `viewer.css` (`touch-action: none` only in
      pointer mode).

### Task 4.2: Smoke and close
- [ ] Move to pair:0 (merge main first, never rebase), `make build`, and ask
      the operator for the iPad smoke through the named tunnel; record it in
      the Log.
- [ ] Full verification (`make -k test`, scratchpad-TMPDIR changelog,
      `go test ./...`, compared against a main archive), `sdlc close`,
      `sdlc pr`, `sdlc merge` on the operator's word.

## Revisions

- **2026-10-08 (Task 1.1)** — the width check failed in the viewer:
  xterm.js 6.0.0 defaults to Unicode 6 widths and counts `👆`/`👽` (and most
  emoji) as one column, while Couch counts two. Per the plan's decision point,
  the fix is in the viewer, not the markers. `@xterm/addon-unicode11` 0.9.0
  is vendored and loaded (with `allowProposedApi`), and the oracle pins and
  loads the same add-on. On a 65-glyph sample it agrees with Couch except for
  emoji with a variation selector and joined/skin-tone sequences (recorded in
  `VENDOR.md`). `@xterm/addon-unicode-graphemes` 0.4.0 was measured and
  rejected: it counted plain emoji as one column. This also fixes emoji
  placement for #395 viewers.
- **2026-10-08 (M2 security review)** — a dedicated fresh-context security
  review of the input path found no blocking issues. Taken: 413 only for a
  real `MaxBytesError` (else 400); body read budget 2s (was 5s); content type
  via `mime.ParseMediaType`; event-stream writes get a deadline of two pings,
  so a viewer that stops reading is dropped and frees its slot (a #395 gap,
  `TestServerDropsStalledViewer`, mutation-checked); `Marks.Add` trims the
  batch to the cap as it grows. **Contract for M3 (finding L1):** a batch can
  pass the session's checks and then reach the Console after pointing turned
  off or the screen turned private. The Console re-checks its pointer phase
  and the frame class under `marksMu` before `Marks.Add`, the overlay skips
  `FramePrivate`, and turning pointing off clears marks. `OnPoints` does
  bounded work and routes coordinates only to `Marks`.
- **2026-10-08 (M4 smoke)** — the status-row rule narrows from "never the last
  row" to "never the broadcast controls": `StatusGuardCols` (the width of
  `LIVE ⏸ 👆 👽`) at the left of the last row, enforced at `Marks.Add` and in
  the overlay; a couchtty test pins that the guard ends where `👽` does. The
  fail-safes read only those cells, so the rest of the tab bar is safe to
  mark. Right-click on `LIVE ⏸` re-copies the view-only link.
- **2026-10-08 (M4 review, BR-14)** — names as shipped, against the Core
  concepts table above: `Marks` exposes `Live`, `NextChange` and `SetBlend`
  (no `Expired`); the viewer's batching is `chunkStroke` (not `batchPoints`),
  and its page wiring is `pointerMode`, with its post and timers injectable for
  tests. The fade is hold 1.5s plus 0.5s in 10 steps (`MarkHold`, `MarkFade`),
  not 3 steps over 3s.
