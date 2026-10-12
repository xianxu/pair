# #488 plan: notes are bounding boxes, in both casts

Issue: `workshop/issues/000488-…md`. Sources read: #470 (web notes), #447 (terminal notes),
`atlas/webcast-format.md` (note region, notes model, NoteCrops), #455/#483 issue bodies.

## Where things stand (measured from the tree, 10-12)

- **Shared:** `broadcast/web/collab.js` (the web cast symlinks it) owns colors, `poster`, and
  `noteDialog`; it has no geometry. Each cast builds its own draft payload.
- **Terminal:** a note is one cell (`broadcast.Note{ID, Col, Row, Color, Text}`, strict
  `ParseNote` on `cols, rows, col, row, color, text`). `viewer.js pointerMode` drafts on
  pointer-down, with no drag. The page draws nothing; Couch paints the 💭 into the frame
  (`DrawNotes`, on the Presenter goroutine) for the operator and every viewer. The notes
  live in memory and clear when pointing goes off. #455 (submission) isn't built.
- **Web:** `annotate.js` already does tap or drag (`MIN_REGION` 8 px, `regionOf`). The
  anchor is `v.nodeAt` at the tap or the region centre, which is the innermost element
  there. `recording.Note.Region` is optional, and `board.ParseNote` doesn't clamp it (#470
  BR-4, deferred). Pins outline only region notes.
- **Seam:** `notes.Model.NoteCrops`/`Crops`/`Hints` (#461) have **no production consumer**;
  `Hints` has no implementation. `query/` uses only Anchor/Context/Live.
- **Couch layout:** the child area (one slot's zellij session: agent pane, draft nvim, and
  optionally the review nvim and a right terminal) sits above Couch's own bottom row.
  The code calls that row the **tab bar** (`RenderStatusRow`, with chip spans and
  `ColumnToActor`). Pane geometry exists (`zellij action list-panes --json --geometry` →
  `zellijpane.Pane{X,Y,Columns,Rows}`, `RoleForPane`), but Couch doesn't query it for the
  screen. Nothing publishes the file or lines a nvim pane shows.

## Design

### Shared gesture → box (one source, ARCH-DRY)

`collab.js` gains a pure `noteBox(from, to, view, min)`, used by both casts. Points and
boxes are in the cast's own units.
- `view = {x, y, w, h}` is the visible screen: terminal `{0, 0, cols, rows}`, web
  `{scrollX, scrollY, vp.w, vp.h}` in top-page px.
- A drag (≥ `min` on either axis; terminal 1 cell, web `MIN_REGION`) → the spanned box,
  clamped to `view`.
- A tap → the **default box**, `view.w/10 × view.h/10` centred on the point and shifted
  (not shrunk) to stay inside `view`. The terminal rounds to whole cells, with a minimum
  of 2×1 so the pin fits.
- Tested once in node (`tests/broadcast-viewer/notes.test.mjs`), plus each cast's use of it.

### Terminal cast

- **Wire:** a note carries `box: [col, row, w, h]` in cells instead of `col, row`.
  `ParseNote` stays strict. The page and server ship in one binary, so there's no
  old-viewer window.
  - `broadcast.Note` gains `Box CellBox`. The pin cell (`Col, Row`) is **derived** from the
    box (its top-right corner, inside the box), never sent.
  - Visibility and guard rules apply to the pin cell. The box is clamped to the grid.
  - Overlap is refused on the pin cell, as today.
- **Viewer:** comment mode tracks pointer down/move/up like annotate.js.
  - It shows a page-side dashed `.drag-box` over `.xterm-screen` while dragging, mapped
    with `cellAt`.
  - On pointer-up it calls `noteBox` → `dialog.draft({cols, rows, box})`.
  - Pointer Events cover touch: tap = default box, drag = box.
- **Display (Couch, `DrawNotes`):** each open note's box gets an outline in its author's
  color, then the 💭 pin at the corner. Resolved notes are already gone from the list.
  This is drawn into the frame, so the operator and every viewer see the same thing.
  Outline style is **UI choice A** (below).
- **Anchor (spec 4b):** `broadcast.NoteAnchor{Kind, Path []string, Text string}`, the same
  shape as the web anchor's new `kind/path` (below). It is resolved in Couch when the note
  is accepted.
  - **Pure core** (`couchtty/noteanchor.go`):
    `AnchorFor(box CellBox, l ScreenLayout, frame terminal.Frame) NoteAnchor`.
    - `ScreenLayout` = the tab-bar row plus chips (from `RenderedStatusRow`), and
      `[]PaneRect{Role, Rect, Slot, Agent, File, TopLine}` for the slot in view.
    - It picks the component holding the box's centre and builds the path:
      - `tab bar › tabs: pair:1, pair:2 ●, Chrome ⏸`
      - `pane pair:3 › agent (Claude Code)`
      - `pane ops:0 › draft nvim › <file>, lines a–b`
      - `review › <doc>`
    - `Text` is the frame's cells under the box, trimmed and bounded (≤ 1 KiB, wide
      cells handled).
    - It is table-tested with synthetic frames.
  - **IO shell** (`couchtty/noteanchor_io.go`):
    - **Pane rects:** one `zellij --session <binding> action list-panes --json --geometry`,
      through the existing `layoutcmd` runtime seam, with a 300 ms timeout. The result is
      cached per slot and refreshed on slot switch and resize, so a note never waits on
      zellij in the common case.
    - **Fake:** the existing list-panes fake shape (`zellijpane` JSON fixtures) stands
      behind the same seam (ARCH-MOCK).
    - **Failure:** a failure or timeout degrades to `Kind: "screen"` plus the text under
      the box. The note still lands, and the anchor says it is partial.
    - **Draft and review file and lines:** pair's nvim (`nvim/init.lua`, `review.lua`)
      writes `{file, top, bottom}` to a per-scope `view.json` on
      `BufEnter/WinScrolled/VimResized`, debounced. Couch reads it when the anchor
      resolves. It is untrusted input: parsed into a typed value, size-capped, and on
      error the path ends at `draft nvim`.
    - **Lifecycle (ARCH-FUNERAL):** one file per scope, overwritten in place. It is
      removed with the scope dir (the existing artifactpath retention). This is the one
      new durable artifact.
  - **Layering:** pair writes `view.json` without knowing Couch; Couch reads it
    (couch → pair is allowed).
