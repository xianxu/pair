# Slots v2: local state and durable recovery plan

> **For agentic workers:** Consult AGENTS.md Section 3 for delegation. Apply
> superpowers-executing-plans or superpowers-subagent-driven-development to
> execute approved tasks with TDD and the existing SDLC gates.

**Goal:** Make numbered slots durable directory-backed Couch threads whose local
state supports resume or start fresh without retiring the slot.

**Architecture:** Discover numbered slots from verified environment directories.
Store their Couch-owned state under `pair-slotN/.couch/`; global listings are
rebuildable indexes. Share existing lifecycle/process protection with ordinary
Couch threads while giving managed slots local storage and recovery behavior.

**Tech Stack:** Go, existing threadrecord validation and lifecycle transitions,
atomic/journaled storage, SDLC workspace v2, Git, #305 WorkspaceReadiness.

**Status:** Operator-approved direction; concrete engineering plan passed fresh
review after corrections. Earlier global-store reviews are superseded. Awaiting
operator approval of this detailed plan before change-code; no code or estimate yet.
**Issue:** `workshop/issues/000306-slots-v2-thread-lifecycle.md`.
**Flow:** Full; in-place branch when entering implementation. Estimate follows
plan-quality review. Preserve unrelated local work.

## Chunk 1 — authoritative model

### Three lifetimes

- **Slot:** the verified numbered environment and its durable Couch identity.
  Exists even with no running agent, a failed setup, or missing Couch metadata.
- **Conversation:** current or retained agent conversation within the slot.
  Resume preserves it; explicit start fresh selects a new conversation handle.
- **Process:** the running Pair/agent incarnation. Existing start, park and resume
  protections govern its creation and recovery.

One slot is one durable thread by construction. Remove the proposed second step
of allocating independent threads into supposedly free slot directories. A missing
global row does not mean a slot is vacant. Conversation handles may change while
slot identity and preferences remain. Native artifacts keep their existing opaque
scope/tag addresses; slot addresses such as :1 identify the durable directory.

### Storage boundary

```text
worktree/pair-slot1/
  .couch/
    thread.json
    preferences.json
    continuation.md       # when a retained continuation needs materialization
  pair/
  ariadne/
```

The local layout uses the existing ThreadRecord and PathLaunchPreference formats.
Current membership derives directly from thread.json; no local manifest is stored.

| Local state | Purpose and lifetime |
| --- | --- |
| thread.json | Stable Couch slot identity, current conversation reference, presentation metadata, timestamps/layout, existing launch/park/continuation recovery facts |
| preferences.json | Agent and supported arguments, independent of the current conversation; survives start fresh |
| continuation.md | Couch-owned checkpoint/orientation material; follows the retained continuation's existing cleanup rules |
| Transaction journal/lock if required | Reuse current atomicity/recovery semantics at the local storage boundary; no new domain state or setup phase |
| Retained conversation/recovery evidence | Preserve superseded handles and damaged records before replacement; existing archive grace for valid records; bounded raw recovery backups below |

Derive slot number, host path and repo membership from convention plus Git proof.
Do not duplicate those as an editable authoritative inventory. Missing local state
is not sufficient evidence to manufacture a new conversation: first inspect
existing native bindings and processes. Malformed, unsupported-version and
inaccessible data have distinct recovery outcomes; preserve rather than overwrite
uninterpretable state. Absolute stored paths are checked against the current
verified directory before use.

Keep global: Couch supervisor ownership and current separate-store behavior;
primary/arbitrary-path thread stores; only rebuildable listings of managed slots.
A persistent repo-root discovery list may still be needed so Couch can find repos
from any cwd. It contains locations, not slot lifecycle truth. Rebuilding means
re-enumerating known roots (or an explicitly supplied repo), not scanning the whole
machine or pretending deleted root-discovery information is recoverable magically.

Keep existing stores: native transcripts and Pair's bindings/drafts/scrollback/
terminal sidecars. Local records reference them; this version is not a portable
whole-session bundle. Keep #305's Git creation lock/intent and setup-success marker
at their existing homes, with no second success marker in `.couch/`.

### Open versus create

