---
id: '000195'
status: wontfix
created: 2026-09-05
updated: 2026-09-06
---

# The ID-reservation broadcast fails on a feature branch, so concurrent sessions collide

## Problem

`sdlc claim` and `sdlc issue new` publish an ID reservation so peers cannot
allocate the same number. In this checkout that publish fails, every time:

    Error: could not find a worktree on branch 'main'. Is main checked out somewhere?

Reproduced 2026-09-05 on a real `sdlc claim --issue 190`, from
`/Users/xianxu/workspace/pair` sitting on branch
`000172-clickable-status-bar`. `git worktree list` shows no worktree on `main` —
which is the normal state for this repo, because `sdlc change-code` branches
**in place** by default rather than creating a worktree.

So the reservation mechanism is unavailable exactly when the workflow's own
default branching mode is in use.

**Three ID collisions have already been caused by it**, each costing a
renumber plus reference-chasing:

| ids | files | resolved |
| --- | --- | --- |
| `000172` | `clickable-status-bar` vs `parallelize-zellij-session-snapshot` | latter → `#191` |
| `000173` | `wire-actor-description` vs `disposition-six-production-symbols` | latter → `#192` |
| `000179` | `layout2-terminal-toggle` (open) vs `reattach-a-detached-thread` (DONE, archived) | former → `#194` |

The third is the worst shape: one side was already closed and archived, so `#179`
meant both a live issue and a shipped one, and `sdlc claim --issue 179` refused
outright.

**And the exposure is current, not historical.** Every issue filed in these
sessions printed the same warning, so `#185` through `#195` are unreserved right
now. Any concurrent session allocating "the next free ID" can collide with any of
them, and will not find out until someone runs `sdlc claim`.
