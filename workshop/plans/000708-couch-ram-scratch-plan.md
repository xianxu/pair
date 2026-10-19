---
collaboration-mode: discussion
---

# Couch RAM scratch implementation plan

> **Current contract:** the approved Revision R1 at the end of this document supersedes the original metadata proposal, metadata concepts/tests/tasks and live arrangement below. The original proposal is retained as design history, not an execution checklist. Slot-temp policy remains approved as stated, with the R1 cleanup clarification.

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use the SDLC plan and boundary gates; this is one atomic change, with one close review.

**Goal:** Move newly launched slots' temporary files and Zellij's repeating session metadata writes to usable TestScratch without disrupting surviving sessions.

**Architecture:** A Couch-owned scratch manager chooses the volume, prepares per-slot environments and reconciles the native cache route. Pure decisions consume typed observations; filesystem/process effects remain behind a portable backend. Existing launch and lifecycle authorities call the manager. No supervisor-wide environment mutation, user-settings edit, durable data relocation or new background service.

**Tech stack:** Go, existing Couch runner/process seams, POSIX filesystem operations and atomic rename exchange on supported hosts; installed Zellij 0.45.1 for isolated conformance.

## Evidence and alternatives

- The ops side doc's write-only 30 s sample counts roughly 3,700 system-temp operations and 1,840 Zellij cache operations. These are operation counts, not attributed byte totals.
- Read-only local cache inventory: 73 session directories, 72 metadata files, 10 layouts, no other files/symlinks, 308,437 bytes at inspection. This is a sizing sample, never deletion authority.
- Exact Zellij tag v0.45.1: background_jobs.rs writes metadata by pathname, create_dir_all then create/truncate/write, with no migration lock. Existing FDs can finish against the previous tree.
- list-sessions discovers live sockets; live attach also calls resurrection_layout and parses an existing session-layout.kdl. A partially copied layout can therefore break a live attach. Metadata parsing additionally drives session-manager views.
- Recommended: shared slot environment plus pre-staged atomic cache exchange, retaining originals. Preserve existing layouts/external files by links to their retained original inodes rather than republishing a potentially truncated copy.
- Rejected: rename-old then create symlink (a missing-path interval lets native writers recreate the directory); new-session-only cache overrides (old servers keep writing SSD); changing only TMPDIR (Claude replaces it in sandbox commands).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ScratchProbe / ChooseScratch | cmd/internal/couchscratch/policy.go | new |
| SlotScratchKey / SlotEnvironment | cmd/internal/couchscratch/policy.go | new |
| MetadataRouteState / NextMetadataStep | cmd/internal/couchscratch/route.go | new |
| CollectionDecision | cmd/internal/couchscratch/policy.go | new |

ChooseScratch consumes mount/free-space/write facts; the root is usable only on a separate device with at least 2 GiB available. SlotScratchKey identifies canonical repository plus slot number; EnvName supplies the human spelling. SlotEnvironment returns the three overrides or none. NextMetadataStep is the sole state/effect reducer for route changes and restart reconciliation. CollectionDecision requires complete, current ownership and liveness evidence and never turns unknown into absent. Literal value fixtures test these functions without IO.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Manager / Backend | cmd/internal/couchscratch/manager.go | new | injected mount, filesystem, native inventory and lock operations |
| OSScratchBackend | cmd/internal/couchscratch/os.go, exchange_darwin.go, exchange_linux.go, exchange_other.go | new | stat/statfs, nofollow file access, rename exchange, procutil |
| PrepareSlot / Collect | cmd/internal/couchscratch/temp.go | new | private directory ownership, liveness-gated removal |
| ReconcileMetadata / stageMetadata | cmd/internal/couchscratch/metadata.go | new | native cache routing and retained originals |
| Couch scratch adapter | cmd/internal/couchcore/scratch.go | new | thread/slot identities, existing catalogs and lifecycle evidence |
| launchTrackedThread / attachListedTerminalOnce | cmd/internal/couchcore/launch_existing.go, warm_attach.go | modified | both production StartBlocked boundaries |
| OSRuntime.NewCouchWith | cmd/internal/couchcmd/run.go | modified | production composition and isolated-root confinement |
| PairLifecycleController.applyCompletion / Couch.ArchiveThread / Couch.Reap | cmd/internal/couchcore/park.go, detach.go, reap.go | modified | one shared collection callback after confirmed terminal retirement |

