---
id: 000084
status: done
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 1.0
started: 2026-06-29T11:13:55-07:00
actual_hours: 0.13
---

# scrollback nvim buffer refresh

The Alt+/ scrollback viewer is currently a static snapshot:
`pair-scrollback-open` renders the agent pane's raw capture to `.ansi`, starts a
read-only nvim, and that buffer never sees later agent output. In a long-running
session, the user has to close and reopen the viewer to inspect new bottom
content.

There should be a way for scrollback nvim to load new content from within the
existing viewer. In particular, `G` should keep its normal meaning — go to the
end — but should first refresh the backing scrollback so "end" means the latest
rendered output.

## Spec

- Keep Alt+/ as a read-only scrollback viewer over the rendered `.ansi` file.
- Add an in-viewer refresh path that re-runs `pair-scrollback-render` against the
  current session's raw scrollback/events files and reloads the current buffer.
- Map `G` in the scrollback viewer to refresh first, then jump to the refreshed
  end of the buffer.
- Preserve the viewer's ANSI decoration/read-only behavior after refresh.
- Keep refresh failures non-destructive: leave the current buffer intact and show
  a warning if render/reload fails.
- Prefer a small, explicit refresh implementation over timed auto-refresh; live
  polling can be a later issue if on-demand refresh is useful.

## Done when

- [x] Pressing `G` inside Alt+/ refreshes the scrollback content and lands at the
      newest end of the refreshed buffer.
- [x] Refresh reuses the existing `.ansi` viewer instead of opening another
      floating pane.
- [x] ANSI highlighting/read-only marker behavior still works after refresh.
- [x] Refresh failure leaves the existing buffer visible and reports the error.
- [x] Headless tests cover refresh reload and `G` bottom-follow behavior.

## Plan

- [x] Add a headless test for scrollback refresh reloading changed `.ansi` content.
- [x] Add a headless test for `G` refreshing and landing at the new end.
- [x] Implement the scrollback refresh helper in `nvim/scrollback.lua`.
- [x] Wire `G` to refresh-then-end in the scrollback viewer.
- [x] Run focused Lua tests and the relevant broader test target.

Detailed implementation plan: `workshop/plans/000084-scrollback-buffer-refresh-plan.md`.

## Estimate

```estimate
model: estimate-logic-v2
familiarity: 0.9
item: lua-neovim design=0.2 impl=0.8
design-buffer: 0.3
total: 1.0
```

## Log

### 2026-06-29
- Clarified scope: standalone semi-live Alt+/ scrollback viewer refresh, with `G`
  as the important UX path because "go to end" should mean "go to the current
  end after re-rendering the latest raw scrollback."
- Planning notes: ARCH-DRY keeps refresh on the existing Go renderer and Lua
  decoration path; ARCH-PURE keeps path derivation/position behavior small and
  headless-testable; ARCH-PURPOSE makes `G` refresh-before-end the required UX,
  not a later enhancement.
- Implemented `G` as refresh-then-end in `nvim/scrollback.lua`. The refresh path
  derives sibling `.raw` / `.events.jsonl` paths from the current `.ansi`, runs
  the existing `pair-scrollback-render`, reloads the current buffer in place,
  redecorates ANSI spans, and relocks the viewer as read-only.
- Verification: `nvim -l nvim/scrollback_test.lua` passed; `make test-lua`
  passed.
- Updated `atlas/architecture.md` to record the scrollback viewer's `G` refresh
  path and the `.ansi` file's refreshed-on-demand lifecycle.
- Close boundary review returned `REWORK`: refresh replaced the annotate-attached
  buffer and could wipe pending `Alt+q` markers / lose the footer affordance.
- Fixed review finding by adding a footer-aware annotate reload hook and making
  scrollback refresh skip visible-buffer replacement when pending annotations
  exist; added regression tests for marker-protected refresh and clean
  footer-restoring refresh.
- Verification after review fix: `nvim -l nvim/scrollback_test.lua` passed;
  `make test-lua` passed; `sdlc issue validate` passed; `git diff --check`
  passed.
