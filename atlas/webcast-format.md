# Couch broadcast-web: the recording format (#457)

The format is the product of couch-broadcast-web (project: ops `teams/devinfra/projects/couch-broadcast-web/project.md`).
A **recording** is the DOM of a cast Chrome tab over time: a full snapshot when a segment starts, then every change,
plus what the DOM alone doesn't hold (form values, scroll, CSSOM, fonts, layout boxes), plus notes, pointer strokes and persistent sketches.
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

Accepted live events and the recording on disk carry **the same records**. A live viewer receives
them one by one as SSE events of type `rec`, each event's data being one record line. A late join starts with derived
catch-up state, including an ephemeral sketch bootstrap when needed (see Joining mid-cast). On disk the accepted
event lines are split by family into files. Splitting and merging are pure functions of `k`:

```
<recording>/
  events.jsonl         head, seg, stop, doc, DOM changes, observer state, context, control, assets, layout, sketches
  notes.jsonl          note, note-edit, note-state
  pointer.jsonl        stroke
  assets/<hash>        captured bodies (images resampled; CSS and fonts exact), named by content hash
  frames/index.jsonl   frame: the sparse low-res screenshots for the fidelity check (not in the live stream)
  frames/<n>.png
  fidelity/            the fidelity check's report (#478): not records; written beside them
```