Backend tests use a stateful world holding mounted volumes, free bytes, files/inodes, routes, open writers, process births and session identities. Effects update that same model; tests can pause a writer or fail any operation. The real backend runs against disposable folders and owned child processes. Never use a stateless command stub to certify session absence.

## Slot temp behavior

1. Probe /Volumes/TestScratch before preparing a launch. Match ariadne#468's DefaultRoot and 2 GiB floor; pair owns the implementation. Never mkdir an unmounted root on the SSD. No new production environment override is necessary: fixtures inject a backend/root.
2. Resolve primary and numbered slots from the existing canonical repository/slot catalog. Prepare tmp/<repo>-slot<N> with mode 0700, validate no redirected path and ownership, and serialize same-slot preparation/collection with a slot-directory lease.
3. A small immutable owner declaration binds the directory to canonical repository identity and slot number. Existing conflicting or malformed declarations cause visible fallback and are never overwritten. Unknown JSON members survive because accepted declarations are never rewritten; known fields, duplicate keys and size remain strict. A future update must retain extensions. No existing Couch/thread format changes.
4. Set TMPDIR, GOTMPDIR and CLAUDE_CODE_TMPDIR to that directory before either StartBlocked call. The shared helper covers new, cold, recovery and warm helper creation; existing runner merge semantics remove duplicate keys. Leave inherited values unchanged on fallback and leave GOCACHE/GOMODCACHE/data/transcript roots alone.
5. A warm attach cannot change an already running agent or shell's environment. Document that old panes need an operator-controlled relaunch; don't claim the first attach retrofits them.
6. Reuse a slot's directory through detach, park, restart and replacement while any prior job can still use it. Never collect on Handle.Exited, Forget, Stop or Couch Leave alone.
7. Collect after a terminal lifecycle operation confirms exit (applyCompletion, ArchiveThread and Reap call the same bounded helper), and at supervisor start for no-longer-existing slots. Use one fresh complete native process/session inventory per collection, include detached job roots and exact open holders, and retain uncertain owners. Current slots and outstanding launches hold their slot lease. Recheck the candidate identity before deletion. No periodic scanner and no filesystem work in the console completion/render path.

## Metadata route and concurrency

The native cache root is resolved from the selected HOME/platform, matching Zellij's ProjectDirs convention. On macOS XDG_CACHE_HOME alone is not isolation. Work only on contract_version_1/session_info and the manager's exact adjacent exchange path; unknown paths/links are refused visibly. All isolation fixtures keep HOME, cache, socket, runtime, config, data, RAM substitute and retained exchange path inside one disposable root.

Use one fixed adjacent counterpart, session_info.pair-scratch, rather than accumulating generations. The Couch singleton and a cache-parent flock serialize Couch changes; native Zellij writers do not acquire that flock. The plan does not claim otherwise.

| Observed route / counterpart | Event | Effect / next state |
|---|---|---|
| real directory / absent | usable RAM | prepare RAM tree; create counterpart symlink; atomically exchange route and counterpart |
| RAM symlink / real original directory | usable RAM | keep route; collect only proved-unused rows |
| RAM symlink / real original directory | absent/unusable RAM | prepare originals from RAM while preserving existing layout inodes; exchange back; unlink only the counterpart symlink |
| dangling expected RAM symlink / absent or real counterpart | reboot lost RAM | restore a real route through exchange; retained originals stay intact |
| real directory / staged expected symlink | crash before exchange | retry/reconcile from observed identities; no second directory copy |
| expected RAM symlink / real counterpart | crash after exchange | recognize completed route; originals remain retained |
| unexpected link, two conflicting real trees, partial inventory or changed inode | any | leave routing and payloads untouched; report retained state |

stageMetadata copies session-metadata.kdl using bounded reads and atomic destination publication, with source identity/size/mtime checks before and after; detected mutation stops the switch. This cache is advisory; do not interpret a stable read as a writer lock. Preserve existing layout/external files by exact links into the retained real counterpart while RAM is active, so native attach sees the original file rather than a copied partial KDL file. On reverse migration the original directory becomes the route again; don't copy a link onto its own referent. Newly created auxiliary files on RAM are copied back with the same stability checks. Native conformance must prove this arrangement before acceptance.

