---
id: 000501
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '0dc5a0254998dc3ad847da650f943156b7d788f6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T14:38:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast-web: a button beside the notes popup in Couch's tab bar brings the cast Chrome to the front

## Problem

The operator (10-12): during a web cast they move between Couch and the cast Chrome often; there's no quick way from Couch to get to the browser. They want a small button in the tab bar, right beside the notes popup (`Chrome ⏸ 💭N`), that focuses the cast Chrome.

## Spec

- A small button cell beside the 💭N notes popup in the `Chrome ▶/⏸` group; clicking it brings the cast Chrome's window to the front with the cast tab active (CDP `Page.bringToFront` / `Target.activateTarget` for the tab; macOS app activation of that Chrome process if needed so the window actually comes forward).
- This is the operator's explicit action, so raising the window is intended (unlike background test runs, which must never take focus).
- Choose a glyph that reads as 'go to browser' and fits the tab bar's width rules (#491's narrow-width collapse keeps it with the Chrome group).
- Also reachable from the Chrome cell's menu.

## Done when

- Clicking the button brings the cast Chrome window and tab to the front; clicking 👆 / 👽 copies the view / drive link (revision 3); tests for the cell and the action (fake browser); the operator smoke-tested it.

## Plan

- [x] `webcast.Browser` gains `Focus() error`: the real Chrome, on its loop, sends `Target.activateTarget` for the cast (active) tab, which makes it the window's tab and raises the window inside Chrome, then activates the Chrome process by PID (macOS: `NSRunningApplication` through `osascript -l JavaScript`, since the cast Chrome shares a bundle id with the operator's own Chrome, so `open -a` could pick the wrong one). The OS activator is set only by `Launch` for a windowed Chrome, so headless and scripted-pipe tests can't take focus. `FakeBrowser` counts focuses. `Cast.Focus` forwards.
- [x] Tab bar: a `↗` cell right after the Chrome label (`Chrome ⏸ ↗ 💭2`), with its own span, which a click on it focuses. Placed before 💭N so its column doesn't move with the note count.
- [x] Revision 1: the Chrome ⏸ cell is drawn in `broadcast.LiveSGR` (the LIVE cell's style). Test that the screen broadcast's `IndicatorShown` still doesn't read it as LIVE.
- [x] Tests: the reserve row (cell, span, style), the console click → `FakeBrowser` focus, and the Chrome loop sending `Target.activateTarget` for the active tab.

## Revisions

- 2026-10-12 (TL, operator request): the Chrome ⏸ / Chrome ▶ cell gets the same style as the screen broadcast's LIVE ⏸ cell (colours, emphasis, spacing), so the two cast indicators look like one family. Applied to Chrome ⏸ (red background, the terminal's own foreground, no bold; the spacing already matches). Chrome ▶ stays plain: the screen broadcast draws nothing while it isn't live, so it has no idle look to copy, and a red ▶ would claim a cast that isn't happening. Raised with the hub.
- 2026-10-12: "Also reachable from the Chrome cell's menu": the Chrome cell has no menu (click toggles live, right-click copies the link, Ctrl+click ends), so the button is the only way in. Not adding a menu for one item.
- 2026-10-12 (TL, operator request): the Chrome group mirrors the terminal's `LIVE ⏸ 👆 👽` family: `Chrome ⏸ ↗ 👆 👽 💭N 🕹Ana`. 👆 copies the view link (watch, point, notes), 👽 the drive link (real here). Right-click and Alt+right-click on the cell stay as shortcuts. Each glyph is its own span (`WebViewLink`, `WebDriveLink`), the hook #495 needs for a hover tooltip with each link's code. The glyphs are the screen broadcast's own `PointerLabel`/`ControlLabel` constants. Done-when now includes: clicking 👆 / 👽 copies the view / drive link.

## Log

### 2026-10-12
- 2026-10-12: closed — Operator smoke test PASSED (via the TL, build cafcf8a07): Chrome ⏸ ↗ 👆 👽 controls worked, ↗ brings the cast Chrome window and tab forward; scrolling while driving failed, which is #505. Tests: reserve row (spans, LIVE style, IndicatorShown not fooled), console clicks (↗ focuses the FakeBrowser, 👆/👽 copy the view/drive links, mutation-checked), Chrome loop sends Target.activateTarget for the cast tab, conformance (fake + live headless Chrome) covers Focus. After the final main merge (#503 removed the broadcast 👽; Chrome group keeps its own WebcastDriveLinkLabel): couchtty, webcast, broadcast, artifactpath ok unsandboxed. Earlier full suite (5e5d7a810 era) green: make -k test shell/Lua, test-changelog, go test ./... 100 packages. 👽 driving-state highlight (11693274c) reverted, on hold with the TL.; review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 161 added lines in code files (limit 100)

- Implemented: `Browser.Focus` (Chrome: `Target.activateTarget` on the cast tab + `activateProcess` by PID; fake counts). `↗` cell after the Chrome label; Chrome ⏸ in `broadcast.LiveSGR`.
- JXA activation script checked with a nonexistent PID (unsandboxed, exit 0, nothing raised). Real activation is the operator's smoke test: macOS 14+ cooperative activation may ignore a background request, in which case `Target.activateTarget` alone has to do it.
- `WEBCAST_LIVE_CHROME=1` live conformance (headless, now with Focus) passes: real Chrome accepts `Target.activateTarget`.
- Revision 4 (👽 lit while driving) was built and committed (11693274c) before the TL's hold arrived; reverted (5abd567cf) so the smoke test sees the current design. Cherry-pick 11693274c if the TL confirms.
- Full suite at c728c46a3 (unsandboxed, five-var scrub): `make -k test` shell/Lua targets pass; `TMPDIR=<scratchpad> make test-changelog` passes; `go test ./... -count=1` exit 0 (100 packages ok, including artifactpath once the two activators were inventoried). The make run's Go failures were the old-layout web-notes tests (moved) and a deliberate mutation check of `TestWebcastLinkButtons` (it fails with the 👽 click disconnected).
- Ready for the operator's smoke test (after #491's, per the TL).
- Merged main twice at the operator's request: the second time #503 had removed the screen broadcast's 👽 (`broadcast.ControlLabel` and the `Remote` span). The Chrome group keeps its real 👽 as `couchtty.WebcastDriveLinkLabel`, and the screen bar is `LIVE ⏸ 👆` as #503 made it.
- **Smoke test PASSED** (operator, via the TL, 2026-10-12, build cafcf8a07): the Chrome ⏸ ↗ 👆 👽 controls worked, including ↗ bringing the cast Chrome forward. Scrolling while driving failed, which is #505's, not this issue's.
