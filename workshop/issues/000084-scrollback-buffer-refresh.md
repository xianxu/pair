---
id: 000084
status: working
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours:
started: 2026-06-29T11:13:55-07:00
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

- [ ] Pressing `G` inside Alt+/ refreshes the scrollback content and lands at the
      newest end of the refreshed buffer.
- [ ] Refresh reuses the existing `.ansi` viewer instead of opening another
      floating pane.
- [ ] ANSI highlighting/read-only marker behavior still works after refresh.
- [ ] Refresh failure leaves the existing buffer visible and reports the error.
- [ ] Headless tests cover refresh reload and `G` bottom-follow behavior.

## Plan

- [ ] Add a headless test for scrollback refresh reloading changed `.ansi` content.
- [ ] Add a headless test for `G` refreshing and landing at the new end.
- [ ] Implement the scrollback refresh helper in `nvim/scrollback.lua`.
- [ ] Wire `G` to refresh-then-end in the scrollback viewer.
- [ ] Run focused Lua tests and the relevant broader test target.

## Log

### 2026-06-29

- Clarified scope: standalone semi-live Alt+/ scrollback viewer refresh, with `G`
  as the important UX path because "go to end" should mean "go to the current
  end after re-rendering the latest raw scrollback."
