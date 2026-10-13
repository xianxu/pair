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

- Responsive fades already finished before capture stay finished with the same animation identity; capture itself emits no induced viewport change. A mutation restoring captureBeyondViewport=true fails the pixel/timeline regression.
- Approved viewport policy is enforced: partial crops carry their actual box and clipped_to_viewport, fully offscreen notes keep context with outside-viewport unavailable, and no page scrolling/style forcing occurs.
- Finite, delayed, paused/infinite and open-shadow animations produce validated settled/unsettled evidence under 1500 ms/note, 50 ms polls, 512 animations/20k traversal and 30 s total; old bundles remain readable as unknown.

## Plan

- [x] Measure the original bundle/recording and reproduce in a separate bounded headless Chrome.
- [x] Obtain TL approval of the measured design and viewport-only crop policy below.
- [x] Add deterministic animation/capture regressions; implement bounded settling and unchanged-viewport screenshot capture.
- [x] Verify metadata propagation, navigation/cancellation, real-browser pixels, full suite and docs.

Detailed proposal: [animation capture plan](../plans/000524-animation-capture-plan.md).

## Log

### 2026-10-12 — capture-time restart measured

Original evidence: the first article already had data-animated=true and no no-intersect at recording start. Its fadeInUp animation is 1 s, opacity 0→1 and translateY(2rem)→0. Periodic recorded frames immediately before and after the dark crop show settled content. The original viewport was 864×997 at DPR 2; crop right edge 870.5 exceeded it by 6.5 CSS px. No live operator browser was touched.

Bounded isolated headless probes (temporary profile, 30 s test ceiling) used the production CaptureCrops path and localhost:4321 blog. A settled card had opacity=1, animation state=finished, currentTime=1000 before capture. It returned to opacity≈0.033 and a new animation startTime afterward. The same failure occurred with a crop entirely inside the viewport, so overflow alone is not the cause.

A direct Page.captureScreenshot A/B changed only captureBeyondViewport, with fromSurface=true and the exact original clip. true→false→true→false yielded bright-pixel counts 0→19891→0→19891. True emitted resize/media-query notifications and restarted the animation; false emitted neither and preserved the finished timeline. Thus pre-capture waiting alone cannot fix this bug: capture itself restarts the responsive animation. Background throttling and an unmet intersection trigger are not required to reproduce it. This identifies the causative CDP option, not every Chromium internal step.

Probe logs: /tmp/pair524-probe/blog.log and flags.log; measurement-only Go overlays live beside them. Synthetic finite-delay probe also showed that still-running fades legitimately produce black pixels before finishing, so bounded settling remains useful after removing the capture-induced restart. No product code changed.

### 2026-10-12 — first implementation verification

Implemented unchanged-viewport screenshots, synchronous bounded animation observations with Go-owned polling, clipped/offscreen evidence, and strict settlement schema/propagation. Targeted schema/recorder/builder/webcast packages pass; existing headless crop coordinate/conformance test passes (1.70 s). New capture option test failed on true before the fix. Fractional CSS box regression exposed spurious clipping from subtraction rounding; unchanged axes now retain their original dimensions. New responsive animation browser fixture and mutation checks are next. #494 notified that its pending branch wraps the screenshot with CaptureScreenshot; preserve that helper if it lands before #524.

### 2026-10-12 — real-browser regression and mutation

The new responsive fade fixture confirms the fix preserves the completed CSSAnimation identity/startTime and bright pixels. Mutation overlay restoring captureBeyondViewport=true failed because animation identity changed to a new running fade. Conformance also corrected two test/model assumptions: Chrome reports one 800 ms finite endTime as 799.9999999999999 (assert finished/progress instead of a rounded constant), and document.getAnimations excludes open-shadow animations. Implement explicit bounded open-shadow-root enumeration; the approved scope already includes these roots.

### 2026-10-12 — final bounded browser evidence

