# Broadcast Sticky Notes Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pointer-link viewers pin plain-text sticky notes to screen cells. Couch draws a 💭 there, so the operator and every viewer see it, and shows a `💭N` count in the tab bar. Any viewer can open a note; pointer-link viewers can resolve any note.

**Architecture:** The note store lives in the broadcast `Session`, in memory: a pure `NoteBoard` held **inside `PointerState`, under its one lock**: notes are the pointer link's collaboration layer, so pointing on/off and the board change atomically, and a note can never land after the clear that turning pointing off performs (ARCH-ORDER). Like the caps, the board closes a channel on every change. Two new pointer-link POST routes (`note`, `resolve`) write to it. Every event stream, view and pointer alike, sends the whole list as `event: notes` when it joins and after each change. Couch gets the same snapshot through `Config.OnNotes`, draws 💭 in the existing screen overlay (so it reaches viewers inside the frames) and counts the notes in the status row. The viewer page draws no 💭 itself: it keeps the list, turns a tap on a 💭 into "open this note", and owns the draft until Save.

**Tech Stack:** Go (`cmd/internal/broadcast`, `cmd/internal/couchtty`), vanilla ES modules for the viewer (`web/notes.js`, `web/viewer.js`), `node --test` driven by `TestViewerNode`.

**Spec:** `workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md` (the Decisions list and Done when).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Note` | `cmd/internal/broadcast/notes.go` | new |
| `NoteBoard` | `cmd/internal/broadcast/notes.go` | new |
| `ParseNote` / `ParseResolve` | `cmd/internal/broadcast/notes.go` | new |
| `NoteVisible` | `cmd/internal/broadcast/notes.go` | new |
| `DrawNotes` | `cmd/internal/broadcast/notes.go` | new |
| `RenderStatusRow` (`StatusModel.Notes`) | `cmd/internal/couchtty/reserve.go` | modified |
| `noteVisible` / `noteAt` / `noteCell` / `noteBytes` / `parseNotes` | `cmd/internal/broadcast/web/notes.js` | new |
| `pointerMode` (tap routing, comment mode) | `cmd/internal/broadcast/web/viewer.js` | modified |

- **Note:** `{ID int, Col, Row int, Color int, Text string}`. The cell is the left half of the two-column 💭. Color is the author's pointer palette index (#446); it is stored for the per-color limit and isn't drawn.
  - **Relationships:** N:1 with a broadcast's `NoteBoard`. Has no grid: a note stays at its cell whatever the grid becomes (operator decision, option A).
  - **Future extensions:** an anchor fingerprint (option C, deferred) would be one more field here.
- **NoteBoard:** ordered notes plus the next ID. `Add(n) (Note, error)` enforces 8 open notes per color, a 1 KB text cap and no overlap with another note's two cells. `Resolve(id) bool` is idempotent. `List()` returns a copy. Errors are fixed values: `ErrNoteColorFull`, `ErrNoteOverlap`.
  - **DRY rationale:** the one place the limits live. The server and the session never re-check them.
- **ParseNote / ParseResolve:** strict JSON body parsing in the style of `ParsePointBatch`: unknown fields refused, integers only, and errors that never echo the input. A note body is `{cols, rows, col, row, color, text}`. Text must be valid UTF-8, between 1 and 1024 bytes, with no control characters except `\n` and `\t`. `col` must leave room for the second cell (`col+1 < cols`), and the cell must not be guarded. A resolve body is `{id}`.
- **NoteVisible(n, cols, rows):** whether a note draws on this grid: both cells on the grid and not guarded (`guarded` from `marks.go`, so marks and notes share one rule).
- **DrawNotes(f, notes) Frame:** a copy of `f` with each visible note's two cells replaced by a width-2 `💭` and its continuation cell. A wide character cut in half on either side is blanked to a space, so no orphan half is left. Blanking keeps the cell's style and sets only content `" "` and width 1 (`uv.EmptyCell` would drop its background). A continuation is recognised as `Cells[i-1].Width == 2` (a zero cell is also the backend's ordinary blank, `terminal/render.go:129`). The checks read the **output** frame, so two nearby notes draw the same whatever their order. With no visible note it returns `f` uncopied, as `Marks.Overlay` does.
- **StatusModel.Notes:** an open-note count. While live and when it is above 0, the row draws ` 💭N` right after 👽, with no click target (rule 8).
- **notes.js helpers:** the viewer's mirror of `NoteVisible` and its hit-test. `noteAt` hits both cells of a 💭. `noteCell` snaps a tap on the last column to `cols-2`. `noteBytes` counts UTF-8 bytes so the 1 KB cap matches Go's. `parseNotes` keeps only well-formed entries. The guard width arrives in the notes event (`guard`), so JS never hardcodes `StatusGuardCols`.
  - **DRY rationale:** the visibility rule necessarily exists in Go (drawing) and in JS (hit-testing). The guard width is the only constant they share, and it crosses the wire rather than being copied.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `PointerState` notes (`pointer_notes.go`) | `cmd/internal/broadcast/pointer.go`, `cmd/internal/broadcast/pointer_notes.go` | modified | the board under the pointer lock, notes change channel, note rate limit |
| `Server` routes `note`, `resolve`; `event: notes` | `cmd/internal/broadcast/server.go` | modified | HTTP and the event stream |
| `Session.acceptNote` / `Config.OnNotes` | `cmd/internal/broadcast/session.go` | modified | the operator's current frame; Couch's callback |
| `Console` notes (`console_notes.go`) | `cmd/internal/couchtty/console_notes.go` | new | Run loop, Presenter overlay, status row |
| `noteDialog` | `cmd/internal/broadcast/web/notes.js` | new | DOM, `fetch` |

- **PointerState notes:** `PointerState` gains `board NoteBoard`, `notesVersion uint64`, `notesChanged chan struct{}` (closed and replaced on every board change) and `noteLimit *RateLimit` (2/s, burst 5, shared by `note` and `resolve`), all under the existing leaf lock `p.mu`. New methods live in `pointer_notes.go`: `Notes() (notes []Note, version uint64, changed <-chan struct{})`, `addNote(n Note, gen uint64) (Note, error)`, `resolveNote(id int) error`, `allowNote(now) bool`. `addNote` refuses with `ErrPointerOff` unless `p.on && p.gen == gen`, and `resolveNote` unless `p.on`, each checked **in the same critical section** as the change. `gen` is the flip generation the caller read *before* its frame check (the `offIfGeneration` pattern), so a note checked against a screen from before an off→on flip is refused, and the frame check itself (`Hub.Current`, a round-trip through the hub loop, which can take `p.mu` via `OnPointerHidden`) runs with **no lock held**. Rule, stated in `pointer.go`'s lock comment: nothing called while `p.mu` is held takes `p.mu` or waits on another goroutine. Every flip to off (`set(false)`, `offIfGeneration`, `stop`) empties the board and bumps `notesVersion` in that same critical section, so "off" and "no notes" are one state, not two that must be kept in step. The version never goes back, across flips too, which is what lets the console drop any snapshot from before a clear. ARCH-FUNERAL: in memory only, bounded at 6 colors × 8 notes × 1 KB ≈ 48 KB, emptied on every pointing-off and gone with the session.
- **Server:** `ServeHTTP` dispatches POSTs on the pointer token to `point`, `note` or `resolve` through one shared `readPointerBody`. The view token still never reads a body (#395). `readPointerBody(w, r, allow func(time.Time) bool, max int64) (body []byte, release func(), ok bool)` keeps today's order for all three routes (pointing on → content type → in-flight slot → rate check → read deadline → body), and the caller defers `release`, so the slot is held while the request is read **and handled**, as `point` does now. `point` keeps its 30/s limit; `note` and `resolve` use the note limit. `MaxNoteBody = 8 KB` (a 1 KB note escaped as `\uXXXX` is about 6 KB). A nil `Pointer` (existing tests' `ServerOptions`) serves no note routes (404) and sends no notes event. `resolve` of an unknown id answers 204, the idempotent case. A refused note gets a fixed human sentence as its body (`noteRefusal`), which the dialog shows as plain text. `events` sends `event: notes` to **every** stream: on join, then on each `changed`.
- **Session.acceptNote(n) error:** a note lands only on what the operator sees now: pointing on, the same grid, a public frame and the pointer marker shown (the checks `acceptPoint` makes, factored into `landsNow(cols, rows)`, which checks only the frame: grid, public, marker; `acceptPoint` keeps its own `On()` check). The order is: `gen := s.pointer.generation()` → `landsNow` with no lock held → `s.pointer.addNote(n, gen)`, which refuses if pointing is off or flipped since `gen`. `resolve` needs only pointing on (same lock). After each change, and after `DisablePointer`/`pointerHidden` flip and clear, it calls `cfg.OnNotes(s, snapshot, version)` on the calling goroutine with no lock held.
- **Console:** `onNotes(s *Session, notes, version)` hops onto the Run loop and keeps the snapshot only if (1) the broadcast is still live **and** `s` is the current session (as `applyPoints` checks the phase), so a request in flight at Stop, or a stale snapshot from the previous broadcast, can't bring notes back, and (2) its version is newer than the one it holds (two request goroutines can deliver out of order, ARCH-ORDER). The snapshot sits in `pointerNotes{mu, notes, version}`, a leaf lock like `pmarks.mu`, because the Run loop writes it and the Presenter goroutine reads it in the overlay. The status row's count reads the same struct, so the 💭s and `💭N` can't disagree. Its own pointer-phase transitions **resync from the session** instead of guessing: `pointerOff` (click and watch) and turning pointing on both set the snapshot to `s.Notes()` and the held version to that version, so a snapshot that was in flight from before the clear (an older version) is dropped whether it arrives while off or after re-enable. The session's flip always happens before the console's transition (`DisablePointer` runs first on a click; `pointerHidden` flips before calling `OnPointerOff`). `resetPointer` clears at broadcast end. `onNotes` is also ignored while the console's phase is off. It then repaints. The overlay becomes `screenOverlay`: `DrawNotes` first, then marks, both only on public frames. `resetPointer` also clears the notes when the broadcast ends.
- **noteDialog:** one fixed-size sticky-note dialog in two modes: draft (textarea, byte counter, Save and Cancel) and view (text through `textContent`, Resolve when allowed, Close). Save posts `note`. A 204 closes the dialog; any other status shows the response text. The draft exists only in the textarea until Save (rule 11).

### Decisions this plan makes (beyond the Spec)

- **Pointing off clears the collaboration layer (operator, 10-11):** turning pointing off, by the operator's click or by the 👆-hidden watch, deletes every note, as it clears marks. `Session` empties the board in `DisablePointer` and in `pointerHidden` when it actually flips, bumps the version and calls `OnNotes`; every stream gets an empty list. The console also draws notes only while its pointer phase is on, so the 💭s and `💭N` go away on the same Run-loop step as the marks. `note` and `resolve` answer 403 while off. Turning pointing back on starts with no notes.
- **Refusal text** is a closed set of server sentences: "Pointing is off.", "This color already has 8 notes. Resolve one first.", "A note is already there.", "The screen changed. Try again.", "Too many requests. Wait a moment.", "That note isn't valid." Like `endReasons`, the set is closed and never echoes input.
- **Mode switch** is one button (`#mode`) below `#swatches` in `#rail`, showing 👆 or 💭 for the current mode. It is shown only on a pointer page while pointing is on, and it resets to pointer mode when pointing turns off.
- **Known overlaps:** the `💭N` count sits outside `StatusGuardCols`, so a note or a mark can cover it on the last row; only the control cells are guarded, as the Spec says. While a private frame is up Couch draws no 💭, but the viewer's hit-test still opens a note at its cell; acceptable, since the switcher is momentary.
- **Two-boundary split:** M1 delivers the whole Couch side, testable with `curl`. M2 is the viewer page. Each closes on its own review.

