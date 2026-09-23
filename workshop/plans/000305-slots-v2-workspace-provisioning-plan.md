# Durable numbered workspace provisioning implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy).
> Use superpowers-subagent-driven-development for bounded tasks where appropriate,
> or superpowers-executing-plans for work relying on the current session context.
> Follow the checklist with TDD. This plan requires operator approval before code.

**Goal:** Couch prepares a durable numbered Git worktree from configured remote
main, composes its private environment, and safely reuses or retries it.

**Architecture:** One internal typed operation delegates to a provisioning
controller. A pure transition model selects effects from verified observations;
the controller owns filesystem/process effects and inherited per-slot leases.
SDLC owns workspace identity, Weave owns dependency acquisition/composition, and
Couch owns the host worktree transaction and evidence that initial setup finished.

**Tech stack:** Go, existing Couch operation dispatcher, Git, SDLC JSON v2,
Weave compile, POSIX flock, existing strict JSON and atomic publication helpers.

**Status:** Fresh review approved; awaiting operator plan approval. Product boundary agreed.
**Issue:** `workshop/issues/000305-slots-v2-workspace-provisioning.md`.
**Flow:** Full: expected production changes exceed 100 lines; design exceeds the
quick-flow design limit. One atomic issue-close review, no artificial milestones.
**Estimate:** Derive after change-code's plan-quality gate accepts this plan.

## Scope and delivered surface

The operation prepares a directory; it never creates or reserves a thread.
The production entry point is:

```text
couch --internal provision-workspace /absolute/primary --slot=1
couch --internal provision-workspace /absolute/primary --slot=1 --remote=upstream
couch --internal provision-workspace /absolute/primary --slot=1 --retry
```

`path` is required. Resolve it through SDLC to a primary repository; a caller in
an existing numbered workspace may name that host path and resolve its primary.
Reject dependency/ordinary-worktree inputs initially: the caller must name the
host primary explicitly instead of inferring a new fleet from a private clone.
`slot` is required, positive canonical decimal (no : prefix, + sign, leading
zeroes, fractional value or overflow). :0 is never provisioned by this operation.
`remote` is optional; `retry` is a boolean explicit permission to continue an
incomplete preparation or establish readiness for a compatible existing host.

Successful output is one JSON object with `schema_version: 1`, `address`,
`path`, `resting_branch`, `baseline_sha`, and `disposition` (`created`, `reused`,
`prepared`). It describes the provisioning result, not the SDLC transport schema.
Progress and actionable diagnostics go to stderr, keeping stdout machine-readable.
`baseline_sha` is the recorded initial baseline, not an assertion about current
HEAD after a workspace has been used for an issue.

The public start form and PrepareStart stay read-only. #306 uses the same typed
operation/controller before its existing launch flow and owns parked admission,
thread occupancy and reservation through launch. #307/#308 own display/preferences.

A new slot at `/fleet/worktree/repo-slot1/repo` starts at fetched remote/main.
No local commit, stash, selective copying or source-workspace transfer occurs.
The primary may have dirty files and unpublished commits. They are preserved.
Later issue branching from another workspace belongs to ariadne#245.

## Core concepts

### Pure entities

All new entities below live in `cmd/internal/couchcore/`. The filenames form
focused units; they do not expand the existing large couch.go with Git workflow.

| Name | Lives in | Status |
| --- | --- | --- |
| WorkspaceIdentity | workspace_identity.go | new |
| ProvisionRequest / ProvisionResult | provision_model.go | new |
| ProvisionRecord / ProvisionPhase | provision_model.go | new |
| ProvisionObservation / ProvisionEvent / ProvisionEffect | provision_model.go | new |
| ProvisionMachine | provision_model.go | new |
| SelectWorkspaceNumber | provision_select.go | new |

`WorkspaceIdentity` is a checked transport view of Ariadne's JSON v2, not a
second Git resolver. Parse a single JSON object, reject unknown schema versions,
malformed/truncated/trailing input and wrong types; allow only known fields of v2.
Use pointer fields for nullable address/slot/branch/head/resting_branch. Validate
required absolute paths, primary identity, canonical decimal slot, complete SHA-1
or SHA-256 OIDs, and consistency between requested identity and returned address.
Treat environment_host through the actual v2 schema, not an untyped map.

