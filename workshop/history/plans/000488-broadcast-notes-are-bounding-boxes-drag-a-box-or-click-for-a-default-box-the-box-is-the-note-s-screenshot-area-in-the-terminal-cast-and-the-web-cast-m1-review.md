# Boundary Review — pair#488 (milestone M1)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 618ba11e4ace65761025cb35aa3119ab5ea40226..7dc2ba7cf21f72c2d7fac8e1a369af7c13cde636 |
| command | sdlc milestone-close --issue 488 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T11:38:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 diff; reading the rest of the notes/recording and JS changes now.

Running the Go and node test suites for the touched packages.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 does what the plan says for this milestone. The shared `noteBox` gesture function is in `collab.js` and pure-tested in node. The web cast now always sends a box (`region`). The board requires it and clamps it server-side, which closes #470 BR-4. Pins are outlined boxes. `NoteAt` gains `kind`/`path`. `NoteCrops` is now "box plus 16 px, merge only on overlap, clip to page", with snapping, `Crops` and `Hints` retired. A grep found no consumers of the removed API left in the tree. `go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/...` passes, and so do all 165 node tests (the Go harnesses glob in the new `notebox.test.mjs`). `go build ./...` stops on a missing embedded runtime asset in `runtimebundle`. That package is outside this diff, so I treat it as a gap in this checkout, not something the diff caused. One thing should change before crossing the boundary: README.md still describes the old web note gesture ("tap an element, or drag a region"). That is an Important docs-gate miss and cheap to fix. Everything else is Minor.

**1. Strengths**
- `cmd/internal/broadcast/web/collab.js:127` `noteBox` is a single pure function for both casts and both units. Taps shift rather than shrink, a drag in cells covers every cell it touched, the box is at least the 2-cell pin, and touch/mouse thresholds come from one `DRAG_MIN` table. The table tests in `tests/broadcast-viewer/notebox.test.mjs` pin each of these rules, including edges and the scrolled view.
- Untrusted input is handled server-side. `board.ParseNote` now requires `region`, and null, 3-element, NaN, huge and empty regions are each rejected. `Board.Note` clamps the box with `notes.ClampToPage` and refuses one that is entirely off the page. `FuzzParse` gained truncated and negative-region seeds and an invariant on `Region` (wire.go:151, board.go:186, wire_test.go:90).
- Retiring the old crop logic made the code smaller and removed duplication. `FakeModel` no longer re-implements crops and calls the shared `noteCrops` (fake.go:63), which deletes about 40 lines of parallel logic (ARCH-DRY).
- Older recordings still work: `Region` stays `omitempty`, `noteBox` in crop.go falls back to `at.box` and then the element's layout box, and that fallback is tested in `TestOlderNoteCropsItsAnchorBox`.
- The merge rule uses a strict `gap < 0`, and a test covers margins that only touch (32 px apart, so not merged), which is exactly where an off-by-one would hide.

**2. Critical findings**
None.

**3. Important findings**
- **README.md:1017–1020 is out of date for the web note gesture.** It still says "💭 (note: tap an element, or drag a region, then write)" and "A note pins to its element (or region)". In M1, every note is a box: a tap gets the default box (a tenth of the view each way), a finger needs 16 px to count as a drag, and the pin is the outlined box with its 💭 inside the top-right corner. Fix: rewrite those two sentences to match the atlas text at `atlas/webcast-format.md:408–416`.

