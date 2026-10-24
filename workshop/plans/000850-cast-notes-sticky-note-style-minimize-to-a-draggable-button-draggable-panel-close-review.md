# Boundary Review — pair#850 (whole-issue close)

| field | value |
|-------|-------|
| issue | 850 — Cast notes: sticky-note style, minimize to a draggable button, draggable panel, visible add-note on sketches |
| repo | pair |
| issue file | workshop/issues/000850-cast-notes-sticky-note-style-minimize-to-a-draggable-button-draggable-panel.md |
| boundary | whole-issue close |
| milestone | — |
| window | 221a833b2dc8504a9498d2f5b4b944b99ea7907d..1c2a16c82e6670aea3739eb5da1c18a2fa8ae73a |
| command | sdlc close --issue 850 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T19:59:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation preserves existing note and submission owners, adds useful regression coverage, and updates README and atlas. However, sketch-note controls can disappear while their ink remains visible, minimized buttons can grow outside the viewport, and mode changes bypass the promised transition boundary. Native browser verification also failed at Chrome startup.

```findings
findings:
  - id: new
    severity: Critical
    family: viewport-reachability
    title: |
      Visible sketches can lose their note control after scrolling.
    detail: |
      cmd/internal/webcast/web/sketch-notes.js:12–19 anchors each sketch control exclusively to its first point; lines 52–54 position it without viewport clamping, while sketch.css:83 clips overflow. A production-function reproduction with a stroke spanning page y=100–700 and placement shifted upward 500px leaves visible ink at viewport y=0–200 but positions View note at top:-392px. This breaks the promised visible control for on-view sketches (ARCH-PURPOSE, ARCH-CONSTRAINTS). Derive the control from visible painted bounds, keep its complete hit area reachable, and cover scrolling, multiple paths, and viewport edges.
  - id: new
    severity: Important
    family: viewport-reachability
    title: |
      Updating the minimized count does not reclamp the growing button.
    detail: |
      cmd/internal/webcast/web/floating-panel.js:150–153 changes the Notes (N) text without measuring its new size or sending a geometry transition. After moving the minimized button to the right boundary, increasing the count can extend it past the viewport. A variable-width DOM reproduction changed its right edge from 788px to 812px in an 800px viewport. Remeasure and reclamp after size-changing updates, preserving mode and keeping drag movement constant-cost; add a count-growth regression (ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    family: authoritative-transition-boundary
    title: |
      Mode changes mutate authoritative state outside the transition function.
    detail: |
      cmd/internal/webcast/web/floating-panel.js:104 directly assigns state.mode before calling floatingPanelTransition. This bypasses the plan's model-owned state boundary and ARCH-ORDER's structural requirement. Route the mode change through the reducer, render that state, then submit measured geometry as a separate event; test the production adapter's transition sequence.
  - id: new
    severity: Important
    kind: unverifiable
    family: boundary-verification-evidence
    title: |
      Required native browser and operator acceptance remain unverified.
    detail: |
      Running WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript|TestServer' -count=1 -timeout 3m -v failed both native tests before opening a page: Chrome reported GPU process exit_code=6 and “GPU process isn't usable.” The issue and plan also leave the staged operator look outstanding. Provide non-skipped passing browser results for the corrected head on a working host and recorded completion of the specified operator look.
```

1. **Strengths**

   - Pure geometry and entry projections are directly testable without IO.
   - Keyed sketch handlers reread current entries; final labels use `textContent` and issue no mutation commands.
   - Both viewer assets and isolated operator assembly include the new modules.
   - README and atlas explain the new controls and their authority boundaries.

2. **Critical findings**

   - Offscreen sketch-note controls: see the first finding. Use visible painted bounds and verify complete button reachability.

3. **Important findings**

   - Recompute minimized geometry when count text changes.
   - Keep authoritative mode changes inside the reducer.
   - Supply native browser and operator acceptance evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All **547 Node tests passed**.
   - Targeted server checks and all four operator module/bundle checks passed.
   - `git diff --check` passed; checkout remains unchanged.
   - Existing tests miss scroll-dependent discovery, count-dependent geometry, and the adapter’s reducer bypass. Native tests could not execute their scenarios.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses existing placement and label owners.
   - **ARCH-PURE — pass:** pure projections remain independent of browser IO.
   - **ARCH-PURPOSE — flag:** visible sketches can lack reachable note discovery.
   - **ARCH-MOCK — pass structurally:** stateful DOM fixtures share production adapters; native conformance remains unverified.
   - **ARCH-CONSTRAINTS — flag:** viewport bounds are not maintained across all geometry changes.
   - **ARCH-SECURE — pass:** labels render as text; authenticated command ownership remains intact.
   - **ARCH-ORDER — flag:** direct mode assignment bypasses the transition boundary.
   - **ARCH-FUNERAL — pass:** new controllers expose cleanup; no persisted artifact family is introduced.
   - **ARCH-EVOLVE — pass:** no persisted format changes.

