---
type: project
name: "couch-slots-v2"
goal: "Enable concurrent development through durable numbered workspaces with predictable locations, flexible roles, and explicit adoption of changes."
done_when: "In parley.nvim, the primary coordinates work while :1 and :2 each complete an independent issue through integration; all three remain identifiable and reusable, dirty work survives park/resume, and neither implementation workspace is silently changed by activity in another."
status: ideation
created: 2026-09-22
updated: 2026-09-23
sources: [pair/workshop/projects/couch-slots.md, brain/workshop/pensive/2026-09-11-01-pensive-couch-slots.md]
---

# couch-slots-v2

A clean definition of durable development workspaces, replacing the accumulated
layout proposals as the basis of the next discussion. **Brain co-tenancy is
outside this initial project.** Workspace numbers do not prescribe agent roles
or SDLC stages. This is a discussion draft, not an approved engineering design;
the original project's issue list is not automatically the v2 commitment.

## PRD

### Current host starting-point contract — 2026-09-23

New host slots start from fetched configured-remote main. Couch captures that
commit, creates main-slotN there, and sets the corresponding remote/main upstream.
Provisioning leaves primary local commits and dirty files untouched and makes no
judgment about which edits to commit or transfer. Interrupted creation retains
the captured baseline; existing slots preserve their current state.

Once a slot is running, the operator and agent can deliberately bring local work
over: prepare relevant source commits, then create an issue branch from another
workspace's current committed snapshot. That separate workflow follows the
agreed source/destination readiness rules and leaves resting branches unchanged.
#305 owns deterministic provisioning; ariadne#245 owns branch-from-workspace
behavior; #309 verifies both actions separately. Private dependency clones retain
the origin/main initialization contract below. This supersedes the local-source
host initialization proposal recorded in the revisions.

### Current layout and dependency contract — 2026-09-23

This agreed decision supersedes the flat paths and unresolved dependency-policy
alternatives in the dated sections below. Other thread, branch, resting-baseline,
and preference behavior remains as previously agreed.

- A primary remains `/workspace/pair`. Its numbered main worktrees live at
  `/workspace/worktree/pair-slot1/pair`, `/workspace/worktree/pair-slot2/pair`,
  and so on. Each enclosing `pair-slotN/` directory holds that environment's
  sibling dependencies, such as `ariadne/`; `../ariadne` therefore resolves
  privately without changing the dependency's logical name.
- Main numbered checkouts remain Git worktrees with the existing `main-slotN`
  resting-branch contract. Dependencies are ordinary independent clones from
  their recorded remote sources, initially selecting `origin/main`. Normal
  compile/setup/resume preserves an existing dependency's chosen revision,
  branch, dirty files and local commits. Operator and agent may explicitly use
  ordinary Git to select another revision or develop in that dependency.
- Local-source dependency provisioning is out of scope: no automatic snapshot
  of the primary dependency, linked dependency worktree, or live primary symlink.
  There is no shared dependency shelf between numbered environments. A new
  lockfile/version-management system is not a product requirement; acquisition
  failure and recovery details belong to the engineering design in ariadne#243.
- Generated links may point into the environment's dependency clones. Editing
  one of those clones intentionally affects its consumers there; another
  environment remains unchanged. Copied/merged outputs require normal explicit
  recompilation. Source isolation does not imply machine-wide package isolation.
- Source bindings and installed tools are separate: ordinary provisioning,
  resume and build do not silently change the machine-wide tool supplier.
  Explicit installation remains available.
- Primary and numbered slots support the same development capabilities,
  including coordinated changes across repositories. Primary repos share the
  existing sibling environment under `/workspace/` and have direct Couch UI
  entries. Each numbered environment has private siblings and one main Couch
  thread; its dependency checkouts are accessed through that thread and do not
  automatically acquire slot addresses, threads or preference records.
- Use each repository's existing SDLC workflow. A thread in `pair:1` may drive
  an Ariadne issue/change, publish it first when Pair depends on it, then publish
  Pair. Reservations and subsequent issue-body updates retain their normal
  explicit publication steps. No automatic recursive merge, multi-repository
  publication transaction, or special primary-only cross-repo mode is added.

