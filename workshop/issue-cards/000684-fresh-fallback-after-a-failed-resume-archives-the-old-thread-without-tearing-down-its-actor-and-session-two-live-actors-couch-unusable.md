---
id: 000684
status: working
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T19:49:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# fresh fallback after a failed resume archives the old thread without tearing down its actor and session: two live actors, Couch unusable

## Problem

On 2026-10-18 the operator relaunched parley.nvim:0 (codex thread 1-parley-nvim-37, zellij session 📁1-188 since Oct 11). The park worked (18:21:47, parked-scrollback-1-parley-nvim-37-20261018T182147.raw saved, agent stopped), but the resume did not, and Couch fell back to start fresh (18:21:57, mode fresh, selected_key fresh:1-parley-nvim-37:143), archived thread 37 (18:21:58) and started 1-parley-nvim-70 in a new session 📁1-311 (18:22:01).

Archiving thread 37 did not tear down what held it: the Couch actor couch-393ac172 (pair resume 1-parley-nvim-37, pid 60305) stayed attached, and the 📁1-188 zellij server (pid 6463, with its nvim 6467/6471 and a pair title 17468 from Oct 14) kept running. Couch then had two live actors on parley.nvim:0; both tabs showed raw thread names instead of :0, the operator could not switch to any regular slot, and had to kill Couch to recover. Investigation by the operator's pair-dev session; full notes in ops qa 2026-10-18-18-couch-hang-bug.md.