**4. Minor findings**
- `atlas/webcast-format.md:206–209,216` and the `recording/record.go:355` doc comment refer to `broadcast.NoteAnchor` and `atlas/broadcast.md`, and describe terminal notes as already having cell boxes and a layout anchor. None of that exists yet; it is M2/M3 work. Either word it as planned work or add it when M2/M3 land.
- `board.go:34` `ErrStale` doc comment ("another document, or the element is gone") doesn't cover the new "box is off the page" refusal. Users see "the page changed" for it. That's acceptable, but the comment should list the case.
- Three different `noteBox` functions now exist: gesture→box in collab.js, note→display box in annotate.js, and note→crop box in crop.go. annotate.js has to import the first as `boxOfGesture` to avoid a clash. Renaming the collab one to something like `gestureBox` would help readers.
- `annotate.css:17`: the 26 px chip now sits inside the box, so a thin drag (for example a 12×1 px box, which is legal) leaves the chip spilling well outside its box.
- `FakeModel.NoteCrops` now delegates to the real `noteCrops`, so running `RunModelContract` over both can no longer catch the two diverging. That is acceptable for an in-process pure model; just noting it.
- The plan names the test file `tests/broadcast-viewer/notes.test.mjs`; the shipped file is `notebox.test.mjs`.
- `pathOf`'s `stepName` keeps any id of 40 characters or fewer, including generated-looking ones (`:r1:`), which `selectorFor` deliberately avoids. That adds noise to the agent's path, but it isn't a correctness problem.

**5. Test coverage notes**
- The pure layers are well covered: the gesture function, `regionOf` wrappers for mouse and touch, `ClampToPage` tables, the merge/clip/margin crop rules, the contract's new kind/path and crop assertions, `pathOf` including the depth cap, and wire refusals plus fuzz.
- `mountAnnotate`'s glue isn't exercised by any test. That covers `pageView()` (box rect → page px), capturing `pointerType` on pointer-down, and `boxCentre` → `nodeAt`. This gap predates #488 (there's no DOM harness for annotate). It's worth adding when M2 builds the terminal pointer-mode tests, since `pointermode.test.mjs` already drives pointer events.

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** Gesture→box is one shared function, and the fake's duplicate crop code is gone. The only issue is the naming one listed under Minor.
- **ARCH-PURE: pass.** `noteBox`, `ClampToPage`, `noteCrops`, `merge` and `pathOf` are pure and tested without IO. The board calls `ClampToPage` as plain glue.
- **ARCH-PURPOSE: pass for M1.** The box is exposed through `NoteCrops`, the seam #483 will crop from, which satisfies "exposed through the notes seam" in Done-when. The terminal half is properly scheduled as M2/M3, not deferred.
- **ARCH-MOCK: pass.** No new external dependencies. The fake sharing the real crop rule is noted under Minor.
- **ARCH-CONSTRAINTS: pass.** The merge is O(n²) over a few notes, capped at 8 per color. The path is capped at 6 steps and each name at 40 characters.
- **ARCH-SECURE: pass.** The region is parsed strictly into a typed `Box` and clamped against board state under the lock. Old recordings without `region`/`kind`/`path` are read safely. When M3 adds `view.json`, apply the same strict parse and keep the "error, never fabricate" rule from the plan.
- **ARCH-ORDER: pass.** No new state is carried between events; the clamp runs against `b.st` under `b.mu` in the same critical section as the node check. M3's async resolver is where this principle will really be tested, and the plan's generation/epoch checks have to be backed by injected-ordering tests.
- **ARCH-FUNERAL: pass.** Nothing new is durable. Each note record grows by `region` (about 30 bytes) and `path` (at most about 6×45 bytes), which is bounded and rate-limited by the existing 2 notes/s budget.

**7. Plan revision recommendations**
- Add a `## Revisions` line: the M1 test file is `tests/broadcast-viewer/notebox.test.mjs`, not `notes.test.mjs`.
- Optional: note that the cells branch of `noteBox` ships in M1 with no consumer until M2's viewer drag uses it.