**Create another slot:** check repo-wide parked work; choose an unused number;
use #305's existing guarded creation/readiness operation; initialize local Couch
state and launch. A concurrent collision refuses/reopens the same slot according
to observed evidence, never silently increments the number after acceptance.
Do not allocate over incomplete known slots to escape failed setup.

**Open :N:** discover that slot, finish provable setup, then attach/resume or offer
start fresh. Opening does not allocate a new durable slot/thread. `.couch` missing
after successful provisioning is a normal initialization/recovery case. Failure
does not remove the directory or reset its active branch to remote main.

**Start fresh here:** explicit operator choice within an existing slot. Confirm
there is no competing live owner, retain old conversation/recovery evidence, then
launch using the slot preferences. Fresh creation failures leave the previous
history available. The action needs no archive gesture and remains available when
another slot is parked because it adds no slot. A recoverable parked conversation
is not silently discarded to clear the repo's parked-work blocker.

Primary :0 remains in the existing open-world store and participates in repo-wide
parked checks. Ordinary worktrees/dependency clones do not acquire numbered slot
identities or automatic threads. Existing tag-based references stay usable.

### Recovery outcomes

| Observation | Ordinary open/action |
| --- | --- |
| Existing live/detached agent | Switch/reattach using existing ownership checks; no compile on warm attachment |
| Stopped, recoverable conversation | Run readiness if cold, revalidate binding, resume |
| Incomplete Git/setup | Repeat #305 readiness for the same number; no retry flag |
| Interrupted agent start/park | Reuse existing reconciliation; distinguish confirmed dead from unknown |
| Missing Couch state | Recover verified identity/binding/preferences where possible; otherwise initialize with explicit fresh-conversation choice after absence proof |
| Corrupt state or lost/ambiguous conversation | Preserve evidence; offer usable same-slot recovery/start-fresh path, without archiving the slot |
| Unknown live ownership | Explain and resolve ownership first; fresh action cannot bypass it |
| Permissions, unavailable tool/service, conflicting Git state | Show concrete failure on this slot; preserve files and retry through ordinary open after resolution |
| Unsupported future record version | Report version mismatch and preserve it; use a compatible binary or deliberate recovery, never automatic downgrade overwrite |

Directory existence guarantees a stable place to recover, not guaranteed launch
success despite environmental failures. Origin/main is only a creation baseline.
No resume/recovery path resets files, commits, branches or private dependencies.

## Core concepts

### Pure entities

| Name | Lives in (under cmd/internal/) | Status |
| --- | --- | --- |
| SlotIdentity | couchcore/slot.go | new |
| SlotObservation / DecideSlotOpen | couchcore/slot.go | new |
| ThreadTarget / ThreadRowKey | couchcore/threadtarget.go | new |
| ParseWorkspaceReference | couchcore/workspaceref.go | new |
| SlotAllocation / SelectNewSlot | couchcore/slotallocation.go | new |
| ThreadRecord / PathLaunchPreference | couchcore/thread.go, launchprofile.go | reused wire formats |
| ActionableThreadSummary / ThreadSummary | couchcore/actionableinventory.go, threadinventory.go | modified |
| StartResolution | couchcore/startresolution.go | modified |

SlotIdentity is a physical primary/environment/host path plus repo common-dir
identity and positive number, derived from checked WorkspaceIdentity. No new UUID.
ThreadTarget is a tagged choice of ordinary ThreadAddress or SlotIdentity; a slot
may have no current conversation address. ThreadRowKey is comparable and uses
slot host path for slot rows, existing scope/tag for ordinary rows. UI selection
therefore survives fresh conversation replacement and can represent damaged or
uninitialized slots without fabricating native conversation tags.

SlotObservation combines discovery, record decode result and existing lifecycle
proofs. DecideSlotOpen yields attach/resume/initialize/offer-fresh/refuse; it writes
no persistent status field. Existing ClassifyThread continues to classify an
actual conversation. New-slot selection is lowest unused positive number, never
an existing directory. Partial/unreadable known slots return attention instead of
silently moving to a higher number. Remove obsolete SelectWorkspaceNumber and its
dead-symbol waiver once SelectNewSlot replaces its former advisory contract.

