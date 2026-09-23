# Slots v2 thread lifecycle implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to
> select execution/delegation; use superpowers-executing-plans for session-warm
> integration work. Follow TDD and the checkboxes below.

**Goal:** Run independently addressable primary and numbered workspace threads,
with repository-wide parked admission and repeatable setup on cold launches.

**Architecture:** Keep Pair's opaque thread address and lifecycle. Persist stable
workspace association, derive admission from the existing classifier, and reserve
with the existing start claim in one checked store transaction. Run #305 readiness
outside the store lock and revalidate before releasing the launch helper.

**Tech Stack:** Go, existing journaled ThreadStore, SDLC workspace JSON v2, Git,
WorkspaceReadiness, Pair blocked helper, stateful test doubles and real temp repos.

**Status:** Spec and plan reviews approved; awaiting operator approval before
change-code/implementation.
**Issue:** `workshop/issues/000306-slots-v2-thread-lifecycle.md`.
**Flow:** Full: expected change exceeds the 100-line quick-flow code limit.
One atomic delivery and one close review, no milestone labels. Estimate follows
plan-quality approval. Work in the primary checkout on an in-place branch.

## Chunk 1: identity, admission and launch integration

## Scope and alternatives

1. **Recommended:** extend ThreadStartClaim and the existing store journal. This
   provides crash recovery and collision control without another reservation owner.
2. Hold the store lock through Weave: fewer steps, but it would block park/resume,
   inventory and unrelated repositories for up to 20 minutes. Reject.
3. Add a reservation table/file and expiry: duplicates existing claim ownership,
   reconciliation and cleanup. Reject.

New creation uses the existing start operation/form. Paths select their workspace
when vacant, otherwise an available numbered host in the same repository. Explicit
`repo:N`/`:N` requests pin a workspace. Preview must display and bind the selected
address/path; changed selection requires a new preview, never silent renumbering.
Opening an existing workspace selects its unique usable thread; ambiguous legacy
rows require explicit tag activation. Opening an empty workspace uses normal fresh
admission. Existing live switching is unaffected by a parked sibling.

Group rendering is #307; preferences UI/inheritance changes are #308. This task
exposes workspace identity/address in summaries and reference matching, and shows
the selected destination in the start preview. Existing path preferences remain
keyed to the actual target path. No preferred-model configuration is added.

## Core concepts

All new paths below are relative to `cmd/internal/` unless otherwise stated.
Tables name intended symbols; reconcile them against delivered code before close.

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| WorkspaceBinding | threadrecord/workspace.go | new |
| WorkspaceReference / ParseWorkspaceReference | couchcore/workspaceref.go | new |
| WorkspaceAdmission / DecideWorkspaceAdmission | couchcore/workspaceadmission.go | new |
| WorkspaceCandidate / SelectWorkspaceNumber | couchcore/provision_select.go | modified |
| StartResolution / StartResolutionFingerprint | couchcore/startresolution.go | modified |
| ThreadRecord / ThreadSummary / ActionableThreadSummary | couchcore/thread.go, threadinventory.go, actionableinventory.go | modified |
| ThreadReferenceFields / MatchThreadReferenceFields | couchcore/threadmetadata.go | modified |

WorkspaceBinding contains physical common Git directory, primary root, worktree
root and nullable slot number. Slot zero is primary; absence is an ordinary
unaddressed worktree/dependency checkout. No branch, HEAD, setup phase or copied
Weave dependency list is stored. One binding per thread, many threads over time
per workspace, many workspaces per repository. Park and continuation retain it;
archive retains it with the record. Expose it by alias/conversion rather than
maintaining multiple structural validators. Old records omit it; strict decoding
accepts absence and rejects malformed bindings. Old binaries cannot read the new
field, matching existing forward-only optional-field compatibility.

References are `repo`, `repo:0`, `repo:N`, and `:N` with caller repo context.
Numbers are canonical nonnegative decimal, bounded by platform int; malformed,
negative, overflowing and leading-zero forms refuse. Opaque tags retain exact
precedence. Colon syntax is parsed before fuzzy name/path matching. Qualified
names resolve within the caller's fleet; multiple same-name repo candidates refuse.
Actual paths remain usable. Hidden Pair scopes stay hashes of worktree root, never
changed to common-Git identity; native bindings and artifacts must keep their keys.

Admission takes a typed candidate and classified rows plus raw transaction state.
It returns admitted destination or a structured refusal with exact blocking tags,
workspace addresses and available activation/inspection actions. Selection is
pure: use the requested workspace if free; for automatic overflow use existing
SelectWorkspaceNumber (lowest known free host, otherwise lowest unused number).
A host without setup success is still reusable if Git identity is valid. Unknown
or partial hosts refuse automatic allocation; explicitly opening that number calls
Ensure to recover only what it can prove. Never skip :1's partial setup to make :2.

