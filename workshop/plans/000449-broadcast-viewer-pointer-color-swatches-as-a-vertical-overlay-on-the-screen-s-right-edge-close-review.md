# Boundary Review — pair#449 (whole-issue close)

| field | value |
|-------|-------|
| issue | 449 — Broadcast viewer: pointer color swatches as a vertical overlay on the screen's right edge |
| repo | pair |
| issue file | workshop/issues/000449-broadcast-viewer-pointer-color-swatches-as-a-vertical-overlay-on-the-screen-s-right-edge.md |
| boundary | whole-issue close |
| milestone | — |
| window | ca3544b71d744023ae9d7d23c0e6fafff7b7ec24..93616acc7c4a32dc82e7a676332d389c66e24a2a |
| command | sdlc close --issue 449 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T12:58:52-07:00 |
| verdict | SHIP |

## Review

I'm checking the docs for stale wording about the old swatch band.

```verdict
verdict: SHIP
confidence: high
```

The change matches the Spec. The swatches move into a fixed `#rail` that sits outside `#stage`. `fit()` runs on caps, on terminal resize and on window resize, and each time it docks the rail to the drawn `.xterm-screen` rect through `railAt`, a pure and tested function. `pointer-events: none` on the rail and `auto` on `.swatch` lets a point over the translucent backing fall through to the stage. Because the rail is outside the stage, a swatch tap can never start a stroke. A drag that starts on the stage and crosses a swatch still belongs to the stage, which captures the pointer at `viewer.js` pointerdown. The rail comes after the stage in the DOM, and the fixed-position stage forms its own stacking context, so xterm's internal z-indexes can't cover the rail. The README and atlas are updated, and no stale "swatches below the screen" wording is left anywhere in the tree. Running `node --test tests/broadcast-viewer/*.test.mjs` gives 38/38 passing. Nothing blocks SHIP. The remaining notes are Minor.

1. **Strengths**
   - `viewer.js:127` `railAt` is pure and handles the edge cases: a letterboxed screen, a screen exactly as wide as the window, and an overflowing screen clamped by `Math.max(0, …)`. All three are tested in `rail.test.mjs:11-23`.
   - The rail is docked inside the existing `fit()`, so it follows every resize path (caps, `band()`, terminal resize, window resize) without new listeners.
   - The view page is unaffected: `#swatches` stays `hidden`, so the empty rail draws no backing.
   - The hint-only band still lifts the stage, and the comments were rewritten to match. `viewer.css:94` and `viewer.js:314` no longer claim the band holds every control.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The Done-when clause "pointing over the rest of the screen works as before (viewer test)" is pinned only by a regex over the CSS and HTML (`rail.test.mjs:25-34`), not by a behavioral test. That's acceptable for CSS hit-testing, which node can't run. The desktop and phone parts of the first clause depend on the logged headless checks plus the operator's smoke test, which hasn't happened yet.
   - `#stage.stale` dims and greys only the stage. The rail is outside it, so the swatches stay at full color on a stale screen. That's a small visual inconsistency and doesn't affect behavior.
   - At 14px, the phone swatches are small touch targets. The Spec allows the column to shrink, so this is only a note.

5. **Test coverage:** `railAt` is tested at its boundaries (inset at width equal to the window, clamped on overflow). The structural invariants — rail outside the stage, `pointer-events` on the rail and on `.swatch` — are pinned. Existing pointermode tests still cover swatch rendering and the remembered pick.

6. **Architecture**
   - **ARCH-DRY: pass.** Nothing is duplicated. `renderSwatches` and `colorStore` are reused unchanged.
   - **ARCH-PURE: pass.** The geometry is in the pure `railAt`. The DOM writes are three lines of glue in `fit()`.
   - **ARCH-PURPOSE: pass.** Every Spec bullet is delivered: right edge, vertical centering, tracking resizes, translucency, pass-through, marked selection, phone width, and a column with room for #447. #447's icon can join as another child of `#rail`.

