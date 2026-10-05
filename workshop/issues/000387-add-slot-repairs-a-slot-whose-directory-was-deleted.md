---
id: 000387
status: working
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-05
estimate_hours:
card_mirror: '878f3ad955c203b5acdfaf966f1677fe2d1083e0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-05T10:47:34-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Slot reconciler: reconcile a Couch slot's dispersed state

## Problem

A Couch `:1+` slot is not one thing. Its state is spread across systems that each
own a piece: the git worktree registration and `main-slotN` branch (git, in the
primary repository), the checkout directory, the slot's dependency clones and
generated files (weave), Couch's setup-success marker and creation intent, Couch's
per-slot store `<env>/.couch` (thread record, archive, preferences), the agent
session (zellij/Pair), the agent's conversation transcript, the operator's own
work in every checkout (uncommitted and untracked files, local-only commits), and
sdlc claims. Any of these can be missing, stale or broken independently, after a
crash, an interrupted setup, or a deleted directory.

Today each Couch operation (add slot, open, resume, reboot, archive) handles the
cases it happens to meet, so failures are partial, messages are unhelpful ("open
again to retry" when a retry cannot succeed), and some states have no repair at
all. Observed in pair#367's smoke test (2026-10-05): `tools:1` was nearly empty
(setup interrupted Sep 27; no `ariadne` clone, no generated files, no setup-success
marker) and reboot could only fail. A deleted slot directory leaves a git
registration and `main-slotN` that block add slot (this issue's original scope,
kept below). A scripted `couch --rebuild` (a phase journal, i.e. a saga) was drafted
in #367 and dropped after four review rounds kept finding uncovered paths: a global
state machine over these pieces grows exponentially.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation. Principle: ariadne#291 (reconcile dispersed state; don't script it).

**Model a slot as resources and reconcile it.**
- **Resource table.** Each piece of slot state is a resource with an owner, a kind
  (Couch-internal, derived — recreatable from a source, user data — preserve
  only, runtime, external — never touched), a desired spec, an observe step that
  yields present / absent / broken / unknown, and an idempotent converge step.
  Starting point: the component table in #367's plan Revision g (C1–C15:
  `.couch` store, agent session, transcript, Pair per-thread artifacts, host
  worktree dir, git registration + setup-success marker, creation intent,
  `main-slotN`, other branches, dirty/untracked files, dependency clones,
  generated files + setup lock, claims, Couch enrollment, saved-work family).
  Verify each against the code.
- **Dependency graph.** Edges between resources (e.g. dependency clones and
  generated files need the checkout; the checkout needs the registration; the
  agent needs the setup marker and the thread record). Converge in topological
  order; tear down in reverse.
- **One reconcile operation.** Observe everything (plan), show the diff, then
  converge in order (apply). Run by the Couch server through the existing
  operation queue; idempotent, so recovery from any partial failure is "run it
  again", and creation (add slot), repair after a crash or interrupted setup,
  and rebuild are the same operation with different desired specs (rebuild =
  "derived resources reset to absent, then reconcile").
- **Guards.** User data is never converged over: it is preserved, or saved first
  (including ignored files, stashes, and local-only refs), with a size bound. An
  unknown observation stops the walk at that node, with the reason shown. A live
  or busy agent is never stopped by reconcile without the existing
  confirmations. `:0` is never recreated.
- **Errors fall out of the model:** "resource X did not converge: cause; fix",
  with the cause classified as permanent (recognized deterministic, e.g. a
  `construct/deps` substrate without a clone source), retryable (weave's "setup
  is active", timeout, cancellation) or unclassified (shown verbatim, may be
  transient). Never a futile "retry".
- **Observability.** `couch --show` and pair#367's recovery report read the
  reconciler's observations; "setup incomplete" and "leftovers block
  provisioning" are named states with their converge step.

**Carry in from #367 (decided or found in review):**
- Provisioning already re-runs `weave compile` for a present, unconfirmed host
  (`couchcore/provision.go:177-198`); a missing declared substrate must also
  trigger it (decided), but repair must never make a working slot worse: a valid
  marker with a failing compile falls back to reuse with a warning (decided).
- The weave setup lock is permanent by design and is not evidence; Couch's
  `couch-setup-success.json` is the "setup completed" signal. An invalid marker
  blocks provisioning ("setup conflict"); an unreadable one is unknown.
- No archive-only slot retirement exists (reboot's is `replaceSlotCurrent` with a
  successor); the `.couch` store lives inside the slot directory, so its archive
  and preferences must survive a directory reset and stay visible to `couch
  --archived` and `archive_gc`.
- The vacancy predicate (what add slot needs of a never-used number) must be
  derived inside `ensureHost`, which also refuses conflicting
  `branch.main-slotN.*` config and a `main-slotN` checked out elsewhere.
- Hold `.weave-setup.lock` across any delete of slot files (probe without
  O_CREAT); a saved-work family needs a real collector (`archive_gc` or the Couch
  store GC, not a storagegc family that does not exist).
- Pair-layer code must not call sdlc/weave (layer rule); couchcore provisioning
  may. Next-step text is single-sourced and names only actions reachable from the
  caller (switcher rows vs CLI).
- Full findings: #367's Log entries "M3 split" and "M3 dropped", and its plan
  Revisions g/h/i (`workshop/history/issues/000367-recover-owned-slots.md`,
  `workshop/history/plans/000367-recover-owned-slots-plan.md`).

