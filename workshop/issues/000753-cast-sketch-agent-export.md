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
- [x] Implement and locally verify the Spec and Done when; submit round1 corrections and exact-source native evidence to full close (recorded-suite and operator landing gates remain explicit).

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

### 2026-10-21 19:51 PDT — preview integration and admitted transport revision

- Manager preview freezes only the acknowledged private publication, with manager→Chrome host fence→Board generation lock order. Controlled manager tests reject unseen/stale/ABA-host selections, cancel and join held publication, and select eight oldest with two remaining. Targeted manager race checks passed 1.764s. One delivery-generation cache prevents repeated durable-store scans on source/host refreshes; only waiting receiver catchup may wake the preview worker after ordinary ink updates.
- Real renderer/native conformance passed: eight-image batch 2.025s including cleanup, cancellation 0.60s; shared browser cleanup has one absolute 5s budget plus at most 1s server shutdown, preserving the 30s preparation/36s client envelope. Existing native fidelity fixtures passed 47.350s. Cast packaging and ordinary annotation ID-context tests passed, with no recording reread during rendering and no sketch clearing on acknowledgment.
- Controlled CDP investigation reproduced active-write cancellation blocking and a queued expired command being written after deadline. Authenticated TL delivery a7274b6d-b64e-40f0-b1db-969c0cc4f4ba approves the narrow transport scope revision below; implementation is in progress, no success claim yet. TL also reports machine-wide github.com DNS failure and explicitly holds all pushes until released. No push/PR/recorded run was attempted under that hold.
- Reader inventory initially named codec_test.go, model_test.go and sketch_test.go. Audited their complete inputs/helpers: in-memory JSON/PNG/record fixtures and own temporary Store only, no repository source reads. Registered only those exact readers as local_or_literal_fixture; make test-affected-inventory then passed. Artifact exhaustive classification passed 1.648s after auditing new files' Couch recording/submission/private export namespaces.

### 2026-10-21 19:58 PDT — transport revision verified; publication hold released

- The admitted CDP repair now serializes writes with context-aware admission, arms/cancels an owned pipe deadline, joins its cancellation callback before deadline reset/token release, and retires partial/short frames. Zero-byte canceled deadline writes preserve the reset pipe; generic owned transports without deadlines explicitly close/retire on active cancellation. Controlled RED→GREEN tests prove active cancellation unblocks, expired/pre-canceled queued commands never emit, and a held old callback cannot reach a successor writer. Existing full-frame late-reply behavior is tested on the real os.Pipe transport used in production.
- Full CDP race passed 1.328s, focused CDP/renderer race passed 1.345s/2.488s. Native renderer passed 6.594s: eight-image batch 2.442s and canceled cleanup 0.71s. Real renderer now also imposes the same single aggregate 30s deadline when invoked from CLI, instead of multiplying per-call deadlines. No timeout increase. This is a protocol-ownership invariant rather than a syntax-only lint rule; the central transport regression covers all callers.
- TL5b0c84f8-b715-4222-ba8b-1e87450bf342 authenticated: operator flushed DNS, push/fetch hold released. Publication can resume once the remaining private-preview capture integration is committed. That integration reuses #752's existing UI-open capture deferral to keep completed preview cards from flashing every five seconds, hides an empty preview so normal captures resume, and adds native/Node regressions.

### 2026-10-21 20:01 PDT — private console preview complete

- Added read-only pressure-ink thumbnails, notes, count and N more through the private console projection. Exact host/generation/revision/operation/stamp/items/cursor acknowledgments fence readiness; scope retirement and manager→host→Board freeze share one no-IO critical section. Negative countermeasurements fail when the retirement guard is removed or pressure wakes are unconditional; final focused projection race passed 1.245s. After Shown, N/2N pressure turns cause zero manager wakes.
- Native Chrome conformance passed 3.325s with two completed sketches, actual thumbnail pressure pixels/notes/count, no browser send button and native label typing/caret preservation across new publication identities, idle time and remote pointer updates. A further RED reproduced periodic capture hiding a watch-mode preview; GREEN4.223s now defers three capture opportunities while cards are visible, retains keyed card/canvas/focus, and resumes capture after an exact empty projection hides the panel. Selected Node110 tests passed0.938s. Parent's broad run overlapped that intentional RED and reported only the expected captureBusy assertion in TestNode; rerun on stable source is required, not a flake retry.
- Mixed-evidence handoff/progress and guides distinguish draw-time crop-N.png from historical replay-rendered sketch-N.png, preserve per-item privacy, and include original URLs/literal IDs. Lua transaction failure matrix and DOM state/adapter/transport/bootstrap checks passed. Shared disposable Claude/Codex fleet verification is running; no live fleet or external model is used by those tests. Real operator chord/preview-and-submit acceptance remains a stage/landing gate.

