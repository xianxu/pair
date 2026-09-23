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
**Status:** Approved and implemented; final verification/close in progress.
**Flow:** Full, one issue-close boundary. Expected code exceeds 100 added lines;
approval of this plan includes using `sdlc change-code --issue 305 --flow=full`.
Derive the estimate after that command's plan-quality gate, before implementation.

## Contract

```text
couch --internal provision-workspace /absolute/primary --slot=1
couch --internal provision-workspace /absolute/primary --slot=1 --remote=upstream
```

Return JSON with schema_version 1, address, path, resting_branch, baseline_sha
and disposition (created/reused/prepared); progress/errors go to stderr.
The operation ensures readiness on every call: reuse verified completed work,
complete provable missing Git steps, and rerun safe setup when success is absent.
Errors end the invocation visibly; there is no background or unbounded retry loop.
#306 calls it when creating/opening a numbered workspace or cold-resuming its
parked thread, before agent launch. Warm reattachment to a still-running agent
only reconnects; it does not run setup. Primary :0 behavior remains unchanged.
The operation creates no agent/thread. #306 supplies automatic number selection,
parked admission and thread reservation through launch. #305 accepts a concrete
slot number and has no thread effects, so it requires no reservation capability.
Its completion grants workspace readiness only, never permission to launch.
#306 must reserve the selected workspace in its authoritative thread lifecycle
before calling provisioning, keep that reservation through launch, and release
it on failure; a competing thread launch must refuse while it is held. Its
reservation representation is designed in #306, which depends on #305, not the
reverse. The standalone provisioning operation is safe without that reservation:
it can prepare/reuse the same workspace but cannot create a duplicate thread. Existing start-form
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
prepared by the same invocation when successful setup is unconfirmed.
For that first preparation without intent or success marker, capture the observed
main-slotN tip under the creation lock as baseline_sha, preserving current HEAD
and files. Otherwise use the recorded intent/marker baseline; ready reuse returns
the marker baseline.

For an absent host, select remote: explicit --remote; otherwise main's configured
remote when its merge target is refs/heads/main; otherwise the sole configured
remote. Reject ambiguity, local-dot tracking, invalid or missing sources.
Fetch main into its usual tracking ref using `git fetch --verbose --porcelain --no-tags
--no-recurse-submodules --no-write-fetch-head --refmap= <remote>
+refs/heads/main:refs/remotes/<remote>/main`. Parse the new full OID from the one
matching porcelain row with ParseFetchBaseline; reject failed/ambiguous output.
Use this command's output, never a subsequent tracking-ref read, as the captured
baseline. Even an external fetch between completion and recording cannot change
that value. Persist CreationIntent before branch/worktree creation. Git lacking
--porcelain fails visibly; do not fall back to a racy capture. No private refs or
local-main updates are needed.

CreationIntent is `<common>/couch-workspaces/<N>/creation.json`, bounded 16 KiB.
If it exists, the same invocation uses its recorded SHA without fetching again.
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
Run setup for both newly created and existing verified hosts when success is
unconfirmed. No retry flag or separate retry operation is needed. Invalid
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
report unconfirmed setup and allow another invocation; another compile is acceptable.
On failure/interruption, leave success unrecorded and expose retry.
A crash after compile succeeded but before marker publication is harmless: the
next invocation reruns compile. A lost success acknowledgment needs no new
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
| Owned partial Git creation | complete provable missing steps |
| Host valid, success marker valid | reuse, no fetch/compile |
| Host valid, success absent | run Weave in this invocation |
| Weave exits 0 | validate host and atomically mark success |
| Weave fails / outcome unconfirmed | no success marker; show diagnostic/retry |
| Foreign paths/refs, corrupt evidence, unverifiable partial host | refuse without overwriting work |

These branches are explicit NextHostAction cases tested with injected observations.
Git and filesystem evidence determine progress; there is no persisted state machine.

### Interrupting events (invocation-local ordering)

These are ordered branches of Ensure/NextHostAction, not stored setup phases.

| Observation + event | Result and ownership |
| --- | --- |
| Creating Git host + cancellation | Cancel/join owned Git group, close lease; retain intent, return error. Next call inspects actual Git and completes only provable steps. |
| Creating Git host + caller death | Direct Git child retains lease until exit; second caller gets busy. After exit, next invocation reconciles intent/Git. |
| Host exists, setup unconfirmed + second request | Both can reach Weave; Weave lock refuses overlap. A later serial compile is safe. No queue or thread launch occurs. |
| Weave running + cancellation | Cancel/join direct Weave process group; return error without marker even if exit races cancellation. Surviving Weave-owned groups retain Weave lock; next call can get busy. |
| Compile completed + canceled context | Do not publish; require successful non-canceled invocation for publication. |
| Compile completed + caller death before marker | No marker; next call compiles again. No background completion handler can publish. |
| Compile completed + concurrent marker publication | Reacquire creation lease; first valid marker wins, later caller returns its result. Busy lease returns error. |

