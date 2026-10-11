# Boundary Review — pair#446 (whole-issue close)

| field | value |
|-------|-------|
| issue | 446 — Broadcast pointer: each viewer picks one of six colors |
| repo | pair |
| issue file | workshop/issues/000446-broadcast-pointer-each-viewer-picks-one-of-six-colors.md |
| boundary | whole-issue close |
| milestone | — |
| window | 06d6d4de0b96195217280f70b004f6357f7d75de..99b1f4e436f93ead71b218532872d777aebf44fe |
| command | sdlc close --issue 446 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T12:19:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Broadcast tests pass. The couchtty setup failure comes from a missing generated asset in this checkout, not from the diff. Next I'm checking the README and whether broadcast frames carry the marks.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The feature is implemented cleanly and matches the Spec. The palette lives in one place (`markPalette` in `marks.go`), and the server sends it to viewers in the `caps` event, so `viewer.js` has no colors of its own. `ParsePointBatch` rejects anything that isn't an integer from 0 to 5. Each pointer stream takes the color the fewest live streams hold and gives it back when the stream ends. The viewer's own pick is saved in `localStorage` inside try/catch, so blocked storage only stops it being remembered. `go test ./cmd/internal/broadcast/` passes (Go plus the node viewer tests). `cmd/internal/couchtty` would not build in this checkout because a generated runtime asset is missing (`runtimebundle/embed.go: no matching files`). That is an environment problem, and the change in couchtty is one line (`console_pointer.go:170`). One thing should be fixed before shipping: README.md still says marks are "amber" and never mentions the color picker.

1. **Strengths**
   - One source for the palette: `PointerPalette()` (`marks.go:176`) builds the colors from `markPalette`, the viewer gets them through `caps` (`server.go:812`), and `PointerColors` sets the size of the `held` array (`pointer.go:31`).
   - Strict parsing: `Color *int` rejects strings and fractions, and the range check rejects -1 and 6 (`point.go:552,560-565`). Both are tested, and the fuzz test has a color invariant (`point_test.go:616`).
   - Color assignment uses a mutex-guarded counter. Each claim is paired with a `defer` release in the events handler (`server.go:804-807`), so a color can't leak. `TestClaimColorPrefersLeastHeld` checks the least-held rule and that a released color is handed out again.
   - The fade test checks every palette color, with and without truecolor, for steps that get brighter. It caught two bad ladders during the work (`marks_test.go:455`).
   - The viewer checks untrusted `caps` input: a palette entry that isn't `#rrggbb` means no swatches are shown (`viewer.js:446`). The old ban on `localStorage` was narrowed to exactly one key, not removed (`server_test.go:827`).

2. **Critical findings:** none.

3. **Important findings**
   - **README.md still says marks are amber (README.md:880).** The text says "leaving amber marks" and never mentions the color swatches, the remembered pick, or the automatic per-viewer color. **Fix:** say that each viewer's marks draw in one of six colors, that viewers who haven't picked get a different color each, and that a viewer can pick a swatch, which is remembered in their browser. Per the docs gate, this is new surface that users can see.

4. **Minor findings**
   - `viewer.css:856`: the swatch bar sits fixed over the top-left of the stage. It hides those terminal cells, and a viewer can't point at them because the taps land on the swatches instead.
   - `marks_test.go:518`: `TestPointerPaletteMatchesMarks` only checks entries 0 and 5. The full list is pinned in `TestPointerCapsEvents`, so this test adds little.
   - The fade ladders without truecolor all darken toward a dark background. On a light terminal the fade gets darker instead of fading out. The Spec says "must read on both dark and light backgrounds"; this pattern predates the issue but now applies to six colors.
   - Swatch labels are "Pointer color 1..6" (`viewer.js:938`). Color names would be more accessible.
   - Done-when bullet 3 ("checked live in a broadcast") was replaced by an in-process test. The plan says so openly, but the live browser check is still outstanding for the close review.

5. **Test coverage notes:** parsing, per-color fades, keeping each batch's color, assignment and release (unit test and through the real server), the viewer's pick, persistence across a reload, blocked or missing storage, and rejecting a bad palette are all covered. `TestPointerStreamsGetDistinctColors` waits for the release with a polling loop. It handles timing but only exercises one interleaving, which is acceptable for a counter.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The palette has one source; the hex lists in the tests are fixtures.
   - **ARCH-PURE: pass.** `Marks`, `pointerColor`, `colorStore` and `renderSwatches` are pure or take their dependencies as arguments, and are tested without real IO.
   - **ARCH-PURPOSE: pass.** The viewer takes the palette from Couch rather than restating it. See the Minor note about the live check.
   - **ARCH-MOCK: N/A.** No new external dependency.
   - **ARCH-CONSTRAINTS: pass.** `PointerPalette()` runs on each `caps` send, which is rare and cheap.
   - **ARCH-SECURE: pass.** Color comes in as an index only; the palette and the stored value are validated in the browser.
   - **ARCH-ORDER: pass.** The `held` counters change only under the lock, and every claim is released on all exit paths. A viewer's pick never changes the counts, which is a stated design choice.
   - **ARCH-FUNERAL: pass.** One fixed `localStorage` key that doesn't grow, and per-stream counts that are released when the stream ends.

