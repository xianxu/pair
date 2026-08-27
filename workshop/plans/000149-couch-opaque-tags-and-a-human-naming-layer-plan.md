---
issue: 000149
status: approved
created: 2026-08-26
updated: 2026-08-26
---

# Couch durable work threads implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to
> determine the appropriate execution approach: use
> superpowers-subagent-driven-development when a task is independently
> capturable, otherwise superpowers-executing-plans. Steps use checkbox syntax
> for tracking. Execute milestone by milestone; each `Mx` closes with
> `sdlc milestone-close` before the next begins.

**Goal:** Replace couch's tree-keyed actor snapshot with safely admitted,
durably named work threads in one restartable singleton namespace, including
remembered per-path/per-agent launch profiles.

**Architecture:** One lock/revision/WAL-backed ThreadStore owns composite thread
records inside the once-canonicalized couch store namespace. One supervisor
lease owns actor creation and terminals; Ariadne supplies normalized admission
policy, while Pair supplies the discoverable operation and artifact surfaces.

**Tech stack:** Go, JSON, OS advisory locks and process identity, atomic file
replacement/WAL recovery, `crypto/rand`, Pair launcher/zellij integration, and
Ariadne's `sdlc fleet policy --json` protocol.

---

Verified park/resume remains #152; managed worktree provisioning/rebinding
remains #153. #149 returns typed refusals for those unavailable outcomes.

## Architecture

### Core concepts

Inventory rule: every milestone-added or milestone-modified architectural
entity has one greppable row with its actual kind, path, and latest milestone
status. Each boundary sweeps the whole milestone diff against this table rather
than adding only the entity named by a review finding.

| Entity | Kind | Lives in | Status |
|---|---|---|---|
| `CouchNamespace` / `ResolveCouchNamespace` | integration | `cmd/internal/couchcore/namespace.go` | new in M1 |
| `PolicyResult` / `PolicyCapacity` | pure | `cmd/internal/couchcore/policyresolver.go` | new in M1 |
| `AdmissionDecision` | pure | `cmd/internal/couchcore/admission.go` | new in M1 |
| `ThreadAddress` / `ThreadRecord` | pure | `cmd/internal/couchcore/thread.go` | new in M1, widened in M2, M3, and M4 |
| `StartTransaction` | pure | `cmd/internal/couchcore/starttransaction.go` | new in M2, widened in M4 |
| `ThreadMetadataPatch` / `ApplyThreadMetadata` | pure | `cmd/internal/couchcore/threadmetadata_model.go` | new in M3 |
| `ThreadSummary` / `BuildThreadInventory` | pure | `cmd/internal/couchcore/threadinventory.go` | new in M3 |
| `Operation` / `ArgSpec` / `Operations` effect-owner declarations | pure | `cmd/internal/couchcore/ops.go` | modified in M1; split from executors in M3; launch selection and value-bearing flags added in M4 |
| `bindArgs` | pure | `cmd/internal/couchcmd/run.go` | value-bearing flag contract modified in M4 review disposition |
| `threadrecord.Record` / `ValidatePersisted` / `DecodePersisted` | pure | `cmd/internal/threadrecord/record.go` | new in M3 boundary disposition, widened in M4 |
| `strictjson.Decode` | pure | `cmd/internal/strictjson/decode.go` | extracted in M3 boundary disposition |
| `LaunchProfile` / `PathLaunchPreference` / `LaunchProfileResolution` | pure | `cmd/internal/couchcore/launchprofile.go` | new in M4 |
| `AgentInventory` | pure | `cmd/internal/launcher/agent_defaults.go` | shared inventory added in M4 |
| `couchLaunchProfileWire` / `BuildCouchLaunchProfile` / `ApplyCouchLaunchProfile` | pure | `cmd/internal/launcher/launch_args_policy.go` | strict one-shot codec added in M4 |
| `mergeChildEnvironment` | pure | `cmd/internal/couchcore/runner.go` | authoritative child overlay added in M4 review disposition |
| `MigrateLegacyRecord` | pure | `cmd/internal/couchcore/migration.go` | new in M5 |
| `artifactpath.Address` / `Paths` / `ScopePaths` / `Binding` | pure | `cmd/internal/artifactpath/paths.go` | new in M5 |
| `artifactpath.PairCachePaths` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |
| `artifactpath.ScrollbackArtifactSet` / `ParkedScrollbackArtifactSet` / `ChangelogArtifactSet` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |
| `artifactpath.Family` / `SourceKind` / `SourceClassification` / `NonArtifactSources` | pure declarations | `cmd/internal/artifactpath/manifest.go` | new in M5; exhaustive source inventory added in boundary disposition |
| `artifactpath.VocabularyContext` / `VocabularyAllowance` | pure declarations | `cmd/internal/artifactpath/manifest.go` | added in M5 boundary disposition |
| `artifactpath.ResolvedBinding` / `ResolvedBindings` | pure declarations | `cmd/internal/artifactpath/manifest.go` | added in M5 boundary disposition |
| `artifactpath.LegacyRootPaths` / `LegacyPaths` / `TagFromHistorySidecar` | pure | `cmd/internal/artifactpath/paths.go` | added in M5 boundary disposition |
| `DecodeSessionNameIndex` | pure | `cmd/internal/launcher/session_index.go` | added in M5 boundary disposition |
| `StandaloneThreadRegistration` / `StandaloneThreadRegistrar` | pure seam types | `cmd/internal/launcher/runtime.go` | new in M5 |

- **`CouchNamespace`** — the absolute physical store path used as the durable
  namespace key. One namespace owns many work threads and zero or one live
  supervisor; process incarnation is not identity.
- **`PolicyResult` and `AdmissionDecision`** — normalized provider evidence and
  the pure occupancy decision over it. They prevent declaration parsing or
  subprocess behavior from entering admission logic.
- **`ThreadAddress` and `ThreadRecord`** — `{repo_scope, tag}` selects exactly
  one durable thread; the record owns zero or more migration-era incarnations.
- **`StartTransaction`** — nonce-addressed pure state transitions around a
  blocked helper. Future park/resume widens transitions rather than inventing a
  second lifecycle authority.
- **`Operation` declaration** — one schema row for every effectful human, CLI,
  and advisor action, including implicit switch/attach. Future clients derive
  capabilities instead of copying verbs.
- **`LaunchProfileResolution`** — independently records agent provenance and
  argv provenance so valid source combinations remain representable.
- **`artifactpath.Family` and `SourceClassification`** — checked inventory of
  every tag-bearing Pair path and every source allowed to mention its filename
  token. `ResolvedBinding` supplies positive resolver/member evidence,
  `VocabularyAllowance` closes exact non-path uses, and `NonArtifactSources`
  makes source participation exhaustive rather than token-discovered; new
  sidecars or production files extend the manifest rather than inheriting an
  implicit default.

### Integration points

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| `CouchNamespace` / `ResolveCouchNamespace` | `cmd/internal/couchcore/namespace.go` | new in M1 | startup cwd, mkdir, physical-path resolution |
| `SupervisorLease` | `cmd/internal/couchcore/supervisorlease.go`, `supervisorlease_unix.go` | new in M1 | non-inheritable OS advisory lock and owner metadata through existing `ProcOps` identity |
| `PolicyResolver` / `ExecPolicyResolver` | `cmd/internal/couchcore/policyresolver.go`, `policyresolver_exec.go` | new in M1 | `sdlc fleet policy` subprocess |
| `ThreadStore` | `cmd/internal/couchcore/threadstore.go` | new in M1, widened in M2, M3, M4, and M5 | filesystem lock, per-thread/path-preference records, WAL/manifest, versioned legacy enrichment |
| `ThreadStore.ApplyThreadMetadata` | `cmd/internal/couchcore/threadmetadata.go` | new in M3 | revision-CAS store transition and reference snapshot |
| `LaunchHelper` | `cmd/internal/couchcore/launchhelper.go`, `cmd/pair-launch-helper/main.go` | new in M2 | fork/exec acknowledgement boundary |
| `SessionQuiescence` | `cmd/internal/launcher/session_quiescence.go` | new in M2 | observable zellij session/server teardown |
| `DispatchOperation` / `OperationExecutors` / `DirectStoreExecutor` / `CouchLiveOwnerExecutor` | `cmd/internal/couchcore/operationdispatch.go` | new in M3 | validated dispatch, direct-store effects, and optional live-owner effects |
| `Console.ExecuteConsoleOperation` | `cmd/internal/couchtty/console.go` | new in M3 | terminal-local switch and typed attach effects |
| `Couch.resolveLaunchProfile` / `RepoAgentDefault` | `cmd/internal/couchcore/couch.go`, `cmd/internal/couchcmd/run.go` | new in M4 | strict path preference reads and injected Pair repo-default reads |
| `OSRuntime.ReadAgentDefault` | `cmd/internal/launcher/osruntime.go` | reused in M4 | Pair-owned scoped argv-default storage |
| `applyCouchLaunchEnvironment` | `cmd/internal/launcher/runcli.go` | new in M4 | consumes the tag-bound profile and cross-checks repo-default provenance at Pair entry |
| `ExecRunner` / `buildExecCommand` | `cmd/internal/couchcore/runner.go` | modified in M4 review disposition | one production command path over inherited environment with authoritative child-key overlay |
| `ThreadStore.MigrateLegacyRecords` | `cmd/internal/couchcore/migration.go` | new in M5 | one locked journal transaction over cutover records and manifest completion |
| `LaunchNativeWithStandaloneRegistrar` / `RegisterStandalonePair` | `cmd/internal/launcher/runcli.go`, `cmd/internal/couchcore/standalone.go`, `cmd/pair-go/main.go` | new in M5 | composition-root injection of direct Pair registration without reversing the launcher→Couch package boundary |
| `ThreadStore.UpsertStandalonePair` | `cmd/internal/couchcore/standalone.go` | new in M5 | locked/revisioned direct-Pair incarnation publication with metadata preservation |
| `OSRuntime.ReadSessionNameIndex` | `cmd/internal/launcher/osruntime.go` | modified in M5 and its boundary disposition | strict merge of legacy-global and selected-scope durable bindings; missing files mean empty, malformed/unreadable files fail closed |
| `readSessionNameIndexes` | `cmd/internal/launcher/session_index.go` | added in M5 boundary disposition | one injected-IO overlap reader used by runtime, address claim, and quiescence |