Workspace references accept canonical :0/:N and repo:N; unqualified repo means
:0. Resolve context through SDLC, preserving its dependency-context refusal for
bare :N. Opaque native tags remain exact references. Resolve an existing slot
without requiring thread.json; reject malformed/overflowing/noncanonical numbers.
StartResolution preserves original input, action (open/create/fresh), chosen target
and accepted profile; CommitArgs reproduces that request. Changed target/profile
refuses submission. Preview performs no setup or migration.

### Integration points

| Name | Lives in (under cmd/internal/) | Status | Wraps |
| --- | --- | --- | --- |
| SlotCatalog / OSSlotCatalog | couchcore/slotcatalog.go | new | registered worktrees, conventional environment dirs, SDLC v2 |
| SlotCatalogFake | couchcore/slotcatalog_fake.go | new | mutable discovered roots/slots/errors |
| StoreLayout | couchcore/threadstore_layout.go | new | global manifest vs local single-current layout |
| storeForAddress / storeForPath | couchcore/threadstore_location.go | new | ThreadStore backend routing |
| EnrollSlotRepository | couchcore/slotmigration.go | new | bounded local copy plus global authority cutover |
| RecoverSlot / StartFreshSlot | couchcore/slotrecovery.go | new | existing ownership probes and local journal |
| ObserveSlotSessions | couchcore/slotsessions.go | new | native scoped bindings/artifacts and current session/process evidence |
| prepareTrackedWorkspace | couchcore/slotlaunch.go | new | #305 readiness before final cold-launch proof |
| CouchReferences | gcruntime/references.go | modified | all five retention entry points |

Keep `Couch.Threads *ThreadStore`. Add backend-only children rooted at `.couch`,
selected by `storeForAddress(address) (*ThreadStore, error)` and
`storeForPath(physicalPath) (*ThreadStore, error)`. Children do not route recursively.
Global namespace remains the supervisor identity; physical backend root owns its
journal/lock/path checks. Do not make `.couch` a second supervisor namespace.

Route these primitives, where lifecycle callers already converge: CreateThread,
GetThread, GetPathLaunchPreference, updateExistingThread, advanceSuccessfulStart,
deleteThreadIf, archiveThread, RestoreThread and continuation materialization.
RecordPath becomes an error-returning routed lookup; failures cannot silently
fall back to a global path. Snapshot/ArchivedThreads aggregate all discovered
backends while preserving native identities. No duplicated park/resume state machine.

StoreLayout owns current membership and membership-journal entries. Global mode
keeps manifest membership. Local mode derives zero/one current address from
thread.json and emits no manifest entry. Invalid local bytes produce a SlotObservation
error and block destructive retention; they do not produce an empty current set.
Each local current read verifies its actual scope/tag against the requested address.
Old tags only address retained archives and cannot mutate the new current record.
Local preferences are one existing PathLaunchPreference at preferences.json;
continuation.md stays a derived rendering of the embedded checkpoint. Existing
start registration journals thread.json and preferences.json atomically.

## Repository discovery and metadata routing

Enroll physical primary roots in the existing global threadstore manifest as
`slot_repositories`; upgrade that manifest to schema 2 on first enrollment. Schema
1 stays supported for ordinary stores. This root-location list is necessary
configuration, not a slot inventory: all numbered membership/current records come
from disk. Keep only an in-memory native-address-to-local-store index. Rebuild at
startup/explicit repo open and on an address miss; detect duplicate addresses rather
than last-write-wins. Slot current-record operations re-read local state even on
index hits. An index holds locations, never decoded lifecycle authority.

A v2 manifest is also the compatibility fence: old Couch and old GC reject it
before accessing a partially understood working set. Create it under the existing
Pair retention coordinator before retiring global references. A missing/unreadable
manifest in an initialized registered namespace blocks GC; it is not an empty
inventory. Losing root configuration requires explicitly reopening/re-enrolling
known repos; it cannot be recovered by a whole-machine scan.

