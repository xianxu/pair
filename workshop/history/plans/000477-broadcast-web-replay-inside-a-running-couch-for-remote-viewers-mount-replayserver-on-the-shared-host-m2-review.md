# Boundary Review — pair#477 (milestone M2)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | fd1b66e9421f6dfda5a073a2a29502d14f662aeb..a6f660bb0de6bfedfe29839e2959c56cb45425eb |
| command | sdlc milestone-close --issue 477 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T06:53:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now annotate.js — the modules being consumed.

I'm reviewing the M2 replay-marks diff. The unit tests pass. The live Chrome test crashed under the sandbox, so I'm retrying it unsandboxed.

Every live Chrome test fails at launch, including the pre-existing ones (CDP EOF), so the problem is the environment. Next I'm checking for stale references and running the node tests.

I've found one display bug and a stale atlas passage. Last step: confirming `serveViewerFile` is the one table the atlas says every viewer server uses.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 does what the plan asks. The replay no longer has its own `drawMarks` (an SVG of strokes plus pinned note text). It now draws with #470's pure helpers: `liveNotes`, `pinLayer`, `strokesAt` and `paintStrokes`. The fidelity harness now serves files through the same `serveViewerFile` table the live and replay servers use. That fixes a real `collab.js` 404. `go vet` and `go test ./cmd/internal/webcast` pass, and the node suite passes 88/88 (`TestNode` too). I could not confirm the live check myself: every `TestLive*` test failed at Chrome launch (`Emulation.setDeviceMetricsOverride: cdp: read: EOF`), with or without the sandbox, and that includes the unchanged `TestLiveReplayServerSeeks`. So the failure comes from this machine, not the diff. Two cheap fixes stand before SHIP. First, a tapped note's card is placed only when a frame runs, and the replay only runs frames while playing, so on a paused replay the card shows in the wrong place. Second, `atlas/webcast-format.md` still describes the `drawMarks` path that M2 removed.

**1. Strengths**
- `replay.js:429-466`: the replay now uses the live viewer's own drawing modules. The only replay-specific code left is `place` (it gets box-relative coordinates) and the read-only card. This is the convergence #470 and #477 promised (ARCH-DRY).
- `replay_test.go:194-199`: the harness now serves files through `serveViewerFile` instead of its own copy of `webFS.ReadFile`, so every server of the viewer resolves imports from one table.
- `replay_live_test.go:162-191`: the stroke expectation is computed in Go from `recording.Reduce`, not from the JS. The test also checks that the canvas has ink exactly when strokes show, and fails if no moment ever paints a stroke, so it can't pass by checking nothing.
- The tapped card sets its text with `textContent` only (`replay.js:451`). Recorded input never reaches HTML.

**2. Critical findings:** none.

**3. Important findings**
- `cmd/internal/webcast/web/replay.js:446-461`: the tapped card is only positioned inside `draw(player)`, which runs from `deps.frame()`. `replaySource.draw` (`replay.js:293-310`) only schedules frames while playing, after a seek, or when the bar changes. A `#t=` link opens paused, and that's the usual way to read a note. Tapping a pin there shows the card with no `left`/`top`, at the page box's top-left corner. After tapping pin A and then pin B, the card shows B's text at A's position. It stays wrong until something else triggers a frame.
  - Fix: move the card placement into a `placeCard()` helper, call it from the tap handler as well as from `draw`, or have the tap request a frame.
  - Test gap: the live test only checks `hidden`, `note` and text. It should also check that the card sits next to the tapped pin.
- `atlas/webcast-format.md:417-418` still says "#471's replay draws marks with its own `drawMarks` today (per chunk, region last); #477 converges it onto these." M2 removed `drawMarks` and finished that convergence. The note-placement order also changed: `noteBox` tries the region first, while the old code tried it last. That's the third finding in family `atlas-surface-not-updated`, so the rule matters more than this one instance.
  - **Rule:** when a milestone removes or renames a symbol, or completes work that the atlas describes as "today" or "to replace", search `atlas/` and `README.md` for the old symbol names and for the issue number, and rewrite every hit in the same commit.
  - For M2 that search would be `drawMarks|STROKE_FADE|replay-layer|replay-strokes|#477 converges`.

**4. Minor findings**
- On a window resize while paused, `layout()` resizes `#box` but nothing redraws the marks. The canvas bitmap stretches and the pins stay where they were until the next frame. The old SVG had the same problem.
- `.box { overflow: hidden }` clips the card when a pin sits near the bottom or right edge of the page box.

