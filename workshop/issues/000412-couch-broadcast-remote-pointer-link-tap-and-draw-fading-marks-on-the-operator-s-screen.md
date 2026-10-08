---
id: 000412
status: open
deps: [pair#395]
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '5489d1d7c2fab58d317bd3709998139f36264e27' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- While broadcasting, a click on `👆` copies a pointer link; a helper opening
  it on an iPad can tap and drag, and fading marks appear on the operator's
  screen and every viewer's, at the touched cells.
- A second click on `👆` turns pointing off at once: the helper's input has no
  effect and their page says pointing is off, while they keep watching. A
  third click turns pointing back on for the same link, with no new link.
- Stopping the broadcast ends the pointer link along with the view-only
  link: open pointer pages show the broadcast ended, and both links then
  refuse every request, shown by a test.
- The view-only link accepts no input (unchanged from #395), shown by a test.
- Pointer input never reaches a child program, shown by a test.
- A test shows that if the active `👆` can't be drawn, pointing turns off and
  the link stays usable as view-only.

## Plan

- [ ] Finish the brainstorm (open questions above), then write the plan

## Log

### 2026-10-08

- Filed from the operator's idea after #395 landed: a narrow remote-pointer
  channel short of full remote control. Status-row design (`LIVE ⏸ 👆 👽`)
  by the operator; 👽 waits for #407.
