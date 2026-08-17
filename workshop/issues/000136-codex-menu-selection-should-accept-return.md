---
id: 000136
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours: 0.4
started: 2026-08-16T21:01:01-07:00
---

# Codex menu selection should accept Return

## Problem

Codex selection/permission menus can require Alt+Return to confirm because Pair
does not recognize the current menu footer as a blocking overlay. When
`pickerActive` is not armed, plain Return follows the Codex textarea remap and
sends LF instead of the bare CR Codex expects for menu confirmation.

## Spec

- Detect the current Codex footer string observed in Pair's adapt log:
  `Press enter to confirm or esc to cancel`.
- Preserve the existing overlay bypass behavior: while a picker is active, one
  plain Return sends bare CR, then normal textarea remapping resumes.
- Keep the harness bring-up guide aligned with the regression so future harness
  integrations include visible selection-menu footer strings.
- Non-goals: do not broaden `promptShape` into a generic arming heuristic, do
  not change the one-shot `pickerActive` lifecycle, and do not add a live Codex
  dependency for this marker-drift fix.

## Done when

- Plain Return confirms Codex menus with the current footer; Alt+Return is not
  required.
- A regression test covers the current Codex footer string from the adapt log.
- The harness bring-up guide explicitly covers Codex visible selection-menu
  markers.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.05 impl=0.15
item: atlas-docs design=0.03 impl=0.05
item: milestone-review design=0.02 impl=0.10
total: 0.40
```

## Plan

- [x] Add a failing Codex overlay detector test for `Press enter to confirm or esc to cancel`.
  Unit-test `detectCodexOverlayOpen`: ANSI-wrapped visible Codex footer text
  must match the exact registered marker after terminal controls are stripped.
  Keep the existing `emitPlainCR` one-shot bypass coverage as the guard for
  Return behavior.
- [x] Add the marker to the existing Codex picker detector (ARCH-DRY, ARCH-PURE).
- [x] Update `atlas/how-to-bring-up-a-new-harness-cli.md` with the Codex selection-menu drift note.
- [x] Verify the wrapcmd tests and whitespace.

## Log

### 2026-08-16

- Adapt evidence: `adapt-pair.jsonl` recorded Codex near-misses with
  `Press enter to confirm or esc to cancel`, confirming the detector drifted
  while the telemetry path worked as intended.
- Plan-quality round 1 blocked on explicit test-surface and non-goal wording;
  updated the plan to name `detectCodexOverlayOpen` and rule out broader overlay
  lifecycle or heuristic changes.
- TDD red: `go test ./cmd/internal/wrapcmd -run TestOverlayDetectorByAgent/codex_permission_picker_cancel_footer_opens_overlay -count=1`
  failed with `open = false`.
- Green verification: focused overlay/Return tests and the full `wrapcmd`
  package pass after adding the current Codex footer marker.