OSSlotCatalog uses ProvisionIO's bounded subprocess seam and the shared SDLC JSON
parser. Reuse/expose WorkspaceProvisioner.identity rather than another parser.
List registered Git worktrees and conventional `<repo>-slotN` directories once per
repo operation; validate main host linkage. Nonexistent/incomplete hosts remain
attention entries with exact paths, never ready launch authority. No recursive
dependency scan or automatic dependency enrollment. Refuse symlink-swapped or
foreign paths. Read-only inventory and GC preview never initialize `.couch`.

`COUCH_STORE_DIR` remains the supervisor namespace. Metadata/continuation commands
resolve inherited scope/tag through the same route map, rebuilding from enrolled
roots if needed, regardless of cwd. No new environment-selected store authority.
The Pair launcher has no direct ThreadStore reader; its trusted launch profile,
scope/tag, and global native artifacts remain unchanged.

## Migration and atomicity

Migration occurs on explicit managed-repo open under the existing supervisor
ownership boundary, not on inventory refresh. Enroll one repo at a time:

1. Discover and validate candidates outside locks. Acquire existing locks in order:
   Pair retention coordinator, global store, then local store(s) in physical-path
   order. Revalidate source records and local path identity before copying. No
   network, agent or session probes while these locks are held.
2. For each slot, select its unique legacy current record, preserving exact native
   address, revision, profile, metadata and continuation. Copy its matching preference
   and existing archived records/grace to the local layout with local journals.
   Also copy slots with zero current records but existing path preferences or
   archives; those facts do not depend on a selected current conversation.
   Multiple plausible current records require explicit tag selection after normal
   ownership checks; no newest-record heuristic. Conflicting local data refuses.
3. Publish all local copies durably first. Before root enrollment commits they are
   staging, and the global records remain authoritative. A retry compares exact
   source/content and reuses matching staged copies; changes are not overwritten.
4. One global journal commits schema 2 plus root enrollment as its FIRST entry,
   then removes migrated
   global records/preferences/membership. After enrollment, local is authoritative
   for every slot in that repo, even if local decode later fails. No stale-global
   fallback. Readers hold/recover that existing global journal before routing.
5. For a newly created repo with no legacy slot records, enrollment is simply the
   schema/root-list publication before local records. A failed new setup still
   leaves a discoverable directory to reopen, not a global hidden reservation.

This avoids a separate migration receipt/state machine or cross-root journal:
local preparation is repeatable; the existing global journal owns the single
root-enrollment commit. Bound one enrollment to 128 slots / 64 MiB copied metadata
(initial workstation envelope; refuse larger batches with explicit diagnosis).
Use content/revision revalidation on interrupted retries. Runtime cannot mutate
staged local records before enrollment. Existing running owners must be brought
under this Couch's proven control before migration; no competing-supervisor protocol.

Catalog IO is split into bounded filesystem candidate enumeration and Git/SDLC
validation for launch. GC uses only the former while holding retention coordination:
read candidate directories/local metadata, protect referenced owners and block on
unverifiable paths. It must never invoke Git, SDLC or session probes under that
lock. Mutation validates Git outside locks and rechecks physical path identity at
publication. This preserves one discovery convention without requiring slow probes
in maintenance critical sections.

GC stays registered by the original namespace. ReadStoreRetention and
CouchReferences.Snapshot enumerate enrolled local stores independently of the
in-memory index. Recover, Onboard, Detach and Forget use the same resolver;
archive references carry the actual backing-store locator so receipts reach the
right journal. Local current records protect global native artifact owners by
scope/tag. Existing archive grace/record hashes/GC receipts retain their meaning.
Missing/corrupt slot current state, roots or pending local journals block GC for
unresolved ownership instead of silently collecting potentially referenced data.
GC apply can recover journals; preview remains strictly read-only.

## Same-slot recovery and fresh conversation

ObserveSlotSessions collects a complete candidate set for the slot's known native
scope from existing session-name bindings, scoped artifact owners, local current/
archive records and Couch's hosted registry. Use existing process-start identities,
SessionPresence/DetachedSessions and native-binding resolvers on that set. A failed
scan, unreadable binding index or unanswerable process probe yields uncertainty.
This is observation of managed Pair sessions, not a promise to discover arbitrary
unmanaged shell processes. A verified absence is required before a fresh launch.

