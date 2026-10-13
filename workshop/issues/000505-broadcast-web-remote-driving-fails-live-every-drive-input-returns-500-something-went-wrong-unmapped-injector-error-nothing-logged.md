---
id: 000505
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '5ccb9c8738b1bcb9ef03cdffb654a7e1af9a1ea5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:30:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "4589c717", done: "daa15728"}
---

# broadcast-web: remote driving fails live — every drive input returns 500 'Something went wrong' (unmapped injector error, nothing logged)

## Problem

Operator's live test of remote driving (Couch 620356e7, has #467; a cast of the blog at localhost:4321, drive link,
control granted): every scroll or link click showed "Something went wrong.", drive.Answer's 500 for an unmapped
error, and Couch logged nothing. The drive tests had all run headless, never scrolled, and used a fake injector or
called the controller directly.

## Spec

Reproduce headless through the whole production chain (real cast Chrome + recorder, `TabInjector` as StartCast
wires it, the viewer routes, a viewer's real drive.js); fix every root cause found; map or log every injector error
so a drive failure is never a silent 500; check #466's whole drive contract against a real cast (TL acceptance:
hover, left click, double click, drag, wheel, typing, editing keys, select-all/undo/redo, IME; refusals: right/
middle click, accelerators, clipboard, pickers, downloads, new windows). A windowed Chrome run needs the TL's OK.

## Done when

- The headless full-chain test (`TestDriveRealCast`) passes, and failed before the fixes.
- Every injector error is mapped to a sentence or logged for the operator; tests pin both.
- The operator's live drive works (their next cast), or the logged cause names what's left.

## Plan

- [x] Bug 1 — hit test at page coordinates. `DOM.getNodeForLocation` hit-tests page (document) px within the
  viewport (measured below); `cdpTab.At` subtracted the scroll, so after any scroll every click and wheel missed and
  was refused stale. `At` passes the act's page px; "No node found" is stale, any other failure `ErrInput`.
- [x] Bug 2 — double click. Chrome's pointer events carry `detail` 0 (measured), and drive.js read the count from
  `pointerup.detail`, so it never sent one. `gestures` counts it (`DOUBLE_MS` 500, within `DRAG_PX`).
- [x] Errors: `drive.ErrInput` (a CDP input call or hit test failed; wraps the cause) → 502 "The cast couldn't do
  that. Try again."; request-context end → 503 "The cast is busy. Try again."; the recorder closing → `ErrClosed`
  (410); a cast with no recorder → `ErrNotLive`; anything else stays 500 "Something went wrong." Every 5xx is logged
  to the operator (`Server.WithDriveLog` ← the cast's `Logf`, Couch's notice line), once per distinct message
  (in memory, at most 64 kept, then forgotten — ARCH-FUNERAL).
- [x] Tests: TestDriveRealCast (WEBCAST_LIVE_CHROME, optional WEBCAST_DRIVE_DPR=2: wheel, click a link after the
  scroll, hover, double click, drag, typing, select-all + Backspace, undo, IME, right click refused); default-run
  TestCdpTabHitTestsPageCoordinates (fails with the scroll subtracted), TestDriveRoutesLogWhatTheViewerCantFix
  (502/503/500 + one log line each), TestAnswer cases, node drive.test.mjs double-click timing (detail 0).
- [x] atlas/webcast-format.md: answers, the operator log, page-px hit tests, the double-click rule.

## Log

### 2026-10-12
- Existing drive tests pass headless (TestLiveDrive, TestLiveDriveNavigation, TestDriveLiveTwoViewers): none
  scrolls, and the viewer test uses the fake injector. New full-chain TestDriveRealCast reproduced a failure: the
  wheel at the top worked, the click after it was refused stale.
- **Measured (headless, scrollY 236):** `DOM.getNodeForLocation` at page (94,405) returns the aimed link (node 30);
  viewport (94,169) and (10,10), (400,100) return "No node found"; (400,300) and (400,500) find nodes. So it
  hit-tests document coordinates, visible part only. At dpr 2 the same.
- **Measured:** a CDP double click on a plain page logs `pointerdown:0 pointerup:0 mouseup:1 click:1 pointerdown:0
  pointerup:0 mouseup:2 click:2 dblclick:2`.
- The operator's 500 itself isn't reproduced headless: bug 1 answers 409 "The page changed", not 500. The live
  500 needs an unmapped error the windowed Chrome produces; with this change it is either mapped (CDP input
  failure → 502, busy → 503) or logged with its cause on Couch's notice line. Asked the TL for one windowed run.
- **Operator data point (TL, #501 build, before these fixes):** everything worked except scrolling. That is bug 1:
  the first wheel from the top hit-tests correctly (scroll 0), every wheel after the page has scrolled misses.
  TestDriveRealCast now wheels from the top AND again after the page has scrolled (asserting it scrolls further);
  with the scroll-subtracting hit test restored, the second wheel times out, matching the operator. The operator
  re-checks scrolling first when this build is staged.