## Chunk 1: M1 — notes in Couch

### Task 1: Note model and board

**Files:**
- Create: `cmd/internal/broadcast/notes.go`
- Test: `cmd/internal/broadcast/notes_test.go`

- [ ] **Step 1: Write the failing tests** (table-driven):

```go
func TestNoteBoardLimits(t *testing.T) {
	var b NoteBoard
	for i := 0; i < MaxNotesPerColor; i++ {
		if _, err := b.Add(Note{Col: i * 2, Row: 0, Color: 3, Text: "x"}); err != nil {
			t.Fatalf("note %d: %v", i, err)
		}
	}
	if _, err := b.Add(Note{Col: 40, Row: 0, Color: 3, Text: "x"}); err != ErrNoteColorFull {
		t.Fatalf("9th note in one color: %v, want ErrNoteColorFull", err)
	}
	if _, err := b.Add(Note{Col: 40, Row: 0, Color: 4, Text: "x"}); err != nil {
		t.Fatalf("another color still has room: %v", err)
	}
}

func TestNoteBoardOverlap(t *testing.T) {
	var b NoteBoard
	b.Add(Note{Col: 10, Row: 5, Text: "a"})
	for _, col := range []int{9, 10, 11} {
		if _, err := b.Add(Note{Col: col, Row: 5, Text: "b"}); err != ErrNoteOverlap {
			t.Errorf("col %d: %v, want ErrNoteOverlap", col, err)
		}
	}
	if _, err := b.Add(Note{Col: 12, Row: 5, Text: "b"}); err != nil {
		t.Errorf("adjacent note: %v", err)
	}
}

func TestNoteBoardResolveIdempotent(t *testing.T) {
	var b NoteBoard
	n, _ := b.Add(Note{Col: 0, Row: 0, Text: "a"})
	if !b.Resolve(n.ID) || b.Resolve(n.ID) || b.Resolve(999) {
		t.Fatal("resolve reports true once, then false")
	}
	if len(b.List()) != 0 {
		t.Fatal("resolved note still listed")
	}
}
```