Atomic exchange keeps the route continuously present and preserves open FDs. Unsupported exchange, copy failure, volume disappearance or cancellation before exchange leaves the old route. After an exchange, inspect current objects and retain confirmed progress; never blindly roll back an unconfirmed exchange. All work is synchronous and bounded by its caller; no task or writer is abandoned at return.

### Pruning boundary requiring TL agreement

The cache is account-wide, but socket namespaces are not. Pair-filtered list-sessions output cannot authorize global deletion. Enumerate native server processes across socket namespaces, not just Pair names, and protect all live/unknown sessions in the RAM and retained trees.

Native standalone Zellij launches do not share Couch admission locks. Therefore do not delete unowned/foreign cache rows solely because one snapshot omitted their name. Proposed safe scope: prune known Couch-owned retired session rows under the existing lifecycle admission/retirement authority and complete native evidence; retain/report foreign or unproved rows. Ask the TL to approve this conservative interpretation before code. It does not prevent moving those rows' repeating metadata writes to RAM.

## User-visible state and recovery

Actions: launch/relaunch a slot, retry Couch start, restore RAM mount, inspect the named retained path. There is no cleanup command that bypasses an unknown inventory.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| usable volume and successful preparation | RAM scratch active | launch normally | temp/cache disappears at reboot |
| absent/full/unwritable volume before preparation | existing temp behavior, reason when useful | restore mount then relaunch | no durable data loss |
| existing pane has old env | existing pane keeps its temp root | relaunch slot | in-flight commands must finish first |
| owner conflict or live/unknown job | scratch retained, reason/path | inspect retained path; retry start/collection later | no automatic deletion |
| interrupted route before/after exchange | observed old/new route retained | retry Couch start | confirmed progress preserved |
| unsafe/unsupported metadata route | current route retained with warning | inspect named path and retry | no guessed repair |
| reboot removes RAM | real cache route restored | launch normally | disposable metadata/temp only |

## Operating envelope and artifact lifecycle

- Startup/launch/batch work, never per byte/event/frame. Probe is O(1) per launch. Directory preparation is O(1) per slot.
- One process/server inventory and one cache tree enumeration per sweep, not per candidate. Build maps once: O(P+S+F+B) work for P processes, S slots/sessions, F entries and B bytes copied. N/2N counters assert constant inventory acquisitions and linear entries/bytes.
- Initial migration cap: 4,096 entries and 64 MiB, versus the observed 73 directories and ~0.3 MiB. A limit refusal preserves the old route and reports it; no partial switch. Copy buffers bounded, no goroutine fan-out. Native work uses existing process budgets; no timeout increases.
- Owner declarations and slot leases die with a proved-unused slot folder. Collect at teardown/startup; RAM reboot is the ultimate disposable bound. No automatic deletion of a current or ambiguous slot.
- One retained disk counterpart per cache root, no backup generation growth. Its live originals survive until their sessions are authoritatively retired. Remove only proved-unused session rows; remove an empty counterpart only when no active route depends on it.
- No credentials, durable store or transcript bodies are read. Cache/owner inputs are untrusted: nofollow access, strict ownership, bounds and explicit route types. Do not recursively follow supplied links.
- ARCH-DRY: one policy and one launch helper; EnvName/runner/catalog/process authority reused. ARCH-PURE: literal policy/transition tests. ARCH-MOCK: shared stateful backend and owned native conformance. ARCH-CONSTRAINTS/FUNERAL: budgets and cleanup above. ARCH-SECURE/ORDER: confinement, typed observations and explicit exchange states. ARCH-EVOLVE: no native/thread format changes; preserve unknown owner declaration members. ARCH-PURPOSE: both measured churn sources, all supported agents and primary/numbered slots are covered.

## Named-function test strategies and guards