TestLiveSubmissionCaptureReady passes in 2.91 s: finished responsive fade identity/pixels, naturally completed finite and open-shadow animations, irrelevant infinite animation, in-box infinite metadata, viewport clip and fully offscreen unavailable. Existing TestLiveSubmissionCrop passed in 1.70 s. The isolated localhost blog production-path check passes in 3.52 s: actual box [0.4,282.2,863.6,138.7], clipped=true, settled, 19891 bright pixels, opacity stays 1 and animation startTime stays unchanged. Measurement logs remain /tmp/pair524-probe/blog-fixed.log and flags.log. Mutation reverting the screenshot option fails the live test on restarted animation identity; mutation dropping settlement propagation fails Cast tests. Latest main (#506) merged cleanly; serial full suite now running with GOMAXPROCS=2 and all nested Go package parallelism limited to 1.

### 2026-10-12 — full-suite close evidence

Serial full make test completed (GOMAXPROCS=2, nested Go package parallelism 1). All shell/editor checks and Go packages passed except TestNotificationBrokerBeforeExecAndCleanup, the pre-existing startup-hook failure tracked as pair#519 and previously reproduced on main during #483. Full output: /tmp/pair524-probe/full-suite.log. Affected packages all passed, including webcast (8.052 s), recorder (4.134 s), submission schema (1.198 s), and submission store (0.176 s). No new suite failure. Browser and mutation evidence above verifies the changed behavior; operator smoke remains a TL-scheduled shipment step.

## Revisions

- 2026-10-12 — Measured cause revises the initial hypothesis: the screenshot path restarts an already-finished responsive fade. Proposed design avoids viewport-changing capture, then waits boundedly for relevant finite animations. Proposed viewport intersection and explicit offscreen-unavailable policy require TL approval before implementation; see the durable plan. ARCH-PURPOSE: fix the capture-induced restart rather than adding a sleep before it.

- 2026-10-12 22:56 PDT — TL approved the full proposal (receipt 5992b8ba-8060-42e0-949a-d52fc973ae69): captureBeyondViewport=false, 1500 ms/note with 50 ms polling and 512-animation cap inside 30 s, explicit settled/unsettled, viewport-intersected crops/clipped_to_viewport, outside-viewport unavailable with recorded context retained. No forced styles or scrolling. change-code admitted implementation.


- 2026-10-12 — Done-when refined to spell out the approved viewport policy, measured responsive-restart regression and metadata/scope bounds. Original goal remains; no new feature scope.

## Shipment handoff

Close through SDLC after verified automated evidence; stage only on TL instruction, then land after acceptance and report to the hub. #494 screenshot-suppression integration remains the second lander's responsibility.


## Revisions (continued)

### 2026-10-12 — BR-1 animation influence correction

ARCH-PURPOSE: target geometry alone does not bound animated paint. Direct targets and enclosing ancestors remain covered; zero-size/off-box targets with light children, shadow content, or generated pseudo-elements can affect overflowing pixels. Conservatively treat those active animations as potentially relevant within the existing 512-animation and 1500 ms limits. Finite animations wait naturally; paused/infinite ones remain explicitly unsettled. No per-animation descendant walk or mutation is needed. Plain off-box leaf targets without generated content remain irrelevant. Regression fixtures cover finite/paused/infinite cases and browser pixel evidence. This corrects influence detection within the approved capture contract, with no new scope or limits.

BR-1 verification: production-probe Node regression first failed on finite overflowing-parent influence, then passes finite/paused/infinite child, shadow, generated pseudo, pseudo-object and pseudo-effect cases. Recorder package passes (4.146 s). Bounded live browser regression with the old probe overlay failed: settled at 49.96 ms of a 900 ms parent fade, still running, 0/28800 bright pixels. Fixed probe passes in 4.88 s, with naturally finished timelines and bright pixels for both overflowing child and ::before. Serial full suite completed again after this correction: all shell/editor checks and Go packages passed except the same pre-existing pair#519 startup-hook failure. Log: /tmp/pair524-probe/full-suite-br1.log. No new failures.


### 2026-10-13 — BR-1 second review and approval hold

Second close returned REWORK: overflowing children/pseudo correction is exercised, but an empty off-box leaf can still paint an opacity-animated box-shadow into the crop. Added production-probe regression and confirmed RED (leaf shadow running falsely settles); this test is intentionally failing pending the policy decision. Current production remains at the first correction. Requested hub approval (receipt 4ed19fc7-774d-4bb6-b118-402976e124d7) for simpler conservative observation of all active animations in the supported top-document/open-shadow scope, within unchanged budgets. Tradeoff: unrelated off-box animations may consume the wait or produce unsettled evidence. No partial CSS paint-footprint analyzer proposed. Do not implement this policy change until hub replies; no staging or merge. Prior full-suite evidence remains accurate for the committed first correction, not a claim that the new RED fixture passes.