StartResolution carries the original request and selected workspace/path separately.
Fingerprint covers target binding/path, original selection mode, profile and existing
preference/default inputs. CommitArgs reproduces the original request, not a guessed
path that does not exist yet. An empty future host reads saved target preferences
and primary defaults for preview; after Ensure, re-resolve target defaults/profile.
If composition changes the accepted result, keep prepared directory, release claim,
and ask for a fresh preview rather than launch with different parameters.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| WorkspaceCatalog / OSWorkspaceCatalog | couchcore/workspacecatalog.go | new | SDLC identity, registered worktrees, physical paths |
| SnapshotForAdmission / CreateAdmittedStart | couchcore/workspaceclaim.go | new | existing ThreadStore lock/journal + snapshot validation |
| AllocateThreadTag | couchcore/threadtag.go | modified | artifact collision claim + admitted store create |
| prepareTrackedWorkspace | couchcore/workspacelaunch.go | new | WorkspaceReadiness and final cold-start validation |
| FakeWorkspaceCatalog | couchcore/workspacecatalog_fake.go | new | mutable repository/workspace observations |
| FakeWorkspaceReadiness | couchcore/workspacelaunch_test.go | new | controlled setup outcome/order and marker state |
| PrepareStart / spawnResolved / StartInteractive | couchcore/couch.go, startup.go | modified | all fresh-start routes |
| launchTrackedThread / ResumeContextWith | couchcore/launch_existing.go, resume.go | modified | cold readiness and final binding proof |

Catalog reads reuse ProvisionIO's bounded process seam and ParseWorkspaceIdentity.
Expose/reuse the provisioner's identity reader instead of another SDLC JSON parser.
Enumerate registered worktrees with Git's machine-readable format; derive hidden
scope keys for every registered path, including non-numbered worktrees. Inspect
conventional slot directories for partial collisions; do not scan their dependency
contents or create threads. A disappeared registered host or contradictory path
is uncertainty, not a free slot. Brain/ordinary unaddressed roots retain existing
exact-workspace starts and receive no automatic numbered provisioning.

Legacy association is observed from physical StartingPath/WorkingPath and registered
scope keys, not from display names or the latest incarnation. Missing/unreadable
records in a known repo scope block that repo; scopes that cannot be associated
must not be guessed into another repo. Preserve current same-scope refusal. Persist
verified binding lazily during that record's next claim, keeping tag/scope intact.
Existing multiple threads in one workspace remain visible/activatable by tag;
fresh creation refuses that workspace until its occupancy is unambiguous.

The catalog fake models repo membership, host existence, partial/conflicting paths,
identity errors and rebinding. Readiness fake models setup success/failure and a
controllable pause; it changes durable marker state and counts effects. Reuse
FakeRunner, FakeProcOps, FakeThreadArtifactCollisionChecker, real temporary
ThreadStore and ProvisionFixture rather than new process/session facades.

## Admission and transaction sequence

| Observation/event | Fresh creation | Existing-thread action |
| --- | --- | --- |
| Live/detached thread | occupies its workspace; another slot may be selected | switch/warm attach permitted |
| Any parked row in same common Git repo | refuse, list every blocker | explicit resume/continuation permitted |
| Start claim or raw reservation | occupies its workspace; same target refuses | existing claim/CAS guards apply |
| Open park/continuation transaction | refuse competing creation requiring uncertain state | existing recovery rules apply |
| Unknown ownership / unreadable relevant record | refuse with inspection path | existing authority decides recovery |
| Positively absent and unresumable debris | existing cleanup permits reuse | archive remains explicit |
| Record set changes before admission commit | refuse stale observation before setup | same-address revision check |
| Setup fails/cancels before helper | roll back only this claim; leave files | retain conversation/park/profile |
| Park appears while fresh setup runs | final admission refuses before helper release | resume itself stays allowed |
| Supervisor dies during setup | existing dead-owner reconciliation clears claim | existing record survives |
| Helper release/registration outcome uncertain | existing post-ack reconciliation; no new launch | same |

1. Resolve request and preview. Gather relevant store snapshot and lifecycle
   evidence outside locks. Widen startupAsks to prove all threads in the candidate
   repo plus existing layout-conflict consumers; reuse ClassifyThread.
