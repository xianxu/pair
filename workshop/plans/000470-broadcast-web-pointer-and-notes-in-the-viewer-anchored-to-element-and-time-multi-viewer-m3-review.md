# Boundary Review — pair#470 (milestone M3)

| field | value |
|-------|-------|
| issue | 470 — broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer |
| repo | pair |
| issue file | workshop/issues/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 871c6f47865114d9e5697a0e1211bd6aa3447781..07bda0056e50f079392ae1cd76dcad76b40dafea |
| command | sdlc milestone-close --issue 470 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T04:26:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and plan read; now checking the menu's click routing and the threading of the new web-notes paths, then running tests.

The Go tests need a generated asset bundle; generating it in the disposable checkout, then running the couchtty tests.

Confirmed the board publishes notes in a stable order; now checking whether the screen↔web menu switch and the stale-delivery guard have tests.

Viewer text is sanitized before it reaches the terminal, so no injection risk there. Next, a quick mutation check: does any test fail if a screen-broadcast stop unconditionally removes the overlay?

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M3 does what it set out to do. The Chrome cell now shows `💭N` with its own click span. A click opens #447's notes menu with the web notes, labelled with each author. A left click copies the plain text, and a right click copies the note and resolves it as the operator. The resolve lands in the recording. README, atlas and the plan's Revisions all describe the new surface. Running `go test -race` on the M3 and notes/webcast tests passes, and `node --test tests/webcast/annotate.test.mjs` passes 8/8. Two things stand between this and SHIP:
- **Wrong note on a click (correctness).** Web menu items are identified by their position in the latest list, but the click is matched against the menu as last drawn. If a viewer's resolve arrives between the draw and the operator's click, the operator copies and resolves a different note, and that resolve is visible to every viewer.
- **Untested overlay sharing.** The new rule that keeps the overlay while either the screen broadcast or a web cast needs it has no test that fails without it. I checked this with a mutation.

Neither needs a redesign; both are cheap.

### 1. Strengths
- **One menu with a source.** `notesSource`, `menuNotesLocked` and `screenHoverLocked` (`cmd/internal/couchtty/console_notes.go:49-73`) let the screen and web lists share #447's menu without mixing. The screen note's hover glow and its timer (`scheduleNotes`) are kept away from web items.
- **Pure, tested pieces.** `webMenuNotes` and the `RenderStatusRow` span are pure functions with direct unit tests (`TestWebMenuNotes`, `TestStatusRowWebNotes`, which also checks the exact span columns).
- **A real end-to-end console test.** `TestWebNotesInCouch` drives a real HTTP POST through the Server and Board into the reference `board.Writer`, then on through the status row, menu, copy and resolve. It checks that the click never reaches the child and that the resolve is in the recording.
- **Stale deliveries are dropped.** `applyWebNotes` ignores notes from a cast that has ended. The notes goroutine ends when the board closes or on `c.stop`, so there is no leftover state.
- **Viewer text stays safe.** It reaches the terminal only through `wrapNote` → `rowtext.Sanitize`. Names are also checked at the board (`nameText`), and `validName` in JS now uses the same rule (`\p{Cc}\p{Cf}`), with a test.

### 2. Critical findings
None.

### 3. Important findings
- **I-1 — The menu acts on a list that may have changed since it was drawn** (`cmd/internal/couchtty/console_notes.go:443`, `:276`, `:386`, `:328`). ARCH-ORDER: an observation is evidence about one entity at one moment, not lasting permission to act on it.
  - **The mechanism:** `webMenuNotes` gives each item `ID: i+1`. `routeNotesMenu` gets the id from the last drawn menu (`menu.ItemAt`) but looks it up in the latest list (`webTexts[id-1]`, `webIDs[id-1]`).
  - **The failure:** the menu shows [a, b]. A viewer resolves a, and the new list [b] arrives before the next frame is drawn. The operator right-clicks the row still showing "a". They copy and resolve **b**, and every viewer loses b.
  - **How screen notes avoid it:** their slot ids are stable, so a stale click finds nothing or the same note.
  - **The plan already said this:** the original Architecture called for "a console-local int id per note id". The M3 Revision changed it to position + 1.
  - **Fix:** keep a stable `map[string]int` from note id to a menu id in `pnotes`, assigned from a counter and pruned when a note leaves the list. Look up text and note id by that menu id. Store one `[]webItem{menuID, noteID, text}` instead of three parallel slices. This also stops the hover from jumping to a different note.
  - **Regression test:** deliver [a, b] and draw the menu. Deliver [b] without redrawing, then right-click a's row. b must stay open.
- **I-2 — The overlay sharing and source switching declared in the Revision have no tests** (`cmd/internal/couchtty/console_pointer.go:57`, `console_broadcast.go:292`, `console_notes.go:250`). **This is the 3rd finding in family `spec-path-untested`.**
  - **Evidence:** I replaced `syncOverlay(overlayScreen, false)` in `detachBroadcastScreen` with the old unconditional `SetOverlay(nil)`. The couchtty suite failed exactly the same four tests as the unmodified tree. All four fail on both runs because this environment can't start a PTY (`operation not permitted`), so the mutation survived.
  - **The rule** (fix this, not the one instance): every behavior a plan `## Revisions` entry declares gets a named test that fails without it. When a milestone closes, each Revision bullet should point to its test.
  - **For M3 that means two tests:**
    - Run a screen broadcast and a web cast together. Stop the broadcast, and the web menu must still draw.
    - Click the screen `💭N` while the web menu is open, and the menu must switch source with its hover reset.

