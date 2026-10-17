---
id: 000646
status: done
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
actual_hours: 0.57
tracker:
    version: 1
    completion:
        token: close-af2959b67216
        repository: github.com/xianxu/pair
        reviewed_head: cb4a239ed7ab14de16a7d86cabf5644de71bc7f6
        evidence_commit: fad3056ce4a5d3339f22231cfff5c26f6f2f2b31
        landed_commit: 59acc320b54624ecb9b72c743e3c01215681bd9e
---

# Couch switcher drops enhanced Ctrl+Y before copying

## Problem

After the operator's 2026-10-17 10:48 Couch restart onto 15d84ef7 (including #638), Ctrl+Y in the switcher still did not copy, and supervision.jsonl contained no clipboard_copy event. #638 instruments the copy tool, but a key dropped before its handler produces nothing.

Read-only reproduction proves a decoder mismatch: Presenter requests Kitty disambiguation/event types (ESC[>3u), but DecodePanelKeys recognizes copy only from legacy 0x19. terminal.Decoder canonicalizes Kitty Ctrl+Y press forms CSI 121;5u and CSI 121;5:1u to CSI 121;5u, which decodeCSIu rejects as a modified printable. No clipboard tool is reached.