7. **Plan revision recommendations:** none. The Plan and Log match the code. The deferred live check is already disclosed in the fourth Plan item.

```findings
findings:
  - id: new
    severity: Important
    family: docs-gate-user-surface
    title: |
      README pointer section still says marks are amber and omits the color picker
    detail: |
      README.md:880 says "leaving amber marks". Describe the six colors, the per-viewer automatic color, and the swatch pick remembered in the browser.
  - id: new
    severity: Minor
    family: ui-overlay-occludes-content
    title: |
      Fixed swatch bar covers the top-left terminal cells, which can't be seen or pointed at
    detail: |
      viewer.css:856 puts #swatches at top:8px left:8px over the stage, so taps there hit the swatches, not the stage.
  - id: new
    severity: Minor
    family: weak-assertion
    title: |
      TestPointerPaletteMatchesMarks checks only two of six entries
  - id: new
    severity: Minor
    family: spec-light-background
    title: |
      Fade ladders without truecolor darken toward a dark background, so on light terminals the fade darkens instead of fading out
  - id: new
    severity: Minor
    family: done-when-live-check
    title: |
      Done-when live two-viewer broadcast check was replaced by an in-process test and left to the operator
```

---

## Re-review — 2026-10-11T12:23:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 446 — Broadcast pointer: each viewer picks one of six colors |
| repo | pair |
| issue file | workshop/issues/000446-broadcast-pointer-each-viewer-picks-one-of-six-colors.md |
| boundary | whole-issue close |
| milestone | — |
| window | 06d6d4de0b96195217280f70b004f6357f7d75de..bc0137b13d3cca228f2c18b9b42c0ad230cd4a5b |
| command | sdlc close --issue 446 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T12:23:28-07:00 |
| verdict | SHIP |

## Review

Still reviewing — checking CSS layout, then running tests.

Wrapping up: one last check of README line 1151 and the CHANGELOG convention.

```verdict
verdict: SHIP
confidence: medium
```

This is ready to ship; nothing blocks it. The six colors live in one place, `markPalette` in `cmd/internal/broadcast/marks.go:46`. Everything else reads from it: the wire format, the fade ladders, the `caps` event and the viewer's swatches. Viewers send only a color number, never a raw color. `ParsePointBatch` refuses anything outside 0–5, and the server checks every value against the palette. `go test ./cmd/internal/broadcast/` passes in this checkout, including the 34 node viewer tests run through `TestViewerNode`.

`cmd/internal/couchtty` could not be compiled here. The build stops on `runtimebundle/embed.go:8` because generated `assets/runtime/files` are missing from the checkout. The only change in that package is a one-line call site (`console_pointer.go:170`), and its signature matches the new `Marks.Add`. Confidence is medium for that reason only.

Three of the five earlier findings are resolved:
- **BR-1:** the README now describes the colors.
- **BR-2:** the swatches moved into a band below the screen.
- **BR-3:** the palette test checks all six colors.

I withdraw BR-4: the dark fade already existed for amber, and the spec's "reads on both backgrounds" applies to the full-strength color. BR-5 is still open because the live check was never run.

**1. Strengths**
- **One palette source.** `marks.go:46` holds the colors; `PointerPalette()` sends them in `caps`, and `viewer.js` contains no color values. `HEX_COLOR` (`viewer.js:450`) also checks the palette in the browser, so a bad `caps` message shows no swatches.
- **Strict input checks.** The new `color` field is read as `*int`, which refuses strings, fractions and `"#ff0000"`. Tests cover each case (`point_test.go:485-489`), and the fuzz test now checks the color range (`point_test.go:506`).
- **Color assignment is simple and safe.** `claimColor`/`releaseColor` (`pointer.go:532-549`) take the same lock as the rest of the pointer state. Each claim is paired with a deferred release in `events()` (`server.go:695`), so the counts stay balanced.
- **The browser logic is testable without a DOM.** `pointerColor`, `colorStore` and `renderSwatches` take their storage and element factory as arguments. The tests cover a pick surviving a reload, storage that throws, missing storage, and an invalid palette (`pointermode.test.mjs:116-185`).
- **The new fade test found real bugs.** It checks that every color's fade only darkens, and that the 256-color fade walks all five steps of that color's ladder (`marks_test.go:339`). It caught two ladders that brightened partway through.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **The hint and the swatches overlap on phone-width screens** (`viewer.css:103-115` and `#hint` at `viewer.css:82`). Both sit in the 40px band at fixed positions: swatches from the left (about 182px wide), the hint from the right (about 280px wide). On a ~390px phone they overlap by roughly 80px and the swatches hide part of the hint. iPad widths are fine.
  - This is the second finding in family `ui-overlay-occludes-content`. The rule that covers both: every fixed control on a pointer page goes inside one reserved band container that lays its children out as a wrapping flex row. No control is positioned independently over the stage or over another control. Put `#swatches` and `#hint` in one band element, and size `#stage.controls { bottom }` from that band's height.