Barrier tests in Ensure and subprocess fixtures reproduce each interruption at
its effect boundary. Marker writes and cleanup share the same lease. A successful
atomic marker rename followed by cancellation may leave success recorded; the
next invocation observes that completed effect and reuses it.

## Bounds and safety

- Local probe: 5 seconds; fetch: 120 seconds; Weave: 20 minutes. These are initial
  engineering defaults, injectable in tests. Timeout preserves work and exposes retry.
- One subprocess at a time per invocation (several sequential commands). Run outside the UI event loop.
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

Each task follows failing test → minimum implementation → passing test → commit.
Use deterministic barriers for concurrent outcomes, stateful fakes for command
outcomes and real temporary Git repositories for actual ref/worktree behavior.

### Task 1 — checked inputs and host decisions

Create workspace_identity.go, provision_host.go, provision_select.go and colocated
tests. Put request/result structs in provision.go, records in provision_store.go.

- [x] ParseWorkspaceIdentity / ParseProvisionRequest: malformed transport/grammar
  → strict decode, required-field checks and fuzzed boundary input.
- [x] NextHostAction: incomplete/conflicting observations → total decision table
  with safe refusal and repeat-invocation properties.
- [x] SelectWorkspaceNumber: unsorted/occupied/partial observations → pure minimum
  selection; output never implies a thread reservation.
- [x] Run `go test ./cmd/internal/couchcore -run 'Test(WorkspaceIdentity|ProvisionRequest|ProvisionHost|SelectWorkspaceNumber)' -count=1`.

### Task 2 — host creation and small durable records

Create provision_io.go, provision_store.go, provision_lock_unix.go,
provision_fake_test.go, provision_git_test.go, provision_subprocess_test.go.

- [x] WorkspaceProvisioner.ensureHost: interrupted Git effects and foreign
  collisions → reconcile owned evidence in stateful fake + real Git fixture;
  verify repeated calls preserve user refs/files.
- [x] ParseFetchBaseline: malformed/ambiguous fetch output and later tracking-ref
  changes → strict porcelain parsing + real Git with deterministic intervening fetch.
- [x] ProvisionStore read/write: malformed, oversized, aliased or interrupted
  records → strict bounded IO and atomic publication; barriers guard cleanup races.
- [x] AcquireHostCreationLease / OSProvisionIO.Run: contention, caller death and
  hanging children → subprocess fixtures prove inherited exclusion and bounded wait.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestProvision(Host|Git|Store|Lease)|TestParseFetchBaseline' -count=1`.

### Task 3 — Weave readiness and production operation

Implement WorkspaceProvisioner.Ensure in provision.go. Modify couch.go, ops.go,
operationdispatch.go, couchcmd/run.go. Add couchcmd/provision_test.go and update
operation/CLI contracts. Inject production IO in OSRuntime.NewCouchWith.

- [x] WorkspaceProvisioner.Ensure: lost acknowledgments and repeated invocations
  → stateful fixture models success/busy/failure; confirmed setup skips Weave,
  unconfirmed setup repeats it without a mode flag or dependency simulation.
- [x] DirectStoreExecutor / RunWithRuntime: malformed calls and setup failures
  → exercise CLI through the real dispatcher with injected IO; assert progress on
  stderr, JSON result on stdout and no supervisor/thread/agent creation.
- [x] Register PresentationInternal + ExecuteDirectStore + EffectProcess + new
  workspace result family; args path, --slot=N, optional --remote=R.
