---
id: 000305
status: working
deps: [ariadne#242, ariadne#243]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T10:54:57-07:00
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

### Proposed provisioning design — 2026-09-23 (awaiting approval)

**Boundary.** Add one reusable workspace-provisioning operation to Couch's
existing operation/dispatch system. Its explicit inputs identify a primary repo
path and positive slot number. It prepares a working directory and returns its
verified address/path/readiness; it does not launch an agent. Keep initial use
on an internal operation surface through `DirectStoreExecutor`, avoiding the
unavailable CLI live-owner route and an unnecessary singleton supervisor lease.
#306 will call the same operation when its
new-thread flow allocates a slot and will enforce repo-wide parked admission.
No fetch, directory creation, or composition runs during start-form preview.

**Identity and layout.** Consume `sdlc workspace --json` schema v2 through an
injected, cancellable process seam. Require valid typed fields and reject older,
unknown, truncated or inconsistent output. Resolve the canonical primary and
Git common directory, derive the agreed nested candidate path, and revalidate
new/existing worktrees through the same resolver. Do not copy Ariadne's Git
identity implementation into Pair. The result is an observation; mutation needs
fresh checks while holding the provisioning lease (ARCH-DRY, ARCH-SECURE).

**Number policy.** Explicit :N requests never silently substitute another number.
For the later automatic caller, expose a pure selection policy: reuse the lowest
ready workspace with no thread, otherwise allocate the lowest unused positive
number. A parked, live or detached thread occupies its workspace. Unknown
occupancy or partial preparation is not free space; report it for inspection or
retry. Thread inventory/admission comes from #306's authoritative caller; this
issue does not invent another registry. Never renumber existing workspaces.

**Create from remote main.** Select the configured remote/main source; an
explicit remote may resolve ambiguity. Fail visibly if no unique source can be
established. Fetch that main ref, capture and record its full SHA, create
main-slotN at that SHA with remote/main as its upstream, and create the registered
nested worktree. Record the captured SHA before creation effects so interrupted
retries reconcile against the same baseline rather than silently choosing a
newer remote tip. Couch does not select, commit, stash or transfer local changes.
Dirty files and unpublished commits in the primary do not prevent creation when
Git can safely create the separate worktree; they remain untouched.

Invoke `weave compile` with default targets from the host worktree. Publish ready
only after exit 0 and final identity validation. Dependency origins remain
independently governed by Weave: private ordinary sibling clones initialize from
their recorded origin/main, without dependency threads or preference records.

**Bring local work over later.** Once the slot is started, the operator and agent
may prepare relevant source commits and explicitly create an issue branch from
another workspace's committed snapshot, following ariadne#245's clean-source and
safe-destination rules. This action records the source address/SHA and leaves
both resting branches unchanged. Local-work transfer is a separate requested
workflow, not a provisioning prerequisite or an automatic Couch action.

**Existing workspace.** A verified ready workspace returns without fetch,
checkout, reset, or compile. Its current issue branch, dirty/untracked files,
resting baseline/upstream, dependency revisions/work, and machine-wide tool
supplier remain unchanged. A registered conventional workspace without a Couch
readiness record can be inspected; dependency preparation requires an explicit
request before marking it ready. Do not infer setup success from directory
existence. A missing or externally changed workspace must be revalidated rather
than trusted because it has a saved receipt.

**Ownership and recovery.** Serialize mutations for a slot across processes,
with repository-wide exclusion for allocation/ref creation where required. Use
OS-held leases and ensure live setup children retain the relevant exclusion if
the caller dies; avoid holding the Couch thread-store lock during network/build
work. Weave owns its dependency setup lock and clone-stage recovery. Pair owns
only the host worktree transaction and its readiness evidence.

Represent absent, reserved, host-created, preparing, ready, retry-needed and
conflicting observations explicitly in a pure transition model (ARCH-ORDER).
Record intent before effects and reconcile actual Git/filesystem state after
uncertain outcomes. Store bounded per-slot metadata outside the working tree,
keyed by canonical repo identity, slot, and operation generation. A marker alone
never authorizes takeover. Reuse only a host/ref proven to belong to this
transaction or independently verified as an existing compatible workspace.
Unrelated directories, branches, symlink aliases and malformed records refuse
without overwriting anything. An interrupted reservation or failed setup stays
attached to the same number, with its diagnostics and an explicit retry path.
Never automatically remove worktrees, branches, dependency clones or user files.

**Process lifetime and progress.** Preparation runs outside the UI event loop,
reports human-readable progress, and supports cancellation. Cancellation stops
and joins owned command processes; if outcome remains uncertain, persist that
uncertainty and reconcile before another mutation. Do not parse Weave diagnostic
strings into state. Design the exact wait/timeout limits and bounded diagnostic
storage in the implementation plan. Existing ready resume performs only local
validation. Long clone/build work is allowed to take visibly longer than that.

**Retention.** Workspaces and dependency clones intentionally persist per slot,
not per launch. One current transaction/readiness record and stable lock files
per slot replace prior generations instead of accumulating attempt history.
Temporary files have owned cleanup; diagnostic retention is bounded. Explicit
environment removal ends their lifetime; this issue adds no automatic deletion
policy (ARCH-FUNERAL).

**Verification.** Pure tests cover number selection and transitions. Stateful
fakes behind the same process/storage seams model refs, worktree membership,
filesystem contents, command outcomes and setup phases; deterministic barriers
exercise concurrency and caller death. Real temporary Git fixtures verify
non-origin remotes, paths with spaces, hyphenated repo names, branch/path
collisions, upstream selection and interrupted creation. Conformance to the
actual SDLC v2 output and Weave setup contract is checked separately from fake
integration. Production dispatcher tests prove prepare/ready/retry behavior and
that failure launches no agent. Existing primary startup remains covered.

The implementation plan must name exact files and seams, the transition table,
locks/process lifetime, metadata validation, resource limits, and test commands
before code changes. This is larger than the quick-flow shell.

## Done when

- Initial host HEAD and main-slotN equal the captured fetched remote/main SHA, with the configured upstream recorded.
- Interrupted retries retain the recorded remote baseline; ready workspace reuse performs no implicit refresh.
- Tests make local HEAD differ from remote main and include dirty/untracked source work: provisioning uses remote main and preserves all local refs/files without committing, stashing or transferring them.
- Primary plus :1/:2 provision at the exact conventional paths with correct resting branches and configured upstreams.
- Repeated provisioning/resume preserves a dirty active branch and does not fetch/reset an existing workspace implicitly.
- Simultaneous requests cannot create duplicate workspace identities; unrelated path/branch collisions are refused.
- Interrupted Git/dependency provisioning has a tested retry path and never destroys pre-existing content.
- Fixtures cover missing remotes, non-origin remotes, repo names containing hyphens, and paths with spaces.

- Exact paths are `/workspace/worktree/<repo>-slotN/<repo>` with separate sibling dependency clones for :1 and :2; dependency origin/main initialization follows the dependency contract; host initialization uses fetched configured-remote main.
- Repeated provision/resume and interrupted setup preserve both main-worktree and dependency-clone work, without creating extra dependency threads or changing shared-tool supply.

## Plan

Engineering plan: [000305-slots-v2-workspace-provisioning-plan.md](../plans/000305-slots-v2-workspace-provisioning-plan.md).
Product direction is agreed; the detailed plan passed fresh review and awaits operator approval.

- [ ] Implement checked identity transport, request grammar and pure selection/transition model.
- [ ] Implement durable evidence, inherited leases and cancellable process execution.
- [ ] Implement and verify host creation, setup, reuse and explicit retry with real Git conformance.
- [ ] Wire the internal operation, production runtime, progress and result rendering.
- [ ] Document the contract for #306, run verification, and close through one review boundary.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

### 2026-09-23 — prerequisites verified and design started

Ariadne #242/#243 are published; the current atlas contracts specify nested
host worktrees, JSON v2 and private ordinary dependency clones. Claimed #305 and
ran start-plan. Initial claim publication encountered newer origin/main; fetched
and merged the remote metadata prerequisite while preserving existing local
edits, then published the claim with `sdlc issue sync --issue 305 --push`.
Rebuilt the existing Ariadne supplier's sdlc/weave binaries (no supplier change);
`sdlc workspace --json` now reports schema_version 2. Proposed the provisioning
boundary above for review; no implementation has started.

### 2026-09-23 — proposed spec reviewed

Fresh-context review approved the proposed scope and ownership split. The detailed
plan must make readiness invalidation concrete when host/dependency checkouts
are removed or replaced, without rejecting normal branch changes or dirty work.
It must also specify the caller-held occupancy/allocation exclusion spanning
number selection and provisioning for #306. Startup exploration confirmed the
internal DirectStoreExecutor route; PrepareStart remains free of setup effects.
Issue schema and diff whitespace checks passed. Awaiting operator design review.

### 2026-09-23 — engineering plan drafted

The operator requested continuing after agreeing remote-main initialization.
Ran start-plan and wrote the durable engineering plan. Startup exploration
confirmed DirectStoreExecutor is the suitable internal operation boundary.
Weave has no source-freshness cache: the plan defines ready as initial setup
completion, with host/dependency-instance validation and explicit recompilation
for later source changes. Pair and Weave each retain their own inherited leases;
Pair must not assume its descriptor propagates through all Weave descendants.
Plan review and operator approval precede implementation.

### 2026-09-23 — engineering plan review complete

Fresh plan review found and resolved two gaps: shared remote-tracking fetch/read
races and interrupted upstream configuration after branch creation. The revised
plan captures the fetched SHA through an attempt-owned ref and separately models
absent/matching/conflicting upstream keys on retry. The second review approved
the plan. A temporary real-Git probe confirmed atomic dual-ref fetching and that
a later slot fetch preserves the first attempt's captured SHA. Issue/project
schemas and diff whitespace checks pass. No production code has changed;
operator approval is the remaining checkpoint before change-code.

## Revisions

### 2026-09-23 — Provision nested environments with private ordinary clones

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — proposed operation and recovery contract

Reason: prerequisites are delivered and the operator requested starting #305.
Delta: added the proposed provisioning design, including explicit numbered
preparation, #306 integration boundary, reuse policy, evidence/lease ownership,
and test expectations. Awaiting design approval; prior requirements remain.

### 2026-09-23 — new slots inherit committed local work

Reason: the operator clarified that relevant local edits should be committed
and carried into newly created slots. Delta: replaced the earlier fetched-main
host initialization with the selected source workspace's accepted local SHA.
Remote/main remains the upstream; dependency clones retain their separate remote
initialization contract. Added source provenance, drift/retry rules and acceptance
cases. Earlier review approval predates this correction; detailed design remains
subject to the normal review/approval gates.

### 2026-09-23 — restore remote-main provisioning

Reason: operator chose predictable provisioning because Couch cannot judge which
local edits should be committed or transferred. Delta: supersedes the preceding
local-source initialization revision. New hosts start from captured fetched
remote/main; source-workspace/accepted-local-SHA provisioning inputs are removed.
Local changes remain untouched. Bringing committed work over after slot startup
is an explicit operator/agent action on an issue branch, owned by ariadne#245.
Updated acceptance and retry evidence to use the remote baseline.

### 2026-09-23 — detailed implementation plan

Reason: operator requested continuing #305 on the settled provisioning contract.
Delta: replaced the initial three-step task outline with the durable plan and
five concrete execution checkpoints, retaining one issue-close review boundary.
Added explicit readiness meaning, subprocess/lease ownership, recovery proofs,
resource bounds and verification commands in the plan. No code changes yet.
