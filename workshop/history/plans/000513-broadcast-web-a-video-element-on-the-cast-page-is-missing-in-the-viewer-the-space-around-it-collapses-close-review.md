# Boundary Review — pair#513 (whole-issue close)

| field | value |
|-------|-------|
| issue | 513 — broadcast-web: a video element on the cast page is missing in the viewer (the space around it collapses) |
| repo | pair |
| issue file | workshop/issues/000513-broadcast-web-a-video-element-on-the-cast-page-is-missing-in-the-viewer-the-space-around-it-collapses.md |
| boundary | whole-issue close |
| milestone | — |
| window | ee1edde5f08bdb4411af2570e3df6476a1223a70..b8ebaf89d7604bd9543746fb6d7d31bb23222a6c |
| command | sdlc close --issue 513 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T18:56:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff read; now checking how layout boxes are collected (relevant to a possible `hidden`-element bug) and running the tests.

Tests pass. Now reverting the recorder fix to confirm its test goes red, then trying the live Chrome test.

All checks are done. Writing up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary.** The root cause is found, measured and fixed in the right place. CDP only sends `childNodeInserted` under a parent whose children it has already sent. The empty `div` that asciinema-player later fills only ever produced `DOM.childNodeCountUpdated`, and the mirror ignored that event. The fix sends it through the existing `Expand` → `DOM.requestChildNodes` path (`recorder/mirror.go:855-865`).

I confirmed the regression test goes red without the fix: I reverted `mirror.go` to base, `TestMirrorExpandsANodeFilledAfterItsInsert` failed with `expand []`, and I restored the file. All webcast Go packages and all 100 node tests pass.

The later scope addition has two correctness gaps:
- **Hidden elements become visible.** The viewer's zero-specificity `display:inline-block` rule is author-origin, so it beats the browser's own `[hidden]{display:none}`. A `<canvas hidden>`, `<video hidden>` or `<iframe hidden>` now shows up in the viewer as a box (300x150 for a canvas, or its width/height attributes) and gets a "not captured" mark.
- **Shadow DOM is missed.** The size rule only goes into the document's sheet, so canvases and videos inside shadow roots still collapse.

`## Done when` also no longer covers the revised scope. All of these are cheap to fix.

I couldn't run `TestLiveReplacedElements` (`WEBCAST_LIVE_CHROME=1`) here: Chrome's GPU process crashed on start in this sandbox. Its claims (sizes match the cast, fails without the fix) are unverified by me.

Done-when: stale — it still names only the blog's demo video and "a test with a video page fails before the fix". It doesn't cover what the Revisions added and this diff builds: placeholders for every uncaptured kind, live sizing with aspect ratio, and the operator's second case (the ariadne-2-5 iPad `<video>` with no box in Safari).

### 1. Strengths
- **Root cause measured before fixing.** The cause was traced from the operator's recording (inserts at record offsets 621634, 2557975 and 2639691 in the recording file) and from a separate headless Chrome. The fix reuses the existing `Expand` path instead of adding a new mechanism (`mirror.go:864` mirrors `index` at `mirror.go:274`).
- **The regression test covers the negative cases too:** no expand for an empty insert, and no second request once the count matches what the mirror holds (`mirror_test.go:335-364`).
- **`placeholders.js` stays small and testable.** It takes the viewer's surface (`uncaptured`, `boxOf`, `fromPage`, `onRecord`) as a parameter and is tested against a fake DOM. The test covers zero boxes, a box that grows and an element that goes away (`tests/webcast/placeholders.test.mjs`).
- **The sizing rule is designed carefully:** `:where()` keeps it at zero specificity so author CSS still wins, it uses border-box, and it carries `aspect-ratio` so author `height:auto` keeps the operator's height. The sheet is only rewritten when its text changes (`render.js:196`).
- **Docs are updated:** README has a user-facing "What a web cast can't show" paragraph and the atlas gains the recorder and renderer paragraphs.

### 2. Critical findings
None.

