---
id: 000472
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '562c00f609f714b8d81687f2343350049c20f601' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: real recorder output as fixtures (the integration guard)

## Problem

couch-broadcast-web's wave-1 components were built against pair#457's fixtures, recorded with #453's prototype and patched. The project's integration guard says: when the recorder (pair#458, landed PR #263) lands, its real recordings replace the fixtures in every component's contract tests. Not done in #458. `replay_test.go`'s `fixtureObserverLag` tolerates the prototype's polled observer.

## Spec

- Record real fixtures with the #458 recorder (headless): the stress page, the app, the blog (`~/workspace/xianxu.dev`), a segments case (pause/resume, tab switch, navigation) and a driven case once #467 exists (until then keep the hand-written driven fixture).
- Add them alongside the prototype fixtures first, so the components being built now (#470 notes UI, #471 replay) keep passing; every component's contract tests and the viewer's checks run on them. Set the observer lag to 0 for real fixtures.
- Once #470 and #471 have landed, retire the prototype fixtures (the TL says when).
- Keep fixture size reasonable (they're in the repo): trim to the pages and moments that cover the format.

## Done when

- Real-recorder fixtures are checked in and every component's tests (reducer, validator, catch-up, viewer, notes, queries, assets) pass on them, with zero observer lag.
- The retirement of the prototype fixtures is either done or filed with its trigger.

## Plan

- [ ]

## Log

### 2026-10-12
