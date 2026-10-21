---
id: 000754
status: open
deps: [pair#753]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f63f90a00d342d517f5d0f900384653427dacc4f' # card fields mirrored from issue-cards; edit via sdlc
---

# cast-sketch: console to Pencil handoff and echo acceptance

## Problem

Prove cast-sketch’s approved operator meeting and iPad/Apple Pencil use cases and the project’s xianxu.dev-to-echo design handoff on the real setup.

## Spec

Approved design: #750, workshop/plans/000750-cast-sketch-plan.md, especially its 2026-10-20 operator approval revision. This issue implements only its assigned slice. Start from main after dependencies land; write a slice-specific execution plan and pass the SDLC gates. No stacked branches.

Add end-to-end regressions for one driving-derived drawer, streaming partial pressure-aware strokes, own-current-sketch Undo/Redo, label-on-leave, held source scene and historical anchors. Force handoff/disconnect/navigation/resize/scroll/clear orderings through the production seams; include operator console and remote current-driver paths. Coordinate real setup with the TL; never open visible windows or stage :0 without that coordination.

Real acceptance: the operator sketches inside the casting Chrome while a second viewer watches/commentates; the operator grants driving to their iPad/Apple Pencil, which becomes the only drawer; revoke/return to console without old writes crossing the handoff. Verify pointer-link users can still point but cannot draw. Finish labels; replay through scroll/resize and inspect anchors/pressure. Operator exports recording/CLI evidence for an agent in echo’s design process. Record commands and factual evidence in this issue; TL owns project updates.

This acceptance issue owns integrated automation and real acceptance only. Recording implementation belongs to slice A, drawing/input/transport to B, CLI/image implementation to C; reproducible feature defects discovered here return to the responsible scope or an explicitly approved follow-up.

## Done when

- Automated integrated fixture validates the recording, early stroke visibility, pressure fidelity, Undo/Redo timeline, held scene and old-holder write fencing across deterministic interruption orderings.
- On real xianxu.dev cast, console drawing and explicit handoff to iPad/Apple Pencil work; exactly one drawer, observed pressure variation, pointer/view spectators and console return are confirmed.
- Real replay survives scroll/resize/navigation and CLI exports historical image/paths/anchors with required labels or explicit unfinished evidence.
- An echo coding agent receiving operator-provided evidence states the intended change and correct anchor elements before using it to build; record the result and any practical limitation.
- Record browser/device, recording identity, commands, results and private output locations. Required recorded checks/review pass and TL confirms acceptance. Do not claim project done from headless-only evidence.

## Plan

- [ ] Claim/start-plan after dependencies land; write the slice-specific execution plan.
- [ ] Implement and verify the Spec and Done when through the normal SDLC gates.

## Log

### 2026-10-20

- Filed from operator-approved #750 design via ops:0. Left unclaimed for TL dispatch.
