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

| Pure entity | Lives in | Status |
|---|---|---|
| `CouchNamespace` | `cmd/internal/couchcore/namespace.go` | new in M1 |
| `PolicyResult` / `PolicyCapacity` | `cmd/internal/couchcore/policyresolver.go` | new in M1 |
| `AdmissionDecision` | `cmd/internal/couchcore/admission.go` | new in M1 |
| `PolicyTable` / repository `Mode` | `cmd/internal/couchcore/policy.go` | deleted in M1 |
| `ThreadAddress` / `ThreadRecord` | `cmd/internal/couchcore/thread.go` | new in M2 |
| `StartTransaction` | `cmd/internal/couchcore/starttransaction.go` | new in M2 |
| `ThreadMetadata` / `ThreadSummary` | `cmd/internal/couchcore/threadmetadata.go`, `threadinventory.go` | new in M3 |
| `Operation` effect/owner declaration | `cmd/internal/couchcore/ops.go` | modified in M1 and M3 |
| `LaunchProfileResolution` | `cmd/internal/couchcore/launchprofile.go` | new in M4 |
| `ArtifactFamily` | `cmd/internal/artifactpath/paths.go` | new in M5 |

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
- **`ArtifactFamily`** — checked inventory of every tag-bearing Pair path; new
  sidecars extend the manifest rather than constructing paths ad hoc.

### Integration points

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| `NamespaceResolver` | `cmd/internal/couchcore/namespace.go` | new in M1 | startup cwd, mkdir, physical-path resolution |
| `SupervisorLease` | `cmd/internal/couchcore/supervisorlease.go`, `supervisorlease_unix.go` | new in M1 | non-inheritable OS advisory lock and owner metadata through existing `ProcOps` identity |
| `PolicyResolver` / `ExecPolicyResolver` | `cmd/internal/couchcore/policyresolver.go`, `policyresolver_exec.go` | new in M1 | `sdlc fleet policy` subprocess |
| `ThreadStore` | `cmd/internal/couchcore/threadstore.go` | new in M1, widened later | filesystem lock, per-thread records, WAL/manifest |
| `LaunchHelper` | `cmd/internal/couchcore/launchhelper.go`, `cmd/pair-launch-helper/main.go` | new in M2 | fork/exec acknowledgement boundary |
| `OperationExecutors` | `cmd/internal/couchcore/operationdispatch.go` | new in M3 | direct-store effects and optional live-owner effects |
| `AgentInventory` / repo defaults | `cmd/internal/launcher/agent_defaults.go` | modified in M4 | supported harnesses and scoped argv defaults |

Every integration has a stateful fake or real-process conformance test. In
particular, namespace/lease tests use independent processes and inherited file
descriptors; policy tests use a changing stateful resolver plus the live
Ariadne binary; store tests use independent store instances over one directory.

### One authority, introduced incrementally

`couchcore.ThreadStore` is the only mutable authority for thread records,
incarnations, admission evidence, start transactions, launch preferences, and
migration state. It owns one global Pair-data lock and performs every mutation
as read/validate/change/revision/write while holding that lock. Callers receive
defensive values and use expected revisions for compare-and-swap updates; they
never retain a mutable registry snapshot and later overwrite the store.

M1 needs durable `creating` reservations to make policy admission safe across
processes. Therefore M1 introduces the non-throwaway ThreadStore transaction
kernel and the admission/incarnation subset of its schema. M2 widens the same
schema and API to full thread identity, tag claims, journaled starts, and the
pre-exec helper. This is a dependency correction to the issue's shorthand
milestone rows, not a second store or a change in final ownership
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
unchanged, and either apply a pure proven-dead prune/group/claim mutation or
retry the read/resolve phase. Provider subprocess IO never runs under the
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

- [ ] **Step 1:** Write failing pure tests for default, absolute, relative, missing-directory,
   `..`, symlink, and invalid store inputs. Assert all aliases resolve to one
   absolute physical `CouchNamespace` before any Couch/store construction and
   that the exact path is inherited by a child launched from another cwd.
