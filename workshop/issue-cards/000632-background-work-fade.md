---
id: 000632
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T22:52:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Keep background-command waits bright without blocking peer admission

## Problem

#625's admitted-wrapper work timestamp fixes open and completed peer-driven turns. A separate background-command case cannot use Settled: a free composer must continue receiving messages while a 10–30 minute suite runs in the background.

Read-only live evidence on 2026-10-16: ariadne:2, Claude 2.1.296, shows an empty composer, "1 shell still running" and a "1 shell" footer while waiting for its recorded suite. The authenticated resource snapshot at 19:33:16 PDT reports idle_known=true, idle=true, implying Settled=true. CPU/rate evidence was partial and is not the work proof.

The existing SessionFrame/SlotLiveness snapshot has no independent background-work field. Detached threads do not give Couch an agent-pane snapshot; peek reads it on demand via messageService.handleTail -> TailReader.Tail. Polling that endpoint would add an IO probe. Checked-in peer/claude/2.1.296 shell-footer.raw and idle-suggestion.raw contain the same positive shell counter.
