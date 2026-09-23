# Durable numbered workspace provisioning implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use the appropriate
> execution skill and TDD after this durable plan receives operator approval.

**Goal:** Prepare a numbered host worktree from remote main, run Weave once
successfully, and reuse the workspace without changing existing work.

**Architecture:** Git supplies host-worktree facts; SDLC validates workspace
identity; Weave owns dependency setup and recovery. Couch needs a short host
creation routine, one host-creation lock per repository, a small creation-intent
record for interrupted Git operations, and a setup-success marker per host.

**Tech stack:** Go, existing typed operation dispatcher, Git, SDLC JSON v2,
Weave compile, flock, existing strict JSON and atomic-file helpers.
**Issue:** `workshop/issues/000305-slots-v2-workspace-provisioning.md`.
**Status:** Simplified plan reviewed; awaiting operator approval.
**Flow:** Full, one issue-close boundary. Expected code exceeds 100 added lines;
approval of this plan includes using `sdlc change-code --issue 305 --flow=full`.
Derive the estimate after that command's plan-quality gate, before implementation.

## Contract

```text
couch --internal provision-workspace /absolute/primary --slot=1
couch --internal provision-workspace /absolute/primary --slot=1 --remote=upstream
couch --internal provision-workspace /absolute/primary --slot=1 --retry
```

Return JSON with schema_version 1, address, path, resting_branch, baseline_sha
and disposition (created/reused/prepared); progress/errors go to stderr.
The operation creates no agent/thread. #306 supplies automatic number selection,
parked admission and the reservation through thread launch. Existing start-form
preview remains read-only; existing primary startup does not invoke provisioning.

The host path is `<fleet>/worktree/<repo>-slotN/<repo>`, resting on main-slotN.
Initial host baseline is fetched configured-remote main; primary dirty files and
unpublished local commits stay untouched. Later local-work transfer is explicit
operator/agent issue branching through ariadne#245. Dependencies are private
ordinary sibling clones handled by Weave under the existing #243 contract.

The successful-setup marker means exactly: Weave exited 0 for this verified host.
It is not a dependency inventory, source fingerprint, or build-freshness cache.
No Couch records or nonce files are written into dependency clones.

## Core concepts

All new production files below live in `cmd/internal/couchcore/` unless qualified.

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| WorkspaceIdentity | workspace_identity.go | new |
| ProvisionRequest / ProvisionResult | provision.go | new |
| CreationIntent / SetupSuccess | provision_store.go | new |
| HostObservation / NextHostAction | provision_host.go | new |
| SelectWorkspaceNumber | provision_select.go | new |

WorkspaceIdentity is the checked JSON v2 transport from SDLC, not another Git
resolver. Reject unknown versions, duplicate keys, missing required fields,
trailing JSON, invalid full OIDs and inconsistent absolute paths/repo/slot.
Preserve nullable fields and consume the published environment_host schema.
ProvisionRequest accepts a canonical positive slot integer and optional existing
remote name; zero, signs, leading zeroes and overflow refuse before effects.

CreationIntent contains only version, canonical repo/host paths, slot, remote,
captured baseline SHA, a creation token for matching branch reflog evidence,
and optional environment-directory filesystem identity recorded after creation.
SetupSuccess contains version, canonical host/common/admin paths, slot and initial
baseline. No phase enum, attempt history, dependency bindings or stored diagnostics.
HostObservation distinguishes absent, verified host, owned partial creation, and
conflicting/unknown state. NextHostAction makes the small decision table below
explicit and directly testable; no generic event/effect framework is introduced.

