---
id: '000087'
status: done
started: 2026-06-29T18:25:25-07:00
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 0.46
actual_hours: 0.13
---

# Fix Codex Alt Enter remap

## Problem

After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:

- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.

So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.