```text
pair       /workspace/pair
  pair:1   /workspace/worktree/pair-slot1/pair
  pair:2   /workspace/worktree/pair-slot2/pair
ariadne    /workspace/ariadne
```

The ordinary clones at `pair-slot1/ariadne` and `pair-slot2/ariadne` are accessed
through their respective Pair threads, not automatically listed as Couch slots.

ariadne#242 remains the completed flat-layout implementation. ariadne#243 now
owns its nested-layout resolver/consumer follow-up as well as dependency setup;
ariadne#244–246 and pair#305–309 consume this revised contract. Engineering plans
and any necessary Weave changes still require their normal design gates.

### UI and task contract — 2026-09-22 (layout superseded above)

This section and the fresh task list below supersede earlier open UI questions,
model-default suggestions, directory alternatives, and the `adopt` terminology.
The worktree behavior in the following section still applies. The new issues
are task specifications, not approved engineering plans or a timeline commitment.

- A repo can host multiple threads: `repo` (alias `repo:0`), `repo:1`, `repo:2`,
  and onward. Within a known repo context, use `:0`, `:1`, and `:2`.
- Any parked thread in the repo blocks creation of another thread there. The
  operator activates parked work first. Other repos remain independent.
- The switcher groups the primary and indented slots, showing full names and
  actual paths. The tab bar groups them as `pair :1 :2 brain ariadne ...`.
- A primary at `/workspace/pair` has slots at
  `/workspace/worktree/pair-slot1`, `/workspace/worktree/pair-slot2`, etc.:
  `../worktree/repo-slotN` relative to its checkout. Resting branches remain
  `main-slotN`; the primary rests on `main`.
- Each slot supports the same thread lifecycle as the primary. Its address and
  directory survive park/resume, landing, and thread replacement. Dirty work
  and the active branch survive park/resume.
- Each workspace has independent agent and supported launch-parameter
  preferences. Preferred-model customization is outside this version.
- Say “In :2, create an issue branch from :1’s current commit.” This is ordinary
  Git branching from a resolved workspace snapshot; no dedicated `adopt` verb
  is required. Preserve the clean-source, safe-destination, exact-SHA, whole-
  snapshot, and provenance requirements described below.

Example switcher:

```text
pair       /workspace/pair
  pair:1   /workspace/worktree/pair-slot1
  pair:2   /workspace/worktree/pair-slot2
```

Pair owns provisioning and thread/UI behavior. Ariadne owns shared workspace
resolution and compatible SDLC operations. Dependency bindings and shared-tool
installation behavior require an explicit decision in ariadne#243; Weave changes
are conditional on that finding. Land preserves the resting baseline; refresh
remains an explicit operation. Branching does not transfer an issue claim.

Remaining task-level design decisions include number reuse/allocation under
concurrency, readiness checks and recovery phases, dependency binding/freshness,
preference inheritance/replacement, and display when the primary is absent.
Each has an owner in the new issues; none revives the old project’s task list.

### Current product contract — stable workspaces, adopt, land, refresh

This section supersedes the branch alternatives in the exploration below and
the earlier revisions. Workspace addresses and roles remain independent of
their current branches. Couch slots is an opinionated workflow over Git
worktrees, not a new concurrency mechanism. Worktrees share a Git repository
and object history; each has its own checkout, index, and working files.

The product consists of stable places to work and three explicit operations.
Couch finds, provisions, and resumes the workspaces. Agent roles, issue
ownership, scheduling, and interpretation of plans belong to the operator or
other tools. SDLC compatibility must honor the workspace lifecycle without
making those responsibilities new Couch concepts.

**Stable places**

- Primary (`:0`) rests on `main`. It can temporarily use an issue branch for
  implementation; the lobby role is a convention, not a main-only restriction.
- Each secondary workspace has a persistent resting branch named `main-slotN`
  (for example, `main-slot1`). It is initially created from fetched `main` on
  the configured remote and tracks that remote branch.