## Done when

- A resource table and dependency graph for a `:1+` slot exist in code as the
  single source the reconciler, `couch --show` and the recovery report read; a
  test derives the resource list from the code paths that touch slot state (no
  hand-maintained second list).
- Reconcile is idempotent and converges from every partial state in a derived
  domain of per-resource starting states (present / absent / broken / unknown per
  resource), proved by a test that runs it twice and checks the second run is a
  no-op; unknown stops at its node; user data is never deleted unsaved.
- Real-git acceptance cases converge: a deleted slot directory with leftover
  registration and `main-slotN` (this issue's original case, which must no longer
  block add slot or allocation for the whole repository); an interrupted setup
  like `tools:1` (missing clone and marker); a missing dependency clone under a
  valid marker; a dirty slot whose broken checkout must be recreated saves the
  work first (host: a stash-shaped commit on a private ref; dependency clones: a
  saved-work entry in the slot store) and the result names how to restore it.
- A failure names the resource that did not converge and the cause verbatim
  (weave's own `Error:` line for setup). Only a retryable cause (setup running
  elsewhere, timeout, cancellation) says to run it again; every other failure says
  to hand it to the repository's `:0` agent. The `tools:1` case shows weave's
  missing-substrate line and the `:0` hand-off, never "retry".
- Add slot, open, resume and reboot converge the workspace through the reconciler
  rather than their own repair logic, with no confirmation for routine repair;
  `couch --show repo:N` prints the observed resources and the plan, and
  `couch --reconcile repo:N` applies it explicitly.

## Plan

To be designed in a fresh context (operator, 2026-10-05), starting from the
resource table. Expected to need a durable plan in `workshop/plans/`.

## Log


### 2026-10-02

Filed at the operator's request from the #363/#367 design talk: a deleted slot
directory should be repaired by prune-then-add-slot, not left blocking.

### 2026-10-05

Re-scoped at the operator's direction from "add slot repairs a deleted slot" to
the slot reconciler, after pair#367's M3 (`couch --rebuild` as a scripted saga) was
dropped. The deleted-slot repair stays as one acceptance case. Related:
ariadne#291 (the ARCH principle), pair#390 (wrapper messaging needs no sdlc).

Design decisions (operator, planning session):
- The desired state is a working slot. Past state matters only where it is the
  operator's: uncommitted and untracked files, local-only commits, the slot
  store's archive and preferences (the transcript is never touched). Everything
  else is derived and repaired automatically, without confirmation. There is no
  operator "rebuild" verb: a derived resource that cannot converge in place is
  saved (if it holds user data), removed and recreated.
- A leftover `main-slotN` without ownership proof is adopted (checked out again,
  its commits kept); refuse only if it is checked out elsewhere or its upstream
  config conflicts.
- Saved work: the host checkout's dirty and untracked files become a
  stash-shaped commit on a private ref `refs/couch/saved/slotN/<id>` (not the
  shared stash stack, which every worktree and session shares); dependency clones'
  dirty work and local-only commits go to `<env>/.couch/saved-work/<id>/`. Both
  are collected after storagegc's 60-day retention. Ignored files are derived and
  not saved.
- A live agent whose Couch record is lost or unreadable is adopted, never
  stopped. Reconcile never removes a checkout under a live agent; that case stops
  and reports.
- An unknown observation stops the walk at that resource.
- A converge step that fails is reported with its verbatim cause and handed to
  the repository's `:0` agent to debug; only retryable causes say run again.
  `tools:1` is this case: `tools/construct/deps` declares `substrate ../ariadne`
  without a URL (verified 2026-10-05; pair's line carries the URL), so a fresh
  `weave compile` fails identically. The fix belongs in tools.
- Surface: reconcile runs inside add slot, open, resume and reboot; `couch --show
  repo:N` shows resources and the plan; `couch --reconcile repo:N` applies it.

Code survey (2026-10-05): nothing in `cmd/` deletes a slot directory, its git
registration, `main-slotN` or its config, so any leftover blocks forever; a
leftover registration blocks add slot for the whole repository
(`SelectStartSlot`, `slotallocation.go`). Slot discovery has two disagreeing
predicates (`EnumerateSlotCandidates` filesystem-only vs `OSSlotCatalog.Discover`
plus `git worktree list`). `selectedSlot` already calls `Workspaces.Ensure` for an
unverified candidate, and `NextHostAction` is already observation-driven: the
reconciler generalizes `Ensure` rather than adding a second provisioner.

## Revisions

### 2026-10-05 — Done-when follows the planning-session decisions

Reason: operator decisions above (automatic repair, `:0` hand-off for failures
reconcile cannot fix, no rebuild verb). Delta: the acceptance bullet for a dirty
slot names where work is saved; the failure bullet drops the tools-specific fix
text and classified-fix wording for verbatim cause + retryable-only "run again" +
`:0` hand-off; the last bullet replaces "rebuild" with add slot/open/resume/reboot
plus `--show` and `--reconcile`.

### 2026-10-05 — re-scoped to the slot reconciler

Reason: operator: "without properly modeling the dispersed state across several
systems, you won't be able to patch out of the problem". Delta: the problem, spec
and Done-when were rewritten from the original deleted-slot repair (preserved in
git history and as one acceptance case) to a declarative resource model with a
dependency graph and idempotent per-resource convergence; the title changed via
`sdlc issue set-title`.
