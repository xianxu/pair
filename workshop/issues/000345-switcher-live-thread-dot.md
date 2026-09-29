---
id: 000345
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '44d8b73e09ad2159c26212b9d16dcd277b4ca6ec' # card fields mirrored from issue-cards; edit via sdlc
---

# Switcher: prefix live threads with a green bullet

## Problem

The switcher currently relies on whole-row liveness color, making live threads hard to distinguish from parked threads. The operator wants an explicit marker at the start of each live thread row.

## Spec

- Prefix live thread rows in the switcher with a green `•` followed by a space. Parked rows have no live bullet; reserve equivalent spacing if needed to keep labels aligned.
- Derive the marker from the existing authoritative thread-liveness classification (ARCH-DRY). Do not add a second liveness probe or infer liveness from row color, selection, or recent output.
- A live but idle thread keeps its bullet. The marker updates when the switcher observes a live/parked state transition; unknown or unproved liveness must not receive a positive live marker.
- Style the bullet independently of whole-row idle fading and selection emphasis so it remains recognizable on selected and unselected rows. Preserve existing labels, row colors, navigation, search, clipping and hit targets.
- The bullet is presentation only, not part of the thread name or identity. In color-disabled output, retain the bullet without ANSI styling.
- Scope is the switcher only. Related #247 controls idle shading; #342 specifies a trailing green period for recent terminal output in the switcher and status bar. This leading live bullet has a different meaning and must coexist without conflating the predicates.

## Done when

- Live rows start with a green `•`; parked and unproved-live rows do not, including when whole-row colors are similar.
- Live idle threads retain the bullet regardless of #342's recent-output threshold.
- Rendering tests cover live/parked/unknown states, selected/unselected rows, idle fading, color-disabled output, and transitions in both directions.
- A switcher smoke check verifies visible distinction, alignment, clipping, selection and unchanged search/navigation behavior.

## Plan

- [ ] Locate the switcher's existing liveness classification and row-decoration path; coordinate with #247 and #342.
- [ ] Add the leading independently styled live bullet and focused rendering/transition regression coverage.
- [ ] Verify the switcher visually and update its relevant UI documentation.

## Log

### 2026-09-29

Filed at the operator's request: add a green `•` at the start of live switcher rows because whole-line liveness coloring is difficult to distinguish from parked rows. Ticket creation only; implementation has not started.
