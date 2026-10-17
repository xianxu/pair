---
id: 000646
status: working
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T10:59:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch switcher drops enhanced Ctrl+Y before copying

## Problem

After the operator's 2026-10-17 10:48 Couch restart onto 15d84ef7 (including #638), Ctrl+Y in the switcher still did not copy, and supervision.jsonl contained no clipboard_copy event. #638 instruments the copy tool, but a key dropped before its handler produces nothing.

Read-only reproduction proves a decoder mismatch: Presenter requests Kitty disambiguation/event types (ESC[>3u), but DecodePanelKeys recognizes copy only from legacy 0x19. terminal.Decoder canonicalizes Kitty Ctrl+Y press forms CSI 121;5u and CSI 121;5:1u to CSI 121;5u, which decodeCSIu rejects as a modified printable. No clipboard tool is reached.