### 3. Important findings
- **`cmd/internal/webcast/web/render.js:192` — hidden elements are forced visible (ARCH-PURPOSE / correctness).**
  - **Cause:** `:where([data-couch-live-size=…]){display:inline-block}` lives in an adopted author sheet, and any author-origin rule beats the browser's own `[hidden]{display:none}`. A hidden element has no layout box, so `drawLiveSizes` falls back to its width/height attributes, or 300x150 for a canvas.
  - **Effect:** `<canvas hidden>`, `<video hidden>` or `<iframe hidden width=… height=…>` becomes a visible box that shifts the layout. Because `boxOf` reads the drawn rect, it also gets a "not captured" mark.
  - **Fix:** make the selector `:where([attr="id"]:not([hidden]))`. Also consider skipping the attribute fallback when a layout record exists (`model.boxesT > 0`) but has no box for the id, since that means the element wasn't laid out for the operator. Add the case to the live test's page.
- **`render.js:496` together with `render.js:192` — the size rule never reaches shadow trees (ARCH-PURPOSE, the class of "every uncaptured element").**
  - **Cause:** `drawLiveSizes` writes rules into `viewerSheet(n.ownerDocument)`. `drawAdopt` gives shadow roots only their recorded sheets, and selectors in a document sheet don't match inside shadow roots.
  - **Effect:** a canvas or video inside a web component (chart libraries, `<model-viewer>`, players built on shadow DOM) still draws as its empty fallback (0x18) or with no box. Its mark is drawn at that collapsed size.
  - **Fix:** add the viewer sheet (or a per-root live-size sheet) to shadow roots' `adoptedStyleSheets`, keyed by `n.getRootNode()`. Add a shadow-hosted canvas to `replacedPage`.
- **Issue `## Done when` is stale against `## Revisions`.** Add bullets for:
  - an element whose content the recording can't hold shows at the operator's live size with a "kind: not captured" mark, and follows size changes;
  - the iPad `<video>` sized only by metadata keeps its box in Safari.

### 4. Minor findings
- `placeholders.js:5-6`: the header comment still says "the renderer sizes a canvas"; it now sizes every uncaptured kind.
- Plan checklist item 2 names `drawCanvasSizes`; the code and atlas use `drawLiveSizes`.
- `mirror.go:864`: `Expand` isn't de-duplicated. Several `childNodeCountUpdated` events in one batch, or a stale intermediate count processed after `setChildNodes`, each trigger a `requestChildNodes` with depth -1 and a full-subtree `Kids` record. This is bounded duplication, not wrong output (ARCH-CONSTRAINTS).
- `viewer.js:156`: `placeholders.draw()` runs on every shell paint and calls `getBoundingClientRect` per uncaptured element, which forces layout each frame. That's fine at today's counts; note it if pages have many canvases.
- `render.js:322-323`: `UNCAPTURED.has(el.localName)` is checked twice on adjacent lines; merge them into one `if`.
- `render.js:192`: the id and box numbers are interpolated straight into CSS text. They are numbers today because the reducer validates records; coercing with `Number()` there would keep that true by construction (ARCH-SECURE, low).
- `server.go:72-73`: whitespace-only gofmt realignment, unrelated to this issue.

### 5. Test coverage notes
- **Recorder fix:** has a default-run unit test, confirmed red without the fix.
- **Placeholders:** has a default-run node test.
- **Renderer (`drawLiveSizes`, `uncaptured()`):** covered only by the opt-in live test. That matches how the repo already tests its renderer, but it means the attribute fallback, the hidden case and the shadow case are never exercised by a default run. Pulling the rule-text computation out into a pure function (records in, CSS text out) would make all three unit-testable (ARCH-PURE).
- **Not reproduced here:** the live test (Chrome GPU crash in this sandbox), the Done-when claim about the blog's demo video, and the replaced-element sweep. The sweep is recorded in the Log.

### 6. Architectural notes
- **Pass:**
  - ARCH-DRY: one `UNCAPTURED` set, one `label`.
  - ARCH-PURE: placeholders get their surface injected.
  - ARCH-MOCK: stateful fake page for the mirror; real Chrome for the live test.
  - ARCH-ORDER: marks are redrawn on every record and every paint; disconnected ids are pruned.
  - ARCH-FUNERAL: `uncapturable`, `sized` and `marks` are pruned on every draw and cleared on `reset`.
- **Minor:** ARCH-CONSTRAINTS (Expand duplicates, per-paint layout reads) and ARCH-SECURE (numeric coercion).
- **Flagged:** ARCH-PURPOSE (hidden and shadow gaps).
- **For #515 (canvas snapshots):** the same per-root sheet and selector seam should carry the snapshot image, so fix the shadow-root gap first.

