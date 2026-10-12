# Couch broadcast-web: the recording format (#457)

The format is the product of couch-broadcast-web (project: ops `teams/devinfra/projects/couch-broadcast-web/project.md`).
A **recording** is the DOM of a cast Chrome tab over time: a full snapshot when a segment starts, then every change,
plus what the DOM alone doesn't hold (form values, scroll, CSSOM, fonts, layout boxes), plus notes and pointer strokes.
Two kinds of reader use it. The **viewer** rebuilds the page from it in a script-free frame, live or replayed. The
**agent** asks what the page was at time t, what happened between two moments, and what each note points at.

The Go package `cmd/internal/webcast/recording` is the executable spec. Its types define the records, its reducer
(`Reduce`) is the meaning of a recording, and its validator (`Validate`) is the conformance check. Where this page
and the package disagree, the package is right and this page has a bug.

## One stream, live and on disk

Every record is one JSON object on one line, with the same envelope everywhere:

```json
{"t":1530,"k":"attr","id":412,"name":"class","v":"btn active"}
```

- `t` is milliseconds since the recording started, an integer. Records are in `t` order, never decreasing; records
  with the same `t` are in the order they happened. **The order of the records is authoritative.** `t` places a
  record on the timeline, and it does not reorder anything.
- `k` is the kind. Every other field belongs to the kind.

The live stream (recorder → viewers) and the recording on disk carry **the same records**. A live viewer receives
them one by one as SSE events of type `rec`, each event's data being one record line. On disk the same lines are split by
family into files. Splitting and merging are pure functions of `k`:

```
<recording>/
  events.jsonl         head, seg, stop, doc, DOM changes, observer state, context, control, assets, layout
  notes.jsonl          note, note-state
  pointer.jsonl        stroke
  assets/<hash>        captured bodies (images resampled; CSS and fonts exact), named by content hash
  frames/index.jsonl   frame: the sparse low-res screenshots for the fidelity check (not in the live stream)
  frames/<n>.png
```

Reading a recording merges the three record files into one stream by `t`. At equal `t`, events come first, then
notes, then pointer strokes. `frames/` is never merged in: screenshots are for the fidelity check, and pixels never go to
viewers (Principle 1).

## Versioning

The first record of `events.jsonl`, and of every live stream, is `head`:

```json
{"t":0,"k":"head","format":"couch-webcast","v":1,"id":"20261011T231955-7f3a","started":"2026-10-11T23:19:55.120-07:00"}
```

- **`v` is the format's major version.** A reader refuses a major version it doesn't know.
- **Additions keep the version.** Within a major version, new kinds and new optional fields may appear. A reader
  ignores kinds it doesn't know. The validator reports them as warnings, not errors. A change in the meaning of an
  existing field is a new major version.
- **Encoding is separate from shape.** v1 is plain JSON Lines. #453 measured our verbose prototype stream at
  1.15–2.05× rrweb's gzipped volume. A compact encoding (field tables, id deltas, gzip per file or per SSE batch)
  comes later as `head.enc`, without changing these logical records.

## Time, segments, documents

A cast is a sequence of **live segments** with gaps where it was paused (project: operator flow, step 7).

