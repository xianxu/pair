# pair#470 — pointer and notes in the web cast viewer: plan

Issue: `workshop/issues/000470-…`. Project: ops `couch-broadcast-web` (wave 2). Seam decisions approved by the TL
(10-12, 1f98c952): self-asserted viewer identity, one always-on link, chunked strokes, the Board as a Sink with an
emit func, `Model.NoteCrops`.

## Goal

Viewers of a web cast point (colored strokes everyone sees) and leave notes pinned on an element or a dragged region.
Notes and strokes are recording records, so they reach every viewer through the live stream, land on disk with the
recording, survive pause/resume and clear (live) on a new document. The operator sees the live notes in Couch as
`💭N` after `Chrome ⏸`, with #447's notes menu. A region note's region reaches `Crops`.

## Architecture

```
viewer shell ──POST <cast>/stroke|note|note-state──▶ webcast.Server ──▶ board.Board ──emit──▶ the stream's one writer
     ▲                                                  (admission:        (validates,          (#458: recorder queue,
     │                                                   rate, in-flight,   anchors via          stamps t; tests: stamp
     └──────────── SSE rec (note, note-state, stroke) ◀── Hub ◀── Tee ◀────  notes.Model)         + Put into the Hub)
                                                                    │
                                              Board follows the stream as a Sink (reducer State) ──▶ Couch 💭N + menu
```