SelectWorkspaceNumber consumes caller-supplied classifications: prefer the lowest
ready/free workspace, then lowest absent positive number. Occupied, unknown and
partial candidates are never free. It reads no thread registry. Selection is
advisory; #306 must revalidate/reserve thread capacity through its own authority.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| WorkspaceProvisioner | provision.go | new | linear orchestration |
| ProvisionIO / OSProvisionIO | provision_io.go | new | Git, SDLC, Weave, progress |
| ProvisionStore | provision_store.go | new | intent/success atomic files |
| HostCreationLease | provision_lock_unix.go | new | repository-wide Git creation flock |
| ProvisionFixture | provision_fake_test.go | new | stateful test seam |
| Couch.Workspaces | couch.go | modified | injected provisioner |
| Operations / DirectStoreExecutor | ops.go / operationdispatch.go | modified | typed internal operation |
| OSRuntime.NewCouchWith / render | ../couchcmd/run.go | modified | production wiring and JSON output |

Reuse strictjson.Decode, writeAtomicBytes and existing path canonicalization.
Add inherited-lock process support to the narrow ProvisionIO adapter; ordinary
startup's existing GitRunner stays unchanged. No new Go dependency on Ariadne.
SDLC's JSON contract and Weave's process exit status are the integration APIs.

## Algorithm

### 1. Verify or create the host

Resolve primary identity through `sdlc workspace --json`, then lock
`<common-git-dir>/couch-workspaces/creation.lock` with nonblocking exclusive flock.
Re-resolve after acquiring the lock. Use one lock across all slots in this repo
only through host verification/creation; release it before Weave runs. Git work
for the same repo is serialized; different slots' slow setup can run concurrently.
A busy request returns an actionable retry message without choosing another slot.

Keep the lock file permanent and release by closing, not LOCK_UN. Direct Git
children inherit it, so caller death cannot admit another Couch creator while
that Git process is still mutating refs/worktree metadata. Do not hold the Couch
thread-store or singleton supervisor lock here. Weave gets no Pair lease.

For a fully Git-verified conventional host, preserve all refs/files and proceed
to the success-marker check. No fetch is needed, and no existing upstream is
silently retargeted. An externally created complete compatible host can be
prepared on explicit --retry; otherwise explain that preparation is unconfirmed.
For that first preparation without intent or success marker, capture the observed
main-slotN tip under the creation lock as baseline_sha, preserving current HEAD
and files. Otherwise use the recorded intent/marker baseline; ready reuse returns
the marker baseline.

For an absent host, select remote: explicit --remote; otherwise main's configured
remote when its merge target is refs/heads/main; otherwise the sole configured
remote. Reject ambiguity, local-dot tracking, invalid or missing sources.
Fetch main into its usual tracking ref, read its commit, and write CreationIntent
before branch/worktree creation. The repository-wide creation lock excludes the
other Couch fetch/read sequences, so private fetch refs and cleanup are unnecessary.
The chosen baseline is the tracking SHA observed after fetch. External user Git
commands remain outside Couch's lock; record and consistently use that observed
SHA, never reread it as the creation start point. No implicit local-main update.

CreationIntent is `<common>/couch-workspaces/<N>/creation.json`, bounded 16 KiB.
If it exists, explicit retry uses its recorded SHA without fetching again.
Invalid JSON/version/path/identity refuses rather than being treated as absent.
Create the environment directory exclusively; existing unrelated content refuses.
Record its filesystem identity before using it. A crash before ownership evidence
is durable can require manual inspection; do not guess based on an empty folder.
The enclosing fleet worktree directory may be created after ancestor validation.

Create main-slotN only if absent, at the captured SHA, using create-only Git ref
update with an attempt-identifying reflog message. On retry, matching ref/SHA and
creation evidence permit continuation; a matching name alone does not. Configure
branch remote/merge: matching keys need no write, missing expected keys can be
filled for this proved-owned branch, duplicates/conflicting keys refuse. Re-read
both before proceeding. Never force a ref or replace an unrelated branch.

Run `git worktree add <host> main-slotN` without force. Validate the result using
SDLC, Git administrative-directory linkage and expected repo/path/resting branch.
If interrupted, Git inspection determines whether a complete compatible host
exists. Unprovable partial registration stops with recovery guidance; no automatic
worktree removal, prune, reset or destructive rollback. Keep intent until setup
success is recorded; subsequent retries can still identify owned partial work.