7. **Plan revision recommendations**

   Add `## Revisions` entries replacing the first-point positioning assumption with viewport-aware discovery from accepted painted bounds, extending geometry verification to count changes, and documenting reducer-only mode transitions. Record native and operator acceptance only after completion.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2bce9d48-e357-4601-a878-7130b97597a1","repository":"github.com/xianxu/pair","issue_id":"000850","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c486f21bfc290e0a","reviewed_head":"1c2a16c82e6670aea3739eb5da1c18a2fa8ae73a","slot":"pair:4","requested_at":"2026-10-24T02:56:09.343153Z"},"phase":"finished","pid":88649,"started_at":"2026-10-24T02:56:09.494632Z","finished_at":"2026-10-24T02:59:07.298822Z","session_id":"01a16bd6-509f-7691-b2c5-4784dbb562bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.0094825s
head: "1c2a16c82e6670aea3739eb5da1c18a2fa8ae73a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-23T20:48:34-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 850 — Cast notes: sticky-note style, minimize to a draggable button, draggable panel, visible add-note on sketches |
| repo | pair |
| issue file | workshop/issues/000850-cast-notes-sticky-note-style-minimize-to-a-draggable-button-draggable-panel.md |
| boundary | whole-issue close |
| milestone | — |
| window | 221a833b2dc8504a9498d2f5b4b944b99ea7907d..db56f5f94d9966026d5e8a51da8e33718a01095b |
| command | sdlc close --issue 850 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T20:48:34-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The panel behavior, documentation, and BR-1–3 corrections are supported by passing tests and regression checks. Retained source-matched browser evidence and recorded operator acceptance answer BR-4. One correctness gap remains: overlapping sketch controls can conceal one another, defeating discovery of individual saved notes.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Painted bounds now flow through the existing cache into viewport-clamped controls. Scroll, multipath, edge, and cache regressions pass; substituting the pre-fix module makes the bounds regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      Size-changing minimized counts retire the gesture, render, measure, and reclamp. The count-growth regression passes and fails with the old setCount implementation.
  - id: BR-3
    disposition: addressed
    note: |
      Mode changes now follow reducer → render → measured geometry. The production-adapter sequence regression passes and fails when the direct mode assignment is restored.
  - id: BR-4
    disposition: addressed
    note: |
      The retained native log records both tests passing non-skipped in 11.242s; its SHA256 and every listed source blob match the pinned head. The plan records operator acceptance under TL receipt a7dcd3bb. This host still fails before page creation with GPU exit_code=6, but the supplied source-bound host evidence answers the prior verification gap.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: viewport-reachability
    title: |
      Overlapping sketch controls conceal individual notes.
    detail: |
      cmd/internal/webcast/web/sketch-notes.js:38–63 positions each button independently without accounting for other controls. A production-function reproduction with two completed sketches sharing bounds {left:100,top:100,right:200,bottom:150} in a 320×480 viewport places both 80×44 View note buttons at left:100px, top:48px. The later sibling covers the earlier button, preventing pointer access to its note. This is the 3rd finding in family viewport-reachability. Earlier rounds fixed instances; state and enforce the rule that every eligible sketch remains individually selectable after all controls are composed. Sweep identical bounds, partially overlapping targets, and edge-clamped collisions using collision-aware placement or an explicit grouped selector, while retaining bounded update costs. Add multi-sketch regression and native hit-target coverage. ARCH-PURPOSE, ARCH-CONSTRAINTS.
