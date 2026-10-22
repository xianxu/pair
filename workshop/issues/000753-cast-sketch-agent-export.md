---
id: 000753
status: working
deps: [pair#752]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '91fd0dc64275e82421b4731ba4af06191eee101d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T18:40:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
---

# cast-sketch: operator CLI queries and historical image export

## Problem

Make approved sketch evidence usable by a coding agent through the operator’s local recording and CLI, including a historical pressure-faithful rendered image.

## Spec

Current scope is the 2026-10-21 operator-submission revision below. It supersedes the older CLI-only/no-chord-extension clause while preserving historical rendering, privacy and no viewer-to-agent dispatch. The detailed proposal is in [the slice plan](../plans/000753-cast-sketch-agent-export-plan.md); TL review precedes implementation.

Approved design: #750, workshop/plans/000750-cast-sketch-plan.md, especially its 2026-10-20 operator approval revision. This issue implements only its assigned slice. Start from main after dependencies land; write a slice-specific execution plan and pass the SDLC gates. No stacked branches.

Add recording query sketches with optional ID and JSON, plus sketch-image --sketch ID --out DIR and optional history cutoff. Include required label or explicit unfinished status, author/colour, source moment/page, original anchors, original vector coordinates and pressure, Undo/Redo projection and truncation reasons. Read the original source DOM at begin while selecting later accepted stroke/label history.

Reuse existing sandboxed Model/Renderer/Replayer and extract the shared headless replay lifetime from fidelity_check.go behind ImageRenderer with a stateful fake. Render only recorded assets, compose pressure-aware strokes, emit bounded PNG and JSON transform/provenance into an explicit private output directory. Missing assets/uncaptured content/empty sketches/clipping/timeouts remain explicit; never substitute current live-tab screenshots. Cancellation joins Chrome/CDP/server/profile cleanup. New exporter must preserve existing fidelity behavior.

Agent handoff is ONLY through the operator: recording plus local CLI export. Add no viewer-side submission, agent dispatch route, or note-submission chord extension. Preserve origin privacy and reject overwriting existing output or writing into source recordings.

Primary surfaces: query/{sketch.go,render_sketch.go,surface.go,fake.go,cli.go,events.go,contract.go}, sketch_image.go, sketchimage/{contract.go,fake.go}, fidelity_check.go, web/sketch-image.js, couchcmd query wiring, shipped Couch query skill guide.

## Done when

- Query contract/goldens cover labelled/unfinished/interrupted/cleared/undone/redone sketches and source-vs-history cutoff identity.
- JSON includes anchors, original x/y/pressure paths and image transform/provenance; one recording pass serves all requested strokes.
- Stateful fake and bounded real-headless conformance exercise success, missing assets, pressure pixels/geometry, empty results, existing-output refusal, timeout/cancel and cleanup.
- Existing fidelity tests pass after renderer extraction. No viewer-side agent submission is introduced.
- README/atlas/shipped query guide explain private operator handoff and replay-rendered image limitations; SDLC checks/review pass.

- Each submitted sketch includes its image (or an explicit qualified rendering limitation), attached note, original page URL, recording anchor handles and bounded declared element IDs/data-* attributes from the source DOM.
- Sketch-only and mixed note/sketch operator submissions use the existing pinned-target transaction; no new browser-to-agent route. Frozen versions survive navigation, uncertain delivery and restart without false acknowledgment or automatic resending of delivered versions.

- Before sending, the console shows the selected count and shapes, oldest eight unsent first with visible N more; stale selected versions refuse rather than silently changing the displayed batch.

## Plan

- [x] Claim/start-plan after dependencies land; write the slice-specific execution plan.
- [ ] Implement and verify the Spec and Done when through the normal SDLC gates.

## Log

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.
- 10-21 16:55 (TL ops:0): scope addition, generic (from ops-echo:0's echo-control-board, operator-approved): every
  sketch or annotation handed to an agent (export and CLI query) carries the **page URL** and the **declared stable
  IDs** of the elements it's anchored to or under: the element's `id` and its `data-*` attributes (bounded: the anchor
  element plus its ancestors up to the first one with a declared ID, and the elements the sketch's bounding box covers,
  capped and listed nearest first). Pair needs no knowledge of what the attributes mean; apps opt in by marking their
  elements (e.g. echo's room/turn/agent data attributes), so an agent gets exact handles instead of guessing from a
  screenshot. Values are recorded as they were at sketch time (from the DOM recording), not re-read later.

## Revisions

### 2026-10-21 — operator-triggered sketch delivery

TL dispatch cbb63f0f-6bb9-4256-bc49-0a4e49eb6e58 and confirmation 0f71ee20-807f-4bd3-9a96-1bb6699f93f5 prioritize #753 before #783 and require each shape's image, note, page URL and element IDs to reach the agent as the operator's submission. This supersedes the old CLI-only/no-submission-chord-extension restriction. Reuse the existing operator draft submission transaction and durable bundle; no browser-to-agent route or viewer dispatch is introduced. A shape means one finished sketch (which may contain multiple strokes sharing one label), matching #752's accepted flow; proposal includes the scope explicitly for TL review.

The earlier generic element-identity addition remains: original anchor, ancestors through first declared ID, and nearest intersected recorded elements carry bounded id/data-* attributes with truncation facts. No application-specific meanings are inferred.

### 2026-10-21 — design checkpoint

#752 landed PR512 after operator acceptance and an integrated recorded suite pass; #771 landed PR527 via train254. This issue started from fresh main after those dependencies, not a stacked branch. Read-only audit identified existing submission selection, pinned-target delivery, durable publication/ack and replay renderer seams. Proposal preserves initial note crop semantics by adding typed sketch evidence, reuses the renderer behind a stateful fake, and derives delivered-version filtering from confirmed existing submission manifests rather than adding a recording event or acknowledgment sidecar (ARCH-DRY/ARCH-ORDER). Plan is pending TL review; no implementation or estimate is authorized yet.

### 2026-10-21 — TL design approval and preview acceptance

TL745ea763-8e84-4ac2-8897-f40e82a884c0 approves the design, one completed multi-stroke sketch per package, oldest eight unsent first with visible N more, and exact dispatched/resolved-version filtering across targets. Adds operator visibility of selected count and shapes before the existing submission chord sends. Plan adds one read-only preview of the same immutable selector through the private console projection; no browser dispatch. Proceed to full-flow change-code, without deriving an estimate before plan-quality passes.

### 2026-10-21 — plan-quality refinement

Round1 returned Important PQ-1 (explicit-event-state-ownership) and PQ-2 (function-level-test-strategy). Specified NextPreview's states/events, exact publication identity and atomic manager→board freeze guard, including worker cancellation/join and opaque preparation token. Replaced test-case prose with named function strategies across all pure risk seams. This refines the approved preview mechanism without changing product policy. No implementation or estimate yet.