```findings
findings:
  - id: new
    severity: Important
    family: docs-gate-readme-surface
    title: |
      README web-cast notes paragraph still says "tap an element, or drag a region" and pins "to its element (or region)"
    detail: |
      README.md:1017-1020. M1 changed the user-facing gesture: every note is a box, a tap gets the default 1/10 box, touch needs 16 px, and the pin is the outlined box with the chip inside its top-right corner. Update the paragraph to match atlas/webcast-format.md:408-416.
  - id: new
    severity: Minor
    family: docs-describe-unbuilt-surface
    title: |
      Atlas and record.go doc comment reference broadcast.NoteAnchor and terminal cell boxes, which do not exist until M2/M3
    detail: |
      atlas/webcast-format.md:206-209,216 and cmd/internal/webcast/recording/record.go:355 describe the terminal anchor and boxes in the present tense; grep finds no NoteAnchor type. Word them as planned, or add them when M2/M3 land.
  - id: new
    severity: Minor
    family: error-doc-drift
    title: |
      ErrStale doc comment not updated for the new "box off the page" refusal
    detail: |
      board.go:34 lists another document or a gone element; board.go:189 now also returns ErrStale when ClampToPage finds nothing on the page.
  - id: new
    severity: Minor
    family: name-collision-across-layers
    title: |
      Three distinct noteBox functions (collab gesture, annotate display, Go crop); annotate imports one as boxOfGesture
    detail: |
      Rename the collab.js gesture function (e.g. gestureBox) so each name reads as one meaning.
  - id: new
    severity: Minor
    family: ui-chip-fit
    title: |
      26 px pin chip now sits inside the box and spills well outside thin boxes (e.g. a 12x1 px drag)
  - id: new
    severity: Minor
    family: glue-untested
    title: |
      mountAnnotate glue (pageView, pointerType capture, boxCentre to nodeAt) has no DOM-level test
    detail: |
      Gap predates #488; pointermode.test.mjs shows a harness that M2 could extend to annotate.
```

---

## Re-review — 2026-10-12T11:43:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 618ba11e4ace65761025cb35aa3119ab5ea40226..9fa55c94b294a1eab258c0105fbce0b87464a3a9 |
| command | sdlc milestone-close --issue 488 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T11:43:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the fix commit against prior findings, then running the tests.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers the web-cast part of #488. There is one shared gesture rule (`collab.js` `gestureBox`), every note now carries a required `region`, the server clamps that box to the page (this closes #470 BR-4), every pin is drawn as an outlined box, and anchors gain `kind`/`path`. Crops are now the box plus 16 px, and snapping and `Hints` are retired. All Go packages under `webcast/` and `broadcast/` pass, and the 16 node tests pass. The fix commit 9fa55c9 resolves BR-1, BR-3 and BR-4.

One thing blocks a clean SHIP. `viewer_annotate_live_test.go` is the Chrome-gated end-to-end test that drives `mountAnnotate`, and it was not updated. It still selects `.pin.region`, a class M1 removed. It also expects a tapped note's pin to match its element's box, but a tap now gets the default 1/10-view box. Chrome would not launch here (`cdp: read: EOF`), so I confirmed this by reading the code, not by a red run.

1. **Strengths**
   - `collab.js:126` `gestureBox` is one pure source for both units. Its cells and px variants are pinned by `notebox.test.mjs`, which covers tap, drag, clamp, shift-not-shrink and the touch threshold (ARCH-DRY, ARCH-PURE).
   - `crop.go`'s `noteCrops` is shared by `Structural` and `FakeModel`. This removes the fake's divergent single-crop rule, and the contract test now pins merge-by-overlap and the 16 px margin.
   - `wire.go` `ParseNote` makes `region` required and rejects null, 3-element, huge and zero-width regions. The fuzz bounds also check `Region[2..3] ≥ 1`, and the seeds include a truncated and a negative box (ARCH-SECURE).
   - `board.go:186` refuses a box that is entirely off the page with `ErrStale`, rather than inventing one. Both the clamp and the refusal are tested.
   - `recording.Note.Region` stays `omitempty`, and `noteBox` falls back to `at.box` for notes recorded before #488, so old recordings still read.

2. **Critical:** none.

