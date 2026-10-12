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
flow: {kind: quick, provenance: inferred, spec: "e905b066", done: "1986943c"}
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

- [x] Record raw runs with the #458 recorder (headless, `TestRecordFixtures` in `cmd/internal/webcast`, gated by env): `stress` idle from testdata, #453's `app` scenario (pages from the prototype branch, served locally), the blog `browse` scenario on `npm run dev`, and a `segments` run (pause, resume, a new tab, a same-document navigation, a cross-document click).
- [x] `fixturegen -recorded RUN -name NAME -out DIR`: a recorder directory plus the notes spec → fixture + pins (no CDP conversion).
- [x] Add them as `rec-stress`, `rec-app`, `rec-blog`, `rec-segments` beside the prototype fixtures; coverage checks per real fixture; query goldens; replay's observer lag 0 for them (150 stays only for the prototype names).
- [x] Size: trim runs; raise the fixture budget only by what both sets need while they coexist, and say it drops at retirement.
- [x] Driven stays hand-written until #467 exists.
- [x] File the retirement of the prototype fixtures, triggered by #470 and #471 landing (TL calls it): #474.
- [x] Verify: `go test ./cmd/internal/webcast/...`, `WEBCAST_LIVE_CHROME=1 -run LiveReplay` on all fixtures, node viewer tests, full `make -k test`.

## Log

### 2026-10-12
- Recorded with the #458 recorder (headless Chrome 154): stress idle 5 s, app 22 s, blog browse 12 s, segments 10 s; 0 dropped, 0 desync, every run validates. Screenshots every 2 s keep the four at 3.3 MB (budget 9 MB while both sets coexist).
- The guard found four things the prototype fixtures hid:
  - **Recorder bug, fixed:** `shotDone` clamped a frame's `t` to the stream's latest record, and it runs after the layout capture, so on a busy page frames were stamped 43–87 ms after their pixels (probe: midpoint 2117 → stamped 2160; 4369 → 4456). The format says a frame's `t` is the capture's midpoint. Now the frame keeps it (never before its segment) and only the layout record takes the stream's time. `TestRecorderFrameKeepsItsCaptureTime` fails on the old clamp (mutation-checked). rec-stress frames went from 4.2% off at 0 ms to 1.2%.
  - **Test harness, fixed:** the replay harness drew every fixture at 1280×800; a tab switch to a different-sized tab (rec-segments, 756×469) compared at the wrong size. It now sizes the page from the model's viewport, as `viewer.js` does. The fidelity tool's offsets also skip moments in another segment.
  - **Query oracle, fixed:** `TestAgentCanTellWhatEachNoteRefersTo` didn't know `selection` or `nav`, which `query.UserAction` counts (#462); the real recorder records selection before notes.
  - **Recorder/viewer gap, filed #475:** a segment's linked stylesheet binds when fetched (rec-stress `stress.css` at 103, first frame at 84), so viewers briefly draw the page unstyled. `replay_test.go` lists rec-stress frame 0 in `knownGaps`, which fails once it passes.
- Observer lag is 0 for the real fixtures (150 stays only for `recording.PrototypeFixtures`). A frame may also compare 50 ms before its `t` (`captureSkew`): the pixels are a compositor frame from inside the screenshot call, measured on rec-app frame 5 (77% at t when a modal closed, 0.4% at t−50).
- Fidelity tool (median / max %): rec-stress 0.05/1.30 (prototype stress 0.65/0.75), rec-app 0.00/0.41, rec-blog 0.06/3.07, rec-segments 0.00/0.00.

