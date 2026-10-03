---
id: 000363
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-10-02
estimate_hours:
card_mirror: '47f8ae0b4867c752fe36cd171053ebd4f4cbff66' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-02T13:26:48-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Slot-world switcher: resume and reboot replace thread actions
## Problem

Couch's switcher still offers the thread-era action set. A live `:1+` slot row
offers the same list as `:0` (rename, describe, add slot). Parked rows offer
switch coding agent, which changes nothing the operator can see. Broken rows
offer up to five repair entries (recover, recover from checkpoint, retry or
dismiss continuation, archive) that the operator cannot choose between. When a
transcript cannot come back, the operator just wants a fresh usable workspace.

Rename is the other leftover: a named thread replaces `repo:N` in the switcher
(`ActionableThreadSummary.Label`) but not on the tab bar, so the two disagree.

A slot is a repository plus its Nth checkout, displayed as `repo:N` on the tab
bar. The switcher should speak that model.

## Spec

Action sets per row (operator decisions, 2026-09-30, from #360's design talk):

| Row | Actions |
|---|---|
| live :0 | detach, relaunch, park, switch coding agent, alias (#360), add slot, remove last slot (#364) |
| live :1+ | detach, relaunch, park, switch coding agent |
| parked or detached, any slot | resume, reboot |

- **Resume**: get the previous conversation back by whatever path works: warm
  reattach, cold resume, adopt a still-running agent whose pointer couch lost,
  or retry an in-flight continuation. It replaces resume, open slot,
  recover-thread and retry-continuation. On failure it says so and points at
  reboot.
- **Reboot**: archive the old thread record with its evidence (checkpoint,
  native binding, scrollback), then start a fresh thread in the same slot or
  path with a new pair tag. It replaces fresh slot, recover-checkpoint,
  dismiss-continuation and archive. For `:1+` this is today's fresh slot; for
  `:0` it is archive plus a fresh start at the same path, as one operation.
- **No reboot on live rows.** Alt+Shift+N already starts a fresh agent there;
  the asymmetry is accepted.
- **Removed from the switcher:** rename and describe on every row, archive,
  switch coding agent on parked rows, and the separate repair entries. Stored
  names and descriptions are no longer displayed or matched; the stored field
  stays. The agent-published summary stays.
- **One primary per repository.** Starting a thread in a repository that
  already has a `:0` (for example a subdirectory of its primary checkout)
  switches to that `:0` or refuses with "one primary slot per repository"; it
  never creates a second primary thread.
- **Directories that are not Git repositories** stay supported as a single
  primary-like row: no add slot, no alias, and a second start there refuses.
- **Busy rows** (a start claimed by another couch) offer nothing and say
  "starting elsewhere". Multiple couches per machine are not a supported
  workflow; this only covers stale claims.
- Taking a repository out of couch is out of scope: no operation needs it.

## Done when

- Each row state offers exactly the table's actions, with a test per row kind
  (live :0, live :1+, parked/detached :0, parked/detached :1+, unusable,
  continuation pending, non-Git directory).
- Resume on a row whose transcript cannot come back reports that and names
  reboot; reboot then yields a usable live slot with a new tag, and the old
  record is in the archive.
- A second start in a repository with a `:0` never creates another primary
  thread.
- Rename and describe are gone from the switcher, CLI help, README and atlas;
  slot rows always label `repo:N` (or `alias:N` once #360 lands).

## Plan

- [ ] Design: the per-row action table as one pure function over row kind and state (live/parked/detached/busy x :0/:1+/non-Git), replacing `menuActionItems`' branches; durable plan if past the quick-flow shell.
- [ ] Resume: one operation that tries warm reattach, cold resume, adoption of a still-running agent, and continuation retry in order, and reports "use reboot" when none can work.
- [ ] Reboot: archive the old record with its evidence and start a fresh thread with a new tag in the same slot or path, as one operation for :0 and :1+.
- [ ] One primary per repository: a second start in a repository with a :0 switches to it or refuses.
- [ ] Remove rename and describe from the switcher, help, README and atlas; slot labels always `repo:N` (or `alias:N`).
- [ ] Tests per row kind, including a resume that cannot succeed followed by reboot.

## Log

### 2026-09-30

- Filed from #360's design conversation with the operator. #360 keeps the
  addressing work (prefix, alias, candidates, `--agent`); this issue owns the
  switcher's action model; #364 owns slot removal.

## Revisions

### 2026-10-02 — boundary with pair#367 settled: actor only

Reason: operator design review after pair#384 closed. Bulk recovery was briefly
folded in here (with a wait on ariadne#288/#289); that was reversed the same
day. Delta, relative to the Spec as filed:

- **This issue is about the actor.** Resume and reboot get a live agent into a
  slot while changing as little disk state as possible. Reboot archives the old
  record with its evidence and starts a fresh agent in the directory as it is;
  it never switches branches, refreshes, or otherwise reshapes the worktree,
  and it needs no readiness verdict. A missing directory is not reboot's job:
  archive, then add slot (pair#387 repairs a deleted slot's leftover
  registration).
- **Automation is pair#367's.** Bulk resume of slots holding work, preparing N
  ready slots for new work, and shaping a slot's workspace (resting branch,
  `weave refresh`, Couch-owned) build on this issue's resume and reboot.
- No ariadne dependency: `deps: []`.
