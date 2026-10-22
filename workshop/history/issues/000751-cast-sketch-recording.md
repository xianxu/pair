---
id: 000751
status: done
deps: [pair#750]
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 3.79
collaboration-mode: collaboration
card_mirror: '2cc02a88c3c1e541d80e126bf6d6639b9f939862' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T21:40:18-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.04
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

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is flagged stale and provisional. Four single-concern modules cover wire preservation, sketch history, browser port and byte budget; existing strictjson, geometry and producer patterns avoid library discovery. Thorough approved design applies x0.2 design discount, implementation uses the calibrated 40% factor and familiar Go/JS uses familiarity 1.0. Two producer integrations plus catch-up snapshots are grouped cross-cutting work. One close boundary includes review; docs and verification are included, suite queue idle is excluded. Design subtotal 1.24, implementation 2.36, design buffer 15%, total 3.786 rounded to 3.79 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.80 impl=1.28
item: smaller-go-module design=0.04 impl=0.20
item: cross-cutting-refactor design=0.32 impl=0.60
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 3.79
```

## Plan

- [x] Claim/start-plan after dependencies land; write the slice-specific execution plan.

Execution plan: [000751-cast-sketch-recording-plan.md](../plans/000751-cast-sketch-recording-plan.md).
- [x] Implement and verify the Spec and Done when through the normal SDLC gates.

## Log

### 2026-10-20
- 2026-10-20: closed — Feature Go packages and webcast TestNode/CatchUpCorpus/OperatorScript PASS; browser prefix/snapshot/privacy-of-authority tests PASS including real signed-zero history accounting; owned-Unicode fuzz PASS; full repolint/artifactpath and affected-reader inventory PASS. BR-1 canonical numeric class and BR-2 production history/control encapsulation fixed with red-to-green regressions; wire bytes remain exact. Recorded full suite on corrected pushed head queued after known pair#738 fallback; tl-worker permits review overlap, recorded pass required before landing. ops:0 owns project updates.; review verdict: SHIP

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.

- #750 landed as PR #501; claimed and started #751 from main. ops:0 approved deriving terminal reserve from worst-case encoded label plus envelope and end/interruption, rounding up to at least 8 KiB while preserving the 32 MiB total cap. Plan records the correction and adversarial regression; no further operator round required.
- Read-only implementation reconnaissance identified existing strictjson preservation helpers, both ordered producers, causal ctl catch-up history and operator module-export constraints; plan covers each without adding UI/transport scope.

- Resume: plan-quality round 1 raised bounded-history and test-strategy findings; ops:0 directed reducer-derived bootstrap snapshots (delivery 432c999b-f034-4fe0-84f3-f36982901ab4). Governing revision committed; round 2 CLEAN with both findings addressed. Derived estimate only after that approval; implementation has not begun.

- Implemented all eight preserving event codecs and frozen legacy-registry compatibility; pure Go/JS history owns operation identity, pressure paths, constant-time Undo/Redo and contextual interruption. Both ordered producers use shared source fences and exact 32 MiB byte admission with independently measured 8/4/4 KiB label/end/interrupt reservations.
- ops:0 snapshot direction is implemented through ephemeral catch-up-only sketch/control frames; repeated Hub compaction and replay seek restore the same validated history. Raw disk readers and ordered writers cannot install/persist bootstrap frames. Snapshot receipt verification reuses the transition core, with contextual interruption/seal facts, and rejects cross-field identity/digest/path/stack inconsistencies before mutation (ARCH-DRY/ORDER/SECURE).
- Ad-hoc fresh-eyes review found and fixed post-end labels, uncertain sink retries, unknown-field byte-accounting drift and bootstrap cross-field inconsistencies. Producer budget charges preserved wire bytes; the shared model counts known-field encoding, keeping additive unknown values outside authoritative projection. Exact retries retain unknown persistence outcomes instead of inventing a durable acknowledgment.
- Targeted verification PASS: all recording/sketch/board/recorder package tests; webcast TestNode/OperatorScript; focused browser model/replay tests; artifact-source classification; bounded codec fuzz run. Shared corpus pins every prefix including driver reset, pause/resume, navigation, clear-open, end and nested unknown fields. Audited the new corpus reader: its only repository input is tests/webcast/sketch-corpus.json; recorded non_go_full_fallback inventory entry. Atlas maps final contracts. Recorded full/affected check and close remain next.

- Published PR #505. Recorded affected check on code head 4ff8fe47eed59303bb9b3b82f4fc74786f23d189 selected the full suite because the detached package graph lacks generated runtime assets (known pair#738/ariadne#517). Full suite is queued under the machine-wide FIFO; no bypass or unrelated fix. Targeted final checks and repolint pass. Boundary review may overlap the queued recorded suite under tl-worker; landing still requires its recorded pass. ops:0 owns the external project update.

- Final integration probe reproduced sketch + driven Click catch-up failing because all ctl records were removed. ops:0 approved preserving exactly the existing one pre-segment control baseline plus contiguous current-segment controls (delivery 5a5156ef-74c4-4782-b77c-9d1b01af1a03); only sketch history is replaced. Driven DOM and repeated-segment regression added in both languages. Canceled only this checkout's old queued suite and the in-flight old-head review gracefully; no review verdict was admitted. Rerunning final recorded checks and close on corrected code.

- Close round 1 returned REWORK: BR-1 signed-zero cross-language byte accounting, BR-2 browser authority encapsulation. Added red-to-green real signed-zero tests/corpus and private Model state with owned-copy mutation regressions; original wire bytes remain exact. An additional codec probe found lone-surrogate normalization drift; owned Unicode validation now rejects malformed strings before Go replacement while preserving unknown raw future fields. Same-head old check was canceled via local-check cancel; final targets/review/check are rerun after these fixes.
