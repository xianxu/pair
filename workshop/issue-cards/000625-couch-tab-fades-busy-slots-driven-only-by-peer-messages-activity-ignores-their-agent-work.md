---
id: 000625
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T19:17:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Couch tab fades busy slots driven only by peer messages (activity ignores their agent work)

## Problem

Couch's tab bar fades a slot's chip when the slot looks idle for a day or more (`couchtty/reserve.go:370-390`,
`idle_shade.go:16-39`), from `MenuState.Activity[thread]` (`console_presentation.go:37-39`, merged at `menu.go:461`):
"each thread's last operator input or agent work". On 2026-10-16 the operator saw `ariadne:2` faded while it was
actively working (running ariadne#412's recorded suite after a day of peer-message-driven turns). Slots driven only by
the TL's peer messages apparently never register "agent work", so they fade even when busy: the same blind spot pair#618
found for codex binding confirmation (only Pair-log turns written by the draft pane count).