A single proven surviving conversation can reconstruct a missing local record
using verified path, native address and available launch profile; missing profile
requires explicit agent/profile selection. Multiple historical conversations never
choose themselves: show candidates and offer explicit fresh after absence proof.
An unsupported record version is preserved and requires a compatible binary or
explicit supported recovery; ordinary open cannot overwrite it.

StartFreshSlot is a distinct confirmed action, available on slot rows with no
resumable conversation. It can also be deliberately selected on a stopped slot.
For live slots the operator first parks using the existing action. Do not silently
kill or abandon an uncertain agent to make fresh possible. Recovery from a checkpoint
remains available and reuses the same local storage and readiness boundaries.

For explicit fresh, allocate a new opaque native tag using the existing collision
claimer. Under the local store lock, revision-check the old record (or compare its
preserved invalid bytes), retain the old valid record via existing archive/grace
paths, and install the new current record with its existing ThreadStartClaim in
one local journal. Keep slot preferences/name/description; new conversation summary
and continuation do not pretend to describe the old conversation. A competing fresh
or resume request observes the same claim and cannot install another current record.
Do not implement this as public ArchiveThread followed by AllocateThreadTag.

A failed first start clears its claim using existing recovery; the slot remains
visible even if no current conversation record survives. On failed fresh launch,
old conversation evidence remains available for inspection in history. Ordinary
slot open offers retry of the new conversation or explicit start fresh once absence
is proved; no historical restore action is promised by this change. Never silently
restore an older record over an uncertain new agent. Existing process failure handling
still distinguishes pre-release confirmed failure from post-release uncertainty.

Valid retired records use existing 60-day archive retention; retained native owners
are visible to the same GC. Damaged raw records are preserved by digest under
`.couch/recovery/` before replacement: at most 16 files / 64 MiB per slot (matching
the existing 4 MiB metadata read envelope). Never truncate or auto-delete unknown
bytes to fit; on overflow report an exact export/remove action before recovery.
Those exceptional backups are operator-owned forensic files and are not scanned
as native ownership authority. Keep unresolved native ownership protected until
normal evidence or explicit stopped-slot fresh recovery establishes the current set.

## Launch and admission sequence

- Preview chooses create/open/fresh and exact slot, without side effects. Explicit
  :N always targets that slot. Ordinary `couch <path>` opens existing work; the
  start form's create action allocates another slot when appropriate.
- New-slot creation reads all same-repo parked evidence, including primary :0 and
  ordinary threads whose verified common Git identity matches. It refuses on
  parked or uncertain relevant ownership and lists activation/inspection actions.
- The existing couchtty operationQueue serializes interactive creation/park
  mutations. Reuse it and #305's Git collision protection; concurrent direct callers
  remain bound to the accepted number, and local no-replace current-record creation
  plus existing claim CAS admits at most one agent. Add no owner-local creation
  mutex, repo ownership lock, durable reservation or store-wide snapshot CAS.
- If setup returns failure/cancellation, leave the incomplete slot visible and
  release operation scope. Retrying ordinary open resumes readiness. If parked work
  appeared during setup, retain the newly created slot but stop before first launch
  and name the blocker. Opening that already-existing slot is then an explicit
  action under the existing-slot rules, never an automatic loop around refusal.
- All non-warm launch shapes hold their existing start claim during readiness.
  prepareTrackedWorkspace runs before final resume-binding/continuation proof and
  before starting the blocked helper. Recheck physical identity/profile after setup;
  accepted-profile drift returns to preview. Primary/unaddressed starts skip setup.
- Existing-slot attach/resume/fresh does not allocate slots and remains permitted
  while siblings are parked. Existing claim CAS, helper handshake, registration,
  cancellation and interrupted-owner recovery prevent competing process starts.

## Chunk 2 — implementation and verification

### Task 1 — local layout, routing and root enrollment

Files: new couchcore/slot.go, slotcatalog.go, slotcatalog_fake.go,
threadstore_layout.go, threadstore_location.go, slotmigration.go and colocated tests;
modify threadstore.go, storejournal.go, continuation_store.go, retention.go,
archive_gc.go, gcruntime/references.go and their tests.

