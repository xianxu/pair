# Couch session identities and repository families implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded work or superpowers-executing-plans for session-warm work. Track the checkboxes below.

**Goal:** Give Couch-managed Pair conversations and Zellij sessions independently allocated identities, and enforce one retained starting-directory choice per repository family.

**Architecture:** A per-user host registry allocates C; each canonical Couch store allocates independent N and M counters. Existing thread/start transactions carry allocated session bindings, while the root thread manifest owns repository-family descriptors. Standalone Pair naming and native agent UUIDs remain separate contracts.

**Tech Stack:** Go, JSON, flock, fsync/atomic rename, existing Couch journal and fake runtimes, real Git fixtures, Zellij conformance fixtures.

## Approved product decisions and implementation scope

The operator approved C/N/M counters, `C-repo-N` Pair tags, `📁C-M` terminal
names, separate conversation/terminal lifetimes, and family reservations that
survive parking. This plan adds implementation choices that need review before
code: storage location, interrupted allocation recovery, legacy compatibility,
and all subdirectory consumers. No automatic renaming of live sessions, no
native UUID migration, and no new slot-removal UI are included.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `StoreRegistration`, `AllocationState`, `AllocationRequest`, `AllocationResult` | `cmd/internal/couchidentity/identity.go` | new |
| `SessionBinding` | `cmd/internal/couchidentity/session.go` | new |
| `SessionOwnerObservation` | `cmd/internal/launcher/session_owner.go` | new |
| `Pane` | `cmd/internal/zellijpane/zellijpane.go` | modified |
| `ThreadRecord`, `ThreadStartClaim`, `StartEvent` | `cmd/internal/couchcore/thread.go`, `starttransaction.go`; persisted mirror in `cmd/internal/threadrecord/record.go` | modified |
| `RepositoryFamily`, `ResolveFamilyStart` | `cmd/internal/couchcore/repository_family.go` | new |
| `threadManifest`, `StartResolution` | `cmd/internal/couchcore/threadstore.go`, `startresolution.go` | modified |

The allocator takes a validated snapshot and returns the next snapshot and
allocated value. C/N/M are positive uint64 values; overflow refuses without
mutation. N and M are independent fields. Pair tags use a normalized, immutable
repository token plus C/N; terminal names use only C/M, in decimal. Pure tests
cover formatting, parsing, counter exhaustion, and invalid snapshots.

One SessionBinding identifies a terminal incarnation: C/M/name, owning Pair
scope/tag, and start nonce. A thread retains its last binding while parked;
the creating incarnation holds the proposed replacement until registration.
Warm reattachment retains the binding, agent-only restart retains it, terminal
recreation allocates another M. Legacy bindings carry the existing name without
fabricating C/M. The existing native transcript binding remains unchanged.

RepositoryFamily contains canonical Git common directory, primary checkout
root, and relative starting directory (`.` allowed). A family owns many slots;
each slot can host successive conversations. A pure resolver compares a path
request with the saved relative directory and either returns its launch path
or a conflict naming the existing family. Root and subdirectory requests are
distinct; explicit slot operations inherit the family's configured directory.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `IdentityStore` | `cmd/internal/couchidentity/store.go`, `store_unix.go` | new | host/store JSON, flock, durable writes |
| `SessionOwnerProbe` | `cmd/internal/launcher/session_owner_os.go` | new | exact Zellij server identity and live pane query |
| managed name handoff | `cmd/internal/couchcore/launch_existing.go`; `cmd/internal/launcher/runcli.go`, `launch_args_policy.go`, `createflow.go` | modified | blocked launcher handoff and session creation |
| family persistence | `cmd/internal/couchcore/repository_family_store.go`, `slotmigration.go` | new/modified | existing root manifest lock and journal |
| slot path routing | `cmd/internal/couchcore/threadstore_location.go`, `threadstore_layout.go`, `slotcontext.go`, `slotrecovery.go` | modified | filesystem containment and local backend routing |

Production composition injects IdentityStore at `couchcmd.OSRuntime.NewCouchWith`.
Tests use temporary host/store directories and stateful file storage, never the
operator's HOME. Ownership tests reuse the stateful Zellij world and extend it
with owner and server-generation observations. Git tests use the existing real
temporary-repository fixture.

