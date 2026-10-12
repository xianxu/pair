---
id: 000478
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '6a464aa06a5d36292c4e994d008ba0d03eed3589' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T04:00:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "726046fc", done: "382f68cf"}
---

# broadcast-web: fidelity check for live casts — sparse tab screenshots vs replay renders, with a report

## Problem

Principle 6 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`): fidelity is measured, not assumed. #453's harness and #458/#460 measured fixtures headless; a live cast has no check, and the recording format reserves `frames/` (sparse low-res screenshots of the real tab) for exactly this.

## Spec

- During a live cast the recorder takes sparse low-res screenshots of the tab (on navigation and every few seconds; `frame` records in `frames/`, never streamed), cheap enough not to disturb the page.
- A check renders the replay at the same moments (headless, #460's renderer) and compares, reusing #453's diff; a report per recording (median/max, the worst moments with side-by-side crops).
- Run it from the CLI on any recording and automatically at the end of a cast; desktop Chromium targets exact fidelity; record which moments exceed the tolerance so fidelity bugs become issues.

## Done when

- A live blog cast produces frames and a fidelity report within the stated tolerance; a deliberately broken render is caught.
- Tests; the report format documented in the atlas.

## Plan

Durable plan: `workshop/plans/000478-broadcast-web-fidelity-check-for-live-casts-sparse-tab-screenshots-vs-replay-renders-with-a-report-plan.md`.

- [x] Pure `webcast/fidelity`: #453's diff (moved from the recorder's dev tool), tolerance, report, side-by-side crops
- [x] `web/replayer.js` (the harness's replay as product code); harness on it
- [x] `fidelity.html`/`fidelity.js` + `CheckFidelity` (headless, the viewer's renderer); live fixture + broken-render tests
- [x] `CastConfig.Check`: the check runs when a recorded cast ends
- [x] `couch --broadcast-web-fidelity [ID|DIR]`
- [x] Live blog cast within tolerance; atlas (webcast.md, webcast-format.md)

## Log

### 2026-10-12

- Frames needed no recorder change: #458 already shoots at each segment start and every 5 s while live, into
  `frames/`, never streamed. No recorder change went through the TL.
- Retired `recorder/internal/fidelity` (a static Go renderer, i.e. not the renderer viewers see). Its #453 diff
  moved to the pure `webcast/fidelity`. The harness's `replayAt` became `web/replayer.js`, shared by #460's
  conformance test and the check (ARCH-DRY).
- Measured with #453's diff, the 3% tolerance:
  - fixtures: stress median 0.63% / max 0.74%; app 0.00 / 0.41; blog 0.06 / 0.12 (blog's frame at t=5810 matched
    at an offset: 14.9% at t, the prototype observer's polled scroll lag, which #460's test allows +150 ms for).
  - live blog cast (`TestLiveRecordURL`, localhost:4321, browse, 20 s, 17 frames, #458's recorder):
    `couch --broadcast-web-fidelity` gave median 0.00%, max 0.10%, exit 0.
  - the deliberately broken render (blog without stylesheets): median 81.33%, max 99.41%, 5 of 5 over.
- Found via the first report's crop: the first live frame was 2.15%. The heading was mid fade-in in the replay but
  done in the tab, because the page had loaded before go-live. A `live`/`resume`/`tab` segment's document was
  already running, so the replayer now finishes its animations (`navigate` stays real-time). That brought it to
  0.10%. Test: `replayer.test.mjs`.
- Retries: a frame over the tolerance is replayed at ±150/±300 ms and the best kept. The report keeps the offset and
  the diff at t, so a timing slip stays visible as one.