### 4. Minor findings
- `pnotes.web`, `webIDs` and `webTexts` are three parallel slices that must stay in step. Fold them into one item struct, which the I-1 fix does anyway.
- **`syncOverlay` ordering is only safe by convention.** It works out the desired overlay under `c.mu` but calls `SetOverlay` after unlocking. Two callers on different goroutines could apply in reverse order (ARCH-ORDER). Likewise, `applyWebNotes` checks the cast is current, then writes `pnotes`. Both are correct only because every caller runs on the terminal loop. A comment should say so.
- The plan's M3 checkbox is still `[ ]`. Tick it in the close commit.

### 5. Test coverage notes
- **Ran:** `go test -race -run 'WebNotes|WebMenuNotes|StatusRow|NotesMenu|Notes|Webcast' ./cmd/internal/couchtty/` passes. I first had to generate the runtime bundle with `go run ./cmd/internal/runtimebundle/generatecmd`, because the checkout doesn't include it.
- **Full couchtty suite:** 4 PTY tests fail on both the unmodified tree and the mutation, because PTY spawn isn't permitted here. Not related to this change.
- **Not covered:**
  - a click on a stale menu (I-1)
  - the overlay staying installed when one of its two users leaves (I-2)
  - switching the menu between screen and web (I-2)
  - a notes delivery arriving after the cast ended (cheap: call `applyWebNotes` with a cast that isn't current)

### 6. Architectural notes
- **ARCH-DRY: pass.** It reuses #447's menu, `broadcast.Note`, the palette and `NoteGlyph`. The parallel slices are a small representation smell (Minor).
- **ARCH-PURE: pass.** `webMenuNotes` and `RenderStatusRow` are pure. The console glue stays thin.
- **ARCH-PURPOSE: pass.** Count, menu, copy and operator resolve are all delivered, and both README and atlas are updated.
- **ARCH-MOCK: pass.** The console test uses `FakeBrowser` and the reference `board.Writer` behind the real `Emit` seam, plus real HTTP to the cast server.
- **ARCH-CONSTRAINTS: pass.** Deliveries are limited by the board's note rate (2/s, burst 5). Each costs a repaint and a Refresh. Keeping the overlay installed for a whole web cast costs a cheap per-frame pass.
- **ARCH-SECURE: pass.** Viewer names and text are checked at the board and cleaned before reaching the terminal. The resolve is recorded as `board.Operator`.
- **ARCH-ORDER: flag (I-1).** A stale snapshot position is used to act. Also, the `syncOverlay` ordering depends on every caller being on the loop (Minor).
- **ARCH-FUNERAL: pass.** Nothing durable is created. The notes goroutine ends when the board closes or `c.stop` fires, and `forgetWebNotes` clears state when the cast ends.

### 7. Plan revision recommendations
- Change the M3 Revision's "web items are `broadcast.Note`s with ID = position + 1" to "a stable console-local menu id per note id (assigned once, pruned when the note leaves)". That matches the original Architecture and the I-1 fix.
- Add to the M3 Revision the tests that pin overlay sharing and source switching (I-2).

```findings
findings:
  - id: new
    severity: Important
    family: hit-by-stable-identity
    title: |
      Web notes menu: a click acts on the latest list by position, not on the note that was drawn
    detail: |
      webMenuNotes gives items ID i+1 (console_notes.go:443). routeNotesMenu takes the id from the last drawn menu (ItemAt, :328) but looks it up in the latest list (webTexts[id-1] :276, webIDs[id-1] :386). If a viewer's resolve arrives between the draw and the click, a right click copies and resolves a different note for everyone (ARCH-ORDER: an observation used as lasting authority to act). Fix: a stable map from note id to menu id in pnotes, assigned once and pruned, with lookups by that id and one []webItem instead of three parallel slices; this also fixes the hover jumping to another note. Test: deliver [a,b] and draw, deliver [b] without redrawing, right-click a's row, and b must stay open.
  - id: new
    severity: Important
    family: spec-path-untested
    title: |
      The overlay sharing (syncOverlay) and menu source switching declared in the M3 Revision have no tests
    detail: |
      3rd finding in this family. Rule to fix: every behavior a plan Revision declares gets a named test that fails without it, and each Revision bullet points to its test when the milestone closes. Evidence: with syncOverlay(overlayScreen,false) in detachBroadcastScreen replaced by an unconditional SetOverlay(nil), the couchtty suite failed exactly the same four PTY tests (unrelated: PTY spawn isn't permitted here) as the unmodified tree, so the mutation survived. Needed: a screen broadcast and a web cast together, stop the broadcast, and the web menu still draws; a click on the screen 💭N while the web menu is open switches the source with the hover reset.
  - id: new
    severity: Minor
    family: shared-state-loop-confinement
    title: |
      syncOverlay and applyWebNotes are correct only because every caller runs on the terminal loop, and nothing says so
    detail: |
      syncOverlay works out the desired overlay under c.mu but calls SetOverlay after unlocking. applyWebNotes checks the cast is current under c.mu, then writes pnotes. Callers on two goroutines could apply out of order. A comment saying both run on the loop (or one lock held across both steps) would make this explicit (ARCH-ORDER).
```