## Storage, authority, and recovery

- Host registry: `~/.local/share/pair-host/couch-identities.json`, independent of
  `COUCH_STORE_DIR`, `PAIR_DATA_DIR`, and `XDG_DATA_HOME`, so those overrides cannot
  divide one user's Zellij namespace. Resolve from explicit HOME only at the
  composition root; inject a directory directly in tests. Registry lock lives
  beside it. Records are canonical store path -> C, plus next C and per-store
  N/M high-water floors.
- Store counters: `<couch-store>/identities.json`, with C/path, last N/M, and
  schema version. The host floors protect against restoring an older local
  snapshot; they are recovery floors, not a second name-selection algorithm.
- Allocate under host lock then allocation-state lock. Never acquire either
  while holding retention/thread-store locks. Persist the host floor first,
  then local state; only then return a value. A crash burns a number. A failed
  thread CAS can also burn a number and cannot publish a runnable child.
- Enrollment with neither file present creates C and zero local counters.
  Retry after host enrollment but before the first local write may initialize
  only when both host floors are zero. Once an allocation has been consumed,
  missing/corrupt local state refuses with exact filenames and restore guidance.
- A valid older local snapshot resumes above `max(local, host floor)`.
  Missing/corrupt host authority with existing local C refuses. A moved/copied
  store path gets a new C for future allocations; already stored tags and live
  bindings remain as recorded. Simultaneously restoring both authorities to an
  older point is not automatically detectable: document that coordinated restore
  is unsupported unless a non-regressed host authority or independently proven
  C/N/M high-water floors can be restored. Merely choosing another store path
  is not recovery: a rolled-back next C could reuse a retired store number.
  Do not provide an automatic reset/re-enrollment fallback. Document this limit
  and test that the recovery path refuses without that evidence.
- Use strict JSON, bounded reads (registry 4 MiB; local state 4 KiB), regular-file
  checks, fsync of file and parent directory, and context-aware lock acquisition.
  Extract/reuse existing durable publication behavior rather than the weaker
  rename-only OSFS writer. Lock waits are bounded by the operation context.
- Counter files stay constant size. Host rows remain permanently reserved,
  capped at 4096 stores with an actionable refusal; never silently prune a row.
  Session bindings are fields in existing thread/start records and follow their
  existing archive/retention lifecycle; do not add a per-launch receipt log.

## Terminal ownership and compatibility

Allocate N through one injected authority at both `AllocateThreadTag` and
`startFreshSlot`. Preserve artifact O_EXCL claims and exact-address validation.
Allocate M in `launchTrackedThread` before releasing the blocked helper whenever
a new terminal will be created. Record the pending binding via a new start
transition; registration promotes it to the thread's current binding. `StartRecoveredUnknown`
also promotes the proposed binding before clearing Start: established registration
proves the terminal handoff committed even when the helper died; it does not prove
that terminal is currently live. Keep the incarnation Unknown and require independent
owner observation before attach or retirement. `ReconcileStart` keeps an unresolved
claim and its pending binding occupied when registration is unknown. Rollback drops
only the pending binding when the helper/owner is proven dead and the proposed
terminal is proven absent (foreign also proves absence for this address); retain
the previous current binding and VerifiedPark. Warm rollback retains its current
binding regardless. Retry may allocate a new M only after this occupied claim has
been retired, and after probing the current binding for cold-vs-warm selection.
An established-registration recovery uses the promoted binding for this probe,
never the previous name. Unknown owner evidence never permits another cold launch. Start nonces remain independent
transaction identifiers, not another resource-name allocator.

Pass an explicit structured managed-session intent, including address, name,
nonce, and create/attach disposition. Validate it separately from native resume
options: warm attach must not accidentally acquire a cold resume profile or a
layout flag. Every managed invocation begins with the mandatory leading CLI
flag `--couch-session-v1`, before `resume <tag>`; the pre-change parser rejects
that leading option before launch effects. New parsing requires both this flag
and a matching structured intent, refusing either on its own. Consume/unset the
intent environment variable at launcher entry. Both
launcher assignment functions bypass the candidate ladder for this intent.
Creation probes the actual socket-path budget, refuses an occupied proposed
name, replaces the compatibility index association for the exact scope/tag at
its existing commit point (atomic rewrite, preserving unrelated addresses),
and never shortens/deduplicates the supplied name. Warm attach cannot fall back
to creation if its session disappeared.

