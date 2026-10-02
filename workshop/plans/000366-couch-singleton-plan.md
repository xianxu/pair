# Couch local singleton implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use bounded subagents for the new ownership package and fixture audit; keep composition wiring in the main session. Steps use checkbox syntax.

**Goal:** One production supervisor and one stable inventory per local OS account, with lossless in-place adoption and explicit isolated runtimes.

**Architecture:** A small `couchsingleton` package owns resolved roots, adoption evidence and host/store lease composition. The CLI resolves it once, then feeds the resulting roots into every Couch consumer and child environment. Existing store identities, transaction locks and messaging lifecycle remain authorities for their existing concerns.

**Tech Stack:** Go, existing Unix flock lease, strict JSON/durablefile, temporary-filesystem and subprocess tests.

## Contract and approval

The operator's “go ahead” approves the reviewed issue design and recommended refusal-only multi-store scope, including its final spec-review revisions. No inventory consolidation or second-terminal attachment. Adopt one unambiguous store in place; ambiguous installations remain explicitly UNMIGRATED until the operator disposes other stores. The account home and state must be machine-local; old Couch binaries must be stopped/upgraded before cutover.

## Core concepts

| Name | Lives in | Status |
|---|---|---|
| Roots, Selection | cmd/internal/couchsingleton/model.go | new |
| Candidate, Report, Request | cmd/internal/couchsingleton/model.go | new |
| DecideAdoption | cmd/internal/couchsingleton/model.go | new |

Roots names the Couch store, Pair data directory and allocation-authority directory. Selection is a versioned immutable configuration, including explicitly excluded legacy paths. One account selects one Roots; an isolated root selects its own. Candidate records observed legacy state (empty, populated, unavailable, live/unknown owner), never a boolean absence inference. DecideAdoption maps observations plus explicit exclusions to a selection or refusal, without IO (ARCH-PURE, ARCH-ORDER). Future changes widen this policy here rather than adding CLI predicates.

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Manager | cmd/internal/couchsingleton/manager.go | new | selection IO + host/store leases |
| Inspect | cmd/internal/couchsingleton/inspect.go | new | bounded registry/store observation |
| IdentityStore.Inspect | cmd/internal/couchidentity/inspect.go | new | existing identity schema/read/validation |
| OSRuntime | cmd/internal/couchcmd/run.go, singleton.go | modified | account lookup, resolved runtime, child environment |
| adoption CLI | cmd/internal/couchcmd/singleton_cli.go | new | Manager preview/apply |
| isolated smoke wrapper | tests/with-isolated-pair.sh | modified | explicit isolated production composition |

Manager receives explicit authority directory, default Roots and ProcOps; tests inject a temporary authority and use real files and locks. OS account lookup is an injected function at the composition boundary; test subprocesses use a helper with injected account home, never an environment override of production authority. Reuse couchcore.ResolveCouchNamespace/ExistingCouchNamespace, AcquireSupervisorLease, durablefile and strictjson. Do not copy identity schemas into the singleton package (ARCH-DRY).

## Operating and trust boundary