| Function / seam | Adversarial strategy | Independent guard |
|---|---|---|
| ChooseScratch | literal mounted/missing/same-device/low-space/error facts at floor edges | no filesystem creation unless an eligible typed probe exists |
| SlotEnvironment / slot identity adapter | generated safe/unsafe names, colliding repository basenames, primary/numbered identities | supplied overlays exactly three unique keys; unchanged durable/cache environment |
| PrepareSlot / owner decode | stateful symlink, replaced inode, unknown fields, duplicate/invalid known fields, lock contention | no path escape or overwrite; byte-identical accepted owner declaration |
| NextMetadataStep | exhaustive route/event pairs and crash-prefix sequence generation | route continuously resolves to old or prepared tree; no deletion effect for live originals |
| stageMetadata | paused truncate/write, oversized file/tree, foreign link, partial copy, existing layouts | original layout inode/bytes remain the attach input; no incomplete destination publication |
| ReconcileMetadata / exchange | failures/cancellation before and after every effect, reverse migration and dangling RAM | retry converges without growing counterparts, duplicate effects or unconfirmed rollback |
| Collect | live detached server/job, foreign socket namespace, PID reuse, unknown process, candidate replacement and concurrent launch | only positively retired owned rows may disappear; held rows byte/inode-identical |
| NewCouchWith / launch wiring | enter real production constructor with injected scratch world, all launch families/agents | mutation removing shared scratch call fails constructor-path regression |
| native integration | isolated Couch, private HOME and Zellij socket roots, owned session and shell reports before/after restart/swap | list-sessions and real attach succeed; sentinels outside fixture unchanged; exact writer joined before cleanup |

## Implementation tasks (one review boundary)

### 1. Pure policy, transition model and portable backend
- [ ] Add literal table/sequence tests first in couchscratch/policy_test.go and route_test.go; run focused tests RED.
- [ ] Implement policy.go / route.go, then manager.go with a stateful fake backend in fake_test.go; run focused tests GREEN. Count native observations and entry/byte work at N=8 and N=16.
- [ ] Add owner declaration, slot lease, nofollow confinement and atomic exchange platform adapters. Test failure prefixes and unknown-member preservation. Register every new source in artifactpath/manifest.go before broad tests.
- [ ] Commit the coherent manager/model change with issue and agent trailers.

### 2. Slot environments and collection integration
- [ ] Write failing couchcore/scratch_test.go and couchcmd/scratch_test.go tests through the real constructor, both StartBlocked call sites, primary/numbered slots, Claude/Codex/Qoder and fallback.
- [ ] Wire one shared prepare helper at launchTrackedThread / attachListedTerminalOnce. Reuse existing slot catalog/retirement evidence to collect after terminal completion and startup; preserve detach/Leave and held background jobs.
- [ ] Add isolated child shell assertions for all three environment keys, private 0700 modes and unchanged durable/cache roots. Mutation-check omission of each overlay and the constructor adapter.
- [ ] Commit environments and collection with targeted package results in the Log.

### 3. Metadata routing and isolated conformance
- [ ] Add failing stateful metadata migration/reverse/restart tests including foreign namespaces, held original FD/layout, partial writes and every crash prefix.
- [ ] Implement bounded staging and fixed-counterpart atomic route exchange; retain original layout/external files as described. Only approved ownership scopes may be pruned.
- [ ] Run isolated native conformance using testpair.ApplyEnvironment and ControlledZellij: private HOME, socket, cache/config/runtime/data, fixture RAM backend, globally unique session name. Verify listing/attach across both exchanges and an isolated Couch restart. No live cache or live Couch, no focus stealing.
- [ ] If native/layout assumptions fail, stop and revise the plan before changing the mechanism. Retain failure evidence, not a blind retry.

### 4. Documentation, gates and rollout
- [ ] Document mount/floor/fallback, Claude sandbox override, old-pane relaunch, temp lifetime, retained layouts and conservative cleanup in README and atlas (update atlas/index.md).
- [ ] Run affected package tests, artifactpath/testpair/source-comment guards, native owned conformance and N/2N budgets. Open early draft PR for CI overlap.
- [ ] Record the pushed-head full suite through sdlc local-check run --name suite; close with mandatory review and measured actuals, then request the hub's promised landing priority and run sdlc merge --yes.
- [ ] Proposed live arrangement for approval: owned isolated native checks before landing; read-only real route preview now; actual fleet migration/environment check only at the operator-controlled restart, with no pair:0 staging or live mutation by this slot. Operator then runs the same 30 s write-only fs_usage sample from ops. Record before/after counts honestly; no performance assertion from timing alone.

## Approval questions

Approve the retained-original layout links and conservative foreign-row pruning boundary, plus the isolated-now/operator-restart live arrangement. No implementation begins until the hub approves the plan.

## Revisions

### 2026-10-19 10:56 PDT — R1: approved reboot-only metadata setup

**Authority:** canonical submitted hub receipt a7a22cfc-b15d-478f-bac2-c9a1089f683f. The hub approved the slot-temp half and directed a simpler metadata mechanism. No further design approval is outstanding for this revision. Run change-code's full-flow plan-quality gate before implementation.