### 7. Plan revision recommendations
- Rename `drawCanvasSizes` → `drawLiveSizes` in Plan item 2 and say it covers every uncaptured kind.
- Add a `## Revisions` entry recording the hidden-element and shadow-root handling once fixed, and extend `## Done when` as described above.

```findings
findings:
  - id: new
    severity: Important
    family: viewer-rule-overrides-ua-hiding
    title: |
      Live-size rule's display:inline-block (author origin) beats UA [hidden]{display:none}: hidden canvas/video/iframe drawn and marked
    detail: |
      render.js:192. A hidden element has no layout box, so it falls back to width/height attrs (canvas 300x150), shows as a phantom box and gets a "not captured" mark. Fix: :not([hidden]) in the selector; skip the attr fallback when a layout record lacks the id; add to the live test page.
  - id: new
    severity: Important
    family: uncaptured-class-coverage
    title: |
      Live-size rule lives only in the document sheet; uncaptured elements inside shadow roots stay collapsed
    detail: |
      drawAdopt (render.js:496) gives shadow roots no viewer sheet, and document rules don't pierce shadow trees, so a canvas in a web component still draws as its 0x18 fallback. Fix: adopt a live-size sheet per root (n.getRootNode()); add a shadow-hosted canvas to replacedPage.
  - id: new
    severity: Important
    family: done-when-tracks-revisions
    title: |
      Done when not updated for the Revisions scope (placeholders and live sizing for every uncaptured kind, iPad video case)
    detail: |
      Done when names only the blog demo video and a failing video test; add bullets for the marks plus live-size behaviour and for the metadata-sized video.
  - id: new
    severity: Minor
    family: stale-naming-in-docs
    title: |
      Plan item names drawCanvasSizes (code: drawLiveSizes); placeholders.js header still says the renderer sizes "a canvas"
  - id: new
    severity: Minor
    family: redundant-cdp-requests
    title: |
      Expand not de-duplicated: repeated or stale childNodeCountUpdated each trigger a depth -1 requestChildNodes and a full-subtree Kids record
  - id: new
    severity: Minor
    family: renderer-logic-live-only-tests
    title: |
      drawLiveSizes / uncaptured() covered only by the opt-in live Chrome test; extract a pure rule builder for default-run tests
```

---

## Re-review — 2026-10-12T19:08:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 513 — broadcast-web: a video element on the cast page is missing in the viewer (the space around it collapses) |
| repo | pair |
| issue file | workshop/issues/000513-broadcast-web-a-video-element-on-the-cast-page-is-missing-in-the-viewer-the-space-around-it-collapses.md |
| boundary | whole-issue close |
| milestone | — |
| window | 605445b697f4e8845ba50ff7ff434c31b1f967f4..447ed9c0c64fb362ae2ba204f20428fb74b271ff |
| command | sdlc close --issue 513 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T19:08:40-07:00 |
| verdict | SHIP |

## Review

Prior-round fixes look in place; running the node and Go test suites now.

```verdict
verdict: SHIP
confidence: medium
```

