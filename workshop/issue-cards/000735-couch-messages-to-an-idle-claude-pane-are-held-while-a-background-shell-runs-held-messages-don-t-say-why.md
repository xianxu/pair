---
id: 000735
status: working
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T11:54:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: messages to an idle Claude pane are held while a background shell runs; held messages don't say why

## Problem

On 10-20, messages to ops:0 (a Claude hub, idle with an empty composer) were accepted but not typed in for 12
minutes, then all arrived at once when the operator typed into the pane ("hmm, are you stuck?"):
- `couch --message-status`: e60561fc accepted 11:09:59, 39e08f54 accepted 11:11:05; both `submitted` only at about
  11:21:42 (RetainUntil 12:21:42 minus the 1 h retention); about ten messages in the batch.
- ops:0's recording (`scrollback-1-ops-71-claude.events.jsonl`): the turn finished at 11:08 ("done 11:08AM"), the
  composer was empty, and **only 28 bytes of output** were written between 11:09:58 and 11:21:33. The footer read
  `✻ Brewed for 6s · done 11:08AM · 1 shell still running` and `⏵⏵ bypass permissions on · 1 shell · ↓ to manage`:
  a background Bash (a `gh` watch loop) was running.
- Senders saw `waiting for recognized composer` on every send to ops:0 all day.

**Hypothesis (unconfirmed):** Couch's Claude composer recognition doesn't accept the idle state while Claude shows a
running background shell in its footer, so it holds delivery until something changes the screen. The operator's
typing then started a turn and the queue flushed.

Also: Couch's supervision log (`~/.local/share/pair/couch/supervision.jsonl`) hasn't been written since 10-19 15:56
and its pair-diagnostics segment is empty, so there is no record of readiness decisions to confirm this from.