Every integration has a stateful fake or real-process conformance test. In
particular, namespace/lease tests use independent processes and inherited file
descriptors; policy tests use a changing stateful resolver plus the live
Ariadne binary; store tests use independent store instances over one directory.

### Risky function test strategies

| Function | Adversarial strategy and mechanical guard |
|---|---|
| `ResolveCouchNamespace` | Fuzz path spellings/filesystem alias fixtures; require one absolute physical output or typed refusal. |
| `AcquireSupervisorLease` / `VerifiedOwner` | Stateful `ProcOps` plus subprocess crash/FD probes; display only identity proven against the held lock. |
| `DecodePolicyResponse` | Fuzz arbitrary bytes, discriminator deletion, and exit/output combinations; accept exactly one validated envelope. |
| `RecoverStoreJournal` | Model crash at each state transition and corrupt after-images; roll forward only expected-before/exact-after states. |
| `AllocateThreadTag` | Script entropy failures and collision streams; no-replace claim or typed failure within eight attempts. |
| `ValidateThreadRecord` | Fuzz composite/schema values; reject invalid boundaries while defensively copying accepted state. |
| `ReconcileAdmission` / `Admission.Decide` | Stateful provider/store interleavings; commit only one manifest generation and coherent provider epoch, never fork on refusal. |
| `AdvanceStartTransaction` / `ReconcileStart` | Generate interruption event sequences; every state has at most one tracked helper and occupied-or-proven-free capacity. |
| `ResolveThreadReference` | Fuzz duplicate/fuzzy candidates; exact scoped tag wins and every ambiguous result refuses with candidates. |
| `threadrecord.ValidatePersisted` / `DecodePersisted` | Mutate every required top-level, address, incarnation, start-claim, policy-shape, generation, and path/address invariant; Couch and standalone Pair must both reject each invalid record. |
| `ApplyThreadMetadata` / `BuildThreadInventory` | Generate stale revisions and equal-path records; preserve independent fields and one exact-state row per thread. |
| `DispatchOperation` | Schema-surface conformance plus stale targets; effect executes only through its declared executor and owner absence never forks. |
| `ResolveLaunchProfile` | Cartesian source properties; agent and argv provenance resolve independently and never cross agents. |
| `MigrateLegacyRecord` | Model interruption/corruption/repeated tags; rerun is idempotent and never drops unreadable input or occupancy. |
| `artifactpath.Resolve` / manifest coverage | Fuzz traversal and scan all production constructors; output remains scoped and every constructor is classified. |

Task test steps below name these functions and their focused/full commands; this
table owns adversarial classes so tasks do not duplicate case inventories.

### One authority, introduced incrementally

`couchcore.ThreadStore` is the only mutable authority for thread records,
incarnations, admission evidence, start transactions, launch preferences, and
migration state. It owns one global Pair-data lock and performs every mutation
as read/validate/change/revision/write while holding that lock. Callers receive
defensive values and use expected revisions for compare-and-swap updates; they
never retain a mutable registry snapshot and later overwrite the store.

M1 needs durable `creating` reservations to make policy admission safe across
processes. Therefore M1 introduces the non-throwaway ThreadStore transaction
kernel, composite opaque identity for new starts, and the
admission/incarnation subset of its schema. M2 widens the same schema and API to
journaled starts and the pre-exec helper. This is a dependency correction to the
issue's shorthand milestone rows, not a second store or a change in final ownership
(ARCH-DRY, ARCH-PURE).

Use an OS advisory lock around the store directory, acquired through an
injectable lock/file seam. Each `{repo_scope, tag}` has one atomic record file;
there is no monolithic all-thread snapshot in the final schema. An update to one
existing record uses atomic rename plus that record's revision and does not
change the manifest. Create/delete/cutover and every multi-record mutation
change manifest membership/generation and use a store-level write-ahead journal
containing all expected-before values and after-images: write the intent
durably, replace targets idempotently, advance the manifest, then clear the
journal. Recovery under the lock accepts each target only at expected-before or
the exact after-image and rolls forward; any third state fails closed. Admission
captures manifest generation so concurrent cohort membership changes force a
retry. Lock/recovery failures are hard errors. Tests use two independently
constructed stores against one directory, not merely goroutines sharing one
object.

### One durable namespace, one live supervisor

Resolve `COUCH_STORE_DIR` once at couch process entry: make an explicit relative
path absolute against startup cwd, clean it, create the directory, resolve its
physical symlinks, and use that exact absolute value for the lifetime lease,
ThreadStore, and inherited environment. The path—not a process
ID—is `CouchNamespace`; restart adopts the same inventory.

The actor-owning process holds a namespace `SupervisorLease` for its lifetime.
Its lock descriptor is close-on-exec/non-inheritable. Only after acquisition
does it atomically publish PID plus process-start identity; a refusal verifies
that identity before displaying it. Console and `--no-console` supervisors obey
the same lease. Read/metadata operations may use the locked store when declared
safe, but actor creation/attach/switch is owner-required and refuses externally
until #147 adds local owner routing (ARCH-PURPOSE, ARCH-MOCK).

Operation declarations contain typed arguments/results, effect class,
confirmation, and execution class, but no execution closures. Generic dispatch
receives an `OperationExecutors` context with a direct-store executor and an
optional live-owner executor. Missing live-owner execution returns a typed
`owner routing requires #147` refusal; #149 creates no endpoint, socket, client,
or discovery mechanism. In M3 the console supplies the live-owner handlers;
#147 later transports a request to those same handlers (ARCH-DRY, ARCH-PURE).

### Policy is evidence, not configuration

Add an injected `PolicyResolver` whose production implementation executes
`sdlc fleet policy --path <canonical-path> --json` with a deadline. Decode into
Pair-owned tagged values for capacity (`bounded{limit}` or `unbounded`) and the
bounded-capacity action (`reject` or `provision-worktree`). Pair does not decode
fleet declarations and does not duplicate Ariadne's admission-key-kind model.

Persist the normalized repo identity, admission key, capacity, action,
provider schema version, and canonical declaration digest with each creating,
live, or unknown incarnation. Before admitting a start, capture relevant record
revisions under the store lock, release it, resolve the candidate, and—when
version/digest differ—resolve every live/unknown/creating incumbent in the same
repository. Reacquire the lock, verify the captured revisions/evidence are
unchanged, and either apply a pure group/claim mutation or retry the
read/resolve phase. Pair-client death does not prove whole-incarnation
quiescence, so M1 never releases durable capacity from PID evidence; #152 owns
that proof and transition. Provider subprocess IO never runs under the
global lock and no admission decision uses a stale snapshot. Every refreshed
result in one repository must have the candidate's provider version and
declaration digest; a mismatch retries the entire cohort, including the
candidate, at most three times and then returns a typed fail-closed
`policy-unstable` refusal. Resolution failure remains occupied and fails
closed. A pure `Admission.Decide` counts normalized-key occupants and returns
either admit or a typed refusal/action. No refusal path forks a child
(ARCH-MOCK, ARCH-PURPOSE).

### A start is a durable transaction

M2 makes start a nonce-addressed state machine:

1. canonicalize the physical requested path and resolve policy/profile;
2. under the ThreadStore lock, atomically claim the composite thread and a
   `creating` reservation after admission;
3. fork a tiny Pair launch helper which reports PID/process identity and blocks
   on a close-on-exec acknowledgement channel;
4. durably record the helper identity, acknowledge it, and allow exec;
5. after Pair readiness/registration evidence, atomically promote the
   reservation to a live incarnation and record the successful profile.

EOF or timeout before acknowledgement makes the helper exit before zellij,
agent, editor, or any other workspace-writing process can start. A pre-fork
failure removes only the matching pristine nonce. A post-fork registration
failure first stops and verifies the helper/child; if verification is unknown,
the occupied recovery record remains. Startup reconciliation uses nonce plus
owner/helper PID identities to finish or report a transaction idempotently.

### Composite identity reaches every artifact

`ThreadAddress{RepoScope, Tag}` is the durable address. The repo scope chooses
Pair's existing repository-scoped data directory; the tag chooses files within
it. New tags are `couch-` plus 16 lowercase hexadecimal digits read from
`crypto/rand`, claimed atomically with at most eight collision retries after
checking both ThreadStore records and scoped tag artifacts. Entropy failure and
retry exhaustion are distinct errors.

All artifact APIs use a `launcher.ScopedPaths` (or an equivalent validated
composite value) rather than joining a caller-provided tag to a global path.
Standalone Pair derives scope from its canonical working repository. Couch
passes `COUCH_THREAD_SCOPE` and `COUCH_THREAD_TAG`; no tag-only cross-scope
fallback exists (ARCH-PURPOSE).

### Metadata and launch preferences

Names, descriptions, and advisor-published summaries are separate ThreadStore
fields. Names need not be unique. Exact composite tag wins; name/path lookup is
repo-scoped and returns all candidates rather than guessing. Zellij's
`SessionNameEntry.SessionName` remains only a socket/display binding.

Each successfully registered incarnation records its exact agent and argv.
Path preferences are keyed by normalized repo identity plus canonical physical
path and store `last_agent` plus the last argv for each agent. New-thread agent
resolution is explicit selection, then path `last_agent`, then root actor agent.
Arguments resolve from that path's entry for the selected agent, then Pair's
repo-scoped default for that agent. Only successful registration updates either
the thread or path preference. Agent choices come from Pair's shared agent
inventory, never a couch enum (ARCH-DRY).

## Chunk 1: Milestone M1 — normalized policy consumer and safe admission

### Task 1: Canonicalize the namespace and enforce one supervisor

**Files:**

- Create `cmd/internal/couchcore/namespace.go`
- Create `cmd/internal/couchcore/namespace_test.go`
- Create `cmd/internal/couchcore/supervisorlease.go`
- Create `cmd/internal/couchcore/supervisorlease_unix.go`
- Create `cmd/internal/couchcore/supervisorlease_test.go`
- Create `cmd/internal/couchcore/supervisorlease_subprocess_test.go`
- Modify `cmd/internal/couchcore/ops.go`
- Modify `cmd/internal/couchcore/ops_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/procops.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`

**Steps:**

- [x] **Step 1:** Write failing `ResolveCouchNamespace`, `AcquireSupervisorLease`, and
   `VerifiedOwner` tests using the strategy table.
