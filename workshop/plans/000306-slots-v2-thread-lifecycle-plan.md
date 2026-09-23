# Slots v2: local state and durable recovery plan

> **For agentic workers:** Consult AGENTS.md Section 3 for delegation. Apply
> superpowers-writing-plans to finalize integration details before change-code;
> execute approved tasks with TDD and the existing SDLC gates.

**Goal:** Make numbered slots durable directory-backed Couch threads whose local
state supports resume or start fresh without retiring the slot.

**Architecture:** Discover numbered slots from verified environment directories.
Store their Couch-owned state under `pair-slotN/.couch/`; global listings are
rebuildable indexes. Share existing lifecycle/process protection with ordinary
Couch threads while giving managed slots local storage and recovery behavior.

**Tech Stack:** Go, existing threadrecord validation and lifecycle transitions,
atomic/journaled storage, SDLC workspace v2, Git, #305 WorkspaceReadiness.

**Status:** Operator-approved direction; revised planning work, not an executable
implementation approval. The earlier global-store plan and its reviews are
superseded. Storage/migration details below must be resolved and reviewed before
change-code. No code changes or estimate are claimed.
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

File names above are the proposed minimal layout; validate them against the
existing transaction/retention integration before implementation.

| Local state | Purpose and lifetime |
| --- | --- |
| thread.json | Stable Couch slot identity, current conversation reference, presentation metadata, timestamps/layout, existing launch/park/continuation recovery facts |
| preferences.json | Agent and supported arguments, independent of the current conversation; survives start fresh |
| continuation.md | Couch-owned checkpoint/orientation material; follows the retained continuation's existing cleanup rules |
| Transaction journal/lock if required | Reuse current atomicity/recovery semantics at the local storage boundary; no new domain state or setup phase |
| Retained conversation/recovery evidence | Preserve superseded handles and damaged records before replacement; storage layout and bounded retention must be specified before implementation |

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

## Core concepts and existing seams

This table identifies real existing reuse points and the required responsibilities.
It intentionally does not promise speculative new class names. Final engineering
planning must name the concrete functions and signatures after storage exploration.

| Concept / existing symbol | Current home | Planned disposition |
| --- | --- | --- |
| WorkspaceIdentity / ParseWorkspaceIdentity | cmd/internal/couchcore/workspace_identity.go | Reuse verified numbered membership; distinguish dependency/ordinary worktree |
| WorkspaceProvisioner.Ensure / WorkspaceReadiness | cmd/internal/couchcore/provision.go, provision_dispatch.go | Reuse repeatable setup; leave Git metadata ownership intact |
| ThreadRecord / threadrecord.Record | cmd/internal/couchcore/thread.go; cmd/internal/threadrecord/record.go | Separate durable slot identity from current conversation handle; retain validators and lifecycle facts |
| PathLaunchPreference | cmd/internal/couchcore/launchprofile.go | Local home for numbered preferences; #308 owns preference UX/inheritance |
| ThreadStore | cmd/internal/couchcore/threadstore.go, storejournal.go | Introduce local authoritative storage without duplicating lifecycle logic; preserve atomic record/preference updates |
| AdvanceStartTransaction / CommitStartClaim | cmd/internal/couchcore/starttransaction.go, threadstore.go | Reuse existing concurrent-start protection and interrupted-start recovery |
| ClassifyThread / DecideResume | cmd/internal/couchcore/actionableinventory.go, resume.go | Retain process/conversation evidence; slot presence must remain visible when resume is unavailable |
| launchTrackedThread | cmd/internal/couchcore/launch_existing.go | Shared cold readiness hook before final conversation proof; warm bypass |
| storagegc store registry and references | cmd/internal/storagegc/stores.go, collector.go | Recognize local current/history references before global records retire |

Pure decisions: slot discovery classification, open-versus-create intent,
recoverable-conversation versus explicit-fresh outcome, parked creation policy.
Integration: Git/SDLC observation, local record/index IO, native session evidence,
existing launch effects. Use the existing stateful fakes and real temp stores/Git.
Do not implement another lifecycle state machine merely to rename existing states.

## Chunk 2 — revised work breakdown

