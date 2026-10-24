# Boundary Review — pair#856 (whole-issue close)

| field | value |
|-------|-------|
| issue | 856 — Cast tool: one Notes panel for notes and sketches, and one toolbar column with tools revealed on toggle |
| repo | pair |
| issue file | workshop/issues/000856-cast-tool-one-notes-panel-for-notes-and-sketches-and-one-toolbar-column-with.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..65c775663adf450c6f832785fc6763417fa3636f |
| command | sdlc close --issue 856 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T00:56:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The unified panel and toolbar reuse existing command owners, and all 592 Node tests plus four operator bundle checks pass. Two reproduced behavior defects block shipping: offscreen sketches disappear from the sole Notes surface, and annotation glow uses incorrect dimensions in scaled viewers. Minimized status signaling is also missing. Native browser qualification could not complete because Chrome failed during startup.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: feedback-membership-independent-of-placement
    title: |
      The unified Notes panel loses sketches when their ink leaves the viewport
    detail: |
      cmd/internal/webcast/web/sketch.js:836 feeds the sole Notes surface through sketchNoteEntries, whose web/sketch-notes.js:6-8 rejects missing or offscreen painted bounds. Reproduction with the same completed sketch returns one row for visible bounds and zero rows for offscreen or null bounds. With the separate preview cards removed, its label and Show on page action become inaccessible, contrary to the issue's one-entry-per-sketch contract (ARCH-PURPOSE). Derive row membership from accepted sketch identity/metadata independently of placement; use geometry only for availability, glow and reveal, and add scroll-away/missing-anchor regressions.
  - id: new
    severity: Critical
    kind: defect
    family: page-to-client-rectangle-projection
    title: |
      Annotation hover glow scales its origin but leaves its dimensions unscaled
    detail: |
      cmd/internal/webcast/web/annotate.js:657-661 transforms the box and clip origins through v.fromPage but passes their page-space widths/heights directly to canvas. A production mount with viewer scale 0.5 paints box [10,20,80,40] as [5,10,80,40], rather than [5,10,40,20]; its clip is similarly twice the intended size. This violates the confirmed-geometry hover contract (ARCH-PURPOSE). Transform both corners of both rectangles, as the existing place helper does, and test non-unit scale plus clipping through the real hover handler.
  - id: new
    severity: Important
    kind: defect
    family: minimized-status-observability
    title: |
      The minimized Notes control gives no indication of pending status
    detail: |
      cmd/internal/webcast/web/feedback-panel.js:128-133 writes status only inside the header; web/floating-panel.js:111-112 hides that header when minimized and always labels the remaining control Notes (N). Reproduction after publishing a stale-preview warning leaves the visible control unchanged. The plan explicitly requires the minimized Notes control to convey status (workshop/plans/000856-cast-feedback-workspace-plan.md:62; ARCH-PURPOSE). Add a visible/accessibly named status indication to the minimized control and cover status arrival, clearing and restoration.
  - id: new
    severity: Important
    kind: unverifiable
    family: native-ui-qualification
    title: |
      Native browser interaction and layout qualification could not run
    detail: |
      Attempted WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorScript.*)$' -count=1 -timeout 120s -v. Both native tests failed before opening a page: Chrome reported GPU process exit_code=6 and FATAL GPU process isn't usable. Thus native hit testing, drawing/editor integration and toolbar separation remain unverified here (ARCH-MOCK, ARCH-CONSTRAINTS). Provide non-skipped passing output tied to the reviewed source from a working Chrome host, including viewport geometry assertions, or restore browser startup in this environment and rerun.