- [x] **Step 2:** Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run
   'Test(CouchNamespace|StoreNamespace|SupervisorLease)' -count=1`; expect FAIL.
- [x] **Step 3:** Implement the namespace value, non-inherited lifetime lease, verified
   `ProcOps` owner metadata, and operation execution classification specified in
   Architecture. Keep the lifetime lease independent from ThreadStore locking.
- [x] **Step 4:** Rerun the focused command and the subprocess crash probe; expect PASS.
- [x] **Step 5:** Commit the namespace/lease slice with `git add` on only the files above and
   `git commit -m '#149 M1: establish the couch supervisor namespace'`.

### Task 2: Specify the provider seam and strict decoder

**Files:**

- Create `cmd/internal/couchcore/policyresolver.go`
- Create `cmd/internal/couchcore/policyresolver_test.go`
- Create `cmd/internal/couchcore/policyresolver_fake.go`
- Create `cmd/internal/couchcore/policyresolver_exec.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`

**Steps:**

- [x] **Step 1:** Write failing fuzz/table tests for `DecodePolicyResponse` using the
   strategy table and production seam contract.
- [x] **Step 2:** Run `go test ./cmd/internal/couchcore -run TestPolicyResolver -count=1`;
   expect FAIL.
- [x] **Step 3:** Implement defensive normalized values, stateful fake, strict decoder,
   deadline-bound executor, and couchcmd injection; Pair never parses fleet
   declarations or models admission-key kind.
- [x] **Step 4:** Rerun the focused command; expect PASS, then commit the provider slice.

### Task 3: Introduce the ThreadStore transaction kernel

**Files:**

- Create `cmd/internal/couchcore/threadstore.go`
- Create `cmd/internal/couchcore/threadstore_test.go`
- Create `cmd/internal/couchcore/thread.go`
- Create `cmd/internal/couchcore/thread_test.go`
- Create `cmd/internal/couchcore/threadtag.go`
- Create `cmd/internal/couchcore/threadtag_test.go`
- Create `cmd/internal/couchcore/storelock.go`
- Create `cmd/internal/couchcore/storelock_unix.go`
- Create `cmd/internal/couchcore/storelock_test.go`
- Create `cmd/internal/couchcore/storejournal.go`
- Create `cmd/internal/couchcore/storejournal_test.go`
- Modify `cmd/internal/couchcore/store.go`
- Modify `cmd/internal/couchcore/store_test.go`
- Modify `cmd/internal/couchcore/actorid.go`
- Modify `cmd/internal/couchcore/clock_test.go`
- Modify `cmd/internal/launcher/tag.go`
- Modify `cmd/internal/launcher/tag_test.go`

**Steps:**

- [x] **Step 1:** Write failing `UpdateExistingThread`, `RecoverStoreJournal`, and
   `AllocateThreadTag` tests using the strategy table.
- [x] **Step 2:** Run `go test ./cmd/internal/couchcore -run
   'Test(ThreadStore|StoreJournal|AllocateThreadTag)' -count=1`; expect FAIL.
- [x] **Step 3:** Implement the M1 composite record/tag and the two persistence classes
   specified in Architecture, including journaled legacy-occupant cutover. New
   starts launch only after their final opaque address is claimed.
- [x] **Step 4:** Rerun focused tests plus independent-store crash probes; expect PASS.
- [x] **Step 5:** Commit the ThreadStore/identity slice.

### Task 4: Make admission pure and conservative

**Files:**

- Create `cmd/internal/couchcore/admission.go`
- Create `cmd/internal/couchcore/admission_test.go`
- Modify `cmd/internal/couchcore/procops.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/guard_live_test.go`

**Steps:**

- [x] **Step 1:** Write failing pure/stateful tests for `Admission.Decide` and
   `ReconcileAdmission` using the strategy table.
- [x] **Step 2:** Run `go test ./cmd/internal/couchcore -run TestAdmission -count=1`;
   expect FAIL.
- [x] **Step 3:** Implement pure admission plus optimistic unlocked-IO reconciliation as
   specified in Architecture. Fork only after committed admission; uncertain
   post-fork state remains occupied until M2.
- [x] **Step 4:** Rerun focused tests with independent Couch/store instances; expect PASS.
- [x] **Step 5:** Commit the admission slice.

### Task 5: Remove every local policy bypass

**Files:**

- Delete `cmd/internal/couchcore/policy.go`
- Modify `cmd/internal/couchcore/registry.go`
- Modify `cmd/internal/couchcore/registry_test.go`
- Modify `cmd/internal/couchcore/ops.go`
- Modify `cmd/internal/couchcore/ops_test.go`
- Modify `cmd/internal/couchcore/startargs.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`
- Modify `cmd/internal/couchcore/store.go`
- Modify `cmd/internal/couchcore/store_test.go`
- Create `cmd/internal/couchcore/policy_shadow_test.go`

**Steps:**

- [x] **Step 1:** Add a source-level shadow sweep that fails on production `PolicyTable`, old
   `Mode` constants, `policy.json`, `Couch.Policy`, and admission use of
   `SameTree`/`--same-tree`.
- [x] **Step 2:** Remove the public `--same-tree` argument and all admission bypass behavior.
   Preserve only the legacy serialized field needed for M5 replay, clearly
   quarantined from new decisions.
- [x] **Step 3:** Replace `TreeOccupiedError.Mode` and policy-specific prose with the typed
   capacity/action refusal. `provision-worktree` names #153 and performs no
   path mutation.
- [x] **Step 4:** Remove `Store.policyPath`, PolicyTable loading, repository-name defaults,
   and tests that bless local policy inference.

### Task 6: Cross-repo conformance and M1 boundary

**Files:**

- Modify `cmd/internal/couchcore/conformance_live_test.go`
- Modify `Makefile`
- Create `.github/workflows/couch-policy-conformance.yml`
- Modify `atlas/couch.md`
- Modify `atlas/index.md` only if a new atlas page is added
- Modify `workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md`
- Modify `workshop/projects/couch.md`

**Steps:**

- [x] **Step 1:** Add `test-couch-policy-live`, accepting an Ariadne `sdlc` binary and
   running production `ExecPolicyResolver` against temporary declarations. Pair
   owns the check; its stateful fake remains the ordinary test backend.
- [x] **Step 2:** Add the Pair-owned weekly and manual GitHub workflow: bootstrap the
   sibling Ariadne dependency, build its current `sdlc`, then run
   `test-couch-policy-live`. Also run the target at M1 close and whenever Pair's
   resolver wire contract changes; the weekly workflow detects Ariadne-side
   drift between Pair changes (ARCH-MOCK).
- [x] **Step 3:** Run focused tests, `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd
   -count=1`, the live provider target against the locally built Ariadne #200
   binary, `go test ./... -count=1`, repository shell/Lua tests, layout checks,
   and `git diff --check`.
- [x] **Step 4:** Run the real-process namespace/lease strategies from the table; assert
   one owner and exact inherited namespace.
- [x] **Step 5:** Update atlas ownership: Ariadne declares/measures/resolves policy; Pair
   validates normalized evidence, owns the singleton namespace, and performs
   runtime admission.
- [x] **Step 6:** Commit the verified M1 window and run
   `sdlc milestone-close --issue 149 --milestone M1`. Fix all Critical/Important
   findings. Only after the Approved boundary may Ariadne #200 close/merge.

## Chunk 2: Milestone M2 — durable identity and recoverable start

### Task 1: Widen the ThreadStore record for recoverable starts

**Files:**

- Modify `cmd/internal/couchcore/thread.go`
- Modify `cmd/internal/couchcore/thread_test.go`
- Modify `cmd/internal/couchcore/threadstore.go`
- Modify `cmd/internal/couchcore/threadstore_test.go`
- Modify `cmd/internal/couchcore/thread.go`
- Modify `cmd/internal/threadrecord/record.go`
- Modify `cmd/internal/threadrecord/record_test.go`
- Modify `cmd/internal/launcher/scope.go`
- Modify `cmd/internal/launcher/scoped_paths.go`
- Modify their colocated tests

**Steps:**

- [x] **Step 1:** Write failing `ValidateThreadRecord` tests from the strategy table.
- [x] **Step 2:** Widen the record for M2 start transactions while preserving M1 identity,
   path, and revision invariants.
- [x] **Step 3:** Run thread/store focused tests; expect PASS, then commit.

### Task 2: Add the blocked pre-exec helper

**Files:**

- Create `cmd/internal/couchcore/launchhelper.go`
- Create `cmd/internal/couchcore/launchhelper_test.go`
- Create `cmd/pair-launch-helper/main.go`
- Modify `cmd/internal/couchcore/runner.go`
- Modify `cmd/internal/couchcore/runner_fake.go`
- Modify `cmd/internal/couchcore/ptyrunner.go`
- Modify their colocated tests

**Steps:**

- [x] **Step 1:** Write subprocess/model tests for the helper boundary and
   `AdvanceStartTransaction` using the strategy table; the oracle is no target
   exec before durable acknowledgement and exactly one exec afterward.
- [x] **Step 2:** Implement the blocked-helper Runner capability specified in Architecture;
   keep OS descriptors outside domain state and mirror behavior in FakeRunner.
- [x] **Step 3:** Run helper/runner focused tests; expect PASS, then commit.

### Task 3: Implement and reconcile the start state machine

**Files:**

- Create `cmd/internal/couchcore/starttransaction.go`
- Create `cmd/internal/couchcore/starttransaction_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/guard_live_test.go`
- Modify `cmd/internal/couchcore/store.go`

**Steps:**

- [x] **Step 1:** Write model tests for `AdvanceStartTransaction` and `ReconcileStart`
   using the strategy table, then run them against `FakeRunner` state.
- [x] **Step 2:** Implement nonce-addressed promotion/reconciliation with the occupied-or-
   proven-free invariant from Architecture and pass the composite address to
   Pair.
- [x] **Step 3:** Run start/recovery focused and restart tests; expect PASS, then commit.

### Task 4: M2 integration boundary

**Files:**

- Modify `atlas/couch.md`
- Modify the issue/project logs

**Steps:**

- [x] **Step 1:** Run the real-process `ReconcileStart` strategy and committed M2 probe.
- [x] **Step 2:** Run the Verification commands and `git diff --check`.
- [x] **Step 3:** Commit and run `sdlc milestone-close --issue 149 --milestone M2`; resolve all
   boundary findings before metadata work.

## Chunk 3: Milestone M3 — metadata, inventory, and standalone resolution

### Task 1: Thread metadata operations

**Files:**

- Create `cmd/internal/couchcore/threadmetadata.go`
- Create `cmd/internal/couchcore/threadmetadata_test.go`
- Modify `cmd/internal/couchcore/ops.go`
- Modify `cmd/internal/couchcore/ops_test.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`

**Steps:**

- [x] **Step 1:** Write failing `ApplyThreadMetadata` and `ResolveThreadReference` tests
   from the strategy table.
- [x] **Step 2:** Implement composite CAS metadata operations and scoped publish context.
- [x] **Step 3:** Run metadata/ops focused tests; expect PASS, then commit.

### Task 2: Shared inventory and rendering

**Files:**

- Create `cmd/internal/couchcore/threadinventory.go`
- Create `cmd/internal/couchcore/threadinventory_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`
- Modify panel-model tests under `cmd/internal/couchcore`/`couchcmd`

**Steps:**

- [x] **Step 1:** Write failing `BuildThreadInventory`/render tests from the strategy table.
- [x] **Step 2:** Implement shared exact-state inventory and name-first common rendering for
   CLI/panel consumers without #151 hierarchy.
- [x] **Step 3:** Run inventory/render focused tests; expect PASS, then commit.

### Task 3: Route every effectful human action through declared operations

**Files:**

- Modify `cmd/internal/couchcore/ops.go`
- Modify `cmd/internal/couchcore/ops_test.go`
- Create `cmd/internal/couchcore/operationdispatch.go`
- Create `cmd/internal/couchcore/operationdispatch_test.go`
- Modify `cmd/internal/couchcmd/run.go`
- Modify `cmd/internal/couchcmd/run_test.go`
- Modify `cmd/internal/couchtty/console.go`
- Modify `cmd/internal/couchtty/console_test.go`
- Modify `cmd/internal/couchtty/panel.go`
- Modify `cmd/internal/couchtty/panel_test.go`

**Steps:**

- [x] **Step 1:** Write `DispatchOperation` schema-surface conformance using the
   strategy table across CLI, explicit/implicit panel effects, and a test-only
   generic consumer. Pure UI navigation is explicitly classified and exempt.
- [x] **Step 2:** Implement declaration-only generic dispatch and console-local typed
   switch/attach handlers specified in Architecture; create no #147/#148 client
   or transport.
- [x] **Step 3:** Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd
   ./cmd/internal/couchtty -run 'Test.*Operation' -count=1`; expect PASS.