**5. Test coverage notes**
- `replayMarks` has no node test. Its card toggle and hide-when-pin-vanishes logic is covered only by the Chrome test, which needs `WEBCAST_LIVE_CHROME=1` to run.
- The pure helpers are covered in `tests/webcast/annotate.test.mjs`.
- The tap is a synthetic `chip.click()`, which ignores `pointer-events`. I read the CSS instead: `.pin .chip` has `pointer-events: auto`, and the layer is stacked above `#overlay`, so a real tap reaches the chip.
- Live check: unverified in this environment, as described above.

**6. Architectural notes**
- **ARCH-DRY: pass.** The replay reuses #470's helpers, and the harness uses the shared `serveViewerFile` table.
- **ARCH-PURE: pass.** `strokesAt` and `liveNotes` are pure and node-tested. `replayMarks` is a thin DOM layer on top of them.
- **ARCH-PURPOSE: pass**, apart from the stale atlas text. The milestone's point (draw with #470's modules) is delivered, not deferred.
- **ARCH-MOCK: pass.** There's no new external dependency, and the Chrome test is opt-in.
- **ARCH-CONSTRAINTS: pass.** Each frame does two `getBoundingClientRect` calls and one canvas repaint. That's bounded.
- **ARCH-SECURE: pass.** Recorded note text goes only into `textContent` and `title`. Pin colors go through `HEX_COLOR`.
- **ARCH-ORDER: flag.** The card's `shown` state changes on a tap, but the view is only updated on a frame. The paused case leaves the card misplaced (the first Important finding).
- **ARCH-FUNERAL: pass.** M2 creates nothing durable. The DOM nodes live as long as the page.

**7. Plan revision recommendations**
- None needed for the design. Once the stale atlas passage is fixed, consider adding to Revisions: "replay note placement now follows `noteBox` (region, then element, then carried box), where the old `drawMarks` tried the region last."

```findings
findings:
  - id: new
    severity: Important
    family: ui-event-without-redraw
    title: |
      Replay note card is positioned only on a frame; a tap while paused shows it at the box's top-left or at the previous pin
    detail: |
      replay.js:446-461 sets card text and visibility in the pinLayer tap handler, but left/top are set only inside draw(player), which runs from deps.frame(). replaySource.draw schedules frames only while playing, after a seek, or on a bar change, so on a paused replay (how a t= link opens) the card stays unplaced until the next frame. Fix: call a placeCard helper from the tap handler too, or request a frame. Make the live test assert the card sits next to the tapped pin.
  - id: new
    severity: Important
    family: atlas-surface-not-updated
    title: |
      atlas/webcast-format.md:417 still describes the drawMarks path M2 removed
    detail: |
      This is the 3rd finding in the family. Rule: when a milestone removes or renames a symbol, or completes work the atlas describes as today or to replace, search atlas/ and README.md for the old symbol names and the issue number, and rewrite every hit in the same commit (for M2: drawMarks, STROKE_FADE, replay-layer, replay-strokes, and 477 converges). The passage should also note that note placement now follows noteBox (region first).
  - id: new
    severity: Minor
    family: ui-event-without-redraw
    title: |
      Window resize while paused leaves replay strokes stretched and pins misplaced until the next frame
    detail: |
      The resize handler runs layout() and drawPointer() but not the marks. The old SVG layer had the same gap.
```

---

## Re-review — 2026-10-12T07:00:57-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | fd1b66e9421f6dfda5a073a2a29502d14f662aeb..70309982bc57dc87c258ea217dc5d032f6e027b6 |
| command | sdlc milestone-close --issue 477 --milestone M2 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:00:57-07:00 |
| verdict | SHIP |

## Review

Reviewing the M2 replay drawing diff: I'm checking the shell wiring, then I'll run the tests.

Still reviewing: checking how a stroke's color is recorded, since the new canvas path drops the old hex-color fallback.

```verdict
verdict: SHIP
confidence: medium
```

M2 does what the plan asked for: the replay now draws notes and strokes with #470's `annotate.js` modules (`liveNotes`, `pinLayer`, `strokesAt`, `paintStrokes`). The old `drawMarks` SVG path is gone, with nothing left behind in code, CSS or docs. All three prior findings are fixed. The tap handler now places the note card itself (`replay.js:455-461`). A resize now redraws the marks on the next frame (`replay.js:500`). The atlas passage was rewritten and the old names no longer appear anywhere. Confidence is medium, not high, for one reason: I couldn't run the Chrome tests here. Every `WEBCAST_LIVE_CHROME=1` test, including ones this diff doesn't touch, failed during browser setup with `Emulation.setDeviceMetricsOverride: cdp: read: EOF`, before any replay code ran. So I judged the new live assertions by reading them, not by reverting the fixes and watching the tests fail. Nothing blocks SHIP.

