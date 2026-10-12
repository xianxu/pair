# #460 live viewer — plan

Goal: replace #454's stub viewer page with the live viewer of #457's contract: the cast's records rebuild the page in a
script-free sandboxed frame, scaled to fit, joinable mid-cast, with the operator's hover and pointer shown.

Seams (TL-confirmed 10-12, delivery 9d90bc7b): `{k:hover,id}` (0 = none) and `{k:ptr,x,y}` (top-page CSS px) in
`events.jsonl`, outside `Canonical` (#458 emits and reduces them); assets at `<cast>/assets/<hash>`, `resolveAsset(abs)`
→ `assets/<hash>` (#459 mounts its proxy there); the viewer rewrites `url()`s in streamed CSS text itself; `boxOf(id)`
gives wave 2 the viewer-computed note box; the frame is `sandbox="allow-same-origin"` (no `allow-scripts`) and the
shell owns all input, so #466's reverse channel lives in the shell.

## Architecture

Three JS modules under `cmd/internal/webcast/web/`, one responsibility each (ARCH-DRY: the reducer's semantics live in
one JS place):

1. **`model.js` — pure, no DOM.** A port of `recording.Reduce`'s `Apply` and `State.Canonical()`: node table, sheets,
   adoptions, fonts, assets, inputs, scrolls, selection, focus, URL/title/viewport, segment and live flag, plus hover
   and pointer. Same refusals as the Go reducer (unknown node → error, record not applied). `canonicalJSON()` emits
   Go's exact bytes (`MarshalIndent` with one-space indent, sorted map keys, HTML-escaped `<>&`, U+2028/9), so its
   SHA-256 equals the fixtures' pins. Tested in node against every fixture pin: the JS port can't drift from the Go
   reducer without a red test.
2. **`render.js` — DOM.** `Renderer(frameEl, model, resolveAsset)` applies each record the model accepted to the
   frame's document: builds real nodes (HTML/SVG/MathML namespaces, doctype, shadow roots open and closed, template
   content, nested iframe documents written with `document.open/write('<!DOCTYPE html>')/close` so they are synchronous
   and in standards mode). Neutralizing (Principle 3, defense in depth over the sandbox):
   - `on*` attributes are never set; `<iframe>` gets `sandbox="allow-same-origin"` and no `src`/`srcdoc` (its content
     comes from `doc` records); `<noscript>` is hidden by a viewer-owned constructed sheet.
   - URL attributes (`src`, `srcset` candidates, stylesheet `href`, `poster`, SVG `href`/`xlink:href`) resolve against
     the document URL and go through `resolveAsset`; no entry → the attribute is dropped (nothing reaches the origin).
   - CSS text (`<style>` text, `style=` attributes, `sheet` records) has `url()`s rewritten and `:hover` rewritten to a
     viewer class; owned `sheet` text replaces a `<style>` owner's text (a `<link>` owner gets a `blob:` stylesheet).
   - `<link rel=stylesheet>` bodies are fetched through the proxy, `:hover`-rewritten and served from a `blob:` URL
     (revoked when replaced or removed).
   Observer state is applied as properties (`value`, `checked`, `selected`, `scrollLeft/Top`, the frame window's
   scroll, re-applied after each batch because content inserted after a scroll can clamp it). Fonts are added as
   `FontFace`s in the frame realm. An iframe element that is moved (a browser reloads a moved frame) is rebuilt from
   the model. `canonical()` walks the **rendered** DOM (structure, ids and order from the DOM; recorded attribute and
   text values from the model, since the render deliberately differs there), so the conformance check tests the
   rebuild, not the model twice. `boxOf(id)` is the node's box in top-page CSS px from the frame's own unscaled
   layout (frame offsets and scroll added).
3. **`viewer.js` — the shell.** Connects to `events`, feeds `rec` events to model+renderer, a `head` resets everything
   (an EventSource reconnect gets a fresh catch-up), `state`/`end` events keep their #454 meaning. Lays the frame out
   once at the operator's viewport (`vp` w×h) and scales it with a CSS transform to fit (`scale = min(W/w, H/h)`,
   never relayout, Principle 2); shows the URL and title as text; Paused / Ended overlays; hover = the viewer class on
   the hovered element and its ancestors (so the page's own `:hover` rules apply); the pointer is a dot in a shell
   overlay above the frame. The frame has `pointer-events: none`; the overlay is where wave 2 and #466 take input.

Go side:
- **`webcast.Hub`** — the real `recording.Live`: fan-out with catch-up, each record encoded once (`recording.Encode`)
  and shared by every viewer; passes `recording.RunLiveContract`. Memory: one segment's records plus assets and the
  document's notes (the catch-up rule); freed at the next `seg` and when the cast ends (ARCH-FUNERAL: in-memory, dies
  with the cast; no durable residue).
