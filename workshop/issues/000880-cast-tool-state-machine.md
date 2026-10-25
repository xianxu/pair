---
id: 000880
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '7ba78dce60dac8dfdb61e32d559a91ff9825561d' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast tool column: make tool and unfinished-label transitions consistent

## Problem

During the evening operator check of combined stage 9bfd7ab5, clicking Pointer, Pen and Note in some sequence left Pen's label form open while Note was also active. Screenshot: ops session.side/assets/2026-10-24-17-evening-combined-check/18-47-51 (local evidence; no screenshot copied). The exact sequence is not yet known.

The operator says: "if something is clickable, then user should be able to click."

## Spec

Audit the cast tool column as a state machine, including Pointer, Pen, Note, the unfinished sketch label form, Done, Cancel where available, Undo and Redo. Every clickable control must either perform its defined transition or show a plain reason why it cannot. Controls unavailable in the current state must be visibly disabled. Avoid hiding controls that then reappear. Preserve unfinished feedback and existing recording, annotation authority and page-load boundaries.

## Done when

- Reproduce or reconstruct the conflicting Pen-label-plus-Note state and pin it with a regression.
- Document reachable tool/editor combinations and explicit transitions, including cancellation and refused transitions.
- Pointer/Pen/Note sequences cannot leave conflicting active tools or a stranded label form.
- Every visible enabled control transitions or explains a refusal plainly; unavailable controls are disabled, with stable placement.
- Tests cover transition ordering and unfinished-label handoff; the operator checks the corrected column.

## Plan

- [ ]

## Log

2026-10-24: Filed unclaimed at TL db1a5886 request. Separate from #857's Notes flashing, Show on page placement and internal-jargon failures. Root cause not yet established.