- Production authority: real UID account home from os/user.LookupId, under `.local/share/pair-host/singleton`; HOME/XDG/COUCH_STORE_DIR do not move it. Local filesystems only; no distributed home support.
- `COUCH_ISOLATED_ROOT` explicitly creates a separate test/diagnostic runtime. It must be an absolute canonical directory. Defaults below it: `data/pair`, `data/pair/couch`, `pair-host`, `singleton`. All effective roots must remain inside it; reject escaping overrides/symlinks. Socket names derive from its unique store as today. Account lookup is unnecessary in this mode.
- Default production Roots use actual account home. Initial explicit legacy roots come from COUCH_STORE_DIR, PAIR_DATA_DIR or XDG_DATA_HOME, and `COUCH_IDENTITY_DIR` for legacy allocation authority. Once selected, explicit incompatible overrides refuse; ordinary unset overrides use the selection. HOME still belongs to the application/user and is never rewritten globally. Couch data consumers must use the explicit resolved Pair root, not infer it from HOME.
- Strict selection parsing: bounded regular file, schema/version/path/duplicate validation, reject symlink metadata and changed physical paths. Missing selected roots refuse; no reset/re-enrollment. A missing selection permits adoption, never implies existing counters are empty.
- Adoption enumerates at most 4096 registered stores plus explicit paths. Registry inputs bounded at 4 MiB; selection/report inputs bounded; regular files only. Unknown/truncated/unreadable state refuses. Enumeration occurs only at adoption. Normal selected reads do no ownership/global subprocess discovery.
- Host/store locks are nonblocking, close-on-exec and held until Console/services/children are torn down. One failed acquisition returns immediately. No new background loops. Existing #365 admission timing remains unchanged.
- Order: host lease → inspect/select → selected store lease → revalidate sources → atomic selection publication → construct Couch/start effects. Close reverses leases. Adoption cannot fence old binaries, so stop/upgrade is a documented precondition, checked where ownership is observable.
- State table: absent + unambiguous evidence → publish; absent + conflict/unknown → refuse; existing + matching request → reuse; existing + conflicting request → refuse; malformed/missing selected resources → recovery refusal. Publication failure preserves source data. Lost response + matching persisted selection → convergent retry. Existing selection never gets silently replaced.

## Chunk 1: ownership, adoption and production composition

### Task 1 — pure selection and identity observations

Files: new `couchsingleton/model.go`, `model_test.go`, `couchidentity/inspect.go`, `inspect_test.go`.

- [x] Write table tests for empty/sole populated/multiple populated/unknown/live candidates, explicit excluded candidates, path collisions and invalid configuration. Use real AllocationState/host registry fixtures for identity consistency, missing consumed authority and regressed counters.
- [x] Run `go test ./cmd/internal/couchsingleton ./cmd/internal/couchidentity -count=1`; confirm missing behavior fails.
- [x] Implement `Roots{StoreDir, PairDataDir, IdentityDir string}`, versioned `Selection`, `Request{Roots Roots; Stores, Exclude []string}`, `Candidate`, `Report` and `DecideAdoption` as pure values/decisions. No persisted state mutation outside Manager.
- [x] Expose read-only identity inspection through couchidentity using its existing strict readers/schema validation, checking the same local/host counter relationship as allocation without allocating IDs. Return known registrations and validity evidence.
- [x] Re-run tables and identity suite; commit tested unit.

### Task 2 — Manager IO and leases

Files: new `couchsingleton/manager.go`, `inspect.go`, corresponding tests and subprocess tests.

Manager API planned:

```go
// All roots and authority are injected; zero Proc uses OSProcOps.
type Manager struct { AuthorityDir string; Defaults Roots; Proc couchcore.ProcOps }
func (m Manager) Read(request Request) (Selection, error)
func (m Manager) Preview(request Request) (Report, error)
func (m Manager) Adopt(request Request, expect string) (Selection, error)
func (m Manager) Acquire(request Request) (Selection, io.Closer, error)
```

- [x] Write failing real-filesystem tests for selection publication, absent/read refusal, digest mismatch, ambiguous store report, identity preservation, explicit retired-store exclusion, unavailable/corrupt state, symlink changes and failed writes.
- [x] Implement bounded registry union (identity registrations + Pair retention registry + requested/default/explicit stores), typed candidate inspection, deterministic report digest over relevant file bytes/observations and request. Candidate live/unknown owners refuse even when excluded; retired missing paths require explicit exclusion. Non-selected populated stores require explicit exclusion. Exclusion is retained in selection/report, never removal from identity/retention registries.
- [x] Read never adopts or takes the host lifetime lock. Preview never creates source directories or mutates source files. Acquire handles a fresh installation by creating only selected roots after deciding admission; it must not create unknown source paths to make them look empty.
- [x] Acquire takes host then selected-store lease, inspects/revalidates while protected and publishes only after both locks. Existing selection path takes no legacy scan. Adopt verifies current report digest under the same leases and returns after releasing; identical already-published retry succeeds. Reuse bounded context-aware existing transaction locks when reading source revisions; no waiting on lifetime owner.
- [x] Add subprocess winner/loser, owner kill/restart and exec lock-release tests. Verify owner diagnostic names selected store when known and preserves unreadable owner as unknown.
- [x] Run `go test -race ./cmd/internal/couchsingleton ./cmd/internal/couchidentity -count=1`; commit.

