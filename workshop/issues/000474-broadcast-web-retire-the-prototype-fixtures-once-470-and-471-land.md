---
id: 000474
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'af2d03b5b143a20f0fcde079d2df01bd782cc6ac' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T05:47:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b6474e38", done: "329bc4e8"}
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
- Replace `testdata/replay/harness.js`'s copy of `viewer.js`'s viewport sizing with the viewer's own (one source), and derive the hand-listed fixture names (`recording.Fixtures`, the fidelity list, `TestFixtureCoverage`'s `want`) from one place.

## Done when

- No prototype fixture remains; every component's tests run on the real recorder's fixtures only; budget 4.5 MB.

## Plan

- [x] Wait for the TL's go (after #470 and #471 land).
- [x] Move every test naming `stress`/`app`/`blog` to its `rec-*` twin; delete the prototype fixtures, goldens and `record-mjs.patch`.
- [x] Drop `PrototypeFixtures` and the 150 ms branch; single-source the harness's viewport sizing and the fixture-name lists.
- [x] Budget back to 4.5 MB; README and atlas rows; `go test ./cmd/internal/webcast/...` and the live replay.

## Log

### 2026-10-12
- TL's go (ops:0) after #470 (PR #269) and #471 landed. Kept the hand-written `driven` and `segments` fixtures.
- Skipped the optional rename of `rec-*` to bare names: `rec-segments` would collide with the hand-made `segments`,
  and the rename churns every golden for no reader's benefit.
- Retired `fixturegen -raw` (`cdpconv.go`, ~950 lines) with the fixtures: with `record-mjs.patch` gone it can't be
  run, and its only output was the prototype fixtures. `excluded` and the prototype note specs went with it.
- Fixture names have one source now (ARCH-DRY): `recording.Fixtures()` lists every directory with a pin file, as
  `tests/webcast/fixtures.mjs` already did; `TestFixtureCoverage` also fails on checks for a fixture that's gone.
  The fidelity list already derived from it. The harness's viewport sizing was single-sourced by #478
  (`Replayer`'s `layout`), so nothing was left there.
- `observerLag` gone, so `frameOffsets` and `checkFidelity`'s offsets parameter went too: every caller passes
  `fidelity.Offsets`.
- **Found by the move:** `query events` didn't fold the real recorder's typing. The recorder reports a collapsed
  caret (`selection`) after every key, which the prototype's polling never saw, so each key was its own line (the
  #472 lesson again). A collapsed caret by the typist now folds into the typed value; another person's caret or a
  range still splits it (`TestEventLinesFoldCaret`, mutation-checked: disabling the rule fails 2 tests). Goldens
  for `rec-app` shrink accordingly; the agent check skips a caret folded into typing.
- Re-pinned: `TestEventLinesFoldTyping` and `TestMainJSON` to `rec-app`'s times and node ids; the viewer live tests'
  `at` from the prototype blog's last layout (10041) to `rec-blog`'s (8284). The board stamps live strokes at the
  last record's time, so the annotate test's stroke filter found none at 10041.
- Fixtures: 3.3 MB, budget 4.5 MB.
- Live (unsandboxed, `WEBCAST_LIVE_CHROME=1`): replay, join, viewer, annotate, fidelity green. `rec-blog` frame 0
  once scored 6.75% in `TestLiveFidelityFixtures` (1 of 4 runs; 0.17% max on the other 3). Its check is unchanged
  from main, so a pre-existing flake, reported to the TL.
- termcmd `TestNoDeclarationCarriesTwoStackedGodocs` passes: #478's `575368d8` fixed `couchcmd/webcast.go`'s godoc.
