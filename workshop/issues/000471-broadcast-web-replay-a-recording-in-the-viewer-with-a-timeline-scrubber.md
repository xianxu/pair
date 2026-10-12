---
id: 000471
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'b74da30f7a6f8bf356f253df7e13f0575a3d0905' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T03:03:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "cb2ab542", done: "da4898b1"}
---

# broadcast-web: replay a recording in the viewer with a timeline scrubber

## Problem

Wave 2 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`). A recording on disk can be queried by agents (pair#462) but not watched: the viewer (pair#460) only plays a live stream.

## Spec

- Open a recording directory in the viewer (served by Couch: a route listing recordings and serving one) and play it with the same model and renderer as live.
- A timeline scrubber: segments and pauses shown as gaps; seek by reducing from the latest `seg` before t (the reference reducer's rule); play, pause, speed.
- Notes appear at their moment on their element; strokes replay; who drove (`by`, `ctl`) is visible on the timeline.
- Agent-facing: the URL of a recording at time t (e.g. `#t=1530`) so an issue or a DOM submission can link to the exact moment.

## Done when

- Every fixture replays with seeking to any pin giving the reducer's canonical state (reuse #460's checks).
- Notes and strokes show at their moments; a `#t=` link opens at that moment.
- The operator smoke-tested replay of a real recording (after #458), scheduled by the TL.

## Plan

Durable plan: `workshop/plans/000471-broadcast-web-replay-a-recording-in-the-viewer-with-a-timeline-scrubber-plan.md`. Single pass, one close.

- [x] catchUp, timeline, placer, #t= (pure JS) with the catch-up invariant at every cut of every fixture
- [x] Player: seek and play state machine; every pin is reached forward, backward and shuffled
- [x] replaySource + the timeline bar; one branch point in viewer.js mount
- [x] assets.Reopen: a read-only Served over a finished recording
- [x] webcast.ReplayServer: listing, records, assets, modules; ids confined to the root
- [x] couch --broadcast-web-replay [ID|DIR] [--at T]: serves on loopback and prints the #t= link
- [x] Live headless check: #t= and scrubbing give the reducer's canonical at every pin; JS catchUp equals Go's
- [x] Atlas, then close

## Log

### 2026-10-12

- TL decisions (ops:0): recordings go in `<store>/webcast/recordings/<id>/` (#458); the loopback CLI is the entry point now, and the Host mount is a follow-up the TL files. #470 (pair:6) edits viewer.js, index.html and viewer.css too, so my edits there stay small and additive, and whoever lands second merges.
- The seek rule is the catch-up rule (`recording.CatchUp`), ported to JS so scrubbing doesn't round-trip; the port is held to Go's by its invariant and by a live parity check (ARCH-DRY).
- Built as planned. Fixtures have no asset before a later segment, and no note on an earlier document, so the catch-up invariant also runs on a synthetic recording that has both. The "check bites" test drops the earlier assets and sees the invariant fail.
- Screenshots of the bar found two bugs that no test caught. The track was laid out before its time label had text, so a paused replay kept marks under the label. A note near the top of the page had its pin clipped. Both are fixed (lay out after the labels; the pin goes below when there's no room above).
- The flow was inferred as quick. The code passes the 100-line shell, so close will upgrade to the full review; I didn't pin it, since `--flow` is the operator's.
- Verified: node 61/61; `go test` for webcast, assets, couchcmd and artifactpath; live headless (`WEBCAST_LIVE_CHROME=1`) replay checks plus #460's viewer checks; `couch --broadcast-web-replay <fixture> --at 1500` serves the page and records. `make -k test`: only `TestCapturePTYBothBoundaries` (a drain timeout under load, which passes alone) and the skill parse, now fixed.
- Follow-ups: mount `ReplayServer` on the shared Host inside running Couch (the TL files it); have replay draw notes and strokes with #470's modules once both have landed.
