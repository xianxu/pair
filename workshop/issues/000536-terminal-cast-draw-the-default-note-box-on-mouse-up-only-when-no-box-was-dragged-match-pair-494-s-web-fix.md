---
id: 000536
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '7bc3142a731c6f56174efcca314b434174b74e6d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T16:03:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "1cd44f48", done: "12379624"}
---

# terminal cast: draw the default note box on mouse up only when no box was dragged (match pair#494's web fix)

## Problem

The terminal cast viewer draws a default note box on pointer press, then replaces it during a drag. Its cell-based threshold also differs from the web viewer's pixel threshold.

## Spec

Use the shared collaboration module for note gesture tracking in terminal and web viewers (ARCH-DRY). A press shows no box. Movement reaching 8 CSS px on either axis for mouse or 16 for touch latches a drag; only its dragged box is previewed. Returning below the threshold remains a drag through release. A release without crossing the threshold creates the default box. Terminal geometry remains cell-based, including its minimum two-cell pin width and existing bounds/guard checks. Web geometry and operator overlay retain existing behavior.

## Done when

- Terminal click shows no box on press or sub-threshold motion, then drafts the default box on release.
- Mouse/touch drags use 8/16px thresholds, show only drag geometry, and remain drags after returning below threshold.
- Shared gesture tracking is used by both viewers; existing web/operator regressions pass.
- Targeted viewer tests and make -k test complete, with failures investigated and recorded. Close and land; operator glances after landing.

## Plan

- [ ] Add failing terminal click/drag/drag-back tests for both input types, including movement across cells below pixel threshold.
- [ ] Extract the web gesture's monotonic drag tracking into collab.js and use it in both viewers; reuse gestureBox preview suppression and geometry.
- [ ] Run targeted and full verification, close and land.

## Log

### 2026-10-13

- Quick flow requested by TL. Existing terminal pointerdown calls dragBox.show immediately; its boxOf uses cell thresholds. Web already suppresses tap previews and latches dragged once threshold crossed. collab.js is shared through the existing webcast symlink/serving seam.
- ARCH-PURE/ARCH-DRY: shared in-memory gesture owns from/to and monotonic pressed-to-dragged transition, handlers own events and rendering. ARCH-PURPOSE: terminal plus existing web/operator tests cover shared consumers. ARCH-STATE: no drag-to-press reversal; cancellation drops gesture without drafting. ARCH-MOCK: existing viewer fakes exercise handlers. ARCH-CONSTRAINTS/ARCH-FUNERAL: constant-size gesture dies on release/cancel; no timers, external services or durable artifacts added.

- Red: all six new terminal mouse/touch click, drag and drag-back cases failed at press drawing a box (/tmp/pair536-red.log). Green: 244 viewer tests passed, including existing web/operator gestures (/tmp/pair536-viewers.log). Terminal sub-threshold tests cross a cell boundary, proving admission uses pixels rather than cells. Shared noteGesture latches threshold crossing; existing gestureBox owns preview suppression and geometry.