### 2. Check initial setup success

Store `couch-setup-success.json` in the host's own Git administrative directory
(resolved with Git, not in the working tree or dependency clones). A normally
removed/recreated Git worktree loses that marker. Validate a present marker's
version and exact host/common/admin paths against current Git identity.
Changed HEAD, issue branch or dirty files do not invalidate it.

Valid marker -> return reused. Missing marker -> initial setup is unconfirmed.
For a newly created host, run setup immediately. For an existing incomplete host,
ordinary invocation explains --retry; that explicit retry reruns Weave. Invalid
marker contents refuse for inspection instead of authorizing host mutation.
If a host is externally removed/replaced, validate Git anew; never accept a
success marker from another administrative directory.

### 3. Run Weave when success is unconfirmed

Call `weave compile` with default targets from the verified host directory.
Weave owns source acquisition, its environment lock, partial clones/builds and
recovery. Couch streams its diagnostics and uses its exit status. No parsing of
Weave messages or observation of individual dependencies is needed.

On exit 0, briefly reacquire the creation lock, revalidate host identity, and
atomically write SetupSuccess. A concurrently published valid marker wins;
use its baseline rather than replacing it. Publication and owned temporary-file
cleanup both hold this same lock, so cleanup cannot remove an active write.
Only then return ready. If the lock is busy or validation/publication fails,
report unconfirmed setup and allow explicit retry; another compile is acceptable.
On failure/interruption, leave success unrecorded and expose retry.
A crash after compile succeeded but before marker publication is harmless: the
next explicit retry reruns compile. A lost success acknowledgment needs no new
Couch state. Repeated compilation is allowed by Weave's retry contract.

If another Weave setup is active (including surviving descendants), its own
lock refuses the competing run. Couch leaves the marker absent and shows that
failure. It never removes Weave's lock or attempts dependency repair. If two
requests observe no marker and compile serially, an extra compile is acceptable;
there is still one host identity. Both successful marker writes are equivalent.

After a valid success marker exists, remove only the owned creation-intent file
under the host-creation lock. If cleanup cannot acquire the lock or fails, retain
that one bounded file and retry cleanup on a later request; readiness still holds.
No growing journal or additional ref/nonce cleanup protocol is required.

### Decision table

| Verified observation | Action |
| --- | --- |
| Host absent, no conflicts | create from remote baseline, run Weave |
| Owned partial Git creation | explicit retry completes provable missing steps |
| Host valid, success marker valid | reuse, no fetch/compile |
| Host valid, success absent | new creation runs Weave; later invocation requires --retry |
| Weave exits 0 | validate host and atomically mark success |
| Weave fails / outcome unconfirmed | no success marker; show diagnostic/retry |
| Foreign paths/refs, corrupt evidence, unverifiable partial host | refuse without overwriting work |

These branches are explicit NextHostAction cases tested with injected observations.
Git and filesystem evidence determine progress; there is no persisted state machine.

## Bounds and safety

- Local probe: 5 seconds; fetch: 120 seconds; Weave: 20 minutes. These are initial
  engineering defaults, injectable in tests. Timeout preserves work and exposes retry.
- One synchronous subprocess per invocation. Run outside the UI event loop.
  Requests use context cancellation; CLI SIGINT/SIGTERM gets a scoped context.
- Direct Git children inherit the creation lease. Close-only flock lifetime
  survives caller death. Weave independently manages its inherited setup lease.
- For owned direct processes use an isolated process group, SIGTERM then a 2s
  grace, bounded force-stop/Wait, and 1s WaitDelay. Weave owns its further groups;
  surviving setup descendants remain excluded by Weave and can cause busy retry.
- Structured subprocess output <=1 MiB, marker/intent <=16 KiB. Keep <=64 KiB
  in-memory diagnostic tail; no durable diagnostic log. Progress goes to stderr.
- Strictly parse transport/records, reject symlink metadata and foreign paths,
  pass argv arrays and avoid credential-bearing remote URLs in stored records.
