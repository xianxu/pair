---
id: 000042
status: done
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1
actual_hours: 0.5
---

# Prevent headless Neovim execution from sending Zellij keystrokes

## Problem

When the headless test suite (such as `tests/queue-send-test.sh`) runs inside an active Zellij session, Neovim loaded with the full `nvim/init.lua` executes keymap callbacks such as `<M-CR>`. These callbacks invoke `send_to_agent()` which shells out to `zellij action` (like `zellij action move-focus up`, `zellij action write-chars`, etc.). Because these shell-outs communicate with the live Zellij server, they send the test inputs (such as `CCC`, `BBB`, `HELLO`) straight to the active user pane as characters, causing them to be queued or executed as prompts, polluting the active agent session.

## Spec

Add a helper function in `nvim/init.lua` to check if Neovim is running in headless mode (no UI attached). Any function that performs Zellij interactions (such as `send_to_agent`, `send_esc_to_agent`, `image_attach_flow`, `zellij_swap`, etc.) should check if Neovim is headless and skip the Zellij `vim.fn.system` shell-outs.

Specifically:
- Check if `#vim.api.nvim_list_uis() == 0` (indicating headless mode).
- Wrap or guard `zellij` shell-outs.

## Done when

- [x] Neovim is running headlessly, no `zellij action` shell-outs are executed.
- [x] running `tests/queue-send-test.sh` inside Zellij no longer types characters into the live agent pane.
- [x] all existing tests pass.

## Plan

- [x] Define helper or guard `is_headless` in `nvim/init.lua`.
- [x] Guard `zellij` calls in `nvim/init.lua` against headless execution.
- [x] Run the test suite via `make test` and verify no inputs are queued in the active agent pane.

## Log


- 2026-06-01: closed — make test passes cleanly, headless nvim no longer types CCC/BBB/HELLO into live Zellij agent pane
### 2026-06-01

- Discovered that the `tests/queue-send-test.sh` drives `nvim --headless` which executes `send_and_clear()` and in turn `send_to_agent()`.
- Since it runs in the same Zellij session, `zellij action write-chars` targets the live session, causing `CCC/BBB/HELLO` to be typed into the agent pane.
- Creating issue #000042 to address this.
- Added a `has_ui()` helper in `nvim/init.lua` that checks if `#vim.api.nvim_list_uis() > 0`.
- Wrapped/guarded all `zellij action` shell-outs inside `nvim/init.lua` with `has_ui()`.
- Verified that all unit/integration tests (`make test`) compile and pass successfully, and that the active Zellij agent pane is no longer polluted by `CCC/BBB/HELLO` keystrokes when tests run.