- [ ] **Step 2:** Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run
   'Test(CouchNamespace|StoreNamespace)' -count=1`; expect failures because the
   resolver and composition seam do not exist.
- [ ] **Step 3:** Implement `ResolveCouchNamespace(startupCWD, configured, defaultPath)` with
   mkdir plus physical resolution. Change `OSRuntime` to resolve it once and
   pass the value into lease, ThreadStore, and child env;
   remove every later interpretation of a raw relative `COUCH_STORE_DIR`.
- [ ] **Step 4:** Add `Operation.Execution` as a closed enum (`direct-store-safe` or
   `owner-required`) plus effect/confirmation metadata. Classify every existing
   operation explicitly; source-level tests fail on an omitted/zero class.
- [ ] **Step 5:** Write failing two-process lease tests: console vs console, console vs
   `--no-console`, and reversed order. A refusal must report the atomically
   published PID/process-start identity only after verifying it through the
   existing `ProcOps` seam. Cover stale/malformed metadata, PID reuse, and
   identity-probe unknown/error; unverified owner metadata is never displayed.
- [ ] **Step 6:** Implement `SupervisorLease` with an injected fake and Unix production lock.
   Acquire before actor-owning Couch construction, set close-on-exec before any
   fork, atomically publish metadata while holding the lock, and hold the handle
   until all supervised work and console teardown complete. Owner-required
   external calls refuse while another owner exists; direct-store-safe calls do
   not take the lifetime lease. Reuse `ProcOps` for PID/start identity rather
   than creating a second process-identity implementation; publication failure
   releases the lease and cannot leave apparent ownership.
- [ ] **Step 7:** Add a subprocess test whose supervisor forks a long-lived child, then is
   killed with SIGKILL. Assert another process immediately acquires the lease
   while the child remains alive; this proves the descriptor was not inherited.
- [ ] **Step 8:** While the lifetime lease is held, perform repeated direct-store-safe
   metadata writes through the short ThreadStore lock. Prove releasing each
   store lock cannot release the supervisor lease and another supervisor still
   refuses; then release the lease and prove acquisition succeeds.
- [ ] **Step 9:** Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -run
   'Test(CouchNamespace|StoreNamespace|SupervisorLease)' -count=1`; expect PASS.
- [ ] **Step 10:** Commit the namespace/lease slice with `git add` on only the files above and
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

- [ ] **Step 1:** Write table tests for bounded/unbounded success envelopes, diagnostic
   envelopes, malformed JSON, missing discriminators, invalid limit/action,
   version/digest absence, unexpected exit status, stderr capture, and timeout.
- [ ] **Step 2:** Define defensive `PolicyResult`, `PolicyCapacity`, `CapacityAction`,
   `PolicyDiagnostic`, and `PolicyResolver.Resolve(ctx, path)` values. Keep
   admission-key kind absent from the consumer model.
- [ ] **Step 3:** Implement a stateful fake keyed by canonical path, recording calls and
   allowing results/errors to change between resolutions.
- [ ] **Step 4:** Implement the deadline-bound production command seam and strict JSON
   decoding. Accept Ariadne's typed diagnostic on its documented nonzero exit;
   refuse malformed, noisy, or protocol-incompatible responses.
- [ ] **Step 5:** Inject the resolver through couchcmd runtime construction; do not discover
   or parse `.sdlc/fleet.json` in Pair.

### Task 3: Introduce the ThreadStore transaction kernel

**Files:**

- Create `cmd/internal/couchcore/threadstore.go`
- Create `cmd/internal/couchcore/threadstore_test.go`
- Create `cmd/internal/couchcore/storelock.go`
- Create `cmd/internal/couchcore/storelock_unix.go`
- Create `cmd/internal/couchcore/storelock_test.go`
- Create `cmd/internal/couchcore/storejournal.go`
- Create `cmd/internal/couchcore/storejournal_test.go`
- Modify `cmd/internal/couchcore/store.go`
- Modify `cmd/internal/couchcore/store_test.go`