**Reason:** live cache migration, layout indirection and global pruning cost more machinery than this small cache warrants. The real metadata route switch can wait until the operator's next reboot, when no Zellij server is running.

**Delta:** remove atomic exchange, live copy/stability checks, layout links, cache pruning and the fixed exchange counterpart from the executable design. Keep session_info.disk permanently. No metadata file content is parsed, copied, rewritten or deleted. Existing layout bytes remain untouched. Retain the temp-folder manager/policy/cleanup design and all three environment overrides.

### Current core concepts and seams

| Pure name | Lives in | Status |
|---|---|---|
| ScratchProbe / ChooseScratch | cmd/internal/couchscratch/policy.go | new |
| SlotScratchKey / SlotEnvironment / CollectionDecision | cmd/internal/couchscratch/policy.go | new |
| MetadataState / DecideMetadata | cmd/internal/couchscratch/metadata_policy.go | new |

MetadataState is an observed tagged route (missing, real, exact-owned RAM link, unexpected), server presence (absent, present, unknown) and disk availability. DecideMetadata returns keep, keep-until-reboot, enable, restore or refuse. It holds no desired routing across events; each startup observes current filesystem facts. Pure fixtures contain literal values only.

| Integration name | Lives in | Status | Wraps |
|---|---|---|---|
| Manager / Backend / OSScratchBackend | cmd/internal/couchscratch/manager.go, os.go | new | injected mount facts, nofollow filesystem effects, process/session inventory and slot leases |
| PrepareSlot / Collect | cmd/internal/couchscratch/temp.go | new | approved per-slot temp setup and liveness-gated cleanup |
| ConfigureMetadata | cmd/internal/couchscratch/metadata.go | new | startup-only directory rename and symlink setup |
| Couch scratch adapter | cmd/internal/couchcore/scratch.go | new | canonical slot/thread catalog and lifecycle evidence |
| launchTrackedThread / attachListedTerminalOnce | cmd/internal/couchcore/launch_existing.go, warm_attach.go | modified | shared preparation at both production StartBlocked sites |
| OSRuntime.NewCouchWith | cmd/internal/couchcmd/run.go | modified | composition and one pre-launch metadata setup under supervisor authority |
| PairLifecycleController.applyCompletion / Couch.ArchiveThread / Couch.Reap | cmd/internal/couchcore/park.go, detach.go, reap.go | modified | shared collection trigger after confirmed terminal retirement |

The stateful fake Backend and portable fixture roots remain as above, narrowed to these operations. There are no exchange platform adapters, metadata-copy buffers or cache sweeps. Native server observation reuses procutil's complete named snapshot and exact server-argv classification; Pair-filtered listing is not absence evidence. Unknown native observation keeps routing unchanged. Tests inject an isolated HOME and a private simulated volume; no fixture can select the account cache or /Volumes/TestScratch by ambient inheritance.

### Current metadata algorithm

Run once when the supervisor owns startup, before any new Zellij launch. A cache-parent lock serializes Couch's own setup, while the singleton serializes its admission. Zellij does not hold that lock: the deliberate operating precondition is a reboot/quiescent native-server inventory, not a live migration guarantee. A standalone concurrent foreign launch is outside that maintenance precondition; keep the window minimal and never claim the lock controls native launches.

1. Resolve the exact native cache path from selected HOME/platform. Validate parents, ownership and route type without traversing an unexpected link.
2. Take one complete native Zellij server observation across namespaces. If any server is live, make **no filesystem changes** and emit once: `Zellij metadata stays on disk until the next reboot`. This rule wins even if RAM is absent. An unavailable/partial observation also makes no changes, with an uncertainty diagnostic.
3. With no server and usable RAM, create the private RAM destination. If session_info is a real directory and session_info.disk does not exist, rename it to session_info.disk, then create session_info as the exact RAM symlink. If the route is missing, create only the symlink. An already-correct link is a no-op after ensuring its expected destination. Never delete or overwrite session_info.disk.
4. With no server and absent/unusable RAM, replace only our exact symlink with a real empty directory. Do not restore the backup, and do not unlink unexpected links. Pair's session_serialization false makes native resurrection restoration unnecessary for the approved use case.
5. Retry after an interrupted rename by recognizing missing route plus retained backup. Retry after link removal by creating the missing real directory. A pre-existing backup plus another nonempty real route is conservatively retained with a diagnostic rather than overwritten or pruned. An empty real route may be removed only after proving it remains empty and no server is present; its backup stays unchanged. No second backup generation is created.

