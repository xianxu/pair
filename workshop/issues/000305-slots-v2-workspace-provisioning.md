---
id: 000305
status: open
deps: [ariadne#242, ariadne#243]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
---

# Slots v2: provision durable numbered workspaces

## Problem

Couch needs a predictable directory and Git worktree for each additional repo thread, created safely and retained for reuse.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Provision :1, :2, and subsequent numbered workspaces at ../worktree/repo-slotN relative to the primary checkout. Initially create main-slotN from fetched configured-remote main with that upstream. Existing workspaces retain their current branch and files; provisioning/resuming is not refresh. Use the shared Ariadne identity contract and dependency setup from the prerequisite tasks.

Specify number allocation and reuse under concurrency, distinguishing available persistent workspaces from occupied threads. Never renumber surviving addresses. Validate existing paths and branches rather than taking them over. Serialize competing provisioning requests and recover from partially created worktree/dependency setup without deleting user work. The repo-wide parked-thread admission rule is owned by the lifecycle task. ARCH-DRY: one provisioning path for every UI entry point; ARCH-FUNERAL: directories persist across thread replacement and issue landing.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

The agreed numbered main-worktree path is `../worktree/<repo>-slotN/<repo>` relative to the primary, for example `/workspace/worktree/pair-slot1/pair`. The enclosing `pair-slot1/` directory holds sibling dependency clones such as `ariadne/`, so existing `../ariadne` declarations stay meaningful. This supersedes the flat worktree path above. The main checkout remains a Git worktree on main-slotN; dependencies are ordinary independent clones, initially from each recorded origin/main, through ariadne#243's setup contract. No local-source dependency worktrees or primary-checkout symlinks.

Existing dependency clones retain selected revisions, dirty files and local commits during provisioning/resume. Allocation and recovery distinguish an enclosing directory, the registered main worktree, and partially acquired dependencies; never take over unrelated content. No dependency clone automatically receives a Couch slot/thread entry. Primary repos retain their ordinary sibling environment and direct UI access; numbered environments are accessed through their main thread.

## Done when

- Primary plus :1/:2 provision at the exact conventional paths with correct resting branches and configured upstreams.
- Repeated provisioning/resume preserves a dirty active branch and does not fetch/reset an existing workspace implicitly.
- Simultaneous requests cannot create duplicate workspace identities; unrelated path/branch collisions are refused.
- Interrupted Git/dependency provisioning has a tested retry path and never destroys pre-existing content.
- Fixtures cover missing remotes, non-origin remotes, repo names containing hyphens, and paths with spaces.

- Exact paths are `/workspace/worktree/<repo>-slotN/<repo>` with separate sibling dependency clones for :1 and :2; origin/main initialization follows the dependency contract.
- Repeated provision/resume and interrupted setup preserve both main-worktree and dependency-clone work, without creating extra dependency threads or changing shared-tool supply.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Specify allocation/reuse and reconcile existing Couch startup with the shared workspace contract.
- [ ] Implement provisioning and recovery behind the existing startup boundary with Git fixtures.
- [ ] Verify repeat setup and publish the lifecycle integration contract.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Provision nested environments with private ordinary clones

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.
