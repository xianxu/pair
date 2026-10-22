---
id: 000786
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'cabf6ae3420cdd911ac9b6808c6dff8a698aaf02' # card fields mirrored from issue-cards; edit via sdlc
---

# Expose picker state and prompt in Couch actors JSON

## Problem

ariadne#533 needs to distinguish a slot blocked behind a picker from an idle agent. Couch already receives the wrapper's picker detection as registry SlotLiveness.Picker, but couch --actors --json does not expose it. Its Resting field means resting Git branch, not agent prompt readiness.

## Spec

Expose the existing registry observation in each Actors row, without new screen detection in consumers:

- Picker: JSON boolean true/false when the connected wrapper's SlotLiveness.Picker is known; null or absent when unavailable (including old wrappers). Never default unknown to false.
- PickerPrompt: JSON string containing only the recognized prompt's first line when Picker=true; empty or absent if unavailable or Picker is false/unknown. Reuse Couch's existing picker interpretation; do not run a new terminal parser or expensive probe per actor. Bound the line using the existing picker-summary limits, and strip terminal control sequences. No full command, options or terminal transcript is required.
- Bind the observations to the same current wrapper incarnation as Binding. A disconnected/replaced wrapper cannot lend its picker state to its successor. Keep Resting and Known meanings unchanged; addition is read-only, additive and compatible with old JSON consumers.

The consumer renders agent·picker with the optional prompt line; only an explicit Picker=false establishes absence of a picker. Resting alone will no longer be called idle. This issue provides the producer contract; ariadne#533 owns the consumer.

## Done when

- Actors JSON exposes the tri-state Picker and bounded PickerPrompt for the matching connected wrapper using Couch's current detection.
- Regression fixtures cover true, false, unknown/old wrapper, picker closure and wrapper replacement, across Claude/Codex and Qoder when available. Include the real capture fixture required by the fleet rule; use the shared disposable Couch harness for live acceptance, never the live fleet.
- Existing actor listing remains a bounded in-memory observation, and older consumers remain compatible.

## Plan

- [ ] Project the existing registry picker observation into actor output and document the contract.
- [ ] Verify observation lifetime and replay captured agent fixtures through the shared harness.

## Log

### 2026-10-21
Filed unclaimed on ops:0's instruction while working ariadne#533. Consumer is blocked on this additive producer field; no Pair checkout changes requested or made by the filing worker.