| Kind | Fields | Meaning |
|---|---|---|
| `seg` | `seg` (1, 2, …), `why`, `tab` `{id,url,title}`, `vp` `{w,h,dpr}` | A segment starts. **All DOM state is dropped**: documents, nodes, sheets, form values, scroll, selection, focus, layout. The asset table and the notes are kept. `why` is `live` (first go-live), `resume` (going live again on **the same document** as before the pause), `navigate` (a new top document, including one loaded while paused), or `tab` (the operator switched tabs). The current document began at the latest segment that isn't a `resume`. |
| `doc` | `doc`, `url`, `tree`, optional `host` | A full snapshot of one document. `tree` is a `doc` node, and `doc` is its id. Without `host` it is the top document; exactly one comes right after each `seg`, before any change. With `host` (an `iframe` element's id) it is that frame's document, replacing whatever the frame showed before. Cross-origin frames come the same way: the recorder stitches them by their owner element. |
| `stop` | `why`: `pause` or `end` | The segment ends. The state is frozen as it was. `end` is final: no `seg` or `ctl` follows it (#466). |

A cross-document navigation of the top document is a new segment (`why: navigate`): "each new document starts with a
full snapshot" (project, step 4). A same-document navigation (`pushState`) is a `nav` record within the segment.

## Node ids

Each node in a recording has an integer `id`, assigned by the recorder. CDP's own node ids are scoped to a CDP session and are reassigned
on each `DOM.getDocument`, so they never reach the format.

- **One namespace per recording.** Nodes of every document and frame, shadow roots, template contents and stylesheets
  draw ids from one counter, so an id alone names a thing. No change record needs a document or frame field.
- **An id names one node for the node's whole life.** Within a document the recorder keeps ids through CDP's
  `backendNodeId`, so a node keeps its id across a pause and resume of the same document. A note made before a pause
  still resolves after it. A node that is removed and inserted again (a move) keeps its id. The validator checks the
  cheap part of this: an id never changes type or tag name.
- **Ids are never reused** for a different node. `0` means "none".

## Nodes

A node is a JSON object with an `id` and a type `y`:

| `y` | Fields | Notes |
|---|---|---|
| `doc` | `c` | A document: its doctype and its root element. |
| `doctype` | `n` | `{"id":2,"y":"doctype","n":"html"}` |
| `el` | `n`, optional `ns`, `a`, `c`, `sr`, `tc` | `n` is the tag name in lower case for HTML. `ns` is `svg` or `math` when not HTML. `a` maps attribute names to values (an object; attribute order carries no meaning). `sr` is the shadow root, `tc` a template's content. |
| `text` | `v` | Full text. CDP truncates node values at 10,000 characters (#453); the recorder must read the full text before emitting it. |
| `comment` | `v` | |
| `shadow` | `mode` (`open`/`closed`), `c` | A shadow root, under its host's `sr`. Closed roots are recorded (CDP reaches them). The browser's own user-agent shadow trees (input editors, `<details>` slots) are **not** page content and never appear. |
| `frag` | `c` | A `<template>`'s content, under the element's `tc`. |

The recording keeps the page's DOM as it was, scripts and event-handler attributes included: an agent may need them.
It is the viewer that neutralizes them (Principle 3). An `iframe` element's content is never inline. It arrives as
a `doc` record with `host` set to the iframe's id.

```json
{"id":10,"y":"el","n":"button","a":{"class":"btn","onclick":"go()"},"c":[{"id":11,"y":"text","v":"Save"}]}
```

## DOM changes

| Kind | Fields | Meaning |
|---|---|---|
| `ins` | `p`, `after`, `node` | Insert `node` (a whole subtree) under parent `p`, right after sibling `after`; `after: 0` inserts first. `p` may be an element, a document, a shadow root or a template's `frag`. |
| `rm` | `id` | Remove the node and its subtree. |
| `kids` | `p`, `c` | Replace all of `p`'s children with `c`. This is the cheap form of a re-render that rebuilds a list. |
| `attr` | `id`, `name`, `v` | Set an attribute; `"v": null` removes it. Style changes made through `el.style` arrive here as the `style` attribute. |
| `text` | `id`, `v` | A text or comment node's new content. |
| `shadow` | `host`, `root` | A shadow root (a `shadow` node) was attached to `host` after its snapshot. |

## Styles and fonts

| Kind | Fields | Meaning |
|---|---|---|
| `sheet` | `id`, `text`, optional `owner` | A stylesheet's complete current text. With `owner` (a `style` or `link` element), the element's CSSOM no longer matches its DOM text (rules added by `insertRule`, or a `<link>` whose body was captured), and the viewer uses this text for that element. Without it, the sheet is a constructed sheet, adopted by `adopt`. Each `sheet` record replaces the previous text. `url()`s in `text` are absolute. |
| `adopt` | `id`, `sheets` | The `adoptedStyleSheets` of a document or shadow root (`id`), in order. |
| `font` | `family`, `src`, optional `style`, `weight`, `stretch`, `range` | A font loaded through the FontFace API, which leaves no trace in the DOM. `src` is an absolute URL, resolved through the asset table. |

## Assets

Images, stylesheets and fonts are captured during recording and served by Couch. Viewers never reach the product's
servers (Principle 4).

| Kind | Fields | Meaning |
|---|---|---|
| `asset` | `url`, `hash`, `mime`, `size`, optional `orig` `{size,w,h}` | The absolute `url` is served by the body stored under `hash`, which is the lowercase hex SHA-256 of the stored bytes. `size` is the stored size. `orig` is set when an image was resampled: the original's size and pixel dimensions. |

- DOM attributes keep the page's own URLs, because an agent wants to see the real ones. The viewer resolves an attribute value
  (`src`, each `srcset` candidate, `href` of a stylesheet link, `poster`) against its document's URL, looks the absolute URL up in the asset table, and
  loads `assets/<hash>` through the proxy. A URL with no entry loads nothing. It is never fetched from the origin.
- CSS bodies (`text/css` assets) and `sheet` text have absolute URLs: in `url()`, `@import` strings and plain strings
  inside `image-set()`. The asset store makes a body's URLs absolute against the stylesheet's own URL when it is
  stored (`assets.Recorded.Put`), so a body shared by URLs in different directories means one thing; the recorder
  does the same for `sheet` text. CSS is otherwise exact. At serve time the proxy (`assets.Proxy`) points each URL
  that has an asset at `<hash>` (relative to `assets/`) and leaves the rest absolute, for the viewer to resolve when
  the asset arrives (#459).
- Images over the budget (`assets.Policy.MaxPixels`, about one 1920×1080 frame) are resampled, aspect kept: JPEG when
  opaque, PNG with transparency, so the asset's `mime` can differ from the page's. SVG, GIF (mostly animations) and formats
  the store can't decode (AVIF) are kept exact. A header claiming over 64 MP is refused unread.
- Assets outlive segments: the table is per recording, and a later `asset` record for the same URL rebinds it.

## Observer state

These records cover state that CDP's DOM domain doesn't report. The recorder's isolated-world observer reads it (#453).
Each may carry `by` (see Attribution and control).

| Kind | Fields | Meaning |
|---|---|---|
| `input` | `id`, one or more of `v` (value), `checked`, `sel` (selected option indexes), `secret` | A form control's current state: a property, not an attribute. It is set by the user or by script; typing is recorded as values, not keystrokes. `secret: true` (never with `v`) says a password control holds a value that is not recorded. |
| `scroll` | `id`, `x`, `y` | Scroll offset of an element, or of a document's viewport when `id` is a document. |
| `selection` | `anchor` `[id,offset]`, `focus` `[id,offset]` | The text selection. A record with neither field clears it. |

## Context

| Kind | Fields | Meaning |
|---|---|---|
| `nav` | `url` | A same-document URL change (`pushState`, hash). |
| `title` | `v` | The top document's title changed. |
| `vp` | `w`, `h`, `dpr` | The viewport changed mid-segment (CSS px; device pixel ratio). |
| `click` | `id`, `x`, `y` | A click on element `id`, at page coordinates. |
| `focus` | `id` | Focus moved to `id`; `0` is none. |
| `hover` | `id` | The element under the operator's pointer; `0` is none. A hover menu or tooltip is CSS `:hover` state the DOM doesn't hold, so a viewer shows it from this. Cleared when the element goes. |
| `ptr` | `x`, `y` | A sampled operator pointer position (on movement, at most a few per second). |
| `layout` | `boxes` `{id: [x,y,w,h]}` | Layout boxes of rendered nodes, replacing the previous set. The recorder captures them (`DOMSnapshot`) when a note is made and with each `frame`, not continuously. The notes model needs them for focused context and crops; a box is as of its record's `t`. |

**Passwords are never recorded.** A password input's value appears neither in an `input` record (`secret` instead)
nor as its `value` attribute (the recorder drops it). The reducer refuses a non-empty `v` on an `input type=password`,
`secret` with `v`, and any record that would leave a password input with a non-empty `value` attribute; so the
validator catches a recorder that leaks one. A value typed while the control was `type=text` was on screen and stays
recorded when a script flips it to `password`.

`hover` and `ptr` are tracked by the reducer (`State.Hover`, `State.Ptr`, reset by `seg`) but are not in
`State.Canonical()`: they are the operator's context, not the page's. `query.UserAction` (#462) doesn't list them;
whether a note's "what the user did just before" should include hovering is #462's call (raised with the TL by #458). Like `click` and `focus`, they may carry `by`
(see Attribution and control).

**Coordinates:** every coordinate in the format is in **CSS pixels of the top document's page** (scroll-independent). This
covers `click`, layout boxes, note boxes, pointer strokes and crops. Something in an iframe is mapped into the
top page by the recorder. The viewport position at time t is the top document's `scroll`.

## Notes and pointer

| Kind | File | Fields | Meaning |
|---|---|---|---|
| `note` | notes | `id` (string), `author` `{id,name,color}`, `text`, `at` `{node,sel,text,box}`, optional `region` `[x,y,w,h]` | A note anchored on an element at its `t`. `at.node` is the element's id. `at.sel` is a stable CSS selector. `at.text` is the element's text, trimmed to 200 characters. `at.box` is its box. `region` is set for a dragged region. |
| `note-state` | notes | `id`, `state`: `open` or `resolved` | A note was resolved or reopened. |
| `stroke` | pointer | `id`, `author`, `color`, `pts` `[[x,y,dt],…]` | One pointer stroke; `dt` is ms after the record's `t`. |
| `frame` | frames/index | `file`, `w`, `h`, `scale` | A screenshot of the real tab at `t` (the fidelity check), `w`×`h` image pixels; `scale` is image pixels per CSS pixel, the one source of that mapping for every consumer: 0.5 (the viewport at half its CSS size) whatever the device pixel ratio, except under a ratio of 0.5, where a screenshot isn't upsampled and `scale` is the ratio. `t` is the capture's midpoint. Not in the live stream. |

Notes stay in the recording after a reload. That live notes clear on reload (project, Direction) is the notes model's
rule (#461): the notes "on" the current document are those since the document began (`State.DocSeg`), so a pause and
resume keeps them and a navigation clears them. The shape is meant to be
shared with the terminal notes of pair#455. `author` and `text` stay the same; `at` is where a DOM note and a grid note differ.

## The notes model (#461)

`notes.Structural` is the real `notes.Model` (`cmd/internal/webcast/notes`), pure over the reducer's state at a
moment. It runs `notes.RunModelContract` like the fake.

**A note** is `recording.Note`: `id`, `author` `{id,name,color}`, `text`, `t`, and the anchor `at`. A terminal note
(pair#455) keeps `author`, `text` and `t` and replaces `at` with a grid position, so both submissions share
everything but the anchor.

**The anchor** `at` is `{node, sel, text, box}`:
- `sel` is a stable selector in a small CSS subset, made and matched in one file (`selector.go`): `#id` for an id
  that is unique and doesn't look generated (no `:`, no run of 4+ digits), `[data-testid="…"]` (also `data-test`,
  `data-cy`), else `tag:nth-of-type(k)` steps joined by ` > ` from the nearest such ancestor or the tree's root
  element. Inside a shadow root, an iframe's document or a template's content it is the host's selector, ` >>> `,
  then the path inside (our deep combinator, not CSS). A test-attribute value with a quote or `>` isn't used, and
  `Anchor` drops any selector that doesn't find its own element, so a stored selector always round-trips.
- `Resolve` keeps the node while it lives (ids are never reused); after a re-render it takes the selector's match
  whose text is the anchor's, else the only match. Several matches with nothing to tell them apart don't resolve.
  A selector outside the subset matches nothing, and such a note resolves by its node id alone.
- `box` is taken as given when the note carries one (a viewer computes it at click time); otherwise the latest
  `layout` box is used.

**The landmark** is the nearest meaningful container at or above the element, through shadow hosts and frame owners
and below the top document's `body`. Meaningful means a sectioning or grouping tag (`header nav main aside footer form
section article dialog fieldset figure table tr ul ol dl li details menu search`), an ARIA role of the same kinds,
`aria-modal="true"`, or a layout box recognized by class on an element with element children: a `card`, `panel`,
`modal`, `dialog`, `popover`, `toolbar`, `navbar`, `sidebar`, `drawer` or `tile` class, or Tailwind's `rounded*` with
`border` or `shadow*`. With none, the landmark is the parent element (at most `body`), used for text but not for
snapping.

**The context** (`notes.Context`) is `{Node, Landmark, Text, Box}`: the resolved element, its landmark, the landmark's
whitespace-collapsed text in a 500-character window centred on the element's own text, and the landmark's box. The
full DOM at that moment stays available through the agent queries (#462).

**Crops**, for a submission's anchors, in CSS px of the top page:
1. each anchor's box padded by a margin (48 px by default);
2. snapped to its meaningful landmark per axis: always across (a header or nav bar goes full width), and down only
   when the landmark is at most half the viewport tall; otherwise the height stays the note plus its margin (the rule
   #455 uses for a pane);
3. clustered: crops at most a margin apart merge, until none do, so far-apart notes get a crop each;
4. clipped to the page (the root element's box), widened to hold every note's own box;
5. offered to `notes.Hints` (per app or URL), whose crop replaces the default when it returns true.

Region notes (`region`) are not in `Crops` yet: `Crops` takes anchors, which carry no region.

## Attribution and control (#466)

A viewer the operator grants control to uses the cast tab through the cast (Remote driving, below). The recording
says who did what.

- **`by`:** an interaction record may carry `by`, a viewer id: the viewer whose injected input caused it. These are
  `click`, `input`, `focus`, `scroll` and `selection`, plus #458's `hover` and `ptr`. In Go it is the embedded
  `recording.Source`, and a new interaction kind embeds it too. If there is no `by`, the operator did it (or the page's
  own script). The recorder stamps `by` on what its observer reports while it dispatches that viewer's input. The
  operator's own input stays unattributed even while a viewer drives. **A `by` that isn't the current driver is an
  error.**
- **`ctl`:** a control change, in `events.jsonl`, valid anywhere after `head` (paused too):

  ```json
  {"t":7000,"k":"ctl","op":"grant","who":{"id":"v2","name":"Bo"},"driver":{"id":"v2","name":"Bo"},"pending":[{"id":"v3","name":"Cy"}]}
  ```

  | `op` | Allowed when | After it |
  |---|---|---|
  | `request` | `who` neither drives nor has asked | `who` is pending |
  | `grant` | `who` is pending | `who` drives; a previous driver doesn't |
  | `deny` | `who` is pending | `who` isn't pending |
  | `revoke` | `who` drives | nobody drives (the operator) |
  | `release` | `who` drives, or is pending (withdraws) | `who` neither drives nor asks |
  | `leave` | `who` drives or is pending | the same; the viewer's seat expired |

  `driver` (absent: the operator) and `pending` are the **control state after** the change. The reducer recomputes
  them with `recording.NextControl`, the one transition function, and refuses a record that states anything else. A
  record must change something. **The first `ctl` a reader sees is its baseline:** a late joiner's catch-up starts at
  the latest one, so a first `ctl` is taken as stated if it is consistent in itself (`who` where its op puts it, no
  viewer twice). `Validate` reads whole recordings and holds the first one to the rule too.
- **Lifetime:** control is per recording, not per segment. It survives pause, resume and navigation (a driver's click
  on a link keeps them driving on the new document). It ends at `stop` `end`, which is final: no `seg` or `ctl` follows it. `State.Control` is the state at t.

## Joining mid-cast

A viewer that joins late receives the **catch-up**, followed by the live records. The catch-up is:
- `head`;
- every `asset` record so far;
- the `seg` the current document began in, as a bare record, so the document's notes keep their segment;
- the notes, note states and strokes since then;
- the latest `ctl` before the latest `seg`, which carries the whole control state;
- everything from the latest `seg` on, except `hover` and `ptr`: of those, only the latest of each, in place (they are the
  operator's pointer now, state rather than history; #458). While paused, that last part is the segment through its
  `stop`. Reduced, the catch-up gives the same DOM, assets, `DocSeg`, current-document notes, control, hover and
  pointer as the whole recording. This is tested at every cut point, and at every pinned
time of every fixture. `recording.CatchUp` is this rule; the
live fan-out (#460, `recording.Hub`) serves it. The hub keeps only `CatchUp(log)` at each segment, so what it holds for late
joiners is bounded by one segment.

## Reading rules (the reducer)

`recording.Reduce(records, t)` gives the state at time t: documents and node trees, sheets and adoptions, fonts,
the asset table, form values, scroll, selection, focus, layout, URL, title, viewport, the segment, live or paused,
the notes, and who drives (`Control`). A record that references a node that doesn't exist is an **error**, not something to skip: the
recorder must not emit one, and the validator reports it with its line. `State.Canonical()` is a deterministic
serialization of the state. The viewer's conformance check compares it with what the viewer rebuilt.

## Lifecycle and size (ARCH-FUNERAL, ARCH-CONSTRAINTS)

- **A recording** is created by the recorder at go-live (#458), under Couch's store. It is needed until the operator is
  done with it. It is kept until the operator deletes its directory: there is no automatic sweep in the MVP. A recording's size is bounded per
  minute: about 1.4 MB/min gzipped on #453's busy stress page, before compact encoding. Assets are deduplicated by hash, and images are
  resampled to a resolution budget (#459). A retention sweep comes with the first upload or listing feature. It is not
  part of the format.
- **The reducer** holds one recording's state in memory, and `Reduce(records, t)` replays from the first record. Seeking from
  the latest `seg` before t (an index of segment offsets) is a later optimization. The index would be derived from the records,
  never stored as a second truth.

## Components and seams (wave 1 starts here)

Each component is behind an interface with a stateful fake and a contract test suite. Run the suite against the fake and against the real
implementation, so the two can't drift. Suites take `recording.TB` (a `*testing.T` satisfies it), so product packages
don't import `testing`. Fixtures are in `cmd/internal/webcast/recording/testdata/fixtures/` and are
named by `recording.Fixture(name)`. A format change found in wave 1 goes back through #457's contract (the TL coordinates
it), never a local fork.

| Issue | Component | Interface | Fake | Contract suite |
|---|---|---|---|---|
| pair#458 | recorder | writes a `recording.Sink` (`DirWriter` for disk, `Tee` to also feed `Live`) | `recording.FakeRecorder` (plays a fixture into a sink, instantly or in real time) | `recording.ValidateDir` on its output; fidelity vs `frames/` |
| pair#459 | asset store and proxy | `assets.Store` (`Put`, `Get`, `Lookup`; `Policy`) | `assets.MemStore` | `assets.RunStoreContract` |
| pair#460 | live viewer | `recording.Live` (fan-out with catch-up; `Ended(recs)` says whether a closed subscription was the fan-out's end or a drop); the JS viewer contract below | `recording.Hub`: the production fan-out is its own double (bounded by one segment; #458 constructs it next to its `Tee`) | `recording.RunLiveContract`; the fixtures' canonical pins |
| pair#461 | notes model | `notes.Model` (`Anchor`, `Resolve`, `Context`, `Crops`, `Live`; `notes.Hints`); real: `notes.Structural` | `notes.FakeModel` | `notes.RunModelContract` |
| pair#462 | agent queries | `query.Surface` (`DOMAt`, `Events`, `Notes`); real: `query.Querier`, CLI `couch --broadcast-web-query` (below) | `query.FakeSurface` | `query.RunSurfaceContract` |
| pair#467 | input injection | `drive.Injector` (`Inject(ctx, by, Batch)`) | `drive.FakeInjector` | `drive.RunInjectorContract` |
| pair#467 | driving control and the reverse channel | `drive.Controller` (seats, request/grant/deny/revoke/release, input) | `drive.FakeController` | `drive.RunControllerContract` |

Hand-written recordings for tests: `recording.Builder` (ids from one counter, `Page` for a segment and its document),
`recording.ContractRecording`, `query.ContractRecording`, `recording.DrivenRecording` (viewers driving).

**Records, bodies and frames.** A `Sink` carries records only, and `Tee` fans them out to the recording directory and
the live fan-out. An asset body is stored once: the asset store (#459) decides what to keep and persists it through
the recording's `recording.AssetWriter` (its `DirWriter`, the one writer of `assets/<hash>`, which writes atomically). Only the
asset record goes through the Sink. Screenshots go to `DirWriter.PutFrame`. A `Live` refuses frames
(`ErrNotLive`), because pixels never reach viewers; `RunLiveContract` checks this.

**Seam into `webcast.Cast` (#454).** Today `webcast.Server.events` streams the Browser's `state` (the stub). The recorder (#458)
adds `LaunchOptions.Sink recording.Sink` to `webcast.Chrome`. The Chrome loop already owns every page session and the
live phase, so it feeds the active tab's CDP events to the recorder while `PhaseLive`. Go-live emits `seg`, pause emits `stop`,
and a tab switch from `Tracker.Active` emits a new `seg`. `CastConfig` gains the `recording.Live` that fans the
same records out, and `Server.events` sends them as `rec` events. The route, the token and the shared `broadcast.Host` stay.

**The JS viewer contract (#460).** The viewer applies records to a sandboxed frame without `allow-scripts`
(`webcast/web/`: `model.js` applies, `render.js` draws). On `window.couchViewer`:
- `resolveAsset(absURL)` returns the proxy URL for an asset (`<cast>/assets/<hash>`), or nothing;
- `canonical()` exports the rebuilt DOM in `State.Canonical()`'s shape (tree and ids from the drawn DOM, values from
  the model), and `ready()` resolves once its stylesheets, fonts and images have loaded;
- `boxOf(id)` is a node's box in top-page CSS px from the viewer's own unscaled layout. A note made in the viewer
  carries this box from click time.
- `nodeAt(pageX, pageY)` is the id of the innermost drawn element at a top-page point (into frames and shadow roots):
  the target of a driver's act (#466). `toPage(clientX, clientY)` maps a point in the viewer's window to top-page CSS px.
- `onRecord(fn)` sees each accepted record and the model, so another component (#467's control state) can follow the
  stream without touching the apply path.
The viewer rewrites `url()`s and `image-set()` strings in streamed CSS (`sheet` records, `<style>` text, `style=`)
through `resolveAsset`. The proxy rewrites only the CSS bodies it serves. Both find the same URL spans:
`tests/webcast/css_urls.json` is the shared corpus (`css.js` and `assets.RewriteCSS`).

Conformance means that, for every fixture at each pinned time (its `frame` times), `canonical()` equals the reducer's
`State.Canonical()`, and the render is within tolerance of `frames/<n>.png`. The fixture pins each expected state by
hash (`canonical.sha256`). The expected value itself is printed on demand
(`fixturegen -canonical <fixture> -at <t>`), so the reducer stays its only source.

## Fixtures

The fixtures are in `cmd/internal/webcast/recording/testdata/fixtures/`; their README says how to regenerate them, and
what they leave out. The `rec-*` ones are the real recorder's (#458) own output (#472): `TestRecordFixtures` records
#453's pages in its scenarios, and `fixturegen -recorded` adds the notes and pins, converting nothing. Every
component's tests run on both sets; #453's prototype fixtures retire once the components built on them land (#474).
The three prototype ones were recorded by #453's prototype recorder (branch `453-recorder-prototypes`,
mode B, headless Chrome 154). For #457 the recorder was patched to add layout boxes, click and focus capture, and asset
bodies. `recording/internal/fixturegen` converted them to this format. The converter applies everything it emits to a
reducer, so a fixture validates by construction. It counts every record the reducer refuses (none in these runs). A few
notes and strokes are placed on known elements. `TestFixtureCoverage` holds each fixture to the cases below, so a
regenerated one can't silently lose a case.

| Fixture | Page | Covers |
|---|---|---|
| `stress` (7 s, 1.5 MB) | #453's stress page | Busy re-renders (3,000 inserts, 1,000 `kids`), a cross-origin iframe, open and closed shadow roots, `insertRule` (owned sheets), `adoptedStyleSheets`, checkbox and select state set by script, inner scroll, SVG, `<template>`, a FontFace font (its file is left out: a macOS system font) |
| `app` (22 s, 0.5 MB) | #453's React + Tailwind app | Typing, clicks, focus, a modal, Tailwind's runtime `<style>` over CDP's 10k text cut, and a cross-document navigation (a second segment). A note is resolved. |
| `blog` (12 s, 2 MB) | The operator's blog post `ariadne-1-building-blocks-of-ai-coding` (Astro + Tailwind, `npm run dev`) | Images, the Inter woff2 font, stylesheets, page scrolling, layout, and the giscus cross-origin iframe |
| `segments` (hand-made) | `fixturegen -segments` | Pause and resume with ids kept, an edit while paused, a tab switch, a same-document navigation; notes span the resume and clear on the switch |
| `rec-stress` (5 s, 0.8 MB) | #453's stress page from `testdata/stress`, by the recorder | As `stress`, but no FontFace font (`testdata/stress` leaves the file out, so it never loads) |
| `rec-app` (22 s, 0.5 MB) | #453's app scenario, by the recorder | As `app`, plus `selection` records |
| `rec-blog` (12 s, 1.9 MB) | The blog post's `browse` scenario, by the recorder | As `blog` |
| `rec-segments` (10 s, 0.3 MB) | #453's reports page, by the recorder | A same-document navigation, a pause with an edit, a resume, a new tab (the cast follows it, at that tab's own viewport), and a cross-document navigation there |
| `driven` (hand-made) | `fixturegen -driven` (`recording.DrivenRecording`) | Viewers request and are granted control. A driver's attributed clicks, typing and scroll, interleaved with the operator's own click. Driving continues through a pause, a resume and a navigation. Release, revoke, deny and leave. A note by the driver. |

Gaps found while recording, closed by the recorder (#458):
- **Clicks:** the prototype's polled observer lost the click that navigates; the recorder's observer reports clicks
  synchronously (`TestLiveRecording` holds a click before its `seg navigate`).
- **Layout:** `DOMSnapshot` on the top session leaves out out-of-process frames; the recorder captures each frame
  session's own and maps its boxes into the top page.

## Agent queries (#462)

`couch --broadcast-web-query DIR dom|events|notes` is what a coding agent runs on a recording. It reads the
directory (`recording.ReadDir`), needs no running Couch, and prints text sized for an agent's context, or
`--json` for structure. Package `cmd/internal/webcast/query`: `Querier` is the real `Surface`, one pass of the
reference reducer with nothing indexed beside the records; `Main` is the CLI; the text renderers are pure.

| Query | Answers | Text | `--json` |
|---|---|---|---|
| `dom [--at T] [--sel S] [--max-lines N]` | the page at T (default: the end) | a header (t, segment, live/paused, URL, viewport, title, driver), then one line per element `@id <tag attrs>` with form state, focus and scroll overlaid | `{t,seg,live,url,title,vp,control,matches}`, nodes in the format's `Node` shape plus `frame`, `state`, `focused`, `scroll` |
| `events --from T1 --to T2` | what happened in the window | one line per record with its time, targets named as they were then, `by <viewer>` on a driver's acts; a run of more than 3 DOM, style, asset or layout records is one line (counts by kind, most-touched elements); a field typed a key at a time is one line with its final value | the records, as JSON lines |
| `notes [--note ID] [--before MS]` | every note | author, time, segment, state now (resolved or open), text, anchor, the element as of the note, the notes model's context, and the user's actions in the `--before` ms (default 10000) before it, including segment starts and stops | `{note,state,seg,url,element,context,landmark,before}` |

- **Node ids are the handle.** Every element in every answer is `@N`; `--sel @N` shows one. A note's own
  selector is matched by the notes model's reader (`notes.Match`), so it names what the note names; anything
  else is read as a CSS subset scoped as CSS is, per tree, with `>>>` to step into a host's shadow root, frame or
  template. The grammar's one statement is `query.SelectorGrammar`, which the bare command's usage and every
  refusal print; a selector outside it is refused, never matched loosely.
- **Budgets:** a page stops at 300 lines (`--max-lines`) with a line saying how many more; a note's element at
  20 lines and its preceding actions at the latest 20; attribute values at 80 characters and text at 200.
  Script and style bodies, a document's head and SVG internals fold to one line each. The text shows the
  attributes an agent reads structure by (id, class, role, aria-*, form and link attributes); `--json`
  keeps them all.
- **The notes model** is `notes.Model` behind its interface; `Main` constructs #461's `notes.Structural`, and
  the contract and golden tests pin what it gives. Selectors accept the model's own anchors (`:nth-of-type`,
  `>>>`), so a note's selector pastes into `--sel`.
- **Tests:** `RunSurfaceContract` on the `Querier`; golden answers for every fixture in
  `query/testdata/golden/` (`go test ./cmd/internal/webcast/query -run Golden -update`, then review the diff);
  a scripted agent check that each note's answer names its element, time and the action before it.

## Remote driving (#466)

A viewer uses the app through the operator's tab: one driver at a time; a viewer requests control, the operator grants
it (and can revoke it at any moment), and the driver gives it back. Internal viewers only in the MVP (project, Remote
driving). Package `cmd/internal/webcast/drive` is the contract; #467 builds the HTTP routes, the CDP injector and the
UI against it.

**Seats.** Anyone with the viewer link can watch. Driving needs a seat and a grant:
- `POST <cast>/drive/hello` `{"name":"Ana"}` returns `{"viewer":"v3","key":"<64 hex>","name":"Ana"}`, from
  `Controller.Join`. The key stays in the shell and authenticates the seat's requests (header
  `drive.SeatHeader`, `X-Couch-Seat: <key>`).
- The shell then opens the record stream as `<cast>/events?viewer=v3`. The events route calls `Controller.Attach` for
  the stream's life.
- A seat with no stream for the grace period (`drive.DefaultGrace`, 10 s) leaves. If it drove or asked, a `ctl leave` hands control back.

Seats are memory only; they die with the cast.

**Control.** `POST <cast>/drive/control` `{"op":"request"}` or `{"op":"release"}`, with the seat key. The operator
grants, denies and revokes from Couch (`Controller.Grant`, `Deny`, `Revoke`). `Revoke` never waits on a viewer: it
takes control back in any state, including mid-batch, and does nothing when nobody drives. Every change is a `ctl`
record. Viewers learn the state from the `ctl` records in their stream, and Couch from `Controller.Subscribe`.

**Input.** `POST <cast>/drive/input`, with the seat key and a `drive.Batch`:

```json
{"seg":3,"acts":[{"a":"click","id":412,"x":530,"y":1210},{"a":"text","v":"invoice"},{"a":"key","type":"down","key":"Enter"}]}
```

- **Acts:** `down`, `up`, `move` and `click` (`count` 2 is a double click), `wheel` (`dx`, `dy`), `key` (`type`
  `down`/`up`, `key`, `code`, `mods`), `text` (inserted at the focus: typing as values), and `ime` (a composition, `v` and
  `sel`).
- **Targets:** pointer coordinates are top-page CSS px (the viewer's `toPage`). `id` is the element the viewer's
  `nodeAt` found there; `down`, `click` and `wheel` need it. `seg` is the segment the viewer was showing.
- **Parsing:** `drive.ParseBatch` is strict: no unknown fields, at most 8 KiB, 64 acts and 1,000 runes of text. Its
  errors never echo the input.
- **Answers:** `drive.Answer(err)` is the one mapping from the error to the status and the sentence the shell shows:
  - 204 when injected;
  - 401 `Join the cast again.` (`ErrNoSeat`);
  - 403 `Ask the operator for control first.` (`ErrNotDriver`);
  - 409 `The page changed. Try again.` (`ErrStale`: the element isn't at that point in the tab any more; the acts
    before it stand);
  - 409 `The cast is paused.` (`ErrNotLive`, also a batch for a gone segment);
  - 409 `Control has already changed.` (a control op the state doesn't allow);
  - 422 `That isn't allowed while driving.` (`ErrRefused`);
  - 400 on a malformed or oversized body;
  - 410 once the cast has ended.

  The sentences are fixed and never echo the request.
- **Limits:** a rate and in-flight budget per seat, as the pointer routes have (`broadcast`'s `RateLimit`, 4 in flight).

**Confinement** (`drive.Confine`, every act, before anything is dispatched). What the page alone handles is allowed;
what Chrome or the operator's machine would handle is not:
- Only the left button, with no modifiers. The right button opens Chrome's context menu; middle and modified clicks
  open tabs or windows or download; Ctrl+wheel zooms.
- Keys: only the chords in `drive`'s closed chord table (a key with an exact set of modifiers). A chord is there only
  if no Chrome or Blink binding for it, on any platform, acts outside the page's document (history, a new tab or
  window, the clipboard, browser UI, DevTools, zoom). Escape's harmless bindings were reviewed and stay: Chrome closes
  find, stops loading and leaves fullscreen. That means the editing and navigation keys, plain or with
  Shift where that only extends a selection (not Shift+Delete, which is cut, and not Shift+Escape, Chrome's task
  manager); printable characters, plain or with Shift; Ctrl or Meta with A (select all) and Z (undo, with Shift redo);
  and Ctrl+Y (not Meta+Y, History on macOS). **The activation keys, Enter and Space, take no modifier.** They are a
  click's keyboard form, and Blink takes the navigation policy from the activating event's modifiers: Ctrl or Meta opens
  a new tab, and Shift opens a new window. So every modifier refused on the pointer is refused on them too. A soft
  newline (Shift+Enter in a chat box) is lost; that is the price. `TestConfineAllowsExactlyTheseChords` walks every
  key against all 16 modifier sets. It compares them with an expectation written by hand, not derived from the table,
  plus a cited list of Chrome and Blink bindings.
- `code`, when given, must name `key`'s own key, so an injector can't be led to another (key `a` with code `KeyW`).
  The injector dispatches from `key` alone; `code` only fills the DOM event's `code`.
- No clipboard keys: copy and cut would write the operator's clipboard, and paste would read it.
- No Alt (back and forward), no function keys, and no other accelerator (closing the tab, the address bar, DevTools).

While a viewer drives, the injector (#467) also intercepts file choosers (`Page.setInterceptFileChooserDialog`, then
cancels them), denies downloads (`Browser.setDownloadBehavior`), and closes any target the cast tab opens (a new
window or tab). Permission prompts and JavaScript dialogs appear only to the operator, who answers them.

**One writer.** The record stream has one writer, the recorder's goroutine (#458). The Chrome loop forwards CDP
events and live/pause/tab inputs into its FIFO queue. Control changes and injections are inputs to the same queue, so
`ctl` records, injected input and the DOM changes it causes are ordered together, and `by` is stamped in queue order.
The injector follows that stream: it sees every record, `ctl` included, in order. A fake is fed as a
`recording.Sink`, and `FakeController` over `FakeInjector` composes through `recording.Tee`.
The injector hit-tests with `DOM.getNodeForLocation` and the recorder's id map (`Recorder.IDOf(session,
backendNodeId)` and `Recorder.NodeOf(id)`, requests with a reply on that goroutine). It rechecks that `by` drives when
it dispatches, so a batch overtaken by a revoke is refused there (`ErrNotDriver`, checked by
`RunInjectorContract`). Passwords follow #458's rule: typing into a
`type=password` field records `input {secret:true}`, never the value.

**The operator sees who drives:** Couch's `Chrome ⏸` indicator names the driver and any pending request, from
`Controller.Subscribe`. An in-page banner ("Ana is driving") is injected the same way as the Go-live control (#454),
in an isolated world and kept out of the recording.

**Lifecycle (ARCH-FUNERAL).** A `ctl` record is about 100 bytes per human control action, and lives as long as its
recording. Seats and their keys are memory only: they end after the grace period without a stream, or with the cast.
## Revisions

- 2026-10-12 (#458, additive v1, approved with #457 by the operator; shapes coordinated by the TL with #460 and
  #466): `hover {id}` and `ptr {x,y}` in events.jsonl, outside Canonical; `input.secret` and the password rules;
  `ValidateDir` also reports anything in `frames/` no frame record names (a stray `.tmp-*`). The catch-up keeps only
  the current segment's latest `hover` and `ptr` (TL, with #460).
- 2026-10-12 (#459, TL ops:0): the asset store, not the recorder, makes stored CSS bodies' URLs absolute (at
  `Put`, against the capture URL); `image-set()` plain strings are URLs to both the proxy and the viewer; the
  resample policy is written down under Assets. Additive: no record kind or field changes.
