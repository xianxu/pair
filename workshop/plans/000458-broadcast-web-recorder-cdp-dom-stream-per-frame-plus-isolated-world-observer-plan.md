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

**The seam, stated:** the recorder is part of the real Browser (`*Chrome`), on the CDP side of the seam, as #457's
format page already places it ("adds `LaunchOptions.Sink` to `webcast.Chrome`; the Chrome loop … feeds the active
tab's CDP events to the recorder"). `Browser` is unchanged and still carries no CDP type; `browser.go`'s comment
("the recorder uses only this") is out of date and is corrected. Couch's UI, the Cast and the Server still see only
`Browser`; what crosses into the recorder is the Chrome loop's CDP event stream and its call function, both inside
package `webcast`'s real implementation. A `FakeBrowser` records nothing; recording behaviour is tested on the
recorder against its fake CDP page (M2) and live (M3), and the Cast wiring with `recording.FakeRecorder`.

**Wiring:**
- `recorder.Output` is what a recorder writes: `Open() (recording.Sink, FrameWriter, error)`, called at the first
  go-live. `FrameWriter` is a consumer-side interface **in package recorder** (`PutFrame(*recording.Frame, []byte)
  error`), satisfied by `*recording.DirWriter`; nothing is added to #457's contract package.
- `CastConfig.Launch` becomes `func(ctx, out recorder.Output) (Browser, error)`; `StartCast` builds the Output from
  `CastConfig.RecordingsDir` (a `DirWriter` at `<dir>/<id>`, `Tee`d with the Live's sink when `CastConfig.NewLive` is
  set: #460 supplies the real Live) and `CastConfig.Assets` (#459). couchcmd passes it to `LaunchOptions.Output`;
  `Launch` builds the recorder when it is set. A FakeBrowser ignores it.
- Server.events streams `rec` events from the Live when one is configured; the stub `state` events stay for the
  indicator.
- Chrome's `finish` ends the recorder (`End`, then waits for its goroutine) before the process wait, so the last
  record is written before `Done` (ARCH-FUNERAL for the goroutine); the Cast closes the sinks after `Done`.
- couchcmd: recordings under `<store>/webcast/recordings/<id>/` (id = start time + random suffix).

## Lifecycle and envelope

- **Recording directory:** created at the first go-live by the recorder's sink; last needed by the operator; removed by
  the operator (no sweep in the MVP, as the format page says). Size: ≈1.4 MB/min gzipped of records on the busy stress
  page (#453) plus frames at ≤12 per minute × ~60–120 KB; measured on the live run and recorded in the Log.
- **Latency (PQ-5):** the recorder goroutine makes only the calls DOM correctness needs in order (`getDocument`,
  `getStyleSheetText`, the rule read, `getAttributes`). Asset bodies and screenshots/layout run on two side workers
  (one goroutine each, one call at a time, bounded queues of 256 that drop with a count); their results come back as
  queue inputs, so the goroutine stays the single writer. Budget: records emitted ≤250 ms p95 after their CDP event
  on the stress page, measured in M3 (the queue stamp vs the emission time).
- **Id map (PQ-6):** per session, `backendNodeId` → id for live nodes, plus a graveyard of the last 50 000 removed
  nodes so a move (remove, then insert) keeps its id; a session's map is dropped when its document is replaced
  (`seg navigate`/`tab`, a frame's new `doc`); the counter never resets, so ids are never reused. Bound: live nodes +
  50 000 entries per session.
- **Queue:** bounded at 100 000 queued events. Over it, DOM events of the live sessions are dropped and the recorder
  marks a resync: it re-snapshots the top document (a `kids` of its root) once it catches up, and logs the overflow. Control inputs are never dropped.
- **Goroutine:** owned by the Chrome; ends with it.

## Ordering (PQ-3)

CDP node ids belong to a snapshot: after `DOM.getDocument` replies, events Chrome sent before the reply describe the
old tree, which the snapshot already includes. **Discriminator:** `cdp.Conn` numbers every inbound message in read
order; `Event.Seq` carries it, and `Conn.CallSeq` returns the reply's number. A snapshot of session S at reply number
R drops every queued event of S (and of the in-process frames it pierces) with `Seq < R` as already reflected. Other
cases, each with a sequence test that injects the interleaving through the fake CDP:
- an event naming a CDP node the mirror doesn't have (a missed `setChildNodes`, a race we didn't foresee): counted,
  and a resync (re-snapshot as `kids` of the root) is scheduled; never applied blindly;
- the active page's session detaches mid-segment (tab closed): the Chrome loop's tracker moves the active tab and the
  recorder gets `Live{next tab}` → `seg tab`; with no page left, `stop end`;
- a frame session detaches mid-segment: its mirror is dropped; the frame's last document stays in the recording
  until its iframe element is removed or receives a new `doc` (a cross-origin navigation attaches a new session);
- a document update arriving while a resync is pending: one snapshot serves both.

## Tests per risky function (PQ-4)

- **mirror translate:** the fake CDP page is a stateful DOM (insert, remove, move, attribute, text, shadow attach,
  set children) that emits the events Chrome would. Property test: seeded random mutation sequences (moves included)
  → `Reduce(output)` canonical DOM equals the fake's own serialization, and `ValidateDir` is clean; a reverted mirror
  rule fails it.
- **path resolver:** `go test -fuzz` target: arbitrary paths (negative, past the end, shadow steps on closed roots,
  stale after a mutation) never panic and resolve to nothing or to exactly the node the fake says.
- **password rules:** property test: random sequences that type values, flip `type` text↔password, set and remove
  `value` attributes, and insert password inputs with values → no record of the recorder's output contains a planted
  password string, and the reducer refuses every hand-made record that would carry one. Accepted (PQ-7): a value
  typed while the control was `type=text` was shown on screen and is recorded; it isn't erased when the type flips.
- **segments:** go-live, pause, resume same/new document, tab switch, end, as a table of input sequences → the
  expected `seg`/`stop` reasons, validated.

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

- Observer-made records may later carry an optional `by` (viewer id; absent = operator) as an embedded
  `recording.Source` (#466 lands the type). `Hover` and `Ptr` are plain `Header` + fields structs, so embedding it
  is a no-op for this work.
- The recorder's goroutine is the record stream's single writer. #467's control records and injections enter its
  queue as inputs, ordered with the DOM events (told the TL; the Chrome loop forwards, it doesn't write records).
- The id map is exposed both ways: `Recorder.IDOf(session, backendNodeId)` and `Recorder.NodeOf(id)`, answered on
  the recorder goroutine (a request input with a reply), backed by the pure mirror's own accessors.

## Revisions

- 2026-10-12 — plan-quality round 1: the seam stated and wired (PQ-1); `FrameWriter` lives in package recorder (PQ-2);
  ordering by CDP message sequence (PQ-3); adversarial tests per risky function (PQ-4); latency budget and side
  workers (PQ-5); id-map bound (PQ-6); type-flip leak accepted (PQ-7). TL: `Source` embedded later in hover/ptr.
- 2026-10-12 — M2 as built (boundary review BR-18): `CastConfig.Launch` is `func(ctx, *RecordOptions)`, carrying
  `recorder.Output`, which returns `Opened{Sink, Frames, Assets}` (the store comes with the recording, since #459's
  persists through its DirWriter): `CastConfig.NewAssets func(AssetWriter) Store`, not `CastConfig.Assets`. The
  viewers' hub is `CastConfig.NewLive` (per cast). A cast tab that vanishes while live is neither paused nor
  ended at once: the next page becomes the cast tab (`seg tab`), or Chrome ends (`stop end`). Snapshots go through a
  per-session dirty set, so a resync, a document update and a desync after one input cost one snapshot. Problems
  reach the console notice (`CastConfig.Logf`); per-item losses are counted in `Stats` and summarized at the end.
- 2026-10-12 — M2 review round 2: the queue has two bounds of `MaxQueue` (100 000) each: DOM events (bursty; dropped
  ones are resynced from a snapshot) and every other CDP event (contexts, observer reports, CSS, Network: page-load
  and human rates, reached only after a stall of minutes; dropped ones are counted and reported). The graveyard
  takes only ids whose nodes are gone after the update is applied (BR-19). One resync notice per recording.
