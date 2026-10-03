---
id: 000363
status: working
deps: []
github_issue:
created: 2026-09-30
updated: 2026-10-02
estimate_hours: 3.29
card_mirror: '88501b8978d0f60988a5f5358e58a66d372964cf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-02T13:26:48-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module     design=0.3 impl=0.32
item: smaller-go-module        design=0.1 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: milestone-review         design=0.0 impl=0.2
item: tui-screen               design=0.2 impl=0.4
item: cross-cutting-refactor   design=0.1 impl=0.2
item: atlas-docs               design=0.05 impl=0.08
item: milestone-review         design=0.0 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: milestone-review         design=0.0 impl=0.2
design-buffer: 0.15
total: 3.29
```

Design hours are discounted because the durable plan is reviewed and approved;
`impl=` values are 40% of the v2 primitive ranges (v3.1).

- `greenfield-go-module` — M1 reboot operation + :0 journaled replace
- `smaller-go-module` — M1 unified resume route over existing executors
- `smaller-go-module` — M1 store journal builders + ReplaceThreadExpected extraction
- `milestone-review` — M1
- `tui-screen` — M2 pure per-row action table + menu wiring
- `cross-cutting-refactor` — M2 remove rename/describe/archive/repair ops; labels and matching
- `atlas-docs` — M2 README + atlas
- `milestone-review` — M2
- `smaller-go-module` — M3 one primary per repository
- `milestone-review` — M3

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (Calibration doc flagged stale; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000363-slot-world-switcher-resume-and-reboot-replace-thread-actions-plan.md`
(reviewed and approved 2026-10-02). Each milestone is a review boundary.

- [ ] M1 — Actor operations in couchcore: unified `resume` (warm, cold, proven adoption, continuation/recovery retry) and `reboot` (profile preflight before quiesce, `:0` journaled replace, `:1+` fresh, missing-directory archive-only, no name/description carry); switcher untouched.
- [ ] M2 — Switcher speaks the slot model: one pure per-row action table (offer equals permission), resume/reboot wired, rename/describe/archive/open-slot/fresh-slot/recover-* removed, labels and matching stop reading stored name/description, README and atlas.
- [ ] M3 — One primary per repository: a subdirectory start resumes the existing `:0`; a console start refuses a second primary; non-Git refusal pinned.

## Log

### 2026-09-30

- Filed from #360's design conversation with the operator. #360 keeps the
  addressing work (prefix, alias, candidates, `--agent`); this issue owns the
  switcher's action model; #364 owns slot removal.

### 2026-10-02 — design decisions before the durable plan

Operator decisions (from the code survey's open points):

- **`:0` reboot is one crash-safe operation.** The main thread store gains the
  same archive-then-create journal `fresh-slot` uses for `:1+`
  (`replaceSlotCurrent`, `couchcore/slotrecovery.go`), so `:0` and `:1+` reboot
  share one shape.
- **Unusable rows offer reboot.** With the directory present, reboot archives the
  broken record and starts a fresh agent. With the directory missing, reboot
  archives the record only, and the row explains that add slot recreates the
  directory (pair#387 repairs the leftover registration). Archive leaves the
  switcher as a separate action.
- **Reboot stops carrying stored name and description.** The fresh record starts
  without them; the archived record keeps the old values.

Survey facts the plan builds on: `open-slot` already resumes by whatever works
(warm, cold, rebuilding a lost thread pointer); `fresh-slot` archives and starts
fresh for `:1+` without touching git; `menuActionItems` (`couchtty/menu.go`)
is the per-row action authority, swept by `menu_action_sweep_test.go`; a `:0`
start in a repository subdirectory creates a second primary today
(`SelectResumableRoot` / `PathHoldsUsableThread` match the exact path).

### 2026-10-02 — plan approved

Operator approved the durable plan and its resolved ambiguities, confirming:
non-Git directories keep today's refusal (pinned by test; support would be a
separate issue); reboot on a detached row stops its running agent first;
add slot appears only on live `:0` rows; `:0` labels show the repository name
(or alias), not `repo:0`. Plan review: two rounds, eight blocking findings
resolved (adoption agent source, recovery route, journal-aware claim release,
profile preflight before quiesce, offer equals permission for unknown rows,
continuation phases, in-flight exemption, console label transport).

### 2026-10-02 — M1 implementation notes

- **Warm-path agent proof (fixed before close).** Resume guesses the agent it adopts a lost slot pointer under, from
  the slot's launch profile. The plan assumed both survivor proofs check that agent; only the cold one does
  (`ResolveEstablished`, the native ledger binds per agent). `DetachedSessions` echoes whatever agent it is asked
  about, and nothing couch reads names the running agent: pane sidecars keep stale twins and are read only for birth.
  Resume now adopts a record-less survivor on a guess only on the cold path, and refuses a detached one with the new
  `resume-survivor-unproven`, advising attach or stop rather than reboot (reboot refuses a live owner too).
  `open-slot` with an operator-chosen agent is unchanged. Warm tests were observed red before the guard; disabling
  the guard, or passing the guess as a chosen agent, turns them red again.
- **`resume` removed from `ContinuationRefuses`.** The routed resume sends a failed or running request to
  `RetryContinuation` and a pending one to `RecoverThread`, so it never meets `continuationGuard`.
  `TestContinuationRefusesMatchesTheGuardForEveryRowAction` exempts it with that reason. No switcher row changes,
  because live rows never offer resume.
- **Unusable `:0` resume usually lands on `ResumeContextWith`.** Fresh classification reads a surviving session as
  `detached`, so the Recover route is reached only when the fresh classification is unusable and a session survives.
  The route is pinned by the pure `TestChooseResumeRoute` table; the outcome (warm reattach, no fresh agent) is
  pinned end-to-end by `TestResumeReattachesASurvivorOfAnUnusablePrimary`.
- **Reboot costs two classification rounds; the plan said one.** The admission classify runs first so that a live or
  busy row is refused before the profile preflight and its idempotent family enrollment, and `prepareRetirement`
  classifies again for archive's own admission. Each round is one host-wide `list-sessions` on an operator keypress,
  off the UI path, so this is acceptable.
- **Test-declaration fixes:** `ops_declarations_test` gained `reboot`; `couchcmd/run_test.go`'s argument counts are
  now resume 5 and reboot 4.

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