`ProvisionRequest` owns explicit slot/remote/retry intent. `ProvisionRecord` is
one versioned current record per slot: common Git identity, primary/environment/
host paths, canonical slot, attempt token, captured remote name and SHA, phase,
creation evidence, setup bindings and bounded last failure. The attempt token
is a nonce for matching records; it is not proof of filesystem ownership.

`ProvisionMachine` owns authoritative phase transitions through an unexported
phase field and an Apply(event) method returning declared effects. The controller
cannot write phase fields directly. Pure transition tests need no IO fake.
There is one machine per provisioning attempt, restored from a validated record.

`SelectWorkspaceNumber` consumes classified candidates supplied by the caller:
ready/free, occupied, incomplete, unknown or absent. Select the lowest ready/free
candidate; otherwise the lowest positive absent number. Incomplete and unknown
numbers are never considered free. The returned number is advisory until the
caller acquires its lease and revalidates occupancy. It never reads ThreadStore.
The selector belongs here because #305 specifies allocation/reuse policy; #306
will supply its authoritative thread classifications.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| ProvisionFixture | provision_fake_test.go | new | stateful test implementation of ProvisionIO |
| WorkspaceProvisioner | provision.go | new | model/effect orchestration |
| ProvisionIO / OSProvisionIO | provision_io.go | new | filesystem and process boundary |
| ProvisionLease | provision_lease_unix.go | new | per-slot inherited flock |
| ProvisionStore | provision_store.go | new | bounded records/atomic writes |
| ProvisionProcess | provision_process_unix.go | new | argv processes, progress, cancellation |
| Couch.Workspaces | couch.go | modified | injected controller |
| Operations | ops.go | modified | typed declaration/result family |
| DirectStoreExecutor | operationdispatch.go | modified | validated request dispatch |
| OSRuntime.NewCouchWith | ../couchcmd/run.go | modified | production composition |
| runTypedOperationWithConsole / render | ../couchcmd/run.go | modified | context/progress/result |

Reuse `strictjson.Decode` and `writeAtomicBytes` for records. Its random temporary
files are cleaned only within this new record directory while its lease is held;
never sweep another store's temporary files. Add no general filesystem framework.
Use existing PathOps canonicalization for initial paths, followed by no-symlink
checks at mutation boundaries. Existing ExecGit/GitRunner remain for ordinary
startup; provisioning's Git commands need the new inherited-lease process seam.

Use `sdlc workspace --json` and `weave compile` subprocesses. Do not import
Ariadne's internal packages or introduce an unpublished Go-module dependency.
Capture the required v2 transport fixture and verify it against the real binary.
Ariadne's `pkg/workspace` remains the sole implementation of Git identity.

## Transaction, ownership and recovery

### Durable placement and evidence

Resolve the primary's canonical common Git directory through SDLC. Store records
at `<common>/couch-workspaces/<N>/record.json` and a permanent `operation.lock`
inside that same directory. These paths are independent of Couch namespaces and
remain outside all working trees. Validate that metadata parents are directories,
not symlinks, and create mode 0700 directories / 0600 files.

Use one stable record file per slot, atomically replaced under its lease. Validate
record version, bounded lengths, enums, all path relationships, slot and common
identity before effects. Unknown version/corruption produces a visible refusal;
never treat an unreadable record as absence or authorize deletion from it.

Host ownership evidence consists of the record's captured remote/SHA plus real
Git common-directory membership, expected registered path, resting ref and host
Git administrative directory. After host verification, create one private nonce
binding file in that host administrative directory and record it. Ordinary branch
changes preserve it; removing/replacing a worktree invalidates it.

For each direct sibling ordinary clone present after successful setup, record
its canonical root/common directory and one nonce binding in that clone's .git
administrative directory. Discover with immediate-directory enumeration and Git
probes; do not parse construct/deps or Weave diagnostics to reimplement its graph.
Ignore non-Git sibling directories; refuse broken Git evidence for recorded
bindings. These bindings only detect replaced/missing initialized repositories.
They do not assert the selected revisions, graph, tools or generated files are
current. More than 256 sibling Git clones is a visible setup-limit error.