One shared owner observer returns owned / foreign / absent / unknown. A live
name or cached pane sidecar alone cannot produce owned. Probe the exact server
PID/start token before and after `list-panes --json --command`; consume positive
Pair wrapper/draft command evidence carrying the exact scoped artifact paths,
using the existing artifactpath owner codec. Match complete arguments, not tag
prefix substrings. A uniquely observed different Pair address produces foreign;
missing fields, conflicting panes, parser failure, or changed server generation
produce unknown. Bound queries by the existing five-second Zellij timeout.
No new registry of observed owners is persisted.

Use that observer at warm attach and mutation boundaries, including PairSession
resolution for park/teardown and slot replacement. Cache only within one operation
and revalidate the exact server generation before effects. For the selected
address, positive foreign proof means its old named session is absent; never
send actions to the foreign session. Unknown refuses with the name and reason.
Keep passive inventory cheap: do not run pane probes on every periodic row refresh.

Legacy live sessions retain their names and tags if exact ownership is proved.
Legacy parked sessions retain Pair artifacts/native bindings and obtain a new M
at verified cold creation. Existing tags are never renamed. Ambiguous live
ownership refuses; stopped legacy names need no global renaming migration.
Tests must cover old/new runtime skew: unsupported managed intent or record
schema refuses rather than silently taking the standalone naming path.

## Repository-family integration

Add optional family descriptors to the root thread manifest, preserving existing
`slot_repositories` as the discovery projection. Validate unique common-dir
identity and canonical roots. Preview reads only; final start rechecks and
reserves the family under the existing journal/lock before provisioning or
publishing a new conversation. Concurrent first starts at different relative
directories have one winner; the loser creates no thread/child/worktree.

Infer legacy families from retained readable starting paths and the enrolled
repository. Root-only families become `.`. Coherent subdirectory families keep
their relative directory. If retained records disagree, preserve all existing
conversations and refuse new family creation with the conflicting paths; do not
pick one arbitrarily. Park and archive never erase a family descriptor. Removal
is an explicit future family-removal concern, not implicit reclamation here.

Centralize checkout containment and relative-directory projection. Use Git common
directory identity, resolving physical aliases, not repository basename or raw
prefixes. Existing-slot open/fresh and add-slot inherit the saved relative
directory; explicit new-path requests must match it. Route local thread storage
by checkout membership even when launch CWD is below root. Preserve recorded
starting/working paths in inventory; keep checkout root separately for slot
grouping and provisioning. Reject `..`, absolute relative paths, sibling-prefix
matches, and symlink escapes. A missing configured directory refuses with its
exact path before spawn; a newly provisioned worktree may remain for retry.

## Architecture and operating envelope

- ARCH-DRY/PURE: one C/N/M allocator, one owner classifier, one family resolver;
  pure format/transition tests and thin injected filesystem/process shells.
- ARCH-PURPOSE: cover new, fresh, warm, cold, continuation/switch-agent terminal
  creation paths, both tag allocation sites, both launcher name assignment sites,
  and slot storage/inventory/menu consumers. Sweep fixed `couch-<hex>` assumptions
  in production and tests, including prefix-overlapping tags such as N=1/N=10.
- ARCH-MOCK: stateful Zellij server generations/owners and real temporary Git;
  conformance checks validate emitted names and actual ownership query fields.
  Run these live checks before each release supporting a changed Zellij version;
  fixture-based owner-query parsing runs in the ordinary test suite.
- ARCH-ORDER/SECURE: allocation before spawn, pending/current binding transitions,
  no native binding invented on failure, strict JSON and exact command evidence.
- ARCH-CONSTRAINTS: only explicit start/resume/park paths incur allocation or
  owner probes; no keystroke/refresh-path IO. At most one selected-session pane
