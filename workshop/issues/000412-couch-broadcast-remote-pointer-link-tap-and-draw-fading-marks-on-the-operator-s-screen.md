---
id: 000412
status: working
deps: [pair#395]
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours: 4.85
card_mirror: 'a6dce03f8514661188d67d4298be09941e71de1e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T09:21:13-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen)

## Problem

#395 broadcasts the composed Couch screen view-only, which makes remote
presentation possible. A remote helper (a colleague, or customer support
walking someone through a problem) wants to point at the screen: tap a spot,
or circle something with a finger on an iPad. Full remote control (#407) is
far more than that needs, and far more dangerous. A pointer channel carries
only grid coordinates and can never reach a child program, so it can be both
useful and narrow.

## Spec

Settled with the operator on 2026-10-08 (brainstorm, in progress):

- **Capability links, independent and revocable.** A broadcast can have
  several links active at once, each granting a different power. The #395
  view-only link stays exactly as it is: no inbound channel. Pointing is a
  **separate pointer link**, handed only to the helper; it shows the same
  live view and also accepts pointer input. Remote control (#407) will be a
  third link of the same kind.
- **Status-row controls.** While live the row leads with `LIVE ⏸ 👆 👽`.
  - `LIVE ⏸` (red) is the broadcast, unchanged. Stopping it, by a click,
    Ctrl+Alt+b, the LIVE fail-safe or Couch exiting, ends **every** link
    (view-only, pointer, later control); their pages show the broadcast ended,
    the links never work again, and the next broadcast mints new ones.
  - `👆` starts inactive. The first plain click mints the pointer link,
    copies it, and turns pointing on. Later clicks toggle pointing off and on
    for **the same link**: off downgrades it to view-only (holders keep
    watching, the page says pointing is off, their input is refused); on
    restores pointing on that link and re-copies it. Right-click re-copies the
    link whenever one exists.
  - The pointer link lives as long as the broadcast. Once handed out, it is at
    least a view link until the broadcast stops; stopping `LIVE ⏸` is how
    the operator cuts every link off.
  - `👽` (remote control) is laid out but inert until #407.
  - The controls appear only while live; a stopped broadcast leaves the row as
    today. No new key bindings: the controls are mouse-only (Ctrl+Alt+b still
    starts and stops the broadcast).
- **Visible-capability invariant**, extending #395's LIVE rule: no capability
  is active unless its marker is drawn as active on the operator's screen. If
  the active `👆` can't be drawn for a second (for example, clipped by a
  narrow terminal), pointing turns off; the link survives as view-only.
- **What a pointer does: fading marks.** A tap highlights that cell; a drag
  leaves a trail of highlighted cells along its path. Marks tint the cell
  background (the text stays readable) and fade out a few seconds after the
  helper stops. Couch draws them as an overlay on the composed frame, so the
  operator and every viewer see them. Pointer input never reaches a child
  program: no keystrokes, no mouse events, only coordinates Couch draws.
- **Emoji width.** `👆` and `👽` default to emoji presentation (2 cells,
  drawn from an emoji font), unlike `⏸` (text presentation, 1 cell). Test
  the viewer with these exact glyphs; if the browser's emoji fallback drifts
  off the 2-cell grid (the #395 italic-face lesson), pin emoji to their cell
  box in the viewer.

- **Many helpers:** anyone holding the pointer link can point (up to the
  16-viewer cap); all marks look alike.

Design (proposed 2026-10-08):

- **Server:** the session gains an optional pointer token (256-bit, separate
  from the view token) and a `pointing` switch. `/<pointer-token>/` serves the
  same page and stream, plus `POST /<pointer-token>/point`. That POST is
  accepted only while pointing is on (403 otherwise). Open streams get an
  `event: caps` when pointing flips, so the page can say so. The view link is
  unchanged: GET-only.
- **Channel:** `{"down":bool,"points":[[col,row],...]}` in grid cells (the
  viewer converts pixels to cells), at most 4 KB, 64 points per request and
  about 30 requests/s per link. Coordinates are clamped to the grid; anything
  else gets 400 with nothing echoed. The coordinates feed only the mark
  overlay.
- **Marks:** a pure `Marks` model (taps, strokes with line fill, timestamps)
  produces an amber background tint that holds 1.5s after the last input,
  then fades fast over 0.5s. A Presenter overlay hook applies it to the composed
  frame before painting, so the operator and every viewer see it. While marks
  are alive, Couch schedules repaints so the fade advances.
- **Status row:** `👆` is dim when off or never minted, on amber when
  pointing is on. Left-click toggles, right-click re-copies. `👽` is drawn
  but inert (#407). The hub checks each frame for the active-`👆` marker, the
  way it checks `LIVE ⏸`; hidden for a second turns pointing off.

- **Spec review decisions (2026-10-08):**
  - **Marks never touch the broadcast's controls.** Points on `LIVE ⏸ 👆 👽`
    (the first `StatusGuardCols` columns of Couch's status row) are dropped,
    and the overlay never tints them, so a helper can't cover an indicator and
    trip a fail-safe. The rest of the tab bar can be marked (revised after the
    M4 smoke).
  - **Strokes:** the page sends a stroke as short batches (each repeating the
    previous batch's last point); line fill happens only within one request, so
    two helpers' strokes never join.
  - **Coverage cap:** at most 1/8 of the grid's cells are marked at once,
    oldest dropped first. The rate limit (about 30 requests/s) is per link.
  - **Off clears:** turning pointing off, by a click or by the `👆` fail-safe,
    removes every mark at once.
  - **The `👆` watch:** the Session owns `pointing`. The hub gains a second
    watch, for the active-`👆` marker (`PointerLabel`/`PointerSGR`, at a fixed
    column right after `LIVE ⏸ `, shared by the drawer and the checker). It
    arms when pointing turns on. Hidden past the 1s grace, it turns pointing off
    without ending anything and tells the Console, which redraws `👆` dim.
    Points are dropped while the marker is hidden.
  - **State on join:** a pointer-link stream gets `event: caps` on join (after
    `theme`) and on every flip; view-link streams never get it.
  - **Repaint and history:** the Presenter keeps the frame without marks and
    paints that frame plus the current marks, repainting as marks fade (once
    when the 1.5s hold ends, then every 50ms through a 0.5s fade). Marks never enter the parent's scrollback: history rows
    come from the endpoint, not the overlaid frame.
  - **Private frames:** while the switcher is open (a private frame), points
    are dropped. Helpers see the placeholder and must not mark the fleet list.
  - **Smaller states:**
    - `👆` and `👽` show only while live (not while starting or stopping).
    - A click on `👽` gives "Remote control isn't available yet".
    - Toggles give "Pointing on — link copied" and "Pointing off".
    - Each turn-on re-copies the link.
    - Each request carries the grid size the page saw (cols×rows); points sent
      against a stale size are dropped, not clamped.
  - **Server hygiene:** #395's "GET only, no body read" holds for every path
    except `POST /<pointer-token>/point`. That route reads through
    `http.MaxBytesReader`, requires `Content-Type: application/json`, and
    rejects unknown fields. Both tokens are compared in constant time and keep
    `no-referrer`/`no-store`. The pointer token never reaches a notice or log.
    Cross-site request forgery is moot: the token in the path is the secret,
    and the page sends no cookies.
  - **Emoji width:** a test asserts `👆` (U+1F446) and `👽` (U+1F47D) are 2
    columns in `textwidth.Width`, `ansi.GraphemeWidth`, Couch's emulator and
    the headless xterm.js oracle. A narrow-width status-row test shows `👆`
    clipping before `LIVE ⏸` does, and its own fail-safe firing.
  - **iPad page:** pointer-link pages set `touch-action: none` on the screen
    and use Pointer Events, so a drag draws instead of scrolling or zooming.

## Done when

Automated (each a test):

- A POST to the pointer link, while pointing is on, puts the mark tint on
  exactly those cells in the frame the operator's terminal is painted with and
  in the broadcast. The tint holds 1.5s after the last input, then fades over
  0.5s (blending into the operator's background with truecolor).
- The first click on `👆` mints a pointer token distinct from the view token
  and copies its link; right-click re-copies it; a new broadcast mints a new
  pointer link.
- A second click on `👆` turns pointing off at once: marks clear, POSTs get 403,
  and open pointer pages get `caps` off but keep streaming. A third click turns
  pointing on for the same link, with no new link.
- Stopping the broadcast ends the pointer link along with the view-only link:
  open pointer pages get `end`, and both links then refuse every request (404,
  or no connection once the listener is down).
- The view-only link accepts no input (unchanged from #395).
- Pointer input never reaches a child program.
- Limits: over 4 KB, more than 64 points, a wrong content type, unknown
  fields, or a stale grid size each get rejected or dropped, with nothing
  echoed. The rate limit holds, and the coverage cap keeps marked cells at or
  under 1/8 of the grid.
- Points on the broadcast's controls and points while the switcher is open
  are dropped; a stroke over `LIVE ⏸ 👆 👽` leaves the indicators intact, and
  a point on the tab bar to their right lands.
- If the active `👆` can't be drawn, pointing turns off (marks cleared) and the
  link stays usable as view-only.
- A pointer page learns its state on join (`caps` after `theme`); a view page
  never gets `caps`.
- Marks never appear in the parent terminal's scrollback.
- `👆` and `👽` are 2 columns everywhere (the width test above). A click on
  `👽` changes nothing but a notice.

Manual smoke (operator):

- While broadcasting through the named tunnel, a click on `👆` copies a
  pointer link. A helper on an iPad taps and draws circles, and fading marks
  appear on the operator's screen and on every viewer's.

## Estimate

Derived per `estimate-logic-v3.1` (`impl=` at 40% of the v2 table). Design
×0.2 for the reviewed spec and durable plan (except the smoke round, which is
operator-bound); +15% design buffer; familiarity 1.0, since #395 built this
code. Items, in order: M1 emoji check, Marks, Presenter overlay; M2 point
parsing and rate limit, hub watch, session and server; M3 status row,
Console wiring; M4 viewer pointer mode, browser Pointer Events discovery, iPad
smoke round; atlas; four milestone reviews.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.06 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.06 impl=0.20
item: smaller-go-module      design=0.06 impl=0.14
item: smaller-go-module      design=0.06 impl=0.20
item: greenfield-go-module   design=0.25 impl=0.30
item: smaller-go-module      design=0.06 impl=0.14
item: tui-screen             design=0.25 impl=0.26
item: greenfield-go-module   design=0.25 impl=0.22
item: real-api-discovery     design=0.00 impl=0.18
item: ux-rename-iteration    design=0.55 impl=0.08
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
item: milestone-review       design=0.00 impl=0.14
design-buffer: 0.15
total: 4.85
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen-plan.md`.

- [x] M1 — emoji width check; `Marks` model; Presenter overlay hook and `Refresh`
- [x] M2 — `PointBatch`/`RateLimit`; hub pointer watch and `Current`; session
      pointer token, `pointing` switch, `POST point`, `caps` events
- [x] M3 — status row `LIVE ⏸ 👆 👽`; Console clicks, marks, fade, fail-safe
- [x] M4 — viewer pointer mode (Pointer Events, cell mapping, batching); iPad
      smoke; close

## Log

### 2026-10-08
- 2026-10-08: closed — All four milestones closed with SHIP (M1-M4), M2 with a dedicated security review and a 60s fuzz run. Operator iPad smoke through the named tunnel verified working (Log, rounds 1-3). Full verification: go test -race ./cmd/internal/{broadcast,couchtty} ok; make -k test fails only test-pair-embedded-runtime (PAIR_DATA_DIR scope conflict, same failure on an origin/main archive); test-changelog passes with scratchpad TMPDIR; go test ./... failures (artifactpath classification, continuation writer, couchcore spawn, gcruntime archive locator, launcher checkpoint/scoped-data-dir) all reproduce identically on the origin/main archive. The one failure in broadcast code, the stacked-godoc lint on #395 records.go, fixed in d73e5ba5. Follow-up #415 filed for viewer symbol glyphs.; review verdict: SHIP
- 2026-10-08: closed M4 — Round-1 fixes: BR-13 Spec/Done-when/atlas state the shipped fade (1.5s hold, 0.5s 10-step fade) and the controls-only guard, with a Revisions entry; BR-14 plan Revisions record shipped names (Live/NextChange/SetBlend, chunkStroke, pointerMode); BR-15 pointerMode injectable and node-tested (caps toggles class and hint, off and out-of-screen post nothing, stroke batches overlap, release posts down:false and stops the timer, off mid-stroke ends the stroke). Earlier M4 evidence stands: cellAt/chunkStroke node tests, one-fetch page lint, tab-bar pointing, LIVE right-click, fade tests; operator iPad smoke verified; go test -race ./cmd/internal/{broadcast,couchtty} ok.; review verdict: SHIP
- 2026-10-08: closed M3 — BR-10 fixed: README documents LIVE ⏸ 👆 👽, the pointer link, toggling, right-click re-copy, marks never typing/clicking or covering the status row, links ending with the broadcast, 👽 reserved (README tests pass). Earlier M3 evidence stands: status-row span/PointerShown tests; console pointer tests (mint+copy, marks on operator and viewer, off clears and 403, same link on re-enable, right-click, 👽 notice, switcher and late-batch drops, clipped 👆 watch, fade, end) and TestPointerStressNoDeadlock under -race.; review verdict: SHIP
- 2026-10-08: closed M2 — Pointer link: ParsePointBatch strict (unknown fields, trailing data, non-integers, out of grid, >64 points) with fixed errors, FuzzParsePointBatch 60s/815k inputs no failures; RateLimit bucket; route table (only POST /<pointer-token>/point reads a body; view link POST 405; wrong token 405); rejections 415/413/400/429 echo nothing; drops for stale grid, private frame, hidden marker; 403 while off with the link still view-only; same link on re-enable; caps on join and flip for pointer streams only; links end with the broadcast; re-entrant callback no deadlock; in-flight cap with slow bodies released by the read deadline. Hub pointer watch + Current, property test models it and catches two behaviour mutants. Dedicated security review: no blocking issues; hardening taken (413 only for MaxBytesError, 2s body budget, mime content type, SSE write deadline drops stalled viewers - TestServerDropsStalledViewer mutation-checked). go test -race ./cmd/internal/{broadcast,terminal,couchtty} ok unsandboxed.; review verdict: SHIP
- 2026-10-08: closed M1 — Round-1 fixes: BR-2 Marks.Add bounded (batch truncated to the cap, one sort per call; TestMarksAddWorstCaseIsCheap 300x100 grid, 64 far-apart points at a full cap, ~2ms per batch vs the 20ms bound); BR-3 overlay never tints the frame last row (TestMarksOverlayNeverTintsStatusRow: a mark moved onto the status row by a resize leaves LIVE intact). Earlier evidence stands: marker widths incl. headless xterm.js with the unicode11 add-on, Marks table tests, Presenter overlay/Refresh/scrollback oracle tests; go test -race ./cmd/internal/{terminal,couchtty,broadcast} ok.; review verdict: SHIP

- Filed from the operator's idea after #395 landed: a narrow remote-pointer
  channel short of full remote control. Status-row design (`LIVE ⏸ 👆 👽`)
  by the operator; 👽 waits for #407.
- Spec review (fresh context): found marks could cover the LIVE/👆
  indicators, strokes could join across helpers, there was no coverage bound,
  off didn't clear marks, the 👆 watch path was undefined, there was no state
  on join, no repaint/history mechanism, private frames and several smaller
  states were undefined, and Done-when had gaps. All folded into the Spec as
  decisions, and Done-when was rewritten as automated tests plus one manual
  smoke.
- Task 1.1 width check: headless xterm.js 6.0.0 counted `👆`/`👽` as 1 column
  (Unicode 6 tables), against Couch's 2, and the same was true for most emoji
  in viewers today. Vendored `@xterm/addon-unicode11` 0.9.0 in the viewer and
  the oracle: 61/65 sample glyphs now agree. The residual mismatch is VS16
  emoji (`❤️`, `⚠️`) and ZWJ/skin-tone sequences, recorded as known. The
  graphemes add-on was measured and was worse.
- M2: the pointer link is built (strict parser, rate limit and in-flight cap,
  per-request read deadline, routes, caps, session checks, hub pointer watch).
  At the operator's request, a dedicated security review of the input path
  found no blocking issues; its hardening was taken (see the plan's
  Revisions). `FuzzParsePointBatch` ran 60s (815k inputs) with no failures.
- M4 smoke round 1 (operator): two changes. Pointing on the tab bar was
  impossible because marks dropped the whole status row; the guard now covers
  only the broadcast controls `LIVE ⏸ 👆 👽` (`StatusGuardCols`, at input and in
  the overlay), so a helper can point at a thread chip. Right-click on
  `LIVE ⏸` now re-copies the view-only link (left-click still stops),
  mirroring `👆`.
- M4 smoke round 2 (operator): the 3-step, one-shade-a-second fade felt
  clumsy. Marks now hold 1.5s at full amber, then fade fast over 0.5s in 10
  steps (50ms each). With truecolor and a known background they blend into
  the operator's real background; otherwise they walk a short 256-colour
  ladder. Repaints follow the steps.
- M4 smoke round 3 (operator, iPad through the named tunnel, pair:0 at
  19f5d543): verified working: pointing on the screen and the tab bar, the
  controls protected, the hold-then-fast fade, toggling with the same link,
  right-click re-copies on `👆` and `LIVE ⏸`.

## Revisions
- **2026-10-08 (M4 review, BR-13)** — the Spec and Done-when now state what
  the smoke led to: marks avoid only the broadcast's controls, not the whole
  status row, and they hold 1.5s then fade over 0.5s (was 3 steps over 3s).
