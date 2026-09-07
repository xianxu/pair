---
id: 000195
status: wontfix
deps: []
github_issue:
created: 2026-09-05
updated: 2026-09-06
estimate_hours:
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

## Spec

The reservation must not require a checked-out `main` worktree, because the
workflow's default branching mode guarantees there is not one.

What the broadcast actually needs is to write the reservation somewhere peers
read. Options worth weighing rather than assuming:

- Push the reservation commit directly to `origin/main` without a local
  worktree (`git push origin HEAD:refs/heads/main` for the issue file alone, or
  a dedicated ref).
- Use a reservation ref rather than the `main` branch — the payload is an ID and
  a name, not issue content.
- Create the worktree on demand, which is the smallest change and the largest
  side effect.

Whatever the mechanism, two properties matter more than which one is chosen:

1. **A failed broadcast must be loud where it is CONSEQUENTIAL.** Today it prints
   a warning at the end of a successful-looking claim; the cost lands weeks later
   on a stranger. If the reservation cannot be published, the operator should
   know that the ID is provisional.
2. **Allocation should detect a collision that already exists.** `sdlc issue new`
   scans `issues/` + `history/` for the next free ID, which is right, but nothing
   ever re-checks. A duplicate is discovered only when `sdlc claim` refuses — by
   which time both files have commits.

## Done when

- `sdlc claim` and `sdlc issue new` publish a reservation from a checkout on a
  feature branch with no `main` worktree — the repo's default state.
- A failed publish is reported in a way that says the ID is not yet reserved,
  rather than as a trailing warning on an otherwise successful command.
- A duplicate ID already present in the tree is reported by a command that runs
  routinely, not only by `claim` on the one issue that collides.
- `#185`–`#195` are checked for collisions against peers and reserved.

## Plan

- [ ] Reproduce from a clean feature-branch checkout, confirming the failure is
      the missing `main` worktree and not this machine.
- [ ] Choose the mechanism; record why the other two were rejected.
- [ ] Make a failed reservation loud at the point of allocation.
- [ ] Add a duplicate-ID check that runs without being asked.
- [ ] Sweep `#185`–`#195` for existing collisions.

## Log

### 2026-09-05

Filed after the third collision. The operator believed this had already been
fixed; a live `sdlc claim` reproduced it, which is why this issue exists rather
than the assumption standing. The probe claimed `#190` as a side effect and it
was reverted to `open` immediately.

## Log

### 2026-09-06 — duplicate of ariadne#207; closing wontfix

Filed in the wrong repo. The failure is `sdlc`'s, and `sdlc` is ariadne's:
the error originates at `ariadne/cmd/sdlc/claim.go:601`, so per the base-layer
rule a fix here would be a fix to a vendored consumer of someone else's code.

ariadne already owns this, in two open issues and one shipped:

- **`ariadne#207`** — "Publish issue files without a main worktree." The same
  bug, same error string, first hit in *this* repo on 2026-09-02. This issue's
  duplicate.
- **`ariadne#188`** — the allocation half: `issue.NextID` reads the local
  working tree with no `git fetch`, so IDs are chosen from a stale view even
  when the publish succeeds.
- **`ariadne#213`** — "nextid from origin", already **done**.

**This issue's evidence has been grafted into `ariadne#207`** rather than lost
with the file: the three real collisions (`000172`, `000173`, `000179`) with
their renumbering costs, the `#179` archived-duplicate shape that made
`sdlc claim` refuse outright, the structural cause (`change-code` branches in
place, so an actively-worked repo has no `main` worktree), and the current
exposure — now **`#185` through `#204`**, extended by the nine issues filed here
on 2026-09-06.

Nothing is lost by closing this and nothing here is actionable in `pair`. Track
the fix at `ariadne#207`.