query per ownership check, bounded five seconds; no transcript scans for IDs.
- ARCH-FUNERAL: fixed counter state, bounded permanent host assignments, existing
  retention for bindings. Family rows persist while their family exists.

## Chunk 1: identity allocation and terminal lifecycle (M1)

### Task 1 — Durable C/N/M allocation

Files: create `cmd/internal/couchidentity/{identity,store,store_unix,session}.go`
and colocated tests; wire `cmd/internal/couchcmd/run.go`,
`cmd/internal/couchcore/{couch,threadtag,slotrecovery}.go`.

- [x] Test `AdvanceAllocation` and `FormatPairTag`/`FormatSessionName` with
  fuzzed snapshots and overflow inputs, asserting monotonic independent counters.
  Test `IdentityStore.Allocate` with subprocess contention and injected publication
  failures, asserting no returned identity repeats and malformed/missing authority
  refuses. Demonstrate red before implementation.
- [x] Implement validated snapshots and durable store; wire both N allocation
  sites using the same namespace-owned allocator, preserving artifact claims.
- [x] Run `go test -count=1 ./cmd/internal/couchidentity ./cmd/internal/couchcore
  ./cmd/internal/couchcmd`; expected all pass. Commit with #355 M1 reference.

### Task 2 — Explicit terminal binding and shared ownership proof

Files: create `cmd/internal/launcher/session_owner{,_os,_test}.go`; modify
`cmd/internal/couchcore/{thread,starttransaction,launch_existing,artifactcollision,
slotsessions,resume,park}.go`, `cmd/internal/threadrecord/record.go`,
`cmd/internal/launcher/{args,runcli,launch_args_policy,createflow,session_quiescence,
session_index,zellij}.go`; continuation target consumers in
`couchcore/{continuation,continuation_recovery,recovery,recovery_execute}.go`
and `checkpoint/request.go`; extend affected colocated tests and
`cmd/internal/couchcore/artifactcollision_zellij_test.go`; extend the shared
`cmd/internal/zellijpane/zellijpane.go` parser and its tests with optional actual
`pane_command`/`pane_cwd` evidence, distinct from the `terminal_command` template.

- [x] Test `AdvanceStartTransaction`/`ReconcileStart` via controllable event
  sequences asserting pending/current binding preservation, no new allocation
  while occupied, and promotion before Start clears on recovered-unknown.
- [x] Test `ClassifySessionOwner` with adversarial pane evidence and generation
  changes, using exact decoded arguments as the guard; fuzz `zellijpane.Parse`
  and owner command parsing against malformed/quoted/overlapping-address inputs.
- [x] Exercise `launchTrackedThread`, `PairSessionContext`, and park through a
  stateful Zellij owner world. Preserve the original two-scope/one-name incident
  regression and assert identity lifetimes from the issue's lifecycle table.
- [x] Test `ParseArgs` and managed intent validation with missing/mismatched
  authority; execute managed create and warm argv against immutable pre-change
  Pair and assert rejection before effects. Test both assignment functions with
  a fail-on-fallback terminal fake to enforce exact naming and warm-loss refusal.
- [x] Implement pending binding persistence, managed intent, shared owner proof,
  and live/parked legacy compatibility through these production boundaries.
- [x] Run focused suites with `go test -count=1 ./cmd/internal/launcher
  ./cmd/internal/threadrecord ./cmd/internal/couchcore ./cmd/internal/couchcmd`.
  Run isolated Zellij conformance tests for socket budget and owner snapshot.
  Mutate name handoff/owner guard to prove the boundary regressions fail.
- [x] Update README and relevant atlas map; commit, then run
  `sdlc milestone-close --issue 355 --milestone M1 --verified '<actual evidence>'`.
  Fix all blocking gate findings and record the verdict in the issue Log.

## Chunk 2: repository families and subdirectory propagation (M2)

### Task 3 — Family authority and admission

Files: create `cmd/internal/couchcore/repository_family{,_store,_test}.go`;
modify `threadstore.go`, `slotmigration.go`, `startresolution.go`, `slotstart.go`,
`slotcontext.go`, `couch.go`; extend `slotstart_test.go`, `slotmigration_test.go`.

