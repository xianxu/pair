---
id: 000306
status: working
deps: [pair#305]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T13:12:13-07:00
---

# Slots v2: multiple threads and parked admission

## Problem

Couch must permit multiple threads in one repo while preventing forgotten parked work from causing endless new threads.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Allow primary repo (alias repo:0) and additional repo:1, repo:2, etc. Each workspace hosts a full thread with the existing main-thread lifecycle: start, activate/attach, park, resume, continuation, replacement/archive. Use workspace identity for occupancy; repository identity still groups threads. An existing parked thread anywhere in that repo blocks new-thread creation until the operator activates parked work. Show which threads require attention and offer the existing activation path; do not silently create another slot or auto-resume an arbitrary thread.

Apply admission at the authoritative startup boundary for every caller, revalidating concurrent requests. Existing live-thread switching remains allowed. Preserve dirty files and active branches through park/resume and keep workspace address/directory through completed-thread replacement. Resolve precise existing states (including detached, failed, and unreadable records) using current lifecycle authority during design. No brain co-tenancy, agent roles, or scheduling system.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Primary and numbered slots support the same development workflow, including edits and normal SDLC operations in sibling dependency repositories. A numbered thread starts in `/workspace/worktree/<repo>-slotN/<repo>`; dependency access is through that thread, without automatic dependency threads or numbered addresses. Existing parked-thread admission and occupancy apply to actual Couch threads of the main repo; merely cloning an Ariadne dependency does not create an Ariadne thread or admission blocker. Park/resume and replacement preserve dependency checkouts and their local work as well as the main checkout.

### Readiness on open/resume — 2026-09-23

Before launching in a numbered workspace or cold-resuming its parked thread,
call #305's repeatable readiness operation. It validates workspace identity,
reuses completed Git setup, and runs weave compile if success is unconfirmed.
No --retry flag or special recovery action: opening/resuming again repeats the
same operation. Surface errors and stop that invocation; do not loop silently.
Warm reattachment to a still-running agent only reconnects and does not compile.
Primary :0 retains existing setup behavior. Preserve normal thread/session
ownership and resume-binding checks; dirty files and issue branches are valid.

## Done when

- Primary and two slots can run concurrently; each is independently addressable and follows existing lifecycle behavior.
- A parked :0 or :N blocks every new-thread entry point for that repo with actionable activation; another repo is unaffected.
- Multiple parked threads remain visible and creation stays blocked while any remains parked.
- Park/resume, continuation, and replacement preserve workspace identity; park/resume preserves dirty/untracked work and active branch.
- Stateful startup tests cover admission races and existing failure/recovery states without duplicate live ownership.

- Nested dependency clones do not create threads or parked-admission blockers; parent-thread park/resume and replacement preserve their dirty files, branches and local commits.

- Numbered-slot open/cold resume invokes readiness; missing setup is recovered
  by the same action without a retry flag. Warm reattach never compiles.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Map existing thread states and specify per-workspace occupancy and repo-wide admission.
- [ ] Implement shared startup/lifecycle wiring with stateful tests at actual launch boundaries.
- [ ] Verify dirty park/resume, continuation, replacement, and primary behavior.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Thread ownership remains with the environment main checkout

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — readiness on ordinary open/resume

Reason: operator agreed setup recovery belongs to the normal slot-opening action.
Delta: require #305 readiness before launch/cold resume, with repeatable missing
setup recovery; warm reattachment and primary setup behavior stay unchanged.
