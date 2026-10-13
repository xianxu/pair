---
id: 000538
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'a6f4ac20e92d887f0e07a9555f1e78186ad098d4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T16:08:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# web cast: notes follow their element (hide when it's gone, return when it's back) and crops are captured when the box is drawn (two-phase draft id)

## Problem

Web-cast note boxes can remain on unrelated content after their anchor disappears, and submit-time screenshots no longer show what the operator annotated. Submit also spends most animated-page latency waiting for crops. Notes need durable draw-time evidence and conservative element-following visibility.

## Spec

Web cast only; terminal cast has no DOM and is out of scope. On mouse up, a viewer or the operator overlay sends the drawn box to the cast owner. The server captures the crop and box text from the cast tab together, using #524 bounded settling and #494 UI suppression, stores them locally, and returns a draft ID. Saving sends authored text plus that ID and attaches its stored capture. A small capture tick indicates when the shot completed. Submission bundles stored captures instead of taking new screenshots.

A note follows its anchor. If the element is removed, hidden, on another app tab/collapsed panel/route, hide the box and keep the note open in the thoughts list marked “not on this view”. Show it again when a confident visible match returns, including replacement nodes after SPA rerenders. Score structural path, text, tag/classes and identity evidence; ambiguous or weak matches remain hidden. Never use a stale rectangle or arbitrary first match to draw on another element. Prefer evidence-ranked candidates over refusing solely because node identity changed.

Submit includes ALL open notes in the cast, including hidden notes. Current visibility must not discard their stored evidence. Cancellation deletes a capture draft and crop; an unsaved draft expires on viewer loss, cast end or a bounded timeout and is deleted. Resolving a saved note without submitting deletes its crop. Submission transfers the capture into the durable #483 bundle; later disappearance of the element does not delete it. Define crash-safe ownership so a failed or duplicate send cannot lose evidence or falsely claim delivery.

Capture artifacts remain in the local Couch store before submit. Do not return crop bytes to the remote viewer as part of the draft protocol. At submission, #526 private-by-default rules remain authoritative, evaluated using the stored capture's origin. Keep a few visible states with clear actions; internal capture/storage states must not leak into the user flow.

## Done when

- [ ] Viewer and operator-overlay boxes disappear on hidden/removed anchors and return only on a confident visible match, with ambiguous/reordered/reused-node cases covered.
- [ ] Draw-time capture and text are attached through a two-phase draft ID; capture feedback is visible and stale/cancelled results cannot attach to another note.
- [ ] All open notes, including hidden and earlier-view notes, submit their original captures without screenshot/settling work at submit.
- [ ] Cancel, unsaved expiry, viewer/cast loss, resolve-before-submit and bundle transfer have tested cleanup and crash/retry behavior; disappearing elements retain evidence.
- [ ] Capture files stay local/private before submission, private-by-default submission rules remain intact, and legacy notes have explicit honest fallback behavior.
- [ ] Pure scoring/lifecycle tests, controlled server/recorder tests, viewer/overlay browser conformance, affected checks and full-suite evidence are recorded; docs describe the resulting behavior.

## Plan

Durable design: [anchored notes and draw-time captures](../plans/000538-anchored-notes-draw-capture-plan.md). Awaiting hub approval before change-code.

- [ ] Map existing anchors, note recording, capture ownership, privacy and submission seams; write durable plan and obtain approval.
- [ ] Implement evidence-scored visibility and draw-time capture lifecycle through viewer and operator paths.
- [ ] Bundle stored captures for every open note and verify cleanup, failures, privacy and latency.
- [ ] Review, build, publish and land on SHIP.

## Log

### 2026-10-13

- Received operator design through ops:0; verified submitted Couch receipt 3db8432f-0b02-44e1-92dd-138eff615d47. #538 already claimed by pair:6. No implementation yet. Initial start-plan refreshed this slot and its ariadne tools, then requested rerun with the refreshed binary.
- Initial inspection: Board.Selection uses model.Live over current DocSeg; submission.Build enforces one selected document and asks the cast owner for new crops. Structural.Resolve accepts a unique selector even without matching text. These seams must change so note evidence survives view changes while projection remains conservative.

- Durable plan completed: shared cast-owned draft service, capture outside the recorder actor, conservative shared anchor fixtures, all-open selection across documents, private storage and crash-safe transfer into submission bundles. Proposed bounds: ten-minute unsaved TTL, one capture worker/eight queued jobs per cast, 256 MiB/128 local capture records; attached evidence is never evicted to make room. ARCH-PURE isolates ranking/transitions; ARCH-DRY gives viewer/operator one lifecycle owner.
- Code exploration found ghost boxes from unconditional note.region/historical at.box fallback and operator boxOf returning null. Current operator prepare creates a token without capturing; calling CaptureCrops inside handleOperator would deadlock its event barriers. Board selection/ack and submission v2 currently assume one document; all require explicit changes. No implementation or test run for #538 yet.
- Coordinated with pair:7 (#537), verified receipt 35db0042-3926-4028-8377-29f59bcc0b67: preserve current-note pin callbacks, author ownership checks, edit-note protocol, and immutable original capture on text edit. They preserve prepare/save/cancel. Reconcile from main before final checks.

- Hub approved plan fbbbcbbf with explicit projection for queue/store bounds: show capture-busy or storage-full reason at mouseup, primary action Save note remains available, and persist honest unavailable evidence without crop bytes. Added authoritative Revisions entry and tests; proceeding to change-code.