- One lock per repo, at most one creation record and one success marker per slot.
  Markers live with host Git metadata; intents are removed after success. Owned
  atomic-write temporaries are cleaned under the creation lock in their own
  directory only. Worktrees/dependencies persist until explicit user removal.
- Successful setup is historical evidence. Later source edits, missing dependency
  files or changed generated output are repaired with explicit Weave/build; Couch
  does not continuously inspect or invalidate setup based on dependency contents.

## Tests and execution tasks

### Task 1 — checked inputs and host decisions

Create `workspace_identity.go`, `provision_host.go`, `provision_select.go` and
colocated tests. Put request/result structs in `provision.go`, record structs
in `provision_store.go`.

- [ ] Write failing tests for v2 transport/nullability/unknown versions/duplicate
  keys/trailing data, slot grammar, identity mismatch and the decision table.
- [ ] Test lowest-ready reuse, numeric order, occupied/unknown/partial exclusions
  and selection races requiring caller revalidation.
- [ ] Run `go test ./cmd/internal/couchcore -run 'Test(WorkspaceIdentity|ProvisionRequest|ProvisionHost|SelectWorkspaceNumber)' -count=1`;
  implement pure parsers/decisions, rerun green and commit.

### Task 2 — host creation and small durable records

Create `provision_io.go`, `provision_store.go`, `provision_lock_unix.go`,
`provision_fake_test.go`, `provision_git_test.go`, `provision_subprocess_test.go`.
The fake models refs/upstreams, Git membership, files, locks and command outcomes
across calls, including a successful effect with a lost acknowledgment. It does
not simulate dependency internals. Real Git fixtures check the consumed commands.

- [ ] Write failing tests for new :1/:2, different local/remote HEAD, primary
  dirty/untracked work, missing/ambiguous/non-origin remotes, spaces/hyphens,
  collisions, branch creation acknowledgment loss and partial upstream writes.
- [ ] Verify same-repo requests serialize fetch/capture/creation and preserve the
  captured SHA across interruption, while different repo requests stay independent.
- [ ] Test inherited Git lease after caller death, strict/atomic records and
  marker placement in actual per-worktree Git directories.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestProvision(Host|Git|Store|Lease)' -count=1`;
  implement host routine/records/lease, rerun green and commit.

### Task 3 — Weave retry and production operation

Implement `WorkspaceProvisioner` in `provision.go`. Modify couch.go, ops.go,
operationdispatch.go, couchcmd/run.go. Add couchcmd/provision_test.go and update
operation/CLI/readme contracts.

- [ ] Write failing tests: missing marker runs Weave; exit 0 writes success;
  failure does not; lost publication repeats compile; valid success skips it;
  malformed/mismatched markers refuse; host branch/dirty changes are preserved.
- [ ] Use a deterministic barrier to overlap marker publication and temporary
  cleanup; verify the shared lock protects active writes. Test externally created
  hosts capture main-slotN as baseline without changing current HEAD/files.
- [ ] Model Weave busy/success/failure through the process seam. Test serial
  duplicate retries as acceptable and no duplicate host creation or thread launch.
- [ ] Wire PresentationInternal + ExecuteDirectStore + EffectProcess + a new
  workspace result family. Args: path, --slot=N, optional --remote=R, --retry.
- [ ] Inject WorkspaceProvisioner in OSRuntime.NewCouchWith; route progress to
  stderr and result JSON to stdout. Tests explicitly inject the fake, never an
  ambient real process/filesystem fallback. Keep PrepareStart unchanged.