| State/event | Effect | Retry / loss |
|---|---|---|
| any live server | leave all paths/inodes alone; one reboot note | operator reboot; no loss |
| unknown server inventory or route | leave all paths alone; reason | inspect/retry; no guessed absence |
| no server, RAM available, ordinary real route | rename once to .disk, create RAM link | reconcile from objects after interruption; original retained |
| no server, expected RAM link already active | ensure expected destination, no backup changes | idempotent |
| no server, RAM absent, our link | unlink link, mkdir real route | repeat completes a missing route; backup never restored/deleted |
| conflicting nonempty route and backup | preserve both; visible fallback | inspect later; no overwrite |

The old atomic-route invariant does not apply: no live server may be present when the non-atomic rename/link or unlink/mkdir sequence runs. A failed effect preserves objects already created, returns a typed diagnostic and does not start a repair worker. There is no periodic routing check and no hot-path metadata observation.

### Current temp cleanup clarification

Keep the approved stable repository/slot ownership and directory leases. Cleanup is only after a complete no-holder proof: active slot/launch, native server, wrapper/agent, detached job, exact open holder or unknown observation retains the folder. The lease serializes Couch preparation/removal; it is not alleged to be held by arbitrary native/background processes. Reuse existing process/slot evidence; do not add a second host census service or scatter direct recursive removals across UI callbacks. Failed/uncertain cleanup is a retained row, not a launch failure.

### Current budgets and lifecycle

- O(1) volume probe/slot preparation; one native inventory and O(P+S+F) work per temp sweep. N=8/16 counters reject per-candidate native acquisition and superlinear entry work.
- Metadata startup: one native server observation, O(1) path operations, **zero metadata content bytes copied or read and zero session-directory enumeration** at N=8/16. No file-count/copy-size migration cap is needed because there is no copy/sweep.
- session_info.disk is explicitly retained forever by operator direction; first creation costs the existing small cache (read-only sample ~0.3 MiB). It is never refreshed into growing generations or deleted. No cache-pruning promise remains.
- The mounted RAM disk remains disposable at reboot. Durable data/transcripts/GOCACHE remain unchanged. Existing panes retain their old environment until relaunched.
- ARCH-ORDER uses the small startup decision above, not the superseded live-migration state machine. ARCH-SECURE fails closed on incomplete server inventory or unknown link. ARCH-EVOLVE makes no native/thread format change; immutable temp-owner declarations retain unknown bytes. ARCH-FUNERAL's retained backup exception is the operator's explicit choice, with bounded single-generation cost.

### Current named-function tests and guards

Keep the original ChooseScratch, SlotEnvironment, PrepareSlot, Collect and production launch-composition strategies. Replace all metadata strategies with:

| Function | Strategy | Independent guard |
|---|---|---|
| DecideMetadata | literal full product of route/server/RAM facts and interrupted prefixes | live or unknown server state never emits a mutating action |
| ConfigureMetadata | stateful fixture HOME: no-server enable, any-server keep, absent-RAM restore, rename/link failures, rerun, unexpected links and existing .disk | original backup bytes/inodes untouched; no content reads, copies or session pruning; zero writes in held-server case |
| native server adapter | exact server/client argv, foreign socket namespaces, unknown birth/argv and unavailable census | no Pair-name filter and no missing/failed observation treated as absent |
| startup composition | real OSRuntime construction with a private backend/isolated HOME | feature-removal mutation fails; sentinel live-account path never touched |

### Current execution checklist (supersedes original tasks)

