# #458 recorder: CDP DOM stream per frame plus isolated-world observer — plan

Issue: `workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md`.
Contract: `atlas/webcast-format.md` (#457) and `cmd/internal/webcast/recording`. Reference mapping: #457's
`recording/internal/fixturegen/cdpconv.go` (prototype CDP log → records) and #453's `record.mjs` (branch
`453-recorder-prototypes`).

**Goal:** a real recorder on headless/windowed Chrome that writes recordings #457's validator accepts and its reducer
rebuilds, plugged into `webcast.Cast`, plus the approved additive v1 format changes.

## Format additions (M1; additive v1, a Revisions note on the format page)

Shapes as coordinated by the TL with #460 (10-12):
- `{"k":"hover","id":N}`: the element under the operator's pointer; `0` is none. events.jsonl.
- `{"k":"ptr","x":X,"y":Y}`: a sampled operator pointer position, top-page CSS px. events.jsonl. Sampled on movement,
  at most 4 per second.
- Both are tracked by the reducer (`State.Hover`, `State.Ptr`), reset by `seg`, `hover` cleared when its node goes
  (as `Focus` is). **Outside `Canonical`**, so no fixture pin moves.
- **Passwords:** `input` gains `secret: true`: the control has a non-empty value that isn't recorded. The reducer
  (hence the validator) refuses an `input` with a non-empty `v` on an element whose current `type` is `password`,
  and `secret` together with `v`; it refuses an `ins`/`doc`/`kids`/`attr` that gives an `input type=password` a
  non-empty `value` attribute. The recorder never reads a password value (the observer reports `secret` only) and
  drops the `value` attribute of a password input. `secret` folds into Canonical's `$input` only when set
  (omitempty), so pins don't move.
- **Advisory from #457:** `ValidateDir` also reports anything in `frames/` that isn't `index.jsonl` or a frame's
  `<n>.png` (a stray `.tmp-*`, as `dir.go` already claims). Test first.

## Recorder (M2): `cmd/internal/webcast/recorder`

ARCH-PURE split:
- **`mirror` (pure):** CDP DOM state per session (node tree from `DOM.getDocument` pierce + `includeWhitespace:
  all`, mutation events) and the recording's id map (`backendNodeId` → id, per session, kept for the whole recording,
  so a resume of the same document keeps ids). Translates DOM events into records the way `cdpconv` does: `ins`,
  `rm`, `kids` (setChildNodes and same-document re-snapshots), `attr`, `text`, `shadow`, user-agent shadow trees and
  pseudo elements left out, the in-page control (`data-couchcast-ui`) left out, password `value` attributes dropped.
  It also resolves observer reports by **path** (child indexes from the frame's document, a shadow-root step for open
  roots) against its own tree: no CDP round trip, so a click that navigates resolves before the navigation's events
  are applied. Every record it emits is first applied to its own `recording.State`; one the reducer refuses is not
  emitted but counted (and logged with the reason), so the output validates by construction and loss is visible.
