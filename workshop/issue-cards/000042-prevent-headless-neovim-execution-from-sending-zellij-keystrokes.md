---
id: '000042'
status: done
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1
actual_hours: 0.5
---

# Prevent headless Neovim execution from sending Zellij keystrokes

## Problem

When the headless test suite (such as `tests/queue-send-test.sh`) runs inside an active Zellij session, Neovim loaded with the full `nvim/init.lua` executes keymap callbacks such as `<M-CR>`. These callbacks invoke `send_to_agent()` which shells out to `zellij action` (like `zellij action move-focus up`, `zellij action write-chars`, etc.). Because these shell-outs communicate with the live Zellij server, they send the test inputs (such as `CCC`, `BBB`, `HELLO`) straight to the active user pane as characters, causing them to be queued or executed as prompts, polluting the active agent session.