- [x] **Step 4:** Commit with `git commit -m '#149 M3: unify human and advisor operations'`
   after adding only the files above.

### Task 4: Standalone Pair lookup and picker

**Files:**

- Create `cmd/internal/launcher/thread_index.go`
- Create `cmd/internal/launcher/thread_index_test.go`
- Modify `cmd/internal/launcher/pick.go`
- Modify `cmd/internal/launcher/pick_test.go`
- Modify `cmd/internal/launcher/createflow.go`
- Modify `cmd/internal/launcher/createflow_test.go`
- Modify `cmd/internal/launcher/session_index.go`
- Modify `cmd/internal/launcher/session_index_test.go`

**Steps:**

- [x] **Step 1:** Write failing standalone index/picker tests using
   `ResolveThreadReference` strategy.
- [x] **Step 2:** Implement scoped ThreadStore lookup/picker display while preserving direct
   Pair prompting and zellij `SessionNameEntry` ownership.
- [x] **Step 3:** Run launcher focused tests; expect PASS, then commit.

### Task 5: M3 integration boundary

**Files:**

- Modify `atlas/couch.md`, `atlas/session-identity.md`, issue/project logs

**Steps:**

- [x] **Step 1:** Run the M3 end-to-end composite identity/metadata scenario and assert
   offline Pair resolution preserves scoped artifacts.
- [x] **Step 2:** Run focused/full suites and `git diff --check`; commit and run
   `sdlc milestone-close --issue 149 --milestone M3`.

## Chunk 4: Milestone M4 — remembered agent and argument profiles

### Task 1: Model preferences and resolution as pure data

**Files:**

- Create `cmd/internal/couchcore/launchprofile.go`
- Create `cmd/internal/couchcore/launchprofile_test.go`
- Modify `cmd/internal/couchcore/threadstore.go`
- Modify `cmd/internal/couchcore/threadstore_test.go`

**Steps:**

- [x] **Step 1:** Write failing `ResolveLaunchProfile` property tests using the strategy
   table; run `go test ./cmd/internal/couchcore -run TestLaunchProfile -count=1`
   and expect FAIL.
- [x] **Step 2:** Implement per-thread/path preferences and independent
   `AgentSource`/`ArgvSource` resolution with the precedence in Architecture.
- [x] **Step 3:** Rerun the focused command; expect PASS, then commit the pure profile slice.

### Task 2: Reuse Pair's shared agent inventory/defaults

**Files:**

- Modify `cmd/internal/launcher/agent_defaults.go`
- Modify `cmd/internal/launcher/agent_defaults_test.go`
- Modify `cmd/internal/launcher/launch_args_policy.go`
- Modify `cmd/internal/launcher/launch_args_policy_test.go`
- Modify `cmd/internal/launcher/args.go`, `rename.go`, `runcli.go`, and tests
- Modify `cmd/internal/couchcore/ops.go`, `operationdispatch.go`, and `couch.go`
- Modify couchcmd runtime composition, argument binding, and tests

**Steps:**

- [x] **Step 1:** Write failing composition tests against Pair's shared agent/default seam.
- [x] **Step 2:** Wire root-agent evidence and resolved argv without a couch agent enum;
   `PAIR_USE_REPO_DEFAULT` is emitted only for its matching provenance.
- [x] **Step 3:** Run launcher/couch focused tests; expect PASS, then commit the wiring slice.

### Task 3: Commit preferences only on successful registration

**Files:**

- Modify `cmd/internal/couchcore/starttransaction.go`
- Modify `cmd/internal/couchcore/starttransaction_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/thread.go`, `threadstore.go`, and tests
- Modify `cmd/internal/threadrecord/record.go` and tests

**Steps:**

- [x] **Step 1:** Drive unsuccessful `AdvanceStartTransaction` states from the model and
   assert none commit thread/path preferences.
- [x] **Step 2:** Commit thread and path preferences in the successful-registration
   transaction only; use the store concurrency invariant from the strategy
   table.
- [x] **Step 3:** Run start/profile focused tests; expect PASS, then commit.

### Task 4: M4 integration boundary

**Files:**

- Modify `README.md`, `atlas/couch.md`, issue/project logs
- Modify cross-reader conformance and CLI value-bearing flag tests
- Modify `cmd/internal/couchcore/runner.go` and production-runner environment tests

**Steps:**

- [x] **Step 1:** Run the end-to-end launch-profile provenance scenario and assert exact
   argv/source persistence across couch restart.
- [x] **Step 2:** Run focused/full suites and `git diff --check`; commit and run
   `sdlc milestone-close --issue 149 --milestone M4`.

## Chunk 5: Milestone M5 — legacy migration and composite artifact proof

### Task 1: Journaled, idempotent legacy migration

**Files:**

- Create `cmd/internal/couchcore/migration.go`
- Create `cmd/internal/couchcore/migration_test.go`
- Modify `cmd/internal/couchcore/store.go`
- Modify `cmd/internal/couchcore/store_test.go`
- Modify `cmd/internal/couchcore/threadstore.go`

**Steps:**

- [x] **Step 1:** Write `MigrateLegacyRecord` model tests from the strategy table,
   beginning with M1 cutover records and treating unreadable legacy input as
   preserved conservative state.
- [x] **Step 2:** Under the global lock, write a schema-versioned nonce journal and advance
   idempotently. Preserve every legacy file and unreadable record. Enrich the
   M1 path-derived composite records without changing their admission evidence;
   retain inseparable co-tenants as multiple incarnations that conservatively
   block admission.
- [x] **Step 3:** Never rename legacy Pair artifact files. Create distinct composite records
   by scope and verify rerunning migration is byte/state stable.

### Task 2: Route all Pair artifacts through the composite namespace

**Files:**

- Create `cmd/internal/artifactpath/paths.go`
- Create `cmd/internal/artifactpath/paths_test.go`
- Create `cmd/internal/artifactpath/manifest.go`
- Create `cmd/internal/artifactpath/coverage_test.go`
- Modify `cmd/internal/adapt/adapt.go`
- Modify `cmd/internal/agentcmd/restart.go`
- Modify `cmd/internal/codexsid/codexsid.go`
- Modify `cmd/internal/clipcmd/clipcmd.go`
- Modify `cmd/internal/clipcmd/run.go`
- Modify `cmd/internal/clipcmd/runcli.go`
- Modify `cmd/internal/clipcmd/runtime.go`
- Modify `cmd/internal/contextcmd/contextcmd.go`
- Modify `cmd/internal/continuationcmd/continuation.go`
- Modify `cmd/internal/continuationcmd/continuationcmd.go`
- Modify `cmd/internal/continuationcmd/draft.go`
- Modify `cmd/internal/draftroute/route.go`
- Modify `cmd/internal/opener/opener.go`
- Modify `cmd/internal/opener/run.go`
- Modify `cmd/internal/opener/runtime.go`
- Modify `cmd/internal/reviewcmd/run.go`
- Modify `cmd/internal/reviewcmd/runtime.go`
- Modify `cmd/internal/scrollbackcmd/scrollbackcmd.go`
- Modify `cmd/internal/sessionwatch/run.go`
- Modify `cmd/internal/sessionwatch/sessionwatch.go`
- Modify `cmd/internal/slugcmd/slugcmd.go`
- Modify `cmd/internal/titlepoller/run.go`
- Modify `cmd/internal/titlepoller/runtime.go`
- Modify `cmd/internal/titlepoller/titlepoller.go`
- Modify `cmd/internal/transcript/transcript.go`
- Modify `cmd/internal/workbenchshortcut/shortcut.go`
- Modify `cmd/internal/wrapcmd/wrap.go`
- Modify `cmd/internal/launcher/scoped_paths.go`
- Modify `cmd/internal/launcher/scoped_paths_test.go`
- Modify `cmd/internal/launcher/agent_defaults.go`
- Modify `cmd/internal/launcher/config.go`
- Modify `cmd/internal/launcher/ledger.go`
- Modify `cmd/internal/launcher/history.go`
- Modify `cmd/internal/launcher/compaction.go`
- Modify `cmd/internal/launcher/createflow.go`
- Modify `cmd/internal/launcher/decision.go`
- Modify `cmd/internal/launcher/lifecycle.go`
- Modify `cmd/internal/launcher/rename.go`
- Modify `cmd/internal/launcher/osruntime.go`
- Modify `cmd/internal/launcher/readiness.go`
- Modify `cmd/internal/launcher/layoutflow.go`
- Modify `cmd/internal/launcher/legacy_live.go`
- Modify `cmd/internal/launcher/migrate.go`
- Modify `cmd/internal/launcher/session_index.go`
- Modify `bin/lib/adapt-log.sh`
- Modify `nvim/adapt.lua`
- Modify `nvim/annotate.lua`
- Modify `nvim/changelog.lua`
- Modify `nvim/doctor.lua`
- Modify `nvim/init.lua`
- Modify `nvim/pair_poke.lua`
- Modify `nvim/review.lua`
- Modify `nvim/scrollback.lua`
- Modify `nvim/slug.lua`
- Modify `nvim/workbench_route.lua`
- Modify `nvim/zellij_trace.lua`
- Modify `zellij/config.kdl`
- Modify `zellij/layouts/main-2.kdl`
- Modify `zellij/layouts/main-3.kdl`
- Modify their colocated tests

