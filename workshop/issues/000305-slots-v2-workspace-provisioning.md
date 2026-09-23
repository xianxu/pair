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

Provision :1, :2, and subsequent numbered workspaces at ../worktree/repo-slotN relative to the primary checkout. Initially create main-slotN at the selected source workspace’s local commit, retaining configured-remote main as its upstream. Existing workspaces retain their current branch and files; provisioning/resuming is not refresh. Use the shared Ariadne identity contract and dependency setup from the prerequisite tasks.

Specify number allocation and reuse under concurrency, distinguishing available persistent workspaces from occupied threads. Never renumber surviving addresses. Validate existing paths and branches rather than taking them over. Serialize competing provisioning requests and recover from partially created worktree/dependency setup without deleting user work. The repo-wide parked-thread admission rule is owned by the lifecycle task. ARCH-DRY: one provisioning path for every UI entry point; ARCH-FUNERAL: directories persist across thread replacement and issue landing.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

The agreed numbered main-worktree path is `../worktree/<repo>-slotN/<repo>` relative to the primary, for example `/workspace/worktree/pair-slot1/pair`. The enclosing `pair-slot1/` directory holds sibling dependency clones such as `ariadne/`, so existing `../ariadne` declarations stay meaningful. This supersedes the flat worktree path above. The main checkout remains a Git worktree on main-slotN; dependencies are ordinary independent clones, initially from each recorded origin/main, through ariadne#243's setup contract. No local-source dependency worktrees or primary-checkout symlinks.

Existing dependency clones retain selected revisions, dirty files and local commits during provisioning/resume. Allocation and recovery distinguish an enclosing directory, the registered main worktree, and partially acquired dependencies; never take over unrelated content. No dependency clone automatically receives a Couch slot/thread entry. Primary repos retain their ordinary sibling environment and direct UI access; numbered environments are accessed through their main thread.

### Proposed provisioning design — 2026-09-23 (awaiting approval)

**Boundary.** Add one reusable workspace-provisioning operation to Couch's
existing operation/dispatch system. Its explicit inputs identify a source workspace
(default :0), the accepted local source commit, and a positive destination slot
number. Resolve the primary repo from that source for layout and identity. It prepares a working directory and returns its
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

**Create from local work.** The operator/agent reviews local changes, commits
those relevant to the intended work, and selects the source workspace's resulting
commit before provisioning. The source defaults to :0; an explicit source such
as :1 selects that workspace. Capture and record its address and full SHA. The
new main-slotN starts at exactly that commit, including unpublished local work.
Git carries the whole committed snapshot and its ancestry; relevance is decided
when preparing the commit, not by Couch extracting selected issue files. The
existing clean-source readiness requirement applies before accepting the source.

Select the configured remote/main upstream separately from the starting commit;
an explicit remote may resolve ambiguity. Fail visibly if no unique upstream
can be established. Fetch only if needed to establish tracking metadata; fetched
main never replaces the accepted local starting SHA. Revalidate the source and
accepted SHA before the first creation effect. If the source changed, refuse
and re-preview rather than silently taking its newer commit. Once reserved,
retries retain the recorded source SHA and reconcile existing creation effects.
Create main-slotN at that SHA, set its remote/main upstream, and create the
registered nested worktree. Source refs/files remain unchanged by provisioning.

Invoke `weave compile` with default targets from the host worktree. Publish ready
only after exit 0 and final identity validation. Dependency origins are governed
by Weave: private ordinary sibling clones still initialize from their recorded
origin/main. Local-source initialization here concerns the host repo; it does
not add local dependency transfer or create dependency threads/preferences.

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

- A source commit containing relevant unpublished local changes becomes the exact initial host HEAD and main-slotN SHA; remote/main tracking is configured independently.
- The source address/SHA is recorded, source drift before creation is refused, and retries preserve the original accepted SHA after reservation.
- Tests distinguish local source HEAD from remote main, prove the local changes arrive, and prove no source ref/files are altered by provisioning.
- Primary plus :1/:2 provision at the exact conventional paths with correct resting branches and configured upstreams.
- Repeated provisioning/resume preserves a dirty active branch and does not fetch/reset an existing workspace implicitly.
- Simultaneous requests cannot create duplicate workspace identities; unrelated path/branch collisions are refused.
- Interrupted Git/dependency provisioning has a tested retry path and never destroys pre-existing content.
- Fixtures cover missing remotes, non-origin remotes, repo names containing hyphens, and paths with spaces.

- Exact paths are `/workspace/worktree/<repo>-slotN/<repo>` with separate sibling dependency clones for :1 and :2; dependency origin/main initialization follows the dependency contract; host initialization uses the accepted local source SHA.
- Repeated provision/resume and interrupted setup preserve both main-worktree and dependency-clone work, without creating extra dependency threads or changing shared-tool supply.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Specify allocation/reuse and reconcile existing Couch startup with the shared workspace contract.
- [ ] Implement provisioning and recovery behind the existing startup boundary with Git fixtures.
- [ ] Verify repeat setup and publish the lifecycle integration contract.

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