- [x] Test `ResolveFamilyStart` with canonical aliases and conflicting relative
  directories, asserting one retained choice per common Git directory. Test
  `ReserveRepositoryFamily` with concurrent and stale-preview requests against
  a real journal, asserting admission losers create no external resources.
  Test `InferRepositoryFamily` against inconsistent retained records, asserting
  ambiguity preserves conversations and refuses admission.
- [x] Implement manifest descriptors and journaled reservation, with preview
  and commit using the same resolver and canonical identity. Preserve all
  existing conversations on legacy ambiguity and show actionable diagnostics.
- [x] Run `go test -count=1 ./cmd/internal/couchcore`; expected pass, including
  no new child/worktree/record on admission refusal. Commit with #355 M2.

### Task 4 — Every slot consumer inherits the starting directory

Files: modify `cmd/internal/couchcore/{threadstore_location,threadstore_layout,
slotmigration,slotrecovery,slotlaunch,slotinventory}.go`,
`cmd/internal/couchtty/{menu,menu_switchagent}.go`; tests in corresponding files,
`cmd/internal/couchtty/menu_add_slot_test.go`, and existing real-Git acceptance.

- [x] Test `ProjectFamilyPath` with fuzzed relative paths and filesystem aliases,
  asserting canonical containment. Exercise `resolveManagedStart`, `OpenSlot`,
  `startFreshSlot`, `storeForPath`, `validateLocalOrigin`, `projectSlotRows`, and
  the menu add-slot handler through real Git fixtures, asserting every consumer
  preserves the family's relative directory and refuses escaping/missing paths.
- [x] Use the shared projection helper throughout; distinguish explicit paths
  from slot actions that inherit the saved directory. Keep the production-boundary
  regression: spawn at `competition/arc-agi-3` in an added worktree and reject
  `arc-agi-2` after parking and Couch restart.
- [x] Run `go test -count=1 ./cmd/internal/couchcore ./cmd/internal/couchtty
  ./cmd/internal/couchcmd`, then the relevant race suites and `make build`.
  Run `git diff --check`. Document any skipped runtime build sentinel explicitly.
- [ ] Update README/atlas and issue checkboxes/Log. Run
  `sdlc milestone-close --issue 355 --milestone M2 --verified '<actual evidence>'`;
  resolve blocking findings. Close with `sdlc close --issue 355 --verified
  '<actual evidence>'`, then publish through `sdlc pr` and `sdlc merge --yes`.

## Review and approval

Fresh-context plan review approved both chunks after round 1 fixes. The issue
and plan are checkpointed locally. The operator approved this durable plan on 2026-09-30.
`sdlc change-code` is the remaining implementation gate. Derive the estimate only after that command's
plan-quality gate accepts the plan. The gate accepted and M1 implementation is
complete; its mandatory milestone review is next.

## Revisions

### 2026-09-30 — Verify the live ownership query shape

A read-only query of the incident's live Zellij session confirmed that
`terminal_command` contains the layout shell template while `pane_command`
contains the actual Pair wrapper/draft command and `pane_cwd` the checkout path.
Added the shared pane parser to the entity/task tables rather than introducing
another JSON walker. Missing actual-command fields remain unknown evidence.

### 2026-09-30 — Fresh-context review round 1

The reviewer found two blocking gaps: old launchers ignore unknown environment
variables, and re-enrollment cannot repair a rolled-back host counter. Added a
mandatory leading CLI protocol flag rejected by the previous parser, with
create/warm skew tests. Replaced reset guidance with refusal unless independent
non-regressed allocation authority is available; include the restore scenario
(snapshot next C=2, consume C=2, restore snapshot, attempt re-enrollment) as a
regression of the recovery policy. No new name allocator or recovery bypass.

Fresh-context re-review approved the revised plan with no residual blocking
findings. A read-only invocation of the installed pre-change Pair confirms that
the proposed leading flag is rejected as a flag rather than treated as an agent.

### 2026-09-30 — Implementation gate review

