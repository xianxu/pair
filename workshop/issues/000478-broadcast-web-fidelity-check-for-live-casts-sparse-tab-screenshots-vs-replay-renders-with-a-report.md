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

- [ ] Pure `webcast/fidelity`: #453's diff (moved from the recorder's dev tool), tolerance, report, side-by-side crops
- [ ] `web/replayer.js` (the harness's replay as product code); harness on it
- [ ] `fidelity.html`/`fidelity.js` + `CheckFidelity` (headless, the viewer's renderer); live fixture + broken-render tests
- [ ] `CastConfig.Check`: the check runs when a recorded cast ends
- [ ] `couch --broadcast-web-fidelity [ID|DIR]`
- [ ] Live blog cast within tolerance; atlas (webcast.md, webcast-format.md)

## Log

### 2026-10-12