### Task 3 — CLI and resolved production runtime

Files: `couchcmd/run.go`, `cli.go`, new `singleton.go`, `singleton_cli.go`, tests; existing root consumers found by rg.

- [x] Add failing parser/runtime tests: new adoption form, refusal before actor construction, listing/message resolution consistency, overrides and isolation, injected account-home production race.
- [x] Add `couch --adopt-store <absolute-path> [--pair-data <path>] [--identity-dir <path>] [--legacy-store <path>]... [--exclude-store <path>]... [--apply <digest>]`. Without apply, emit a JSON preservation report including digest, paths, observations and blockers. With apply, revalidate and publish. Reject layout/unknown/duplicate scalar flags. Help explains source preservation, UNMIGRATED and stop/upgrade prerequisite.
- [x] Prepare OSRuntime once after CLI classification and before source mutation. Help/skill remain unprepared; fake Runtime keeps its injection seam. Owner operations call Manager.Acquire; metadata/read/message calls use Manager.Read. Adoption uses Preview/Adopt. Lifetime cleanup encloses all existing deferred service/Console teardown.
- [x] Store resolved Selection on OSRuntime. ResolveNamespace uses the selected namespace; NewCouchWith uses selected PairDataDir and IdentityDir. Add a single Pair-root helper for traces, idle fading, terminal bundle, continuation/slug/default/session readers. CurrentRepoScope remains repo-derived. Messaging consumes resolved COUCH_STORE_DIR while preserving caller scope/tag/nonce authorization.
- [x] Wrap both Runner.Start and StartBlocked to append configured Pair/Couch/identity/isolation roots to children, preserving actor-specific env and terminal capabilities. Strip conflicting inherited explicit session-artifact overrides at the existing subprocess boundary; preserve legitimate operation-specific arguments. Verify actual child argv/env through a real helper, not only a fake runtime.
- [x] Read-only CLI before adoption refuses with exact adoption command; isolated smoke helper initializes a fresh selected configuration before invoking commands that require it. Fresh launch auto-adopts sole/default roots.
- [x] Run couchcmd CLI/runtime and couchcore launch suites, then commit.

### Task 4 — production-path isolation, docs and acceptance

Files: `tests/with-isolated-pair.sh`, actual-process fixtures found by `rg 'HOME|COUCH_STORE_DIR|OSRuntime|exec.Command' cmd/internal/couchcmd`, README.md, atlas/couch.md, atlas/session-identity.md, atlas/storage-retention.md, workshop/lessons.md.

- [x] Extend isolated wrapper and actual-process fixtures to use explicit isolation root and bootstrap selection without touching production. Existing fixtures may locate roots anywhere under their temporary isolation root; validate every resolved path. All subprocess tests using Run/OSRuntime must be audited before the broad suite.
- [x] Add end-to-end preservation fixture with populated legacy identity/store/preferences and dirty checkout; preview/refusal and successful sole adoption preserve byte snapshots and slot identity. Competing live legacy lease refuses; message reconnect uses unchanged namespace. Add production-root sentinels and descendant writes in the isolation wrapper test.
- [x] Mutation-check: remove host lease → two distinct stores acquire; restore old ambient Pair-root derivation → selected-root test fails; drop isolated mode → sentinel/production-boundary test fails without accessing actual production.
- [x] Update operator docs with singleton, adoption commands, exclusions, unsupported shared home/old-binary concurrency, crash retry, and isolation. Keep existing C/N/M identity descriptions. Add review lessons about root provenance and migration refusal claims.
- [x] Run `go test ./cmd/internal/couchsingleton ./cmd/internal/couchidentity ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchmessage -count=1`, the same relevant packages under `-race`, and `git diff --check`. Run required repository checks discovered in Makefile/CI. Record exact results.
- [ ] Update issue and project, commit, then `sdlc close --issue 366 --verified '<evidence>'` (binary owns fresh review). Fix blocking findings and rerun affected checks. Open draft PR via `sdlc pr` with its required flags; no deployment or running-session cutover is implied.