3. **Important**
   - `cmd/internal/webcast/viewer_annotate_live_test.go:146,128-135`: the live two-viewer test still expects the pre-#488 behaviour.
     - Line 146 waits for `.pin.region`, but `annotate.js:222` now sets the class to plain `pin`, so the wait can never succeed.
     - Lines 128-135 expect A's tapped note to match the element box `nb` within 2 px, but a tap now gets the default box.
     - Fix: select `.pin[data-note=<Bo's id>]`, and expect A's pin at `noted.Region` rather than `nb`.
     - This is a retired-surface sweep that missed a gated test (ARCH-PURPOSE). The rule: when a class, function or field is retired, grep every consumer, including env-gated tests and atlas text, in the same commit.

4. **Minor**
   - `atlas/index.md:123` still advertises "snapped crops". `atlas/webcast-format.md:241` still says the parent fallback is "not for snapping". Same family as the Important finding.
   - `atlas/webcast-format.md:217` still says a terminal note has "a box in cells and Couch's layout anchor" in the present tense. Those arrive in M2/M3 (BR-2 residue).
   - BR-5 (the 26 px chip overflowing thin boxes) is unchanged.

5. **Test coverage**
   - Node: the gesture rule, `regionOf` for mouse and touch, and the `noteDraft` shape are covered.
   - Go:
     - wire refusals and fuzz;
     - board clamp and refusal;
     - the crop contract and its rewritten tests.
   - The only DOM-level coverage of `mountAnnotate` (`pageView`, `pointerType` capture, `boxCentre` → `nodeAt`) is the stale live test. It needs to be green again before the operator smoke test relies on it.

6. **Architecture**
   - ARCH-DRY: pass. One gesture rule and one crop rule.
   - ARCH-PURE: pass. Geometry, clamp and crops are pure; `pageView` is thin glue.
   - ARCH-PURPOSE: flag. The retirement sweep missed the live test and two atlas lines (see above).
   - ARCH-MOCK: pass. The fake now shares the real crop rule; no new external dependency.
   - ARCH-CONSTRAINTS: pass. Merge is O(n²) over at most 64 notes on the submission path, not a hot path.
   - ARCH-SECURE: pass. The viewer's box is parsed strictly and clamped on the server, and old recordings are tolerated.
   - ARCH-ORDER: pass for M1. The board's `pending` handling is unchanged. M3's async anchor resolver is where this check will matter.
   - ARCH-FUNERAL: pass. Nothing durable is new; `region` adds a fixed 4 floats per note record.

7. **Plan revisions:** none needed. The plan's `notes.test.mjs` name is really `notebox.test.mjs`, which is cosmetic.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1014-1022 now describes drag-or-tap-for-default-box, the touch threshold, and the outlined box with the chip inside top-right, matching annotate.css and collab.js DRAG_MIN.
  - id: BR-2
    disposition: not-addressed
    note: |
      atlas:206-209 and record.go:355 fixed; atlas/webcast-format.md:217 still states terminal notes have a cell box and layout anchor in present tense (M2/M3).
  - id: BR-3
    disposition: addressed
    note: |
      board.go:34-35 now lists the off-page box as an ErrStale cause, matching board.go:186-189.
  - id: BR-4
    disposition: addressed
    note: |
      collab.js gesture function renamed gestureBox; annotate.js imports it unaliased; the remaining noteBox (JS display, Go crop.go) now share one meaning.
  - id: BR-5
    disposition: not-addressed
    note: |
      Unchanged; annotate.css still fixes a 26 px chip at right:2px top:2px regardless of box size. Minor, non-blocking.
  - id: BR-6
    disposition: withdrawn
    note: |
      A DOM-level test of mountAnnotate exists (viewer_annotate_live_test.go, Chrome-gated); the real gap is that it is stale, raised as a new finding.