- `colorStore.get` accepts any single digit (`/^[0-9]$/`). `pointerColor` then limits it to the palette size, so this is harmless; a comment saying so would help.

**5. Test coverage notes**
- **Parsing:** `color: 3` is accepted; `6`, `-1`, a string and a fraction are refused. This covers the first Done-when item.
- **Fading:** each color is checked holding and fading, with and without truecolor. The blend is only tested on a dark background (`0x1e1e1e`). No test uses a light background, which the spec mentions.
- **Assignment:** `TestPointerStreamsGetDistinctColors` runs through the real server: two streams get colors 0 and 1, and color 0 is handed out again after its stream ends. It applies the batches to its own `Marks` rather than going through `Console.applyPoints`. That is acceptable, because the call site is a one-line pass-through.
- **Layout:** nothing tests the controls band (`#stage.controls`, and `fit` measuring the stage instead of the window); there is no DOM harness for it. For BR-2, I checked by reading the CSS that both controls fit inside the 40px band:

  | Control | Bottom offset | Height | Top edge (from bottom) |
  |---|---|---|---|
  | Swatches | 6px | about 30px | about 36px |
  | Hint | 8px | about 26px | about 34px |

**6. Architectural notes for upcoming work**
- **ARCH-DRY:** pass. One palette source. The hex list written out in two tests is a deliberate pin, not duplication.
- **ARCH-PURE:** pass. The color decision and storage wrapper are pure; the DOM and `localStorage` are passed in from `start()`.
- **ARCH-PURPOSE:** pass. Every use of the colors reads from `markPalette`. A search for `amber`/`markFadeLadder` found no leftover pointer-mark references, only the separate `PointerSGR` indicator and unrelated features.
- **ARCH-MOCK:** N/A. No external binary or service is involved; the tests use a fake EventSource and in-memory storage behind the same seams the real code uses.
- **ARCH-CONSTRAINTS:** pass. `PointerPalette()` builds a six-item list on each `caps` send, which only happens on join and when pointing is turned on or off.
- **ARCH-SECURE:** pass. Viewer input is parsed into a number in range at the server, so no viewer-supplied color string reaches the operator's terminal. In the browser, both the server's palette and the stored pick are checked before use.
- **ARCH-ORDER:** pass. The per-color count only changes through `claimColor`/`releaseColor` under the lock, with a deferred release when the stream ends. One thing to know: a viewer who never picked a color and reconnects may get a different color. That fits the documented "assignment counts connections, not picks."
- **ARCH-FUNERAL:** pass. The only lasting thing is one `localStorage` key holding a single digit. It is bounded and overwritten in place.

**7. Plan revision recommendations**
- Done-when line 3 still says "checked live in a broadcast," while Plan item 4 records an in-process check instead. Do one of the following:
  - run the live two-viewer check and log it, or
  - add a `## Revisions` entry changing Done-when to "two pointer streams get distinct colors and land distinct marks (in-process, `TestPointerStreamsGetDistinctColors`); live browser check by the operator."
- Add a one-line `## Revisions` note that the 256-color fade ladders darken toward a dark background (as amber's did), so a light-background fade without truecolor is out of scope.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:880-884 now says colored marks, six colorblind-safe colors, a per-viewer automatic color, and a swatch pick remembered by the browser; this matches pointer.go claimColor and viewer.js colorStore.
  - id: BR-2
    disposition: addressed
    note: |
      viewer.css #stage.controls reserves a 40px band and fit measures #stage; by inspection the swatches top out about 36px and the hint about 34px from the bottom, so neither covers a cell. There is no DOM harness for an automated check.
  - id: BR-3
    disposition: addressed
    note: |
      TestPointerPaletteMatchesMarks (marks_test.go:402) checks all six hex values and PointerColors; TestPointerCapsEvents also checks the full palette in caps.
  - id: BR-4
    disposition: withdrawn
    note: |
      The darkening fade without truecolor is amber's existing behavior, now applied to each color; the spec's "reads on both backgrounds" covers the full-strength color, and truecolor blends into the real background. Recorded as a plan revision, not a defect.
  - id: BR-5
    disposition: not-addressed
    note: |
      Done-when still says "checked live in a broadcast" and no live check was run; either run and log it, or revise Done-when to name the in-process TestPointerStreamsGetDistinctColors.
findings:
  - id: new
    severity: Minor
    family: ui-overlay-occludes-content
    title: |
      Hint and swatches are positioned separately in the band and overlap on phone-width screens
    detail: |
      2nd in this family. Rule: every fixed control on a pointer page lives in one reserved band container laid out as a wrapping flex row, never positioned independently over the stage or another control. Put #swatches and #hint in one band element and size #stage.controls bottom from it. At about 390px wide, the swatches (about 182px from the left) and the hint (about 280px from the right) overlap by about 80px.
```