```

1. **Strengths**

   - Bounds reuse the existing painter/cache, avoiding another point-history projection.
   - Panel movement remains constant-cost; count changes preserve minimized mode.
   - Final labels render through `textContent` and remain read-only.
   - README and atlas updates cover the new controls, keyboard behavior, and ownership boundaries.

2. **Critical findings**

   - **Overlapping controls:** `cmd/internal/webcast/web/sketch-notes.js:38–63`. Make every sketch individually selectable when control positions collide, including after viewport clamping.

3. **Important findings**

   None newly raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All **550 Node tests passed**.
   - Focused Go assembly, server, and panel checks passed.
   - In-memory pre-fix substitutions made the BR-1–3 regressions fail; checkout files remained unchanged.
   - Native tests locally failed before opening a page because Chrome’s GPU process exited. Retained passing evidence matches the pinned source.
   - Current edge tests exercise one sketch identity at a time; they miss collisions between different sketches.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing note commands and paint cache are reused.
   - **ARCH-PURE — pass:** geometry/projection logic is separated from DOM effects.
   - **ARCH-PURPOSE — flag:** colliding controls prevent individual note discovery.
   - **ARCH-MOCK — pass:** stateful DOM fixtures cover capture and viewport behavior, supplemented by retained native evidence.
   - **ARCH-CONSTRAINTS — flag:** independent viewport clamping does not ensure reachability after controls are composed.
   - **ARCH-SECURE — pass:** labels use text rendering; no new credential or persistence boundary.
   - **ARCH-ORDER — pass:** authoritative panel transitions now pass through the reducer.
   - **ARCH-FUNERAL — pass:** controls and capture resources have disposal paths; no new durable runtime artifacts.
   - **ARCH-EVOLVE — pass:** persisted formats are unchanged.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining **individual sketch selectability after composition**. Enumerate identical bounds, partial overlap, and collisions caused by edge clamping; specify the placement/grouping policy, its cost bound, and multi-sketch hit-target verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"be30478e-2b1e-4f72-a780-b45a47c14320","repository":"github.com/xianxu/pair","issue_id":"000850","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a106d736ce961fa5","reviewed_head":"db56f5f94d9966026d5e8a51da8e33718a01095b","slot":"pair:4","requested_at":"2026-10-24T03:45:56.403337Z"},"phase":"finished","pid":59328,"started_at":"2026-10-24T03:45:56.528735Z","finished_at":"2026-10-24T03:48:33.671369Z","session_id":"01a16c03-e495-7490-8511-ee652a45fc49","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.835931167s
head: "db56f5f94d9966026d5e8a51da8e33718a01095b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T20:55:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 850 — Cast notes: sticky-note style, minimize to a draggable button, draggable panel, visible add-note on sketches |
| repo | pair |
| issue file | workshop/issues/000850-cast-notes-sticky-note-style-minimize-to-a-draggable-button-draggable-panel.md |
| boundary | whole-issue close |
| milestone | — |
| window | 221a833b2dc8504a9498d2f5b4b944b99ea7907d..33b974d890c0e0a54cbf7bdc327c7a62b72ecc1c |
| command | sdlc close --issue 850 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T20:55:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range delivers the sticky notes panel and revised sketch-note selector. BR-5 is addressed at the underlying rule: overlapping sketches now share one entry button and remain individually selectable through scrolling rows. No new defects found. Confidence is medium because native Chrome execution failed during GPU startup on this host; retained native evidence matches the pinned source.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      sketch-notes.js replaces geometric per-sketch buttons with one clamped selector and keyed scrolling rows. Current tests pass; substituting the pre-fix module from 1b7764629 in memory makes four regressions fail, including overlap selection and the one-measurement cost assertion. Native coverage independently hit-tests and clicks all 24 overlapping rows; the retained passing log checksum and all 11 tested source blobs match the pinned head.
```

1. **Strengths**

   - The selector fixes the collision class without spatial packing: each eligible sketch retains a distinct row and latest-identity handler (`cmd/internal/webcast/web/sketch-notes.js:45–68`).
   - Panel mode changes pass through the pure reducer before rendering and measurement (`cmd/internal/webcast/web/floating-panel.js:101–105`).
   - Sketch discovery reuses cached painter bounds, avoiding another point-history projection (`cmd/internal/webcast/web/sketch.js:810`).
   - README and atlas updates describe the actual selector, keyboard controls, and unchanged submission authority.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - All **552 webcast Node tests passed**.
   - Focused Go server, module-assembly, and panel checks passed.
   - Regression sensitivity confirmed: the pre-fix sketch-controls module produces **four failures** against current tests.
   - Native panel/operator tests were attempted but Chrome exited before page execution with “GPU process isn't usable.” The retained non-skipped native run reports success; its SHA256 and all 11 source-blob references were independently verified against the pinned head. BR-4 remains disposed.
   - The full pre-close suite was not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Existing painter/cache, keyed rows, and label actions are reused.
   - **ARCH-PURE — pass:** Transition and entry derivation functions are directly tested without IO; browser effects remain in adapters.
   - **ARCH-PURPOSE — pass:** The revised selector addresses individual selection across overlapping drawings, with bounded scrolling.
   - **ARCH-MOCK — pass:** Stateful browser fixtures exercise capture and event ordering; retained native tests check actual layout and hit targets.
   - **ARCH-CONSTRAINTS — pass:** Tests pin constant drag costs, linear grouping/row updates, and one summary measurement.
   - **ARCH-SECURE — pass:** Labels use `textContent`; Add note rechecks policy ownership; final inspection sends no mutation.
   - **ARCH-ORDER — pass:** Panel transitions govern gesture identity, cancellation, mode changes, and disposal; stale sketch handlers reread current membership.
   - **ARCH-FUNERAL — pass:** New UI resources have disposal paths; no growing runtime artifact family is introduced.
   - **ARCH-EVOLVE — pass:** No persisted format changes.

   The Core concepts entries match their implementations and classifications. No contradictory entity or missing documentation surface was found.

7. **Plan revision recommendations:** None. The BR-5 revision explicitly supersedes individual geometric controls. Remaining pre-close and landing steps are recorded as orchestration work.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"40aeb2a5-4594-4125-892b-df6588c0418d","repository":"github.com/xianxu/pair","issue_id":"000850","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/57f823e6fcb773e5","reviewed_head":"33b974d890c0e0a54cbf7bdc327c7a62b72ecc1c","slot":"pair:4","requested_at":"2026-10-24T03:53:58.80258Z"},"phase":"finished","pid":43128,"started_at":"2026-10-24T03:53:58.97476Z","finished_at":"2026-10-24T03:55:33.01131Z","session_id":"01a16c0b-4205-75a0-88f9-d79d7cf7bfb2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.094307458s
head: "33b974d890c0e0a54cbf7bdc327c7a62b72ecc1c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