- [ ] Write failing tests for local single-current layout, atomic successful-start
  record/preferences update, journal recovery, wrong-tag refusal, routed metadata
  mutation from another cwd, and schema-1 primary compatibility.
- [ ] Implement the layout adapter and routed primitives; do not copy lifecycle
  methods. Make namespace-vs-backend root path validation explicit.
- [ ] Test root enrollment fault injection after each local publication and global
  journal entry. Test stale source, conflicting local bytes, ambiguous candidates,
  native-address preservation, zero-current preference/archive migration and
  old-format refusal immediately after the first cutover journal entry and before
  global refs disappear.
- [ ] Wire all five GC methods; test local-only references, index rebuild,
  malformed/missing records, receipt replay against a newer archive, and concurrent
  publication/GC. Preview must make no writes.
- [ ] Run targeted store/retention/gcruntime tests and race sequences; commit.

### Task 2 — slot discovery, references and stable rows

Files: new couchcore/threadtarget.go, workspaceref.go, slotallocation.go and tests;
modify actionableinventory.go, threadinventory.go, threadmetadata.go,
startresolution.go, startup.go and couchtty/menu.go/menu_refresh.go/menu_render.go.

- [ ] Test disk-only slots, incomplete/foreign/symlink candidates, lost index,
  missing/corrupt thread.json, nested dependencies and arbitrary worktree behavior.
- [ ] Implement SlotObservation and ThreadTarget/ThreadRowKey in the common
  inventory. Migrate menu selection/in-flight row matching to stable row keys;
  keep native-address maps for terminal/attention/continuation ownership.
- [ ] Test :0/:N/repo:N and opaque references through CLI and operation dispatch;
  preview fingerprint must retain chosen slot/action. No synthetic native address.
- [ ] Wire full repo evidence in startupAsks and update its narrowed/full proof
  equivalence tests. Remove obsolete advisory selector and one-thread-per-path prose.
- [ ] Run inventory, metadata, startup and menu suites; commit.

### Task 3 — recover, start fresh, and readiness

Files: new couchcore/slotsessions.go, slotrecovery.go, slotlaunch.go and tests;
modify recovery.go/recovery_execute.go, couch.go, resume.go, switchagent.go,
continuation.go, launch_existing.go, threadtag.go, ops.go, operationdispatch.go,
couchcmd/run.go, couchcmd/continuation.go and corresponding CLI/menu wiring/tests.

- [ ] Build stateful SlotCatalogFake plus controlled readiness/session fakes on
  the existing FakeRunner/FakeProcOps/artifact fixture and real temporary stores.
- [ ] Test actual open/fresh operations across live/detached/parked/no-record/
  corrupt/unsupported/multiple-history states. Confirm recoverable choices work,
  unknown ownership refuses, old evidence survives and no archive gesture is needed.
- [ ] Implement RecoverSlot and atomic fresh replacement; keep current transition
  validators and native launch handles. Exercise concurrent fresh/resume and dead
  owner recovery using pause channels, never timing-dependent sleeps.
- [ ] Wire readiness for new/cold/fresh-existing launches; test warm bypass,
  cancellation/failure then ordinary retry, binding changes during compile,
  primary unchanged and responsive console cancellation while setup runs. The
  existing serial mutation queue may delay another repo's queued launch; this task
  does not introduce parallel scheduling. Repo independence means admission policy,
  not concurrent execution of queued mutations.
- [ ] Cover parked primary/sibling/all parked producers at every create entry
  point, park during setup, and explicit existing-slot recovery while siblings
  remain parked. No number escalation on partial setup.
- [ ] Run targeted lifecycle/CLI/TUI tests and race sequences; commit.

### Task 4 — integrated acceptance and publication

