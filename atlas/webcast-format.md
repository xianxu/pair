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
  events.jsonl         head, seg, stop, doc, DOM changes, observer state, context, assets, layout
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
| `stop` | `why`: `pause` or `end` | The segment ends. The state is frozen as it was. |

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
- CSS bodies (`text/css` assets) and `sheet` text have absolute `url()`s; the proxy rewrites them for viewers at
  serve time (#459).
- Assets outlive segments: the table is per recording, and a later `asset` record for the same URL rebinds it.

## Observer state

These records cover state that CDP's DOM domain doesn't report. The recorder's isolated-world observer reads it (#453).

| Kind | Fields | Meaning |
|---|---|---|
| `input` | `id`, one or more of `v` (value), `checked`, `sel` (selected option indexes) | A form control's current state: a property, not an attribute. It is set by the user or by script; typing is recorded as values, not keystrokes. |
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
| `layout` | `boxes` `{id: [x,y,w,h]}` | Layout boxes of rendered nodes, replacing the previous set. The recorder captures them (`DOMSnapshot`) when a note is made and with each `frame`, not continuously. The notes model needs them for focused context and crops; a box is as of its record's `t`. |

**Coordinates:** every coordinate in the format is in **CSS pixels of the top document's page** (scroll-independent). This
covers `click`, layout boxes, note boxes, pointer strokes and crops. Something in an iframe is mapped into the
top page by the recorder. The viewport position at time t is the top document's `scroll`.

## Notes and pointer

| Kind | File | Fields | Meaning |
|---|---|---|---|
| `note` | notes | `id` (string), `author` `{id,name,color}`, `text`, `at` `{node,sel,text,box}`, optional `region` `[x,y,w,h]` | A note anchored on an element at its `t`. `at.node` is the element's id. `at.sel` is a stable CSS selector. `at.text` is the element's text, trimmed to 200 characters. `at.box` is its box. `region` is set for a dragged region. |
| `note-state` | notes | `id`, `state`: `open` or `resolved` | A note was resolved or reopened. |
| `stroke` | pointer | `id`, `author`, `color`, `pts` `[[x,y,dt],…]` | One pointer stroke; `dt` is ms after the record's `t`. |
| `frame` | frames/index | `file`, `w`, `h`, `scale` | A screenshot of the real tab at `t` (the fidelity check). Not in the live stream. |

Notes stay in the recording after a reload. That live notes clear on reload (project, Direction) is the notes model's
rule (#461): the notes "on" the current document are those since the document began (`State.DocSeg`), so a pause and
resume keeps them and a navigation clears them. The shape is meant to be
shared with the terminal notes of pair#455. `author` and `text` stay the same; `at` is where a DOM note and a grid note differ.

## Joining mid-cast

A viewer that joins late receives the **catch-up**, followed by the live records. The catch-up is `head`; every `asset`
record so far; the notes, note states and strokes since the current document began; then everything from the latest
`seg` on. While paused, that last part is the segment through its `stop`. Reduced, the catch-up gives the same DOM, assets and
current-document notes as the whole recording (tested at every cut point). `recording.CatchUp` is this rule; the
live fan-out (#460) serves it. The fake hub keeps only `CatchUp(log)` at each segment, so what it holds for late
joiners is bounded by one segment.

## Reading rules (the reducer)

`recording.Reduce(records, t)` gives the state at time t: documents and node trees, sheets and adoptions, fonts,
the asset table, form values, scroll, selection, focus, layout, URL, title, viewport, the segment, live or paused,
and the notes. A record that references a node that doesn't exist is an **error**, not something to skip: the
recorder must not emit one, and the validator reports it with its line. `State.Canonical()` is a deterministic
serialization of the state. The viewer's conformance check compares it with what the viewer rebuilt.

## Lifecycle and size (ARCH-FUNERAL, ARCH-CONSTRAINTS)

- **A recording** is created by the recorder at go-live (#458), under Couch's store. It is needed until the operator is
  done with it. It is kept until the operator deletes its directory: there is no automatic sweep in the MVP. A recording's size is bounded per
  minute: about 1.4 MB/min gzipped on #453's busy stress page, before compact encoding. Assets are deduplicated by hash, and images are
  resampled to a resolution budget (#459). A retention sweep comes with the first upload or listing feature. It is not
  part of the format.
- **The reducer** holds one recording's state in memory and seeks by replaying from the latest `seg` before t. An index of
  segment offsets is a later optimization, derived from the records and never stored as a second truth.

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
| pair#460 | live viewer | `recording.Live` (fan-out with catch-up); the JS viewer contract below | `recording.FakeHub` | `recording.RunLiveContract`; the fixtures' canonical goldens |
| pair#461 | notes model | `notes.Model` (`Anchor`, `Resolve`, `Context`, `Crops`, `Live`; `notes.Hints`) | `notes.FakeModel` | `notes.RunModelContract` |
| pair#462 | agent queries | `query.Surface` (`DOMAt`, `Events`, `Notes`) | `query.FakeSurface` | `query.RunSurfaceContract` |

Hand-written recordings for tests: `recording.Builder` (ids from one counter, `Page` for a segment and its document),
`recording.ContractRecording`, `query.ContractRecording`.

**Seam into `webcast.Cast` (#454).** Today `webcast.Server.events` streams the Browser's `state` (the stub). The recorder (#458)
adds `LaunchOptions.Sink recording.Sink` to `webcast.Chrome`. The Chrome loop already owns every page session and the
live phase, so it feeds the active tab's CDP events to the recorder while `PhaseLive`. Go-live emits `seg`, pause emits `stop`,
and a tab switch from `Tracker.Active` emits a new `seg`. `CastConfig` gains the `recording.Live` that fans the
same records out, and `Server.events` sends them as `rec` events. The route, the token and the shared `broadcast.Host` stay.

**The JS viewer contract (#460).** The viewer applies records to a sandboxed frame without `allow-scripts`:
- `apply(record)` applies one record;
- `resolveAsset(absURL)` returns the proxy URL for an asset, or nothing;
- `canonical()` exports the rebuilt DOM in `State.Canonical()`'s shape.

Conformance means that, for every fixture at each pinned time (its `frame` times), `canonical()` equals the reducer's
`State.Canonical()`, and the render is within tolerance of `frames/<n>.png`. The fixture pins each expected state by
hash (`canonical.sha256`). The expected value itself is printed on demand
(`fixturegen -canonical <fixture> -at <t>`), so the reducer stays its only source.

## Fixtures

The fixtures are in `cmd/internal/webcast/recording/testdata/fixtures/`; their README says how to regenerate them, and
what they leave out. The three real ones were recorded by #453's prototype recorder (branch `453-recorder-prototypes`,
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

Gaps found while recording, for #458:
- **Clicks:** the prototype's polled observer lost the click that navigates, so clicks must be reported synchronously.
- **Layout:** `DOMSnapshot` on the top session leaves out out-of-process frames, so each frame session needs its own capture.