2. Generate opaque tag and acquire its existing artifact claim. Under ThreadStore's
   existing lock re-read the observed manifest/records, compare exact bytes (not
   only Generation: ordinary record writes only increment Revision), apply the
   pure admission decision, and journal the new record already carrying its
   ThreadStartClaim. No invisible Reservation-only intermediate row. Release
   artifact claim on refusal. Existing-thread claim keeps current CAS/shape guards.
3. Release the store lock. Run numbered readiness with PrimaryRoot and Slot; keep
   the durable start claim visible as busy. No process lock or store lock spans
   Weave. Primary and unaddressed worktrees skip readiness.
4. Resolve target identity/profile again. For fresh creation re-gather repo evidence,
   excluding only this exact nonce; under the store lock verify the new snapshot
   and that its claim is still owned, then authorize launch. This is the final
   admission linearization point: later parks are ordered after this start.
   On drift/refusal release this claim; no automatic selection/setup loop.
5. After slow setup, recheck the exact cold native binding or retained continuation
   authority and worktree identity before child effects. Refactor the current
   ResumeContextWith pre-launch check into the shared preparation boundary so
   compile never sits after the final proof. Warm session proof remains unchanged.
6. Use existing launchTrackedThread/blocked-helper/StartHelperRecorded/registration
   transitions and cleanup. Only confirmed registration clears park evidence.

Atomic admission covers Spawn, SpawnPrepared and StartInteractive; no production
path can directly allocate-and-launch around CreateAdmittedStart. Same-address
resume/continuation/switch-agent keep their own lifecycle permission and occupy
the same workspace. Store locks are short filesystem transactions; no additional
background task, expiry timestamp, capability file or scheduler is introduced.

## Constraints, trust and residue

Interactive workload: a workstation with tens of workspaces and up to roughly
100 active/history-visible thread rows (design assumption). Catalog work is O(repo
worktrees + relevant records), not one subprocess per historical incarnation.
Use one repo discovery per operation and a shared session inventory; no polling.
Local probes retain 5s limits, fetch 120s, Weave 20min, and existing 10s helper /
15s registration budgets. Setup time is outside registration's clock. Snapshot
contention returns an actionable retry, never unbounded waiting/retry churn.
Cancellation joins owned setup processes and invokes existing start rollback.
Other repos remain operable while Weave runs; tests pause one setup to prove this.

Parse SDLC/Git/record inputs at the boundary, validate physical association, and
fail closed on changed identities. No path based on a display name alone grants
launch authority. No new credentials or network services; transport inherits #305.
All tests use temp stores/data roots and synthetic/local origins. Never launch
agents against the operator's actual checkouts for automated acceptance.

The optional binding is bounded metadata in existing records; existing archive
and storage-GC policies own its end. Start claim lifetime/recovery is unchanged.
Snapshot bytes and admission values die with each operation. Numbered directories
remain operator-owned persistent workspaces under #305's explicit removal policy.
No metadata is written into sibling dependencies by this task.

Architecture: ARCH-DRY reuses lifecycle/claims/identity parsing; ARCH-PURE owns
selection and admission in deterministic functions; ARCH-PURPOSE sweeps every
launch route; ARCH-MOCK supplies stateful catalog/setup doubles and live contract
checks; ARCH-CONSTRAINTS bounds setup and snapshot contention; ARCH-SECURE validates
external identity and strict persisted schema; ARCH-ORDER tests competing events
at observed/claimed/setup/final-admitted/helper/registered boundaries; ARCH-FUNERAL
adds no durable artifact family.

## Tasks and verification

### Task 1 — stable workspace identity and address resolution

Files: create the workspace binding/reference/catalog files above and colocated
tests; modify threadrecord/record.go, couchcore/thread.go, threadinventory.go,
actionableinventory.go, threadmetadata.go, startresolution.go and their tests.

- [ ] Add failing parser/roundtrip/reference tests: :0/:1/repo:N, ambiguous repo
  names, bad numbers, subdirectories/symlinks, strict record decoding, absent legacy
  binding, dependencies lacking numbered identity, and retained opaque scope/tag.
- [ ] Implement the pure types and catalog seam. Prove legacy parked membership
  without incarnations and missing-path conservatism against temp Git worktrees.
- [ ] Make start preview bind original request and chosen target; verify no setup
  effects during preview and stale selection/profile refusal on commit.
- [ ] Run focused tests, reconcile symbol names, and commit this coherent layer.

### Task 2 — atomic fresh admission using existing start claims

Files: new couchcore/workspaceadmission.go, workspaceclaim.go and colocated tests;
modify couch.go, threadtag.go, threadstore.go and startup.go/startup_proof_test.go.
Extract only admission/snapshot helpers from large files; no unrelated refactor.

- [ ] Write failing production-boundary tests for parked :0/:1/:2, every parked
  producer, multiple blockers, other repo independence and subdirectory occupancy.
