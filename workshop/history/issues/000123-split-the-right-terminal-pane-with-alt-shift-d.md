---
id: 000123
status: done
deps: []
github_issue:
created: 2026-07-27
updated: 2026-07-28
estimate_hours: 1.6
started: 2026-07-27T17:29:35-07:00
actual_hours: 5.56
---

# Split the right terminal pane with Alt+Shift+d

## Problem

Layout 3 has a Pair-owned right-side terminal area, but it only supports
terminal tabs inside one floating pane. When two live terminal views are needed
side-by-side in the right area, the user has to leave Pair's keybinding model and
manually invoke Zellij splitting. The desired workflow is one terminal shortcut:
`Alt+Shift+d` creates a top/bottom split in the right-side terminal area.

## Spec

- In layout 3, `Alt+Shift+d` while focus is in the right-side terminal context
  creates a top/bottom split inside that right terminal area.
- The existing terminal pane remains above; the newly created terminal pane is
  below and receives focus.
- The new pane runs the same right-terminal command shape as the original
  terminal pane, so it is a real Pair terminal process and remains inside
  Pair/Zellij lifecycle management.
- The split uses Zellij panes, not `pair term` internal tabs, but Pair keeps the
  panes fixed in place rather than allowing mouse drag movement/resizing.
- The shortcut is terminal-local: left-stack draft/agent/review behavior must not
  be hijacked by the new binding.
- Mouse focus behavior remains unchanged. Zellij mouse layout manipulation stays
  disabled so dragging does not move or resize workbench panes.
- `ARCH-DRY`: reuse the existing terminal-local shortcut routing patterns and
  layout command strings; do not introduce an unrelated split subsystem.
- `ARCH-PURPOSE`: deliver the actual pane split and draggable boundary, not just
  another `pair term` tab.

## Done when

- `Alt+Shift+d` in the right terminal creates a top/bottom Zellij split and
  focuses the new lower pane.
- Dragging the right split panes or left workbench boundaries does not move
  Pair's layout. (Boundary drag may still *resize* panes — Zellij 0.44.3 has
  no gate for tiled boundary resize; reconciled 2026-07-27 with the tiled
  pivot, see Revisions.)
- Existing terminal shortcuts (`Alt+t`, `Alt+w`, `Alt+r`, tab switching,
  geometry toggle) still behave as before.
- Tests cover the shortcut routing/action shape and prove left/review contexts do
  not claim `Alt+Shift+d` unexpectedly.

## Revisions

- 2026-07-27: Live testing showed native `new-pane --direction down` followed
  ambiguous Zellij client focus in the pinned floating layout. The implementation
  was revised to split the invoking right terminal's explicit floating geometry
  and create a pinned lower floating terminal pane, preserving visible bordered
  Zellij panes without depending on focused-client routing.
- 2026-07-27: Live testing showed bordered floating panes could be dragged out
  of place, and left workbench boundaries still resized by mouse drag. The
  implementation was revised again to create borderless split terminal panes and
  disable Zellij mouse layout manipulation for the fixed workbench.
- 2026-07-27 (post-close rework): dogfooding the branch produced a total focus
  lockout — "focus on the right pane" could not be moved and no chord reached
  draft/agent. Root cause (reproduced in an isolated Zellij session): every
  left→right jump (`PairFocusTerminal` in draft nvim, `ActionFocusRightTerminal`
  in pair wrap) used relative `zellij action move-focus right`, which lands
  tiled focus on the invisible terminal-filler (`tail -f /dev/null`) behind the
  pinned floating terminal; the filler swallows all keys. On main a mouse click
  was the (unnoticed) rescue; this branch's `mouse_mode false` removed it,
  converting the latent trap into a hard lockout. Fix: new shared
  `pair layout focus-terminal` (layoutcmd.FocusRightTerminal) focuses the
  floating right terminal by pane id (preferring the layer-focused split pane),
  used by draft nvim and pair wrap; `mouse_mode false` reverted — it also
  killed click-to-focus, copy-on-select, and scroll. The "no drag/resize by
  mouse" Done-when line is relaxed accordingly (needs a live mouse check).
- 2026-07-27 (post-close rework 2): live testing showed borderless split panes
  render with no visible divider between the halves. Reverted the split panes
  to bordered (`--borderless false`, still pinned): the frame is the divider
  and carries the #118 tab title and scroll indicator. Residual frame-drag
  exposure is the accepted trade-off.