- **`Recorder` (thin IO shell):** one goroutine owns the mirror and the segment state. Inputs arrive in a FIFO queue
  from the Chrome loop, stamped with their arrival time (the record `t`, monotonic since the first go-live). It
  makes its CDP calls synchronously through a `Caller` (the Chrome's `call`), in queue order, so ordering is the
  queue's (ARCH-ORDER). Inputs: `Live{tab, session}`, `Pause`, `End`, `FrameAttached{session, parent, frameID}`,
  `Detached{session}`, `Event{session, method, params}`.
- **Segments:** first go-live opens the sink (lazily: a cast never live leaves no directory) and emits `head`; then
  `seg` `live`; pause emits `stop pause` and `DOM.disable`s; go-live again emits `seg resume` if the top document's
  root `backendNodeId` is unchanged, else `seg navigate`; `DOM.documentUpdated` on the top session re-snapshots:
  same root → `kids`, new root → `seg navigate`; the loop's active-tab change while live → `seg tab`; `End` emits
  `stop end`. Each `seg` is followed by the top `doc`, then frame `doc`s, adopted sheets, then an observer resync
  (the observer re-reports every form value and scroll offset on request).
- **Frames:** iframes in the same process come inline (`contentDocument`) as `doc` records with `host`. Out-of-process
  frames are auto-attached sessions: the Chrome loop sets `Target.setAutoAttach` (iframes) on each page session and
  each frame session, sets them up like pages (binding, observer, released), and tells the recorder; while live the
  recorder enables DOM/CSS/Network on every session under the active page and emits the frame's `doc` hosted by the
  iframe element whose `frameId` matches.
- **CSS:** `CSS.styleSheetAdded` with an owner over CDP's 10k text cut → `CSS.getStyleSheetText` into the `<style>`'s
  text node (`text`); constructed sheets → `sheet` + `adopt`; `CSS.styleSheetChanged` on an owned sheet → its rule
  text read in the isolated world (`DOM.resolveNode` + `callFunctionOn`) → `sheet` with `owner`; `CSS.fontsUpdated`
  with a `src` → `font`.
- **Assets:** `Network.loadingFinished` for Image/Stylesheet/Font → `getResponseBody` → `assets.Store.Put` (#459's
  interface; nil store = no asset records) → the returned `asset` record.
- **Layout and frames (fidelity):** every 5 s while live, and at each `seg`: `Page.captureScreenshot` at scale 0.5 →
  `FrameWriter.PutFrame` (the DirWriter; never the Live), and `DOMSnapshot.captureSnapshot` **per session** (fixes
  #457's gap: the top session's snapshot leaves out out-of-process frames), frame boxes offset into the top page by
  their owner's box → one `layout` record.
- **Coordinates:** the observer reports frame-viewport coordinates; the recorder maps a subframe's into the top page
  by its owner iframe's content box (`DOM.getFrameOwner` + `DOM.getBoxModel` in the parent session, recursively), plus
  the top document's scroll.

### Observer (`recorder/observer.js`, isolated world `couchcast`, every frame)

Reports through the existing binding as `{t:"obs", items:[…]}`, synchronously in the event handler for clicks (the
#457 gap: the polled prototype lost the click that navigates), batched per animation frame otherwise. It reports
only while the recorder has switched it on (`__couchcastRec(true)`), so a page that isn't cast pays only idle
listeners:
- `click` (capture): path, client x/y — sent at once.
- `input`/`change` (capture) and a 250 ms scan for values set by script: `{path, v | checked | sel}`; a password input
  reports `{path, secret: bool}`, never its value.
- `scroll` (capture): element or document offsets.
- `selectionchange`, `focusin`/`focusout`.
- `pointerover` (capture) → hover path; `pointermove` → ptr, throttled to 250 ms; `pointerleave` of the document →
  hover 0.
- `resync()`: report every form control's state and every scrolled element (after each `seg`/`doc`).

Closed shadow roots: their contents are out of the observer's reach (#453); events retarget to the host, which is
what is recorded. Nothing runs in the page's main world (no measured gap needs it).

## Plug-in (M2): Chrome loop and Cast

- `LaunchOptions.Recorder *recorder.Recorder` (constructed by the caller with its sink). Chrome forwards inputs while
  the recorder exists; `finish` ends it (`End`, then waits for its goroutine) before the process wait, so the last
  record is written before `Done` (ARCH-FUNERAL for the goroutine).
- `CastConfig.Recording func(id string) (recording.Sink, recording.FrameWriter, error)` plus optional
  `NewLive func() (recording.Sink, recording.Live)` (#460's to supply) and `Assets` (#459's). Server.events streams
  `rec` events from the Live when one is configured; the stub `state` events stay for the indicator.
- couchcmd: recordings under `<store>/webcast/recordings/<id>/` (id = start time + random suffix).

## Lifecycle and envelope

- **Recording directory:** created at the first go-live by the recorder's sink; last needed by the operator; removed by
  the operator (no sweep in the MVP, as the format page says). Size: ≈1.4 MB/min gzipped of records on the busy stress
  page (#453) plus frames at ≤12 per minute × ~60–120 KB; measured on the live run and recorded in the Log.
- **Queue:** bounded at 100 000 queued events. Over it, DOM events of the live sessions are dropped and the recorder
  marks a resync: it re-snapshots the top document (a `kids` of its root) once it catches up, and logs the overflow. Control inputs are never dropped.
- **Goroutine:** owned by the Chrome; ends with it.

## Milestones

- [ ] M1 — format: `hover`, `ptr`, `input.secret` + password rules in types, reducer, validator, Canonical
  (secret only), format page Revisions; `ValidateDir` reports strays in `frames/`. Tests first.
- [ ] M2 — recorder: mirror (pure) + observer + Recorder shell + Chrome loop and Cast plug-in, unit tests against a
  stateful fake CDP page (a DOM tree that answers `getDocument` and emits mutation events), contract: every recorder
  output passes `ValidateDir`; a test that a click whose handler navigates is recorded before the `seg navigate`.
- [ ] M3 — live and fidelity: live headless checks (stress page served from testdata, the blog on `npm run dev`):
  validator clean, reducer rebuilds; a fidelity tool (`recorder/internal/fidelity`) that renders the reduced state at
  each frame's `t` in headless Chrome (records → static HTML, assets served through `Fetch` interception, scroll set
  by CDP) and pixel-diffs against `frames/`; compare with #453's own-on-CDP numbers (stress 0.3/1.8, blog
  via #457's fixtures); atlas.

## Out of scope / coordinated

- Replacing the #457 fixtures with real recordings (the project's integration guard) and retiring `fixturegen -raw`:
  touches every wave-1 component's tests at once, so it goes through the TL after this lands (ARCH-DRY: until then
  `cdpconv.go` and the mirror both map CDP to records; `cdpconv` is a frozen reference over the prototype's log).
- Layout on note creation (#461 asks for it through the recorder later).

## Room for remote driving (#466/#467; TL, 10-12)

- Observer-made records may later carry an optional `by` (viewer id; absent = operator). Nothing here refuses it.
- The recorder's goroutine is the record stream's single writer. #467's control records and injections enter its
  queue as inputs, ordered with the DOM events (told the TL; the Chrome loop forwards, it doesn't write records).
- The id map is exposed both ways: `Recorder.IDOf(session, backendNodeId)` and `Recorder.NodeOf(id)`, answered on
  the recorder goroutine (a request input with a reply), backed by the pure mirror's own accessors.