### 2026-10-21 20:05 PDT — final pre-recorded-check checkpoint

- Stable executable candidate d9bc7aee0d1bab22fd7548e099ff94611fa70dbb is published as PR537 (https://github.com/xianxu/pair/pull/537). Full affected component packages passed: webcast22.318s (including all488 Node tests), couchtty72.336s, couchcmd141.526s, cdp0.263s and submission4.971s. Earlier TestNode failure was the intentional captureBusy RED during development, now repaired and independently rerun GREEN.
- Final stable-candidate native fixture checks passed for Codex84.22s and Claude57.10s: each used four disposable personality actors, actual submitted receipts plus independent accepted-input logs, resize/detach/restart recovery and joined root/process/IO cleanup. Original combined working-tree attempt at f1e292a22 had a Codex startup admission timeout before delivery; retained as diagnostic with different source provenance, not registered or rewritten as a same-head flake. These checks do not claim real model image understanding or replace the operator's mixed sketch-chord smoke.
- Added one production-wiring test from actual Cast.PrepareSubmission through the real durable Store lifecycle and a fresh reopen. Two delivered shapes retain original PNG bytes/dimensions, URL/IDs/note/pressure and exact global suppression; a third merely Prepared shape for another target remains sendable. ACK/Complete leaves the board's ink and candidate metadata unchanged. Focused Cast sketch group race passed2.375s. No new product code since the stable native candidate.
- All implementation is locally checkpointed before the required recorded check. Reader inventory was rerun after the final integration test and passed; no new repair command was needed. Required recorded affected/suite, sole SDLC close review, hub-authorized stage/operator preview-and-submit acceptance and landing remain. TL owns the portfolio/project record, so close must use --no-project and report the outcome to hub.

### 2026-10-21 20:08 PDT — retained recorded suite and close-review entry

- Pushed candidate a7d5ab92d525127d77bc3caddd166d3f58664642 updates PR537. Independent parent verification passed full CDP race1.392s and focused real Cast sketch/store plus private projection race3.217s. Reader inventory passed immediately before close; source classifications and current private UI/transport changes are committed.
- Required recorded affected check selected the full suite because the fresh checkout's runtimebundle embed graph is unavailable (existing generated-runtime graph fallback). It retained run lease-4062627020 on a7d5ab92, exec session91005; checkout /Volumes/TestScratch/local-checks/sdlc-local-3153519690; raw log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/08ac650dff91d43b60f3a4e3d0409e76/run.log. It is queued normal FIFO position5 behind the current Ariadne holder; preserve it, do not start a duplicate. No passing recorded result is claimed.
- Enter the sole SDLC close review while that retained check waits, as tl-worker explicitly permits. Any code fix invalidates the candidate and must receive final-head verification; a docs-only close/evidence descendant may reuse the existing source-equivalent recorded proof. Live operator preview-and-submit acceptance remains before landing, after hub-authorized staging; no real operator smoke is claimed.

### 2026-10-21 — close round1 REWORK and supported native evidence

- SDLC's sole fresh review returned REWORK: BR-1 canceled publications can remain Publishing; BR-2 unselected DOM attributes starve anchor facts; BR-3 region-only notes emit invalid node-zero declarations; BR-4 null numeric evidence becomes fabricated zero; BR-5 inventory paths/PURE classification drifted; BR-6 native conformance failed in the review checkout (GPU unusable / EPERM / CDP EOF). Behavioral findings are being reproduced and fixed by class with regressions; no close is finalized or codecomplete claim made.
- Canceled only this workspace's obsolete queued source run via sdlc local-check cancel --run lease-4062627020; command confirmed cancellation of suite on a7d5ab92 (pid44390). Code fixes require a new pushed-head recorded check; the old FIFO place was not kept by testing obsolete code. No other slot's run was changed.
- BR-5: corrected Core concepts to actual entities/paths and split pure selection from filesystem-backed delivered-version derivation; appended the exact before/after map delta to plan Revisions. BR-1–4 fixes remain in progress.
- Parent supported-environment native fidelity command completed after the CDP repair: WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestLiveFidelity' -count=1 -v PASS63.743s, including five historical fixtures and deliberate broken-render detection. Reviewer's native failure is preserved as environment-specific unverifiability, not erased. Final-source combined native rendering/preview/fidelity will be rerun after the fixes, with retained raw output/hash for BR-6.

### 2026-10-21 — round1 correction checkpoint (BR-1 through BR-5)

- BR-1: refresh cancellation now cancels/joins and feeds the exact publication through PreviewPublishFailed before refreshing identical inputs. A result returned after its original five-second IO deadline cannot become Shown. Deterministic synctest cases cover preloaded/duplicate/catchup/candidate/explicit wakes, terminal channel/context closure, host replacement and late nil results. Removing the fix reproduces the five stuck-Publishing cases plus expired-success false acceptance.
- BR-2: original source snapshots own at most 20,000 node headers and 16 bounded data pairs plus one ID per node; immutable strings are shared, mutable maps are copied. Unselected source attributes spend none of the 64KiB selected-output budget. Selection spends bytes in anchor→ancestor→nearest-context order and reserves minimal facts for all remaining legal mandatory handles. Tests reproduce both starvation and a 64-anchor overflow, retain immutable source facts, and count N/2N growth (64→128 nodes: attributes1344→2688, retained declaration headers1088→2176, allocations337→660).
- BR-3: the shared extractor and ordinary-note context omit absent/nonpositive anchor handles. Region-only Build and draw-time ContextAt remain valid and retain original-region intersections rather than inventing node-zero identity.
- BR-4: the explicit non-reflective wire schema checks scalar types, string dictionaries, array elements and every fixed numeric tuple member before Go decoding. Null/wrong types cannot fabricate zeros. Legitimate pointer/container nulls and PolicyEvidence's legacy union remain compatible; unfamiliar fields/numbers remain raw. More than150 actual scalar/tuple leaves, Store read-path tampering, compatibility cases and fuzz were exercised. Agent scalar fuzz passed193762 executions; extension fuzz passed55532 executions.
- BR-5: current Core concepts names actual symbols/paths and separates pure selection/validation from Store filesystem derivation and owned IO; Revisions records the proposed-to-implemented correction.
- Parent class sweep also reproduced a stale derived delivery cache after retrying a previously visible ACK: successful durability confirmation had not invalidated the clock when the manifest already said Dispatched. Every confirmed ACK now invalidates that derived cache, including idempotent recovery. The controlled retry test was RED with the previous Delivering-only condition, then GREEN; no acknowledgment sidecar or new authority was introduced.
- Independent parent verification passed submission/query/builder full race6.369s/31.193s/1.118s and the complete focused submission-manager race1.747s. No syntax-only lint can establish these protocol/budget/schema invariants without duplicating domain authority; counted, transition and strict-wire regression contracts enforce the classes. Inventory and final-source native evidence precede the next full close (previous REWORK did not finalize a close).

### 2026-10-21 — final-source native evidence and publication recovery

- BR-6 supported-environment verification on clean committed source 85406e6816976252618372a7a9f16eaab8b81a14: `env WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestLiveSketchImage|TestOperatorSketchLiveDrawing|TestLiveFidelity' -count=1 -v` passed77.227s. Fidelity fixtures passed52.12s, deliberate broken-render detection16.17s, console drawing3.41s, historical pressure/batch4.69s and cancellation/join0.56s. Eight historical shapes rendered and cleaned in1.796754583s. The stress fixture retains its documented expected limitation (one of three frames over3%); all fixture assertions passed. This supplies exact-source native evidence without rewriting the reviewer's restricted-environment failure.
- Retained raw output: `/Volumes/TestScratch/tmp/pair-slot9/pair753-pr.b94Wej/native-85406e681.log`; SHA256 `d7942734429e891879c962c39102fc7781eaf7f3c4d9daa39e82440efae7799e`, rechecked before this checkpoint. BR-1–5 corrections and the ACK recovery regression are committed; their disposition still belongs to the next full SDLC close.
- Publication and replacement recorded-check attempts failed at GitHub DNS fetch before a new lease existed. Report b7d4dd5a-f1e5-4192-af81-fb1c7783427e was submitted to hub. Retrying the supported publication path after recovery; no duplicate suite runner, result rewrite, DNS change or gate bypass. Recorded pass, full close, authorized stage/operator preview-and-submit acceptance and landing remain pending.

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

### 2026-10-21 19:51 PDT — CDP cancellation scope revision (approved)

Authenticated TL a7274b6d-b64e-40f0-b1db-969c0cc4f4ba approves folding the discovered transport blocker into #753: context-aware serialized write admission; owned pipe write-deadline cancellation joined before admission is released; partial writes retire the connection; no detached writer or timeout increase. Required regressions: a canceled active call unblocks, an expired queued command is never written, and cancellation callbacks cannot change a successor command's deadline. This supports the already-approved bounded preview/render lifetimes without changing routing, permissions or delivery policy.
