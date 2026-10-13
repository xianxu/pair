# Boundary Review — pair#524 (whole-issue close)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | addd45e19d528471163e0d6803e8cb65718b89a7..bff299dae247d10846bc6bba22846d09f69fd7d0 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:35:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The viewport-preserving capture path, metadata propagation, and compatibility handling are well covered. One correctness gap blocks shipment: animations affecting overflowing descendants can be skipped and falsely reported as settled. The affected Go packages pass, but independent browser verification was blocked by Chrome crashing during startup.

1. **Strengths**

   - Screenshot capture disables `captureBeyondViewport`; the browser regression checks pixels and animation identity.
   - Settlement and clipping metadata propagate through recorder, Cast, validation, and storage. Older manifests retain unknown settlement.
   - Polling has explicit deadlines, cancellation handling, and viewport/document checks. README and atlas describe the changed behavior.

2. **Critical findings**

   - [capture_ready.js:58](/Users/xianxu/Library/Caches/sdlc/review-checkouts/6fb8d5e15fb8be81/cmd/internal/webcast/recorder/capture_ready.js:58) — **ARCH-PURPOSE:** The probe excludes an animation solely because its target’s bounding box misses the crop. A zero-size parent with `overflow: visible` can still fade a visible child inside the crop. The probe returns `pending:false`, causing Go to publish `settled` while that fade remains active. Account for affected descendants and overflowing pseudo-elements within the inspection budget; report uncertainty when influence cannot be determined.

3. **Important findings:** None additional.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: submission, recorder, submission builder, and webcast packages.
   - An independent Node fixture exercising the production probe fails for all three overflowing-parent cases: finite, paused, and infinite animations.
   - Existing fixtures cover enclosing ancestors, not overflowing descendants.
   - Both live capture tests failed before opening a page because Chrome’s GPU process crashed; disabling GPU rendering did not resolve this. Browser pixel and mutation checks remain unverified in this review.
   - Artifact inventory tests also failed because generated runtime assets are absent and related exception entries appear stale. These failures are not established as regressions from this diff.
   - Checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared settlement validation and crop intersection.
   - **ARCH-PURE — pass:** Geometry/schema logic remains separate from browser observation.
   - **ARCH-PURPOSE — flag:** Animation influence detection misses overflowing descendants.
   - **ARCH-MOCK — pass structurally:** Existing injected CDP seam, controlled fixtures, and live conformance tests; live execution was blocked.
   - **ARCH-CONSTRAINTS — pass:** Explicit polling, traversal, animation, pixel, and overall limits.
   - **ARCH-SECURE — pass:** Geometry and metadata validation precede evidence acceptance; tests use temporary profiles.
   - **ARCH-ORDER — pass:** Sequential capture preserves document barriers and rejects viewport drift.
   - **ARCH-FUNERAL — pass:** No new durable artifact family; metadata uses the existing bounded store.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating animation influence cases—direct targets, enclosing ancestors, overflowing descendants, and pseudo-elements—and the bounded handling and regression coverage for each.

```findings
findings:
  - id: new
    severity: Critical
    family: animation-influence-coverage
    title: |
      Target-only geometry can falsely label an animated crop settled
    detail: |
      cmd/internal/webcast/recorder/capture_ready.js:58 skips animations whose target box misses the crop, even when an overflowing descendant inside the crop inherits the target's animated opacity. A production-probe fixture returns pending:false for finite, paused, and infinite animations on such a parent. ARCH-PURPOSE: account for descendant and pseudo-element influence within the inspection budget, conservatively report uncertainty where necessary, and add failing regression coverage including browser pixels.
```