- Tracking is not automatic synchronization. The resting branch retains the
  workspace's last explicitly adopted baseline; another workspace's activity
  does not refresh it.
- Ordinary park/resume preserves the active branch and dirty working files.

**Adopt:** require a clean, locally committed source and a destination ready to
switch without losing work. Resolve the source workspace's current commit,
create a new issue branch there, and check it out in the destination. Record
the source address and commit. Neither workspace's resting branch nor upstream
changes. Subsequent source changes do not propagate. Do not automatically
commit or stash to satisfy the preconditions.

**Land:** integrate the issue branch into the configured remote's `main` through
the existing publication/review workflow and confirm success. Switch back to
the workspace's existing resting branch, then safely delete the completed local
issue branch. Preserve the directory and any unpublished work. Landing does
not advance the resting branch: returning to an older baseline is intentional.

**Refresh:** explicitly fetch and update the resting branch against the
configured remote's `main`. Ordinarily this is a fast-forward. If it contains
local planning commits, rebasing those commits is a separate explicit choice
that may require conflict resolution. Never silently reset away local work.
Starting another issue does not imply refresh; the operator chooses when to
adopt a newer baseline.

Local planning commits are allowed on a resting branch. They must be published,
moved to an issue branch, or explicitly reconciled before refreshing could
discard them. The resting branch is a reusable baseline and a place for planning;
issue branches carry implementation.

Adoption uses the whole committed snapshot, without issue-specific file
extraction. Included unpublished changes remain relevant to eventual integration
even when the destination's task ignores them. Claim ownership and any review
handoff remain separate workflow questions.

### Purpose and settled direction

Increase concurrent development without making the operator lose track of where
work lives. Human and agent share stable addresses such as `pair:1` and
`parley.nvim:2`; `repo` is shorthand for `repo:0`.

Each work repo has a small set of durable, isolated working directories. The
directory survives completed issues and thread park/resume. Resuming unfinished
work preserves its dirty files and active branch. Integration must not delete
the workspace or silently discard unpublished work.

Roles are conventions that the operator can change. A workspace can have a
default agent/model appropriate to its usual work, without the slot number
granting permissions or bypassing review gates. The primary can be a lobby for
coordination, or the operator's implementation workspace. The current product
contract above defines the branch behavior for both uses.

### Workflows the product should accommodate

One possible arrangement is:

| Workspace | Convention |
| --- | --- |
| `:0` | TL/TPM: orientation, tracking, coordination, quick questions |
| `:1` | Product/project lead with the most capable agent: product exploration, system design, engineering plan review through `change-code` |
| `:2` | Implementation and testing with a medium-level agent |
| `:3` | Small bug fixes, potentially using an Argos-style quick workflow |

Product and engineering design are iterative; engineering findings can change
the product definition. Roles must not force a one-way waterfall. A quick bugfix
flow is a workflow choice, not an unconditional slot-based gate bypass.

Another arrangement uses the primary for product and engineering design and
two numbered workspaces for independent implementations. Alternatively, primary
handles tracking and questions while each numbered workspace claims and designs
its own issue. This is the initial `parley.nvim`, `parley.nvim:1`,
`parley.nvim:2` trial.

For ariadne, the operator anticipates one implementation workspace supplying
the machine's shared binaries, with other workspaces supporting design. Which
workspace supplies shared dependencies or installed tools is therefore distinct
from which workspace hosts a particular conversation.

### Earlier branch and handoff exploration — superseded

The following records the initial alternatives. The current product contract
above supersedes its branch and snapshot-adoption questions; it does not require
an issue-file-aware handoff mechanism.

The operator's candidate model gives each numbered workspace a durable
`slot-main` branch. It normally follows remote main, but can remain at an older
snapshot while work proceeds. A design workspace might refresh until claim,
then freeze its base. An implementation workspace could explicitly adopt the
design workspace's snapshot, implement one of its ready issues, integrate to
remote main, and later adopt another snapshot.

The operator is considering working directly on these durable branches rather
than requiring an issue branch per task. Whether to retain issue branches,
what `slot-main` is named, and how repeated publication works remain open.