- [ ] Write RED pure policy/decision and stateful backend tests for the approved temp behavior and three metadata branches; implement the minimal shared manager and platform root probe.
- [ ] Write RED constructor-path launch tests for primary/numbered slots and all three agent environments; implement the shared preparation helper at both StartBlocked sites. Cover fallback, 0700 ownership, conflicting roots, detach/reuse and held background jobs.
- [ ] Write RED no-holder cleanup tests with one shared complete inventory; implement startup and confirmed-retirement collection, retain unknown owners and instrument N/2N work.
- [ ] Implement startup-only metadata configure from the fixture regressions. Rename into one retained .disk, link only at no-server startup, leave live servers unchanged, restore an empty real directory only from our link. No atomic exchange, layout linking or pruning.
- [ ] Qualify environments with owned child shells; qualify the startup/metadata composition with isolated Couch/HOME/Zellij facts only. Three branch fixture tests are the required metadata acceptance; do not launch or modify the live fleet to exercise them.
- [ ] Register all new source files, run package and source-inventory/comment guards, document README/atlas/index and the exact live note/fallback/reboot behavior. Open draft PR for CI overlap.
- [ ] Run the pushed-head recorded suite, close through the mandatory review and measured actuals, request promised landing priority, merge normally and report the verified landing.
- [ ] The operator's next **reboot**, not a Couch-only restart, performs the real metadata switch. Existing slots require relaunch for temp environment changes. The operator runs the 30 s write-only sample afterward. No pair:0 staging or live cache mutation by this slot.

### 2026-10-19 11:04 PDT — R2: PQ-1 detached-job lifetime evidence

The first plan-quality round accepted R1 metadata behavior but raised Important PQ-1 (`cleanup-requires-lifetime-evidence`). This refinement makes the already-approved no-holder cleanup concrete; it adds no new daemon, lease protocol or permission to terminate a job.

**Evidence source:** add `ObserveScratchUsers` in `cmd/internal/couchscratch/holders.go`, behind Backend. Reuse `procutil.NativeResourceReader.Table` and its PID/birth-fenced `Environment` (resource.go), requesting only TMPDIR, GOTMPDIR, CLAUDE_CODE_TMPDIR, COUCH_THREAD_SCOPE and COUCH_THREAD_TAG. Observe all potentially relevant generations in the one complete table, not merely current descendants or launchd children; the attribution sampler in couchcmd/resources.go is diagnostic precedent, not destructive authority. Map exact scratch roots and all retained slot thread addresses to owners. A live process matching either remains a holder even after reparenting and even with no open files. Environment reads are cached only within this sweep and never reused after PID/birth replacement.

**Lifetime and uncertainty:** immutable owner declarations also record their creation instant. Only process generations born after that directory was established could have inherited this feature's new temp environment; an unconvertible birth remains ambiguous. Existing initial-environment APIs can return an empty map for protected platform binaries, so an empty map is **unknown**, never proof of no relationship. An unavailable/empty environment, unknown birth/command, disappearing process during enrichment, truncated native inventory or failed identity fence retains every candidate that process could have inherited. Definitive unrelated nonempty environments may exclude a process; known holders block their exact roots. Unknown evidence is reported and cleanup can reclaim nothing on a busy/opaque host; the existing RAM-reboot lifetime bound and low-space fallback remain, rather than weakening the proof. Original owner bytes and unknown members are never rewritten.

The inheritance model covers jobs launched with this feature's slot environment, including reparented or env-cleared descendants: unproved descendants block rather than disappear. It does not assert that ancestry or a momentarily closed FD establishes lifetime. Slot preparation/collection share the owner lease; active reservations and current slots block collection before native work. When a possible parent disappears while evidence is gathered, retain and retry a later sweep rather than assuming its children died. A newly spawned descendant either has a still-observed matching parent or forces an uncertain observation; no same-pass retry converts that uncertainty to absence.

**Cost:** one table and at most one environment read/fence per relevant process, shared by all candidates. Hash maps map roots/addresses to holders. Reduce unattributed birth times to the latest potentially relevant instant, then check each owner once; do not compare every process with every folder. Older processes are excluded only by a proved birth ordering; a missing creation time blocks. Budget O(P+S+F), and N/2N tests count table reads, environment reads and candidate visits. Native open-file inventory remains an additional blocker, not a substitute for this environment proof.

**Strategy/guard:** `ObserveScratchUsers` uses a stateful process world with a parent and a child that reparents, closes all scratch FDs, waits, then uses its inherited scratch path. The `Collect` sequence test must preserve the directory after the parent/server exits, through the child's delayed use, and reclaim only after that exact child generation is confirmed gone in a later complete sweep. The same sequence with an unreadable/empty child environment or PID reuse must retain. Mutation guards remove environment attribution or treat unknown as unrelated; either must fail the preservation assertion. Native owned-child conformance uses the same reader seam and an explicit handshake before reparent/use/exit; tests never depend on a fixed sleep or touch other processes.