Before using a ready receipt, validate the host identity and recorded bindings.
A dirty branch, changed HEAD or dependency revision is valid. Missing/replaced
host fails visibly without automatic recreation, reset or deletion. A missing/
replaced dependency marks readiness unconfirmed and requires explicit --retry;
Weave then owns acquisition/validation and new binding publication on success.

Readiness means initial setup succeeded for these repository instances. It is
not a build cache or source-freshness guarantee. After deliberate source/manifest/
generator edits, the operator/agent explicitly runs Weave/build as appropriate.
No broad fingerprint scheme is added. New dependency edges are not discovered
by ready resume; explicitly composing the changed source remains the developer's
normal responsibility.

### Leases and caller responsibility

Take nonblocking exclusive flock on the permanent per-slot operation.lock.
A busy slot returns an actionable retry message immediately; no hidden queue.
Close the file to release; do not call LOCK_UN because a child may still hold an
inherited copy. Pass its descriptor to every direct mutating Git/Weave child.
Never unlink the lock file while the environment remains managed.

Explicit :N provisioning needs no global long-running lock: different names
have different paths/refs; Git itself arbitrates ref/index mutations. Reprobe
ownership after lease acquisition and before every host-creation effect.
Different slots may compose concurrently; no unbounded task fan-out is created.

For #306, export the lease acquisition plus an already-leased Provision method.
The automatic caller selects a candidate, acquires that slot's lease, rereads
thread occupancy/admission, then provisions and reserves/creates its thread
before releasing. Busy/raced candidates are reconsidered, never assumed free.
The #305 convenience entry point acquires/releases the same lease around its
explicit request and does not promise a thread reservation on return.
#306 must use the already-leased method to avoid nested lock acquisition.

Weave independently locks `.weave-setup.lock` and passes that lease to its setup
children. It does not forward arbitrary Pair descriptors through its entire
subprocess tree. If Weave dies while its writers survive, Weave's setup lock
continues protecting dependency/build effects. Pair retries must preserve host
state and accept Weave's busy refusal; they never remove its lock/stages or run
dependency repair themselves. Pair lock inheritance protects direct Git writers.

Uncooperative external Git/filesystem edits are outside the cooperative lock,
so revalidate immediately before effects and never force a conflicting effect.
Prefer Git's atomic create-only operations, never branch -f or worktree --force.

### Remote baseline and Git sequence

Choose remote in this order: explicit `--remote`; primary branch.main.remote when
branch.main.merge is refs/heads/main; otherwise the only configured remote. Reject
multiple choices, the local-dot remote, empty/missing remotes, and inconsistent
tracking configuration. Validate remote names through git config/remotes and pass
argv arrays, not shell strings. No remote URL is stored in progress/receipts.

Before fetching, persist a reservation with the generation token, selected
remote and an absent private ref `refs/couch/provision/<N>/<attempt>`. Fetch with
`--atomic --no-tags --no-write-fetch-head` and two explicit refspecs:

```text
refs/heads/main:refs/couch/provision/<N>/<attempt>
refs/heads/main:refs/remotes/<remote>/main
```

Read the full commit OID from the attempt-owned ref, never from the shared
remote-tracking ref or FETCH_HEAD. Different slots can update tracking without
changing each other's captured baseline. No force refspec is used. A concurrent
tracking update that makes the atomic fetch fail leaves the attempt retryable;
inspect its private ref before any retry rather than assuming no effect occurred.
A valid complete private ref from that attempt proves its fetched baseline even
if command acknowledgment was lost. An absent private ref permits another fetch
only through explicit retry; malformed/conflicting state refuses.

Persist the private ref's SHA before host creation. The primary's local
main/HEAD, index and working files stay untouched. Upstream is a separate
relationship and may point at a subsequently advanced remote-tracking commit.
Keep the attempt-owned ref while incomplete, including failure before the
baseline record write. Once the host and resting ref prove the captured baseline
is reachable, delete only that exact private ref with expected-old-OID checking.
Deletion failure leaves a cleanup-pending field, not a failed successful setup;
next operation retries that one ref cleanup under the slot lease. One active
private ref per slot is permitted; do not start a new generation until the old
ref/receipt is reconciled. Tests assert no ref growth across repeated retries.

