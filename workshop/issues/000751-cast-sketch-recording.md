---
id: 000751
status: open
deps: [pair#750]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd0cabc7f95b9419eb5dcacafb00780fe8ef1bdb0' # card fields mirrored from issue-cards; edit via sdlc
---

# cast-sketch: recording model, pressure paths and undo history

## Problem

Implement the approved cast-sketch recording model before its drawing consumers. Existing pointer trails must retain their current fading semantics; sketches are persistent, undoable design evidence.

## Spec

Approved design: #750, workshop/plans/000750-cast-sketch-plan.md, especially its 2026-10-20 operator approval revision. This issue implements only its assigned slice. Start from main after dependencies land; write a slice-specific execution plan and pass the SDLC gates. No stacked branches.

Add additive version-1 sketch events in events.jsonl, pure Go/JS reducers, bounded idempotent chunk history, Undo/Redo stacks, labels and interruption semantics. Points retain original top-page CSS x/y plus normalized pen pressure; mouse/touch have constant-width fallback. One recorded source moment, one element anchor per stroke, fixed pointer colour per sketch. Reuse existing anchor confidence/projection and shared fixtures.

Derive driving handoff interruption from the existing ordered ctl event, before exposing its new holder. Separate sketch-interrupt events represent standalone expiry. Preserve historical paths, partial/truncated strokes, metadata-only label completion and annotation-clear barriers across replay and catch-up. Source-moment fencing must work at equal timestamps across split files. New sketch records preserve nested unknown fields; old readers safely ignore the new event kinds. Enforce writer growth limits with terminal/required-label capacity reserved before drawing admission.

Primary surfaces: cmd/internal/webcast/recording/{sketch.go,sketch_state.go,record.go,codec.go,reduce.go,catchup.go}, sketch/geometry.go, web/{sketch-model.js,model.js,replay.js}; tests and atlas/webcast-format.md.

## Done when

- Go and JS shared corpus agrees at every prefix for chunks, duplicates/conflicts, pressure, Undo/Redo/new branches, labels, interruptions, driving handoff, navigation, pause/resume and clear.
- Original page paths and pressure round-trip; shared geometry/pressure fixtures pin live/replay/export semantics.
- Old-reader whole-recording validation and new nested-unknown preservation tests pass without changing pointer-stroke meaning.
- Equal-timestamp source fencing and writer finalization-reserve tests pass; recording/catch-up/history bounds are enforced.
- Atlas describes the format and compatibility limits. SDLC review/checks pass.

## Plan

- [ ] Claim/start-plan after dependencies land; write the slice-specific execution plan.
- [ ] Implement and verify the Spec and Done when through the normal SDLC gates.

## Log

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.
