# Boundary Review — pair#488 (whole-issue close)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | whole-issue close |
| milestone | — |
| window | 340d175014bd6aee3b1acc2cfed1e3fa96df5f14..6b42e6edfd93f18ed25b7fb52ca8cacaca78667c |
| command | sdlc close --issue 488 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T15:05:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the close boundary for pair#488: checking the 7 open findings against the head before I look at the latest commits.

Still verifying fixes: checking that the BR-22 cleanup actually runs when the broadcast ends.

Prior findings checked. Now running the Go and node test suites for the touched packages.

Test environment has temp-dir permission issues; retrying the note/anchor tests with a writable temp dir.

The BR-22 wiring is untested: removing the `forgetAnchors()` call from `resetPointer` leaves every test green. Now scanning the core anchor code for anything earlier rounds missed.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue does what it set out to do. Web and terminal notes are both bounding boxes built by one shared gesture helper. Web crops are now the box plus 16 px, and the old crop snapping and per-URL `Hints` are gone. Terminal boxes are outlined by Couch, and each note gets a frame-only anchor straight away, then a pane path filled in asynchronously behind the `PaneRects` seam. The smoke-test change (both mode buttons visible at once) is small and has a test.

Tests run at head after generating the runtime bundle: `broadcast`, `webcast/...` and `reviewcmd` pass; the `couchtty` note, anchor, viewer and pointer tests pass under `-race -count=2`; the `couchcmd` note-pane and review-file tests pass, with the live zellij conformance check skipped; and all 168 node tests pass. Other failures in `couchcmd`, `couchtty` and `artifactpath` came from this checkout's environment (the `/tmp` and `/Volumes/AgentScratch` temp paths weren't writable, and stale lines in the bundle-violations file), not from this change.

Nothing blocks the close. All remaining items are Minor. The two cheapest are adding the plan Revisions entry for BR-20 and making the BR-22 test go through `resetPointer`.

### 1. Strengths
- **Anchor jobs carry what they saw** (`console_noteanchor.go:66-72`). Each job keeps its own frame and session. When applied, it re-checks the endpoint, geometry epoch and geometry. The resolver is single-flight, and it starts `GoTracked` only after releasing the lock.
- **Untrusted input is bounded before any arithmetic** (`notes.go:247`, and `within` compares by subtraction). The 5-second deadline test from BR-10 still guards against the overflow.
- **The resolved anchor stays off the public wire.** `Note.Anchor` is tagged `json:"-"` (`notes.go:56`), so the file names and headings it holds never reach viewers.
- **Cross-language parity is pinned.** One shared case table, `tests/broadcast-viewer/pinspots.json`, drives both the Go and node tests.
- **The smoke-test change has a test.** Switching mode ends a gesture in progress (`viewer.js:423`), and `pointermode.test.mjs:329-347` asserts that the old stroke neither keeps posting nor turns into a note.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **BR-22, not addressed (test gap only):** `forgetAnchors` is called from `resetPointer` (`console_pointer.go:270`), which runs from `detachBroadcastScreen` when the broadcast ends. But the new test calls `forgetAnchors()` directly. I deleted the call from `resetPointer` and all the Anchor, Note, Broadcast and Pointer tests stayed green. Fix: have `TestForgetAnchorsLeavesNothingOfTheSession` call `c.resetPointer()` instead.
- **BR-20, not addressed:** plan lines 65 and 76 still name `AnchorFor(box, l, frame)` and `noteanchor_io.go`. The code has `AnchorFor(box, f, l)` and no `_io` file. Two notes also still say `tried` is only "pruned on every snapshot": plan line 268 and `console_noteanchor.go:51-53`. It is now also cleared when the broadcast ends.
- **BR-5 and BR-13:** still open and unchanged.
- **New (`retired-surface-sweep`, 3rd in the family):** comments still describe the retired single mode button. They are at `viewer.css:91`, `viewer.css:141-142` ("one button below the swatches… Only the button…") and `pointermode.test.mjs:2`.

