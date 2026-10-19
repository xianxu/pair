---
id: 000686
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T18:47:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# session-watch busy-loops on freshly started Codex threads (45-80% CPU each)

## Problem

On 2026-10-18 at 18:47, the `pair session-watch codex` processes of the two Codex slots that were just started fresh were busy: pid 70442 (`session-watch codex 1-pair-69`, pair:0, started 18:17) at 45–81% CPU and pid 84747 (`session-watch codex 1-parley-nvim-70`, parley.nvim:0, started 18:22) at 34–70% CPU, continuously. Older Codex slots' session-watch processes sit at 0%. The machine load average was 55 at the time (suites and builds too), and slot launches were timing out on zellij pane birth.

Likely shape (unverified): a fresh Codex thread has no bound rollout yet, and session-watch polls or rescans ~/.codex/sessions in a tight loop looking for it.