7. **Plan revisions:** none. The plan and Log match the code.

```findings
findings:
  - id: new
    severity: Minor
    family: done-when-clause-behavioral-evidence
    title: |
      Point-through-the-rail Done-when clause is pinned only by a CSS/HTML regex test
    detail: |
      rail.test.mjs:25-34 asserts the selectors, not hit-testing. This is acceptable because node cannot hit-test, but the evidence is structural, not behavioral. It is the only instance in this window.
  - id: new
    severity: Minor
    family: stale-state-styling-scope
    title: |
      Rail is outside the stage, so the stale dimming does not apply to the swatches
    detail: |
      The #stage.stale opacity and grayscale (viewer.css) skip #rail. If wanted, add a body-level stale class or a matching #rail rule. #rail is the only sibling of the stage that overlays the screen.
```

---

## Re-review — 2026-10-11T13:00:17-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 449 — Broadcast viewer: pointer color swatches as a vertical overlay on the screen's right edge |
| repo | pair |
| issue file | workshop/issues/000449-broadcast-viewer-pointer-color-swatches-as-a-vertical-overlay-on-the-screen-s-right-edge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4f08a427063b4914be1acd76f531e3c9ab612405..859f6bd37268dd1f1652e02ce261383baf3e46d2 |
| command | sdlc close --issue 449 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T13:00:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This fix window is small and correct. It covers commits 3d6142be and 859f6bd3: a card-mirror update, plus one change that dims the rail whenever the screen goes stale. BR-2 is fixed properly. The CSS rule now applies to `#stage.stale, #rail.stale` together (`viewer.css:72-78`), and `setStale` in `viewer.js:354-357` toggles `stale` on both elements. `rail` is defined in the same `start()` scope at `viewer.js:285`, so the toggle can reach it. The finding's class is "siblings of the stage that overlay the screen". I swept that class against `viewer.html:15-19`. `#rail` is the only member: `#status` is the stale indicator itself, and `#controls`/`#hint` sit in the bottom band. BR-1 has no fix, and the Log says so on purpose. It stays a non-blocking Minor. There are no conflict resolutions in this window. The issue-file change adds a status flip, `actual_hours` and Log entries, and leaves `## Spec` and `## Done when` alone, so the scope did not change. `node --test tests/broadcast-viewer/*.test.mjs` passes 39/39.

1. **Strengths**
   - `viewer.css:72-75` uses one shared selector list instead of a copied `#rail.stale` block, so the stale look is defined in one place (ARCH-DRY).
   - `viewer.js:354-357`: the fix lives in the injected `setStale` glue. The tested `connect` logic that decides *when* a screen is stale is untouched.
   - `rail.test.mjs:36-39` pins the shared rule. Deleting `#rail.stale` from the CSS turns the test red.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The JS half of the fix, `rail.classList.toggle`, has no test. That matches the untested `stage.classList.toggle` glue it sits beside, since `connect.test.mjs` injects a fake `setStale`. Two lines of DOM glue aren't worth adding a DOM harness for, so I'm not raising it as a finding.

5. **Test coverage:** The CSS half of the BR-2 fix has a regression test. The stale timing is still covered behaviourally by `connect.test.mjs`.

6. **Architecture (interdiff only)**
   - **ARCH-DRY: pass.** The fix extends one selector list rather than duplicating the rule.
   - **ARCH-PURE: pass.** The change is confined to the IO glue (`setStale`). No logic moved into it.
   - **ARCH-PURPOSE: pass.** The class sweep is done, and the rail is the only overlaying sibling.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Left unfixed on purpose and still Minor and non-blocking: node cannot hit-test, the pass-through is pinned structurally, and headless Chrome checked the layout per the Log.
  - id: BR-2
    disposition: addressed
    note: |
      viewer.css shares the rule as #stage.stale, #rail.stale and setStale (viewer.js:354-357) toggles both; rail.test.mjs:36-39 pins the CSS; #rail is the only stage sibling that overlays the screen.
```