- **#455 seam:** a pure `broadcast.NoteCrop(n Note, cols, rows, margin int) CellBox`, the box
  plus one cell and clamped. With the anchor on the note, #455 reads both from
  `Session.Notes()`. No clustering.

### Web cast

- **Always a region:**
  - `annotate.js` uses `noteBox` with `view` from `vp` and the top scroll. A tap's region
    is the default box.
  - The anchor is `nodeAt` at the box's centre. That is the innermost element at that
    point, so the smallest element containing it.
  - `board.ParseNote` **requires** `region` and clamps it to the page box (closes #470
    BR-4).
  - `recording.Note.Region` stays `omitempty` for reading old recordings
    (ARCH-SECURE: older-version input). The board always writes it.
- **Pins:** every live note gets the outlined box (drop the `.region`-only rule); the chip
  goes at the top-right corner, as now.
- **Anchor shape:** `recording.NoteAt` gains `Kind` (`"element"`) and `Path` (landmark
  chain, e.g. `main › article › h2`) beside `Sel/Text`. The path comes from the
  existing `landmark` code.
  - Terminal and web anchors then both read `{kind, path, text}`. `atlas/webcast-format.md`
    documents the shared shape for #455 and #483.
- **NoteCrops (spec 6), the seam #483 crops from:**
  - `NoteCrops` = each note's region, padded by a **small margin** (16 px, was 48), with
    overlapping crops merged and the result clipped to the page.
  - **Retire** structure snapping (`snap`) and the per-URL `Hints` interface. Neither has
    a production consumer (Simplicity First).
  - Keep `landmark` for Context and the anchor path.
  - An old note without a region falls back to `at.box` padded, not snapped.
  - `crop_test.go` is rewritten to these rules.
  - **This is a shared surface with pair:2 (#483). It is coordinated through the hub
    before merge.**

## ARCH notes

- **PURE:** gesture→box, the pin cell, the anchor, crops, and the clamp are pure and
  table-tested. IO is only list-panes and the `view.json` read and write.
- **CONSTRAINTS:**
  - Note placement is a UI-response path. The anchor resolves from cached pane rects
    plus one small file read (target < 5 ms), with zellij refreshed off-path with a
    300 ms bound.
  - The text under the box is capped at 1 KiB.
  - Outline drawing is O(box perimeter) per note, at most 8 notes × 8 colors.
- **ORDER:**
  - Terminal notes already gate on the frame geometry at accept time (`landsNow`).
  - An anchor resolved against a stale pane cache, for example right after a pane
    resize, is the mis-ordering most likely to bite. The pane rects carry their
    geometry epoch, and a mismatch falls back to `screen`. That fallback is tested.
  - Holds no other cross-event state.
- **SECURE:** viewer-sent boxes are clamped server-side. `view.json` and old recordings
  are untrusted, as above.

## Milestones

- [ ] M1: shared `noteBox`, plus the web cast.
  - Always-region with the default box and the BR-4 clamp, outlined pins, and
    NoteAt `kind/path`.
  - NoteCrops simplified, with snap and Hints retired, after the hub's OK.
  - Node tests (mouse, touch, and tap/drag), and board, wire and crop Go tests.
  - Atlas update.
- [ ] M2: terminal boxes.
  - Wire `box` and the derived pin, the viewer drag and `.drag-box`, `DrawNotes` outline,
    and `NoteCrop`.
  - Node pointer-mode tests (mouse and touch) and Go notes/console tests.
- [ ] M3: terminal anchor.
  - `AnchorFor` (pure), the pane-rect cache over list-panes (fake), and nvim `view.json`.
  - Table tests and a console integration test.
  - Atlas update.
- [ ] Operator smoke test in both casts, then close.

## Open UI choices (for the operator, via the hub)

- **A. Terminal outline style:**
  - (a) tint only the box's edge cells with the author color (background), keeping the
    text;
  - (b) a faint tint over the whole box;
  - (c) corner marks only.
  - Recommendation: (a).
- **B. Tap vs drag threshold for touch:** the terminal uses 1 cell and the web 8 px for
  mouse. For a finger, should it be 2 cells or 16 px? Recommendation: yes, a larger
  threshold on `pointerType === 'touch'`.
- **C. Pin position:** top-right inside the box (web today) vs top-left. Recommendation:
  top-right in both.
- **D. "status line" in the issue's examples:** Couch's own bottom row is the tab bar
  (chips plus LIVE controls). Is "status line" that row's right side (LIVE ⏸ 👆 💭), or
  the nvim statusline of the minimized draft? The plan treats the nvim statusline as part
  of the draft pane, and the LIVE controls as `tab bar › controls`.
