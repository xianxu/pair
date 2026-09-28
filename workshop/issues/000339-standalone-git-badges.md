---
id: 000339
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '217c1cfee2e48e42f66588bb18f99326c900120e' # card fields mirrored from issue-cards; edit via sdlc
---

# Show Git badges for standalone repositories

## Problem

Couch omits Git status badges for standalone repositories. Ducks is clean on
`main` with `branch.ab +1 -0`, but displays `ducks` rather than `ducks+`.
Creating an additional slot should not be necessary to see repository status.

`slotGitProbePaths` only probes numbered slot groups and their primary checkout.
`PresentThreads` additionally suppresses primary badges unless `g.hasSlots`.
`TestPresentThreadsSlotGlyphScope` currently codifies that exclusion.

## Spec

Show Git status badges for standalone repository checkouts in both the switcher
and tab bar, using the same status rules as a slot group's primary checkout.
Include standalone repositories in background status polling. Preserve existing
numbered-slot behavior, refresh timing, and failure handling. Reuse the shared
Git probe and glyph projection (ARCH-DRY).

## Done when

- A standalone repository on `main`, clean and ahead of its upstream, displays
  `+` in both the switcher and tab bar without creating a numbered slot.
- Standalone checkouts support the existing behind, diverged, dirty, and
  off-resting-branch badges; clean synchronized checkouts have no badge and
  missing upstream evidence does not fabricate divergence.
- Background refresh updates standalone badges, including while a child is idle.
- Regression tests cover standalone probe inclusion and presentation, alongside
  unchanged primary and numbered-slot behavior.

## Plan


## Log

### 2026-09-28

- Operator requested this ticket after confirming the slot-only restriction.
  Live Ducks observation: `main`, clean, upstream `origin/main`, one commit ahead.
  Filed for future implementation; no behavior changes made.
