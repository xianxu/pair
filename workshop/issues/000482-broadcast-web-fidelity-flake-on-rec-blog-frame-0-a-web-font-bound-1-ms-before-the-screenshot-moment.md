---
id: 000482
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '18129e25c037053fde6cd2ef65d1671b24542165' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