**1. Strengths**
- **One drawing path:** live and replay now share `pinLayer`/`paintStrokes`, so a pin is placed by `noteBox` (its region first) the same way in both. Both now use the same `.annotate` layer markup and CSS (`replay.js:480`).
- **An independent test oracle:** `replay_live_test.go:162-186` works out the expected stroke IDs from the Go reducer by its own rule. It doesn't restate `strokesAt`. It also checks that the canvas really has paint on it, and fails if no moment ever paints a stroke (`painted == 0`).
- **The paused case is tested properly:** the card check clicks and then measures in the same step, with no frame in between. The resize check waits exactly two frames, without polling. The lesson recorded at `workshop/lessons.md:1265` explains why a polling test would pass even with the fix removed.
- **Safe input handling:** the card is filled with `textContent`, and the note ID goes through `CSS.escape` before it's used in a selector (`replay.js:448`).
- **One file table for every server:** the fidelity harness now serves files through the shared `serveViewerFile` (`replay_test.go:197`), so it can't drift from the module graph again.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- `replay.js:482` reads `devicePixelRatio` once, when the replay mounts. The live viewer reads it on every draw (`annotate.js:484`). If you zoom the browser or drag the window to another display, the replay canvas keeps the old pixel ratio, so strokes look blurry or over-sampled until you reload. The fix is to pass `() => win.devicePixelRatio || 1`, or read `win.devicePixelRatio` inside `draw`.

**5. Test coverage notes**
- **Tap test:** without the fix, the card has no `left`/`top` and sits in its static position, so `dx != 0` and the test fails. The test therefore pins BR-12.
- **Resize test:** without the fix, the canvas keeps its old pixel size after the shell's `layout()`, so the width/height comparison fails. It pins BR-14.
- **Not run:** I couldn't confirm either of these by reverting the fix, because Chrome won't start in this environment.
- **Real taps:** `chip.click()` skips the browser's hit-testing. I checked the layering by reading instead: the `.annotate` layer is added after `#overlay` inside `#box`, and only `.chip` takes pointer events, so a real tap reaches it. A test that sends a real pointer event would close this gap.
- **Non-live run:** the non-Chrome suite (`go test ./cmd/internal/webcast/`) passes.

**6. Architectural notes**
- **ARCH-DRY:** pass. The `place` closure looks like the live one, but it maps to a different frame (box px, where live uses window px minus the box rect), and that difference is real. The read-only card is specific to the replay.
- **ARCH-PURE:** pass. `strokesAt`, `liveNotes` and `noteBox` are pure. `replayMarks` is thin DOM glue around them.
- **ARCH-PURPOSE:** pass. The replay's only drawing path now uses #470's modules. No hand-copied drawing logic is left, and the old names are gone from the code, atlas and README.
- **ARCH-MOCK:** N/A. There are no new external dependencies.
- **ARCH-CONSTRAINTS:** pass. Each frame filters the notes, moves the pins (they're only rebuilt when the set of notes changes), runs one `querySelector`, and repaints one canvas.
- **ARCH-SECURE:** pass. Recorded text reaches the page only through `textContent`, and IDs are escaped before use in a selector. Stroke colors come from the server's palette index (`board/wire.go:94`).
- **ARCH-ORDER:** pass. The only state kept between events is the nullable `shown`. Every draw re-checks it against what's on screen, and it's cleared when its pin goes away.
- **ARCH-FUNERAL:** pass. It creates nothing durable: the DOM nodes live as long as the page.

**7. Plan revision recommendations:** none. The plan's M2 line (the replay draws with #470's modules, plus a live check that a note shows at its moment) matches the code.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      replay.js:455-461 now calls placeCard from the pinLayer tap handler; replay_live_test.go measures the card in the same step as the click (dx 0, dy 4), which fails if left/top are unset. Read, not run: Chrome could not start here.
  - id: BR-13
    disposition: addressed
    note: |
      atlas/webcast-format.md:417 rewritten (replayMarks, noteBox region first); atlas/webcast.md:239 updated. A grep of atlas, README and cmd finds no drawMarks, STROKE_FADE, replay-layer, replay-strokes, PIN_ROOM or "converges".
  - id: BR-14
    disposition: addressed
    note: |
      replay.js:500 redraws the marks on the frame after a resize; the live test checks the canvas size against the box after exactly two frames, with the replay paused.
findings:
  - id: new
    severity: Minor
    family: captured-environment-value
    title: |
      replayMarks reads devicePixelRatio once, at mount, so the stroke canvas goes stale after a zoom or a move to another display
    detail: |
      replay.js:482 passes win.devicePixelRatio as a fixed value; the live viewer reads it on every draw (annotate.js:484). Fix: pass a getter, or read it inside draw.
```
