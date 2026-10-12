---
id: 000474
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4f92388a605b987d57f29119d519a8064369536d' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: retire the prototype fixtures once #470 and #471 land

## Problem

pair#472 added the real recorder's fixtures (`rec-stress`, `rec-app`, `rec-blog`, `rec-segments`) beside #453's
prototype ones (`stress`, `app`, `blog`), so the components being built on the prototype fixtures (#470 notes UI,
#471 replay) keep passing. Two sets cost about 7 MB in the repository (the fixture budget is 9 MB while they coexist).

**Trigger:** #470 and #471 have landed and the TL (ops:0) says to retire. Not before.

## Spec

- Delete `stress`, `app`, `blog` and their query goldens, `record-mjs.patch`, and the prototype rows of the fixtures
  README; drop `recording.PrototypeFixtures` and `observerLag`'s 150 ms branch (`replay_test.go`).
- Move every test that names a prototype fixture (`recording.Fixture("app")`, `"stress"`, `"blog"`: query, notes,
  replay, `viewer_live_test.go`, fidelity) to its `rec-*` twin; re-pin expectations that read a fixture's content.
- Optionally rename `rec-*` to the bare names in the same change (one rename, goldens included).
- Budget back to 4.5 MB.

## Done when

- No prototype fixture remains; every component's tests run on the real recorder's fixtures only; budget 4.5 MB.

## Plan

- [ ]

## Log

### 2026-10-12
