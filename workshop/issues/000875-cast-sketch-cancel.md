---
id: 000875
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '9485d1ff521295689089c7feaedfdc4d2546a558' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast sketch: Cancel beside Done discards the in-progress sketch

## Problem

During the combined operator check, the operator submitted a sketch on the cast tool column over their blog at localhost:4321 labelled “there should be a cancel tool here”. The submission worked end to end. The missing action is next to Done: discard the sketch in progress and leave sketch mode without saving it. This is operator feedback relayed by TL delivery f4c2f489; no private recording or screenshot is copied into this text-only issue.

## Spec

Add a clearly labelled Cancel control beside Done in the live cast browser tool column while a sketch is in progress. Cancel discards that current sketch and exits sketch mode without saving or requiring a label. It must not discard earlier completed sketches or ordinary notes.

Keep the actions distinct: Done follows the existing finish/save and label flow; Undo removes the last stroke within the current sketch and keeps sketch mode available; Cancel abandons the whole current sketch and leaves sketch mode. Cancelling after Undo or Redo must not restore the abandoned sketch through redo or a late in-flight acknowledgment. Preserve recording integrity; “without saving” means no completed/live feedback item or fresh submission entry for the abandoned sketch, not rewriting immutable recording history.

Reuse the existing tool column, sketch lifecycle and command/confirmation ownership. Do not implement Cancel by invoking Done or by clearing every sketch. Settle the precise pending-command interaction in the plan before code; preserve authority and page-load fences.

## Done when

- Cancel appears beside Done for an in-progress sketch and is accessible by pointer and keyboard.
- Cancel removes only the current in-progress sketch from live feedback, leaves sketch mode, and neither saves it nor requests a label; it contributes nothing to a fresh submission.
- Done retains its finish/save behavior. Undo/Redo retain their stroke-level behavior while drawing; Cancel after either still abandons the whole current sketch.
- Earlier completed sketches and ordinary notes survive Cancel. Late command results cannot restore cancelled live feedback.
- Deterministic lifecycle/UI tests cover those distinctions, plus cancellation with a stroke or command in flight; a disposable native check and operator look verify the actual tool column.

## Plan

Unclaimed follow-up. TL assigns pair:4 to take this after #857 lands; claim and plan then, without folding it into #857.

## Log

Filed from operator feedback relayed by ops:0. The cast submission itself succeeded end to end; this is a missing tool action, not a submission failure.