## Review boundaries

One atomic feature, one close boundary; no Mx tags. Full-flow plan review and estimate precede implementation. The user authorized proceeding after the reviewed design; routine implementation choices do not need another confirmation.

## Revisions

### 2026-10-01 — plan review PQ-1/PQ-2 and fixture ordering

This revision supersedes ordering/observation details above; scope is unchanged.

**PQ-1: observation-effect separation.** Add a read-only lease observation helper
in couchcore beside supervisorlease.go: open an existing lock without O_CREATE,
reject symlink/nonregular files, attempt nonblocking flock and release immediately;
missing lock means no established lease. Read owner metadata only when held, using
bounded strict decoding, keeping held/unknown separate from free. Preview must not
call VerifiedOwner/ResolveCouchNamespace/Snapshot or another initializing path.
Reuse ThreadStore.PreviewSnapshot for source decoding where appropriate; inspect
identity state through the new read-only API. Missing paths are observations, never
created during preview. Hash source records/config/identity/registry observations
and request roots/exclusions, but exclude supervisor.lock and supervisor-owner.json
from the preservation digest. Owner contention is a separate admission predicate.
Apply acquires the host lease, checks digest from a fresh non-mutating inspection,
then acquires the selected-store lease; its own lease is explicitly supplied as
owned authority for the final source revalidation so it cannot veto itself. Source
bytes/revisions are rechecked under existing short transaction locks before publish.
Nothing attributes an observed PID to this process without the owned lease handle.

**Fixture ordering.** Before Task 3 enables production preparation, execute Task 4's
complete actual-process fixture audit/migration, including cmd/couch/main_test.go,
couchcmd/stale_store_test.go, direct OSRuntime tests and shell entrypoints. Do not
run any broad production-path suite between enabling the resolver and finishing
this isolation conversion. Fake-runtime tests do not require a new isolation root.

**PQ-2: function-level strategies.** These strategies replace the task-level case
inventories as the governing verification method:

| Function/boundary | Adversarial class | Mechanical strategy / oracle |
|---|---|---|
| DecideAdoption | permutations of candidate knowledge, exclusions, existing selection | table plus generated candidate-order permutations; same decision independent of order; unknown never auto-admits |
| IdentityStore.Inspect | malformed/truncated/regressed or missing consumed authority | real byte fixtures through existing decoder; compare all source bytes before/after; allocation counters never advance |
| read-only owner probe / Inspect | missing, nonregular, alias-changing and held/unverifiable state | temporary files + real flock; full directory snapshot remains byte-identical, including file set |
| Manager.Preview | concurrent source mutations and transient lease metadata | repeatable evidence digest, source mutation changes it while own lock metadata does not; bounded reads reject oversized input |
| Manager.Adopt / Acquire | race at absent selection, after observation and at publication | injected optional synchronous test hooks after inspection and before publish plus an injected atomic publisher; barriers coordinate two real processes/Managers; changed source refuses, failed pre-rename write leaves absent selection, post-rename error recovers by Read/identical retry |
| Manager.Read | persisted config drift or incompatible overrides | strict fixture matrices, no process probe seam invoked on selected reads; matching optional overrides converge |
| production prepare / child environment | ambient roots and inherited artifact overrides disagree with selected roots | fake account directory + real temp Manager and executable env helper; assert resolved readers and actual descendant writes use selected paths |
| adoption CLI | invalid argv and stale/apply receipts | table parser tests plus real temporary Manager command execution; rejected requests produce no stored selection |
| explicit isolation | root escape and omitted environment handoff | wrapper subprocess with sentinel files at consumed paths; mutate isolation wiring against injected account root, never real account state |