PQ-1: specified promotion on established-registration recovered-unknown before
Start clears, occupied retention for uncertain registration, and proven-absence
rollback/retry. PQ-2: replaced test-case inventories with named functions and
adversarial strategies. Also bounded compatibility index growth to one entry
per scope/tag and named the Zellij-version release conformance trigger. These
refinements preserve the operator-approved scope.

### 2026-09-30 — M1 integration discoveries

The independent terminal lifetime reaches continuation target-generation validation,
recovery admission, and the post-launch receipt reader. Those consumers now use
the promoted terminal binding/exact retained target witness; historical source
identity remains unchanged. Added corresponding different-M and wrong-receipt
regressions. Recovery also probes a proposed terminal before trusting a historical
Pair-address registration marker, which survives previous launches and cannot
prove a newly allocated terminal committed. Both changes complete the approved
terminal-lifetime contract rather than changing the product scope.

### 2026-09-30 — M1 boundary review: terminal authority

The first boundary review returned REWORK (BR-1/BR-2). Sweep exact proposed
terminal authority through cold admission, registration, and failed-launch
cleanup: an attached existing terminal cannot authorize a new M, and the old
address index cannot establish or retire the proposed terminal. Add composed
regressions proving no duplicate agent, no registration from the old terminal,
and no deletion of that terminal during failed replacement cleanup. Carry warm
ownership proof through retention/cmux preparation and revalidate immediately
before attachment; replace the server during preparation in the regression.
M2 remains pending this boundary. ARCH-ORDER and ARCH-PURPOSE require these
checks across all consumers of the independent terminal lifetime.

### 2026-09-30 — M1 boundary review: descriptive repository tokens

Round 2 accepted BR-1/BR-2 and identified BR-3: ASCII normalization must not
reject repositories whose names contain only Unicode or punctuation. Use a
deterministic safe `repo` fallback in the shared formatter and bound the
descriptive token so long repository basenames cannot exceed artifact filename
limits. C/N remain the identity; repository text is only a label. Cover both
new and fresh conversation allocation with the real allocator.

### 2026-09-30 — M2 integration refinements

Explicit root and bare relative paths retain their requested meaning; only
actual slot/repository references inherit. Add-slot projects the saved directory
into the primary checkout. Legacy inference matches checkout scope/common Git
identity, excluding nested independent repositories and refusing known same-repo
paths that cannot be projected. Existing ambiguous conversations remain openable
through both direct and prepared slot actions. Dependency checkouts reserve their
own family without gaining numbered-slot enrollment. Snapshot filtering uses
discovered backends so a missing legacy checkout cannot hide retained records.

### 2026-09-30 — M2 boundary review: membership and retention

BR-4 requires repository membership, not containment alone, across routing,
current/archive migration, preference migration, local-origin validation, and
snapshot filtering. Pass existing checkout-scope/common-Git identity evidence
through one shared membership rule; test outer-slot and nested independent
repository conversations together. BR-5 bounds permanent family descriptors at
4096 per store, refusing new families at capacity while preserving existing
family use. Park/archive do not reclaim reservations; removal remains a future
explicit operation. ARCH-DRY, ARCH-SECURE, and ARCH-FUNERAL shape these fixes.

### 2026-09-30 — M2 re-review: hosted actor membership

BR-5 is addressed. BR-4 also reaches hosted-session observation: an actor from
a nested independent repository must not block the enclosing slot. Extend the
shared membership sweep to in-memory/durable actors and open/fresh/recovery
callers, preserving conservative vetoes for unresolved same-scope owners. Add
production-boundary regressions for both registries. ARCH-PURPOSE requires
enumerating runtime observers alongside storage consumers.

### 2026-09-30 — M2 re-review: reconstructed backend authority

The actor correction passes. BR-4 also exposes a pre-existing conventional
path assumption now consumed as authority: reconstructed storage backends use
`primary/.git`, which is false for `git init --separate-git-dir`. Preserve the
verified common-directory identity through discovery, routing, preferences,
local-origin validation and inventory, including enrollment/restart. Add real
Git layout coverage while retaining nested-repository isolation. Conventional
paths locate checkouts; they do not establish repository identity.