Also: IDs are unique and increasing, including after a resolve, and `List` returns a copy (mutating it doesn't change the board).

- [ ] **Step 2:** `go test ./cmd/internal/broadcast -run NoteBoard`. Expect a FAIL (undefined).
- [ ] **Step 3: Implement** `Note`, `NoteBoard`, `MaxNotesPerColor = 8`, `MaxNoteBytes = 1024`, `ErrNoteColorFull`, `ErrNoteOverlap`. Overlap: same row and `|a.Col-b.Col| < 2`.
- [ ] **Step 4:** run the tests. Expect PASS.
- [ ] **Step 5:** commit `#447 M1: note board with per-color and overlap limits`.

### Task 2: Strict parsing

**Files:** Modify `cmd/internal/broadcast/notes.go`; Test `notes_test.go`

- [ ] **Step 1: Failing tests:** `ParseNote` accepts `{"cols":80,"rows":24,"col":3,"row":4,"color":2,"text":"hi\nthere"}`. It refuses (each a separate case, each returning a fixed error): an unknown field; a string col; a fractional color; color 6 and -1; `col = cols-1` (no room for the second cell); a guarded cell (`row = rows-1, col = 0`); empty text; text of 1025 bytes (`strings.Repeat("é", 513)`); invalid UTF-8; text containing `\x1b` or U+202E; trailing data after the object. 1024 bytes exactly is accepted. `ParseResolve` accepts `{"id":3}` and refuses `{"id":"3"}`, `{}` and extra fields. No error message contains input bytes (assert `!strings.Contains(err.Error(), "hi")`).
- [ ] **Step 2:** run. Expect FAIL.
- [ ] **Step 3: Implement**, mirroring `ParsePointBatch` (`DisallowUnknownFields`, pointer fields to detect absence, `dec.Token() == io.EOF`). The grid limits reuse `maxPointGrid`; the color check reuses `PointerColors`. Text: `utf8.ValidString`, length, and a rune scan refusing control characters except `\n` and `\t` **and format characters** (`unicode.Is(unicode.Cf, r)`: bidi overrides, zero-width), so the dialog can't show spoofed text.
- [ ] **Step 3b: Fuzz** `FuzzParseNote`, seeded with every refusal case above plus the accepted one: never panics, any accepted note satisfies every rule (re-check the invariants independently), and no error text contains input bytes.
- [ ] **Step 4:** PASS. **Step 5:** commit `#447 M1: strict note and resolve parsing`.

### Task 3: Visibility and drawing

**Files:** Modify `notes.go`; Test `notes_test.go`

- [ ] **Step 1: Failing tests** for `NoteVisible`: visible inside the grid; hidden when `col+1 >= cols` or `row >= rows` after a shrink; hidden when the cell becomes guarded because the grid shrank to put it on the last row. For `DrawNotes` on a frame built with the existing test helpers (see `marks_test.go` for how frames are made):
  - the note's cell holds `💭`, width 2, and the next cell is the zero continuation;
  - a wide char whose left half sat at `col+1` leaves its right half at `col+2` blanked to a space;
  - a wide char whose left half sat at `col-1` (so `col` was its continuation) is blanked to a space at `col-1`;
  - no visible notes → the same frame value, uncopied (compare `&f.Cells[0]`);
  - the input frame is never mutated;
  - a cut wide character's blank keeps its background style;
  - two notes two columns apart over wide text give the same frame in either order;
  - **property** (seeded `math/rand`, 500 cases): random rows mixing narrow and wide characters × random note sets → the result is the same for every note order, no wide character is left without its other half, and cells outside notes' and their neighbours' columns are unchanged.
- [ ] **Step 2:** FAIL. **Step 3: Implement** `DrawNotes` per Core concepts. **Step 4:** PASS. **Step 5:** commit `#447 M1: draw notes as 💭 on a frame`.

### Task 4: Notes in PointerState and the server routes

**Files:**
- Create: `cmd/internal/broadcast/pointer_notes.go`, `cmd/internal/broadcast/pointer_notes_test.go`
- Modify: `cmd/internal/broadcast/pointer.go` (fields; clear on every flip to off)
- Modify: `cmd/internal/broadcast/server.go`, `cmd/internal/broadcast/server_test.go`

- [ ] **Step 1: Failing tests.**
  - `pointer_notes_test.go`:
    - `addNote` bumps the version and closes the channel `Notes` returned earlier; a refused add changes neither; `resolveNote` of an unknown id changes neither.
    - `addNote`/`resolveNote` with pointing off → `ErrPointerOff`.
    - Every flip to off (`set(false)`, `offIfGeneration` that flips, `stop`) empties the board, bumps the version and closes `notesChanged`; `offIfGeneration` with a stale generation changes nothing; `set(true)` afterwards starts empty with a still-higher version.
    - **Injected ordering** (the generation is the seam): read `gen`; flip off (`set(false, nil)`); `addNote(n, gen)` → `ErrPointerOff`, board empty. Read `gen`; flip off then on; `addNote(n, gen)` → refused (stale generation), board empty. `addNote` succeeds, then a flip off → board empty, version past the add's. With the flip and the clear in one critical section and the add conditional on `gen`, these are every interleaving of one add and one flip.
    - A `-race` stress: 4 goroutines adding and resolving while one flips on/off 1000 times; after the final off, `Notes()` is empty.
  - `server_test.go`, using the existing httptest helpers for the pointer token:
    - (a) POST `note` on the **view** token → 405, and the handler never reads the body (reuse the existing "view link never reads a body" test's body-reader probe);
    - (b) with pointing off → 403 "Pointing is off." for both `note` and `resolve`;
    - (c) a well-formed note → 204, and an open pointer stream **and** an open view stream each receive `event: notes` whose data lists it with `guard` = `StatusGuardCols`;
    - (d) a 9th note in one color → 409 with the fixed sentence;
    - (e) a malformed body → 400 "That note isn't valid.";
    - (f) `resolve` → 204, and both streams get a list without the note;
    - (g) a new stream gets the current list right after connecting;
    - (h) the shared rate limit: a burst past 5 → 429;
    - (i) `resolve` of an unknown id → 204;
    - (j) a `ServerOptions` without `Pointer` serves 404 on `note`/`resolve` and sends no notes event; existing `point` tests still pass unchanged (rate check before body read, slot held through handling).
  - The test server's `OnNote`/`OnResolve` fakes call `Pointer.addNote`/`Pointer.resolveNote`, so the limits in (d) and the events in (c)/(f) are the real ones.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
  - The `PointerState` notes as in Core concepts; `set`, `offIfGeneration` and `stop` call the shared `clearNotesLocked()`.
  - `ServerOptions` gains `OnNote func(Note) error` and `OnResolve func(id int) error`; the board is reached through the existing `Pointer`.
  - Factor `point`'s guard sequence into `readPointerBody` (returns the body and a `release` the caller defers; takes the route's rate check and body cap) and use it in all three handlers. Add `MaxNoteBody = 8 << 10`.
  - `noteRefusal(err) (status int, text string)` maps errors to the fixed sentences.
  - In `events`, add `notesChanged` alongside `capsChanged`, plus `sendNotes()` marshalling `{guard, notes:[{id,col,row,color,text}]}`; call it on join for every stream.
- [ ] **Step 4:** `go test ./cmd/internal/broadcast`. Expect PASS, and existing tests still pass.
- [ ] **Step 5:** commit `#447 M1: note and resolve routes; notes event to every stream`.

### Task 5: Session wiring

**Files:** Modify `cmd/internal/broadcast/session.go`, `cmd/internal/broadcast/session_test.go`

- [ ] **Step 1: Failing tests** (with the fake tunnel `session_test.go` already uses):
  - a note posted while the operator's current frame is private, has another grid, or lacks the pointer marker → 409 "The screen changed. Try again.", and `OnNotes` is not called;
  - a landed note → `OnNotes` is called once with the snapshot and version 1, then resolve → version 2 and an empty list;
  - resolve with pointing off → 403 and no `OnNotes`;
  - **no deadlock with the hub loop:** on a real hub (as `session_test.go` builds), post notes in a loop while the pointer grace timer fires `OnPointerHidden` (hide the 👆 marker in offered frames); the test finishes within a timeout and, after the hidden-flip, the board is empty;
  - with two notes open, `DisablePointer` → `OnNotes` with an empty list and a newer version, and both streams get an empty `notes` event; the same through the 👆-hidden watch (`pointerHidden`); a stale watch report (older generation) clears nothing; `EnablePointer` again starts empty.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
  - Factor `landsNow(cols, rows) bool` out of `acceptPoint`.
  - `DisablePointer` and `pointerHidden`, when they flip, call `OnNotes` with the (now empty) snapshot; the clearing itself already happened inside the flip.
  - Add `acceptNote(n Note) error` and `resolveNote(id int) error`.
  - Add `Config.OnNotes func(*Session, []Note, uint64)`, documented like `OnPoints` (request goroutine, no session lock held).
  - `Start` passes both callbacks to `NewServer`.
- [ ] **Step 4:** PASS. **Step 5:** commit `#447 M1: session accepts notes on the live screen`.

### Task 6: Couch draws notes and counts them

**Files:**
- Create: `cmd/internal/couchtty/console_notes.go`, `cmd/internal/couchtty/console_notes_test.go`
- Modify: `cmd/internal/couchtty/console_pointer.go` (overlay → `screenOverlay`; `resetPointer` clears notes), `console_broadcast.go:145-150,197` (wire `OnNotes`, install `screenOverlay`), `console_presentation.go:8` (`Notes:` in the model), `reserve.go` (`StatusModel.Notes`, draw ` 💭N`), `reserve_test.go`

- [ ] **Step 1: Failing tests.**
  - `reserve_test.go`: live with `Notes: 3` draws `LIVE ⏸ 👆 👽 💭3` at the start of the row; `Notes: 0` draws no 💭; not live with `Notes: 3` draws no 💭; at a narrow width it clips through `appendText` like every other segment; `Pointer` and `Remote` spans are unchanged.
  - `console_notes_test.go`, with the console test harness `console_pointer_test.go` uses: `onNotes(v2)` then `onNotes(v1)` keeps v2; a public frame gets the 💭 and a private frame doesn't; after the broadcast stops the overlay draws nothing and the count is 0; an `onNotes` delivered **after** stop, or naming a previous session, is ignored; after `pointerOff` (click or watch) the overlay draws no 💭 and the count is 0, and an `onNotes` arriving while pointing is off is ignored; **turning pointing back on shows no 💭 and a count of 0, and a pre-off snapshot (older version) delivered after re-enable is dropped**; the overlay and the count read the same snapshot. Run with `-race`.
- [ ] **Step 2:** FAIL. **Step 3: Implement.** **Step 4:** `go test ./cmd/internal/couchtty ./cmd/internal/broadcast`. Expect PASS.
- [ ] **Step 5:** commit `#447 M1: Couch draws 💭 notes and counts them in the tab bar`.

### Task 7: M1 verification and boundary

- [ ] Run the full suite per memory (`make -k test > $TMPDIR/447-m1.log 2>&1`, scratchpad-TMPDIR `test-changelog`, `go test ./...`). Compare failures against main's known three.
- [ ] Curl smoke on a local-only broadcast: post a note through the pointer link and see 💭 on the operator's screen and `💭1` in the tab bar; resolve it and see both go away. Record in `## Log`.
- [ ] Atlas: the broadcast page gains the notes routes, the `notes` event and the lifetime line.
- [ ] `sdlc milestone-close --issue 447 --milestone M1`.

## Chunk 2: M2 — the viewer page

### Task 8: notes.js pure helpers

**Files:**
- Create: `cmd/internal/broadcast/web/notes.js`, `tests/broadcast-viewer/notes.test.mjs`
- Modify: `server.go` (`assets["notes.js"]`), `server_test.go` (served with a JS content type)

- [ ] **Step 1: Failing node tests:**
  - `noteAt` (signature in Core concepts: it takes the parsed list, guard included) hits `col` and `col+1`, misses `col+2`, misses hidden notes;
  - `noteCell(79, 80)` gives 78, and `noteCell(5, 80)` gives 5;
  - `noteBytes('é')` is 2, so 512 × 'é' is allowed (1024 bytes) and 513 isn't;
  - `parseNotes` drops entries with non-integer fields or a non-string text, and returns `{guard: 0, notes: []}` for junk.
- [ ] **Step 2:** `node --test tests/broadcast-viewer/notes.test.mjs`. Expect FAIL. **Step 3:** implement. **Step 4:** PASS. **Step 5:** commit `#447 M2: viewer note helpers`.

### Task 9: Tap routing and comment mode in pointerMode

**Files:** Modify `web/viewer.js` (`pointerMode`, `connect` gains a `notes` event dep, `start` wiring), `tests/broadcast-viewer/pointermode.test.mjs`, `tests/broadcast-viewer/connect.test.mjs`

- [ ] **Step 1: Failing tests** (extend the `fakes()` harness with `notes` and `dialog` fakes):
  - pointing off (a view page): a tap on a 💭 calls `dialog.view(note, {resolvable: false})` and posts nothing;
  - pointer mode, pointing on: a tap on a 💭 opens it (`resolvable: true`) and posts no point; a tap elsewhere strokes as before;
  - comment mode: a tap on an empty cell calls `dialog.draft([col,row])` with the snapped cell, and **nothing is posted** (rule 11); a tap on a 💭 opens it rather than drafting;
  - comment mode on a guarded cell (last row, inside the guard) opens no draft;
  - `setOn(false)` resets to pointer mode and closes a draft;
  - `connect` forwards `event: notes` to `deps.notes`, and its absence is tolerated.
- [ ] **Step 2:** FAIL. **Step 3: Implement.** `pointerMode` gains `deps.notes()` (returns the parsed list plus the guard) and `deps.dialog`, `mode` state, and `setMode`. The `pointerdown` order is: cell → `noteAt` → open; else if not on → return; else if comment → draft; else stroke.
- [ ] **Step 4:** PASS. **Step 5:** commit `#447 M2: tap routing, comment mode`.

### Task 10: The dialog, the mode button and styles

**Files:** Modify `web/notes.js` (`noteDialog`), `web/index.html` (`#mode` in `#rail` after `#swatches`; `#note` dialog), `web/viewer.css`, `web/viewer.js` (`start`); Test `tests/broadcast-viewer/notes.test.mjs`

- [ ] **Step 1: Failing tests** for `noteDialog` with a fake element factory and a fake `post`:
  - `draft` shows the textarea and Save; Save posts `note` with `{cols, rows, col, row, color, text}` and closes on 204; a 409 response shows its body text via `textContent` and keeps the draft;
  - Save is disabled at 0 bytes and over 1024 bytes, and the counter shows bytes;
  - `view` renders text with `<b>x</b>` as literal characters (assert `textContent`, and that `innerHTML` is never assigned);
  - Resolve is present only when `resolvable` and posts `resolve {id}`;
  - Cancel discards the draft without a request;
  - an open view dialog closes when a `notes` event no longer lists its note (resolved elsewhere).
- [ ] **Step 2:** FAIL. **Step 3: Implement.** CSS: a fixed 280×200px sticky note (`min(280px, calc(100vw - 32px))` wide on phones), a yellow `#fff59d` background, dark text, a soft shadow, centered over the stage; `#mode` styled like the swatch group (same backing, `pointer-events: auto` on the button only).
- [ ] **Step 4:** `go test ./cmd/internal/broadcast -run TestViewerNode`. Expect PASS.
- [ ] **Step 5:** commit `#447 M2: sticky-note dialog and mode switch`.

### Task 11: Live check and close

- [ ] `make build` in this slot, then start a broadcast with pointing on. Ask the operator to run the Done-when walk with two viewers (desktop and phone width) plus one view-only viewer: place, open, resolve, the count, a draft never posted (check the browser network tab), the 1 KB and 9th-note refusals.
- [ ] Atlas update for the viewer side (mode switch, dialog).
- [ ] Full suite as in Task 7; then `sdlc close --issue 447 --verified '<evidence>'`.

## Chunk 3: M3 — visibility (scope revision, 2026-10-11)

The operator's smoke: a 💭 is easy to miss on a full screen. Each 💭 gets its author's color as a chip and a round glow, flashing on arrival and every 15s until resolved. The viewer's hint bar becomes tooltips.

| Name | Lives in | Status |
|------|----------|--------|
| `paletteTint` | `cmd/internal/broadcast/marks.go` | new (factored out of `Marks.tint`) |
| `Note.At` | `cmd/internal/broadcast/notes.go` | modified (save time, `json:"-"`) |
| `NoteStyle` / `noteGlow` / `NoteNextChange` | `cmd/internal/broadcast/notehalo.go` | new |
| `DrawNotes` | `cmd/internal/broadcast/notes.go` | modified (takes a `NoteStyle`; draws chip and glow) |
| viewer rail tooltips | `cmd/internal/broadcast/web/viewer.js`, `index.html`, `viewer.css` | modified (hint bar and band removed) |

- **paletteTint(c, t, bg, blend) color.Color:** the one place a palette color is weakened. `t` runs from 0 (the full color: truecolor RGB, or the xterm index) to 1 (the background). Above 0 it is blended into `bg` with truecolor, or else picked from the color's five-step ladder by `t`. `Marks.tint` becomes "age → t → paletteTint", so marks and notes fade the same way (ARCH-DRY). The existing mark tests pin that marks are unchanged.
- **Glow geometry:** the glow is centered on the 💭's two cells, with rows −2..+2. Columns are scaled by the cell aspect (2:1) so the glow looks round: a cell is in it when `(dx/2)² + dy² ≤ 2.5²`, with dx measured from the nearer chip cell. Resting `t` by distance: the chip 0 (full color), `d ≤ 1.5` 0.5, otherwise 0.75. Where glows overlap, the stronger wins. The glow never covers the guarded cells and never paints another note's chip.
- **Flash:** a note flashes at full color when it is saved (its arrival flash), and every open note flashes together on one shared beat every 15s (`now mod 15s`), until it is resolved. A flash holds 1.5s, then steps down to rest over 0.5s in 10 steps (`MarkFade`'s shape); arrival and beat take the stronger. A hovered note (M4) is held at the flash level. *Revised at the M3 review:* per-note phases cost about 11 repaints per note per 15s, about 35 full repaints a second to every viewer at the 48-note limit; one shared beat bounds it at about 11 per 15s plus 11 per arrival (saves are rate limited at 2/s), whatever the number of notes (ARCH-CONSTRAINTS).
- **NoteNextChange(notes, now)** gives the next instant the drawing changes (a flash start, a fade step, a flash end), so the console schedules exactly one repaint timer while notes are open, as `Marks.NextChange` does. Its cost is one timer, plus about 11 repaints per beat and per arrival.
- **The console** passes `NoteStyle{BG, Blend, Hover, Now}` from the same palette facts marks use, and adds a notes timer next to `pmarks.timer`, stopped in `resetPointer` and on pointer-off.
- **Viewer:** `#controls`, `#hint` and `band()` go. The stage takes the whole window again (as it did before #446). The swatches keep `aria-label` and gain `title`, "Pointer color N". `#mode` gets `title` and `aria-label`: "Pointer mode: tap or drag to point" or "Comment mode: tap to place a note". When pointing is off, the rail shows nothing, which already says it.

### Task 12: paletteTint and the glow

**Files:** `marks.go`, `notehalo.go` (new), `notes.go`; tests `marks_test.go`, `notehalo_test.go` (new), `notes_test.go`.

- [ ] **Step 1: Failing tests.**
  - `paletteTint` at strength 1 equals today's mark colors; at 0.5 with blend it is the midpoint RGB, and without blend a ladder index.
  - The mark-fade tests stay green unchanged, which pins the refactor.
  - Glow cells for a note at (10,5) on 80×24: the exact cell set, the strengths per ring, and the guard excluded when the note is near the last row.
  - Overlap: the stronger strength wins, and another note's chip is untouched.
  - Flash timing: at ages 0, 1.4s, 1.6s, 1.95s, 2.1s, 15.0s and 16.6s the strength is full, full, fading, fading, rest, full and rest.
  - `NoteNextChange` returns the right boundary for each of those ages, and false with no notes.
  - `DrawNotes` keeps the M1 properties: order-independent, no orphan halves, input not mutated, uncopied with nothing visible. The chip cells get the full color. Glow cells change only `Style.Bg`.
- [ ] **Step 2:** FAIL. **Step 3: Implement.** **Step 4:** `go test ./cmd/internal/broadcast` PASS. **Step 5:** commit `#447 M3: note chip, round glow and flash timing`.

### Task 13: Console timing

**Files:** `couchtty/console_notes.go`, `console_pointer.go`; test `console_notes_test.go`.

- [ ] **Step 1: Failing tests.** A saved note's chip cells show the author's color on the operator's screen (truecolor fixture). The timer repaints at the flash boundary: drive `c.now` with the console's test clock, if the harness has one, else assert the scheduled duration through a seam. After pointer-off or a broadcast stop, no notes timer is left (`pnotes.timer == nil`).
- [ ] **Step 2:** FAIL. **Step 3: Implement** `refreshNotes` (repaint, then schedule from `NoteNextChange`), mirroring `refreshMarks`. **Step 4:** PASS with `-race`. **Step 5:** commit.

### Task 14: Tooltips instead of the hint bar

**Files:** `web/viewer.js`, `web/index.html`, `web/viewer.css`; tests `tests/broadcast-viewer/pointermode.test.mjs`, `rail.test.mjs`.

- [ ] **Step 1: Failing tests.**
  - `pointerMode` no longer takes a `hint`. The mode button's `show` receives the mode and the tooltip text.
  - `renderSwatches` sets `title`.
  - The page has no `#hint`/`#controls`, and the stage is not lifted (a `start` wiring assertion through the existing fakes, or a page-asset check in `server_test.go`).
- [ ] **Step 2:** FAIL. **Step 3: Implement.** **Step 4:** node tests and `TestViewerNode` PASS. Check in headless Chrome over CDP (the M2 smoke script) that the screen fills the window at 1200×800 and 390×844. **Step 5:** commit.

### Task 15: M3 boundary

- [ ] Atlas (broadcast page: chip, glow, flash, tooltips); README sentence on the glow; full suite as in Task 7; `sdlc milestone-close --issue 447 --milestone M3`.

## Chunk 4: M4 — notes in the terminal

| Name | Lives in | Status |
|------|----------|--------|
| `RenderNotesMenu` / `NotesMenu` | `cmd/internal/couchtty/notes_menu.go` | new (pure) |
| `RenderedStatusRow.Notes` | `cmd/internal/couchtty/reserve.go` | modified (span of `💭N`) |
| console notes menu state and routing | `cmd/internal/couchtty/console_notes.go`, `terminal_input.go` | modified |

- **RenderNotesMenu(notes, cols, rows) NotesMenu** is pure; `NotesMenu.Draw(f, hover, style)` draws it. It returns the box's placement, its lines, and each item's row span and note ID; `Draw` renders the lines onto a frame.
  - **Placement:** the box sits bottom-right, directly above the status row. Its width is `min(48, cols − 2)`. Its height fits as many items as there is room for, up to `rows − 3`, with a final "+k more" line when they don't all fit.
  - **Header:** `Notes` on the left and a close mark `×` on the right.
  - **Items:** each item is a two-cell chip in the author's color (background), then the text, wrapped to the box width. An item shows at most 3 lines, the last ending in `…` if cut; a copy still takes the full text.
  - **Hover:** the hovered item's rows are drawn on a lighter background (`notesMenuHover`).
  - **Text safety:** text goes through `rowtext.Sanitize`. It is already plain text, but this is defence in depth, since it is drawn into the operator's terminal.
- **The box is an overlay.** `screenOverlay` draws notes, then marks, then the menu (`menuOverlay`, its own step, last), on public frames only, so viewers see the menu as they see the rest of the operator's screen. It never covers the status row.
- **Routing, in `routeMouseEvent`, before the child gets the event:**
  - A left click on the `💭N` span toggles the menu.
  - While the menu is open, every mouse event inside its rectangle (press, motion, wheel) is Couch's and never reaches the child.
    - Motion over an item sets `hover` to its note ID; motion off the items clears it. The status row's existing motion handling is untouched.
    - A click on an item calls `presenter.Copy` with its text and gives the notice "Note copied."
    - A click on `×` closes the menu.
  - Outside the box everything routes as today.
  - Esc is not taken: agents use it.
- **Lifecycle:** the menu is closed by a toggle, by `×`, when the snapshot has no notes, when pointing turns off, or when the broadcast ends. A hover whose note has gone is cleared.
- **Hover glow:** `NoteStyle.Hover` holds the hovered note at the flash level while the menu shows it.

### Task 16: Menu rendering (pure)

- [ ] Tests:
  - Placement at 80×24 and at 40×10.
  - Wrapping, and the 3-line cap with `…`.
  - "+k more" when items overflow.
  - Hover on the lighter `notesMenuHover` background.
  - A chip in each author's color.
  - Text with an escape character drawn literally, never as a control.
  - `ItemAt(x, y)` and `CloseAt(x, y)` hit-tests.
  - Nothing drawn on the last row.
- [ ] Implement; PASS; commit `#447 M4: notes menu rendering`.

### Task 17: Count span, routing, copy, hover glow

- [ ] **Tests** (console fixture):
  - Clicking `💭1` opens the menu and the item text is on screen; clicking again closes it.
  - Motion over the item highlights it and holds its 💭 at the flash color.
  - A click copies: OSC 52 carries the full text, with the notice "Note copied."
  - A click and motion inside the box never reach the child (`f.child.Writes()` unchanged).
  - Resolving the last note closes the menu, and so does pointer-off.
  - Status-row spans: `RenderedStatusRow.Notes` covers exactly `💭N`, and the pointer and remote spans are unchanged.
- [ ] Implement; PASS with `-race`; commit `#447 M4: notes menu in the terminal: open, hover, copy`.

### Task 18: M4 boundary and final smoke

- [ ] Atlas (couch side: the menu and routing), README (Sticky notes paragraph: the menu, click-to-copy, hover glow, and Shift/Option-drag selection), lessons, and the full suite.
- [ ] `sdlc milestone-close --issue 447 --milestone M4`.
- [ ] `sdlc move :0`, `make build`, then the operator's full Done-when walk; then `sdlc close`.

## Revisions

- 2026-10-11 — M4 review round 2: a console-level hover test (glow at full color and row highlight, against an unhovered control off the beat; mutation-checked); a gesture that starts in the menu is the menu's until its release; Task 16 wording matches the code.

- 2026-10-11 — M4 review (REWORK): the menu's hit-test layout is the layout on the frame now on screen and is cleared on any frame that doesn't show it (a private panel); the menu is its own overlay step after the marks; hover lives in `pnotes.hover`, apart from the wholesale-replaced style; one table-driven lifecycle test (resolve-last, pointer-off, broadcast end); Core concepts match the code.

- 2026-10-11 — M3 close review (Minors): Core concepts state `paletteTint` in `t` (0 = full color), matching the code; a palette reply restyles notes at once (`restyleNotes`).

- 2026-10-11 — M3 review: one shared flash beat instead of per-note phases (bounds repaints at the 48-note limit); `Config.Now` gives notes the console's clock; `blendable` shared by marks and notes; timer tests cover re-arm and both teardowns for both timers, asserted at once.

- 2026-10-11 — operator smoke on pair:0 → scope revision: Chunk 3 (M3: chip, round glow, arrival and 15s flash; the viewer's hint bar becomes tooltips) and Chunk 4 (M4: a `💭N` notes menu in the terminal with hover highlight and glow, and click-to-copy; read and copy only; Esc not taken). M2 closes on the headless-Chrome smoke plus the operator's pair:0 smoke, which placed and saw notes.

- 2026-10-11 — plan-quality PQ-6 (Critical, lock held across a foreign call): `addNote` no longer runs a frame check under `p.mu`; the session reads the flip generation, checks the frame with no lock held, then `addNote(n, gen)` refuses if pointing is off or flipped since. `landsNow` checks only the frame. Lock rule written into `pointer.go`; hub-loop deadlock test added.

- 2026-10-11 — plan-quality PQ-1/PQ-2 (pointer-off ordering): the note board moves inside `PointerState` under its one lock, so a flip to off and the clear are one critical section and an add can't slip in after the clear; the console resyncs its snapshot from the session on every pointer-phase transition, dropping in-flight pre-clear snapshots. Minors: the guard order is stated once; `FuzzParseNote` and a `DrawNotes` property test; format characters (Cf) refused.

- 2026-10-11 — operator: turning pointing off deletes every note (the whole collaboration layer goes), replacing "notes stay until the broadcast ends"; session clears the board on the flip, the console gates drawing on its pointer phase; tests added.
- 2026-10-11 — fresh-eyes plan review folded in before approval: the console gates `onNotes` on live + same session and keeps the snapshot under a leaf lock shared with the count; `readPointerBody` keeps the rate check before the body read and holds the slot through handling; `MaxNoteBody` 8 KB; nil `Notes` and unknown-id resolve specified; `DrawNotes` blanking keeps style and reads the output frame; known overlaps recorded; added tests for each.
- 2026-10-11 — operator smoke of M3/M4: flash hold 300ms, fade in 25 steps of 20ms, fainter rest (0.7 / 0.88); right click on a notes-menu item copies and resolves it.
- 2026-10-11 — operator: rest is one very light shade on the inner ring only (t 0.9); the outer ring shows only during a flash.