Test hooks belong to Manager's IO shell, are nil in production, and report failures
through the same functions that production executes. They do not replace the pure
policy or establish authority. Use the existing durable atomic writer by default.
The two-contender mutation test holds the first contender after observing absent
selection: with the host lock, the second refuses before reaching that hook; with
the lock removed both reach it and can attempt publication. Assert the forbidden
second admitted owner/effect, not merely a generic later configuration error.
Fresh-context plan review also confirmed no additional operator approval is needed.

### 2026-10-01 — implementation mapping and verification discoveries

Implemented Tasks 1–3 and Task 4 fixtures, acceptance cases, mutations and operator
docs. The runtime holds a revocable `runtimeOwnership` lease distinct from its
readable `Selection`; copied or released runtimes cannot reacquire authority through
metadata. `configuredRunner` covers Start and StartBlocked, with
`COUCH_PAIR_DATA_DIR` carrying the selected Pair root independently of the legacy
repo-scoped `PAIR_DATA_DIR`. Explicit Pair roots also determine a fresh default store.

Read-only seams are `IdentityStore.Inspect`, `ObserveSupervisor`,
`WithInspectionLocks` and `WithStoreInspectionLocks`. The adoption inspector validates
the exact captured retention bytes through `StoreRegistry.ValidateStructure`, avoiding
a second registry read between the evidence digest and enumeration. Supporting-root
provenance must come from retention evidence or an explicit store/Pair-root tuple;
identity registration alone does not establish the artifact root. Adoption is bounded
by a five-second context, 4096 stores, 65536 entries, 64 MiB total source payload and
4 MiB per file. No recurring discovery was added (ARCH-DRY, ARCH-PURPOSE).

The installed-command fixture had a stale managed-session argv oracle; it now asserts
the existing exact protocol, including missing/extra argument mutations. Broad race
verification exposed unsynchronized FakeGit call recording: a dedicated concurrent
regression reproduced both the race and lost calls, then passed after serializing
the fake's RunContext state. This verification sidequest changes no production Git
behavior. Both fixture corrections are included so the required tests remain useful.

The exhaustive artifact inventory still fails on 49 diagnostics already present at
base f0c1e56689469666b1aaaac708538cbf95f06f1d after generating identical runtime mirrors.
The diagnostic lists are byte-for-byte equal after classifying all seven new #366
production sources. Existing inventory drift remains tracked by #348.

### 2026-10-01 — repository verification results

Full Go suite completed. In addition to baseline artifact inventory drift,
TestCouchReferencesLocalArchiveLocatorRoundTrip fails identically on original base
and this branch (“missing slot Couch metadata”, references_test.go:350). The sole
new documentation guard failure was corrected and the full termcmd suite passed.
All requested runtime packages pass under -race, combining the broad run with the
post-FakeGit-fix full couchcmd rerun. Normal Couch core passed in 444.029s and race
in 482.823s; the time is advancing real-filesystem fixture work, not a deadlock.
The issue Log contains commands and exact outcomes for the review. No full-suite
pass or live installation migration is claimed.

### 2026-10-01 — boundary review rework

Round 1 returned REWORK with BR-1/2/3. The implementation's adoption evidence must
cover every backend represented by the inventory: global records, numbered-slot
errors and files, and recorded live/unknown incarnations in stores outside the
chosen inventory. Hashing only the global namespace and checking only its lease
is insufficient. Add regressions for corrupt external slot records, changed slot
evidence before publication, and free-supervisor/live-wrapper exclusion; extend
existing read-only/transaction seams to cover those cases. Selected-store surviving
wrappers remain eligible for the existing reconnect protocol.

The persisted selection has one size contract shared by serialization and bounded
reading. Add exact-boundary/round-trip tests and reject oversize exclusions before
publication. This applies to both explicit adoption and first launch. No source
state is discarded or migrated to address any finding. Re-run affected package and
race coverage, then the binary-owned close review; acceptance scope is unchanged.