- **`Server`** gains `rec` events from `CastConfig.Live` (nil keeps the state-only stub, until #458 wires the recorder's
  `Tee` into it), and `model.js`/`render.js` among its embedded files. CSP: the frame inherits the shell's policy, so it
  gains `style-src 'self' 'unsafe-inline' blob:`, `img-src 'self' blob:`, `font-src 'self'`; `script-src 'self'`
  stays (inline scripts of the recorded page are blocked twice: CSP and sandbox).

## Tests

- **node** (`tests/webcast/*.test.mjs`, run by `TestNode`): model vs every fixture pin (hash equality); model refusals
  mirror the reducer's; catch-up at every cut of `segments` gives the same canonical; scale math; asset URL resolution
  (`srcset`, relative URLs, missing entries); CSS rewriting (`url()`, `:hover`); shell event handling with a fake
  EventSource (head resets, pause/end overlays, URL text).
- **Go**: `Hub` passes `RunLiveContract`; `Server` streams `rec` lines after `head` from a `FakeHub` with join-mid-cast
  and segment changes; CSP header.
- **Headless Chrome** (`WEBCAST_LIVE_CHROME=1`, like #454's live checks; never focuses): a fixture server serves the
  viewer, each fixture's records and `assets/<hash>` (rewriting served CSS `url()`s as #459 will). For every fixture at
  every pin: the rendered DOM's `canonical()` equals the Go reducer's `Canonical()`, and a 640×400 capture of the frame
  is compared with `frames/<n>.png` (fraction of pixels differing by more than a per-channel threshold), under a stated
  tolerance set from the measured values. A join-mid-cast run (catch-up then the rest) and the `segments` fixture's
  pause/resume/tab switch land on the same canonical as a full replay.

## Steps

- [ ] `model.js` + node tests against every fixture pin
- [ ] `render.js` + `viewer.js` shell (scale, URL, overlays, hover, pointer, `boxOf`)
- [ ] `Hub`, `Server` `rec` events, CSP; Go tests
- [ ] Headless fixture replay: canonical equality and screenshot comparison; tolerance measured and stated
- [ ] Atlas (`webcast.md`, `webcast-format.md` viewer section) and README if user-visible
- [ ] Smoke test (operator, desktop Chrome + iPad; via the TL, after the recorder lands)

## Lifecycle (ARCH-FUNERAL)

- Hub log: in memory, created with the cast, bounded by the catch-up rule (one segment plus assets and the document's
  notes; ~1.4 MB/min on the stress page), dropped at each `seg` and with the cast.
- `blob:` stylesheet URLs: created per `<link>`/owned sheet in a viewer tab, revoked on replace/remove and die with the
  tab.
- Nothing durable is created by the viewer.

## Revisions

### 2026-10-12 — #466's remote-driving seams (TL, delivery d44edcc4)
- **Presence:** the `events` route takes an optional `?viewer=ID` and calls `CastConfig.Presence(id) (leave func())`
  for the stream's life (nil: no hook). The seat logic (grace period, control returning to the operator) is #467's.
- **Shell contract gains** `nodeAt(pageX, pageY) → id` (the frame's `elementFromPoint` after page→client mapping,
  descending into same-origin child frames and open/closed shadow roots we hold) and `toPage(clientX, clientY)` (shell
  client px → top-page CSS px: undo the stage offset and scale, add the frame's scroll). Node-tested math; Chrome-tested
  hit-testing.
- **`ctl` records:** not rendered now (#467). The model ignores unknown kinds as the reducer does; the shell exposes an
  `onRecord` hook so #467 renders who's driving without touching the apply path. `POST /TOKEN/drive/*` is #467's.

### 2026-10-12 — as built, after the close review (round 1)
- **Fan-out:** no `webcast.Hub`. The TL promoted `recording.FakeHub` to `recording.Hub` (deprecated alias kept), and
  `Server.events` encodes each record per viewer. That is cheap next to the SSE write, and sharing encoded bytes would
  double the Hub's memory per segment. `recording.Live` gained `Done()`, so the route ends a dropped viewer's stream
  but keeps an ended fan-out's viewers on state/end (BR-1). This is an additive seam change, made under the TL's
  one-writer grant for that file.
- **CSP** also allows `data:` in `img-src` and `font-src`: inline data URLs in recorded CSS and attributes, which
  fetch nothing.
- **Records outside Canonical** (hover) never refuse in the viewer, so they never force a resync (BR-2).
- **Not done here, for the TL:** `CatchUp` keeps every `ptr`/`hover` of the segment. A latest-only rule is a format
  change (#457/#458). `image-set()` string URLs are not rewritten; the CSP's `img-src 'self'` stops them reaching
  the origin.
