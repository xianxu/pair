---
id: 000268
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# filter Ctrl mouse-move escape sequences in pair

## Problem

`couchtty.stripWheelResizeModifier` (`cmd/internal/couchtty/mouse.go` — #213) strips the `Ctrl` bit from SGR mouse *wheel* reports (`button 64/65 + ModCtrl 16 = 80/81 → 64/65`) so `Ctrl+Wheel` scrolls instead of triggering Zellij's pane-resize. That fix lives in the `couch` host (the parent that owns the PTY and sees bytes before Zellij).

When running `pair` directly (`pair term` / standalone `pair` without `couch` in the path), the same Zellij resize still fires on `Ctrl+MouseMove`/`Ctrl+Drag`. Zellij maps `Ctrl` + pointer motion to "resize the pane under the cursor" (same family as `Ctrl+Wheel → resize`). `pair term` forwards raw SGR reports (`\x1b[<button;col;rowM`) to its child / to Zellij's scroll handling without stripping the modifier, so holding `Ctrl` while moving/dragging the mouse resizes Zellij panels instead of selecting/scrolling inside the terminal.

Reproduction: start `pair term` (or `pair` with the `pair term` right pane) inside Zellij 0.44/0.45, hold `Ctrl`, drag to select text or move the mouse over the pane — Zellij shows the resize overlay / changes panel size. With `couch` in front the wheel case is gone, but the move/drag case remains in the direct-`pair` path.

## Spec

- Strip the `Ctrl` modifier (`ModCtrl = 16`, `ModMask = 4|8|16`) from SGR mouse *move/drag* reports in the `pair` path (the equivalent of `stripWheelResizeModifier` but for motion), so Zellij does not see `Ctrl` as a resize gesture.
- Scope is the same narrow shape as the wheel fix:
  - Motion only — `BaseButton` in `{32, 35}` (SGR motion / drag with no button vs. with button; 1002 → 32 drag, 1003 → 35 all-motion — see `cmd/internal/terminalqualify/input_cases.go`). Do not alter click/press (`0-2`), release (`m`), or wheel (`64/65`, already handled in `couch` and, after #226, via `mouse_scroll_resize`).
  - `Ctrl` only — `Button & ModCtrl != 0`; `Ctrl+Shift` should still arrive as `Shift` (strip only `ModCtrl`, preserve `ModShift`/`ModAlt`), matching the wheel fix's `&^ ModCtrl`.
  - Splice, not re-encode — use `mouseinput.WithButton` like the wheel path so the wire format has one source of truth; coordinates/terminator stay exactly as the terminal sent.
- Where to apply: the `pair term` input path that parses SGR reports before forwarding (`cmd/internal/termcmd` — `mouseinput.Find`/`ParsePrefix`; also any host-level `couchtty` path if `pair` reuses it). No new config key; the behavior mirrors what `couch` already does for wheel.
- No swallowing — stripping makes `Ctrl+Move` behave like `Move` (select/drag), just as stripping makes `Ctrl+Wheel` scroll. Swallowing would make the gesture do nothing, which is not what the operator expects when their hand rests on `Ctrl`.
- Preserve existing wheel handling: `couch` keeps its `stripWheelResizeModifier` until #226 deletes it after the `mouse_scroll_resize false` floor is enforced; this ticket is for the missing *move/drag* counterpart in `pair`.

## Done when

- Holding `Ctrl` while moving/dragging the mouse over a `pair term` pane inside Zellij no longer resizes Zellij panels. The gesture completes as a normal terminal motion (text selection / cursor motion in the child app) — same as without `Ctrl`.
- `Ctrl+Wheel` behavior is unchanged (still scrolls in `couch`; in `pair` it either scrolls or is governed by #226's `mouse_scroll_resize` — this ticket does not re-break it).
- Modified clicks other than motion are unchanged — `Ctrl+Click` still reaches the child as before (narrowing per `couchtty/mouse.go:54-59` rationale).
- Covered by a unit test mirroring `couchtty/mouse_test.go` / `termcmd/run_test.go` shape: `BaseButton` in `{32,35}` + `ModCtrl` → stripped raw `\x1b[<48;…M` / `\x1b[<51;…M` becomes `\x1b[<32;…M` / `\x1b[<35;…M`, coordinates preserved, non-`Ctrl` motion passes through, non-motion buttons untouched. Existing mouse and `BaseButton`-lint tests stay green.

## Estimate

```estimate
# refined estimate pending plan approval
```

## Plan

- [ ] Reproduce: `pair term` in Zellij 0.45, `Ctrl` + mouse-move/drag over the pane — observe Zellij resize vs. host selection. Capture one raw report (`\x1b[<48;…M` / `\x1b[<51;…M`) via `mouseinput.Parse`.
- [ ] Locate the forward/parse seam in `pair term` (`cmd/internal/termcmd/run.go` — `mouseinput.Find` / `appMouse` branch; `cmd/internal/hostty` if it forwards host mouse) and mirror `couchtty.stripWheelResizeModifier` for `BaseButton` 32/35 (name to reflect motion, extract if shared).
- [ ] Add `stripCtrlFromMotion` (or generalize the wheel helper) using `mouseinput.BaseButton` + `WithButton`; narrow to `Ctrl`-only, preserve `Shift`/`Alt`.
- [ ] Add tests: `mouseinput` `BaseButton(32/35)` cases + `termcmd`/`couchtty`-style table for `{plain, Ctrl, Ctrl+Shift}` × `{32,35}` and negative cases for clicks/wheel/release.
- [ ] Manual smoke: `pair term` `Ctrl`+drag selects, `Ctrl`+move doesn't resize, plain motion still works, `Ctrl+Wheel` still scrolls.

## Log

### 2026-09-15

- Filed from operator report: `Ctrl` mouse-move still resizes Zellij panels when running `pair` directly — the `Ctrl+Wheel` filter exists only in `couch` (`stripWheelResizeModifier`, #213), no equivalent for motion (32/35) in the `pair term` path. Created as `000268`.

### 2026-09-16

- Ran `sdlc issue sync` to persist issue file.