**Steps:**

- [ ] **Step 1:** Write failing two-store/process-boundary tests proving serialized updates,
   monotonic revisions, stale expected-revision refusal, reader immutability,
   atomic failure behavior, and lock acquisition/release error propagation.
- [ ] **Step 2:** Define the versioned minimal M1 per-thread record: the existing
   path-derived tag plus derived repo scope, reservation nonce/status
   (`creating`, `live`, `unknown`), canonical path, process identity when
   known, normalized policy evidence, and revision. Store it at
   `threads/<scope>/<tag>.json`; M2 changes new-start allocation to opaque tags
   without replacing this format.
- [ ] **Step 3:** Implement `UpdateExistingThread`: acquire the short global lock,
   strictly decode the addressed record, apply a pure mutation, increment its
   revision, and atomically replace only that record. Add crash-point tests
   proving readers see old or new complete content and manifest generation does
   not change.
- [ ] **Step 4:** Implement journaled membership/multi-record transactions for
   create, delete, cutover, and later preference/migration writes. Journal all
   expected-before values and after-images, replace idempotently, then advance
   manifest membership/generation and clear. Recovery accepts only
   expected-before or exact after-image; crash-point tests cover every boundary
   and third-state/corrupt input fails closed.
- [ ] **Step 5:** Route all M1 admission/reservation mutations through these primitives. The
   existing `registry.json` becomes read-only after the admission bootstrap in
   the next step; no new admission code calls legacy `Store.Load` + `Store.Save`
   and no legacy writer can participate in admission.
- [ ] **Step 6:** Before admitting through the new store, perform an idempotent cutover under
   the global lock: strictly decode every legacy actor, derive its minimal
   composite record, preserve same-tree co-tenants as distinct conservative
   incarnations, and persist a manifest cutover marker in the same journaled
   transaction. Corrupt/ambiguous input refuses cutover and admission. Test
   interruption after every after-image and prove rerun convergence. M5 later
   enriches metadata/artifacts; it does not import admission occupants.

### Task 4: Make admission pure and conservative

**Files:**

- Create `cmd/internal/couchcore/admission.go`
- Create `cmd/internal/couchcore/admission_test.go`
- Modify `cmd/internal/couchcore/procops.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/guard_live_test.go`

**Steps:**

- [ ] **Step 1:** Write a decision matrix for zero/below-limit/at-limit bounded occupancy,
   unbounded capacity, `reject`, `provision-worktree`, live, unknown, creating,
   and proven-dead records.
- [ ] **Step 2:** Write stale-evidence tests for the optimistic protocol: capture the manifest
   and all possibly relevant occupant revisions under lock; resolve the
   candidate plus stale same-repository incumbents outside the lock; retry when
   the manifest or any captured record changes before commit. Digest/version
   match avoids incumbent refresh; mismatch re-resolves each same-repository
   incumbent path; moved keys are counted under the refreshed result;
   unresolved/invalid incumbents remain occupied; creating records are never
   silently rekeyed.
- [ ] **Step 3:** With the stateful resolver, change the provider digest between candidate and
   incumbent calls. Require a full-cohort retry whose successful results share
   one `{version,digest}`; after three mixed epochs, return `policy-unstable`,
   retain occupancy, and prove no fork.
- [ ] **Step 4:** Implement pure occupancy grouping and `Admission.Decide`, returning typed
   `CapacityRefusal` values suitable for both CLI and future TUI clients.
- [ ] **Step 5:** Change `Couch.Spawn` to run read-under-lock/revision capture → unlocked
   provider/liveness IO → relock/revision validation → pure
   prune/group/reserve. Retry on concurrent change. Continue to use
   three-valued `ProcOps` liveness and prune only proven-dead records.