The design workspace may advance while implementation continues. Adoption must
therefore distinguish a snapshot from continuous following. It also needs to
distinguish the common code base from the commits containing the reviewed spec
and plan. Copying the old base alone would not transfer a newly written plan.

Several issues may be ready in the design workspace while implementation takes
only one. Decide how planning artifacts are published and selected without
accidentally taking another issue's implementation. Ownership transfer after
claim and review is also unresolved. Reading a transcript or TTY log may supply
additional reasoning, but whether and how to do that is still exploratory.

### Open questions recorded on 2026-09-22 (dependency/layout settled above)

- Choose the canonical directory spelling and how new slot numbers are allocated
  or bounded. `main-slotN` and display address `repo:N` are settled.
- Decide where the three operations are exposed: documented agent procedures,
  commands, or both; identify the owner of each operation without duplicating
  the existing SDLC publication/review workflow.
- Decide dependency binding and freshness. Linking every dependency to its
  primary checkout is under reconsideration because those checkouts can move.
- Separate shared binary installation from isolated source workspaces,
  particularly for ariadne.
- Define exact readiness and recovery rules: which untracked files block
  adoption, destination work already in progress, source changes during capture,
  partially provisioned paths, and integration confirmed but cleanup interrupted.
- Define thread replacement within a durable workspace and minimum display of
  address, branch, and state. Decide whether configurable agent defaults ship
  here or remain a separate enhancement.

Claim transfer and review applicability need SDLC compatibility decisions where
an agent continues another agent's issue. They are not effects of `adopt` and do
not justify adding artifact interpretation to Couch.

### Initial acceptance boundary

The first trial uses `parley.nvim` plus `:1` and `:2`. Primary coordinates work;
the two implementation workspaces complete separate issues concurrently and
integrate their work. Their addresses stay recognizable, and the directories
remain reusable after integration. Park/resume preserves unfinished dirty work.
Activity in one workspace does not silently switch, reset, or refresh the other.

The trial must exercise `adopt`, `land`, and `refresh`: adoption captures a clean
source commit on a new destination branch without moving either resting branch;
landing returns to the existing resting branch without refreshing it; explicit
refresh advances a clean, behind resting branch. Attempts that would lose dirty
or unpublished work must stop visibly. Starting a new issue from the resting
baseline must not silently refresh it.

This trial does not require hard-coded roles, brain co-tenancy, automatic task
scheduling, or the entire PRD/Argos projects. Git snapshot adoption is in scope;
automatic issue-ownership transfer or transcript-driven design handoff is not
part of the adoption primitive.

## Estimate

Not estimated or committed to a timeline. Settle the operation surface,
dependency policy, and recovery details before deriving implementation scope.

## Breakdown

Current scope: the 2026-09-23 contract above governs all outstanding tasks. #242
is complete; #243 is working and includes the nested identity follow-up. The
2026-09-22 baseline below is retained as the original breakdown. Dependency
policy is now settled; detailed implementation/recovery design remains.

Fresh task baseline requested on 2026-09-22. Every issue below is newly created;
no earlier slot issue is reused or a dependency. Issue bodies contain the detailed
scope, completion criteria, verification expectations, and blocking references.
No implementation has started and no estimates or deadline are committed.