On first request, reject any pre-existing destination environment directory,
resting ref or registered worktree unless the whole existing host resolves as a
compatible slot and --retry explicitly requests initial readiness adoption.
A complete ready record is always eligible for validation/reuse without --retry.
An unrelated directory or ref must never be taken over because its name matches.

Create the enclosing directory using create-only semantics. Persist filesystem
identity after creation; an interruption before that evidence is durable is
ambiguous and refuses for inspection, even if the directory appears empty.
Do not use MkdirAll to silently adopt an existing environment. Its fleet-level
worktree parent may be created if absent after validating every ancestor.

Record branch-creation intent, create the branch at the recorded SHA with an
atomic absent-ref update and attempt-identifying reflog message, then set upstream.
Record resulting evidence before worktree registration. On uncertain completion,
only an exact ref/SHA plus matching creation evidence permits continuation. A
missing/unreliable reflog refuses for manual inspection instead of guessing.

Model upstream configuration as its own intent/effects. After branch ownership
is proved, observe all values of branch.main-slotN.remote and .merge. Expected
values are the reserved remote name and refs/heads/main. Absent values are safe
to fill after persisting configuration intent; matching values are idempotent.
Conflicting or duplicate values refuse without replacing them. Write missing
keys separately and re-read both before publishing upstream-ready. A crash after
either write is reconciled by the same rule on explicit retry. No worktree-add
effect is admitted until the ref and both upstream keys are proved. Apply this
repair only to an attempt-owned new branch; adoption of an existing complete
host requires its already-unambiguous upstream and never rewrites it.

Create the host with `git worktree add <host-path> main-slotN` without force.
Reconcile an interrupted add by checking SDLC identity, registration, .git link,
expected administrative directory and captured creation evidence. A partially
registered host that cannot meet that proof remains retry-needed with precise
inspection guidance; never prune/remove it automatically. A complete compatible
host can continue after verification. Source files are never copied manually.

For adopted existing hosts, --retry accepts only a complete Git-verified slot
with unambiguous upstream and no conflicting creation record. It may already be
on an issue branch or dirty. Baseline is its existing resting ref, not fetched
main. No fetch or host ref mutation is performed; explicit Weave setup establishes
bindings/readiness while retaining local dependency and host work.

### State/event/effect table

States encode known progress; failure carries the last confirmed phase and a
reason, not an invented rollback. The table is implemented in ProvisionMachine.

| State / observation | Event | Next state / effect |
| --- | --- | --- |
| no record, empty candidate | explicit create | resolve remote, persist fetch intent/private ref |
| fetch-intent, private ref absent | continue / explicit retry | atomic fetch into private + tracking refs |
| fetch outcome uncertain | retry | inspect owned private ref; pin it or refetch if absent |
| baseline observed in private ref | capture succeeds | persist baseline, reserve creation |
| reserved baseline, destination absent | continue | create environment, record identity |
| reserved, environment ownership proved | continue | record branch intent, create-only ref |
| branch intent, effect uncertain | retry | inspect ref/reflog; confirm or refuse |
| owned branch, upstream missing/partial | continue / explicit retry | persist config intent, fill only absent expected keys |
| owned branch, upstream conflicts/duplicates | any | refuse without config replacement |
| owned branch, verified complete upstream | continue | register worktree |
| add intent, effect uncertain | retry | inspect registration/host; confirm or refuse |
| verified host | continue | bind host, persist host-created |
| host-created | setup request | persist preparing, invoke Weave |
| preparing | exit 0 + final valid bindings | persist ready, delete exact temporary ref, return result |
| ready with cleanup-pending ref | next request | compare-and-delete owned ref, retain pending if failed |
| preparing | nonzero/cancel/unknown outcome | persist retry-needed, retain progress |
| retry-needed | ordinary request | report explicit retry instruction |
| retry-needed | explicit retry, evidence valid | resume last confirmed phase |
| ready | request, identity/bindings valid | return reused; no fetch/compile |
| ready | host identity missing/changed | refuse; retain evidence |
| ready | dependency binding missing/changed | require retry; preserve files |
| existing verified host, no record | ordinary request | report --retry requirement |
| existing verified host, no record | explicit retry | capture existing refs, setup |
| any | malformed evidence/path or foreign state | refuse, no new mutation |
| any in-flight | second request | busy; no duplicate effects |
| any in-flight | caller death | leases protect writers; next caller reconciles |