- [ ] **Step 6:** Prove with two independently constructed Couch instances that simultaneous
   starts at a limit of one produce exactly one reservation/fork. Prove every
   diagnostic/refusal/failure path forks zero children.
- [ ] **Step 7:** In M1, release a pristine reservation on pre-fork failure. Until M2's helper
   handshake exists, any uncertain post-fork failure remains an occupied
   recovery record; never claim that such a record is free.

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

- [ ] **Step 1:** Add a source-level shadow sweep that fails on production `PolicyTable`, old
   `Mode` constants, `policy.json`, `Couch.Policy`, and admission use of
   `SameTree`/`--same-tree`.
- [ ] **Step 2:** Remove the public `--same-tree` argument and all admission bypass behavior.
   Preserve only the legacy serialized field needed for M5 replay, clearly
   quarantined from new decisions.
- [ ] **Step 3:** Replace `TreeOccupiedError.Mode` and policy-specific prose with the typed
   capacity/action refusal. `provision-worktree` names #153 and performs no
   path mutation.
- [ ] **Step 4:** Remove `Store.policyPath`, PolicyTable loading, repository-name defaults,
   and tests that bless local policy inference.

### Task 6: Cross-repo conformance and M1 boundary

**Files:**

- Modify `cmd/internal/couchcore/conformance_live_test.go`
- Modify `Makefile`
- Modify `atlas/couch.md`
- Modify `atlas/index.md` only if a new atlas page is added
- Modify `workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md`
- Modify `workshop/projects/couch.md`

**Steps:**

- [ ] **Step 1:** Add an opt-in live conformance target that accepts an Ariadne `sdlc` binary
   path, creates temporary declared repositories, and proves local-tool,
   brain-unbounded, kbench declared-root, and worktree-capacity results pass
   through the production resolver unchanged. Standard unit tests continue to
   use the stateful fake.
- [ ] **Step 2:** Run focused tests, `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd
   -count=1`, the live provider target against the locally built Ariadne #200
   binary, `go test ./... -count=1`, repository shell/Lua tests, layout checks,
   and `git diff --check`.
- [ ] **Step 3:** Run the real namespace matrix from distinct cwd/symlink spellings plus the
   SIGKILL-with-live-child lease test; assert one owner and exact inherited
   canonical `COUCH_STORE_DIR`.
- [ ] **Step 4:** Update atlas ownership: Ariadne declares/measures/resolves policy; Pair
   validates normalized evidence, owns the singleton namespace, and performs
   runtime admission.
- [ ] **Step 5:** Commit the verified M1 window and run
   `sdlc milestone-close --issue 149 --milestone M1`. Fix all Critical/Important
   findings. Only after the Approved boundary may Ariadne #200 close/merge.

## Chunk 2: Milestone M2 — durable identity and recoverable start

### Task 1: Complete the ThreadStore schema and composite address

**Files:**

- Create `cmd/internal/couchcore/thread.go`
- Create `cmd/internal/couchcore/thread_test.go`
- Modify `cmd/internal/couchcore/threadstore.go`
- Modify `cmd/internal/couchcore/threadstore_test.go`
- Modify `cmd/internal/launcher/scope.go`
- Modify `cmd/internal/launcher/scoped_paths.go`
- Modify their colocated tests

**Steps:**

- [ ] **Step 1:** Define `ThreadAddress{RepoScope, Tag}`, immutable `StartingPath`, current
   `WorkingPath`, creation time, metadata placeholders, incarnations,
   transactions, and per-record revision. Validate tags and scope keys at the
   boundary and defensively copy all slices/maps.
- [ ] **Step 2:** Make canonical physical path resolution preserve the exact requested nested
   directory while deriving its repository scope separately. Refuse missing
   paths at start; enumeration remains able to report a later-missing path.
- [ ] **Step 3:** Prove equal tags in different scopes coexist and no API performs tag-only
   lookup across scopes.

### Task 2: Allocate and claim opaque tags atomically

