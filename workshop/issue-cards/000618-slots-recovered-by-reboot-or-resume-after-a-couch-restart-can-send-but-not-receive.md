---
id: 000618
status: codecomplete
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 4.39
github_issue:
started: 2026-10-16T15:08:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 3.14
tracker:
    version: 1
    completion:
        token: close-7ba29d91d774
        repository: github.com/xianxu/pair
        reviewed_head: ba41cd271292567a44b4709c16f8cef65cb5db04
        evidence_commit: f1979269f52f7ca430e9cfafbf5b15eab841ec3f
---

# Slots recovered by reboot or resume after a Couch restart can send but not receive

## Problem

After the 2026-10-16 14:57 Couch restart, a slot recovered with `couch --reboot` (pair:3, fresh, tag 1-pair-58) or
`couch --resume` (ariadne:5, tag 1-ariadne-46) can **send** to the hub but **cannot receive**: every
`couch --send-to` to it returns "Query couch --message-status … before any further send", and the status says
"receipt unavailable: no connected recipient retains this message". The agent is live and idle at its prompt.

- ariadne:5 recovered only after `couch --relaunch ariadne:5 --confirm --same-binary` (a plain relaunch refused as
  stale-binary, "changes nothing").
- pair:3 can't be relaunched either: "resume-binding-provisional: its agent has not completed a turn yet", although
  its agent completed a turn (it sent the hub its recovery report).
- The first `--relaunch` of pair:7 and of ariadne:5 each timed out at ~16 s and left a "recover / abandon" choice;
  a second call then succeeded.

So a recovered slot looks healthy in `couch --list` and to the operator, yet silently drops the TL's dispatches.