- **`board` package** (`cmd/internal/webcast/board`, new): the collaboration layer's logic, pure but for its lock.
  - `wire.go`: `ParseStroke`, `ParseNote`, `ParseNoteState` — strict JSON (no unknown fields, size caps from
    `broadcast.MaxPointBody`/`MaxNoteBody`, finite coordinates within ±1e6 px, ≤ `broadcast.MaxPointsPerBatch` points,
    text ≤ `broadcast.MaxNoteBytes` valid UTF-8 and not blank, color in `0..broadcast.PointerColors-1`). Each body
    carries `viewer` `{id, name}`: id `^[A-Za-z0-9_-]{1,64}$` (the events route's `?viewer=` shape), name ≤ 40 runes,
    control characters stripped. Errors never echo input.
  - `Board`: `Put(rec)`/`Close()` (a `recording.Sink`: follows the stream into a `recording.State`), `Stroke(v,
    in)`, `Note(v, in) (id, error)`, `SetState(v, id, state)`, `Live() []LiveNote`, `Subscribe()` (latest
    live-notes snapshot, one-slot channel, never blocks the board — the `drive.Controller.Subscribe` shape).
  - Rules, each a named error: ended (`stop end` seen) → `ErrEnded`; no segment yet → `ErrNotLive`; `in.Seg` older
    than the current document (`State.DocSeg`) → `ErrStale` (the viewer was showing another page); note's node not
    live in the state → `ErrStale`; a ninth open live note in one color → `ErrColorFull` (reuses
    `broadcast.MaxNotesPerColor`); `SetState` on a note not live on this document → `ErrUnknown`. Paused is fine
    (TL: notes while paused yes).
  - Records built: `note{id, author{id,name,color hex from broadcast.PointerPalette()}, text, at, region?}` with
    `at = model.Anchor(st, node)` and `at.Box` replaced by the viewer's `boxOf` box (approved 10-12); a region note's
    node is the innermost element the viewer found at the region's centre. Note ids are 8 random bytes hex (unique
    without coordination). `stroke{id: viewer's stroke id, author, color, pts}`: a stroke streams as several records
    sharing an id, one per POST (~50 ms), each continuing it (format revision, additive). `note-state{id,state}`.
  - **The writer re-checks the document (ordering).** A note's segment is the one current when the record is
    reduced (`reduce.go`: `Seg: s.Seg`); the record carries none. So the Board's check can't be the last word: a
    navigation ordered between the check and the write would file the note on the new page. The emit seam is
    `Emit func(r recording.Record, doc int) error`: write `r` only if the document that began at segment `doc` is
    still current **when the writer orders it**, else drop it. The rule has one statement, `board.OnDoc(st, doc)`
    (`st.DocSeg == doc && !st.Ended`), which the recorder (#458) calls on its own reducer state at dequeue, and
    `board.Writer` (the reference writer: follows a Sink's State, stamps `t`, applies `OnDoc`) calls in tests. Strokes
    and note states use the same seam. A drop after the viewer's 204 is the price of an asynchronous queue: the note
    is for a page nobody is on any more, which the new document would clear anyway.
  - Locking: `emit` is called **outside** the board's lock (a synchronous emit into a Tee that feeds this board's
    own `Put` would deadlock). Notes emitted and not yet seen back are kept in `pending` (tagged with their `doc`) and
    count toward the color limit, so two concurrent notes can't both pass it. A pending entry ends when its record is
    seen in `Put`; when `emit` returns an error (removed at once); or when `Put` sees a new document (every entry of
    an older doc: the writer dropped it, or it is on a page that no longer counts). So a pending entry lives at most
    one document.
  - Reserved id: `operator` is Couch's own author id; the parsers refuse it as a viewer id.
- **Server** (`webcast.Server`): `POST <cast>/stroke`, `/note`, `/note-state` — the only body-reading routes. Same
  admission as the terminal pointer routes: JSON content type, an in-flight slot (`broadcast.MaxPointInFlight`), a
  rate budget (`broadcast.NewRateLimit`: strokes at `PointRatePerSec`, notes and states at `NoteRatePerSec`/`NoteBurst`),
  a read deadline, `MaxBytesReader`. Answers: 204; fixed refusal sentences (the terminal's where they fit: "This color
  already has 8 notes. Resolve one first.", "The page changed. Try again.", "Too many requests. Wait a moment.";
  plus "The cast has ended.", "The cast isn't live yet.", "That note isn't valid."). One mapping function
  `answer(err)`, as `drive.Answer`. Without a board the routes are 404. The CSP is unchanged (`connect-src 'self'`).
- **Wiring**: `CastConfig.Board *board.Board` → `NewServer(..., board)`; `Cast` exposes the board's subscription to
  Couch. `couchcmd` constructs a board only where a Live stream exists; on main today nothing feeds `Live`, and #458
  (pair:1, TL told them) wires the board as a Sink beside the Hub on its Tee and `emit` into the recorder queue. If
  #458 lands first I wire it on merge; if I land first, pair:1 does.
- **NoteCrops** (`notes.Model`, additive): `NoteCrops(st, ns []*recording.Note, hints) ([]Box, error)`. A region note
  crops `region` padded by the margin with no landmark snap (the viewer cropped by hand); an element note is as
  `Crops`. Clustering, clipping and hints apply to both. `Crops(st, ats, h)` becomes `NoteCrops` over anchor-only
  notes; `FakeModel` and `RunModelContract` gain the region case.
- **Viewer** (`webcast/web/`):
  - `collab.js`, shared with the terminal viewer (ARCH-DRY): `pointerColor`, `colorStore`, `renderSwatches`,
    `railAt`, `noteBytes`/`MAX_NOTE_BYTES` and `noteDialog`, moved out of `broadcast/web/{viewer,notes}.js`;
    `noteDialog.draft(payload)` takes an opaque payload merged into the note body (the terminal passes its cell and
    grid). `broadcast` exports the file (`broadcast.CollabJS`) and both servers serve it at `collab.js`.
  - `annotate.js` (new): pure helpers — stroke chunking into POSTs (50 ms, ≤ 64 points, dt relative to the chunk),
    the stroke store joined by id with fade (drawn 1.5 s after a stroke's last chunk; strokes in a reconnect's
    catch-up whose last point is > 1.5 s behind the newest record time are not drawn), live notes from the model
    (`seg >= docSeg`, open), a note's pin place (`boxOf(node)` while the node is drawn, else its recorded box; a
    region note at its region), region from a drag (normalized, min 8 px), and the identity store (id generated once
    per browser, name asked once, both in localStorage behind try/catch).
  - Shell: a mode button (watch / point / note, the terminal's MODE_TIPS idea) and the swatch rail docked to the box
    (`railAt`). Point mode captures pointer events on the overlay (`toPage`) and posts strokes; strokes are drawn into
    an overlay `<canvas>` scaled with the page. Note mode: tap/click → `nodeAt` → draft dialog; drag → region draft.
    Pins: a 💭 chip in the author's color at the element's top-right (region: an outlined box), redrawn each frame
    from `boxOf`/`fromPage`; tapping a pin opens the view dialog with Resolve. Touch: pointer events (tap to anchor,
    drag to region, `touch-action: none` on the overlay in point/note modes only, so watching still scrolls the page
    on iPad). Note text only ever as `textContent`.
- **Couch**: `Chrome ⏸ 💭N` (`StatusModel.WebNotes`, a span `RenderedStatusRow.WebNotes`); clicking it opens #447's
  notes menu fed with the web notes (mapped to the menu's `broadcast.Note` items: a console-local int id per note id,
  color index, text); click copies the text, right click resolves (`Board.SetState` as the operator, author id
  `operator`). The menu shows one source at a time (`notesMenuSource`: screen or web), so the two lists never mix.

## ARCH notes

- ARCH-FUNERAL: note and stroke records grow `notes.jsonl`/`pointer.jsonl` per viewer action, bounded by the rate
  limits (strokes 30/s cast-wide, ≤ 64 points; notes 2/s) for the cast's life, removed with the recording directory
  (#458's retention). The Hub keeps the current document's notes and strokes for catch-up and drops them at the next
  document's catch-up compaction. The board's `pending` entries die when their record is seen or the board closes;
  its `State` dies with the cast. localStorage: two keys per browser (`couch.viewerId`, `couch.viewerName`), plus the
  existing color key.
- ARCH-DRY: one palette (`broadcast.PointerPalette`), one set of note limits, one dialog and swatch module across
  both viewers; anchors made only by `notes.Model.Anchor` (no selector logic in JS).
- Identity is attribution, not authentication: anyone with the link can write as any id (atlas states it; #467 swaps
  in seat identity).

## Plan

- [ ] M1 — Go: `NoteCrops` + contract; `board` package (wire, Board, `OnDoc`, `Writer`, errors); server routes +
  admission + `answer`; `CastConfig.Board` wiring; atlas (format: chunked strokes, identity note, the routes,
  NoteCrops). Test strategy: `Parse*` over untrusted bodies → table + fuzz seeded with oversize, NaN/Inf, invalid
  UTF-8, unknown fields, the reserved id; Board rules → one test per named error, over the `segments` fixture fed
  through a Hub; the check/emit race → a test emit that orders a navigation before the note (the writer drops it,
  pending clears) and concurrent notes at the color limit (`-race`, exactly 8 land); server → route tests per answer.
- [ ] M2 — Viewer: `collab.js` extraction (terminal viewer tests stay green), `annotate.js` + node tests, shell UI,
  CSS. Headless e2e (`WEBCAST_LIVE_CHROME=1`): two viewer pages on the blog fixture through Server + Hub + Board
  (emit stamps and Puts into the Hub): viewer A draws a stroke and B sees it; A notes an element (pin over the
  element's box ± 2 px in B), B notes a region; a pause/resume keeps the pins; a navigation record clears them; the
  Hub's records validate and contain both notes and the stroke chunks.
- [ ] M3 — Couch: `💭N` after `Chrome ⏸`, the notes menu for web notes, copy and resolve; tests at the row/menu and
  console levels. Smoke checklist for the operator's desktop + iPad test (scheduled by the TL).

## Manual smoke (operator, after #458; TL schedules)

1. Start a cast on the blog, go live, open the link on desktop Chrome and on iPad Safari.
2. Each picks a different swatch; draw on the page — both see both strokes fade.
3. Desktop: note mode, click a heading, write a note — the iPad shows the 💭 on the heading; Couch shows `💭1`.
4. iPad: note mode, drag a region, write — both see the region; `💭2`.
5. Pause, scroll in Chrome, resume — pins stay on their elements. Navigate — pins clear; `💭` count clears.
6. `couch --broadcast-web-query <rec> notes` lists both notes with authors.

## Revisions

- 2026-10-12 (M1, after #458 landed; TL 110c6da0; M1 review BR-1). Delta from the Architecture above:
  - **Wiring:** no `CastConfig.Board` and nothing in `couchcmd`. `StartCast` makes the board itself for a recorded cast
    (`RecordingsDir` set) and tees it into the recording's sink after the directory and **before** the Hub (a viewer
    acts on what it was shown, which the board must know first). `Server.WithBoard(bd)` serves the routes. The board's
    emit is `webcast.Emitter.Emit(r, doc)`, which `Chrome` implements by sending `recorder.Emit{Record, Doc}` into the
    recorder's queue; the recorder closes the board with the recording, the cast when the browser ends.
  - **The ordering rule** is `recording.State.OnDoc(doc)` (not `board.OnDoc`): the recorder checks it at dequeue
    (`Emit.Doc > 0`), and `board.Writer` checks it in tests. The recorder drops silently; the Writer returns `ErrStale`.
  - **Couch API:** `Cast.Notes()` (the board's `Subscribe`) and `Cast.ResolveNote(id)` (as `board.Operator`).
  - **Stroke record id:** `<viewer id>.<stroke id>`, so two viewers' strokes never join.
  - **Answers:** 204; 409 "The page changed. Try again." / "This color already has 8 notes. Resolve one first." /
    "That note is gone." / "The cast isn't showing a page yet."; 410 "The cast has ended."; 429 "Too many requests.
    Wait a moment."; 400 "That isn't valid."; 503 "Notes aren't working in this cast." (`ErrBroken`: the board's state
    parted from the stream, logged once through `Board.Logf`). `Board.Put` never fails the Tee.
  - **Budgets:** strokes 60/s per cast (two viewers at the 50 ms cadence), notes and states 2/s, burst 5.
  - The board drops strokes from its own state (it never checks against them).
- 2026-10-12 (M3). The notes menu takes a source (`notesSource`: screen or web) instead of a second menu; web items are
  `broadcast.Note`s with ID = position + 1 and text `name: text`, and the console keeps each item's note id and plain
  text (what a click copies). The collaboration overlay, installed only during a screen broadcast before, is now
  installed while either the screen broadcast or a web cast needs it (`syncOverlay`), so the menu draws for a web cast
  alone. For #477 the viewer's drawing is exported pure over the model at t (`strokesAt`, `paintStrokes`, `pinLayer`,
  `liveNotes`).
- 2026-10-12 (M3 review, BR-21/22). A web note keeps one menu id while it is open (`webMenuIDs`, pruned each
  delivery), so a click on a row drawn before the list changed acts on that note or on nothing. Tests per declared
  behavior: the source switch and stable ids → `TestWebNotesMenuSourceAndStableIDs`; the shared overlay →
  `TestWebNotesMenuOutlivesTheScreenBroadcast`; 💭N on the row → `TestStatusRowWebNotes`; copy and resolve →
  `TestWebNotesInCouch`. Each fails with its rule mutated (detach removing the overlay unconditionally; no hover reset;
  positional ids).