**Files:**

- Replace `cmd/internal/couchcore/actorid.go` with thread-tag allocation
- Modify `cmd/internal/couchcore/clock_test.go`
- Create `cmd/internal/couchcore/tagclaim_test.go`
- Modify `cmd/internal/launcher/tag.go`
- Modify `cmd/internal/launcher/tag_test.go`

**Steps:**

- [ ] **Step 1:** Test the exact `couch-[0-9a-f]{16}` shape, entropy errors, record collisions,
   scoped-artifact collisions, success after retries, and distinct exhaustion
   after eight attempts. Remove the all-zero entropy fallback.
- [ ] **Step 2:** Inject a byte-source that can fail/script collisions. Under the store lock,
   no-replace claim the composite record only after both record and
   `ScopedPaths` artifact checks pass.
- [ ] **Step 3:** Keep standalone human Pair tags accepted; generated-tag rules apply only to
   couch allocation.

### Task 3: Add the blocked pre-exec helper

**Files:**

- Create `cmd/internal/couchcore/launchhelper.go`
- Create `cmd/internal/couchcore/launchhelper_test.go`
- Create `cmd/pair-launch-helper/main.go`
- Modify `cmd/internal/couchcore/runner.go`
- Modify `cmd/internal/couchcore/runner_fake.go`
- Modify `cmd/internal/couchcore/ptyrunner.go`
- Modify their colocated tests

**Steps:**

- [ ] **Step 1:** Write subprocess tests proving no acknowledgement, parent EOF, malformed
   nonce, and deadline expiry exit before the target command; acknowledgement
   execs once with exact argv/env/cwd; descriptors are close-on-exec.
- [ ] **Step 2:** Extend the Runner seam to return the blocked helper's PID/process identity
   and an acknowledge/cancel capability without exposing OS pipes to the domain.
- [ ] **Step 3:** Have production runners fork only the helper. The parent records its identity
   under the transaction nonce before acknowledging. FakeRunner models each
   boundary and supports injected parent death/registration failures.

### Task 4: Implement and reconcile the start state machine

**Files:**

- Create `cmd/internal/couchcore/starttransaction.go`
- Create `cmd/internal/couchcore/starttransaction_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`
- Modify `cmd/internal/couchcore/guard_live_test.go`
- Modify `cmd/internal/couchcore/store.go`

**Steps:**

- [ ] **Step 1:** Table-test interruption before claim, after claim, after helper fork, after
   helper record, after acknowledgement, and after readiness/registration.
- [ ] **Step 2:** Persist one nonce through reservation, helper identity, and live promotion.
   Roll back only the exact matching pristine transaction.
- [ ] **Step 3:** If post-fork cleanup cannot verify death, retain occupied recovery evidence.
   Reconciliation classifies owner/helper identities idempotently, releases only
   failed pre-exec transactions with no matching live helper, and never forks a
   duplicate child.
- [ ] **Step 4:** Pass `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG`, store location, and nonce to
   Pair; remove `COUCH_TREE` as identity (it may remain temporarily as a
   compatibility path hint until M5).
- [ ] **Step 5:** Prove restart reloads the same address and a second call to `couch start`
   always claims a new thread where policy permits.

### Task 5: M2 integration boundary

**Files:**

- Modify `atlas/couch.md`
- Modify the issue/project logs

**Steps:**

- [ ] **Step 1:** Add a real-process integration test that kills the parent at each exposed
   helper boundary and asserts no untracked target process and no falsely free
   slot.
- [ ] **Step 2:** Run focused, full Go, shell/Lua, zellij layout, race-enabled ThreadStore
   tests, and `git diff --check`.
- [ ] **Step 3:** Commit and run `sdlc milestone-close --issue 149 --milestone M2`; resolve all
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

- [ ] **Step 1:** Implement CAS-backed rename, clear-name, describe, and publish-summary
   mutations over a selected composite address. Preserve independent fields and
   refuse stale revisions without lost updates.
