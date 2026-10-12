---
id: 000477
status: codecomplete
deps: [pair#470]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 2.09
card_mirror: '3e85da85535541e31d579140ba8566f5c0060386' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T05:09:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.54
---

# broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host)

## Problem

pair#471 (PR #266) replays recordings via `couch --broadcast-web-replay`, on loopback only. A colleague on another machine can't open a replay link, and an issue or a DOM submission can't point a reviewer at a moment of a recording (couch-broadcast-web, ops `teams/devinfra/projects/couch-broadcast-web/project.md`).

## Spec

- Mount `webcast.ReplayServer` on the shared broadcast Host inside a running Couch (the endpoint and tunnel #454 shares), behind the cast's token model, so replay links (`…/r/<id>/#t=MS`) work for remote viewers.
- How a replay link is produced: from Couch (e.g. a menu action on a recording, or `couch --broadcast-web-replay ID` printing the public link while Couch runs), and for agents (the query CLI can print the link to a moment).
- #471's advisory: a recording whose last line doesn't parse lists its length as unknown, not 0.
- Replay draws notes and strokes with #470's modules once #470 lands (coordinate through the TL).

## Done when

- A replay link opened from another device plays a real recording through the tunnel, seeks, and shows notes.
- The length advisory fixed; tests; the operator's smoke test, scheduled by the TL.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.20 impl=0.35
item: smaller-go-module design=0.05 impl=0.10
item: smaller-go-module design=0.10 impl=0.15
item: smaller-go-module design=0.10 impl=0.20
item: smaller-go-module design=0.10 impl=0.20
item: atlas-docs design=0.03 impl=0.07
item: milestone-review design=0.05 impl=0.20
total: 2.09
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Items in order: `ReplayShares` state machine (greenfield); scoped `ReplayServer` + length advisory; broker op + Couch wiring; both CLIs' share-and-poll; M2 drawing with #470's modules + live check; atlas/skill; two milestone reviews.

## Plan

Durable plan: `workshop/plans/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host-plan.md`.

- [x] M1 — replay shares (one token per recording) on the shared Host; broker op `replay-share`; `couch --broadcast-web-replay ID` prints the public link while Couch runs (loopback fallback, `--local`); `--broadcast-web-query DIR link`; length-unknown advisory; atlas + skill
- [x] M2 — after #470 lands: merge main; replay draws notes and strokes with #470's modules; live check

## Log

### 2026-10-12
- 2026-10-12: closed — M1+M2 SHIP. Full make -k test exit 0 on the merged branch (251 Go packages); WEBCAST_LIVE_CHROME TestLive* webcast all pass: TestLiveReplayShareSeeks (a share link through the Host opens at #t= and seeks), TestLiveReplayServerMarks (#470 modules: pins + canvas strokes vs reducer, tap card, resize); node 88/88. Tunnel smoke test from another device is the operator, scheduled by the TL.; review verdict: SHIP
- 2026-10-12: closed M2 — review round fixes: card placed on tap + resize redraw (each mutation-checked: removing either fails TestLiveReplayServerMarks); atlas webcast-format/webcast updated; all TestLiveReplay* pass; make -k test exit 0 before these JS/test/doc-only changes; node 88/88; review verdict: SHIP
- 2026-10-12: closed M1 — round 2 fixes 7f856ae6: docs swept (README synopsis+prose, couch --help, atlas broadcast.md/webcast.md/webcast-format.md, SKILL); notice Local-aware with TestReplayShareReadyIsANotice; touched packages green unsandboxed; live Chrome share test passes; review verdict: SHIP

- M1 built. `webcast.ReplayShares` mounts one `NewRecordingServer` per recording on the shared Host. The broker ops
  are `replay-share` and `replay-status`. `couch --broadcast-web-replay ID` prints the shared link and exits; with no
  Couch, it serves on loopback. `couch --broadcast-web-query DIR link [--at]` prints the same link for agents. The
  length advisory is fixed (`Listed.Ended`, `Length()` says `unknown`). See the plan's Revisions for the
  `Share`/`Status` split (ARCH-ORDER: a poll must not restart a failure).
- Pre-existing test race fixed in passing: `couchcmd/replay_test.go`'s `lineWriter` had its channel field read by
  the test while `Write` cleared it. It failed under `-race`.
- Live: `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'LiveReplayShare|LiveReplayServerSeeks'` passes.
  A real Chrome opens a share's link through a LocalOnly Host and seeks with `#t=`.
- M2: merged origin/main (#470, PR #269). Replay's own `drawMarks` (an SVG of strokes, and note text pinned
  inline) is replaced by `replayMarks`, which uses #470's `liveNotes`/`pinLayer` and `strokesAt`/`paintStrokes`.
  A pin's tap opens the note read-only. `TestLiveReplayServerMarks` now compares the pins and the canvas's
  stroke ids with the reducer at ten moments, and checks that the canvas has ink exactly when strokes show and
  that a tap opens a note. Main's fidelity harness served `/web/` straight from `webFS`, so `collab.js` 404'd once
  `replay.js` imported `annotate.js`. It now goes through `serveViewerFile`, the one table (ARCH-DRY). All
  `TestLive*` webcast tests pass with `WEBCAST_LIVE_CHROME=1`, and the node tests pass 88/88.