Every emitted mutating effect has a recorded intent or a proved compatible
existing host. Store/lease failures stop the machine before further effects.
A failed final record write leaves preparation unconfirmed; explicit retry may
repeat idempotent Weave setup. Never report ready from only a directory or exit
status when final identity validation fails.

## Operating envelope and trust

These are engineering defaults, not measured performance claims:

- Local identity/Git probes: 5 seconds each; bounded 1 MiB structured output.
- Fetch: 120 seconds; setup: 20 minutes; both cancellable. Timeout preserves
  partial state and explains explicit retry. Inject durations in tests.
- Lease acquisition: nonblocking; ready path has no network/build subprocesses.
- Provisioning runs on caller's worker context, never on the console event loop.
- Keep at most 64 KiB diagnostic tail per operation and 64 KiB stored last error;
  stream progress separately with backpressure bounded by the caller's writer.
- Record maximum 256 KiB, 256 dependency bindings, one current generation per slot.
  Exceeding limits refuses before publishing partial ready evidence.
- One synchronous process at a time per slot; caller owns cancellation and joins
  the direct child. No background cleanup worker or retry timer outlives the call.

ProvisionProcess uses argv and an owned process group, a context, inherited lease
files and bounded pipes. On cancellation send SIGTERM, allow 2 seconds, then kill
the owned direct process group if it remains; WaitDelay is 1 second. Preserve an
uncertain receipt if completion/descendant release cannot be confirmed. Never
signal an unrelated PID found in a receipt. Weave handles its own subgroups;
surviving descendants hold its setup lease and cause retry refusal until quiescent.

For the internal CLI, add a scoped signal.NotifyContext for this operation's
SIGINT/SIGTERM path and pass it into dispatch; do not change unrelated operations'
signal behavior. Stop signal registration on return. All subprocesses share that
context plus phase deadlines. OSRuntime wires a stderr progress callback without
changing launch/terminal byte routing.

JSON, receipts, process responses and paths are untrusted data (ARCH-SECURE).
Use strict parsing, no-follow regular-file opens, bounded input, canonical paths
and Git evidence. Weave manifests and generators remain executable developer
inputs under its existing trust model. Avoid logging credential-bearing URLs;
progress/errors from external tools should redact URL userinfo and credential
query parameters before storing or rendering. Tests use fake credentials only.

Worktree files/dependency clones persist intentionally per user-created slot.
Records and lock files cost bounded metadata per slot and survive resume; explicit
managed-environment removal retires them. No new per-launch/per-keystroke artifact.
Owned publication temp files are collected under the per-slot lease after failed
publication/restart; never clean unknown environment contents (ARCH-FUNERAL).

## Architecture decisions

- ARCH-DRY: consume SDLC identity and Weave setup; reuse dispatcher, strict JSON
  and atomic write helper; no second dependency graph/resolver/registry.
- ARCH-PURE: request parsing, number policy, observations and transition decisions
  are pure; IO is confined to ProvisionIO/controller/lease/process/store.
- ARCH-PURPOSE: ship a reachable internal operation, result rendering, retry and
  preserved-work tests, not an unused provisioning helper.
- ARCH-MOCK: one stateful ProvisionFixture models filesystem/ref/lease/process
  evolution behind the production seam; compare consumed behavior to real tools.
- ARCH-CONSTRAINTS: explicit phase bounds, output/record caps, nonblocking locks,
  bounded child extent and no implicit expensive ready-resume work.
- ARCH-SECURE: parser/identity/ownership checks at boundaries; a receipt or path
  cannot alone authorize taking a directory or deleting user work.
- ARCH-ORDER: production effects run only through the machine, with durable intent
  and observable uncertain outcomes; test interrupted/interleaved sequences.
- ARCH-FUNERAL: reusable environments intentionally persist; metadata replaces
  prior generations, with owned temporary-file cleanup, one owned fetch ref per slot until reconciled, and bounded diagnostics.