- [ ] **Step 2:** Change advisor publish context to `COUCH_THREAD_SCOPE` and
   `COUCH_THREAD_TAG`; do not infer identity from path or zellij name.
- [ ] **Step 3:** Keep duplicate human names legal and test exact tag precedence plus
   ambiguous name/path refusal with returned candidates.

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

- [ ] **Step 1:** Produce one row per thread, even at a shared path, with exact live,
   non-live, unknown, or creating state from durable/process evidence.
- [ ] **Step 2:** Render named rows name-first and unnamed rows tag-first; common `couch list`
   output never leads a named row with the system tag. Keep tag in structured
   output/show/diagnostics.
- [ ] **Step 3:** Make the console/panel consume the same inventory values and operation table
   as the CLI, without implementing #151's final hierarchical UI.

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

- [ ] **Step 1:** Replace `Operation.Invoke(*Couch)` with declaration-only rows plus
   generic dispatch over injected `OperationExecutors{DirectStore,
   LiveOwner}`. The direct executor owns declared safe ThreadStore effects; the
   optional owner executor owns console-local effects. Missing owner execution
   returns typed `OwnerUnavailable{Reason: "owner routing requires #147"}` and
   never falls back or forks.
- [ ] **Step 2:** Write a conformance test that enumerates CLI dispatch, explicit panel
   actions, implicit row actions, and a generic advisor-client fixture. Require
   every effectful behavior to resolve the same declared operation and typed
   arguments; rendering, filtering, cursor movement, and submenu navigation are
   the only exempt pure UI behaviors.
- [ ] **Step 3:** Add `switch` and `attach` rows with typed
   `ThreadTargetArgs{Scope,Tag,ExpectedRevision}` and
   `ThreadTargetResult{Address,Revision,Action}`. `switch` requires a live
   incarnation already hosted in the console and selects its terminal; `attach`
   requires a live owner-held terminal not yet selected and binds/selects it.
   Missing, non-live, unknown, stale-revision, or non-owner targets return typed
   refusals; neither operation creates, revives, or forks a thread.
- [ ] **Step 4:** Implement #149's console-local `LiveOwner` handlers and replace
   Enter-on-live-row's direct `forceSwitch` call and every other
   effectful console shortcut with generic operation dispatch. Keep terminal
   mechanics behind the owner implementation. #147 later transports to this
   handler; this task creates no endpoint/client, and its schema-consumer
   fixture is test-only rather than a #148 advisor implementation.
- [ ] **Step 5:** Prove the generic schema-consumer fixture can invoke name, rename,
   describe, start, switch,
   and attach with the same schema, while an external owner-required call gets
   a typed “owner routing requires #147” refusal rather than executing locally.
