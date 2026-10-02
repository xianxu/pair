---
id: 000366
status: codecomplete
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-02
estimate_hours: 2.66
card_mirror: '2ffcf44660c682ebb422080d6477a15c256fa8a7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-01T22:14:48-07:00
flow: {kind: full, provenance: operator}
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 4.45
---

# Make Couch a local singleton

## Problem

Durable assignment should identify machine + slot without a Couch instance ID. Today namespace-scoped supervisors require an explicit singleton/migration contract before that assumption is valid.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Make Couch a local singleton with one authoritative slot inventory. Proposed boundary is one supervisor per OS user on a machine; validate and settle that boundary in design. Multiple Ariadne operators/machines may independently run their own local Couch and share issue trackers. Multi-machine Couch routing is out of scope.

Define existing-namespace migration/adoption, second-launch behavior, crash restart and slot identity preservation. Reuse supervision/lease mechanisms and ensure tests/isolated diagnostic stores have an explicit supported isolation path. Do not delete existing conversations, reset worktrees or silently adopt conflicting state.

## Done when

- The singleton boundary is explicit; concurrent starts cannot establish two production supervisors inside it.
- A second invocation attaches/routes to the existing supervisor or gives a concrete diagnostic.
- A sole legacy store adopts in place with supporting roots and identities preserved; ambiguous multiple-store installations receive an actionable preservation report, remain explicitly unmigrated, and require operator reconciliation before adoption.
- Restart and isolated-test behavior are verified; durable issue ownership requires no Couch ID.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source,
so these hours are provisional ship wall-clock, not a deadline.

