---
id: 000305
status: done
deps: [ariadne#242, ariadne#243]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours: 6.95
started: 2026-09-23T10:54:57-07:00
flow: {kind: full, provenance: operator}
actual_hours: 4.66
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
#306 calls this readiness operation before launching/opening a numbered workspace
or cold-resuming a parked thread. Reattachment to a still-running agent reconnects
without setup; primary :0 behavior stays unchanged. Errors return visibly; the
next ordinary invocation recovers without an automatic retry loop.
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
readiness record can be inspected; the same readiness operation prepares dependencies before marking it ready. Do not infer setup success from directory
existence. A missing or externally changed workspace must be revalidated rather
than trusted because it has a saved receipt.

**Ownership and recovery.** One repository-wide OS lock serializes host Git
verification, fetch/capture and creation; direct Git children inherit it. Release
it before Weave setup. Keep a small creation-intent record with the captured SHA
and ownership evidence to reconcile interrupted Git operations. Git/filesystem
facts determine the next step; there is no persisted provisioning phase machine.
Refuse foreign paths/refs and unverifiable partial creation without overwriting
work. Never automatically remove worktrees, branches or dependency clones.

**Setup success and retry.** Store one success marker in the host's Git
administrative directory only after Weave exits 0 and host validation succeeds.
A valid marker skips compilation. Without one, the same invocation runs `weave compile` for new or existing
verified hosts. No retry flag or separate retry operation is needed. Weave owns dependency locking,
partial setup and retry recovery. Couch keeps no dependency inventory, nonce
bindings or setup phases. A crash between compile success and marker publication
simply causes another compile. The marker records initial setup success, not
ongoing build freshness; later dependency/source changes need explicit Weave/build.

**Process lifetime and progress.** Run preparation outside the UI event loop,
stream diagnostics, and support bounded cancellation of owned processes. Weave's
own inherited lock protects any surviving setup descendants. No persisted
uncertain-outcome state or diagnostic history is needed: absence of the success
marker remains sufficient evidence to rerun safe setup on the next ordinary invocation.

**Retention.** One host-creation lock per repo; at most one small intent and one
success marker per slot. Remove owned intent after success, or retry that bounded
cleanup later. Keep diagnostic tails in memory only. Worktrees and clones persist
until explicit removal (ARCH-FUNERAL).

**Verification.** Test pure number selection and a small host decision table.
Stateful process/storage fakes cover host refs, membership, files, locks and
command outcomes without simulating dependency internals. Real Git fixtures
cover collisions, captured remote baselines, interrupted creation and upstream
configuration. Check real SDLC v2 and Weave conformance separately. Test failed
compile, lost success publication, repeat compile, and ready reuse through the
production dispatcher; provisioning launches no agent.

The durable plan names files, seams, bounded processes and test commands.
This remains larger than the quick-flow shell.

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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source;
figures are provisional ship wall-clock hours, not a deadline.

The first estimate lumped independently risky host/procedure concerns into single
modules. Estimate-quality requested finer decomposition and explicit integration
verification time. The revised breakdown uses the same calibrated table values:

| Concern | Primitive | Base design → adjusted | Base impl → v3.1 |
| --- | --- | ---: | ---: |
| Identity/request/selection parsing | greenfield-go-module | 1 × library .5 × spec .2 = .10 | .6 × .4 = .24 |
| Fetch/remote capture and selection | greenfield-go-module | 2 × spec .2 = .40 | .8 × .4 = .32 |
| Host branch/worktree creation | greenfield-go-module | 2 × spec .2 = .40 | .8 × .4 = .32 |
| Interrupted host reconciliation | greenfield-go-module | 2 × spec .2 = .40 | .8 × .4 = .32 |
| Bounded subprocess execution | greenfield-go-module | 1 × library .5 × spec .2 = .10 | .8 × .4 = .32 |
| Inherited creation lease | greenfield-go-module | 1 × library .5 × spec .2 = .10 | .8 × .4 = .32 |
| Intent/marker storage | greenfield-go-module | 1 × library .5 × spec .2 = .10 | .8 × .4 = .32 |
| Weave readiness orchestration | api-integration | 2 × spec .2 = .40 | 1.5 × .4 = .60 |
| Typed dispatcher integration | smaller-go-module | .2 × spec .2 = .04 | .5 × .4 = .20 |
| CLI runtime/progress/JSON integration | smaller-go-module | .2 × spec .2 = .04 | .5 × .4 = .20 |
| Independent live SDLC/Weave conformance | api-integration | 1 × spec .2 = .20 | 1.5 × .4 = .60 |
| Full/race suites, builds and isolated smoke | real-api-discovery | 0 | .6 × .4 = .24 |
| Atlas documentation | atlas-docs | .1 × spec .2 = .02 | .2 × .4 = .08 |
| One issue-close review | milestone-review | .1 × spec .2 = .02 | .5 × .4 = .20 |

Library check: existing strictjson/atomic-file helpers, os/exec and unix shorten
transport/storage/process design. No library supplies host ownership policy.
Colocated tests belong to their modules; live conformance is separate executable
integration against actual Ariadne binaries, not another unit-test allowance.
Familiarity 1.0 for this Go/Git code; thorough plan gives 15% design buffer.
Total = 2.32 × 1.15 + 4.28 = 6.948, rounded to 6.95 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.24
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: api-integration design=0.20 impl=0.60
item: real-api-discovery design=0.00 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 6.95
```

## Plan

Engineering plan: [000305-slots-v2-workspace-provisioning-plan.md](../plans/000305-slots-v2-workspace-provisioning-plan.md).
The operator approved implementation; code is implemented and final verification/close is underway.

- [x] Implement checked identity transport, request grammar and pure selection/host decision table.
- [x] Implement host creation intent, one Git creation lock and cancellable process execution.
- [x] Implement and verify host creation, setup, reuse and repeated invocation with real Git conformance.
- [x] Wire the internal operation, production runtime, progress and result rendering.
- [x] Document the contract for #306 and run verification; one SDLC close review follows.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

### 2026-09-23 — prerequisites verified and design started
- 2026-09-23: closed — Full go test ./... -count=1 passed independently by author and second reviewer; targeted race/vet/build/live SDLC-Weave/CLI smoke and parser fuzz passed. No code changes since ea2b507f; BR-3 docs corrected, BR-1/BR-2 disputed with executable same-slot/no-thread evidence in issue Log.; review verdict: SHIP

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

### 2026-09-23 — simplify setup recovery

The operator requested a simpler design after confirming repeatable Weave setup.
Replaced dependency inventory and provisioning phases with one setup-success
marker. A missing marker permits explicit compile retry. One repo lock covers
host Git creation only; Weave owns setup exclusion. Retained small creation
intent for safely reconciling interrupted Git effects. Prior review applies to
the superseded design; no implementation has started.

### 2026-09-23 — simplified plan reviewed

Fresh review accepted the simplified architecture with two bounded corrections:
serialize success publication and temporary cleanup under the existing lock,
and define an external host's first baseline from its resting-branch tip.
Both corrections and focused tests are in the plan. Issue/project schema checks
and diff whitespace checks pass. Implementation awaits operator approval.

### 2026-09-23 — implementation and focused verification

Passed change-code and created the in-place implementation branch. Plan-quality
accepted at the configured round cap with the disputed #306 reverse-dependency
finding retained; the operation requires no thread reservation or launch API.
Estimate-quality accepted 6.95 calibrated ship hours. Implemented repeatable
provisioning through the internal dispatcher, remote-main capture from fetch
porcelain, Git-owned partial recovery, one host creation lock and one setup marker.

Verification so far: original couchcore/couchcmd baseline passed; focused tests,
real Git recovery and race tests passed; actual SDLC/Weave conformance passed.
Built Pair and Couch (correct target: make pair bin/couch). Isolated built-CLI
smoke returned created → reused → prepared after marker removal, preserving local
primary/slot files and the feature branch. Full-suite inventory guards exposed
missing registrations for new files; those were corrected. Final suite/close is
still pending. No UI or thread-lifecycle behavior was changed (#306 owns that).

### 2026-09-23 — final verification before close

`go test ./... -count=1` passes after inventory registration fixes. Targeted
`go test -race` for provisioning/identity/selection/CLI and recovery tests passes;
`go vet ./cmd/internal/couchcore ./cmd/internal/couchcmd` passes. `make pair
bin/couch` succeeds. Live `TestProvisionConformance` passes with real SDLC/Weave.
Built-CLI smoke confirms created/reused/prepared with baseline and dirty work
preserved. Parser fuzz seeds and short single-worker campaigns pass. All code is
committed as 76ba5bfd; closing review now owns the remaining acceptance boundary.

### 2026-09-23 — close review correction

First boundary review passed targeted/race/live tests and all architecture
principles, with no production-code findings. It returned REWORK for BR-3: the
active plan still named provision_fake_test.go although the delivered fixture
lives in provision_git_test.go. Corrected all active entity/file mappings and
added the lesson. BR-1/BR-2 are carried copies of the disputed reverse dependency;
the review accepted the actual #305/#306 ownership split but omitted explicit
ledger dispositions. Requesting those withdrawals on re-review. No code changes.

### 2026-09-23 — reservation finding disputed with executable evidence

Second review passed its own full Go suite, targeted/race evidence and live
conformance, and disposed BR-3. It retained BR-1/BR-2 despite acknowledging no
thread/supervisor/agent effects. Requesting a different supported gate reviewer
for that technical disagreement, using the receiving-code-review skill.

The reusable boundary rule is: a readiness operation grants directory readiness,
not thread capacity. There is no token handoff across this API. Its only consumer
in #305 is DirectStoreExecutor's explicit path/slot call. Ensure never calls
SelectWorkspaceNumber, a thread store, or a launch API. Two Ensure callers for
the same slot are deliberately allowed: TestProvisionRecoveryConcurrentWeaveBusy
and TestProvisionRecoveryConcurrentPublishedMarkerWins exercise them; Git host
creation and marker publication serialize under HostCreationLease, while Weave
owns setup exclusion. TestProvisionCLI asserts zero runner/supervisor effects.
Thus missing #306 cannot produce a duplicate thread or invalidate #305's contract.
#306 already declares deps: [pair#305]. The requested reverse edge is a cycle,
not a prerequisite. BR-1/BR-2 should be withdrawn unless a concrete violated
#305 invariant can be shown. Future admission/launch invariants belong to #306.
No code changed since both independent verification runs.

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

### 2026-09-23 — simplify provisioning design

Reason: operator requested simplification around repeatable weave compile.
Delta: replaced active ownership/recovery, retention and verification design with
Git-derived host decisions, one host creation lock and a setup-success marker.
Removed Couch dependency inventory, persisted setup phases and private fetch refs;
retained bounded intent for interrupted Git creation. Historical logs remain.

### 2026-09-23 — remove retry mode

Reason: operator wants repeatable operations to reapply safe commands whenever
success is unconfirmed. Delta: readiness now runs missing setup on every ordinary
invocation, without --retry. #306 uses it before numbered-slot launch/cold resume;
warm reattachment skips setup. Repeat-call recovery is an explicit test obligation.

- 2026-09-23: Close review returned SHIP (window 570f8566..835e6a03); BR-1/BR-2 withdrawn after scope evidence, BR-3 corrected. Actual measured 4.66h. Minor advisories: Ensure rejects corrupt/conflicting observations before NextHostAction; immediate busy-lock refusal after compile intentionally requires another readiness call and may recompile. Live conformance cadence documented in atlas: provisioning/consumed-contract changes and #309 acceptance. No blocking findings remain.