- 2026-07-27 (tiled pivot): after a live drag reproduced the pane moving off
  position, the user first accepted the exposure as a Zellij 0.44.3 limitation
  (see Log), then chose the structural fix: move the right terminal from the
  floating layer into the tiled tree. Tiled panes have no mouse-move operation,
  so drag-immunity is architectural; frames (divider, #118 title, scroll
  indicator) and full mouse support are kept, and the terminal-filler
  focus-trap class disappears with the floating layer. `Alt+Shift+Enter`
  expand becomes a tiled boundary resize toggling 50/50 ↔ 1/3–2/3 (user
  decision; left stack reflows while expanded). Plan Chunk 2 in the plan file
  carries the design; estimate revised 0.8 → 1.6 (+0.3 layout pivot, +0.2
  resize planner, +0.15 focus/split rework, +0.15 docs/tests/smoke).

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.08 impl=0.04
item: smaller-go-module design=0.27 impl=0.25
item: atlas-docs design=0.05 impl=0.05
design-buffer: 0.15
item: cross-cutting-refactor design=0.10 impl=0.30
item: smaller-go-module design=0.05 impl=0.15
item: atlas-docs design=0.05 impl=0.05
item: scope-pivot design=0.05 impl=0.05
total: 1.60
```

Revision 2026-07-27 (tiled pivot): four items appended for plan Chunk 2 —
cross-cutting-refactor (layout pivot + focus/classification rework across
kdl/termcmd/layoutcmd/launcher/nvim), smaller-go-module (resize planner),
atlas-docs (README/atlas updates), scope-pivot (mid-flight pivot overhead:
rebuild + live smoke). +0.8 total, matching the issue Revisions delta.

## Plan

- [x] Add failing shortcut/config tests for right-terminal `Alt+Shift+d`.
- [x] Implement the minimal Zellij action routing and config updates.
- [x] Update docs/atlas for the new keybinding.
- [x] Run focused and full verification.
- [x] Chunk 2: pivot `main-3.kdl` to a tiled right terminal (no floating layer, no filler).
- [x] Chunk 2: native tiled split action + tiled focus routing.
- [x] Chunk 2: `Alt+Shift+Enter` expand as tiled resize (50/50 ↔ 1/3–2/3), delete floating helpers.
- [x] Chunk 2: docs, full verification, live smoke.

## Log

### 2026-07-27
- 2026-07-27: closed — Implemented Alt+Shift+d as a right-terminal-local Zellij downward split running the layout3 pair term shell, with bordered panes for mouse boundary resizing. Verified with RED focused failures before implementation; then go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1; go test ./... -count=1; make test-lua; bash tests/term-pane-shortcuts-test.sh; bash tests/review-toggle-test.sh; zellij --config-dir zellij setup --check; git diff --check.; review verdict: FIX-THEN-SHIP
- Claimed locally. Broadcast failed because this checkout has no `main` worktree.
- Design approved: Zellij-native top/bottom split in the right terminal area,
  focus the new bottom pane, preserve mouse boundary resizing.
- Plan-quality found the split action command shape under-specified. Refined the
  durable plan to pin `new-pane --direction down --name terminal -- sh -c ...`,
  require the layout-3 `pair term` shell command shape, keep pane borders, and
  avoid disabling Zellij mouse mode while preserving `focus_follows_mouse false`.
- Implemented `Alt+Shift+d` as a right-terminal-local split action. Verified RED
  before code with focused Go compile failures for missing chord/action symbols
  and shell failure `term: unknown shortcut "Alt+Shift+d"`.
- Verified GREEN/full: `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `go test ./... -count=1`; `make test-lua`; `bash tests/term-pane-shortcuts-test.sh`;
  `bash tests/review-toggle-test.sh`; `zellij --config-dir zellij setup --check`;
  `git diff --check`.
- Close review verdict was FIX-THEN-SHIP: the terminal split's new KKP forwarding
  needed proof it did not regress the existing review-pane `Shift+Alt+d`
  definition shortcut. Added a review-pane raw `ESC[68;4u` visual mapping alias
  and smoke coverage. Verified with `bash tests/review-window-test.sh`;
  `bash tests/review-toggle-test.sh`; `make test-lua`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test showed `zellij action new-pane` prints the created pane id
  (`terminal_N`) to stdout, leaking into the terminal after `Alt+Shift+d`.
  Root cause: `OSRuntime.RunZellijAction` streams Zellij stdout to the focused
  pane, and `new-pane` documents the pane id as its return value. Added a quiet
  Zellij action path for the split only, rebuilt `bin/pair`, and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test then showed the split could appear in the left stack. Root cause:
  `new-pane --direction down` follows Zellij's client focus, which can differ
  from the right `pair term` process that received the forwarded key bytes in
  the pinned floating layout. Added `--near-current-pane` so Zellij splits near
  the invoking terminal pane, rebuilt `bin/pair`, and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `bin/pair term --test-shortcut Alt+Shift+d`;
  `git diff --check`.
- Live log/state check after another `Alt+Shift+d` press showed no stale binary:
  live `pair term` processes mapped to the current rebuilt `bin/pair` inode, and
  Zellij pane inventory still showed one right floating terminal plus the left
  stack. Replaced native split routing with explicit pane-id/geometry actions:
  shrink the invoking right terminal to the upper half, then create a pinned
  bordered floating terminal in the lower half while discarding `new-pane`
  stdout. Added ambiguous-focus regression coverage and `pane_y` parsing.
  Rebuilt `bin/pair` and verified with
  `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd -count=1`;
  `bash tests/term-pane-shortcuts-test.sh`; `git diff --check`.
