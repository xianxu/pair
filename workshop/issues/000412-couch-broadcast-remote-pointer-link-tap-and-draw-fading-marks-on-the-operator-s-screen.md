---
id: 000412
status: working
deps: [pair#395]
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '64909a604af8ffe163e9692dbd5910ea7cce397a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T09:21:13-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
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
  produces an amber background tint that fades in steps and is gone about 3s
  after the last input. A Presenter overlay hook applies it to the composed
  frame before painting, so the operator and every viewer see it. While marks
  are alive, Couch schedules repaints so the fade advances.
- **Status row:** `👆` is dim when off or never minted, on amber when
  pointing is on. Left-click toggles, right-click re-copies. `👽` is drawn
  but inert (#407). The hub checks each frame for the active-`👆` marker, the
  way it checks `LIVE ⏸`; hidden for a second turns pointing off.

- **Spec review decisions (2026-10-08):**
  - **Marks never touch the status row.** Points on Couch's chrome row are
    dropped, so a helper can't cover `LIVE ⏸` or `👆` and trip a fail-safe.
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
    paints that frame plus the current marks, repainting while marks fade (3
    steps over about 3s). Marks never enter the parent's scrollback: history rows
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
  in the broadcast. The tint fades in steps and is gone about 3s after the last
  input.
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
- Points on the status row and points while the switcher is open are
  dropped; a stroke over `LIVE ⏸ 👆` leaves both indicators intact.
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

## Plan

- [ ] Finish the brainstorm (open questions above), then write the plan

## Log

### 2026-10-08

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
