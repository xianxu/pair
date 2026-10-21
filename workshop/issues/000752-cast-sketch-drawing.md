---
id: 000752
status: working
deps: [pair#751]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'ed3d001d26a410dd0641da6fed0f994c30a7ae36' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T00:30:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
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
- [ ] Implement and verify the Spec and Done when through the normal SDLC gates.

## Log

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.

### 2026-10-21 — execution design

- #751 landed as PR505 in train216/PR508, with branch full suite10m7s and integrated full suite11m57s; final CI passed. Its post-land local cleanup hit known ariadne#512. Per TL instruction preserved the landed branch, added evidence to that issue, returned to main-slot9 and fast-forwarded before claiming/starting this issue.
- Using writing-plans with read-only bounded browser/protocol/producer exploration. ARCH-ORDER: extend existing driving owner with a sketch-only revocation witness/epoch to fence A→B→A and release-in-progress; actor rechecks it without taking controller locks. ARCH-DRY: capture exact prospective source from Hub on the writer, reuse shared model/placement/tool, and deliver small operator render patches rather than copying full history per chunk.
- Draft plan records source/queue/delivery bounds, state/recovery mapping, generated/held-interleaving tests and required real console smoke. Checking actual console frame isolation before approval. No implementation or estimate yet.
- Disposable Chrome156 privacy probe found a named same-origin held iframe inside a closed shadow root can be read by page `window.open` targeting its known name, despite being absent from `window.frames`. Top isolated worlds (including universal access) cannot simply render an opaque-origin sandbox frame. Qualifying a concrete correction before approval; probe artifacts are private scratch, the durable result is recorded here and in plan Revisions.
- Qualified the actual shared Model/Renderer inside an opaque data shell's scoped child isolated world: nested frames, recorded name/CSS and private blob image rendered; parent named targeting was denied with SecurityError and binding/renderer globals stayed private. Chose this origin boundary over stripping recorded names, which changes source layout. Parent keeps the tool/label draft; child exposes only frozen rendering and read-only hit tests through trusted CDP. Plan also adds optional stroke-end truncation, needed to record pointercancel while continuing the same sketch.
