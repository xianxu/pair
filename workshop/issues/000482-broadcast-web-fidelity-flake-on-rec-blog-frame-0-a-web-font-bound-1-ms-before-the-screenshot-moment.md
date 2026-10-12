---
id: 000482
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '5a7da3545e84ecf0b42d6fb8bc3a83196e2d195b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T06:43:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6c5fbfe7", done: "eeabba4d"}
actual_hours: 0.47
---

# broadcast-web: fidelity flake on rec-blog frame 0 — a web font bound 1 ms before the screenshot moment

## Problem

`TestLiveFidelityFixtures` (#478) scores rec-blog frame 0 at exactly 6.75% in about 1 run in 8 on main (2 in 8 on #475's branch), otherwise ~0.10%. pair:1's reading: the Inter variable font's asset is bound at 146 ms and the frame is at 147 ms, so whether the replay at 147 has finished applying the font depends on the viewer's font load and decode under load. A live cast hits the same race at every new document whose fonts arrive just before a screenshot.

## Spec

Find whether the replay at a frame time can be judged before fonts bound before that time have finished loading (FontFace load and decode in the viewer), and make the render deterministic: e.g. the viewer's render at t waits on `document.fonts` for faces bound at or before t (with a bound), mirroring the browser, which had the font when it painted. Fix in the viewer/replayer, not by widening the tolerance or retrying. Check the other fixtures for the same pattern.

## Done when

- rec-blog frame 0 scores ~0.1% on 100 consecutive runs under load; the cause is in the Log.
- A test pins the font-bound-just-before-frame case.

## Plan

- [x] Pin it: a live test with a font and an image bound 1 ms before the moment (`TestLiveReplayAtLoadsAssetsBoundJustBefore`), and a fake-deps unit test of the capture's order (pause at t → wait for the loads → pause what they started).
- [x] Fix `Replayer.replayAt`: after the window, pause at t, `settle()` (the renderer's bounded `ready`: sheets, fonts, images), pause anything newly started.
- [x] Check the other fixtures; atlas line in `atlas/webcast.md`.
- [x] 100 runs of rec-blog under load (4 concurrent Chromes + 4 CPU burners).

## Log

### 2026-10-12
- 2026-10-12: closed — Pinning test TestLiveReplayAtLoadsAssetsBoundJustBefore fails 3/3 on main (font loading, img incomplete), passes 5/5 with the fix; unit test of capture order fails on the mutant. rec-blog frame 0 = 0.10% on 100/100 runs (4 concurrent Chromes + 4 CPU burners). Full live webcast suite green (fidelity: all fixtures within, known gaps unchanged). make -k test and go test ./... green unsandboxed.; review verdict: FIX-THEN-SHIP
- **Cause (measured):** `replayAt` captured right after `until(t)`. It waited only for a held paint's stylesheets (#475), never for assets fed during the real-time window. rec-blog frame 0 (t=147) replays entirely inside that window. The Inter font's asset arrives at 146, and the avatar image and the giscus sheet arrive just before it, so whether the capture saw them depended on load and decode speed under load. pair:5 independently saw the 6.75% frame as the missing avatar and scrollbar, not the font. Both are the same cause. The pinning test fails 3 of 3 on main (`Late:loading img:false`) and passes with the fix.
- **Fix:** pause the animations at t, then `settle()` (the renderer's `ready`, bounded at 5 s), then pause only the animations the loads started. Fonts, images and sheets bound by t are all covered. A font-only `document.fonts` wait would have missed the avatar.
- **Soak (Done when):** 100 consecutive rec-blog fidelity checks, run as 4 concurrent headless Chromes x 25 rounds with 4 CPU burners alongside, scored frame 0 at exactly 0.10% in all 100 runs (worst 0.10%). The report is in the scratchpad's soak.log.
- **Other fixtures:** no other asset or font is bound within 150 ms before a frame. The first frames of rec-app, rec-segments and rec-stress (56, 101, 84 ms) also replay entirely in the window, so the fix covers them too. rec-stress's known gap is different: photo.png's bytes arrive after the frame.
