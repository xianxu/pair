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
