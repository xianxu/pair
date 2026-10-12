---
id: 000472
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '23a793b37a8ac71b170266ac8dacf27d65af7c77' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T03:13:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

Quick flow (fixturegen gains one small mode; the rest is tests and data).

- [ ] Record raw runs with the #458 recorder (headless, `TestRecordFixtures` in `cmd/internal/webcast`, gated by env): `stress` idle from testdata, #453's `app` scenario (pages from the prototype branch, served locally), the blog `browse` scenario on `npm run dev`, and a `segments` run (pause, resume, a new tab, a same-document navigation, a cross-document click).
- [ ] `fixturegen -recorded RUN -name NAME -out DIR`: a recorder directory plus the notes spec → fixture + pins (no CDP conversion).
- [ ] Add them as `rec-stress`, `rec-app`, `rec-blog`, `rec-segments` beside the prototype fixtures; coverage checks per real fixture; query goldens; replay's observer lag 0 for them (150 stays only for the prototype names).
- [ ] Size: trim runs; raise the fixture budget only by what both sets need while they coexist, and say it drops at retirement.
- [ ] Driven stays hand-written until #467 exists.
- [ ] File the retirement of the prototype fixtures, triggered by #470 and #471 landing (TL calls it).
- [ ] Verify: `go test ./cmd/internal/webcast/...`, `WEBCAST_LIVE_CHROME=1 -run LiveReplay` on all fixtures, node viewer tests, full `make -k test`.

## Log

### 2026-10-12
