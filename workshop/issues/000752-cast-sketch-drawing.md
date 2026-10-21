---
id: 000752
status: working
deps: [pair#751]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 10.63
collaboration-mode: collaboration
card_mirror: 'b4ba8913f171b9edd57756de0aed951a6611aab8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T00:30:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# cast-sketch: live console and driver drawing with pen pressure

## Problem

Deliver the approved live drawing tool on both required surfaces: operator inside the casting Chrome and the currently granted drive-link viewer, including iPad/Apple Pencil.

## Spec

Approved design: #750, workshop/plans/000750-cast-sketch-plan.md, especially its 2026-10-20 operator approval revision. This issue implements only its assigned slice. Start from main after dependencies land; write a slice-specific execution plan and pass the SDLC gates. No stacked branches.

Drawing follows driving, exactly one drawer cast-wide: console user when control is local; current granted drive-link holder when remote. A drive link without the current seat grants no drawing. Pointer-link and view-only viewers watch; pointer-link users retain pointing. Use current driving ownership/generation as authority, not an independent multi-drawer grant system or attribution ID.

Implement acknowledged begin/chunk/end/Undo/Redo/label/status operations, exact retries, writer admission checks and a held source-moment scene on BOTH casting Chrome and remote drawer. Label is required on leaving the tool, Undo/Redo affect the current sketch only, colour is fixed per sketch, one anchor per stroke. Old-author metadata-only label finalization may survive a handoff but never grants drawing or blocks the new holder. No takeover after expired link/session.

Casting Chrome controls use the privileged excluded isolated-world host and recorder binding, with bounded private snapshot/asset transport; no page script receives credentials/bindings. Reuse the shared sketch UI/renderer and existing snapshot/asset seams; this slice does not depend on the later headless CLI exporter.

Support pen Pointer Events and normalized pressure, coalesced samples, pointer capture, hover exclusion, release/cancel handling and deterministic mouse/touch fallback. Stream partial strokes to observers before pointer-up. Grant/revoke/release/leave/reset must seal prior accepted work and fence old-generation queued writes before enabling the next drawer. Pause/end/reconnect/expiry, backpressure, limits and failure UI follow the approved state model.

Primary surfaces: board/{sketch.go,sketch_authority.go,sketch_wire.go,writer.go,operator.go,operator_view.go}, recorder/{sketch.go,operator.go}, server_sketch.go, cast.go, browser.go, operator.js, operator_script.go, web/{sketch.js,sketch-policy.js,sketch-scene.js,sketch.css,viewer.js,annotate.js,annotation-placement.js}.

## Done when

- Two-viewer test observes first incomplete stroke before drawer pointer-up; Undo/Redo change everyone’s overlay and preserve replay history.
- Real casting-Chrome operator can draw in its held scene, label on leave and Undo/Redo; currently granted drive-link drawer can do the same.
- Pointer/view links and ungranted drive links cannot draw; pointer-link pointing still works.
- Held-write tests force every driving handoff across admission/queue/sink/reply and prove one drawer, old-generation refusal, retained partial evidence and independent pending-label completion.
- Pen pressure survives live transport; coalesced samples, hover/cancel/release and mouse/touch fallbacks are covered. Budget, source synchronization, exact retries and N/2N counters pass.
- README/atlas map the controls and recovery. Required checks/review pass, and TL-arranged real console input smoke succeeds before landing.

## Plan

- [x] Claim/start-plan after dependencies land; write the slice-specific execution plan at [000752-cast-sketch-drawing-plan.md](../plans/000752-cast-sketch-drawing-plan.md).
- [x] Implement both surfaces and verify protocol, pressure, recovery, privacy and placement through automated checks.
- [x] Complete automated implementation verification and the recorded check; run the close review at the acceptance gate. TL console smoke remains a separate pre-landing hold.

## Log

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.

### 2026-10-21 — execution design

- #751 landed as PR505 in train216/PR508, with branch full suite10m7s and integrated full suite11m57s; final CI passed. Its post-land local cleanup hit known ariadne#512. Per TL instruction preserved the landed branch, added evidence to that issue, returned to main-slot9 and fast-forwarded before claiming/starting this issue.
- Using writing-plans with read-only bounded browser/protocol/producer exploration. ARCH-ORDER: extend existing driving owner with a sketch-only revocation witness/epoch to fence A→B→A and release-in-progress; actor rechecks it without taking controller locks. ARCH-DRY: capture exact prospective source from Hub on the writer, reuse shared model/placement/tool, and deliver small operator render patches rather than copying full history per chunk.
- Draft plan records source/queue/delivery bounds, state/recovery mapping, generated/held-interleaving tests and required real console smoke. Checking actual console frame isolation before approval. No implementation or estimate yet.
- Disposable Chrome156 privacy probe found a named same-origin held iframe inside a closed shadow root can be read by page `window.open` targeting its known name, despite being absent from `window.frames`. Top isolated worlds (including universal access) cannot simply render an opaque-origin sandbox frame. Qualifying a concrete correction before approval; probe artifacts are private scratch, the durable result is recorded here and in plan Revisions.
- Qualified the actual shared Model/Renderer inside an opaque data shell's scoped child isolated world: nested frames, recorded name/CSS and private blob image rendered; parent named targeting was denied with SecurityError and binding/renderer globals stayed private. Chose this origin boundary over stripping recorded names, which changes source layout. Parent keeps the tool/label draft; child exposes only frozen rendering and read-only hit tests through trusted CDP. Plan also adds optional stroke-end truncation, needed to record pointercancel while continuing the same sketch.
- ops:0 approved the full execution plan (delivery96710fe7-a135-4b67-81e3-9189c71f5d63), including opaque scene and named-target regression. TL will do console smoke when staged; requested atlas invariant that the casting page never reaches the recorded scene, bindings or assets. Proceeding to plan-quality, then estimate, before implementation.


### 2026-10-21 — first implementation checkpoint

- Added strict ingress/replies, small owned reducer queries and additive truncated stroke-end semantics with Go/JS corpus coverage. Added a lock-free driving-turn witness and separate seat-lifetime metadata witness; A→B→A, pending-only transitions, held control/release and Close are covered.
- Frozen source preflight owns the exact fenced begin prefix and reduced scene, refuses independent 16 MiB prefix / 8 MiB state budgets without mutation, survives navigation compaction, and indexes only source-bound asset hashes. ARCH-ORDER/ARCH-CONSTRAINTS shaped this seam; node/sheet accounting visits each once.
- Verification returned green: focused recording/strictjson tests, race-enabled driving tests, and 87 Node sketch model/policy/render/delivery tests. Source received an independent review with no concrete ownership/order/accounting violation found. Shared Board/recorder/routes and actual UI integration are still in progress; this checkpoint does not claim end-to-end completion.
- Added the TL-required atlas invariant. Mandatory named-target Chrome regression and TL console smoke remain outstanding before landing.

### 2026-10-21 — ordered service and private delivery checkpoint

- Implemented Board ingress/origin receipts, original-owner late labels, a single frozen-source lease with one reusable cancelable timer, acknowledged reference/recorder writes, healthy observation barriers, scoped source/asset reads and strict drive-seat HTTP routes. Unknown persistence remains unknown. Added compact bounded operator journal/bootstrap with independent receiver verification.
- Fresh service review found epoch/sketch lookup collisions, lease deadline/held-reply ordering, last-refusal retention and late unknown-source resurrection gaps; fixed with focused regressions. Actor acceptance now renews the lease and ends the global begin reservation, independently of transport replies; old-label work stays separate. Sketch chunks bypass unrelated historical note projection scans.
- Returned verification: all Board, drive and recorder package tests pass; focused writer/service/journal race checks pass. HTTP race coverage includes pre-body role/seat/epoch refusal, pressure streaming, replay, old-author label during a new sketch, scoped reads and a held asset read retired by label/expiry. Cast fallback reuses the real Hub or adds one private source hub for absent/custom live factories.
- The mandatory actual Chrome privacy regression passed (unchanged shared Renderer, private SVG2×3, recorded nested frame name/CSS127 border-box, named-target SecurityError and private blob fetch TypeError). Console UI/observer exclusion integration, two-viewer live streaming and TL console smoke are still pending. The scene's asset transfer envelope is aligned to the existing recorder fallback capture32 MiB, separate from the16 MiB source prefix.

### 2026-10-21 — both surfaces and live proof

- Shared pressure tool, FIFO recovery, required label, Undo/Redo and incremental painter now run on both viewer and isolated console. Drawing suspends page drive input. Render-only journal delivery and anchor indexes avoid full path/point copies on normal samples; empty/undone anchors retain trusted current-document original-node placement, including common Save controls without an open note. Identical replacements receive no historical identity.
- Opaque child scene owns its renderer, bounded source/assets and CDP context lifetime. Recorder frame admission/exclusion precedes observer setup; exact private blob URL exclusion preserves ordinary sibling fonts. Trusted parent-context retirement fences late requests, including numeric context-ID reuse.
- Fresh actual Chrome run: TestOperatorSketchLiveScenePrivacy PASS2.10s, TestOperatorSketchLiveDrawing PASS1.57s, TestSketchLiveTwoViewers PASS2.77s (package6.597s). The tests exercised private DOM/named-frame/blob denial, actual StartCast console pen input, three pressure samples durable before pointer-up, single echo, Undo/Redo, required label and replay agreement. Main-world private hooks stayed unavailable. These owned fixtures do not replace TL console smoke.
- Focused fresh review found a coalesced clear→same-anchor bootstrap gap: the browser could prune its original proof while Go retained a bound acknowledgment. Ready anchor revisions now invalidate old/new proof attempts; incomplete journal delivery defers sketch-only retries. The failing regression now passes with the original-binding race suite (1.329s).
- Final consistency regression exposed durable width3 versus optimistic width4; the service now persists4 and its focused test fails before the fix and passes after it. Other targeted Go packages and Node/assembly checks passed; broader checks are being finalized. Audited18 production-file artifact dispositions and individually audited source-reader inputs; no new Pair artifact family.
- Additional actual StartCast regression PASS2.371s: completed ink remains on a common Save button with no ID and no note, follows that exact button by +200/+80 after leaving the held scene, and disappears for an identical replacement across worker retry turns. No fabricated binding or new production test hook. Shared Node tests214/214, original/assembly checks, targeted race checks, local make build, artifact classification and reader inventory pass.
- Staging is explicitly held by ops:0 delivery d053ca5e-cb78-44ee-b89c-d3db95f99841 until the coordinated Couch restart preserves the current main build. Ready-state report will precede staging, and TL go plus real console smoke remain mandatory before landing.

### 2026-10-21 — pre-smoke acceptance order

- PR512 is open on pushed implementation10e2d7b59465d1f955becbc6ceb8184e03a6ab3c; local make build returned exit0. Recorded affected selected the full suite because its fresh checkout lacks the generated runtime embed graph (known #738/ariadne#517 fallback); retaining its runner.
- ops:0 delivery b1d858cd-0dfc-445b-a623-c3e78b62a282 explicitly requests recorded check and close review now, with console smoke the sole remaining acceptance step afterward. Staging remains held until the operator restart; no landing before smoke and TL go. Close verification will identify this pending live acceptance rather than claim it happened.

### 2026-10-21 — recorded full suite

- Retained local-check runner acquired capacity after366s and completed the full fallback suite exit0 in11m11s. Recorded sdlc-local/suite on10e2d7b59465d1f955becbc6ceb8184e03a6ab3c; log SHA256 prefix c157a785c710. All Go packages and shell/editor checks passed. Later commits only document the acceptance hold, so this evidence covers unchanged implementation.
- Entering mandatory close review under the TL-directed pre-smoke order. Console smoke remains unperformed; stage and merge remain held until restart admission and that acceptance.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Derived after plan-quality round1 passed without blockers. Calibration is tagged stale by `sdlc estimate-source`, so this is provisional. Single repo; includes both UI surfaces, owned/live smoke, documentation and final review. Browser pure modules/screens use the nearest greenfield-module/TUI-screen primitives; this is an explicit analogy, not a new vocabulary item.

Each row below corresponds in order to an item in the block. Design uses ×0.2 for the approved concrete plan. Library availability: standard JSON/crypto/context/HTTP and existing NextSketch, Hub, controller, Writer and Model/Renderer cover the greenfield foundations, so those raw greenfield design picks are halved before the spec discount. API integration design picks are 1–1.5 hours before the spec discount. All implementation values are already ×0.40 of the v2/v2.1 table; familiarity1.0 because this extends the qualified existing stack. One external API discovery budget covers Chrome/CDP. No separate queue-wait estimate or double implementation scaling.

| Component | Design h | Scaled impl h |
| --- | --- | --- |
| slice specification/approved architecture | 0.20 | 0.08 |
| strict command union/digest | 0.10 | 0.20 |
| receipt queries and additive cancellation fact | 0.04 | 0.16 |
| drawing witness | 0.15 | 0.28 |
| source preflight | 0.15 | 0.28 |
| Board lease/ownership | 0.20 | 0.32 |
| settled recorder writer/barrier | 0.30 | 0.60 |
| HTTP owner routes | 0.20 | 0.40 |
| private source/asset transfer | 0.30 | 0.60 |
| operator journal/bootstrap | 0.15 | 0.28 |
| render patch projection/painter | 0.10 | 0.24 |
| pointer interaction/recovery policy (browser screen analogy) | 0.20 | 0.40 |
| shared controls/label coordinator (browser screen analogy) | 0.20 | 0.40 |
| held scene integration | 0.30 | 0.60 |
| opaque Chrome child scene | 0.30 | 0.60 |
| placement source identity | 0.04 | 0.16 |
| module assembly/subscriptions | 0.06 | 0.16 |
| two-viewer/console acceptance integration | 0.20 | 0.60 |
| Chrome CDP isolation discovery | 0.00 | 0.24 |
| controls/invariant documentation | 0.03 | 0.08 |
| one final boundary | 0.04 | 0.20 |

Design subtotal3.26h ×1.15 thorough-plan buffer + implementation6.88h ×1.0 =10.629h, rounded to **10.63h**.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.2 impl=0.08
item: greenfield-go-module design=0.1 impl=0.2
item: smaller-go-module design=0.04 impl=0.16
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.3 impl=0.6
item: api-integration design=0.2 impl=0.4
item: api-integration design=0.3 impl=0.6
item: greenfield-go-module design=0.15 impl=0.28
item: greenfield-go-module design=0.1 impl=0.24
item: tui-screen design=0.2 impl=0.4
item: tui-screen design=0.2 impl=0.4
item: api-integration design=0.3 impl=0.6
item: api-integration design=0.3 impl=0.6
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.06 impl=0.16
item: api-integration design=0.2 impl=0.6
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 10.63
```


## Revisions

### 2026-10-21 — TL-directed pre-smoke close review

Implementation and automated verification can close locally before the deferred real console smoke, per ops:0 delivery b1d858cd-0dfc-445b-a623-c3e78b62a282. The Done-when requirement for smoke before landing is unchanged. The operator restart owns staging admission; stage and landing remain held.