- [x] Resolve repository and workspace identity [ariadne#242]
- [x] Resolve dependency and shared-tool bindings [ariadne#243]
- [x] Make concurrent issue workflows safe [ariadne#244]
- [ ] Support branching from a workspace and explicit refresh [ariadne#245]
- [ ] Land without removing or refreshing the workspace [ariadne#246]
- [x] Provision durable numbered workspaces [pair#305]
- [ ] Support full slot threads and parked-thread admission [pair#306]
- [ ] Group slots in the switcher and tab bar [pair#307]
- [ ] Persist independent workspace preferences [pair#308]
- [ ] Run the three-workspace acceptance trial [pair#309]

Sequence: workspace identity comes first. Dependency setup, concurrent workflow
safety, and branch/refresh behavior can then be designed independently. Landing
uses the identity, concurrency, and branch contracts. Couch provisioning requires
identity plus dependency setup; thread lifecycle follows provisioning, then UI
grouping and preferences can proceed independently. The trial joins both sides.
These are issue-level tasks; implementation plans and any genuine milestone
boundaries are derived when each task starts.

The first outstanding product decision is dependency binding in ariadne#243.
No change to old issue statuses or bodies is part of this breakdown.

<a id="ariadne-242"></a>
### ariadne#242 — Resolve repository and workspace identity

**est:** 2.65h
**actual:** 1.35h
**started:** 2026-09-22
**closed:** 2026-09-22

Shared Git-verified workspace identity and the read-only JSON command are
implemented, including SDLC artifact/project/calibration consumers. Close review
returned SHIP after strict OID validation and README corrections. The regression
suite passed with the known #210 missing-plan test excluded; publication follows
the close gate. Provisioning and lifecycle operations remain subsequent tasks.

<a id="ariadne-243"></a>
### ariadne#243 — Nested identity and dependency setup

**status:** done — [Ariadne PR128](https://github.com/xianxu/ariadne/pull/128) merged; SHIP review
**actual:** 2.89h
**closed:** 2026-09-23
**started:** 2026-09-22

**Scope event 2026-09-23:** adopts ordinary remote dependency clones in each
numbered environment and adds the nested-layout follow-up to #242's shared
resolver. The shared-baseline/local-source alternatives are superseded. Existing
SDLC handles dependency-first publication; the task does not add recursive merge.

**Implementation event 2026-09-23:** nested identity and JSON v2, environment-local
SDLC content lookup, ordinary remote-main dependency acquisition and inherited
setup exclusion are implemented. Pair#310/PR154 and parley.nvim#274/PR199 supply
remote metadata. Two real Parley environments passed initial/repeat composition,
private feature/dirty-work preservation and runtime acceptance. The full Go suite
and vet passed; existing #210 remains excluded. One Parley performance spec timed
out in the full run and passed all three cases on an isolated rerun. Boundary
review and Ariadne publication follow; #242 stays closed.

<a id="ariadne-244"></a>
### ariadne#244 — Concurrent issue workflows

**status:** done — M1, M2 and final issue review SHIP; PR #129 merged
**est:** 17.01h (revised simplified scope)
**started:** 2026-09-23

M1 provides fresh status-only claims and explicit selected documentation-commit
publication in every checkout, including :0 and private dependency clones.
Real claim/allocation races and caller-state preservation tests pass; the full
workspace/SDLC suite and vet pass (known ariadne#210 fixture excluded). M2 retains
short external-review locks and validation before result persistence; its integrated
workflow, real signal/race, and full regression tests pass; its review returned SHIP.

<a id="ariadne-244-m1"></a>
### ariadne#244 M1 — Claims and selected documentation publication
**closed:** 2026-09-23
**actual:** 4.76h

**status:** closed; SHIP review, no findings

Fresh remote status reserves work; agents explicitly select documentation commits
for three-way publication in every checkout. Full workspace/SDLC tests and vet
passed, including real clone/worktree claim races and isolated caller state.
Identical Git commits do not establish caller ownership; ambiguous reservation
acknowledgments remain uncertain. Review lock changes belong to M2.

<a id="ariadne-244-m2"></a>
### ariadne#244 M2 — Review lock scope and integration
**closed:** 2026-09-23
**actual:** 1.34h

**status:** closed; SHIP review, no findings

Planning and close reviewers run without the repository lock. Prepared inputs,
branch identity and ledger generations are revalidated before recording results.
Stale, interrupted and failed-relock reviews cannot persist authority. Real CLI
SIGINT/SIGTERM cleanup, full regression tests, vet and nested dependency
publication/conflict recovery passed (known #210 fixture excluded).

<a id="pair-305"></a>
### pair#305 — Provision durable numbered workspaces

**status:** done — [Pair PR155](https://github.com/xianxu/pair/pull/155) merged; SHIP review
**actual:** 4.66h
**closed:** 2026-09-23
**started:** 2026-09-23

Both prerequisite contracts are available. The [implementation plan](../history/plans/000305-slots-v2-workspace-provisioning-plan.md)
uses remote-main initialization, one repository lock for host Git creation,
small creation intent and a setup-success marker. Missing success runs Weave again on the same readiness invocation; Weave owns dependency locking and recovery. Ready reuse validates
the host without fetching or composing. Thread admission and launch remain with pair#306.

## Log

### 2026-09-22 — fresh definition requested

The operator requested `couch-slots-v2` to separate current intent from many
iterations of concurrency design. The latest discussion favors durable worktrees
with flexible uses over fixed numbered roles, excludes brain co-tenancy, and
reopens branch and dependency policy. The earlier project and pensive remain
sources of rationale, not inherited implementation commitments. No code or issue
status changes are part of this capture.

### 2026-09-22 — fresh task breakdown recorded

Created ariadne#242–246 and pair#305–309 with explicit dependencies and
observable completion criteria. Fresh-context review approved the coverage and
dependency graph; all ten issue records and this project passed schema
validation. Issue bodies were checkpointed with `sdlc issue sync`. No task was
claimed and implementation has not started. Existing issues remain untouched.

### 2026-09-22 — workspace identity published

ariadne#242 merged through PR #127 and was archived after a SHIP close review.
The shared resolver and SDLC consumers are available in ariadne main. The next
project task is ariadne#243, whose dependency-binding decision remains open.

### 2026-09-23 — nested environments and dependency workflow agreed

Recorded the operator's four-point agreement: nested main worktrees with private
sibling clones; dependencies initially from origin/main with explicit later
revision selection; symmetric development capabilities with different Couch UI
visibility; existing per-repository SDLC for coordinated work. Updated all nine
outstanding task contracts and their acceptance criteria. #242 stays complete;
#243 owns the layout follow-up. No implementation, timeline, or project lifecycle
transition is implied by this documentation update.

## Revisions

### 2026-09-22 — committed snapshots and issue branches

Reason: the operator accepted stable workspaces with flexible roles and explicit
adoption, and asked to simplify handoff without coupling it to issue-file layout.
These decisions supersede the corresponding alternatives above.

**Agreed direction:** durable workspaces host issue branches; they need not use
one permanent branch for successive implementations. Before another workspace
adopts work from a source workspace, the source must be clean with its intended
work committed locally. Adoption should identify the exact source commit, so
subsequent movement of the source branch does not change what was handed over.
This requirement applies to adoption, not to ordinary dirty park/resume.

**Handoff scope correction:** Couch should not infer which files belong to an
issue or cherry-pick an artifact set based on Ariadne's directory conventions.
Starting a destination issue branch from the source's committed snapshot can
include other planning work. Whether that inherited work is ready to publish
must be considered at integration; ignoring a file during implementation does
not exclude its unmerged changes from the eventual merge.

**Open lifecycle question:** a parking branch is not yet required. Compare a
small persistent resting branch per workspace against resting at a detached
commit after confirmed integration. Retaining completed issue branches would
need an explicit cleanup policy; generic Git garbage collection does not remove
live branch references. Preserve the workspace under every option.

**Agent recommendation, not yet adopted:** accept whole committed snapshots as
the simple handoff primitive; show their unpublished changes before adoption.
Prefer publishing ready planning artifacts separately when practical, but do
not require a file-aware transfer mechanism. Use ordinary three-way merging and
explicit reconciliation for competing edits, not timestamp-based overwrites.
The source being clean provides traceability, not proof that every included
change is ready to publish. Decide issue ownership separately from moving Git
state; a snapshot alone does not transfer the claim.

### 2026-09-22 — resting branches and temporary snapshot adoption agreed

Reason: the operator accepted a persistent resting branch and the refinement
that cross-workspace adoption changes only the new issue branch, not the
destination's resting branch or upstream. The destination must also be ready
to switch.

Delta: added the current branch contract at the start of the PRD so readers do
not have to reconstruct it from exploration history. `main-slotN` tracks the
configured remote's `main` without continuous synchronization; independent work
starts from a refreshed baseline, adopted work starts from an exact source
commit, and confirmed integration returns the workspace to its resting branch
with safe issue-branch cleanup. Local planning commits and dirty park/resume
remain supported. This resolves the earlier parking-branch, branch-name,
primary-role, and basic integration alternatives; directory naming, dependency
policy, and claim/review ownership transfer still need design.

### 2026-09-22 — worktree workflow and three explicit verbs

Reason: the operator converged on slots as stable, memorable addresses for
durable Git worktrees, with usage left to the operator, and accepted `adopt`,
`land`, and `refresh` as the workflow vocabulary.

Delta: consolidated the current PRD contract around stable workspaces and those
three operations; marked earlier exploration as superseded and narrowed the
remaining questions. This explicitly replaces the previous automatic refresh
at issue start and after landing: refreshing is now a separate requested action.
Land confirms remote integration, switches to the unchanged resting branch,
then safely removes the completed local issue branch. Snapshot adoption is in
the acceptance trial but does not select files or transfer issue ownership.
The product boundary is agreed; implementation design and dependency behavior
remain unfinished. No lifecycle status or timeline commitment changes here.

### 2026-09-22 — UI contract and fresh implementation tasks

Reason: the operator specified the thread/UI behavior and requested fresh tasks
to avoid carrying historical musing into implementation.

Delta: added the current UI/task contract above, settled the flat slot directory
convention and contextual :0/:N addresses, specified repo-wide parked-thread
admission, grouped display, and independent agent/parameter preferences without
preferred-model customization. “Branch from another workspace” replaces `adopt`
as the user-facing description; snapshot semantics remain unchanged. Added ten
fresh tasks across Pair and Ariadne with completion criteria and dependencies.
Existing issues are not reused, closed, or made prerequisites.

The previous Breakdown was a discussion sequence: identity/roles and three-verb
semantics were checked; operation/recovery design, dependency/tool behavior,
acceptance refinement, and an audit of old issues were still open. The new tasks
replace that sequence. In particular, auditing/reusing historical issues is no
longer the route to this project’s implementation scope. Prior revision entries
remain historical context; the current UI/task contract takes precedence.

### 2026-09-23 — nested layout replaces flat slots

Reason: concurrent base-layer development exposed conflicts from live shared
sources; the operator chose isolated sibling environments while retaining
relative dependency declarations and fast local cross-repository edits.
Delta: the new leading contract replaces flat slot paths, the shared dependency
baseline proposal, and local-source acquisition alternatives. Dependencies start
from their recorded origin/main and retain state until explicitly changed.
Couch visibility distinguishes primary repos from numbered environments; SDLC
publication remains per repository. #243 adds the nested identity correction;
#244/#245/#246 now depend on that contract. Provisioning, UI, lifecycle,
preferences and acceptance tasks were aligned without reopening #242.

[ariadne#242]: #ariadne-242

[ariadne#243]: ../../../ariadne/workshop/history/issues/000243-slots-v2-dependency-bindings.md

[ariadne#244]: ../../../ariadne/workshop/issues/000244-slots-v2-concurrent-workflows.md

[ariadne#245]: ../../../ariadne/workshop/issues/000245-slots-v2-branch-and-refresh.md

[ariadne#246]: ../../../ariadne/workshop/issues/000246-slots-v2-durable-slot-landing.md

[pair#305]: ../history/issues/000305-slots-v2-workspace-provisioning.md

[pair#306]: ../issues/000306-slots-v2-thread-lifecycle.md

[pair#307]: ../issues/000307-slots-v2-grouped-thread-display.md

[pair#308]: ../issues/000308-slots-v2-workspace-preferences.md

[pair#309]: ../issues/000309-slots-v2-three-workspace-trial.md

### 2026-09-23 — #243 implementation checkpoint

Reason: approved engineering plan executed. Delta: updated the #243 detail block
with implemented scope and real verification, preserving earlier design history.

### 2026-09-23 — #243 published

Reason: server-side merge confirmed. Delta: #243 is done via Ariadne PR128,
measured actual 2.89h, with metadata prerequisites merged via Pair PR154 and
Parley PR199. Updated its detail status and archived issue link. The next
Ariadne task remains #244; #242 was not reopened.

### 2026-09-23 — host slots start from local commits

Reason: operator explicitly requested carrying relevant committed local changes
into new slots. Delta: host main-slotN starts at the accepted source-workspace
SHA, with remote/main tracking configured separately. Updated #305 and #309;
private dependency initialization and explicit refresh retain their contracts.

### 2026-09-23 — restore remote-main host initialization

Reason: operator confirmed that deciding which local work to commit belongs to
the agent/operator after slot startup. Delta: restored fetched remote/main as
the new host baseline in #305 and #309, removed local-source preparation from
provisioning, and retained explicit later issue-branch transfer via ariadne#245.
This supersedes the preceding local-commit initialization revision.

### 2026-09-23 — Pair provisioning engineering plan

Reason: operator requested continuing pair#305 after settling remote-main
initialization. Delta: recorded #305's working design state and durable plan;
no implementation completion, estimate or timeline is claimed.

### 2026-09-23 — simplify #305 setup recovery

Reason: operator requested a simpler design using repeatable Weave compilation.
Delta: #305 now uses one Git creation lock, bounded host intent and one successful
setup marker; removes Couch dependency inventory and provisioning phase tracking.
The revised plan needs fresh review and approval before implementation.

### 2026-09-23 — repeatable readiness on slot open

Reason: operator prefers idempotent operations over a separate retry mode.
Delta: #305 removes --retry and ensures readiness on each invocation. #306 calls
it before numbered-slot launch/cold resume; warm reattachment only reconnects.
Missing setup success reruns Weave; failures are visible and another ordinary
open/resume retries. Primary :0 setup behavior remains unchanged.

### 2026-09-23 — ariadne#244 publication scope simplified

The operator confirmed origin/main status as the reservation authority: fresh conditional open→working claims, with already-working refusal and no owner tokens or private receipt store. The agent selects a coherent documentation commit, potentially grouping issue, separate plan and related project records; SDLC publishes its Git change against fresh origin/main with merge/conflict handling. This applies equally to :0, numbered worktrees and ordinary dependency clones. Local issue sync remains issue-file-only. External reviews must release local transaction locks and revalidate before persisting results. The earlier #244 engineering estimate is superseded; no implementation completion is implied.

### 2026-09-23 — #305 implementation

The repeatable readiness operation is implemented, with internal CLI access,
remote-main SHA capture, host creation recovery and a Weave success marker.
Focused race tests, actual SDLC/Weave conformance and built-CLI smoke pass.
Full-suite verification and close review remain in progress. #306 will wire
normal slot open/cold resume to readiness; grouped UI and preferences follow.

### 2026-09-23 — #305 close review passed

Provisioning passed the full Go suite, targeted race/vet checks, build, parser fuzz,
CLI smoke and live SDLC/Weave conformance. The close review returned SHIP;
measured actual is 4.66h. Publication follows; #306 owns thread integration.

### 2026-09-23 — #305 published

Pair PR155 merged; SDLC marked #305 done and archived its issue and plan.
Updated the portfolio status and links. #306 can now integrate directory readiness
with thread creation and cold resume.

### 2026-09-23 — ariadne#244 M1 verification checkpoint

Implemented the simplified publication scope and passed the full workspace/SDLC regression suite plus vet. Reservations use remote status without a slot ownership registry; the branch name associates local work with its issue. Identical Git candidate commits do not establish caller ownership. The M1 review returned SHIP with no findings; external-review lock changes are the separate M2 boundary.

[ariadne#244 M1]: #ariadne-244-m1

[ariadne#244 M2]: #ariadne-244-m2

### 2026-09-23 — ariadne#244 published

Ariadne PR https://github.com/xianxu/ariadne/pull/129 merged. SDLC marked #244 done, archived its issue/plan/review records and adopted measured actual 6.38h. Claims use fresh remote status; explicit documentation commits publish with three-way conflict handling in every checkout. Planning/close reviewers run unlocked and reject stale or interrupted results before persistence. Full workspace/SDLC tests, vet, real Git races, signal/race checks and nested dependency recovery passed; known #210 fixture remains excluded.
