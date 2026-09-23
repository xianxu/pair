---
id: 000306
status: working
deps: [pair#305]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours: 14.37
started: 2026-09-23T13:12:13-07:00
flow: {kind: full, provenance: inferred}
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

- [x] Finalize local storage/migration and recovery integration against the revised durable-slot model.
- [ ] Implement shared startup/lifecycle wiring with stateful tests at actual launch boundaries.
- [ ] Verify local authority/index rebuilding, dirty-work preservation, resume/start-fresh recovery, and primary compatibility.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.* Source calibration is marked stale;
this is a provisional ship-wall-clock estimate.

Derived after plan-quality acceptance. Estimate review found that the first
breakdown grouped distinct integrations too coarsely. The revised decomposition
below separates the actual implementation boundaries, especially all five GC
consumers, fixture construction, and the three UI flows; the total is their sum.
Tests belong to their owning component; integrated acceptance fixture construction
and installed external conformance are distinct work. Scope includes docs and ship.

Existing Go journals, lifecycle transitions and provisioning are reused; no external
library implements the repo-specific local authority migration, so there is no
extra library-availability discount. Thorough-plan design discount is ×0.2, except
initial issue/spec dialogue. Familiarity is 1.0. Each greenfield row starts with
v2 design 1.0 and impl 0.8; smaller-module rows with 0.3/0.5; consumer refactors
with 0.5/0.5; UI flows with 1.0/1.0; docs with 0.2/0.2; close with 0.2/0.5.
The implementation column applies v3.1's ×0.4 once. Real-API rows use 0/0.4.
The initial dialogue row uses 1.0/0.2 without spec discount. Design buffer is 15%.

| Boundary (same order as estimate rows) | Primitive | Design | Impl |
| --- | --- | --- | --- |
| issue/spec dialogue | issue-spec | 1.00 | 0.08 |
| slot identity and observation decisions | greenfield-go-module | 0.20 | 0.32 |
| workspace references and allocation | greenfield-go-module | 0.20 | 0.32 |
| stable thread targets | greenfield-go-module | 0.20 | 0.32 |
| filesystem/Git slot catalog | greenfield-go-module | 0.20 | 0.32 |
| stateful catalog fixture and conformance | greenfield-go-module | 0.20 | 0.32 |
| local layout adapter | greenfield-go-module | 0.20 | 0.32 |
| backend routing and index rebuild | greenfield-go-module | 0.20 | 0.32 |
| repository enrollment migration | greenfield-go-module | 0.20 | 0.32 |
| native session evidence collection | greenfield-go-module | 0.20 | 0.32 |
| same-slot reconstruction | greenfield-go-module | 0.20 | 0.32 |
| atomic fresh replacement | greenfield-go-module | 0.20 | 0.32 |
| cold-launch readiness orchestration | greenfield-go-module | 0.20 | 0.32 |
| retained-evidence recovery backup handling | greenfield-go-module | 0.20 | 0.32 |
| integrated multi-slot acceptance fixture | greenfield-go-module | 0.20 | 0.32 |
| GC Snapshot reference inventory | smaller-go-module | 0.06 | 0.20 |
| GC Recover journals | smaller-go-module | 0.06 | 0.20 |
| GC Onboard archive grace | smaller-go-module | 0.06 | 0.20 |
| GC Detach exact archived owner | smaller-go-module | 0.06 | 0.20 |
| GC Forget receipt routing | smaller-go-module | 0.06 | 0.20 |
| ThreadStore mutation and metadata consumers | cross-cutting-refactor | 0.10 | 0.20 |
| continuation and launch preference consumers | cross-cutting-refactor | 0.10 | 0.20 |
| CLI startup/reference dispatch | cross-cutting-refactor | 0.10 | 0.20 |
| startup evidence and parked admission consumers | cross-cutting-refactor | 0.10 | 0.20 |
| stable inventory row selection | tui-screen | 0.20 | 0.40 |
| slot start/preview submission | tui-screen | 0.20 | 0.40 |
| same-slot recovery actions | tui-screen | 0.20 | 0.40 |
| README operator workflow | atlas-docs | 0.04 | 0.08 |
| atlas maps/index | atlas-docs | 0.04 | 0.08 |
| project and downstream issue contracts | atlas-docs | 0.04 | 0.08 |
| one close/publication boundary | milestone-review | 0.04 | 0.20 |
| installed SDLC identity conformance | real-api-discovery | 0.00 | 0.16 |
| installed Weave setup conformance | real-api-discovery | 0.00 | 0.16 |

Design 5.26 × 1.15 + implementation 8.32 = **14.37 hours**.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.00 impl=0.08
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: tui-screen design=0.20 impl=0.40
item: tui-screen design=0.20 impl=0.40
item: atlas-docs design=0.04 impl=0.08
item: atlas-docs design=0.04 impl=0.08
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: real-api-discovery design=0.00 impl=0.16
item: real-api-discovery design=0.00 impl=0.16
design-buffer: 0.15
total: 14.37
```

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

### 2026-09-23 — concrete local-storage planning

Traced shared ThreadStore mutation primitives and all five gcruntime reference
operations. The engineering plan now specifies a routed local backend, no redundant
local manifest, root-enrollment cutover, a format fence for old readers, stable UI
slot targets, scope-based recovery evidence and atomic fresh replacement. Existing
serial operation scheduling and supervisor ownership remain unchanged. The retained
conversation/archive machinery is reused; a separate bounded raw-backup policy
covers damaged metadata. Fresh engineering review is in progress; no runtime edits.

### 2026-09-23 — engineering plan review passed

Fresh-context review approved the detailed plan after removing an unsupported
historical-conversation restore promise, covering migration with no current record,
and ordering the old-reader format fence before global reference removal. The
durable plan now specifies concrete storage, GC, launch and recovery integration
and verification boundaries. Awaiting operator plan approval; no runtime edits.

### 2026-09-23 — execution approved; gate refinement

Operator approved the detailed plan. First change-code review became stale after
a peer project commit moved HEAD, so no gate result was persisted. Its feedback
was checked against existing start transitions and operationQueue; refined the
plan with explicit transition authority, operating envelope, artifact cleanup
ownership and function-level test strategies. No new lifecycle state machinery.
Rerunning change-code before any runtime edits.

### 2026-09-23 — implementation gate passed; storage foundation

change-code passed plan-quality and estimate-quality and created the in-place issue
branch. Accepted estimate: 14.37h after separating implementation boundaries.
Baseline couchcore and gcruntime suites passed. Local backend tests first exposed
wrong-tag archive and symlink-lock writes; guards now reject both. Focused layout,
atomic successful-start recovery, absent-state read and schema compatibility tests
pass (explicit Go file set while parallel migration tests are in their red phase).
Catalog/reference/allocation foundations have focused and parser-fuzz evidence;
routing, migration and GC integration are in progress. No completion claim.
Plan gate PQ-5 is carried to implementation: audit direct metadata and retention
consumers as well as lifecycle primitives for local authority.

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