```

1. **Strengths**

   - `feedback-panel.js` preserves typed row identities and inserts labels through `textContent`.
   - `cast-tools.js` adopts existing controls and listeners, preserving command ownership.
   - `annotation-placement.js:159` checks confirmed originals before scrolling and rechecks placement afterward.
   - README and both relevant atlas documents were updated in the range.

2. **Critical findings**

   The first two findings above require separating sketch membership from visibility and correctly projecting annotation glow rectangles.

3. **Important findings**

   Restore minimized status signaling and obtain native qualification evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   All 592 Node tests passed; four operator assembly/execution tests passed; diff whitespace checks passed. Read-only probes reproduced all three behavior findings. Existing sketch tests explicitly retain viewport-filtered membership, while hover tests omit scaled annotation geometry. Add regressions exercising those production paths. There are no prior findings to dispose of.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** existing feedback commands and control owners are reused.
   - **ARCH-PURE — pass:** entry/layout calculations and panel transitions remain separate from DOM effects.
   - **ARCH-PURPOSE — flag:** sole-panel sketch discovery, scaled glow and minimized status are incomplete.
   - **ARCH-MOCK — flag:** fixture coverage passes, but native conformance could not execute.
   - **ARCH-CONSTRAINTS — flag:** arithmetic/linear-work checks pass; native interaction and geometry remain unverified.
   - **ARCH-SECURE — pass:** labels use text insertion; reveal requires original-anchor evidence.
   - **ARCH-ORDER — pass:** floating-panel transitions retain explicit cancellation and resize behavior.
   - **ARCH-FUNERAL — pass:** new transient owners expose teardown; no new durable runtime artifact family.
   - **ARCH-EVOLVE — pass:** this #856 range introduces no persisted-format change; #857 remains explicitly separate.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording that Notes membership includes offscreen/unavailable sketches and that annotation glow projects both rectangle corners. Correct the atlas’s “eligible on-view sketches” description after implementing complete membership. Retain the existing minimized-status requirement and add executable coverage for it.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3ead4b48-d9b9-4c34-8887-1a056169ab49","repository":"github.com/xianxu/pair","issue_id":"000856","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/992c80e3ee587d09","reviewed_head":"65c775663adf450c6f832785fc6763417fa3636f","slot":"pair:4","requested_at":"2026-10-24T07:52:31.010654Z"},"phase":"finished","pid":49790,"started_at":"2026-10-24T07:52:31.143759Z","finished_at":"2026-10-24T07:56:17.217766Z","session_id":"01a16ce5-a3fa-7431-8739-21dc8ba4ba6b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.436683458s
head: "65c775663adf450c6f832785fc6763417fa3636f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T01:08:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 856 — Cast tool: one Notes panel for notes and sketches, and one toolbar column with tools revealed on toggle |
| repo | pair |
| issue file | workshop/issues/000856-cast-tool-one-notes-panel-for-notes-and-sketches-and-one-toolbar-column-with.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..79bac7f1c24df9f034e42b08b01315bff48b3b25 |
| command | sdlc close --issue 856 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T01:08:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The three behavior defects are fixed and backed by regressions that fail when the fixes are removed. All 603 Node tests pass. Retained native evidence matches the reviewed production/test source, but does not establish the outstanding panel-to-toolbar separation requirement; BR-4 remains partially unaddressed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      sketch-notes.js now derives membership independently of bounds. Pure and mounted-panel regressions cover offscreen, missing and restored placement; restoring the previous helper caused six failures.
  - id: BR-2
    disposition: addressed
    note: |
      annotate.js projects both corners of box and clip through clientBox. Mounted hover tests cover non-unit, nonuniform scales; restoring unscaled dimensions caused both geometry regressions to fail.
  - id: BR-3
    disposition: addressed
    note: |
      Minimized Notes visibly indicates Status and exposes the full message through its accessible name and tooltip. Tests cover arrival, clearing, restoration and clamping; removing the visible suffix failed the regression.
  - id: BR-4
    disposition: not-addressed
    note: |
      Retained native-review-fixes.log supplies eight non-skipped passes on 791e8742c, whose cmd/ and tests/ trees match this head. However, cast_notes_panel_live_test.go:32-42 mounts only the standalone panel, without cast-tools CSS, toolbar or protected workspace layout; its 1280/320 bounds assertions cannot establish toolbar separation. operator_sketch_live_test.go:162 checks button reachability, without the required viewport separation matrix. The repeated WEBCAST_LIVE_CHROME=1 GOMAXPROCS=2 go test -p 1 ./cmd/internal/webcast -run '^(TestCastNotesPanelLive|TestOperatorSketchLiveDrawing|TestOperatorLiveAnnotation|TestOperatorFeedbackLivePrepareAndStalePanel|TestOperatorSketchLiveOpenContactHandoffLabel|TestOperatorModuleAssembly|TestOperatorSharedModulesExecute|TestOperatorSketchSharedBundleExecutes)$' -count=1 -v -timeout 180s still failed native startup with GPU exit_code=6. Supply source-bound, non-skipped host results measuring the actual toolbar against expanded/minimized Notes, editors and status at common desktop/narrow widths (ARCH-MOCK, ARCH-CONSTRAINTS).
```

1. **Strengths**
   - Accepted sketch identities survive unavailable geometry while preserving keyed rows and label access.
   - Annotation placement and hover share the corrected rectangle projection.
   - Existing controls and editor owners retain command authority; the new adapters handle presentation.
   - README and both relevant atlas documents describe the new surface.

2. **Critical findings:** None remaining.

3. **Important findings:** BR-4 remains open only for native qualification of toolbar separation. Extend the production workspace fixture with independent rectangle assertions and retain passing host output.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Node: **603 passed**, zero skipped.
   - Disposable mutation checks made BR-1, BR-2 and BR-3 regressions fail.
   - Three Go bundle assembly/execution tests passed.
   - Native tests failed locally before opening pages. Retained host results establish interaction coverage, subject to the qualification gap above.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared projection and existing command owners reused.
   - **ARCH-PURE — pass:** projection, geometry and panel transitions remain directly testable.
   - **ARCH-PURPOSE — pass:** unified panel and column delivered; documented refinements respected.
   - **ARCH-MOCK — flag:** native workspace qualification remains incomplete under BR-4.
   - **ARCH-CONSTRAINTS — flag:** arithmetic containment is tested; actual narrow-browser toolbar separation remains unmeasured.
   - **ARCH-SECURE — pass:** labels use text rendering; reveal requires validated original anchors.
   - **ARCH-ORDER — pass:** panel transitions retain explicit gesture ownership and cancellation.
   - **ARCH-FUNERAL — pass:** new presentation owners dispose their canvases, listeners and keyed caches.
   - **ARCH-EVOLVE — pass:** this boundary changes no persisted format; #857 remains separate.

7. **Plan revision recommendation:** Add a `## Revisions` entry distinguishing completed standalone-panel/native-interaction checks from pending production workspace separation checks. Record completion after measuring the actual toolbar and panel together.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c9ca4987-1d23-4678-be3a-8c7188e27ffc","repository":"github.com/xianxu/pair","issue_id":"000856","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/127a30ea17f8f3f4","reviewed_head":"79bac7f1c24df9f034e42b08b01315bff48b3b25","slot":"pair:4","requested_at":"2026-10-24T08:06:09.907947Z"},"phase":"finished","pid":22078,"started_at":"2026-10-24T08:06:10.089741Z","finished_at":"2026-10-24T08:08:21.198784Z","session_id":"01a16cf2-2369-7b92-8d59-cc92924b8bd2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2m53.258607333s
head: "79bac7f1c24df9f034e42b08b01315bff48b3b25"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000415
