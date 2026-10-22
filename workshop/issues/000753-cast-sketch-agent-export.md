---
id: 000753
status: working
deps: [pair#752]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 8.4
collaboration-mode: collaboration
card_mirror: '75006839ae1ea2b30db875516294910eb19ff9ec' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T18:40:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

## Estimate

*Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is marked stale by sdlc estimate-source; the derivation is provisional rather than a new calibration claim.*

Library availability: no new dependency or novel platform is assumed. Reuse encoding/json plus the existing strict/preserving recording-codec pattern, the existing query reducer, CDP/replay session, pressure painter and durable submission store. Replay lifetime and file inventory are costed as extraction/refactor rather than greenfield services; no external shim replaces the domain-specific version/preview contracts. Familiarity is 1.0 for this session's recently worked recording/console stack.

Apply the thorough-spec design discount of 0.2 to the chosen v2 table design values and the v3.1 implementation scale of 0.4 once. The rows below map in order to: scope/spec; historical sketch query; declared element facts; render plan using geometry; shared replay session lifetime; recorded image adapter/export; typed sketch evidence; preserving submission codec; one evidence-file inventory; delivered version selection; preview policy/freeze; private preview cards; private preview publication; submission status and text; query/privacy guides; single close review; native Chrome conformance. Each row is a distinct seam including its colocated tests. Design subtotal 3.44h +15% buffer; scaled implementation subtotal 4.44h; total 8.40 focused ship-hours. One final close review is included; no invented milestone boundaries or cross-repo overhead.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: greenfield-go-module design=0.3 impl=0.32
item: greenfield-go-module design=0.2 impl=0.24
item: smaller-go-module design=0.06 impl=0.2
item: cross-cutting-refactor design=0.12 impl=0.2
item: api-integration design=0.4 impl=0.6
item: greenfield-go-module design=0.2 impl=0.24
item: greenfield-go-module design=0.3 impl=0.32
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.4 impl=0.4
item: api-integration design=0.4 impl=0.4
item: lua-neovim design=0.3 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 8.40
```

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

### 2026-10-21 — implementation admitted; first pure slices

Full-flow change-code passed after plan-quality disposed PQ-1/PQ-2; the 8.4h v3.1 estimate reconciles. Estimate-quality INFO notes are advisory: the estimate covers the whole issue including authored design, native fixture/agent-path work is included in the integration/UI rows, and real-api-discovery covers Chrome requalification rather than all acceptance tests. Calibration remains provisional as recorded.

TDD started with independent pure query, preview and render-plan slices. Render Plan was RED for missing API, then GREEN; stronger tests caught original anchor metadata aliasing and inconsistent crop/scale under a one-pixel narrow-image cap, both fixed. Focused tests pass0.148s; ten-second bounded-transform fuzz passes334554 executions in10.222s. Preview policy observed RED before implementation and for missing teardown/no-op transitions, then GREEN; controlled/generated ordering, deep ownership and explicit worker Stop are implemented, integration not yet wired. Four new pure production files are explicitly classified in artifactpath after its exhaustive inventory named them; targeted classifier passes1.970s. No claim of end-to-end submission completion: renderer IO, typed bundle/store, manager/preview projection, CLI wiring and native acceptance remain.

### 2026-10-21 implementation checkpoint — historical evidence and storage

- Implemented one-pass historical sketch query/declared element facts, exact source and selected-history record indices (same-time label/clear regression RED→GREEN), bounded pressure-image plans, injected serial renderer contract/stateful fake, typed sketch evidence, strict nested-extension-preserving v3 codec and one evidence-file inventory.
- Store tests exercise real prepare→begin→ack→complete, mixed note/sketch images, symlink refusal, immutable evidence, unknown huge JSON numbers, and exact-version suppression across targets derived only from durability-confirmed dispatched/resolved manifests. Core race checks passed 14.947s; preserving-codec fuzz passed 26,145 executions. Query/renderer fake package tests passed; renderer seam race passed 1.389s.
- Added a cheap ordered candidate metadata query and board generation fence. Pressure chunks and ordinary page updates do not wake candidate preview; label/clear changes do. Atomic freeze rejects changed generation or forged selected versions and owns its selected slice. Targeted board RED→GREEN passed 0.157s.
- Combined targeted core/query/sketchimage/board checks passed (10.704s/8.184s/0.185s/4.941s respectively). Real renderer/export and console preview integration remain in progress; no full-suite, close or live acceptance claimed yet. ARCH-DRY: rendering, declared-fact extraction, evidence enumeration and oldest-unsent selection each have one reusable authority.
