---
id: 000524
status: working
deps: [pair#483]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd3a52c351fa4be9cc3dd441a79d0e1cfc33b397f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T22:49:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a10a038e", done: "7d2bc21a"}
---

# broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture

## Problem

pair#483's smoke test (2026-10-12, the operator's blog): one of three crops (`crop-0.png`, the 'Couch: A Sneak Peek' post card) came out nearly black, text barely visible, while its box text was exact. The blog's cards use AstroWind's scroll-in effect (`intersect-once intersect-quarter motion-safe:md:opacity-0 motion-safe:md:intersect:animate-fade` in src/components/blog/ListItem.astro / GridItem.astro): they start at opacity 0 and fade in once in view. The capture caught the card at or near opacity 0. Bundle: ~/.local/share/pair/couch/webcast/submissions/9dc7b091…/crop-0.png.

## Spec

Find why the card was still faded at capture (measure first: does the CDP capture path re-render or restart the animation; is the cast Chrome throttling animations when not frontmost; is the intersection never reached). Then capture settled pixels: before each crop, wait (bounded) for animations/transitions intersecting the box to finish (`document.getAnimations()`), or force their finished state for the capture, and restore after. Never steal focus. Record in the crop's metadata when a capture couldn't settle.

## Done when

- The operator's blog card crop shows its text at full opacity; a test page with a fade-in on intersection reproduces the dark crop before the fix and passes after; a capture that can't settle is marked.

## Plan

- [x] Measure the original bundle/recording and reproduce in a separate bounded headless Chrome.
- [x] Obtain TL approval of the measured design and viewport-only crop policy below.
- [ ] Add deterministic animation/capture regressions; implement bounded settling and unchanged-viewport screenshot capture.
- [ ] Verify metadata propagation, navigation/cancellation, real-browser pixels, full suite and docs; close, then stage only on TL instruction.

Detailed proposal: [animation capture plan](../plans/000524-animation-capture-plan.md).

## Log

### 2026-10-12

### 2026-10-12 — capture-time restart measured

Original evidence: the first article already had data-animated=true and no no-intersect at recording start. Its fadeInUp animation is 1 s, opacity 0→1 and translateY(2rem)→0. Periodic recorded frames immediately before and after the dark crop show settled content. The original viewport was 864×997 at DPR 2; crop right edge 870.5 exceeded it by 6.5 CSS px. No live operator browser was touched.

Bounded isolated headless probes (temporary profile, 30 s test ceiling) used the production CaptureCrops path and localhost:4321 blog. A settled card had opacity=1, animation state=finished, currentTime=1000 before capture. It returned to opacity≈0.033 and a new animation startTime afterward. The same failure occurred with a crop entirely inside the viewport, so overflow alone is not the cause.

A direct Page.captureScreenshot A/B changed only captureBeyondViewport, with fromSurface=true and the exact original clip. true→false→true→false yielded bright-pixel counts 0→19891→0→19891. True emitted resize/media-query notifications and restarted the animation; false emitted neither and preserved the finished timeline. Thus pre-capture waiting alone cannot fix this bug: capture itself restarts the responsive animation. Background throttling and an unmet intersection trigger are not required to reproduce it. This identifies the causative CDP option, not every Chromium internal step.

Probe logs: /tmp/pair524-probe/blog.log and flags.log; measurement-only Go overlays live beside them. Synthetic finite-delay probe also showed that still-running fades legitimately produce black pixels before finishing, so bounded settling remains useful after removing the capture-induced restart. No product code changed.

## Revisions

- 2026-10-12 — Measured cause revises the initial hypothesis: the screenshot path restarts an already-finished responsive fade. Proposed design avoids viewport-changing capture, then waits boundedly for relevant finite animations. Proposed viewport intersection and explicit offscreen-unavailable policy require TL approval before implementation; see the durable plan. ARCH-PURPOSE: fix the capture-induced restart rather than adding a sleep before it.

- 2026-10-12 22:56 PDT — TL approved the full proposal (receipt 5992b8ba-8060-42e0-949a-d52fc973ae69): captureBeyondViewport=false, 1500 ms/note with 50 ms polling and 512-animation cap inside 30 s, explicit settled/unsettled, viewport-intersected crops/clipped_to_viewport, outside-viewport unavailable with recorded context retained. No forced styles or scrolling. change-code admitted implementation.

### 2026-10-12 — first implementation verification

Implemented unchanged-viewport screenshots, synchronous bounded animation observations with Go-owned polling, clipped/offscreen evidence, and strict settlement schema/propagation. Targeted schema/recorder/builder/webcast packages pass; existing headless crop coordinate/conformance test passes (1.70 s). New capture option test failed on true before the fix. Fractional CSS box regression exposed spurious clipping from subtraction rounding; unchanged axes now retain their original dimensions. New responsive animation browser fixture and mutation checks are next. #494 notified that its pending branch wraps the screenshot with CaptureScreenshot; preserve that helper if it lands before #524.

### 2026-10-12 — real-browser regression and mutation

The new responsive fade fixture confirms the fix preserves the completed CSSAnimation identity/startTime and bright pixels. Mutation overlay restoring captureBeyondViewport=true failed because animation identity changed to a new running fade. Conformance also corrected two test/model assumptions: Chrome reports one 800 ms finite endTime as 799.9999999999999 (assert finished/progress instead of a rounded constant), and document.getAnimations excludes open-shadow animations. Implement explicit bounded open-shadow-root enumeration; the approved scope already includes these roots.
