---
id: 000521
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'dbb6e478f6afc6be3b9c744aa5414c8450ff9669' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T21:51:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "49889759", done: "8024e07e"}
---

# couch: draw the hub's tab in inverted colours, like a bookmark

## Problem

Couch's tab bar leads with the hub (pair#491), but it looks like every other tab, so the eye has to read its label to find it. The operator (2026-10-12) wants the hub marked like a bookmark.

## Spec

Draw the hub's tab in inverted (reverse-video) colours of its normal style, so it stands out as a bookmark wherever the bar is. The other state cues (red, bell, focused) stay readable on it: decide how each combines with the inversion and show them in the issue (a small table), then the operator approves on a live look. Same treatment for the hub's row in the switcher, if it reads well. No new keys.

### Approved composition

Apply SGR 7 after each fragment's existing colour. Reverse label, glyph and
focus brackets together; gaps, separators and +N remain normal.

| Red | Bell | Focused | Hub tab |
| --- | --- | --- | --- |
| no | no | no | Normal/idle colours reversed |
| no | yes | no | Amber reversed |
| yes | either | no | Red reversed |
| no | either | yes | Normal reversed, with focus brackets |
| yes | either | yes | Red reversed, with focus brackets |

Bell is the existing amber colour cue, not a separate glyph; red and focus
keep their existing precedence over it. Slot glyphs retain their own colours,
also reversed. Placeholders stay grey and unclickable, also reversed.
The switcher reverses the hub's main row; a selected hub keeps its arrow and
adds bold. Red remains visible. Attention detail lines keep current styling.
Use native reverse video rather than manually swapping RGB palette values.

## Done when

- The hub's tab is inverted in the tab bar at every width (including when other tabs fold into +N); red, belled and focused hub tabs stay distinguishable; tests pin the rendering; the operator approves the look.

## Plan

- [x] Pin tab state combinations, widths/folding, glyphs and style containment with failing tests.
- [x] Compose hub inversion in tab and switcher renderers; update presentation docs.
- [ ] Run targeted tests and full suite; stage only on TL instruction for operator approval, then close/PR/SHIP.

## Log

### 2026-10-12

- Claimed and started in pair:7; TL approved the composition table and implementation via Couch.
- ARCH-DRY/PURE: reuse existing Hub projection, colour precedence and pure renderers.
  ARCH-PURPOSE: cover tab and switcher plus folding and clipping. ARCH-CONSTRAINTS:
  terminal paint path, constant style work per existing fragment, no extra IO.
  ARCH-MOCK: no new external dependency. ARCH-STATE: render-only, retains no state
  between calls. ARCH-FUNERAL: creates no runtime durable artifact. Remaining
  architecture constraints preserve existing owners and lifecycles.
- Live acceptance and staging await TL scheduling; no visible window opened.

- Full `make test` passed (exit 0): shell/Lua prerequisites and all Go packages.
  Log: `/tmp/pair521-full.log`. Awaiting TL staging after #506 and operator live
  approval before close/PR/SHIP.

## Revisions

- 2026-10-12: Expanded Spec with TL-approved state table and small implementation plan.

- Implemented hub tab fragment inversion and switcher main-row inversion; selected
  hub rows retain their cursor and add bold, retaining blocked red.
- TDD: new cell-level tests failed on missing reverse video before implementation.
  Targeted inversion tests pass (768 tab state/idle/width combinations, folded
  counters and switcher selection/red/colour combinations). Full couchtty package
  passed; full repository make test is running. Live acceptance still pending.
- Clarification of the sent table: the existing bell cue is amber, with no bell
  glyph. Red and focus override amber exactly as before; no new glyph was added.