Reading a recording merges the three record files into one stream by `t`. At equal `t`, events come first, then
notes, then pointer strokes. `frames/` is never merged in: screenshots are for the fidelity check, and pixels never go to
viewers (Principle 1). The live writers use `recording.OrderedTime` to advance a
record by one millisecond when an equal-time transition would reverse this file
precedence. A `sketch-begin` is strictly later than the previous record, and the next accepted record is strictly
later than that begin, so its timestamp names exactly the preceding source prefix. `annotation-clear` also
strictly follows previous records. Timestamp overflow refuses admission; raw file readers never restamp history.
Splitting and merging therefore preserve the accepted live order.

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
| `note` | notes | `id` (string), `author` `{id,name,color}`, `text`, `at` `{node,kind,path,sel,text,box,tag,classes,stable,route}`, `region` `[x,y,w,h]`, optional `capture` | A note on a box of the page at its `t` (#488). `region` is the note's box: what the viewer dragged, or the default box of a tap. It is the area captured at draw time, and the board writes it on every note. `capture` is the opaque private capture reference; it exposes no file path or pixels. A note from before #488 may lack it. `at` anchors the note on the smallest element containing the box's centre. `at.node` is the element's id. `at.kind` is `element`. `at.path` is the meaningful containers down to it (`main › article › p#first`). `at.sel` is a stable CSS selector. `at.text` is the element's text, trimmed to 200 characters. `at.box` is the element's box. |
| `note-edit` | notes | `id`, `text` | A text revision of any open note, admitted against the current document. Preserves the original author, creation time, anchor, region and capture; marks the projected note `edited`. |
| `note-state` | notes | `id`, `state`: `open` or `resolved`, optional `who` `{id,name,color}` | A note was resolved or reopened. Resolver attribution is retained in `NoteView.ResolvedBy` and note query results. |
| `stroke` | pointer | `id`, `author`, `color`, `pts` `[[x,y,dt],…]` | One pointer stroke, or one chunk of it; `dt` is ms after the record's `t`. A live stroke streams as several records sharing an `id` (#470: one per viewer POST, about every 50 ms), each continuing it; a viewer's stroke id is its viewer id, `.`, and the shell's own stroke id, so two viewers' strokes never join. |
| `frame` | frames/index | `file`, `w`, `h`, `scale` | A screenshot of the real tab at `t` (the fidelity check), `w`×`h` image pixels; `scale` is image pixels per CSS pixel, the one source of that mapping for every consumer: 0.5 (the viewport at half its CSS size) whatever the device pixel ratio, except under a ratio of 0.5, where a screenshot isn't upsampled and `scale` is the ratio. `t` is the capture's midpoint. Not in the live stream. |

**Who wrote it (#470).** A viewer's note or stroke carries the `author` the viewer's own shell names: an id it
generated once per browser and a name it asked for once, with the color of the swatch it picked. This is
**attribution, not authentication**: anyone with the cast link can write as any id (one always-on link for internal
viewers, TL 10-12). Remote driving (#467) swaps in seat identity. The id `operator` is reserved for Couch.

Notes stay open in the recording after navigation until resolved. The live open-note list spans documents;
a pin appears only when its anchor is confidently placed on the current view. Historical replay's `liveNotes`
filter uses `State.DocSeg`, retaining notes across pause/resume while limiting its historical page's pins. The shape is shared with the terminal's notes (#455, #488). `author`
and `text` stay the same. Both carry a box, in their own units: page px here, cells on the terminal. Both carry an
anchor read as `{kind, path, text}`: here `kind: element` and the containers' path; on the terminal, Couch's layout
path, which #488 M3 adds (atlas/broadcast.md).

## Persistent sketches (#751)

These additive version-1 kinds live in `events.jsonl`. Every kind carries `operation` (bounded identity),
`sequence` (per-sketch browser-safe nonnegative integer), and `digest` (lowercase SHA256 hex64), beside `t` and `k`.

| Kind | Payload | Meaning |
|---|---|---|
| `sketch-begin` | `id`, `author`, `owner`, `doc`, `tab`, `vp` | Freeze the source document segment, viewport, author/owner and `author.color`; one drawing sketch cast-wide. |
| `sketch-stroke-begin` | `sketch`, `id`, `at`, `width` | Begin one stroke with its recorded anchor and positive base width; discard the redo branch. |
| `sketch-points` | `sketch`, `id`, `chunk`, `pts` | Append the next chunk of original page-coordinate `[x,y,pressure]` triples. |
| `sketch-stroke-end` | `sketch`, `id`, optional `truncated` | Seal a stroke; only a nonempty stroke enters the visible stack. Pointer cancellation records `truncated: true` and permits another stroke. |
| `sketch-undo` | `sketch`, `id` | Move the named visible stack top to redo, with no open stroke. |
| `sketch-redo` | `sketch`, `id` | Restore the named redo stack top, with no open stroke. |
| `sketch-label` | `sketch`, `label` | Supply the required, nonblank label once; complete drawing or label interrupted work. |
| `sketch-interrupt` | `sketch`, `reason` | Seal accepted partial work and retain truncation evidence; reasons are `expiry`, `navigation`, `handoff`, `reset`, `end`. |

Pressure is explicitly recorded in `[0,1]`; a persisted point must have exactly three numbers. It never changes
pointer `stroke`'s `[x,y,dt]` meaning. `sketch/geometry.go` and browser `annotation-anchor.js` project points between
positive anchor boxes while retaining pressure; width is `base * (.25 + .75 * pressure)`, scaled with the viewer.
Sketch/stroke/operation/tab/author identities are bounded to 128 UTF-8 bytes, labels to 1024 bytes. The owner is
`operator` or an existing public session digest. Coordinates are finite and at most `1e7` in magnitude.
The live tool records base width 3 CSS px, as specified by the approved design. An absent or false stroke-end `truncated` has the same semantic identity;
true preserves cancellation evidence through exact retries, snapshots and replay.

`SketchCanonical` defines known request fields excluding writer time, kind and digest. `SketchDigest` hashes those
bytes; receipts also retain the concrete kind and actual semantic request. An exact retry is a no-op even with a
changed producer timestamp. Reusing an operation/sequence with a changed body conflicts even if its claimed digest
is unchanged. Producers reconcile before restamping or emission, so a retry also consumes no extra bytes or sink writes.
The preserving sketch codec rejects malformed owned fields and duplicate keys recursively, and retains additive
unknown members on the original record, including nested anchor/author/viewport members. Authoritative history and
snapshot projections contain known state; they do not copy wire-only unknown members.

`recording/sketch_state.go` owns validated transitions and indexed history; `web/sketch-model.js` is its browser
port, held to `tests/webcast/sketch-corpus.json` at each prefix. Appends visit only new points; Undo/Redo change
stack references. Pause seals partial points and blocks drawing; same-document resume keeps the sketch source.
A changed driving holder, reset, navigation or cast end interrupts drawing through the same reducer. Pending-only
control requests do not interrupt. The original owner's late label can complete interrupted metadata without old
live DOM; cast end and annotation clear forbid new labels. An interrupted sketch retains its interruption evidence
after labeling. Repeated ends after implicit sealing never duplicate a visible stroke.

Bounds are 64 points/chunk, 4096 points/stroke, 16384 points/sketch, 64 stroke identities/sketch and 256 sketch begins.
Both producers use `recording.SketchBudget`: exact encoded sketch JSONL bytes, including newlines, plus outstanding
reservations total at most 32 MiB. A sketch reserves 16 KiB: 8 KiB for its label, 4 KiB for
stroke end and 4 KiB for interruption. The maximum-envelope regression uses actual `Encode` output with maximal
identities/counters and a 1024-byte HTML-escaping-heavy label. Ending a stroke releases its end allowance; the next
stroke must reserve it again. Explicit interruption retains label capacity; derived interruption conservatively
retains the full remainder. Label, clear and cast end release unused reservations; used history bytes remain charged.
The pure core separately bounds canonical known-field encoded history to 32 MiB, normalizing signed zero to positive zero in model accounting and owned point/anchor projections, and keeping `history_bytes` in snapshots; this
counter excludes preserved unknown wire members, which the producer's exact byte budget charges.

Reducer acceptance and sink persistence are separate. A sink error after acceptance leaves state and byte charges
in place. `board.Writer` remembers that operation's uncertainty and returns it on exact retries without another
write. The recorder's mirror accepts into a queue; its shell retains sink errors in `r.err` and returns them from
`Close`. A receipt alone cannot prove durability; transport acknowledgment and query behavior belong to #752.

## The notes model (#461)

[DOM submission](webcast-submission.md) (#483) packages all open notes selected
across the recording's documents at admission. It reuses this model for recorded
context and reads each note's stored draw-time capture, preserving separate PNGs
even when boxes overlap. Its version-3 manifest carries per-note page/document
and privacy, distinguishing recorded context from saved pixels and pinning note
revisions for guarded acknowledgment. Missing legacy evidence is unavailable,
never silently recaptured from the current page.

`notes.Structural` is the real `notes.Model` (`cmd/internal/webcast/notes`), pure over the reducer's state at a
moment. It runs `notes.RunModelContract` like the fake.

**A note** is `recording.Note`: `id`, `author` `{id,name,color}`, `text`, `t`, its box `region`, and the anchor `at`.
A terminal note keeps `author` and `text`, with a box in cells and Couch's layout anchor (#488).

**The anchor** `at` retains `{node, kind, path, sel, text, box}` and freezes
`tag`, bounded `classes`, stable identity attributes and exact `route` for live
replacement matching:
- `kind` is `element`. `path` is `pathOf`: the meaningful containers above the element (the landmark rule below),
  outermost first, then the element itself. Each step is named by its tag and its id or role. The innermost six
  steps are kept.
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
`border` or `shadow*`. With none, the landmark is the parent element (at most `body`), used for the context's text but
not named in an anchor's path.

**The context** (`notes.Context`) is `{Node, Landmark, Text, Box}`: the resolved element, its landmark, the landmark's
rendered text (`DOMNode.RenderedText`, #469: what a person reads, unlike `textContent` it skips script, style,
template and noscript bodies and elements the recording marks hidden, and keeps blocks, cells, list items, `br`
and flex/grid items apart by a space) in a 500-character window centred on the element's own text, and the
landmark's box. The full DOM at that moment stays available through the agent queries (#462).

**Crops** (#488) are `NoteCrops(st, notes)`, the seam a submission crops from (#483), in CSS px of the top page:
1. each note's box (its `region`; a note from before #488 has its `at.box`, else its element's layout box), padded
   by a margin of 16 px;
2. crops that overlap merge into one image, until none do, so notes far apart get a crop each;
3. each crop is clipped to the page (the root element's box), widened to hold every note's own box.

The operator's box is the crop, so there is no snapping to structure and no per-URL hint. #461's landmark snapping
and `notes.Hints` were retired by #488. The landmark still gives the context and the anchor's path.

## Annotation clear (#523)

`annotation-clear` is an operator-authored event in `events.jsonl`:
`{"t":7001,"k":"annotation-clear","who":{"id":"operator"}}`.
It clears the current notes and pointer strokes and retires sketch editing/late labels, including while paused;
earlier replay positions remain unchanged. It is invalid after recording end. The timestamp is
strictly after preceding annotations so split-file merge cannot resurrect marks
at an equal timestamp. Catch-up retains the clear barrier and only subsequent
notes. Go reduction, browser Model, replay catchUp and Player.seek apply the
same clear barrier; the mounted annotation UI also clears its transient strokes.

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
  | `reset` | `who.id` is `operator` | No driver and no pending requests, atomically |
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

A viewer that joins late receives the **catch-up**, followed by new live records. Its DOM/assets/notes portion keeps
`head`, asset bindings, bare historical `seg`/`stop` records preserving note document identity, all note creations,
revisions/states and clear barriers, current-document pointer strokes, and the latest segment's page records.
Only the latest `hover` and `ptr` remain. Every recording keeps the existing latest-pre-segment `ctl`
baseline and contiguous current-segment control events, which driver-attributed DOM changes need.

For sketch-bearing prefixes, `recording.CatchUp` reduces the accepted prefix through the same State, removes sketch
history from the retained portion, and appends one derived `sketch-snapshot` bootstrap. It carries the
current driving holder/pending control plus bounded sketch history: source metadata, finished and active sketches,
accepted stroke points/chunks, open/sealed/truncated flags, visible/redo stacks, receipts, sequence and byte counters.
It adds no historical control retention beyond that existing baseline/current-segment contract. Repeated compaction consumes the prior bootstrap
and new events; the append-only recording on disk remains complete.

`sketch-snapshot` is catch-up-only, absent from the persisted kind registry and refused by ordered producers.
`NewCatchUpState`/`ReduceCatchUp` explicitly enable validated, atomic restoration; browser Model has the equivalent
`allowSketchSnapshot` option. Ordinary raw replay input refuses a bootstrap, and subsequent `ctl` validation remains
strict. Snapshot restoration owns copies and validates bounds and references before installing control/sketch state.
Historical anchors do not require old DOM snapshots. Invalid source prefixes are retained for ordinary validation.

The live viewer and replay seeking use this same catch-up projection. `CheckCatchUp` compares sketch projection
independently of legacy DOM canonical hashes. `recording.Hub` compacts at segment boundaries; retained DOM work is
bounded by the latest segment, alongside existing note/asset history and the bounded sketch snapshot.

## Reading rules (the reducer)

`recording.Reduce(records, t)` gives the state at time t: documents and node trees, sheets and adoptions, fonts,
the asset table, form values, scroll, selection, focus, layout, URL, title, viewport, the segment, live or paused,
the notes, sketches, and who drives (`Control`). A record that references a node that doesn't exist is an **error**, not something to skip: the
recorder must not emit one, and the validator reports it with its line. `State.Canonical()` is a deterministic
serialization of the DOM state; sketch projections are checked separately. The viewer's conformance check compares
both with what the viewer rebuilt.

## Lifecycle and size (ARCH-FUNERAL, ARCH-CONSTRAINTS)

- **A recording** is created by the recorder at go-live (#458), under Couch's store. It is needed until the operator is
  done with it. It is kept until the operator deletes its directory: there is no automatic sweep in the MVP. A recording's size is bounded per
  minute: about 1.4 MB/min gzipped on #453's busy stress page, before compact encoding. Assets are deduplicated by hash, and images are
  resampled to a resolution budget (#459). A retention sweep comes with the first upload or listing feature. It is not
  part of the format.
- **Viewers' notes and strokes** (#470) grow `notes.jsonl` and `pointer.jsonl` per viewer action, bounded by the
  board routes' budgets (strokes at most 60 a second cast-wide, 64 points each; notes 2 a second), and go with the
  recording's directory.
- **The reducer** holds one recording's state in memory, and `Reduce(records, t)` replays from the first record. The viewer's
  replay (#471) seeks from the latest `seg` before t: it reduces the catch-up of the records up to t (`catchUp` in
  `web/replay.js`, held to `recording.CatchUp`), so a seek replays at most one segment. Nothing is indexed or stored beside
  the records.

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
| pair#461 | notes model | `notes.Model` (`Anchor`, `Resolve`, `Context`, `NoteCrops`, `Live`; #488 retired `Crops` and `notes.Hints`); real: `notes.Structural` | `notes.FakeModel` | `notes.RunModelContract` |
| pair#470 | viewers' strokes and notes | `board.Board` (a `Sink` following the stream; `Stroke`, `Note`, `SetState`, `Live`, `Subscribe`) writing through an `Emit` (the recorder's queue via `webcast.Emitter`) | `board.Writer`, the reference single writer | board tests over fixtures; `Board viewer` below |
| pair#462 | agent queries | `query.Surface` (`DOMAt`, `Events`, `Notes`); real: `query.Querier`, CLI `couch --broadcast-web-query` (below) | `query.FakeSurface` | `query.RunSurfaceContract` |
| pair#467 | input injection | `drive.Injector` (`Inject(ctx, by, Batch)`) | `drive.FakeInjector` | `drive.RunInjectorContract` |
| pair#467 | driving control and the reverse channel | `drive.Controller` (seats, request/grant/deny/revoke/release, input) | `drive.MemController` | `drive.RunControllerContract` |

Hand-written recordings for tests: `recording.Builder` (ids from one counter, `Page` for a segment and its document),
`recording.ContractRecording`, `query.ContractRecording`, `recording.DrivenRecording` (viewers driving: the drive contracts' page, no longer a fixture).

**Records, bodies and frames.** A `Sink` carries records only, and `Tee` fans them out to the recording directory and
the live fan-out. An asset body is stored once: the asset store (#459) decides what to keep and persists it through
the recording's `recording.AssetWriter` (its `DirWriter`, the one writer of `assets/<hash>`, which writes atomically). Only the
asset record goes through the Sink. Screenshots go to `DirWriter.PutFrame`. A `Live` refuses frames
(`ErrNotLive`), because pixels never reach viewers; `RunLiveContract` checks this.
`DirWriter.Put` appends each record to its file as it is put, unbuffered (pair#493): a recording is read while it is
written (the live replay, `--broadcast-web-query`), and per-file buffers held a cast's few notes until it stopped. A
reader can race a write and see a half-written last line, so `ReadDir` decodes complete lines only (its `Files` keep
the raw bytes, and `ValidateDir` still reports a torn tail on an ended recording).
`TestCastBoardNotesAndStrokesReachDiskWhileLive` reads a live cast's directory without closing it.

**Seam into `webcast.Cast` (#454).** Today `webcast.Server.events` streams the Browser's `state` (the stub). The recorder (#458)
adds `LaunchOptions.Sink recording.Sink` to `webcast.Chrome`. The Chrome loop already owns every page session and the
live phase, so it feeds the active tab's CDP events to the recorder while `PhaseLive`. Go-live emits `seg`, pause emits `stop`,
and a tab switch from `Tracker.Active` emits a new `seg`. `CastConfig` gains the `recording.Live` that fans the
same records out, and `Server.events` sends them as `rec` events. The route, the token and the shared `broadcast.Host` stay.

**Viewers’ strokes and notes (#470).** The `POST <cast>/stroke`, `/note`, `/note-edit` and `/note-state` routes read
a body (`webcast/server_board.go`), admitted like the terminal pointer routes: a JSON content type, 4 in flight, a
budget per cast (strokes 60/s, notes and states 2/s with a burst of 5), a prompt body within 8 KiB. Bodies are parsed
strictly by `board.Parse*` (each names its `viewer` `{id,name}` and the `seg` it was made on; a note its `node`, the
`box` from `couchViewer.boxOf`, its `region` (required since #488; the board clips it to the page widened to the
viewport, `notes.ClampToPage`, and refuses one that is off it), a palette `color` and plain text of at most 1,024 bytes). The
`board.Board` follows the recording as one of its sinks (the recorder's `Tee`: the directory, the board, then the Hub, so the board knows a
page before any viewer can act on it) and
checks each request against that state: the cast is on a page and hasn't ended, the `seg` is within the current
document (a resume keeps it), the node is a live element, a color has at most 8 open notes on the page (notes emitted
and not yet seen back count). It anchors a note with `notes.Model.Anchor` and the viewer's box, and hands the record to
the cast's recorder (`webcast.Emitter`, `recorder.Emit{Record, Doc}`), which stamps `t` and writes it **only if the
document the board checked is still current, and live, when it orders it** (`recording.State.OnDoc`), so a navigation
between the check and the write can't file a note on the new page, nor a pause file one during it. Answers are fixed sentences (`boardAnswer`): 204; 409 "The page
changed. Try again.", "This color already has 8 notes. Resolve one first.", "That note is gone.", "The cast isn't
showing a page yet.", "The cast is paused."; 410 "The cast has ended."; 429 "Too many requests. Wait a moment."; 400 "That isn't valid.";
503 "Notes aren't working in this cast." (the board lost track of the page: a bug, logged once).
**While paused, viewers do nothing (pair#499, reversing #470's "notes allowed while paused"):** the board refuses
`stroke`, `note`, `note-edit` and `note-state` (`ErrPaused`, 409 "The cast is paused."), `OnDoc` drops a record admitted just before
the pause, and the viewer's rail disables its swatches, ✏️ and 💭 until a `seg` resumes (annotate.js `paused()`), as does the
note dialog's Save and Resolve (collab.js `noteDialog.hold`; a draft keeps its text). `board.Writer` answers a pause
drop with `ErrPaused`. Notes
made before the pause stay pinned through it. The terminal broadcast has no paused state. Couch follows the open notes with `Cast.Notes` and resolves as the operator with
`Cast.ResolveNote`: `💭N` follows the Chrome cell (`StatusModel.WebNotes`, its own span), and a click opens #447's notes
menu with the web notes (one menu with a source, screen or web; items show `name: text`, a click copies the text, a right
click copies and resolves). The collaboration overlay the menu draws in is installed while the screen broadcast or a web
cast needs it (`syncOverlay`). Lifecycle: the board lives with the cast (the recorder closes it with the recording); its pending
entries end when their record is seen, the emit fails, or a new document begins.

**Draw-time capture (#538).** New viewer notes use `POST prepare-note` with
`{request,seg,node,box,region}`, then `draft-status` with `{request,draft?}`.
`save-note` sends `{request,draft,viewer,color,text}`; `cancel-draft` retires the
pending request or signed token. These routes keep the annotation-generation
header, existing note-rate budget and bounded admission; the envelope limit is
40 KiB to accommodate opaque tokens up to 32 KiB. The operator uses corresponding
isolated-world operations, supplying the recorder's structural `path` at Prepare
and private owner/session authority from CDP. Browser request IDs are correlation,
not ownership. A private capture record owns pixels and metadata; the public
recording note carries only its opaque `capture` reference.

The shared controller opens an editable draft immediately at mouseup, projects
pending/ready/failure without replacing text, and enables Save once ready. The
capture tick requires confirmed captured time. Ordinary capture failure offers
Retry capture; bounded queue/store pressure offers Save with explicitly unavailable
evidence. Save waits for the writer's acknowledgment; pending/unknown is never
success, and edited retry is admitted only after the previous writer is proved
absent. Cancellation is fenced from late answers and cannot retire an attached note.
The capture store is local, outside served recording roots; preparation alone never
adds pixels to the public asset map. Details and limits are in
[submission evidence](webcast-submission.md#draw-capture-and-save-538).

**The JS viewer contract (#460).** The viewer applies records to a sandboxed frame without `allow-scripts`
(`webcast/web/`: `model.js` applies, `render.js` draws). On `window.couchViewer`:
- `resolveAsset(absURL)` returns the proxy URL for an asset (`<cast>/assets/<hash>`), or nothing;
- `canonical()` exports the rebuilt DOM in `State.Canonical()`'s shape (tree and ids from the drawn DOM, values from
  the model), and `ready()` resolves once its stylesheets, fonts and images have loaded;
- `boxOf(id)` is a node's box in top-page CSS px from the viewer's own unscaled layout. A note made in the viewer
  carries this box from click time.
- `placementOf(note)` returns a verified live region or null; `onPlacementChange(fn)`
  requests repaint after DOM/layout changes without replacing the record pipeline.
- `nodeAt(pageX, pageY)` is the id of the innermost drawn element at a top-page point (into frames and shadow roots):
  the target of a driver's act (#466). `toPage(clientX, clientY)` maps a point in the viewer's window to top-page CSS px,
  and `fromPage(pageX, pageY)` back (#470's pins and strokes).
- `onRecord(fn)` sees each accepted record and the model, so another component (#467's control state) can follow the
  stream without touching the apply path.
**Pointing and notes in the viewer (#470).** `web/annotate.js` (styles in `annotate.css`), mounted by one call from
`viewer.js` when the cast takes notes (`palette.json` answers only with a board). It owns its own elements: a layer over
the scaled page (input only in point or note mode, `touch-action: none` there, so watching leaves touches alone), the
swatch rail with ✏️ and 💭 mode buttons docked to the page box, a strokes canvas, the pins and the note dialog. The rail's
swatches and the dialog are `collab.js`, shared with the terminal viewer (`broadcast.CollabJS`; the file in
`webcast/web/` is a symlink for the node tests). Point mode posts a stroke's points every 50 ms and draws them at once
(its echo is skipped); others' strokes fade 1.5 s after their last point, and a reconnect's catch-up strokes (more than
1.5 s behind the newest record) are not drawn. Note mode (#488) has one gesture rule, `collab.js`'s `gestureBox`, shared with
the terminal viewer:
- A drag of 8 px or more draws the note's box; a finger needs 16 px (`DRAG_MIN`).
- A tap gets the default box: a tenth of the page shown each way, centred on the tap. It is shifted, not shrunk,
  to stay in view.
- The note anchors on `nodeAt`'s element at the box's centre.
- `openNotes` lists every open note across recording documents. Live pins use
  `placementOf` only: removed, hidden, clipped, offscreen or ambiguous anchors have
  no box, while their list entry reads **not on this view**. No historical box is
  substituted into the live page.
- The 💭 chip sits inside the box's top-right corner and opens the note with Resolve (without it under the view-only link, pair#506). The identity (`couch.viewerId`, made once;
`couch.viewerName`, asked in the first draft) is kept in localStorage behind try/catch, and a stored name is read by
the board's rules (else asked again). Controls that take input (swatches, mode buttons, pin chips) are rebuilt only
when their data changes; each frame only places them. For replay (#477), the drawing is pure over the model at t:
`liveNotes(model)`, `strokesAt(model, t)` (chunks joined by id, the points drawn by t, fading 1.5 s after the last),
`paintStrokes(canvas, strokes, toClient, rect, dpr)` and `pinLayer(doc, container, open).draw(notes, boxOf, place)`.
The live viewer draws with `openNotes`, strict `placementOf`, `pinLayer` and
`paintStrokes` (its strokes fade on arrival time, `strokeStore`); `strokesAt` is the
at-t counterpart. Replay (#477, `replayMarks` in `replay.js`) deliberately retains
`liveNotes` and historical `noteBox` (recorded region, then element, then carried box).

`annotation-anchor.js` shares its scoring contract with Go: matching tag is
mandatory; stable identity contributes 50, useful exact normalized text 40,
meaningful ancestor suffix 20, class overlap at most 10. Replacement requires at
least 60 and a 20-point margin; weak/common text and truncated scans cannot win.
`annotation-placement.js` performs the shared bounded DOM/visibility walk (2,000
nodes), preserves shadow scope uniqueness, and refuses frame replacement without
frame-route evidence. A connected recording ID is authoritative only for a note
from the current document; conflicting evidence hides it. Its original normalized
region follows the live anchor geometry, through visibility/clipping checks.
DOM/text/attribute mutation invalidates candidate facts; scroll, resize and
animation completion request another placement. Operator IDs cannot map to live
recording IDs, so the operator always uses scored evidence. Pin and list callbacks
read the latest note record; moving or hiding a pin never rewrites saved pixels.
In replay mode the notes module doesn't mount (the replay server
serves no `palette.json`).

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
#453's pages in its scenarios, and `fixturegen -recorded` adds the notes and pins, converting nothing. #453's prototype
recorder's fixtures (`stress`, `app`, `blog`) and its CDP-log converter retired once every component ran on these
(#474). `recording.Fixtures()` is every directory there with a pin file, as the node tests' `fixtures.mjs` finds them,
so nothing lists them by hand. A few notes and strokes are placed on known elements. `TestFixtureCoverage` holds each
fixture to the cases below (and fails on checks for a fixture that's gone), so a regenerated one can't silently lose a
case.

| Fixture | Page | Covers |
|---|---|---|
| `segments` (hand-made) | `fixturegen -segments` | Pause and resume with ids kept, an edit while paused, a tab switch, a same-document navigation; notes span the resume and clear on the switch |
| `rec-stress` (5 s, 0.8 MB) | #453's stress page from `testdata/stress`, by the recorder | Busy re-renders (3,000 inserts, 1,000 `kids`), a cross-origin iframe, open and closed shadow roots, `insertRule` (owned sheets), `adoptedStyleSheets`, checkbox and select state set by script, inner scroll, SVG, `<template>`. No FontFace font: the page's late font is a macOS system font, which `testdata/stress` leaves out |
| `rec-app` (22 s, 0.5 MB) | #453's React + Tailwind app, `app` scenario, by the recorder | Typing (with the caret's `selection` after each key), clicks, focus, a modal, Tailwind's runtime `<style>` over CDP's 10k text cut, and a cross-document navigation (a second segment). A note is resolved. |
| `rec-blog` (12 s, 1.9 MB) | The operator's blog post `ariadne-1-building-blocks-of-ai-coding` (Astro + Tailwind, `npm run dev`), `browse` scenario, by the recorder | Images, the Inter woff2 font, stylesheets, page scrolling, layout, and the giscus cross-origin iframe |
| `rec-segments` (10 s, 0.3 MB) | #453's reports page, by the recorder | A same-document navigation, a pause with an edit, a resume, a new tab (the cast follows it, at that tab's own viewport), and a cross-document navigation there |
| `rec-driven` | the real recorder and remote driving (#467): `TestRecordFixtures/driven` over `serveDriven`'s two pages, through MemController and TabInjector, then `fixturegen -recorded … -name rec-driven` | Ana asks, is granted, clicks and types (`by`). The operator clicks while she drives. Her click follows a link (a navigation while driving). A pause and resume. A release, a deny (Bo), a second grant and a revoke, and a leave (Cy). A note by the driver, from fixturegen's spec. |

Gaps the prototype's recordings showed, closed by the recorder (#458):
- **Clicks:** the prototype's polled observer lost the click that navigates; the recorder's observer reports clicks
  synchronously (`TestLiveRecording` holds a click before its `seg navigate`).
- **Layout:** `DOMSnapshot` on the top session leaves out out-of-process frames; the recorder captures each frame
  session's own and maps its boxes into the top page.

## The fidelity report (#478)

`<recording>/fidelity/` is written by the fidelity check (`webcast.CheckFidelity`, mapped in
[webcast.md](webcast.md#the-fidelity-check-478)), from the CLI or when a recorded cast ends. A rerun replaces it
whole. It is not part of the record stream: readers and `ValidateDir` ignore it, and it goes when the recording's
directory does. At most 10 crops at most the frame's size plus two small text files.

- `report.json` is `fidelity.Report`: `recording`, `tolerance` (% of a frame's pixels), `frames`, `skipped` (frames
  with nothing live to replay), `median`, `max`, `over` (frames over the tolerance), `worst` (indexes into `shots`,
  worst first: every frame over the tolerance and at least 3, at most 10). Each shot has `file` (the tab's frame),
  `t`, `diff` (%), `over`, the `offset` (ms) and `at_t` (its diff at t) when it matched best a little earlier or
  later, and `crop` for a worst one.
- `report.md` is the same for a person: the summary, the worst moments, and every frame.
- `worst-<n>.png` is a worst moment at the frame's resolution: the tab, the replay, and where they differ (red),
  cropped to the difference.

## Agent queries (#462)

`couch --broadcast-web-query DIR dom|events|notes|link` is what a coding agent runs on a recording.

- **The queries** (`dom`, `events`, `notes`) read the directory (`recording.ReadDir`) and need no running Couch.
  They print text sized for an agent's context, or `--json` for structure.
- **`link [--at T]`** is the one verb that asks the running Couch. It shares the recording through the broadcast
  tunnel (#477, [webcast.md](webcast.md#replay-471)) and prints its replay link at T (default 0) on stdout. With no
  Couch it refuses, and names `couch --broadcast-web-replay DIR`.
- **Package `cmd/internal/webcast/query`:** `Querier` is the real `Surface`, one pass of the reference reducer with
  nothing indexed beside the records. `Run` is the CLI, and couchcmd hands it the `Linker` that `link` calls. The
  text renderers are pure.

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
  Script and style bodies, a document's head and SVG internals fold to one line each. `events` folds a field typed
  a key at a time into one line with its final value, the framework's `value` attribute and the typist's collapsed
  caret after each key included (#474: the recorder reports that caret; another person's caret or a range splits it). The text shows the
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

**Three links (#467, pair#506).** Each has its own token and its own code (#495), and each grants what the one
before it does, and more (`castLink`, `webcast/link.go`):
- **The view-only link** (the cast's token, `Update.Link`) watches. Its board POSTs answer 403 "This link is
  view-only." before admission, and its `palette.json` says `annotate: false`, so its viewer draws others' strokes
  and notes read-only.
- **The pointer link** (`Update.PointerLink`, only on a recorded cast) also points and leaves and resolves notes.
- **The drive link** (`Server.DriveToken`, `Update.DriveLink`) serves the same viewer plus the drive routes below,
  which answer 404 under the other two. A stream holds a seat only under the drive link.

Couch copies them with a right-click on the Chrome cell, its 👆 and its 👽, or from the Codes menu (Alt+click on the
cell, #495). `couch --broadcast-list` lists each share by kind (`screen`, `web view`, `web drive`), never with its
token. A pointer-link viewer counts under `web view`.

**Seats.** Anyone with the drive link can watch and take a seat; driving needs a grant:
- `POST <cast>/drive/hello` `{"name":"Ana"}` returns `{"viewer":"v3","key":"<64 hex>","name":"Ana"}`, from
  `Controller.Join`. The key stays in the shell and authenticates the seat's requests (header
  `drive.SeatHeader`, `X-Couch-Seat: <key>`).
- The shell then opens the record stream as `<cast>/events?viewer=v3`. The events route calls `Controller.Attach` for
  the stream's life.
- A seat with no stream for the grace period (`drive.DefaultGrace`, 10 s) leaves. If it drove or asked, a `ctl leave` hands control back.

Seats are memory only; they die with the cast.

**Capability provenance (#523).** Mutating annotation POSTs require
`X-Couch-Annotation-Generation`; drive control/input POSTs require
`X-Couch-Drive-Generation`, alongside the seat key. Each is the decimal generation
from the capability snapshot at request creation. Missing, malformed or retired
generations answer 403. Taking a seat remains available while drive is off.

**Control.** `POST <cast>/drive/control` `{"op":"request"}` or `{"op":"release"}`, with the seat key. The operator
grants, denies and revokes from Couch (`Controller.Grant`, `Deny`, `Revoke`). `Revoke` never waits on a viewer: it
takes control back in any state, including mid-batch, and does nothing when nobody drives. Every change is a `ctl`
record. Viewers learn the state from the `ctl` records in their stream, and Couch from `Controller.Subscribe`.

**Input.** `POST <cast>/drive/input`, with the seat key and a `drive.Batch`:

```json
{"seg":3,"acts":[{"a":"click","id":412,"x":530,"y":1210},{"a":"text","v":"invoice"},{"a":"key","type":"down","key":"Enter"}]}
```

- **Acts:** `down`, `up`, `move` and `click` (`count` 2 is a double click; the viewer's `gestures` counts it from
  time and place, `DOUBLE_MS` and `DRAG_PX`, because Chrome's pointer events carry `detail` 0 — pair#505), `wheel` (`dx`, `dy`), `key` (`type`
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
  - 409 `The page changed. Try again.` (`ErrStale`: a down's or click's element isn't at that point in the tab any
    more, or a wheel's point has nothing under it; the acts before it stand);
  - 409 `The cast is paused.` (`ErrNotLive`, also a batch for a gone segment);
  - 409 `Control has already changed.` (a control op the state doesn't allow);
  - 422 `That isn't allowed while driving.` (`ErrRefused`);
  - 400 on a malformed or oversized body;
  - 410 once the cast has ended (also the recorder closing under a request);
  - 502 `The cast couldn't do that. Try again.` (`ErrInput`: a CDP input call or hit test failed; it wraps the cause);
  - 503 `The cast is busy. Try again.` (the request's context ended while it waited on the recorder);
  - 500 `Something went wrong.` for anything unmapped (a bug).

  The sentences are fixed and never echo the request. Every 5xx is logged for the operator with its cause (the
  cast's `Logf`, Couch's notice line; `Server.WithDriveLog`), once per distinct message (pair#505: every input
  answered 500 and nothing said why).
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
`recording.Sink`, and `MemController` over `FakeInjector` composes through `recording.Tee`.
The injector hit-tests with `DOM.getNodeForLocation` and the recorder's id map (`Recorder.IDOf(session,
backendNodeId)` and `Recorder.NodeOf(id)`, requests with a reply on that goroutine). It rechecks that `by` drives when
it dispatches, so a batch overtaken by a revoke is refused there (`ErrNotDriver`, checked by
`RunInjectorContract`). Passwords follow #458's rule: typing into a
`type=password` field records `input {secret:true}`, never the value.

**As built (#467).** The injector's rules are stated once, in `drive.TabInjector`, over a `drive.Tab` seam: who
drives, live and segment, `Confine`, then per act a hit test (`onTarget`: the element at the point is the aimed one
or inside it, or an out-of-process frame's owner holding it) and the dispatch. A wheel skips the identity check
(pair#516, the operator's call: it scrolls whatever is under the pointer, and a flick scrolls the tab ahead of the
viewer); it keeps the bounds check, a point with nothing under it. The fake injector is the same rules
over a reduced page. The real tab is the recorder (`Recorder.WithTab`). A batch is one queue input:
- **Hold.** The recorder holds the in-page observer (`__couchcastHold(by)`, in every observer context of the cast
  tab) and records what it had pending without `by`. During the hold the observer tags everything it sends with
  `by`, through the binding and in order. A click goes at once, so one that follows a link is recorded before its
  document goes (a take can't reach a document that has gone).
- **Dispatch.** CDP input (`cdpAct`: `Input.dispatchMouseEvent`/`dispatchKeyEvent` with editing commands for the
  A/Z/Y chords, `insertText`, `imeSetComposition`). Input coordinates are viewport px (the act's page px less the top
  document's scroll). Hit tests use `DOM.getNodeForLocation` at the act's **page** px as they are (pair#505,
  measured: it hit-tests document coordinates within the viewport; subtracting the scroll made every click after a
  scroll stale), without user-agent shadow DOM: an input's inner editor isn't a recorded node. "No node found" is
  stale; any other hit-test failure is `ErrInput`.
- **Take.** `__couchcastTake` sends what is still coalesced, tagged, and returns nothing; it ends the hold. A
  tagged report ordered after control moved (a revoke between a batch and its reports) keeps the act and loses the
  attribution: the recorder writes it without `by`, since the stream can't state a driver who no longer drives.

What a driver holds down (a button, keys) is `TabInjector`'s to end. It lets go before another driver's input,
before input on another page, when the driver's batch is refused, and on every control change (`drive.Releaser`,
naming the outgoing driver).

Every exit takes. A hold no take ends expires in the page after 1 s, and what it then sends has no `by`. What stays
unattributed is whatever the page reports after the take: the scroll a wheel causes on the next frame, page script
reacting later, and the items of a hold a dialog interrupted. A JavaScript dialog the driver opens returns
`drive.ErrDialog` (409, "The page is showing a dialog; the operator must answer it.") at once; the Chrome loop
signals the dialog to the recorder, and the stopped CDP call finishes on its own.

The controller is `drive.MemController`. In a cast (`Cast.Drive()`), its sink is `liveGate`: control records go into
the recorder's queue once `Recorder.Opened()`, and are refused before (409 paused). The gate asks the recorder, not
Chrome's published phase. While anyone drives, `Chrome.SetDriving`:
- denies downloads browser-wide;
- intercepts file choosers on every page session, including pages opened later;
- closes a page the cast tab opens before the tracker can follow it;
- names the driver on the in-page control.

All of it is undone when nobody drives. The routes share the board's admission (`admitBody`). Budgets: per seat for
control (2/s) and input (30/s), 4 in flight; per cast for hello (2/s, burst 8). An events stream with
`?viewer=<seat>` holds its seat.

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
- 2026-10-12 (#470, TL ops:0, additive): a live stroke may stream as several `stroke` records sharing one `id`, each
  continuing it; the note and stroke `author` is the viewer's self-asserted identity (attribution only); `operator` is
  a reserved author id; `notes.Model.NoteCrops` brings region notes to crops.
- 2026-10-12 (#488, additive for readers):
  - Every new `note` carries `region`, its box. A tap's is the default box.
  - `at` gains `kind` and `path`.
  - Readers keep `region` optional for older recordings.
  - `NoteCrops` crops the box plus 16 px, with no snapping, clustering only by overlap, and no hints.
- 2026-10-12 (#478, additive): `fidelity/` in the recording directory, the fidelity check's report. No record kind
  or field changes; frames are as #457 defined them.
- 2026-10-12 (#467, additive, as built): no record kind changes. `by` is stamped by the recorder from the
  observer's tagged reports during a driver's batch. `ctl` records come from the real controller through the
  recorder's queue, gated on the recording having opened. The hand-written `driven` fixture is replaced by
  `rec-driven`, a recorded driven session. Two links then: a view link (no drive routes) and a drive link (its own
  token). pair#506 later split the view link into view-only and pointer links (above).


### Own-note revisions (#537)

`POST <cast>/note-edit` accepts only `seg`, `id` and `text`. The Host lock gate
supplies `broadcast.NoteOwnerFromContext` after validating the existing unlock
cookie; public viewer IDs, names and request headers never grant edit authority.
Creation records carry a public opaque `owner` digest. Only that session can edit
an open current-document note; legacy notes without an owner are read-only.
The trusted operator context uses `OperatorNoteOwner` and cannot edit viewers'
notes. Board admission and the ordered recorder both enforce annotation state.
Accepted edits apply in Board order and invalidate submission revisions.

Reducers copy the original note before changing text, so query context still
comes from its creation record. Catch-up, replay and live pins use the latest
revision; retired notes cannot be revived by a late edit. Resolve remains shared
housekeeping and records `who`, including operator submission resolution.

A reconnect catch-up retains note records, revisions and states across documents, with bare segment records preserving anchor provenance. It sends only the latest document snapshot and current-document strokes; off-view notes remain available in the open-note list.