## Chunk 1: Implement and verify one provision operation

### Task 1 — transport, requests and pure policy

Files: create workspace_identity.go, provision_model.go, provision_select.go
and corresponding `_test.go` files under cmd/internal/couchcore.

- [ ] Write `TestWorkspaceIdentityV2Contract` using captured real primary and
  slot JSON fixtures; assert nullability, exact requested repo/slot/path identity,
  bad schema, duplicate keys, missing fields and trailing-object refusals.
- [ ] Write `TestProvisionRequest` for slot/remote/retry grammar and
  `TestSelectWorkspaceNumber` for holes, :2/:10 numeric order, occupied/unknown/
  partial entries, reuse priority, duplicate identities and integer limits.
- [ ] Run `go test ./cmd/internal/couchcore -run 'Test(WorkspaceIdentity|ProvisionRequest|SelectWorkspaceNumber)' -count=1`.
  Expect failures naming missing symbols/behavior, not environmental setup.
- [ ] Implement pure types/parsers/selector. Use explicit variants for unknown
  occupancy; do not treat probe error as absence.
- [ ] Rerun the targeted command and commit with `#305` and model trailer.

### Task 2 — transition model and stateful integration fixture

Files: provision_model.go, provision_model_test.go, provision_fake_test.go,
provision_test.go (new).

- [ ] Write table/sequence tests from every state/event row above. Assertions
  name invariants: baseline never moves, no foreign takeover, no ready before
  setup success, no repeated fetch on retry, no new effect after store failure.
- [ ] Model refs/upstreams, worktree membership, directory ownership, durable
  intents, setup clone state, inherited leases and controllable child barriers
  in ProvisionFixture. Failures can occur before an effect, after it, or during
  durable evidence publication; those are distinct outcomes.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestProvision(Machine|Sequence)' -count=1`.
  Confirm failing assertions, implement transitions, then rerun to green.
- [ ] Fuzz bounded event sequences; assert independent invariants and model only
  reachable ownership. Rejected events must emit zero mutations.
- [ ] Commit model and fake before layering OS glue over them.

### Task 3 — durable records, leases and process execution

Files: provision_store.go, provision_lease_unix.go, provision_process_unix.go,
matching `_test.go` files and provision_subprocess_test.go (new).

- [ ] Write bounded strict-record decode/round-trip and atomic publication tests,
  including truncation, unknown version, oversized records, symlink parents,
  interrupted rename, temp ownership and read/write failure propagation.
- [ ] Write cross-process lease tests: direct child retains flock after parent
  death; another caller remains busy; closing the last inherited descriptor
  permits retry. An explicit LOCK_UN mutation must break this test.
- [ ] Write process tests for cancellation, late exit, full output pipe, bounded
  diagnostics, phase timeout and child-start failure. Use isolated process
  fixtures with deterministic pipes/barriers, not sleeps as ordering evidence.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestProvision(Store|Lease|Process)' -count=1`.
- [ ] Implement OS file/lease/process helpers. Reuse atomic publication, but
  close-only inherited leases must not call existing unlockThreadStoreFile.
- [ ] Verify cancellation through real subprocess helpers and commit.

### Task 4 — controller and actual Git conformance

Files: provision.go, provision_io.go, provision_git_test.go,
provision_conformance_test.go (new); workspace_identity.go integration.

- [ ] Write end-to-end controller tests with the stateful fixture for new :1/:2,
  ready reuse, explicit retry, adopted complete host, every interrupted phase,
  invalid receipt/binding, and foreign directory/ref collisions.
- [ ] Create real temporary bare remotes and primary repos. Give local HEAD a
  different commit and dirty/untracked files; assert new host HEAD equals fetched
  remote main while source state and files remain byte-for-byte unchanged.
- [ ] Cover non-origin/multiple/missing remotes, remote default branch other
  than main, SHA formats, path spaces, hyphenated names, branch checked out in
  another tree, upstream mismatch and no forced ref/worktree operations.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestProvision(Controller|Git)' -count=1`.
- [ ] Implement controller as Observe → machine event → persist intent → effect
  → observe outcome; do not bypass the model for convenient retries.