findings:
  - id: new
    severity: Important
    family: retired-surface-sweep
    title: |
      Chrome-gated live annotate test still asserts the retired .pin.region class and element-sized tap pins
    detail: |
      viewer_annotate_live_test.go:146 waits for .pin.region, which annotate.js:222 no longer sets, and :128-135 expects A's tapped pin at the element box nb, but a tap now gets the default 1/10-view box. Select .pin[data-note=id] and expect noted.Region. Rule: when retiring a class, function or field, grep all consumers, including env-gated tests and the atlas, in the same commit.
  - id: new
    severity: Minor
    family: retired-surface-sweep
    title: |
      atlas still mentions crop snapping after #488 retired it
    detail: |
      atlas/index.md:123 lists "snapped crops"; atlas/webcast-format.md:241 says the parent fallback is "not for snapping". Same sweep rule as the live-test finding.
```

---

## Re-review — 2026-10-12T11:50:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 618ba11e4ace65761025cb35aa3119ab5ea40226..9bfeae3c25ebc4e24fdcc359c19d1e0bd87997a6 |
| command | sdlc milestone-close --issue 488 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T11:50:20-07:00 |
| verdict | SHIP |

## Review

Focus first on open findings: last commit.

Tests pass for the webcast and broadcast packages. Next I'm running the Chrome-gated live test to check BR-7.

Wrapping up: checked the open findings and the retired-surface sweep, and ran the tests. Last, a look at the README diff.

```verdict
verdict: SHIP
confidence: medium
```

The M1 code delivers what the plan promises:

- **Shared gesture code:** one `gestureBox`/`dragMin`/`boxCentre` in `collab.js`. The web cast uses it and the terminal cast will use it in M2.
- **Box on every note:** the server requires a `region` and clamps it to the page (`ParseNote` and `ClampToPage`, which closes #470 BR-4).
- **Pins:** every pin is outlined; the `.pin.region` class is gone.
- **Anchor:** `NoteAt` now has `kind` and `path`.
- **Crops:** `NoteCrops` is now the box plus 16 px, merged only where crops overlap. Snapping, `Crops` and `Hints` are retired, and nothing in `cmd/` still references them.

Build checks:

- `go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/...` passes. `go vet` was not clean: it failed on `cmd/internal/runtimebundle/embed.go` because of a missing gitignored asset, which has nothing to do with this diff.
- The node tests `notebox.test.mjs` and `annotate.test.mjs` pass, 16 of 16.

**I could not run the Chrome-gated live test here.** `WEBCAST_LIVE_CHROME=1 go test -run TestAnnotateLiveTwoViewers` fails while setting up the browser (`Target.createTarget: cdp: read: EOF`). That is a problem with this environment, not a test assertion. So BR-7 is judged addressed from reading the code only. Two Minor findings from earlier rounds are still open (BR-2 at two lines, BR-5 unchanged). Neither blocks.

1. **Strengths**
   - `collab.js:110-152`: one pure, unit-tested source for the gesture box in both casts (ARCH-DRY, ARCH-PURE). `tests/broadcast-viewer/notebox.test.mjs` covers taps, clamping at screen edges, very small screens, drags in either direction, and the larger touch threshold.
   - `notes/fake.go:62-64`: the fake now calls the real `noteCrops`, so the crop rules have one source and can't drift between fake and real.
   - Untrusted input is handled server-side. `ParseNote` requires and validates `region` (bounds, w/h ≥ 1). `Board.Note` clamps it and refuses a box that is entirely off the page. The fuzz test now asserts the region bounds too.
   - `crop.go:noteBox` still falls back to the old anchor box when a note has no `region`, so recordings made before this change still produce crops.
   - `workshop/lessons.md` now states the rule for the `retired-surface-sweep` family, and I confirmed the sweep: the code, atlas and README no longer refer to `MIN_REGION`, `.pin.region`, `Crops(`, `notes.Hints` or crop snapping, except where they say it was retired.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-2 is still open in two places: `atlas/webcast-format.md:217` and `recording/record.go:356-357`. Details in the disposition below.
   - BR-5 is unchanged: `annotate.css:17` puts a 26 px chip at `right: 2px; top: 2px` inside every box, including a 50×1 px drag.
   - The plan still says `noteBox` and `notes.test.mjs` (lines 32, 41, 55, 102, 149). The code uses `gestureBox` (renamed under BR-4) and `notebox.test.mjs`.

5. **Test coverage**
   - The live test's new check (`viewer_annotate_live_test.go:129`) assumes the tap's default box is centred within 1 px on the target link. `gestureBox` shifts the box to keep it on screen, so if `hoverTarget` picks a link within about 36 px of the viewport edge, this check would fail even though nothing is wrong. It is worth running the gated test once on a machine where Chrome starts.
   - The board tests cover the clamp, the off-page refusal and the required region. The contract test covers the 16 px margin, overlap merging, and old notes without a box.

6. **Architecture**
   - Pass on all eight principles:
     - **ARCH-DRY:** one gesture helper, and the fake shares the real crop code.
     - **ARCH-PURE:** the box, clamp and crop logic are pure; the IO glue in `mountAnnotate` is thin.
     - **ARCH-PURPOSE:** M1 is the web half of the issue, and M2/M3 (the terminal cast) are separate milestones of the same plan, not a deferred "follow-up".
     - **ARCH-MOCK:** `FakeModel` sits behind the same seam; the live Chrome test is the conformance check.
     - **ARCH-CONSTRAINTS:** viewer boxes are bounded at parse time.
     - **ARCH-SECURE:** the region is parsed into a typed value and clamped at the boundary, and older recordings are handled.
     - **ARCH-ORDER:** the only state carried between events is `state.drag`, which holds `pointerType` from pointer-down to pointer-up.
     - **ARCH-FUNERAL:** nothing new is stored. Each note record gains a 4-number `region`, a small fixed amount per note.
   - `notes.Model` changed in a breaking way (`Crops` and `Hints` removed, `NoteCrops` lost its `hints` parameter). The plan's Revisions record that pair:2 (#483) was told through the hub; confirm before merge.

7. **Plan revisions**
   - Add a `## Revisions` entry: the shared helper is `gestureBox(from, to, view, {min, cells})` with `dragMin(unit, pointerType)` and `boxCentre` (renamed from `noteBox` per BR-4), tested in `tests/broadcast-viewer/notebox.test.mjs`.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      atlas/webcast-format.md:206-209 now says the path is one "#488 M3 adds", but :217 still says in the present tense that a terminal note has "a box in cells and Couch's layout anchor", and record.go:356-357 says #488 "gives those a layout anchor". Neither exists until M2/M3. Reword both as planned (M2/M3).
  - id: BR-5
    disposition: not-addressed
    note: |
      annotate.css:17 still puts the 26 px chip at right:2px top:2px inside every box. The 50x1 px drag in annotate.test.mjs still gets a chip far larger than the box, and no rule for small boxes was added.
  - id: BR-7
    disposition: addressed
    note: |
      viewer_annotate_live_test.go:123-160 now checks noted.Region, centres the expected pin on that box, and waits on .pin[data-note=id]. The :240 check expects kind/path and a region. Verified by reading only: Chrome would not start in this checkout (cdp EOF at createTarget). lessons.md records the sweep rule.
  - id: BR-8
    disposition: addressed
    note: |
      atlas/index.md:123 now says "crops that are the note's box plus a margin (#488)". atlas/webcast-format.md:240-241 now says the landmark is "used for the context's text but not named in an anchor's path". landmark.go:57 was updated to match. A grep for snap* finds only "retired" or "never snapped" wording.
findings:
  - id: new
    severity: Minor
    family: plan-names-match-code
    title: |
      The plan still names the shared helper noteBox and its test file notes.test.mjs; the code has gestureBox and notebox.test.mjs
    detail: |
      Plan lines 32, 41, 55, 102 and 149 say noteBox(from, to, view, min). The code's collab.js:127 is gestureBox(from, to, view, {min, cells}) plus dragMin and boxCentre, renamed under BR-4. Add a Revisions entry so M2 builds against the real name.
```