- Live test confirmed `Alt+Shift+d` now creates the right split, but the split
  panes could be dragged around and the left panel boundary could still be
  resized with the mouse. Root cause: in Zellij, bordered floating panes are
  mouse-movable, and global mouse mode enables mouse layout manipulation for
  tiled boundaries. Switched the split panes to `--borderless true` and set
  `mouse_mode false` in the Pair Zellij config so the workbench stays fixed.
- Post-close rework (focus lockout): reproduced the reported "focus stuck on
  right pane" in an isolated session (`zellij action write 27 107` into draft →
  probe showed `Pane Terminal(1) is already focused`, i.e. keyboard captured by
  terminal-filler). Fixed left→right navigation to `focus-pane-id` via new
  `pair layout focus-terminal` (ARCH-DRY: one helper shared by nvim draft,
  pair wrap, and pair term's pane pick, mirroring review-poke's existing
  "id-based, never relative move-focus" rule) and reverted `mouse_mode false`.
  Verified GREEN: go test layoutcmd/wrapcmd/termcmd/workbenchshortcut/
  dispatcher; tests/term-pane-shortcuts-test.sh; zellij setup --check; and a
  live isolated-session replay — draft↔terminal round trip lands keyboard on
  the floating terminal (probe: `Pane Terminal(3) is already focused`), split
  still works, and Alt+k returns to the layer-focused split pane.
- Mouse-drag exposure accepted as a Zellij limitation (user sign-off after a
  live drag reproduced the pane moving off-position). Root cause verified in
  Zellij 0.44.3 source (`zellij-server/src/tab/mouse_handler.rs`,
  `determine_mouse_action`): a plain left-press on a floating pane's frame
  unconditionally returns `StartMovingFloatingPane` — no config gates it; and a
  plain left-drag on a tiled pane's frame edge starts a boundary resize, also
  ungated. `advanced_mouse_actions false` only disables hover effects/grouping
  in this version, and `mouse_mode false` (the one switch that would freeze
  drags) is off the table because it kills click-to-focus (the focus-lockout
  rescue), copy-on-select, and scroll. Revisit if a future Zellij adds a gate
  for frame-drag move/resize (e.g. folding them under
  `advanced_mouse_actions`); a pair-side geometry guard (snap-back via
  `change-floating-pane-coordinates` riding the titlepoller loop) was designed
  but not built — the drag would still visibly happen and only be undone a
  poll-tick later.

### 2026-07-28
- 2026-07-28: closed — Chunk 2 tiled pivot: full make test green (pre-existing scrollback-open Alt+x drift + parley golden excepted); zellij setup --check; git diff --check; live smoke in isolated zellij session with real client keystrokes — tiled inventory no filler, Alt+Shift+d native split, draft<->terminal round trip returns to recorded half, mouse click-to-focus, #118 tabs in split half, Alt+Shift+Enter 50%<->2/3 both split and unsplit (collapse exactly 50%), full rung ladder with split present via small-split, ClassifyLiveLayout=layout3 against live pane dump (kept as env-gated conformance probe). Frame-drag pane-move structurally impossible: no floating workbench panes exist.
- Chunk 2 (tiled pivot) implemented per plan Tasks 4–8: layout pivoted to a
  tiled right terminal (floating layer + filler deleted, swap rungs gained
  4-pane `-split` variants incl. an explicit `small-split` placed last),
  native `new-pane --direction down` split, tiled focus routing with a
  recorded last-used split half (`LastTerminalPaneStore`), launcher
  `ClassifyLiveLayout` re-keyed on the tiled terminal (legacy floating
  signature still recognized), and `Alt+Shift+Enter` as a pure-planner tiled
  resize loop (50% ↔ 2/3) with an 80ms async-settle pause.
- Live smoke (isolated zellij session, real client keystrokes through a
  PTY-attached driven client) surfaced four defects, all fixed + re-verified
  live (details in plan Revisions 2026-07-28): split halves invisible to all
  classifiers (zellij omits `terminal_command` for `--direction`-created
  panes + #118 titles defeat the fallback → new pid-verified
  `TerminalPaneRegistry` sidecar + `RoleForPaneWith` overlay + own-pane-first
  focus resolution in pair term); `--near-current-pane` creates invisible
  orphan panes (reverted); recorded-half must outrank zellij's stale
  `is_focused` in the picker; rung ladder needed `small-split`. Smoke items
  all pass: tiled inventory, split, draft↔terminal round trip returning to
  the recorded half, mouse click-to-focus, tabs in the split half, toggle
  both split/unsplit (collapse lands exactly 50%), full rung ladder with a
  split present, and `ClassifyLiveLayout` = layout3 against the live pane
  dump (kept as env-gated conformance probe
  `launcher/live_classify_probe_test.go`). Frame-drag pane-move is now
  structurally impossible (no floating workbench panes exist; zellij's
  drag-move path requires one); boundary drag may still resize (reconciled
  Done-when).