Eight primitives: issue/spec dialogue (0.5 design, 0.2 raw impl), pure selection
and Manager IO (two greenfield modules, each 1.0 raw design, 0.8 raw impl), identity
inspection and CLI/runtime composition (two smaller modules, each 0.3 raw design,
0.5 raw impl), fixture/root sweep (cross-cutting, 0.3 raw design, 0.5 raw impl),
docs (0.1 raw design, 0.2 raw impl), and one review (0.1 raw design, 0.5 raw impl).
Existing flock/durablefile/strictjson supply the library shortcut for the two
new modules: design ×0.5, then thorough-spec ×0.2. All other implementation design
uses ×0.2; the already-incurred issue/spec dialogue is undiscounted. Every impl
value below is raw ×0.4 exactly once. Familiar Unix/Go work uses ×1.0 familiarity;
thorough plan uses +15% design buffer. Design subtotal 0.92; implementation 1.60;
0.92 × 1.15 + 1.60 = 2.658, rounded 2.66 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.08
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.66
```

## Plan

- [x] Implement and verify the reviewed singleton/adoption plan in `workshop/plans/000366-couch-singleton-plan.md`.
- [x] Complete production composition, isolated fixtures and migration acceptance.
- [x] Update docs/project and prepare verification evidence for the SDLC close review.

## Log

### 2026-10-01
- 2026-10-01: closed — BR-4 fixed end-to-end: parent clears stale scoped dir; launcher and embedded extraction share validated selected global root. Real Pair binary create/list/continuation-list/warm-resume checks scoped artifacts and global claim/index records; ignoring selected root mutation fails, restored passes. Full launcher race11.386s; fresh restored couchcmd race43.000s and pair-go race14.976s; installed Couch2.168s and termcmd0.453s pass. Prior post-fix singleton/core/identity/messaging/retention race suites pass, including core385.636s; BR-1/2/3 disposed addressed round2. Diff check clean. Remaining full-repo failures independently match original base: artifact inventory #348 (49 identical diagnostics) and GC archive fixture scope. No live cutover.; review verdict: SHIP

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.

## Proposed design — awaiting operator review

### Scope and alternatives

Recommend one production Couch supervisor per OS user per machine, backed by
one persistently selected store. A second launch refuses immediately and names
the selected store and verified owner when available. Connecting another terminal
to the running Console is outside this proposal.

A host-wide lock alone is smaller but allows sequential launches to switch slot
inventories. A shared daemon with multiple attached terminals solves a broader
presentation problem and is unnecessary for the project's ownership contract.
The selected-store design keeps one inventory without introducing that daemon.

### Ownership and storage

Production authority must be independent of COUCH_STORE_DIR, XDG_DATA_HOME and
an overridden HOME. Resolve the actual OS account's home from its UID through an
injected account lookup; failure refuses production startup. Keep the host lease
and selected-store record under that account's pair-host directory. Existing
store identity allocation remains compatible; mismatched legacy host authority
is a migration refusal, never a fresh counter allocation.

Acquire a non-inherited, nonblocking host lease before selecting/adopting a store,
then the existing store supervisor lease before constructing Couch, attaching a
terminal, starting a broker or launching a child. Keep per-store transaction locks.
Release the store lease before the host lease. Kernel lock ownership authorizes;
PID/start-token metadata only explains contention. No polling or PID-based takeover.

Persist the canonical physical selected-store path atomically. A selected path
that is unavailable, corrupt, or no longer canonical refuses with recovery context;
it never selects a fresh default. Ordinary production launch and inventory commands
resolve that same selection. A conflicting explicit store override refuses and
names both paths. Read-only observations do not take the lifetime supervisor lease.
Child routing continues to use the selected physical namespace. Existing C/N/M,
tags, native UUIDs, worktrees, preferences and conversation history remain intact.
C is a historical store identity, not a required field in issue assignment.

### Adoption and legacy reconciliation

Adopt the chosen existing store in place; do not copy it into the default directory.
On first use, inspect the default/requested store and registered legacy stores.
Discovery is bounded by existing registry limits, runs at adoption rather than idle,
and reports unavailable or unreadable entries as unknown. Arbitrary unregistered
custom stores cannot be discovered exhaustively; operators supply those paths.

Provide an explicit adoption command whose preview identifies the selected path,
other known stores, active legacy owners, unresolved observations and conflicting
inventories. Application revalidates the preview under the host lease and refuses
changed evidence. All older Couch supervisors must be stopped and upgraded before
cutover: older executables cannot be made to respect a new host lock. Retain the
selected store's old lease for compatibility with its existing consumers.

Fresh installation and a sole usable legacy store can select/adopt without a
separate confirmation. Multiple populated stores or unresolved ownership require
explicit reconciliation; no selection based on recency and no silent merge.
The proposed first version reports these conflicts and preserves every store.
Whether #366 must additionally merge nonconflicting legacy inventories is the
remaining scope decision for operator review; do not claim migration complete for
an unresolved multi-store installation.

Surviving wrappers belonging to the selected namespace reconnect through #365's
existing lifecycle protocol. Wrappers belonging to other stores are reported as
migration conflicts, never retargeted by changing their identity or socket path.
Crash restart reacquires the kernel lease and reads the durable store selection;
stale owner metadata does not authorize takeover or block a free kernel lease.

### Explicit isolation

Provide an explicit isolated runtime mode for tests and diagnostics with a
caller-supplied absolute root. It scopes Couch ownership, selected-store metadata,
allocation authority, inventory and sockets away from production, and propagates
that configuration to children. Production directory overrides alone do not imply
isolation. Extend tests/with-isolated-pair.sh so actual process tests isolate Pair
artifacts as well, clear inherited session path overrides and protect production
sentinels. Tests use the same runtime resolution and lease path as production.
Isolated runtimes are intentionally separate from production slot scheduling.

### Verification and operating envelope

- Concurrent production starts with distinct store/HOME/XDG overrides produce
  exactly one owner and no losing child, broker or terminal attachment.
- Same-store aliases contend; independently rooted isolated runtimes coexist;
  isolated subprocesses and descendants cannot alter production sentinels.
- Kill/restart releases ownership without deleting lock files or resetting counters;
  child processes do not inherit the lifetime lock descriptor.
- In-place adoption preserves tags, UUIDs, preferences, history, dirty worktrees and
  selected-store identity. Refused adoption changes no source inventory.
- Legacy live owners, ambiguous stores, missing/corrupt selected state and a stale
  preview each refuse visibly, preserving unknown observations.
- Production launch, inventory reads and child messaging agree on the selected
  namespace; existing delivery/reconnect and idle-no-discovery regressions stay green.

Ownership acquisition uses one nonblocking kernel-lock attempt; no waiting loop.
Steady-state operation introduces zero recurring discovery or subprocess probes.
Adoption uses bounded registry enumeration and existing observation timeouts;
exhaustion remains an explicit unresolved result. Supported platforms remain the
existing Unix targets. No remote routing, automatic takeover, daemon, or durable
message queue. ARCH-PURPOSE separates durable selection from transient ownership;
ARCH-DRY reuses leases and #365 lifecycle delivery; ARCH-MOCK requires isolated
production-path subprocess tests; ARCH-ORDER requires explicit adoption outcomes.

## Revisions

### 2026-10-01 — initial design exploration

Claimed #366 and entered planning on its own branch from fresh main. Expanded the
project requirements into the proposed design above without replacing the captured
Spec. Second-launch refusal is a recommendation pending the operator's answer;
legacy multi-store reconciliation remains an explicit scope decision. No code changed.

### 2026-10-01 — second-launch decision and baseline

Operator selected immediate refusal with a clear running-owner diagnostic for a
second launch. This settles that part of the proposal; attaching another terminal
is excluded. Multi-store migration scope is still awaiting an answer.

Existing focused supervisor/namespace/CLI lease tests passed:
`go test ./cmd/internal/couchcore ./cmd/internal/couchidentity ./cmd/internal/couchcmd -run 'Test(Supervisor|AcquireSupervisor|ResolveCouchNamespace|StartAcquires|ResumeAcquires|HeldSupervisor|OSRuntimeRefuses)' -count=1`.
The identity package had no tests matching that filter; this is baseline evidence
for existing lease behavior only, not verification of #366. A fresh-context spec
review is in progress before implementation planning.

### 2026-10-01 — spec review revisions (supersede matching proposal paragraphs)

Fresh-context review identified supporting-root drift, incomplete legacy recovery
semantics, and a mismatch between refusal-only migration and the captured acceptance.

- Persist the adopted runtime's canonical Couch store, Pair data root and existing
  allocation-authority root together as one bounded, versioned configuration. The
  production singleton lease still belongs to the real OS account, independent of
  those paths. Preserve a valid legacy authority in place even if it is under an
  older HOME; never create substitute counters to make adoption pass. All Couch
  readers, launch configuration, continuation/slug/history/session readers, traces,
  inherited child paths and messaging consume this resolved configuration. Explicit
  conflicting ambient paths refuse with the configured path and corrective action.
  A fresh installation uses the real account's default local data roots. Adoption
  previews the associated paths and accepts explicit legacy paths when discovery
  cannot establish them; missing/regressed authority requires restoring the valid
  existing authority, not re-enrollment. Standalone Pair's configuration is outside
  this change; hosted Pair receives the selected configuration explicitly.
- Authority and adopted state must reside on machine-local storage. Shared/network
  home/state directories are outside this first version's supported envelope; the
  per-machine guarantee assumes local storage and does not introduce a distributed
  filesystem lock protocol.
- The adoption preview is ephemeral. Apply reacquires host and selected-store leases,
  re-reads configuration and relevant source revisions under existing transaction
  locks, then publishes one atomic configuration file. Failure before publication
  leaves selection absent and sources untouched; lost acknowledgment is recovered
  by reading the selection, with an identical retry converging on that configuration.
  A different existing selection refuses. No source-copy or migration journal.
- Known obsolete/missing registrations stay visible in the report. An operator can
  explicitly acknowledge one as retired/outside the adopted production inventory;
  that acknowledgment neither deletes its data/identity reservation nor proves a
  process absent. Potentially live or unreadable stores remain unresolved unless
  the operator completes the stop/upgrade and disposition steps. No automatic
  inference from a failed probe. Unregistered legacy paths must be supplied.
- For ambiguous populated stores, guidance is: stop/upgrade all legacy supervisors,
  retain backups of each store and its identity authority, inspect the report and
  explicitly choose which stores remain outside the production inventory. If the
  operator needs their inventories combined, adoption stays refused until a reviewed
  consolidation is performed; #366's recommended scope includes no manual JSON
  editing recipe or automatic merge. The report labels these installations
  UNMIGRATED and identifies the paths and unresolved decisions. No work is discarded.
- Sweep every actual-process test that currently relies on HOME/XDG isolation,
  including continuation/stale-store/host-identity tests. Inject account lookup when
  testing production resolution so the tests cannot take the real account's lease.
  Clear inherited session-artifact variables and verify descendant sentinel safety.

Proposed acceptance revision, contingent on approving refusal-only migration:
replace the captured migration bullet with “A sole legacy store adopts in place
with its supporting roots and identities preserved; ambiguous multiple-store
installations receive an actionable preservation report, remain explicitly
unmigrated, and require operator reconciliation before adoption.” All remaining
Done when bullets remain applicable. This is a scope revision awaiting approval,
not a claim that refusing a conflicting installation completes its migration.

### 2026-10-01 — operator approved proceeding

Operator said “go ahead” after the reviewed draft. This approves second-launch
refusal and the recommended refusal-only ambiguous-store migration scope. Updated
the migration Done when bullet accordingly, preserving the prior wording and
rationale in the revisions above. Implementation plan is now recorded at
`workshop/plans/000366-couch-singleton-plan.md`; routine plan execution is authorized.

### 2026-10-01 — implementation and verification checkpoint

Implemented the approved ownership manager, read-only inspections, adoption CLI,
production runtime resolution and isolated child environment. Added filesystem,
subprocess and public-command acceptance coverage; documented adoption and operating
limits in README and atlas. Durable plan records the final symbol mapping and two
fixture corrections discovered in verification. Host-lock removal, ambient Pair-root
fallback and omitted isolation mutations each failed their targeted regression, then
passed after restoration. Selected/readable state cannot substitute for a live lease.

Focused singleton/identity suites and command acceptance pass under the race detector.
The installed Couch and full couchcmd normal suites passed. Broad race verification
found a preexisting concurrent FakeGit Ops append; a dedicated regression reproduced
lost calls and the race, and the fix plus exact recovery acceptance pass under -race.
The broad suites and post-fix couchcmd race rerun are still in progress at this checkpoint.

Classified all seven new production source files in the artifact inventory. Its
remaining 49 diagnostics exactly match original base f0c1e566 after regenerating the
runtime mirror in an isolated source export; existing drift belongs to #348. This is
an explicit baseline failure, not a passing full-repository test claim.

### 2026-10-01 — final verification before boundary review

`go test ./... -count=1` completed: Couch core passed (444.029s), and all changed
behavior packages passed. Three failing checks were investigated: artifact inventory
has the baseline-identical diagnostics above; gcruntime's
TestCouchReferencesLocalArchiveLocatorRoundTrip fails with the identical “missing
slot Couch metadata” at line 350 on both this branch and original base; the new
runtimeOwnership comment had attached to leaseResource and is now corrected, with
`go test ./cmd/internal/termcmd -count=1` passing. No blanket full-suite pass claimed.

`go test -race ./cmd/internal/couchsingleton ./cmd/internal/couchidentity
./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchmessage
./cmd/internal/storagegc -count=1` passed every package except the discovered
FakeGit fixture race. After its fix, the entire couchcmd race suite passed
(53.232s), as did the dedicated concurrent FakeGit regression and exact recovery
acceptance. Couch core race passed (482.823s). This covers every requested package
under the race detector. `git diff --check` passes. The issue Plan's final checkbox
now describes preparation for this gate; review completion is owned by sdlc close.

### 2026-10-01 — boundary review round 1

REWORK: BR-1 found numbered-slot errors and bytes missing from adoption evidence;
BR-2 found excluded stores with surviving live wrappers were admitted when their
supervisor lease was free; BR-3 found serialized selections could exceed the
reader's 64 KiB bound. Confirmed the code paths and extended the durable plan with
class-wide rework and regressions. Issue remains working; no PR or cutover yet.
The gate also required `sdlc claim --issue 366 --adopt` to record this pre-#277
claim's current workspace ownership; adoption succeeded without changing scope.

### 2026-10-01 — boundary findings implemented

BR-1/2: `StoreInspection` now retains existing global and discovered slot transaction
locks through publication, reuses read-only strict decoders and refuses changed or
expired handles. The digest includes slot-local payloads and topology. Every slot
error and every nonselected recorded incarnation/start owner is checked; live and
unknown ownership blocks exclusion, while the selected store can reconnect survivors.
Checks cover corrupt data, pending journals, symlinks, missing checkouts, post-preview
changes, lock contention, unknown/recycled PIDs, and liveness changes before publish.
Context cancellation also reaches backend discovery and process observation.

BR-3: one 64 KiB encoded selection limit governs decision, publication and reading.
Exact-boundary records round-trip; oversize normal/escaped paths refuse before
creating selected roots or invoking the publisher. Applies to Adopt and Acquire.

Verification: singleton full -race passes (6.583s); core new inspection/process tests
-race pass (1.834s), broader targeted core layout/preview/lease tests pass (3.212s).
Post-fix full couchcmd/identity/messaging/retention race suites pass (56.151s,
6.388s, 4.880s, 31.701s). Installed Couch and full termcmd tests pass. Public-command
liveness regression uses the actual current PID/start token and verifies immutable
preview bytes and selected-store reconnect; its restored race rerun passes (1.610s).
Removing the slot error check, slot evidence hash or incarnation predicate fails the
corresponding regression; restoring all guards passes. Artifact inventory remains
exactly the same 49 baseline diagnostics with the new inspection file classified.
Full post-fix core race run is still in progress; no second verdict requested yet.

### 2026-10-01 — post-rework broad verification complete

`go test -race ./cmd/internal/couchcore -count=1` passed (385.636s). Together with
the post-fix singleton, command, identity, messaging and retention runs above, every
affected runtime package has passing race coverage. Installed-command and termcmd
checks also pass; `git diff --check` is clean. Resubmitting BR-1/2/3 to the boundary
gate with the two original-base repository failures still explicitly disclosed.

### 2026-10-01 — boundary review round 2

BR-1/2/3 explicitly addressed. REWORK for BR-4: the parent's global Pair root was
passed in the launcher's already-scoped PAIR_DATA_DIR variable, while its global
readers still derived HOME/XDG. Confirmed both sides and revised the durable plan.
Parent tests now fail on the old export and require cleared PAIR_DATA_DIR plus
COUCH_PAIR_DATA_DIR carrying the selected global root; launcher-level artifact and
resume regressions will verify the complete contract before another review.

### 2026-10-01 — BR-4 implementation and consumer verification

Couch now clears PAIR_DATA_DIR when launching a new actor and exports the immutable
global root as COUCH_PAIR_DATA_DIR. Launcher `ResolveGlobalDataDir` validates that
selected root, feeds global claims/readers and derives the repository-scoped artifact
directory. A matching hosted scoped override works; a conflicting override refuses;
standalone explicit PAIR_DATA_DIR behavior remains supported. Installed Pair uses the
same resolver before embedded runtime extraction, preventing pre-launch ambient writes.

Real Pair-binary acceptance with a stateful fake terminal service now verifies create,
list, continuation listing and warm resume from a custom root distinct from HOME/XDG.
It checks actual scoped artifacts, claim/index records, child environment and
create→attach effects. The old behavior failed with a missing claim in ambient storage;
ignoring the selected global root as a mutation reproduces that failure. Source was
restored and the focused consumer suite passed (1.903s); full launcher race suite passed
(11.386s). Real embedded extraction tests fail before the fix and pass normally and
under race (1.494s); invalid selection produces no ambient writes. Parent ordinary and
blocked launch tests also passed under race (1.378s).

The targeted consumer sweep covers wrappers, continuation reinvocation, session
watchers, context/title, review and retention; these use the launcher's scoped variable
or explicit artifact bindings. Native agent HOME stores are separate by design.
Fresh full command/installed-binary reruns started after mutation restoration so their
recorded results cannot include the temporarily mutated source.

### 2026-10-01 — BR-4 restored integration verification complete

After source restoration, `go test -race ./cmd/internal/couchcmd ./cmd/pair-go
-count=1` passed (43.000s and 14.976s). Installed Couch and full termcmd tests passed
(2.168s and 0.453s). Artifact inventory still reports exactly the same 49 original-base
diagnostics; `git diff --check` passes. No further code changes after these tests.
Submitting BR-4 for disposition by the binary-owned boundary review.

### 2026-10-02 — review accepted and draft PR

Third close review: SHIP, no open findings; BR-1/2/3/4 all addressed. Gate recorded
codecomplete and measured 4.45h actual. Draft PR: https://github.com/xianxu/pair/pull/196.
Project and durable plan completion notes updated. No merge or live installation
cutover was performed. Remaining full-repository failures are the disclosed,
independently reproduced baseline cases, not passing-suite claims.
