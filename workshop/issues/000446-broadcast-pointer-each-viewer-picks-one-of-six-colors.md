---
id: 000446
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '62cf9a34529ebb0bb1e2d6225e391b9a9b18f4e7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T11:57:39-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a95453dc", done: "1cff5ae0"}
---

# Broadcast pointer: each viewer picks one of six colors

## Problem

Every broadcast viewer's laser-pointer marks draw in the same amber (xterm 214, `cmd/internal/broadcast/marks.go:23`), so when several viewers point at once nobody can tell whose mark is whose. The operator (10-11) wants each viewer to pick a color from a fixed set of six.

## Spec

- **Palette:** six colors, colorblind-safe (Okabe–Ito based): orange (index 0, closest to today's amber, the default), sky blue, bluish green, yellow, vermillion, reddish purple. Each has a truecolor value and an xterm-256 index, and must read on both dark and light backgrounds.
- **Wire:** `PointBatch` gains `color` (integer 0–5, optional; absent = 0). `ParsePointBatch` stays strict: any other value is refused. Viewers never send a raw color, so nothing arbitrary reaches the operator's terminal.
- **Drawing:** a mark draws in its batch's palette color. The truecolor fade still blends into the operator's background (`SetBlend`); without truecolor, each palette color has its own five-step fade ladder (today `markFadeLadder` is amber only).
- **Picking:** the viewer page shows six swatches; the choice is remembered per browser (localStorage, wrapped in try/catch, so the page still works without it). A viewer that hasn't picked gets the next unused color from Couch (per connection), so two viewers rarely share one by accident.
- Out of scope: a legend mapping colors to names.

## Done when

- A batch with `color: 3` draws in palette color 3; `color: 6`, `-1` or a string is refused (tests).
- Each color fades correctly with and without truecolor (tests on the ladder and the blend).
- Two live viewers pointing at once show two different colors on the operator's terminal and on each other's view (checked live in a broadcast).
- The picker persists across a page reload, and works when storage is blocked.

## Plan

- [ ] Palette with truecolor values, 256-color indexes and fade ladders; `color` in `PointBatch` with strict parsing; tests.
- [ ] Draw marks by palette color; per-connection default assignment.
- [ ] Viewer swatches and remembered choice; viewer.js tests.
- [ ] Live check with two viewers; atlas broadcast page.

## Log

### 2026-10-11
- Design (ARCH-DRY): the palette is single-sourced in Go (`marks.go`); the pointer stream's `caps` event carries it as `palette` (hex list) plus this connection's assigned `color`, so viewer.js hard-codes no colors. `PointerState` counts the colors held by live pointer streams and assigns the least-held (lowest index on ties); a stream releases its color when it ends. A viewer's own pick only overrides locally (Couch never learns picks except through batches), so assignment counts connections, not picks. `Marks` keeps a color per cell; `Add` takes the batch's color. `color` parses into `*int`, which refuses strings and non-integers outright; the range check refuses the rest.