- [ ] **Step 6:** Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd
   ./cmd/internal/couchtty -run 'Test.*Operation' -count=1`; expect PASS.
- [ ] **Step 7:** Commit with `git commit -m '#149 M3: unify human and advisor operations'`
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

- [ ] **Step 1:** Add a read/write seam for the same ThreadStore records usable when couch is
   absent. Derive repo scope from Pair's canonical repository and restrict all
   resolution to it.
- [ ] **Step 2:** Show mutable human names in the picker, falling back to opaque tag. Refuse
   duplicate/fuzzy ambiguity with candidates; never mutate
   `SessionNameEntry.SessionName` to hold the human thread name.
- [ ] **Step 3:** Preserve direct `pair claude`/`pair codex` tag prompting exactly when no
   couch composite context is present.

### Task 5: M3 integration boundary

**Files:**

- Modify `atlas/couch.md`, `atlas/session-identity.md`, issue/project logs

**Steps:**

- [ ] **Step 1:** Integration-test two threads in one allowed path with separate records,
   mutable names/descriptions, couch-offline picker resolution, and no file
   moves after rename.
- [ ] **Step 2:** Run focused/full suites and `git diff --check`; commit and run
   `sdlc milestone-close --issue 149 --milestone M3`.

## Chunk 4: Milestone M4 — remembered agent and argument profiles

### Task 1: Model preferences and resolution as pure data

**Files:**

- Create `cmd/internal/couchcore/launchprofile.go`
- Create `cmd/internal/couchcore/launchprofile_test.go`
- Modify `cmd/internal/couchcore/threadstore.go`
- Modify `cmd/internal/couchcore/threadstore_test.go`

**Steps:**

- [ ] **Step 1:** Add per-thread latest-successful `{agent, argv}` and path preference keyed by
   normalized repo identity plus canonical physical path, with `last_agent` and
   `argv_by_agent`.
- [ ] **Step 2:** Test the complete resolution matrix: explicit agent > path last agent > root
   agent; selected-agent path argv > selected-agent Pair repo default; switching
   agents never leaks another agent's argv; returning restores its own argv.
- [ ] **Step 3:** Return two independent provenance values so #151 need not reverse-infer
   them: `AgentSource{explicit,path,root}` and
   `ArgvSource{path,repo-default}`. Test their Cartesian combinations,
   including root-agent plus path argv and explicit-agent plus repo-default
   argv.

### Task 2: Reuse Pair's shared agent inventory/defaults

**Files:**

- Modify `cmd/internal/launcher/agent_defaults.go`
- Modify `cmd/internal/launcher/agent_defaults_test.go`
- Modify `cmd/internal/launcher/launch_args_policy.go`
- Modify `cmd/internal/launcher/launch_args_policy_test.go`
- Modify couchcmd runtime composition/tests

**Steps:**

- [ ] **Step 1:** Extract or expose a narrow agent-inventory and repo-default seam already
   backed by Pair's scoped defaults; do not add a couch-specific agent enum.
- [ ] **Step 2:** Supply the root actor's actual agent explicitly to Couch. Treat missing root
   evidence as a typed resolution failure rather than guessing a harness.
- [ ] **Step 3:** Pass the resolved exact argv to the child while retaining the existing
   one-shot `PAIR_USE_REPO_DEFAULT` behavior only when `ArgvSource` is
   `repo-default`.

### Task 3: Commit preferences only on successful registration

**Files:**

- Modify `cmd/internal/couchcore/starttransaction.go`
- Modify `cmd/internal/couchcore/starttransaction_test.go`
- Modify `cmd/internal/couchcore/couch.go`
- Modify `cmd/internal/couchcore/couch_test.go`

**Steps:**

- [ ] **Step 1:** Test selection cancellation, policy refusal, tag failure, fork failure,
   helper timeout, failed registration, and unknown cleanup: none updates
   preferences.
- [ ] **Step 2:** Atomically promote the incarnation and update both its exact profile and the
   path preference only after successful registration/readiness evidence.
- [ ] **Step 3:** Prove two concurrent successful registrations at different paths merge
   preferences without stale-snapshot loss.

### Task 4: M4 integration boundary

**Files:**

- Modify `atlas/couch.md`, issue/project logs

**Steps:**

- [ ] **Step 1:** Integration-test Claude→Codex→Claude at one path, fallback at a fresh path,
   and couch restart persistence, asserting exact argv and recorded sources.
- [ ] **Step 2:** Run focused/full suites and `git diff --check`; commit and run
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

- [ ] **Step 1:** Begin from M1's cut-over minimal admission records, then fixture legacy
   names/descriptions, repeated tags in different repo scopes, same-tree
   co-tenants, corrupt metadata, and failures after every enrichment step.
- [ ] **Step 2:** Under the global lock, write a schema-versioned nonce journal and advance
   idempotently. Preserve every legacy file and unreadable record. Enrich the
   M1 path-derived composite records without changing their admission evidence;
   retain inseparable co-tenants as multiple incarnations that conservatively
   block admission.
- [ ] **Step 3:** Never rename legacy Pair artifact files. Create distinct composite records
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

- [ ] **Step 1:** Create dependency-leaf package `artifactpath` (stdlib imports only) with
   the validated composite path value, checked artifact-family manifest, and
   constructors covering draft,
   ledger, log, scrollback, queue, config/native-session identity, pane/agent/
   ready/PID/outer-tty ownership, parked/parked-scrollback, adapt, image,
   continuation, layout, restart, picker, and session-name bindings. Add tests
   that scan every production Go, shell, Lua, and KDL source and fail when a
   tag-bearing constructor or consumer is neither registered nor explicitly
   classified. `launcher` becomes a consumer/wrapper of this leaf so
   `sessionwatch` can import it without an import cycle.
- [ ] **Step 2:** Change every Go accessor listed above—including
   `launcher/legacy_live.go`, `osruntime.go`, `readiness.go`, `layoutflow.go`,
   `migrate.go`, and non-launcher command packages—to accept/derive the leaf
   validated paths. Enforce that every result remains below the selected scope
   directory; remove ad hoc `dataDir + tag` joins.
- [ ] **Step 3:** Classify every listed shell/Lua/KDL consumer. A consumer that needs an
   artifact receives its exact resolved path from Go/environment rather than
   rebuilding it from tag/data-dir; pure consumers of an exact path remain
   classified reads. Run `make runtimebundle-generate` and commit the generated
   `cmd/internal/runtimebundle/assets/runtime` mirror instead of hand-editing it.
- [ ] **Step 4:** Test two repositories with the same legacy tag across every manifest family;
   reads/writes/rename/picker/session-name/native-session lookup in one scope
   never observes or mutates the other. Run the same matrix through at least one
   Go command, one shell helper, one Neovim consumer, and both layout modes.
- [ ] **Step 5:** Have standalone Pair upsert its thread record in the same transaction model
   without changing its ordinary tag prompt.

### Task 3: Final reconciliation and closure

**Files:**

- Modify `workshop/issues/000135-*` with an appended revision/log entry
- Modify `workshop/issues/000149-*`
- Modify `workshop/projects/couch.md`
- Modify `atlas/couch.md`, `atlas/session-identity.md`, and `atlas/index.md`

**Steps:**

- [ ] **Step 1:** Reconcile #135 terminology and addressing with composite durable work-thread
   identity. Record #152 as owner of verified park/resume/`last_active_at` and
   #153 as owner of provisioning/path rebind; do not implement either here.
- [ ] **Step 2:** Run migration interruption/conformance tests, all Go tests, shell/Lua tests,
   zellij layout checks, race tests for the store/start transaction, a real
   couch smoke for multi-thread Brain plus local-tool refusal, and
   `git diff --check`.
- [ ] **Step 3:** Update issue/project/atlas evidence and check every issue plan row.
- [ ] **Step 4:** Commit the final window and run
   `sdlc milestone-close --issue 149 --milestone M5`, fixing all boundary
   findings. Then run `sdlc close --issue 149 --verified '<exact evidence>'`;
   let SDLC measure actual time and archive the issue/plan.

## Verification matrix

- **Pure/unit:** strict provider decoding, admission matrix, profile resolution,
  composite validation, tag generation, lookup ambiguity, migration steps.
- **Stateful fake:** changing provider results, three-valued liveness, runner
  acknowledgement/failure boundaries, concurrent ThreadStore clients.
- **Real process:** canonical namespace aliases, non-inherited supervisor lease,
  SIGKILL with a surviving child, advisory store locking, helper
  EOF/timeout/no-exec, parent-death recovery, and production `sdlc fleet policy`
  protocol.
- **End to end:** local-tool rejection, Brain multiple in-place threads, kbench
  per-competition keys, worktree typed action, duplicate path threads with
  separate artifacts, restart persistence, agent-profile memory.

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
- The live provider test is opt-in and consumes the locally built Ariadne #200
  binary until that issue merges; ordinary Pair CI remains hermetic.
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
