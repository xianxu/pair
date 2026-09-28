---
id: '000266'
status: done
started: 2026-09-15T22:29:06-07:00
created: 2026-09-15
updated: 2026-09-16
actual_hours: 4.09
---

# muse Alt+Return from draft stays in composer; agent pane Return should be Send

## Problem

When `muse` is the active harness under pair, `Alt+Return` (Alt+Enter) submit from the nvim draft pane does not send. The authored text lands in the Muse composer buffer and sits there — no turn is opened.

Separately, when the Muse agent pane itself has focus, bare `Return` currently inserts a newline instead of sending. Pair's convention for coding harnesses (claude/codex/agy/muse) is `Return = Send, Alt+Return = newline` when the composer is active and no overlay/picker is open — the agent pane should behave the same. Muse currently violates that: draft-originated `Alt+Return` fails to become a send, and agent-pane `Return` fails to be a send.

This is the exact gap pair's return-remap seam exists to close (`cmd/internal/wrapcmd/harness_tty.go: harnessTTYProfiles["muse"]`, `museComposerActive`, `overlayDetectorByAgent["muse"]`, `sendKeymapByAgent["muse"]` with `plainCR=\n / altCR=\r`). The draft send path itself uses `zellij action send-keys 'Alt Enter'` (`nvim/draft_send.lua:17`), which relies on the wrap proxy translating `Alt Enter` → bare `CR` for the harness. Something in that chain is not closing for muse.