### 1. Finalize storage and migration integration before implementation

- [ ] Trace ThreadStore callers, Pair record readers, scope/tag assumptions and GC
  references. Choose the smallest storage seam preserving the existing pure
  lifecycle transitions, revision checks and atomic record/preference writes.
- [ ] Specify local initialization, schema validation, evidence retention layout,
  index rebuilding and repo-root discovery. Specify which independent existing
  evidence can recover a conversation when thread.json is damaged; do not add
  another authoritative slot registry merely to recover the first one.
- [ ] Specify migration of existing numbered records without changing native
  conversation handles. Disambiguate legacy multiple-record cases explicitly.
  Select one authority at every step; use repeatable local publication and an
  explicit cutover rule, not concurrent writes to global and local copies.
- [ ] Make all readers/retention consumers honor that cutover before retiring the
  global copy. A failed/interrupted migration preserves the previous authority;
  an index can lag without losing the locally owned slot. Test each boundary.
- [ ] Define repo parked-check/create ordering under the existing owner/creation
  mechanisms. Do not add repository ownership, cross-instance coordination,
  whole-store snapshot CAS, or separate workspace reservation machinery.
- [ ] Write the concrete integration signatures/files/test cases, run fresh plan
  review and operator review, then full `sdlc change-code --issue 306` gates.

### 2. Implement authoritative local records and derived discovery

- [ ] TDD: discover verified :1/:2 with no global rows; missing local metadata
  still yields visible existing slots. Reject foreign/conflicting conventional
  directories; nested dependency clones yield no extra slot rows.
- [ ] Implement local record/preferences/continuation persistence and shared store
  seam. Migrate numbered records with their conversation handles unchanged.
- [ ] Test interruption/corruption at each migration/cutover step; verify global
  index deletion/rebuild and duplicate index entries do not duplicate slots.
- [ ] Exercise GC with references held only in local current/history records:
  those artifacts must remain protected. Define end/retention for every new file.

### 3. Implement shared open/resume/start-fresh behavior

- [ ] TDD at real operation boundaries: open live, detached, parked, uninitialized,
  incomplete, failed-conversation and damaged-record slots; visible recovery actions
  must work. No archive prerequisite for any confirmed-stopped recovery case.
- [ ] Keep existing duplicate-start claims/CAS and stopped-process reconciliation;
  prove competing resume/fresh requests cannot launch two agents in one slot.
- [ ] Add cold readiness before final native/continuation binding validation for
  new, resumed and replacement conversations; warm attach skips compilation.
- [ ] Preserve and retain old conversation evidence on explicit start fresh;
  missing/ambiguous binding alone never proves no live agent remains.
- [ ] Make create-another obey parked :0/:N checks; opening/repairing existing slots
  stays available. Show all parked blockers; another repo remains independent.
- [ ] Wire exact slot references and actual paths through CLI/start form/switcher
  actions; keep grouping layout in #307. Rework #305's advisory selector so it no
  longer treats an existing slot as a free container for a different durable thread;
  remove it if the new allocation path does not need it, rather than retain dead code.

### 4. Acceptance, docs and publication

- [ ] Run primary/:1/:2 through create, park/resume, continuation and start fresh
  using stateful production-boundary tests. Snapshot dirty/untracked files, active
  branches and unpublished commits in host and private dependencies before/after.
- [ ] Verify closed-set slot discovery plus open-world primary compatibility,
  exact references, readiness retry, and unchanged supervisor/separate-store behavior.
- [ ] Run focused suites and deterministic race tests; full `go test ./... -count=1`,
  changed-package vet, `make runtimebundle-generate`, `make pair bin/couch`.
- [ ] Run isolated real SDLC/Weave conformance using #305's fixture; fake the agent
  launcher. Repeat when consumed contracts change and in #309 acceptance.
- [ ] Update README/atlas/project and downstream #307/#308/#309 acceptance contracts;
  document metadata recovery and retained conversation behavior before close.
- [ ] Checkpoint observed evidence. `sdlc close` owns the mandatory boundary review;
  fix blockers, then commit close records, `sdlc pr` and `sdlc merge --yes`.

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