**Explicit classification inputs:**

- Classify `doctor/doctor.sh` and `doctor/emitter-health.sh` as exact-path
  consumers or migrate any constructors to Go-exported resolved paths.
- Classify `cmd/internal/runtimebundle/assets/runtime/manifest.json` and every
  file below `cmd/internal/runtimebundle/assets/runtime/files/` as generated
  mirrors; regenerate them only through `make runtimebundle-generate`.
- Classify remaining production matches found by `coverage_test.go` as either an
  `artifactpath` constructor call or a pure consumer of an already-resolved
  path. The checked classification contains exact files and family names; no
  wildcard exemption may suppress a new constructor.

**Steps:**

- [x] **Step 1:** Create dependency-leaf package `artifactpath` (stdlib imports only) with
   the validated composite path value, checked artifact-family manifest, and
   constructors covering draft,
   ledger, log, scrollback, queue, config/native-session identity, pane/agent/
   ready/PID/outer-tty ownership, parked/parked-scrollback, adapt, image,
   continuation, layout, restart, picker, and session-name bindings. Add tests
   that scan every production Go, shell, Lua, and KDL source and fail when a
   tag-bearing constructor or consumer is neither registered nor explicitly
   classified. `launcher` becomes a consumer/wrapper of this leaf so
   `sessionwatch` can import it without an import cycle.
- [x] **Step 2:** Change every Go accessor listed above—including
   `launcher/legacy_live.go`, `osruntime.go`, `readiness.go`, `layoutflow.go`,
   `migrate.go`, and non-launcher command packages—to accept/derive the leaf
   validated paths. Enforce that every result remains below the selected scope
   directory; remove ad hoc `dataDir + tag` joins.
- [x] **Step 3:** Classify every listed shell/Lua/KDL consumer. A consumer that needs an
   artifact receives its exact resolved path from Go/environment rather than
   rebuilding it from tag/data-dir; pure consumers of an exact path remain
   classified reads. Run `make runtimebundle-generate` and commit the generated
   `cmd/internal/runtimebundle/assets/runtime` mirror instead of hand-editing it.
- [x] **Step 4:** Run `artifactpath.Resolve` and manifest-coverage strategies across two
   scopes with the same legacy tag, including representative Go, shell, Neovim,
   and both layout consumers; cross-scope observation/mutation is the oracle.
- [x] **Step 5:** Have standalone Pair upsert its thread record in the same transaction model
   without changing its ordinary tag prompt.

### Task 3: Final reconciliation and closure

**Files:**

- Modify `workshop/issues/000135-*` with an appended revision/log entry
- Modify `workshop/issues/000149-*`
- Modify `workshop/projects/couch.md`
- Modify `atlas/couch.md`, `atlas/session-identity.md`, and `atlas/index.md`

**Steps:**

- [x] **Step 1:** Reconcile #135 terminology and addressing with composite durable work-thread
   identity. Record #152 as owner of verified park/resume/`last_active_at` and
   #153 as owner of provisioning/path rebind; do not implement either here.
- [x] **Step 2:** Run the Verification commands, repository full suite, race suite, real couch
   smoke, and `git diff --check`.
- [x] **Step 3:** Update issue/project/atlas evidence and check every issue plan row.
- [ ] **Step 4:** Commit the final window and run
   `sdlc milestone-close --issue 149 --milestone M5`, fixing all boundary
   findings. Then run `sdlc close --issue 149 --verified '<exact evidence>'`;
   let SDLC measure actual time and archive the issue/plan.

## Verification commands

- Focused pure/stateful functions: commands named in each task, implementing the
  risky-function strategy table through shared fakes.
- Real boundaries: `make test-live` plus `make test-couch-policy-live
  SDLC_BIN=../ariadne/bin/sdlc`.
- Repository: `go test ./... -count=1`, `go test -race ./cmd/internal/couchcore
  -count=1`, `make test`, `zellij --config-dir zellij setup --check`, both
  `zellij setup --dump-layout` commands, and `git diff --check`.
- Milestone smoke commands are committed probes/targets before their evidence is
  cited; no remembered one-off command satisfies a boundary.

## Risks and constraints

- Advisory locking is cross-process coordination, not crash-proof storage by
  itself; atomic writes, strict decoding, nonce journals, and fail-closed
  recovery are all required.
- The supervisor lease and short ThreadStore transaction lock are distinct: the
  lifetime lease owns actor creation/terminal routing, while operation clients
  still serialize record writes. Neither lock may be reused for the other's
  lifetime.
- An explicit alternate `COUCH_STORE_DIR` remains test isolation only. The plan
  does not add namespace discovery, naming, federation, or multi-couch UX.
- M1's helper-less post-fork interval remains conservatively occupied. It is
  not described as fully recoverable until M2 closes.
- The live provider test consumes a built Ariadne binary: deterministic tests
  stay on the fake, while Pair's weekly/manual workflow owns recurring protocol
  conformance after Ariadne #200 merges.
- Legacy global tag files are not moved. Scope selection contains them; M5
  proves the containment before #149 closes.
- Picker hierarchy and presentation belong to #151. M3 supplies shared rows and
  operations only.

## Revisions

### 2026-08-26 — add singleton namespace and root-operation parity

**Reason:** operator review established that durable threads belong to one couch
namespace across process restarts, while more than one supervisor would create
unroutable children. It also required every human operation to be available to
the root agent through the same surface.

**Delta:** M1 now canonicalizes the physical store once and adds a
non-inheritable lifetime supervisor lease, verified by real-process crash tests.
Operation declarations gain execution/effect/confirmation metadata in M1 so
external owner-required calls fail safely. M3 declares switch/attach and removes
implicit human-only effect paths. #147 remains responsible for routing those
owner-required client calls, and #148 for the thin Ariadne-distributed skill;
#149 supplies the shared schema but implements neither downstream subsystem.

### 2026-08-26 — close implementation-gate ordering and test-contract gaps

**Reason:** the stateful SDLC plan gate found that M1 could admit two Brain
starts before M2 gave them distinct tags, and that prose case inventories plus
a one-shot live provider check did not define durable test ownership.