All six round-1 findings are fixed, and I checked the code and tests for each rather than relying on the commit message. The recorder fix (a `DOM.childNodeCountUpdated` for children the mirror doesn't hold goes on `Expand`) is the real root cause of the blog embed collapsing, and a default-run test covers it. The live-size rule now skips `[hidden]` elements and elements that have no layout box, and it goes into a separate stylesheet for each document or shadow root. The rule builder is now a pure function with default-run node tests. README and atlas both cover the new behaviour. Node tests pass (103/103), and `go test ./cmd/internal/webcast/...` passes. **One gap:** I could not run the opt-in `TestLiveReplacedElements` here, because Chrome aborts in this sandbox ("GPU process isn't usable"). So the shadow-root and hidden-canvas checks are reviewed from the code only; I didn't see them run. Nothing blocks shipping.

**1. Strengths**
- `render.js:50` `liveSizeRule` is pure and its inputs are coerced: `w > 0 && h > 0` and `Number(attrs.*)` stop non-numeric box values from the stream from being injected into the CSS text (ARCH-SECURE passes).
- `mirror.go:856-870`: `Expand` is only queued when the count differs from the children held, and only once while queued. `TestMirrorExpandsANodeFilledAfterItsInsert` covers the empty insert, repeated counts, the fill, and a matching count.
- `drawLiveSizes` (`render.js:197`) gathers rules per `getRootNode()`, clears a root that no longer has any rules, and only calls `replaceSync` when the text changes. The cost per draw scales with the number of uncaptured elements.
- `placeholders.js` reconciles marks by id (adds, moves, removes) and is tested with a fake DOM, with no browser needed.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- `render.js:228` and `atlas/webcast.md` say the sheet sits "before the page's own sheets… so on a tie the page's CSS wins". That only holds against the page's adopted sheets. Adopted sheets cascade after `<style>` and `<link>` sheets, so a zero-specificity page rule (for example `*{display:block}`) loses to the live-size rule. The edge case is rare, but the comment overstates what the code does.
- `liveSizeRule`'s `layoutKnown = this.model.boxesT > 0` treats a layout record stamped `t=0` as no layout yet. A hidden canvas would then fall back to its attributes until the next layout record arrives. This is a narrow nit.

**5. Test coverage notes**
- For BR-1, removing `:not([hidden])` or the `layoutKnown` guard makes `placeholders.test.mjs` fail (the regex and the "laid out without a box" assertions).
- The BR-2 per-root routing is covered only by the opt-in live test. The issue log says it was mutation-checked, but I couldn't reproduce that here.

**6. Architecture**
- **ARCH-DRY:** pass. `liveSheet` and `viewerSheet` share a small pattern, which is acceptable.
- **ARCH-PURE:** pass. The rule logic is pure and the per-root routing is a thin layer around it.
- **ARCH-PURPOSE:** pass. It fixes both the blog embed and the iPad video sized only by metadata, and the marks cover every uncaptured kind.
- **ARCH-MOCK:** pass. There's a fake page for the mirror, and the live Chrome test is opt-in.
- **ARCH-CONSTRAINTS:** pass. Per-frame work is bounded by the uncaptured count, and the `Expand` is de-duplicated.
- **ARCH-SECURE:** pass, as described under Strengths.
- **ARCH-ORDER:** pass. The marks map and the `uncapturable` set are reconciled on each draw, and the expand queue is idempotent.
- **ARCH-FUNERAL:** pass. Marks go when their element goes, sheets die with their roots, and `uncapturable` drops disconnected nodes.

**7. Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      :not([hidden]) in liveSizeRule (render.js:56), no attr fallback once layoutKnown; pinned by placeholders.test.mjs regex and the laid-out-without-box case; live page adds hiddencv.
  - id: BR-2
    disposition: addressed
    note: |
      Per-root live sheet via getRootNode() and adoptLive/drawAdopt; live test adds shadow-hosted canvas (not runnable in this sandbox, reviewed by code).
  - id: BR-3
    disposition: addressed
    note: |
      Done when gains the Revision bullet covering live size, iPad metadata video, shadow roots, hidden, and marks.
  - id: BR-4
    disposition: addressed
    note: |
      Plan names drawLiveSizes; placeholders.js header says each such element.
  - id: BR-5
    disposition: addressed
    note: |
      slices.Contains de-dup while queued plus count != len(Children) guard; test asserts one Expand and no re-ask on matching count.
  - id: BR-6
    disposition: addressed
    note: |
      liveSizeRule and isUncaptured exported pure with default-run node tests.
findings:
  - id: new
    severity: Minor
    family: docs-overstate-behavior
    title: |
      "On a tie the page's CSS wins" holds only against adopted sheets, not style/link sheets
    detail: |
      render.js:228 and atlas/webcast.md: adopted sheets cascade after the document's style and link sheets, so a zero-specificity page rule ties and loses to the live-size rule. Reword, or note the limit.
  - id: new
    severity: Minor
    family: viewer-rule-overrides-ua-hiding
    title: |
      layoutKnown = boxesT > 0 treats a layout at t=0 as no layout
    detail: |
      This is the 2nd finding in family viewer-rule-overrides-ua-hiding. The rule covering both: the viewer's sizing applies only to elements the operator's layout rendered. layoutKnown should come from whether a layout record arrived (e.g. a flag), not from its timestamp.
```
