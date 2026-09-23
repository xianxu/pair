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

Allow primary repo (alias repo:0) and additional repo:1, repo:2, etc. Each numbered workspace is a durable thread with start, activate/attach, park, resume, continuation and start-fresh behavior. Slot identity persists across conversation replacement; repository identity groups threads. An existing parked thread anywhere in that repo blocks adding another slot until the operator activates parked work. Show which threads require attention and offer the existing activation path; do not silently create another slot or auto-resume an arbitrary thread.

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

### Authoritative local-slot design — 2026-09-23

This revision supersedes the initial global-store admission proposal. Each verified
numbered environment is a durable Couch slot/thread. Its `pair-slotN/.couch/` owns
Couch thread/conversation references, lifecycle recovery records, preferences and
continuation material. Slot identity survives conversation replacement; existing
Pair/native conversation tags remain conversation handles. The directory convention
and Git establish membership; an absent global registration never makes a slot free.
Global slot listings are rebuildable indexes, not a second authoritative store.

New-slot creation chooses an unused number, checks repo-wide parked work and uses
#305's creation/readiness path. Existing slots—including incomplete setup—are opened
or recovered, never reused as free containers for newly allocated slot identities.
An existing parked :0 or :N blocks adding slots; open/resume/start-fresh within an
existing slot remains available. Reuse current launch/start-claim/park machinery for
competing agent starts; do not add a store-wide snapshot reservation system, new
workspace occupancy states, setup phases, lease expirations or repository ownership.

Opening attaches to a running agent or resumes a recoverable conversation. When
conversation recovery fails, explicitly offer start fresh in the same slot. Preserve
old evidence and all host/dependency work and preferences; no archive gesture is
required. Reconstruct missing metadata only from verified evidence; preserve damaged
records before replacement. A missing binding is not proof no process is running.
Unknown live ownership, access errors, unsupported versions and Git conflicts must
be resolved or explained, never silently overwritten or classified as vacancy.

The existing primary/arbitrary-path storage model and per-store supervisor lock
remain. Numbered slots use local storage while sharing lifecycle behavior. Keep
Pair sidecars/native transcripts and #305 Git setup records at their existing homes.
Migration and retention consumers must learn the local authoritative record before
its global copy is retired; migration must preserve conversation identity and be
repeatable after interruption. No dual-authoritative operation is acceptable.

Detailed design and remaining engineering work:
[implementation plan](../plans/000306-slots-v2-thread-lifecycle-plan.md).

## Done when

- Primary and two slots can run concurrently; each is independently addressable and follows existing lifecycle behavior.
- A parked :0 or :N blocks every new-slot creation entry point for that repo with actionable activation; existing-slot recovery remains possible and another repo is unaffected.
- Multiple parked threads remain visible and creation stays blocked while any remains parked.
- Park/resume, continuation, and replacement preserve workspace identity; park/resume preserves dirty/untracked work and active branch.
- Stateful startup tests cover admission races and existing failure/recovery states without duplicate live ownership.

- Nested dependency clones do not create threads or parked-admission blockers; parent-thread park/resume and replacement preserve their dirty files, branches and local commits.

- Numbered-slot open/cold resume invokes readiness; missing setup is recovered
  by the same action without a retry flag. Warm reattach never compiles.

- Numbered-slot state is authoritative under its environment's `.couch/`; deleting
  only the global index does not lose slots, preferences or conversation references.
- Missing/corrupt Couch metadata and lost conversation bindings allow verified
  reconstruction or explicit start fresh within the same slot, without archive.
  Unknown live ownership cannot be bypassed by start fresh.
- Migrated slot records keep existing conversation handles; interrupted migration
  is repeatable, and retention/GC cannot delete locally referenced session data.

## Plan

Execute the durable plan after operator approval and the full change-code gate.

- [ ] Finalize local storage/migration and recovery integration against the revised durable-slot model.
- [ ] Implement shared startup/lifecycle wiring with stateful tests at actual launch boundaries.
- [ ] Verify local authority/index rebuilding, dirty-work preservation, resume/start-fresh recovery, and primary compatibility.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

### 2026-09-23 — claim and design

Claimed #306 after #305 merged. Mapped all fresh starts to spawnResolved and all
child creation to launchTrackedThread. Found path-only occupancy, non-atomic
allocation/claim, narrowed startup evidence and loss of incarnation repo identity
on park. Proposed stable workspace association plus admission checked under the
existing store lock, reusing ThreadStartClaim through readiness (ARCH-DRY,
ARCH-PURE, ARCH-ORDER). No implementation changes yet.


### 2026-09-23 — design reviews passed

Fresh-context spec and plan reviews approved the proposal without blocking findings.
The durable plan includes both sides of the park/final-admission ordering test.
Awaiting operator approval before change-code; no implementation or estimate yet.

### 2026-09-23 — local authority and durable-slot recovery

Operator approved `.couch/` as authoritative, rebuildable global slot listings and
resume-or-start-fresh within a durable slot. Preserve the current supervisor model.
Revised active spec/plan to remove free-workspace reuse, archive prerequisite and
new admission-reservation machinery. Earlier design reviews are superseded; no
code changes or new estimate. Migration/retention integration remains planning work.

## Revisions

### 2026-09-23 — Thread ownership remains with the environment main checkout

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — readiness on ordinary open/resume

Reason: operator agreed setup recovery belongs to the normal slot-opening action.
Delta: require #305 readiness before launch/cold resume, with repeatable missing
setup recovery; warm reattachment and primary setup behavior stay unchanged.

### 2026-09-23 — lifecycle design proposal

Reason: turn the agreed multiple-thread contract into executable boundaries.
Delta: specify automatic versus exact workspace selection, conservative legacy
handling, atomic admission/claim, and final checks after readiness. Preserve the
existing lifecycle and scope keys; grouped UI and preference UX remain #307/#308.

### 2026-09-23 — supersede global-store allocation design

Reason: managed numbered directories define durable slots; unrecoverable
conversation state must not force slot retirement. Delta: local Couch authority,
separate conversation recovery, existing lifecycle protection, and explicit storage
migration/retention tasks. Removed the earlier free-container and whole-store CAS
proposal from the active spec; its review approvals no longer authorize execution.