- [ ] Write deterministic interleavings using pause channels: two starts choose
  the same target, park between evidence/claim, raw reservation, unreadable record,
  and record-only Revision mutation without Generation change.
- [ ] Implement checked snapshot plus admitted create as one journal transaction;
  make all fresh-start routes consume it. Losing requests leave no artifact claim,
  invisible row, provision call or child. Remove obsolete path-only admission.
- [ ] Update narrowed/full startup-proof equivalence oracle for repository readers.
  Run focused tests/race, inspect diff, and commit.

### Task 3 — readiness across cold lifecycle and exact workspace opening

Files: new couchcore/workspacelaunch.go and tests; modify launch_existing.go,
resume.go, continuation.go, switchagent.go, startup.go, operationdispatch.go,
ops.go, couchcmd/run.go and couchtty/menu.go/menu_render.go where wiring requires.

- [ ] Add failing boundary tests for readiness before fresh/cold/fresh-existing
  launches, warm/primary bypass, cancellation, failed readiness retry, and binding
  replacement during compile. Readiness must occur while the owned claim is busy.
- [ ] Add blocked-setup tests: competing same-slot creation refuses, another repo
  progresses, park during setup blocks fresh child release, a park after final
  admission does not retroactively cancel the admitted launch, dead-owner recovery
  preserves files and releases only the abandoned claim.
- [ ] Implement shared preparation before final binding/continuation proof; retain
  existing helper acknowledgment/registration semantics and error diagnostics.
- [ ] Wire precise addresses and preview destination into existing operations/form.
  Opening an exact workspace resumes its unique thread; multiple legacy matches
  give tag-based activation guidance. No arbitrary parked-row auto-selection.
- [ ] Test Spawn, SpawnPrepared, StartInteractive and operation/TUI dispatch—not
  only helper functions—then remove SelectWorkspaceNumber's #306 dead-code waiver.
- [ ] Run targeted lifecycle suites and race tests; commit.

### Task 4 — lifecycle acceptance, docs and publication

Files: new couchcore/workspace_lifecycle_test.go and workspace_conformance_test.go;
update atlas/workspace-provisioning.md, atlas/couch.md, atlas/index.md, README.md,
this issue/plan, workshop/projects/couch-slots-v2.md and affected inventory fixtures.

- [ ] In isolated temp Git environments run primary + :1 + :2 concurrently through
  stateful production startup, independent park/resume, continuation, archive and
  explicit same-workspace replacement. Assert distinct native bindings/opaque tags.
- [ ] Before/after lifecycle operations compare host and private dependency dirty,
  untracked files, branch refs and local commit SHAs. No Git reset/switch/remove is
  a lifecycle effect; dependency discovery creates no extra thread rows/blockers.
- [ ] Live conformance: use real installed SDLC/Weave with isolated local Git
  transport via ProvisionFixture; fake the agent launcher. Run whenever consumed
  identity/setup contracts change and in #309. Document one optional real Couch
  operator smoke of :0/:1/:2 addressing/park/refusal/resume for #309.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore
  ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1` (all pass), targeted
  race tests on new concurrent sequences, and `go vet` on changed packages.
- [ ] Run `make runtimebundle-generate`, `go test ./... -count=1`, then
  `make pair bin/couch`. Run opt-in real SDLC/Weave conformance and isolated built
  CLI reference/diagnostic smoke. All commands must exit zero.
- [ ] Update operator help/README/atlas for new addresses, parked diagnostics,
  existing-workspace replacement and cold readiness. Sweep obsolete one-thread-
  per-path prose; preserve #307 grouping and #308 preference scope.
- [ ] Record verification and checkpoint issue/plan; `sdlc close --issue 306
  --verified '<observed evidence>'` owns the fresh review. Fix blocking findings.
- [ ] Commit close records with review trailers, `sdlc pr`, `sdlc merge --yes`;
  update project completion and restore unrelated edits. No extra review outside
  the SDLC boundary. Final delivery reports actual tests and any unrun live smoke.

## Revisions

### 2026-09-23 — initial implementation proposal

Derived from #306's agreed scope and live code exploration after #305 merged.
Reuse the existing claim and journal; make workspace association durable and
apply repo-wide policy at fresh creation. Pending spec/plan review and operator
approval; no code or estimate has been produced.

### 2026-09-23 — spec and plan review approved

Fresh-context spec review and subsequent plan review found no blocking issues.
Added the complementary ordering test: a park after final admission does not
retroactively cancel the admitted start. Awaiting operator design approval under
AGENTS.md §2 and the brainstorming/writing-plans skills.