- [ ] Test internal CLI -> dispatcher -> provisioner, cancellation, nonzero
  failure, no supervisor acquisition and no AllocateThreadTag/agent launch.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`;
  fix regressions and commit.

### Task 4 — conformance, documentation and one close boundary

Add `provision_conformance_test.go`, atlas/workspace-provisioning.md; update
README.md, atlas/couch.md, atlas/index.md, issue and project state.

- [ ] Opt-in `PAIR_LIVE_WORKSPACE=1` test runs actual SDLC v2 + Weave on isolated
  minimal Git fixtures without packages/tools/generators. Exercise real marker
  publication, missing-marker repeat and ready reuse; host remotes can be local
  test bare repos, and the minimal Weave fixture has no local-source dependencies.
- [ ] Run `PAIR_LIVE_WORKSPACE=1 go test ./cmd/internal/couchcore -run '^TestProvisionConformance$' -count=1 -v`.
- [ ] Document internal invocation, retry, setup-success meaning, #306 ownership
  and manual recovery for unverifiable partial hosts. Link the atlas page.
- [ ] Run targeted race tests, `make runtimebundle-generate`, `go test ./... -count=1`,
  and `go vet ./cmd/internal/couchcore ./cmd/internal/couchcmd`.
- [ ] Build `make pair couch`; exercise CLI against temporary repos and isolated
  Couch/Pair data roots. Verify refs/files and no setup on ready reuse.
- [ ] Reconcile checkboxes/evidence and commit; run `sdlc close --issue 305
  --verified '<observed evidence>'` once. Its review owns the boundary.
- [ ] After successful close, use `sdlc pr` and `sdlc merge` for publication.

## Architecture and approval

ARCH-DRY/PURPOSE: reuse SDLC/Weave and deliver one reachable operation, with no
second dependency catalog. ARCH-PURE/ORDER: a small typed host decision table
plus linear IO replaces generic transition machinery; interrupted effects are
reconciled from Git evidence. ARCH-MOCK: stateful process/filesystem fixture and
real Git/SDLC/Weave conformance. ARCH-CONSTRAINTS: bounded processes/outputs and
short host-only lock extent. ARCH-SECURE: strict records, canonical Git identity
and non-destructive collision refusals. ARCH-FUNERAL: one success marker, bounded
creation evidence, no attempt history/dependency nonce files/private fetch refs.

Obtain fresh plan review and checkpoint the issue/plan, then present this revised
plan for operator approval. Continue with change-code's plan-quality/estimate
steps and an in-place branch in this checkout; preserve unrelated local files.

## Revisions

### 2026-09-23 — first plan review: fetch and upstream recovery

Reason: fresh review found a fetch/read race through shared remote-tracking refs
and an unspecified interruption between branch creation and upstream writes.
Delta: reserve an attempt-owned fetch ref, use atomic dual-ref fetch and read
only that private ref; retain/reconcile it with bounded cleanup. Added explicit
upstream intent, absent/matching/conflicting-key handling and retry transitions.
Added deterministic tests for both classes before seeking implementation approval.

### 2026-09-23 — review approved and fetch contract probed

Fresh review approved the revised plan with no remaining significant gaps.
A temporary real-Git fixture proved atomic dual-ref fetch accepts both refspecs,
and a second slot fetch after remote advancement leaves the first private SHA
unchanged. This checks the planned primitive, not implementation completion.

### 2026-09-23 — simplify around repeatable Weave setup

Reason: operator pointed out that absent evidence of a successful weave compile
can be handled by rerunning it. Delta: active plan replaces the generic persisted
state machine and dependency inventory/bindings with a host-success marker and
linear provisioning. One repository-wide lock protects host Git creation only;
Weave setup runs after releasing it and uses Weave's own lock/recovery. This also
removes private fetch refs: cooperative fetch/capture is serialized. Retain only
small host creation intent/evidence for interrupted Git operations. Earlier
review approvals apply to the superseded design, not this simplified proposal.

### 2026-09-23 — simplified plan review corrections

Reason: fresh review found a marker-publication/temporary-cleanup race and an
undefined baseline for externally created hosts. Delta: briefly reacquire the
existing creation lock for final validation/publication; a valid concurrent marker
wins. Capture the resting-branch tip for first preparation of an external host.
Added focused tests. Weave still runs outside the lock; no new state or lock.