- [ ] Real temporary Git host/private dependency fixtures: primary + :1 + :2,
  independent park/resume, continuation, lost binding, corrupt record, start fresh.
  Compare dirty/untracked bytes, active refs and local SHAs before/after recovery.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd
  ./cmd/internal/couchtty ./cmd/internal/gcruntime ./cmd/internal/storagegc -count=1`
  and targeted `-race` tests; vet changed packages.
- [ ] Run `make runtimebundle-generate`, `go test ./... -count=1`,
  `make pair bin/couch`, and isolated installed-SDLC/Weave conformance using #305's
  ProvisionFixture plus fake agent runner. All must exit zero. Repeat live contract
  checks when provisioning/identity dependencies change and in #309 acceptance.
- [ ] Update README, atlas/couch.md, atlas/workspace-provisioning.md, atlas/index.md,
  project and #307–309 consumer specs. No grouped tab layout or preference UX in #306.
- [ ] Reconcile concept tables and acceptance evidence; `sdlc close --issue 306
  --verified '<observed evidence>'` owns the one fresh-context boundary review.
  Fix blockers, commit review trailers, `sdlc pr`, `sdlc merge --yes`, preserve
  unrelated local files. One close boundary; no artificial milestone tags.

## Constraints and simplifications

Reuse #305's command/output limits and cancellation; keep setup outside store
critical sections and registration timeouts. Discovery is scoped to known repo
roots and numbered environments, not the whole filesystem. Old records, unfamiliar
schema and ownership probes are external evidence; unknown never grants permission
to launch or overwrite. No new credentials or production-path test fixtures.

Removed from the earlier proposal: free-existing-workspace selection, durable
workspace occupancy enumeration, store-wide admission snapshot/byte-CAS, separate
slot-thread allocation and archive-before-replacement. Existing process claims,
atomic persistence and cautious ownership probes remain necessary. Local storage
adds reader/migration/GC integration work; fewer domain states do not make this a
prompt-only change or eliminate real IO failures.

ARCH-DRY: share lifecycle transitions and provisioning. ARCH-PURE: decisions consume
observed facts. ARCH-PURPOSE: local authority includes every reader and GC consumer.
ARCH-MOCK: stateful storage/session/setup fakes plus real Git/SDLC conformance.
ARCH-CONSTRAINTS: bounded probes/setup and scoped discovery. ARCH-SECURE: validate
Git membership, records and current ownership. ARCH-ORDER: exercise interrupted
migration and existing process claims. ARCH-FUNERAL: bounded retained evidence and
explicit slot removal; no automatic workspace deletion on conversation retirement.

## Revisions

### 2026-09-23 — initial global-store proposal (superseded)

The initial plan added stable workspace bindings, free-workspace selection and
atomic repo admission via whole-store snapshots and ThreadStartClaim. Its spec and
plan reviews passed before operator discussion exposed the wrong lifetime model.
Those approvals do not apply to this revision; original content remains in Git.

### 2026-09-23 — authoritative .couch and durable slot recovery

Reason: operator treats numbered directories as a closed set of durable threads
and requires resume or start fresh without archiving the slot. Delta: local Couch
authority, rebuildable global listings and shared lifecycle execution. Preserve
existing supervisor behavior, primary storage, native/Pair sidecars and #305 Git
metadata. Replace free-container allocation and state machinery with existing-slot
recovery; add explicit migration, reader and retention integration work. This
revision captures the agreed design and next planning tasks, not implementation.

Documentation review: fresh-context review approved this project/spec/plan revision
without blockers. This confirms consistency with the agreed direction; it does not
replace the unfinished storage/migration design or its implementation approval.

### 2026-09-23 — concrete storage and recovery integration

Read-only exploration found lifecycle IO concentrated in ThreadStore and all
external retention consumption in gcruntime.CouchReferences. Chose a routed
ThreadStore with single-current local layout, repository-root enrollment as the
authority cutover, shared GC routing/version fence and slot-target UI identity.
Specified migration ordering, no synthetic native addresses, atomic same-slot
fresh replacement, finite raw evidence retention, and actual boundary tests.
The existing process/supervisor model remains; no additional durable admission
reservation or whole-store snapshot transaction is introduced.

### 2026-09-23 — engineering review corrections

Review found a promise to resume arbitrary archived conversation records without
a corresponding operation. Removed that unrequested promise: failed fresh retains
old evidence for inspection and offers retry/start-fresh in the durable slot.
Clarified migration of zero-current slots with retained preferences/history and
made the schema-2 fence the first global cutover journal entry.

Fresh-context engineering recheck approved these corrections with no remaining
blockers. Implementation still requires operator plan approval and change-code.