- [ ] Test source/remote movement between simultaneous slot fetches, exact private-OID capture,
  partial upstream writes, conflicting config, owned-ref cleanup, and effect
  acknowledgment loss. Same-slot
  concurrent requests converge on one identity; different slots do not renumber.
- [ ] Add `PAIR_LIVE_WORKSPACE=1` conformance using installed SDLC v2 and Weave
  against a temporary minimal repo with no packages/tools/generators. Real Git
  uses file remotes only for the host; Weave fixture has no local-source deps.
  Fake integration models private dependency success/failure, matching #243's
  separately verified remote-clone contract.
- [ ] Run `PAIR_LIVE_WORKSPACE=1 go test ./cmd/internal/couchcore -run '^TestProvisionConformance$' -count=1 -v`.
- [ ] Commit controller and conformance evidence.

### Task 5 — production operation, result and startup compatibility

Files: couchcore/couch.go, ops.go, operationdispatch.go,
couchcmd/run.go; new couchcmd/provision_test.go; existing cli_test.go,
readme_test.go, operationdispatch_test.go, ops_test.go as needed.

- [ ] Declare `provision-workspace`: PresentationInternal, ExecuteDirectStore,
  EffectProcess, ConfirmNone, new ResultWorkspace family. Declare path, slot,
  optional remote and retry arguments through existing ArgSpec grammar.
- [ ] Write CLI → dispatcher → injected controller tests asserting exact inputs,
  one valid result object, stderr progress, no supervisor acquisition, no agent
  launch or AllocateThreadTag call. Verify failure returns nonzero and no result.
- [ ] Add signal cancellation to this CLI operation and test bounded shutdown
  via a gated child, preserving retryable progress.
- [ ] Wire Couch.Workspaces in OSRuntime.NewCouchWith, dispatch to its entry
  point and render ProvisionResult as JSON. Tests override the seam explicitly;
  no fake falls back to real user filesystem/PATH unexpectedly.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchcore -count=1`.
- [ ] Verify existing PrepareStart/StartInteractive/SpawnPrepared tests still
  pass and that they perform no implicit provision/fetch/compile effects.
- [ ] Commit production wiring.

### Task 6 — documentation, verification and one close boundary

Files: README.md, atlas/couch.md, atlas/index.md, new
atlas/workspace-provisioning.md, issue #305 and project couch-slots-v2.md.

- [ ] Document exact internal invocation, setup/retry/provenance output,
  remote-main default, readiness meaning, later local-work transfer, durable
  paths, explicit recovery and #306 caller lease/occupancy responsibilities.
- [ ] Map components and source seams in the new atlas page and link it from
  both index and Couch map. Keep operator guidance free of storage internals.
- [ ] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`.
- [ ] Run `make runtimebundle-generate`, then `go test ./... -count=1` and
  `go vet ./cmd/internal/couchcore ./cmd/internal/couchcmd`.
- [ ] Build the normal binaries (`make pair couch`) and run the new internal operation
  against temporary fixture repos with COUCH_STORE_DIR, PAIR_DATA_DIR, HOME and
  XDG data/config roots supplied by the harness, not production directories.
  Use task-specific shell variable names for those root paths.
- [ ] Verify :1/:2 paths/refs/upstreams; dirty ready reuse is unchanged;
  interrupted setup refuses ordinary reuse and succeeds only on explicit retry.
- [ ] Reconcile issue checkboxes, project state and evidence. Run diff --check.
- [ ] Commit implementation/evidence, then `sdlc close --issue 305 --verified
  '<actual commands/results and preservation/concurrency evidence>'` once.
  The binary owns the fresh boundary review. Fix important findings before ship.
- [ ] Publish through `sdlc pr` and `sdlc merge` after completion, following the
  existing issue workflow and preserving unrelated files in this checkout.

## Approval and next checkpoint

This plan is the reviewable engineering record. Run fresh-context plan review,
resolve material findings, and checkpoint issue/plan before presenting it.
After operator approval, `sdlc change-code --issue 305` owns plan-quality,
estimate derivation and the in-place implementation branch. No separate worktree
is created for this Parley development session.

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