- [x] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`.

### Task 4 — conformance, documentation and one close boundary

Add provision_conformance_test.go, atlas/workspace-provisioning.md; update
README.md, atlas/couch.md, atlas/index.md, issue and project state.

- [x] TestProvisionConformance: actual SDLC v2 + Weave against isolated Git
  fixtures → verify initial setup, repeat readiness and preservation of host and
  dependency work. Minimal manifests omit package/tool/generator effects.
- [x] Run `PAIR_LIVE_WORKSPACE=1 go test ./cmd/internal/couchcore -run '^TestProvisionConformance$' -count=1 -v`.
- [x] Document repeatable readiness, marker semantics, #306 ownership and manual
  recovery for unverifiable partial hosts; link the atlas page.
- [x] Run targeted race tests, `make runtimebundle-generate`, `go test ./... -count=1`,
  and `go vet ./cmd/internal/couchcore ./cmd/internal/couchcmd`.
- [x] Build `make pair bin/couch`; smoke the CLI against isolated temporary repos/data.
- [ ] Reconcile evidence and commit; run `sdlc close --issue 305
  --verified '<observed evidence>'` once, then sdlc pr / sdlc merge.

## Operating envelope and exclusions

This is an interactive local-workstation operation. Five-second probes detect a
stalled local Git/SDLC command; 120-second fetch and 20-minute setup allow network
and build work while bounding a stuck invocation. These are policy ceilings,
not measured latency promises. The 1 MiB structured-output cap is ample for one
identity/ref response; the 64 KiB diagnostic tail bounds verbose build failures.
Tests inject shorter limits. There is no background scheduler.

The operator owns retained workspaces and their disk cost. Git worktree list
shows hosts now; #307 adds grouped discovery. Removal is deliberate ordinary Git
worktree removal followed by operator inspection/removal of sibling clones.
Couch never guesses whether an abandoned directory's work is disposable. Slot
count increases retained disk consumption, not per-launch metadata history.

Non-goals: thread launch/admission/reservations (#306), UI grouping (#307),
automatic refresh/branch transfer (explicit Ariadne workflow), dependency inventory
or repair (Weave), a retry mode (same operation repeats), and automatic removal
(operator owns retained user work).

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

### 2026-09-23 — one repeatable readiness operation

Reason: operator removed the distinction between initial setup and explicit retry.
Delta: removed --retry; every call reuses verified completion and reapplies safe
unconfirmed steps. #306 invokes readiness before numbered-workspace launch/cold
resume; warm reattachment skips setup. Failure returns visibly; the next ordinary
invocation recovers without a separate mode. Existing collision safeguards remain.

### 2026-09-23 — implementation gate refinements

PQ-1: capture the fetch command's porcelain OID instead of rereading a mutable
tracking ref; no private refs. PQ-2: clarify readiness grants no thread authority;
#306 owns reservations before calling this independently usable operation, so
adding a reverse dependency would create a cycle. PQ-3: replace test case prose
with named function strategies. PQ-4–6: clarify workload budgets, retention owner
and exclusions. Product behavior and approved simplification are unchanged.

Fetch probe: --verbose is required with --porcelain to emit the up-to-date row;
a temporary real-Git fixture confirmed the full new OID is present in that case.

### 2026-09-23 — clarify gate scope and interruption ordering

PQ-2 is disputed: please withdraw the requested reverse dependency. #306 already
depends on #305, and no code, token, API or reservation from #306 is consumed here.
The executable #305 boundary is Ensure(ctx, ProvisionRequest{Path, Slot, Remote});
it returns ProvisionResult or error and has zero thread effects. Two callers of
Ensure for the same slot are supported without any thread reservation. Tests
exercise this directly. SelectWorkspaceNumber is a pure advisory helper only.
The future caller's thread reservation has no handoff into this API. Requiring
#306 first creates a dependency cycle and defeats the agreed independent task
split; documentation of that caller's obligations does not make it a prerequisite.

PQ-7: added explicit interruption/event branches with ownership, busy/cancel
policy, publication rules and deterministic test seams, retaining linear Ensure
and no persisted setup-phase machinery as the operator requested.

### 2026-09-23 — implementation verification command correction

The Makefile exposes bin/couch, not a phony couch target. Use make pair bin/couch;
the earlier make pair couch built Pair then refused the nonexistent target.
The implementation reuses a parameterized atomic writer for a provisioning-only
temporary prefix; the selector remains an explicitly documented #306 integration API.

### 2026-09-23 — implementation checkpoint

Operator approved implementation. SDLC change-code passed plan-quality at its
configured review cap; disputed PQ-2/PQ-8 reservation demands remain recorded for
close review. #305 consumes no #306 capability and performs no thread effects.
Estimate-quality accepted the calibrated decomposition. Created the in-place
000305 branch. Implemented the operation, pure transport/decisions, inherited
creation lease, bounded subprocesses, host intent and atomic setup success.
Live conformance requires a recorded HTTPS dependency origin; fixtures use Git
insteadOf transport rewriting to isolated local repositories, not local-source
Weave declarations. Requests need no retry flag. Task-1 request/result types live
in provision_request.go; WorkspaceReadiness lives in provision_dispatch.go; the
stateful Git fixture lives in provision_git_test.go. No extra fake framework.
Focused/race and live tests passed; full-suite inventory checks were updated for
new files and the deliberately deferred #306 selector consumer. Final suite and
SDLC close remain to be completed.