**Delta:** final opaque composite allocation/claim moves into M1 before
admission/fork; M2 now only widens start recovery. A named risky-function table
single-sources adversarial strategies used by compact task steps. Pair owns a
weekly/manual Ariadne provider conformance workflow in addition to M1 boundary
evidence and stateful fake tests (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — incorporate M1 boundary review

**Reason:** the first mandatory boundary review found that Pair-client death
was being mistaken for whole zellij-session quiescence, opaque allocation did
not account for preexisting scoped Pair artifacts, and several public and plan
contracts had drifted from the implementation.

**Delta:** M1 retains occupied incarnations until #152 can prove quiescence and
checks every current scoped artifact family plus detached-session bindings
before claiming an opaque address. Reconciliation now makes exactly three
cohort attempts and returns typed `PolicyUnstableError`. The architecture table
classifies namespace resolution as integration, the README documents only
provider-owned admission, and the live conformance target accepts the canonical
`SDLC_BIN` interface with relative paths (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — make composite allocation one producer authority

**Reason:** the second M1 boundary review proved that scanning artifacts before
claiming ThreadStore left a creation interval in which another Pair producer
could acquire the same address. It also found that the current M1 `Operation`
surface still embeds effectful closures, external live-owner commands lack #147
routing, and project close metadata preceded boundary acceptance.

**Delta:** an O_EXCL scoped thread-claim marker now serializes Couch allocation
and the native Pair create flow before either writes artifacts or session
bindings. Couch reservations are distinct from established Pair claims; every
current constructor derives collision membership from `ScopedPaths`, malformed
session indexes fail closed, and only a failed ThreadStore claim releases its
marker. `Operation` is classified as integration until M3 separates declaration
from execution. README exposes live-owner start/stop through the root console
and names #147 for second-process routing; project `closed` remains absent until
the boundary succeeds (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — close the full artifact and project-state classes

**Reason:** the third M1 boundary review showed that an inventory limited to
`ScopedPaths` omitted active tag-bearing paths produced by Go, Lua, and layouts,
and that removing premature project close metadata lacked a load-bearing
invariant.

**Delta:** collision recognition is now a structural filename-boundary rule
inside Pair's owned scope rather than a hand-maintained family list. It covers
all current non-`ScopedPaths` artifacts and arbitrary future families while
rejecting neighboring tags; the global session index remains a strict separate
representation. A reusable project-artifact contract walks every milestone
detail block and rejects non-empty `closed` metadata whenever its referenced
task row is unchecked. Both corrections have explicit red-without-fix tests
(ARCH-DRY, ARCH-PURPOSE).

### 2026-08-26 — carry the acknowledgement descriptor through the real PTY seam

**Reason:** implementing M2's planned separate pre-exec command made two implied
integration files explicit: the helper must be built and installed beside
`couch`, and the production PTY child must be able to inherit the same
close-on-exec acknowledgement descriptor as the stdio runner. Without the
latter, console starts would test a different safety boundary from
`--no-console` starts.

**Delta:** Task 2 also modifies `Makefile.local` and
`cmd/internal/ptychild/child.go` plus its conformance coverage. The helper is an
internal installed binary, not a new user command. `Runner.StartBlocked` passes
one read descriptor to either real child path; its stateful fake models the
same no-exec/ack/cancel lifecycle and the PTY wrapper preserves
`TerminalHandle` capability (ARCH-DRY, ARCH-MOCK).

### 2026-08-26 — use Pair's established address claim as registration evidence

**Reason:** M2's promotion step needs durable Pair-owned evidence after helper
exec; PID liveness or a successful pipe write proves neither that Pair reached
its own startup path nor that it adopted the intended composite address. The
existing reserved → established thread-address claim is the earliest exact
evidence already written by the production Pair launcher.

**Delta:** Task 3 also widens `ThreadArtifactClaimer`, its stateful fake,
`launcher/thread_claim.go`, `ProcOps`, and ThreadStore's exact-nonce mutation
helpers. Couch records the current supervisor identity through `ProcOps`,
acknowledges only after the helper tuple commits, waits for the matching claim
to become established, then promotes. Startup reconciliation reads the same
oracle and exact process-start identities; unreadable evidence reports an error,
unknown liveness stays occupied, and rollback uses nonce plus revision before
releasing the address claim (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — incorporate M2 boundary review

**Reason:** the mandatory M2 review enumerated three post-ack exits that could
return an error while Pair still wrote the workspace: registration evidence,
durable promotion CAS, and legacy registry persistence. It also found that
PURE start tests inherited temporary-directory IO and that stdio/PTY blocked
starts duplicated the full acknowledgement-pipe protocol.

**Delta:** every post-ack exit before handle handoff now goes through one exact-
incarnation quiescence rule: TERM, bounded escalation to KILL, reap proof, then
durable reconciliation while uncertain evidence remains occupied. One table-
driven integration test covers the complete three-site enumeration, including
concurrent metadata preservation. Direct `ThreadRecord`, `StartTransaction`,
and admission tests are mechanically barred from filesystem/process/fake
seams; runner and store lifecycle checks live in an explicitly integration-
named file. `startBlockedChild` is the sole pipe/helper/descriptor authority
used by both production runners (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — incorporate second M2 boundary review

**Reason:** the next review demonstrated that acknowledgement outcome is not
binary—a byte can be delivered before close reports an error—and that Pair's
held client can leave a durable zellij session with workspace-writing panes.
It also found the reserved → established registration oracle used an in-place
truncate/write window, and that the shared runner authority lacked a load-
bearing regression.

**Delta:** every acknowledgement error is treated as possibly delivered. The
post-attempt failure rule first reaps the exact client, then resolves and force-
deletes the zellij session bound to the exact composite address; only complete
whole-incarnation quiescence permits reconciliation. The integration inventory
now injects all four exits against a real target with a TERM-resistant child
and proves both processes gone. Registration publication uses synced temp +
atomic rename + directory sync with a synchronized concurrent-reader test.
The structural runner contract requires both `StartBlocked` methods to delegate
to the one handshake authority and rejects local pipe/wrapper construction
(ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — make quiescence production-shaped

**Reason:** the third M2 review showed that the real-descendant regression used
a fake artifact hook to kill the orphan itself. Production could delete an
indexed zellij session but did not own arbitrary pre-session descendants, so
the test asserted a stronger boundary than the shipped implementation.

**Delta:** the pre-handoff process inventory is now explicit. The helper/Pair
client leads one actor-owned process group; Couch-launched session-watcher and
title-poller sidecars remain in it; the zellij server and panes are the one
separately detached class and remain controlled by the exact composite-address
session binding. Rollback unconditionally kills and proves the process group
empty before deleting the indexed session. The four-site real descendant table
runs through both stdio and PTY production runners with no cleanup fake, and a
real subprocess test proves Couch sidecars inherit the owned group while direct
Pair sidecars retain `setsid` (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — require observed absence from destructive seams

**Reason:** the fourth M2 review found that exact session selection still ended
in best-effort deletion: `DeleteSession` discarded lingering-server kill errors
and never observed the session/server absent. Its recording doubles proved only
that deletion was attempted, not that quiescence was achieved.

**Delta:** `SessionQuiescence` is now an explicit integration boundary shared by
production and a stateful re-registration fake. It repeatedly observes the
exact session record and exact server PID set, deletes the record, kills only
those servers, and returns success only after two stable absent observations.
Query, deletion, or kill errors fail closed. The stateful regression requires a
server to re-register once, then proves the second delete and exact kill; parser
coverage rejects neighboring names and shell-command false positives
(ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — reuse cleanup wait ownership and make live checks load-bearing

**Reason:** the sixth M2 review showed that an unbounded ownership retry also
needs one reusable reap observer: starting a fresh `Wait` goroutine on every
attempt can strand one waiter per failure. It also showed that final-session
absence alone did not prove the live test exercised server enumeration or
exact-server escalation, because ordinary zellij deletion can reach the same
end state.

**Delta:** post-ack cleanup now constructs one handle-cleanup owner with one
wait-result channel and reuses both across every attempt; a delayed-reap
regression requires multiple KILL retries, one waiter, and eventual absence.
The live real-zellij decorator records and requires server observation,
session-record deletion, and `KillServer` dispatch before accepting the final
absence assertion (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — retain ownership through cleanup failure

**Reason:** the fifth M2 review separated successful cleanup from failed cleanup.
Although success now proved absence, a query/delete/kill/timeout error still
returned from `Spawn`; `Operation.Invoke` then discarded the handle. The review
also found destructive zellij escalation carried only a previously observed
PID and that the new external seam lacked live conformance.

**Delta:** post-ack cleanup has no give-up return. It retains the exact handle
on the start call stack and retries each unproven cleanup class until absence is
proved, so persistent external failure blocks under an active owner rather than
creating an untracked writer. Zellij server observations now carry PID plus
kernel start identity and session; immediately before SIGKILL production checks
identity, exact argv, then identity again. A stateful table covers PID reuse,
exec-away, reuse during reauthorization, and the exact accepted incarnation.
An ephemeral real-zellij PTY test exercises list, process observation, deletion,
and verified absence through production; `make test-live` exposes it locally
and a weekly/manual macOS workflow supplies the committed cadence. The live run
also captured zellij's exit-2-after-success behavior: command errors remain in
timeout diagnostics but only observed absence decides success
(ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — drive live conformance through the lowest kill seam

**Reason:** the seventh M2 review accepted `KillServer` reachability but showed
that its observing decorator set the flag before exact-identity reauthorization
and the underlying `killProcess` call. Because ordinary zellij deletion usually
terminates the real server first, that assertion could remain green with an
ineffective OS-kill boundary.

**Delta:** the live test adds a separately owned test process whose exact argv
matches the target zellij server and records the injected production
`killProcess` dispatch for that PID. Session deletion cannot remove the
sentinel, so successful conformance now requires real process-table discovery,
start-identity reauthorization, and the underlying OS kill operation before
verified absence (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — incorporate M3 boundary review

**Reason:** the first M3 boundary review found five unreachable or parallel
contracts plus stale public documentation: authoritative ThreadIndex failures
fell back to legacy launch, empty required metadata values were treated as
missing, CLI reference consumers lost repository scope, initial terminal
attachment bypassed the operation dispatcher, and the Core concepts table no
longer matched the implemented purity boundary.

**Delta:** absence of a Couch index is now a typed condition distinct from
malformed or incomplete durable state, which fails closed. Every CLI consumer
of a mutable composite reference (`show`, `name`, and `describe`) derives the
current Git-root scope; exact-address callers continue to carry both scope and
tag. Required operation arguments validate map presence so empty metadata
values retain their clearing meaning. Both initial and later attachment use
the declared typed `attach` effect, and the exact pane-registration primitive
is private to the console package. The architecture tables now obey the rule
that each row names a greppable entity with exactly one architectural kind:
metadata transition and operation declarations are PURE, while store CAS and
dispatch executors are INTEGRATION. Task 2's implemented inventory placement
is `threadinventory.go`; its planned `couch.go`/`couch_test.go` changes were not
needed. README now inventories the full M3 command, lookup, rendering, picker,
and empty-value behavior (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — incorporate second M3 boundary review

**Reason:** the second M3 review confirmed BR-18 through BR-23 but found that
the shared compact renderer erased a named thread's immutable tag from `show`,
and that both panel callbacks converted authoritative ThreadStore failures to
empty results.

**Delta:** rendering now has an explicit detail contract: `list` remains
name-first and compact, while `show` always emits the full `{repo scope, tag}`
address for diagnostics and exact follow-up. The durable-read invariant covers
every panel record read, not only standalone Pair: inventory and reference
callbacks return errors, the console preserves the last valid model when one
exists, and the owned screen displays the failure. A named-show CLI regression,
a real corrupt-ThreadStore wiring regression for both callbacks, and visible
inventory/reference error tests pin the class (ARCH-DRY, ARCH-PURPOSE,
ARCH-MOCK).

### 2026-08-26 — incorporate third M3 boundary review

**Reason:** the third M3 review confirmed BR-24/BR-25 but found that Launcher's
partial `threadIndexRecord` schema accepted records Couch rejected, repeating
the durable-read authority family for a third round.

**Delta:** persisted record acceptance is now one lower-layer contract rather
than two structs. `threadrecord.Record` defines the complete top-level,
address, incarnation, start-claim, and policy JSON shape; its pure validators
own schema version, component validation, absolute paths, creation time,
revision, incarnation state/PID/start relationships, single tracked start,
positive claim generation, and path/address equality. `strictjson.Decode` owns
duplicate-key, unknown-field, and trailing-value rejection. ThreadStore writes
and reads this wire type; Launcher projects the same decoded value into
ThreadIndex. A real-file mutation table changes every enumerated invariant and
requires both readers to reject it, preventing future acceptance drift
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — align the complete M4 inventory and flag contract

**Reason:** the first M4 boundary review found the architecture inventory named
only the resolution result even though the milestone added or widened several
durable, pure, and integration entities. It also found the generic CLI binder
treated every named flag without `=` as a boolean and therefore turned bare or
empty explicit agent selection into fallback or synthetic-agent behavior.

**Delta:** the Core concepts contract now states and applies a whole-diff rule:
every milestone-added or milestone-modified architectural entity gets one
greppable row with its actual kind, path, and latest milestone status. The M4
task file inventories likewise name the implemented Couch, Launcher, shared
wire, store, and CLI surfaces. `ArgSpec.ValueRequired` distinguishes named
value flags from switches; `bindArgs` rejects both `--agent` and `--agent=`
before dispatch, with generic-binder and public-CLI regressions proving no
runner operation occurs (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — enforce value requirements at the transport-neutral boundary

**Reason:** the second M4 boundary review confirmed BR-27 and the CLI half of
BR-28, then showed that advisor or console callers could bypass `bindArgs` and
send an empty value directly through the shared operation dispatcher. It also
found generated review prose introduced trailing whitespace after the earlier
working-tree check.

**Delta:** `validateOperationCall` now derives non-empty validation from every
`ArgSpec.ValueRequired` declaration before choosing an executor. A pure generic
dispatch regression proves the live executor is not invoked, complementing the
public CLI regressions. Boundary verification now finishes with
`git diff --check <previous-boundary>..HEAD` after every generated review and
disposition artifact is included, not merely a pre-review working-tree check
(ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — isolate launch policy from inherited child environment

**Reason:** the third M4 boundary review disposed BR-28/BR-29, then showed that
omitting `PAIR_USE_REPO_DEFAULT` for path-derived argv did not express false at
the real process boundary: `ExecRunner` inherits Couch's environment, so stale
parent state could contradict the strict profile.

**Delta:** Couch now supplies exactly one value for the launch-owned key in
both states, and the production runner overlays all supplied child keys after
removing inherited and supplied duplicates. A real subprocess probe verifies a
stale parent `=1` becomes one authoritative empty child entry, while a pure
merge table fixes last-supplied-wins semantics and the restart scenario pins
Couch's negative emission (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — make production environment wiring load-bearing

**Reason:** the fourth M4 review confirmed the child behavior but reverted the
production merge call without reddening the subprocess test. Go's `os/exec`
normalizes duplicate keys before the Go helper observes them, so that test
could not distinguish our sanitizer from the runtime's incidental behavior.

**Delta:** `buildExecCommand` is now the single production command-construction
path. Its regression inspects the raw `exec.Cmd.Env` before runtime
normalization and requires exactly one authoritative policy entry; changing
that production line back to inherited-plus-appended entries makes the test
fail with both `=1` and empty values. The real subprocess probe remains as
end-to-end behavior evidence (ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — reuse the ThreadStore journal for M5 legacy enrichment

**Reason:** Task 1 anticipated changes to the old `Store`, but M1 already
preserved and strictly loaded `registry.json` and atomically cut its actors over
to composite ThreadStore records. A second migration engine would duplicate
the transaction authority and risk rewriting the historical input.

**Delta:** `MigrateLegacyRecord` is the pure metadata enrichment over an M1
cutover record; `ThreadStore.MigrateLegacyRecords` performs the complete
locked migration with the existing recoverable journal and a versioned
manifest marker. New journals carry a content-addressed transaction nonce;
pre-nonce interrupted journals remain readable for compatibility. The old
Store and registry bytes are never targets, so no `store.go` production change
is required. Tests retain same-tag cross-scope separation and co-tenant
occupancy from M1, prove exact metadata enrichment, preserve corrupt bytes and
an incomplete marker on refusal, recover an interrupted multi-target journal,
and require byte-stable reruns (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — close constructor and durable-index compatibility classes

**Reason:** the first M5 boundary review showed that moving
`session-names.jsonl` to the selected scope had no compatibility read for live
sessions indexed only in the former global file, extensionless scripts escaped
the artifact-source guard, two non-Go consumers still derived selected paths,
and the Core concepts table named a nonexistent `ArtifactFamily` in the wrong
file.

**Delta:** durable session-index reads strictly merge the legacy-global and
selected-scope files, treat absence as empty, and fail closed on malformed or
unreadable present state at every destructive/identity-dependent caller. The
artifact source class now includes extensionless shebang programs; intentional
legacy-flat reads derive through `LegacyRootPaths`/`LegacyPaths`, while current
shell and Neovim consumers require their exact exported bindings. The complete
M5 entity sweep records the actual `Address`/`Paths`/`ScopePaths`/`Binding`,
`Family`/`SourceClassification`, standalone-registration, migration, legacy
compatibility, and strict index-decoder entities at their real paths. Regression
tests start with only a global binding plus a detached session, inject corrupt
and unreadable indexes through each effect path, discover an extensionless
constructor, and forbid the two selected-path derivations (ARCH-DRY,
ARCH-PURE, ARCH-PURPOSE).

### 2026-08-26 — complete the M5 authority classes and preserve the #152 boundary

**Reason:** the second M5 review showed the first disposition fixed named
instances rather than the complete classes: claim/quiescence still bypassed
the overlap reader, durable rows lacked structural validation, and other
scrollback/changelog consumers still derived companion suffixes. It also
repeated global-layout prose and proposed releasing standalone capacity without
the whole-session proof explicitly assigned to #152.

**Delta:** every session-binding consumer now calls one strict legacy-global
then selected-scope reader; required session name, scope, repository root/name,
and tag fields validate before any identity or destructive decision. Composite
artifact authority now owns complete scrollback, parked-scrollback, and
changelog companion sets. A repository-wide negative guard scans all production
Go, shell, Lua, and KDL sources—including extensionless shebang programs—and
forbids companion-suffix construction outside `artifactpath`; opener, wrapper,
renderer, lifecycle cleanup, parking, and draft/viewer notification consume
exact paths. Atlas prose describes those bindings and the compatibility-only
legacy rename inventory rather than publishing a second constructor scheme
(ARCH-DRY, ARCH-PURPOSE).

BR-35 is not imported into M5: the approved issue and plan assign verified
whole-incarnation quiescence, park, and capacity release to #152. A Pair client
return, detach, best-effort `DeleteSession`, or external client death is not
proof that the zellij server and panes can no longer write. Standalone records
therefore remain conservatively occupied until #152 supplies that stateful
session proof; freeing them here would violate the fail-closed safety invariant
rather than complete #149 (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-26 — make artifact constructor closure mechanically exhaustive

**Reason:** the third M5 review confirmed the intended constructor authority
but showed its enforcement still scanned `cmd/internal` rather than all
production command packages and allowed any classified source to call itself a
constructor. The same review found two historical atlas passages that still
described derived scrollback siblings and session-keyed ready-marker lookup.

**Delta:** the production-source closure now begins at `cmd`, covering every
top-level command package as well as internal packages, while an independent
classification invariant rejects `Constructor` anywhere outside
`cmd/internal/artifactpath`. Mutation tests inject an artifact constructor into
both `cmd/pair-go` and an internal launcher source and prove that an
unclassified file fails the complete scan while a self-classified constructor
fails the authority check. The atlas now states exact scrollback companion
bindings, exact viewer refresh inputs, session-keyed changelog data, and the
separate stable `$PAIR_CHANGELOG_READY_PATH` binding throughout (ARCH-DRY,
ARCH-PURPOSE).

### 2026-08-26 — keep generated runtime mirrors outside review windows

**Reason:** Step 3's instruction to commit the generated runtime mirror
contradicted the repository's established ignored-build-output boundary and
expanded the M5 review window by roughly 930 KB without adding an authoritative
implementation surface.

**Delta:** the instruction to commit
`cmd/internal/runtimebundle/assets/runtime` is superseded. The mirror remains
ignored and is regenerated only by `make runtimebundle-generate`; deterministic
generation and drift tests prove that it matches the tracked source inputs.
Coverage still classifies every generated path through the generated manifest,
but reviewers inspect the authoritative Go, shell, Lua, and KDL sources rather
than duplicated mirror bytes (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-26 — enforce constructor authority across classified consumers

**Reason:** the fourth M5 review found that a production file already labeled
`ResolvedConsumer` could still construct a selected-scope enumeration glob,
and that atlas prose continued to publish current filename formulas after named
instances were corrected.

**Delta:** every current selected-scope path, enumeration glob, and filename
parser is produced by `artifactpath`; a `ResolvedConsumer` may only receive an
already-resolved value. The mutation matrix covers unclassified sources, false
`Constructor` labels, and false `ResolvedConsumer` labels in both top-level and
internal command packages. The documentation inventory names exact methods or
bindings for current consumers; literal filename shapes are explicitly limited
to compatibility behavior or descriptive storage vocabulary (ARCH-DRY,
ARCH-PURPOSE).

### 2026-08-27 — make constructor and verification rules syntax-independent

**Reason:** the fifth M5 review showed that recognizing selected Go expression
forms was not constructor closure, and reproduced a clean-archive failure where
coverage read ignored generated files left in the developer checkout.

**Delta:** resolved consumers may contain no artifact-family literal except an
exact, centrally allowlisted non-path protocol/CLI value. The mutation matrix
now covers concatenation, formatting, joins, builders, replacements, helper
calls, and `filepath.Join` across top-level and internal command packages.
Generated-mirror coverage always generates into a temporary directory from
tracked sources, then checks exact classification in both directions; it never
reads the ignored working mirror. The tracked-input-only invariant is verified
from a clean archive before the boundary rerun (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-27 — replace negative proof with positive bindings and explicit bootstrap

**Reason:** the sixth M5 review demonstrated that a lexical deny-list cannot
prove constructor closure: a consumer can split a family token across literals.
It also showed that the verification contract named bare `go test ./...` even
though Go's compile-time embed requires the ignored runtime mirror to exist
first. Both gaps were visible at plan time: the plan did not require a positive
derivation witness for each consumer, and it did not state the initial
filesystem state for each verification recipe.

**Delta:** `SourceClassification` gains a positive binding witness. Every Go
`ResolvedConsumer` must name at least one exact exported `artifactpath` selector
and the source AST must actually call or select it; a false classification can
no longer turn arbitrary source into an accepted consumer. Files that mention
only protocol or CLI vocabulary use a separate `VocabularyConsumer` kind and
remain constrained by exact per-file allowlists. The existing negative source
scan remains defense in depth, but no longer carries the architectural proof.
Regression tests first prove that split-token source and a claimed-but-unused
binding fail, while every production resolved consumer proves its declared
dependency (ARCH-DRY, ARCH-PURPOSE).

Runtime generation becomes an explicit prerequisite of the repository test
target and of every documented raw Go verification recipe. The clean-source
generator regression copies/generates from the generator's declared source
inputs and must run without `.git`; it no longer uses Git metadata as an
unstated dependency. A clean archive runs generation before `go test ./...`,
and that exact sequence is the boundary oracle (ARCH-PURPOSE).

**Implementation sequence:**

1. RED: add split-token, missing-binding, and unused-binding mutations; reproduce
   the current constructor guard accepting them.
2. GREEN: add `VocabularyConsumer` and exact Go binding witnesses; validate
   imported `artifactpath` selector use from the AST and migrate every current
   classification to the appropriate kind.
3. RED: reproduce the clean-archive generator test's `.git` dependency and the
   pre-generation embed failure.
4. GREEN: make runtime generation an up-front `make test` prerequisite, remove
   Git discovery from the generator regression, and revise every bare full-Go
   verification recipe to generate first.
5. Verify focused tests, a Git-metadata-free archive sequence, `make test`,
   runtime drift, issue validation, and diff hygiene before retrying M5.

**Plan-review rule:** for every claimed single-authority migration, ask what
positive dependency or derivation witness a new consumer must supply; a
self-assigned label is not evidence. For every verification command, state and
test its starting filesystem/environment state, including generated and ignored
inputs. These are ARCH-PURPOSE checks, not implementation-stage cleanup.

### 2026-08-27 — correlate witnesses per family and make the clean oracle executable

**Reason:** fresh-context review of the preceding revision found three holes
before implementation: one unrelated selector could witness several family
claims, `VocabularyConsumer` did not forbid path IO, and the clean bootstrap
still named categories rather than executable commands.

**Delta:** canonical binding definitions map one artifact family to an exact Go
resolver selector and, where the resolver returns an aggregate, the exact
family-specific result member consumed (for example `ResolveScoped` plus
`Paths.Draft`). Each `ResolvedConsumer` classification names a binding for
**every** family it claims. The AST validator resolves the actual import path
and local alias, requires the resolver as a call expression, follows its local
result identifier, and requires the declared member in a non-blank consuming
expression. A missing-family witness, a binding for the wrong family, a bare
function reference, or a declared selector that is never called fails. This is
positive derivation evidence plus a defense-in-depth negative scan, not a claim
to prove arbitrary deliberately obfuscated Go source (ARCH-DRY,
ARCH-PURPOSE).

`VocabularyConsumer` is valid only with no resolved bindings and an exhaustive
list of `{family, exact value, syntax context, occurrence count}` allowances.
Go contexts are parsed string literals (including struct tags); non-Go contexts
are exact trimmed source lines. Go comments do not count as production
occurrences. Every token-bearing occurrence must match one allowance, every
allowance must be observed exactly, and the classification's families must
equal the allowance families. A vocabulary-only Go file may not import
`artifactpath` or call filesystem path construction/read/write selectors
(`path/filepath` or file-affecting `os` APIs); such a file must instead become a
family-witnessed `ResolvedConsumer` or a constructor. Mixed resolved consumers
may carry exact vocabulary allowances in addition to their per-family
bindings, but vocabulary allowances never witness a family. Mutations cover a
valid vocabulary-only file, an unlisted literal, a vocabulary-labeled
`filepath.Join`/`os.ReadFile`, and mixed resolved-vocabulary source.

**Exact clean bootstrap contract:** start from a source archive containing the
current tracked patch, with no `.git` directory and no
`cmd/internal/runtimebundle/assets/runtime`; all paths enumerated by
`runtimebundlegen` are present. Run:

```sh
go run ./cmd/internal/runtimebundle/generatecmd \
  --repo . --out cmd/internal/runtimebundle/assets/runtime
go test ./... -count=1
```

The clean-source generator regression derives its fixture solely from
`runtimebundlegen`'s declared `explicitAssetPaths` and `assetDirs`, never from
`git ls-files`. `make test` declares `runtimebundle-generate` as its up-front
prerequisite. A dedicated non-recursive clean-bootstrap test creates the same
archive, supplies the repository's resolved base Makefile explicitly (because
Pair's tracked `Makefile` is a sibling symlink), runs `make test` with a child
sentinel that prevents the bootstrap test from recursively dispatching itself,
and proves success from the same absent-mirror state.

The M5 repository verification recipe is superseded by these exact commands:

```sh
go test ./cmd/internal/artifactpath ./cmd/internal/runtimebundlegen -count=1
go run ./cmd/internal/runtimebundle/generatecmd \
  --repo . --out cmd/internal/runtimebundle/assets/runtime
go test ./... -count=1
go test -race ./cmd/internal/couchcore -count=1
make test
make runtimebundle-drift-check
sdlc issue validate --issue 149
git diff --check
```

In the no-`.git` clean archive, the first two full-tree commands are the direct
generator then `go test ./...` sequence shown above; `make test` is separately
proved by the dedicated bootstrap regression using the resolved base Makefile.

### 2026-08-27 — distinguish vocabulary-derived IO from unrelated exact-path IO

**Reason:** implementation enumeration found vocabulary-only commands such as
`scrollbackcmd` that legitimately read/write exact paths supplied by their
caller, while the matched `scrollback-render` text is only a CLI command name.
Likewise, native-session vocabulary files inspect native harness directories
unrelated to Pair's artifact namespace. A file-wide ban on filesystem APIs
would therefore reject correct thin IO shells rather than prove artifact
authority.

**Delta:** `VocabularyConsumer` forbids a vocabulary allowance value from being
used as, or assembled into, a filesystem/path-construction argument. The AST
mutation boundary covers direct token-bearing arguments to `path/filepath` and
file-affecting `os` calls. Unrelated exact-path parameters and native harness
paths remain legal. Any Pair artifact path must still arrive through a
family-correlated resolved binding; an exact CLI/protocol allowance never
witnesses that family. This preserves ARCH-PURE's thin IO shells while the
positive binding manifest carries ARCH-PURPOSE's derivation proof.

### 2026-08-27 — close vocabulary allowances around non-path AST use sites

**Reason:** follow-up plan review showed that checking only direct
token-bearing filesystem arguments still permits an allowed literal to be
stored in a constant/local/helper result and later laundered into a path sink.

**Delta:** every Go vocabulary allowance names a closed AST use site as well as
its exact file, family, value, and count. Permitted contexts are: a named struct
field's tag; an exact argument position of an import-qualified standard-library
callee or named existing logging/trace method; a case value in a named
function; or a comparison operand in a named parser function. Vocabulary
literals are rejected in constants, assignments, general returns, arbitrary
helper calls, and any unlisted context. Where current code returns a diagnostic
prefix as a concatenated literal, it is changed to an exact `fmt.Sprintf`
format argument so the non-path use is closed. The validator builds parent and
enclosing-function context from the AST and requires each occurrence to match
one declared site exactly. A local/const/helper laundering mutation must fail;
unrelated caller-supplied exact-path IO remains legal because it contains no
vocabulary occurrence. Non-Go allowances remain exact trimmed lines.

### 2026-08-27 — pin the public bootstrap entry point

**Reason:** the clean-source package test proves generator independence, but it
does not by itself prove the command contributors actually run starts from the
same absent-mirror state. Pair's tracked top-level Makefile is also a sibling
symlink, so a source archive must declare how that external build layer is
supplied.

**Delta:** `test-clean-bootstrap` creates the archive from `HEAD` plus the
current tracked patch, asserts both `.git` and the generated runtime directory
are absent, reconnects the resolved Ariadne base Makefiles, and invokes
`make -f Makefile.local test`. Its dry-run assertion requires the runtime
generator to be the first command. The bootstrap target remains outside
`test`, preventing recursive full-suite dispatch while making the exact
starting state executable and reviewable (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-27 — make source participation exhaustive and resweep M5 entities

**Reason:** boundary review showed that positive bindings still began after a
token-discovery filter: an unlisted production file with a split family literal
could therefore evade the classification boundary. It also showed that binding
member matching used identifier spelling rather than lexical identity, counted
discarded calls, and that the Core concepts table had not incorporated the
authority types added during review disposition.

**Delta:** every production Go, shell, Lua, KDL, and extensionless shebang file
under the checked roots is now present in exactly one exhaustive inventory:
either `SourceClassifications` or `NonArtifactSources`. A new file fails even
when it contains no recognizable token. Non-artifact Go files are AST-scanned
with constant concatenation evaluation, so a listed file cannot launder
`"dra" + "ft-"` through that class. Resolved witnesses track `*ast.Object`
identity from the actual resolver result (including local propagation), and a
family member counts only when its result is not an expression statement or
blank assignment. Mutations pin unlisted plain and split-token files, shadowed
resolver variables, and discarded/blank member calls. The M5 Core concepts and
integration tables are reswept across the full milestone diff, including cache
paths, vocabulary/binding declarations, exhaustive inventory, and standalone
registration (ARCH-DRY, ARCH-PURPOSE).

The bootstrap patch application now distinguishes an empty clean-HEAD diff
from a real patch before invoking `git apply`; the same committed
`test-clean-bootstrap` command is the clean-HEAD regression (ARCH-PURPOSE,
ARCH-MOCK).

### 2026-08-27 — prove exclusive derivation and the M5 concept inventory

**Reason:** the exhaustive source inventory and one positive binding per family
proved that a file participated, but not that every construction in that file
used the authority. A classified file could keep a legitimate family witness
while adding a second split-literal constructor. The reswept Core concepts
table likewise corrected prose without an executable row contract.

**Delta:** resolved-consumer construction scanning now evaluates Go constant
concatenation expressions, not only individual literals, and applies the
family/vocabulary rule to every resulting constant independently of the
file-level witness. The mutation combines a valid `ResolveScoped`/`Draft`
return with an illicit `filepath.Join(root, "dra" + "ft-" + tag + ".md")`
in the same file and must fail. The issue-149 plan contract enumerates every M5
pure-declaration and integration row as an exact `{entity, kind/path, status}`
record; table deletion or field mutation for each row must fail. Together these
tests distinguish participation from exclusive derivation and make the
whole-diff entity sweep executable (ARCH-DRY, ARCH-PURPOSE).