### 5. Test coverage notes
- The fix for BR-22 is reachable in production but no test reaches it (above).
- The live zellij conformance check and the Chrome-gated annotate test were skipped here. Their correctness rests on earlier rounds and the operator's smoke test.

### 6. Architectural notes, principle by principle
- **ARCH-DRY: pass.** One gesture helper, one pin table and one review-script constant.
- **ARCH-PURE: pass.** `AnchorFor`, `locate`, `sectionOf`, `TextUnder` and `notePanes` (apart from the injected file read) are pure. The IO stays in `paneRectsOf`.
- **ARCH-PURPOSE: pass.** Both casts, both anchors and both crop seams are delivered.
- **ARCH-MOCK: pass.** The `PaneRects` seam has a fake behind it plus `TestNotePanesZellijConformance`.
- **ARCH-CONSTRAINTS: pass, with BR-13 noted.** Pane queries have a 2-second bound, `locate` is capped at 1M comparisons, file reads at 1 MiB and anchor text at 1 KiB.
- **ARCH-SECURE: pass.** Requests are parsed strictly, and the anchor is excluded from the wire.
- **ARCH-ORDER: pass.** Jobs are keyed per session and carry their frame. There are tests for a slot switch mid-pass and for two sessions.
- **ARCH-FUNERAL: pass at runtime.** Nothing durable is created, and resolver state is dropped when the broadcast ends; only its test is missing (BR-22).

### 7. Plan revision recommendations
Add a "Close as built" Revisions entry that:
- reconciles the Design names: `AnchorFor(box, f, l)`, the shell in `console_noteanchor.go`, and no `noteanchor_io.go` or view.json fuzzing;
- notes that `tried` and `pending` are cleared when the broadcast ends (`forgetAnchors`).

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      M2/M3 built the referents: broadcast.NoteAnchor (notes.go), CellBox and AnchorFor exist at head, so the atlas line 217 and record.go present-tense wording now describe built surface.
  - id: BR-5
    disposition: not-addressed
    note: |
      annotate.css:17 is unchanged: the 26 px chip still sits at right:2px top:2px in every box, however thin.
  - id: BR-9
    disposition: addressed
    note: |
      Plan "M1 as built" (lines 239-242) names gestureBox, dragMin, boxCentre and notebox.test.mjs, and says to read noteBox above as gestureBox; it matches collab.js.
  - id: BR-13
    disposition: not-addressed
    note: |
      notes.go:255 still calls len(pinSpots(...)) == 0 in ParseNote, and Add still builds the full list before taking the first free spot.
  - id: BR-14
    disposition: addressed
    note: |
      Plan "M2 as built" (lines 244-254) says off-grid boxes are refused and NoteBoard.Add places the pin via pinSpots, matching notes.go:148-176.
  - id: BR-20
    disposition: not-addressed
    note: |
      Plan lines 65/76 still say AnchorFor(box, l, frame) and noteanchor_io.go (code: AnchorFor(box, f, l), no _io file); plan line 268 and console_noteanchor.go:51-53 say tried is only pruned per snapshot, but forgetAnchors now also clears it.
  - id: BR-22
    disposition: not-addressed
    note: |
      forgetAnchors is wired into resetPointer, but deleting that call leaves every couchtty test green, because the test calls forgetAnchors directly; drive it through resetPointer/detachBroadcastScreen.
findings:
  - id: new
    severity: Minor
    family: retired-surface-sweep
    title: |
      Comments still describe the retired single mode button at viewer.css:91, viewer.css:141-142 and pointermode.test.mjs:2
    detail: |
      This is the 3rd finding in family retired-surface-sweep (prevalence 3 across M1 to close). Rule, already in lessons.md from BR-7, widened: when a commit retires an element id, class or function, grep its old name in code, CSS, comments and tests (here the "#mode" id and "mode button" in the singular) in that same commit, and fix every hit. Apply that grep rather than fixing these three lines one by one.
```
