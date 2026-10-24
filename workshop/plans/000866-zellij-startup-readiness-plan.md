---
collaboration-mode: discussion
---

# Zellij startup readiness implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose execution. This is one atomic review boundary, with no Mx milestones.

**Goal:** Pair must not open or remove a probe client against a Zellij session while its first client is still initializing.

**Architecture:** Use the first executed layout pane as the readiness witness, delivered through existing exact-launch agent readiness. Serialize cooperating Pair creates and native probes within their configured Zellij namespace. A namespace lease protects the interval; a bounded outstanding-startup intent preserves uncertainty if the lease owner disappears before readiness.

**Tech stack:** Go; existing launcher terminal handoff, readiness decoder, canonical artifact paths and context-bounded file leases; Zellij0.45.1; stateful fake and shared disposable native fleet.

**Approval status:** PROPOSAL awaiting TL review. No production code changed or change-code entered. The shared namespace lease and retained uncertainty are a wider contour than adding another local wait; the TL must approve that contour before implementation. #722 stays parked, codecomplete and ready for its separate watched check.

## Findings and limits

The installed CLI reports0.45.1. Inspected tagged source **efd8fd5a89a20c07a111d248ad7fce53848d2c18** using git show; the source checkout currently points elsewhere and was not changed.

- Server starts with absent session_data (zellij-server/src/lib.rs905), publishes its listener933 and accepts regular clients950–962 before FirstClientConnected. Metadata becomes Some1058, before initial NewTab1099–1118. Removal of an early regular client dereferences None1458–1462.
- list-sessions calls assert_socket (zellij-utils/src/sessions.rs35,148–158), which sends ConnStatus. The router queues that instruction and then RemoveClient (route.rs2685–2687,2941). Connected is returned without checking session metadata (lib.rs1617–1619). Successful listing/exit is not readiness and can itself enter the failure window.
- A query action can be skipped while metadata is absent, yet return UnblockInputThread (route.rs2452–2484,2938); empty successful CLI output is not readiness either.
- Ordinary startup terminal bytes precede initialization. A genuine server Render is ordered after initialization (lib.rs1556–1569; client/lib.rs1418–1433), but distinguishing it from client-generated terminal bytes would create a new decoder. Prefer the already available exact-launch pane/agent readiness witness.
- session-metadata.kdl is lagging, non-atomic, optionally disabled, reusable after unclean death and not keyed by socket generation. Do not use its presence/mtime as authority.
- Pair's own terminalOSAttempt.observeAdmission already waits for this launch's agent-ready identity before its admission query. The title poller also waits for pane birth. These local waits do not exclude Couch's ListingPoller, other launcher preflight/listing calls, or watchBirth's late query while no pane exists. Their unsafe overlap is source-established; the exact probing process in the historical captures remains unattributed.

Primary references: [tagged server](https://github.com/zellij-org/zellij/blob/efd8fd5a89a20c07a111d248ad7fce53848d2c18/zellij-server/src/lib.rs), [tagged route](https://github.com/zellij-org/zellij/blob/efd8fd5a89a20c07a111d248ad7fce53848d2c18/zellij-server/src/route.rs), [documented commands](https://zellij.dev/documentation/commands), [documented actions](https://zellij.dev/documentation/cli-actions.html). Installed help documents listing and attach/actions, but no bootstrap readiness command; do not infer a handshake from a command's name.

Three failing captures are retained on pair#863's remote branch, including pinned main4053f3da. The third file is workshop/evidence/000863/branch-required-native-7a71f335.log, SHA256217ecb09534658cc33ca98bdc08f71ff87141e1305d0ca28b8ba42af424f0f11. Preserve them as failures; three occurrences are not a calibrated rate.

A scratch direct-server diagnostic failed in its process-list parser before its test query. It is NOT a successful standalone reproducer. TL5ead059e confirmed its exact private server cleaned; no unresolved cleanup is claimed. Per TL78d1179c, further qualification uses documented CLI/output and the existing shared fleet, with no direct socket/process-owner experiment or upstream report.

## Alternatives

1. **Local extra wait only:** smallest patch, but the create handoff already has that wait; another process's global listing still connects to the unborn server. Rejected as incomplete.
2. **Query/cache/first bytes as readiness:** no new coordination, but the query itself creates the dangerous client; cache and initial bytes have no current-initialization authority. Rejected.
3. **Recommended: shared startup/probe lease with exact pane witness:** one resource excludes cooperating probes from the hazardous interval. Retain unfinished intent on uncertain ending rather than let process death silently reopen it. This changes coordination/refusal behavior and needs TL approval.

Scope is cooperating Pair/Couch clients in one configured native namespace. An unrelated old binary or external user running Zellij directly does not take the lease; this cannot repair Zellij's upstream lifecycle bug. Rollout therefore requires rebuilt cooperating clients before claiming live protection. No Zellij patch, general retry policy, timeout increase, new upstream report or #722 change.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| StartupState / StartupEvent / ReduceStartup | cmd/internal/zellijquery/startup_model.go | new |
| StartupIdentity | cmd/internal/zellijquery/startup_model.go | new |

StartupIdentity binds the configured namespace, selected native name and launch scope/tag/nonce; it is not a socket/process census. A namespace has at most one startup intent. ReduceStartup is the sole authority for permitting query effects, removing resolved intent and retaining an uncertain outcome. Tests use values and explicit event ordering without IO. Reuse readiness.Decode and its exact identity comparison rather than define another ready-file format. Future change is another affirmative creator witness through the same reducer, never implicit success from a query.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| StartupGuard | cmd/internal/zellijquery/startup_guard.go | new | portable bounded intent + context-bounded OS lease |
| PairCachePaths startup guard resolver | cmd/internal/artifactpath/session_lock.go | modified | canonical existing namespace/cache identity |
| terminalOSAttempt Start/Admit and handoff cleanup | cmd/internal/launcher/session_handoff_os.go; session_handoff.go | modified | initial creator and exact ready witness |
| Source.ReadContext admission | cmd/internal/zellijquery/query.go | modified | all shared native query invocations |
| existing native lifecycle/mutation adapter | cmd/internal/launcher/session_handoff_os.go; recovery_terminal_os.go | modified | supported attach/retirement boundaries |
| startup conformance consumer | cmd/couch/zellij_startup_readiness_fleet_test.go | new | existing testcouch fleet, real CLI/fixtures |

StartupGuard is shared by production and a portable folder-backed fake; no fake daemon/PTY. The fake models unopened, initializing, pane-ready, query admission, late completion and owner loss, and fails a query admitted while initializing. Existing shared testcouch owns native setup/cleanup. No native ownership proof is weakened and no new independent process scanner is introduced.

## One-resource ordering and state table

A kernel-backed lease per effective Zellij namespace serializes creates against all cooperating probe admissions, including global list-sessions. Its lifetime is startup admission, never the interactive session. Queries share admission after any startup is resolved; a create takes exclusive admission before native creation. Use existing canonical lease/OS-file safety mechanisms. Do not put ambient bypass flags in the environment.

The creator's own preflight reads run under its owned permit before Start; after Start no query under that permit is permitted until the exact pane witness. After readiness, resolve intent/release exclusive permission before the creator's ordinary final native query. This prevents self-deadlock and two concurrent creators waiting on each other's queries. An already admitted read completes before a new create starts.

Actions: wait/retry the existing operation; cancel the current launch; use existing explicit relaunch/recovery.

| Internal state/event | Visible result | Recovery action | Possible loss |
|---|---|---|---|
| no outstanding intent; read admission | existing query result | ordinary operation | normal observation uncertainty |
| creator reserved, not started | startup pending; probes wait within their original deadline | wait | no native effect yet |
| native creator started, exact witness absent | startup pending; zero probe connections | wait | startup can fail without readiness |
| exact scope/tag/session/nonce readiness | resolve intent, permit queries; normal startup continues | ordinary operation | readiness proves initialization only |
| cancellation before Start | remove intent, release lease, no probe effect | retry operation | no native session created |
| cancellation/error after Start, no witness or confirmed retirement | startup unverified; retained intent blocks probes | existing explicit relaunch/recovery | partial server/agent may remain |
| lease owner disappears with unresolved intent | startup unverified, never implicit ready | existing explicit relaunch/recovery | diagnostic detail may be partial |
| stale/foreign readiness or completion | ignore for current intent, keep pending/unverified | wait/recovery | no false progress |
| truncated/invalid/newer unsafe intent | observation unavailable, no query effect | explicit supported recovery/repair | no invented empty inventory |
| exact existing retirement confirms the attempt ended | resolve matching intent only | retry operation | no broader retirement authority |

No unready cancellation is treated as server absence. Existing retirement must confirm the matching startup generation before clearing its intent; failure leaves the explicit refusal. The implementation must not automatically broaden retirement or infer authorization from a vanished lease. Old completion cannot clear a later intent.

## Constraints, persistence and trust

- Startup and waiting use the existing absolute caller startup deadline. Native observations retain their existing5s/three-attempt bound; guard acquisition consumes that same budget. No fixed startup delay, extra readiness timeout or faster polling. Existing pane evidence wait is reused.
- One creator per namespace is serialized, while ready sessions keep ordinary bounded query behavior. No background worker, unjoined goroutine or unbounded queue is added. A blocked observer returns no-answer, not an empty session list.
- Each guard admission does constant bounded work; no host-wide process/socket scan or per-keystroke probe is added. A counted fake at8/16 admissions requires linear total lease/intent reads and zero native calls while blocked; tests count query effects independently.
- Use a bounded stable lock family analogous to existing256 session-lock stripes, with at most one unresolved intent per stripe. Collisions may conservatively serialize/refuse unrelated namespaces, never replace another namespace's unresolved record. Resolve path through artifactpath; reject malformed/oversized/symlinked metadata. Max intent8KiB, mode0600. No secrets are added.
- Stable lock inodes are never unlinked. Matching ready/retired outcome removes the transient intent; owner loss retains one unresolved intent until exact supported recovery, so it cannot silently collect per-launch history. Capacity refusal precedes native creation.
- New intent fields evolve additively and preserve unknown members on rewrite, including identity. Duplicate keys and known types/required fields stay strict. Unsupported safety semantics refuse rather than erase unknown data. Use repository metadata helpers rather than trust file location.

## Implementation tasks — one full close boundary

- [ ] Add pure startup transitions and the portable stateful guard fake. Prove that every read/attach/detach probe effect is rejected before affirmative readiness, including delayed completion, caller cancellation and owner loss. Record RED before production changes.
- [ ] Add artifact resolver/guard lifetime and integrate the create handoff and shared query seam. Enumerate all direct native CLI callers using rg, classify startup observations/mutations/initial creator, and migrate relevant probe adapters; add a normal catalog/AST rule so a new bootstrap query cannot bypass the guard. Preserve existing process joins and query evidence.
- [ ] Reconcile exact retirement through the existing terminal lifecycle seam; retain uncertain intent on every unconfirmed path. Test subprocess lease exclusion, crash release with retained intent, additive metadata, cancellation/deadline consumption, no new interactive lifetime bound, and namespace isolation. Do not add a numeric-PID cleanup fallback.
- [ ] Qualify through shared testcouch with Claude/Codex fixtures, controlled pane readiness and concurrent ordinary observation traffic. Use documented CLI/output and inherited ActorCLI; retain complete output, identities, duration and hash, exact cleanup. A passing bounded sample is conformance, not a failure-rate estimate. If a new unrelated native failure occurs, preserve/classify it rather than retry blindly.
- [ ] Update atlas/README startup semantics, artifact/source inventories and audited reader certificates; run focused model/guard/launcher races, relevant complete packages, repo lint and make test-affected-inventory. Push then record sdlc local-check run --pre-close. Close through the binary-owned full review after TL-approved implementation; land only through current workflow authority and report outcome. #722 remains separate.

Risky-function test strategy: ReduceStartup gets a direct event-sequence table; StartupGuard gets a portable folder-backed fake plus subprocess lease controls; the handoff gets a controlled creator that withholds its exact ready witness; Source.ReadContext gets an independently counted fake native endpoint that rejects any pre-ready admission. The catalog test enumerates relevant bypass sites, and the native shared-harness case checks the actual emitted-profile path. No test merely mirrors a mock call list or labels an untouched path qualified.

## Architecture lenses

ARCH-DRY: existing readiness, lease, shared query and retirement seams. ARCH-PURE: one reducer owns the permission transition. ARCH-PURPOSE: exclude global concurrent probes as well as the creator's local query. ARCH-MOCK: portable fake records legal dependency states; real qualification uses shared native fleet. ARCH-CONSTRAINTS: existing deadlines, one creator per namespace, bounded state/work. ARCH-SECURE: exact launch identity and strict external metadata, no ambient bypass/ownership weakening. ARCH-ORDER: state table, explicit uncertainty and old-outcome fencing. ARCH-FUNERAL: bounded intent/lock family, named resolution/cleanup, joined operations. ARCH-EVOLVE: additive unknown-preserving new intent, refusal for unsafe formats.


## Revisions

### 2026-10-24 03:20 PDT — marker-first comparison; initial mechanism not approved

Submitted TL e242a4dc rejects approval of the initial namespace lease plus durable unfinished-intent contour and asks for the simplest creator-published per-session readiness marker first. **This Revision governs; the initial proposal above is retained as history, not permission to implement.** No change-code or production edit has occurred. In particular, the proposed separate unfinished-intent artifact and its repair lifecycle are withdrawn from the next design.

#### Marker authority

The creator publishes one ready marker only after the existing exact scope/tag/session/nonce AgentReady check, which demonstrates that a real layout pane ran after Zellij initialized session_data. Invalidate the prior matching marker before a new native create; publish atomically after validation. A missing, stale, foreign, malformed or partially written marker is unknown/starting, never native absence or permission to attach. A creator that dies without publishing produces no ready permission. Existing launch identity and lifecycle resolve stale markers; no socket-connect, successful listing, cache metadata, arbitrary delay or process census is promoted to a readiness witness.

The marker concept is useful and is the baseline. **What is not expressible with the current native CLI is skipping only unmarked sockets inside its enumeration.**

#### Complete CLI comparison (tagged source; no live socket/process inspection)

| Prober or command | Can Pair select a marked endpoint without an internal global scan? | Evidence at efd8fd5a |
|---|---|---|
| ListingPoller / full list-sessions | No | sessions.rs19–35 read_dir plus assert_socket for every socket |
| Helper reads, including -s NAME action list-clients/list-panes/dump-layout | No | commands.rs409 get_active_session -> sessions.rs283–284 get_sessions; requested name used only at415/433 |
| Action with ZELLIJ_SESSION_NAME | No | fallback commands.rs444 occurs after the initial scan, and another scan428 in Many |
| run/plugin/edit/pipe/layout actions | No | main.rs102/156/228/285 call send_action_to_session |
| subscribe, exact target | No | commands.rs459 get_active_session,478 get_sessions |
| attach, exact target/create/background/index | No | commands.rs763–766 session_exists;568 match_session_name;537 sorted scan; sessions.rs129 probes sockets |
| watch, exact target | No | commands.rs988 match_session_name -> sessions.rs412–413 get_sessions |
| public kill-session / delete-session | No | commands.rs127 assert_session -> sessions.rs474–475,435–436,412–413; delete checks session_exists499–500 |
| First-pane/AgentReady marker read, local projection and cached inventory | Yes, no native client | existing Pair evidence/readiness and cache APIs |
| Internal Rust start_cli_client / kill_session(target) primitives | Yes in Rust, but no public local CLI switch exposes them | cli_client.rs24–32; sessions.rs295–321; not a proposed new Pair IPC adapter |
| Hidden --server PATH | Not a prober; starts a listener | no safe readiness/query primitive; abandoned diagnostic is not reused |

ZELLIJ_SOCKET_DIR changes the entire lookup namespace; it does not filter selected endpoints. -s and ZELLIJ_PANE_ID do not suppress the preflight scan. A newly created filtered socket view, raw IPC client, hard links or proxy would be another mechanism; none is proposed as a small marker fix.

Consequently, simply iterating marked names in Pair and invoking the standard CLI for each still exposes every unmarked native session. A marker-only precheck of all currently visible names also has a check/use race: a cooperating creator can publish a new unready endpoint after the check and before the native read_dir/assert_socket iteration.

#### Revised narrow coordination proposal

1. Marker-only for direct readiness reads and projections: no lease or native client. Unknown/starting sessions remain represented through existing conservative launch/observation results; missing markers never synthesize an authoritative empty inventory.
2. Only **native CLI calls that perform the unavoidable global scans above** require an admission lease for the effective namespace. Hold it from complete marker qualification through the joined native command. Cooperating native creates must take the corresponding exclusion before publishing an unready session and keep it through their exact ready-marker publication. The initial creator/preflight has an explicit owned permit; its post-ready query runs after publication/release, preventing self-deadlock. Local marker reads and cached reads never take this global-scan lease. Internal direct-endpoint primitives are not added.
3. If any enumerated session has no qualifying marker, do not launch the scanning native CLI at all: retain cached/unknown/starting observations and retry only through existing bounded operation/poller policy. Do not change namespace views or claim that other ready targets can be probed independently through this CLI. This is the concrete limitation imposed by the dependency, not a need to probe sessions Pair did not create.
4. Do not add the initial proposal's separate durable unfinished-intent record. Markers are positive evidence, never a pending-operation journal. Use existing current-launch lifecycle evidence where it already records an incomplete create; do not invent a second record of the same fact.
5. **Unresolved edge before implementation:** the lease owner can die while its daemonizing child has not yet published a discoverable endpoint. No ready marker exists, but an observation of no visible candidate cannot prove that the delayed creator cannot publish during a subsequent native scan. Establish whether the existing owned create/client join and launch-intent lifecycle close that gap without new state. If not, report that exact limit to TL rather than silently restore the withdrawn durable mechanism. No prototype claims this case solved. Likewise, no claim covers foreign/older non-cooperating creators or a stale marker reused by an external same-name server; exact current-generation binding must be qualified through existing launch authority before code.

This comparison retains a lease **only where the public CLI unavoidably scans all sessions**. It does not establish that marker-only admission works through that CLI, nor approve the remaining coordination design. The simplest alternative is marker checks alone; it fails both hidden enumeration and check/use ordering. An upstream scan-free/readiness-safe API could remove the need for this cooperation, but changing Zellij remains outside this issue and needs separate approval.

#### Revised visible state/recovery table

| Evidence/state | Native probe behavior | Existing visible result | Recovery |
|---|---|---|---|
| exact marker/current generation and no unqualified namespace candidates | allow the qualified CLI under its scan lease | ordinary observation | ordinary operation |
| missing/invalid marker for a visible candidate | do not invoke native CLI | starting/unknown observation, preserve cache age | wait or existing explicit recovery |
| creator dies without ready marker | never infer readiness from owner death | unknown; no fabricated absence | existing supported recovery |
| waiting scan lease reaches caller deadline | no new native effect | no-answer/context cause | retry existing operation |
| exact ready marker arrives for this creator | publish before releasing creator exclusion | normal admission can continue | ordinary operation |
| late/foreign ready outcome | ignore for current launch | unknown remains | wait/recovery |
| creator death before delayed endpoint publication | unresolved until existing lifecycle proof is inspected | no success claim | bring precise gap to TL before code |

#### Next plan tasks, superseding initial implementation tasks

- [ ] Review this marker/CLI comparison with TL; resolve the explicitly named crash/publication and current-generation identity gaps using existing lifecycle authority, or report why another mechanism is necessary.
- [ ] Only after approval, define the small pure marker-qualification model and portable stateful fake. Independently count zero native probe admissions before exact readiness, including two sessions where only one is marked and the named command internally scans both.
- [ ] Put the optional scan lease only at unavoidable native enumeration boundaries and creator publication; do not add a pending intent store. Mechanically enumerate/guard bypass callers. Preserve caller deadlines, conservative unknown results and joined ownership.
- [ ] Retain the original bounded shared-fleet Claude/Codex qualification, normal catalog/inventory guards and recorded pre-close/full review requirements; no new standalone server/process probe. Preserve all three failing native captures and distinguish conformance from historical trigger attribution.


### 2026-10-24 — approved marker plus kernel lease; accepted creator-crash limit

Submitted TL734d1a09 approves the narrower marker/CLI revision and entry to change-code. This section is the final governing design; all initial unfinished-intent/reducer/store tasks above are superseded. **No journal or persisted pending state is permitted.**

**Accepted limit:** the creator holds a kernel file lock (flock/fcntl) from before native spawn until exact pane readiness and marker publication. If that creator dies, the kernel releases the lock automatically. A scanner may then encounter a server whose creator died, including a delayed endpoint publication: Zellij's original early-removal risk remains in that crash case. This issue protects cooperating live creators, not crash-proof native bootstrap. No lock disappearance is called readiness. Missing markers remain unknown/starting; no durable orphan latch or stronger ownership guarantee is introduced to close this explicitly accepted limit. Foreign/non-cooperating binaries also remain outside exclusion.

#### Final core concepts and files (supersede initial tables)

| Pure entity | File | Status | Test strategy |
|---|---|---|---|
| ReadyMarker / ReadyIdentity / QualifyReadyMarker | cmd/internal/zellijquery/readiness.go | new | exact identity, stale/missing/malformed evidence, direct value tests |
| PlanProbeAdmission | cmd/internal/zellijquery/readiness.go | new | complete candidate set versus positive markers; never convert unknown into absence; count8/16 inputs |

| Integration | File | Status | Dependency/fake |
|---|---|---|---|
| ProbeGuard and marker storage | cmd/internal/zellijquery/readiness_os.go | new | context-bounded kernel lease and portable private files; stateful initializing/ready fake |
| PairCachePaths readiness/scan-lease resolvers | cmd/internal/artifactpath/session_lock.go | modified | canonical configured namespace, existing cache/lease conventions |
| terminalOSAttempt Start/Admit | cmd/internal/launcher/session_handoff_os.go | modified | creator exclusion; existing exact AgentReady; marker publication before release |
| Source.ReadContext native-scan admission | cmd/internal/zellijquery/query.go | modified | same production guard, independent fake native scanner |
| Existing attach/mutation/native command boundaries | cmd/internal/launcher and current CLI adapters | modified | classify every globally scanning native caller; retain existing process join/retirement ownership |
| Native consumer | cmd/couch/zellij_startup_readiness_fleet_test.go | new | shared testcouch only, Claude/Codex fixtures, public CLI/output |

The single lease excludes native global scans from the creator interval; exact-launch positive marker checks remain separate lock-free reads. Marker storage is bounded current positive evidence, not a journal: one marker per current session, remove before replacement/after matching retirement, reuse session-retention cleanup for no-longer-needed evidence. A stale marker must never certify a new cooperating launch. Resolver/manifest and additive strict-typed marker decoding are part of the deliverable; no new secret, ownership census or live-root test lane. Before spawn, refuse capacity/metadata uncertainty rather than create an unmarkable session. Maintain at most the existing supported actor envelope in a namespace; no silently truncated selection or per-launch history.

Probe admission obtains the scan lease across complete candidate/marker qualification and the joined native CLI call; it does not release between check and use. Creator admission must not recursively reacquire that lease: its pre-Start queries use its explicit owned permit; after Start, exact readiness is observed without native CLI, marker is published and creator exclusion released, then the existing final native admission query uses the ordinary scan path. Direct marker/cache reads never acquire the global-scan lease. The existing absolute caller deadline bounds acquisition, readiness and queries; no timeout increase or extra retry loop.

#### Final state/recovery rows

| Current condition/event | Effect/visible result | Recovery/limit |
|---|---|---|
| live creator owns exclusion, readiness absent | native scans wait under existing caller deadline; startup remains pending | wait/cancel current operation |
| exact ready identity observed | publish positive marker, release creator exclusion, ordinary admission query | normal operation |
| stale/foreign/malformed marker | no target readiness; retain unknown/starting, never fabricate absence | existing explicit recovery |
| scanning CLI admitted | retain kernel lease through candidate qualification and joined CLI completion | existing bounded query policy |
| cancellation while waiting for lease/readiness | cancel/join owned work, release kernel lock, no journal | normal caller error; any native partial progress stays uncertain |
| creator death | kernel unlocks; missing marker is not success | accepted crash-only upstream risk; no claimed bootstrap guarantee |
| old completion after a new launch | cannot publish marker for the new exact identity | ignore/refuse stale evidence |

#### Approved execution sequence

- [ ] Establish RED with the shared production query seam and a stateful scanner that internally touches every candidate even for a named target; hold creator readiness and independently assert zero premature native admissions.
- [ ] Implement the final pure marker qualification and IO guard, creator marker publication/exclusion, and complete globally-scanning native caller sweep. Add the normal catalog guard for bypasses; preserve existing wire validation and cleanup authority.
- [ ] Prove deadline/cancellation, same-namespace two-session interleavings, stale identity, separate namespace isolation, subprocess kernel owner-death release and linear8/16 work. Assert accepted crash behavior as a limit, not a success claim. Run targeted and relevant complete packages/races and inventories.
- [ ] Run one bounded shared disposable native qualification with Claude/Codex and public CLI/output, retaining exact cleanup/log/hash. No direct socket/process-owner experiment or standalone upstream report. Register/attribute unrelated failures before any rerun.
- [ ] Update atlas/README, push, record sdlc local-check run --pre-close, run make test-affected-inventory, then mandatory full close and ordinary landing/report. #722 stays separately ready for operator-watched acceptance.

### 2026-10-24 — consolidated third revision after PQ-1/PQ-2 (for TL approval)

**Reason:** plan-quality round1 blocked entry to implementation on PQ-1 (`exclusion-covers-native-scan-lifetime`); PQ-2 requests explicit unknown-field preservation. The caller sweep also found a bootstrap dependency cycle. Submitted TL c8f6e5d4 permits bounded-command/creator coverage with an explicit interactive-attach limit if an attributable existing witness does not hold up. This is the one consolidated third revision requested by TL, not authorization to keep adding mechanisms. **This entire section supersedes the previous concept tables, tasks and state tables; earlier proposals remain history. No production code has changed. Await TL approval of this consolidation before rerunning change-code.**

#### Coverage and the simplest alternative

The deliverable excludes cooperating **bounded native CLI commands** from the live creator's spawn-to-exact-first-pane interval. A kernel-held namespace lock is the one exclusion resource. Direct reads of marker/cache files take no lock. There is no pending journal, namespace-wide unfinished intent, daemon, background guard service, new PTY or output decoder.

The simplest alternative, target-marker checks without exclusion, fails because even a named native action enumerates and probes the whole namespace; a creator may also publish a new endpoint between a marker check and the command. Keeping an exclusion lease for an interactive client's entire lifetime fails because an attached client could indefinitely prevent another session from starting.

**Interactive witness assessment and chosen limit:** `terminalOSAttempt.Start` only observes process creation; `Admit(false)` immediately returns and observes no native handshake. A mixed Pair PTY stream is not attributable native-client output. In tagged Zellij, alternate-screen/theme bytes are emitted at client/lib.rs973–989 after session lookup but **before** connect_to_server/send_to_server; arbitrary first bytes also do not distinguish diagnostics from client startup. The shared `Starting Zellij client!` log line lacks PID/session attribution. A new output tap/decoder or per-attach temporary logging namespace would add a mechanism. Use TL's fallback instead: **ordinary interactive attachment takes no namespace exclusion lock and retains its existing behavior. Its initial scan, and native in-process session switching/reconnect, can still race a concurrent creator.** Do not claim full protection against every native client. A fake test must demonstrate this uncovered interleaving, so future tests cannot accidentally claim it safe.

Other accepted limits remain: creator death releases the kernel lock and can expose an incompletely initialized or delayed server; foreign/older clients do not cooperate; an external same-name replacement is not certified by an old Pair marker. No marker, vanished lock or missing endpoint is called proof of absence. These are coverage limits, not new permission to retire or kill anything.

#### Readiness before bootstrap commands

Use actual **first agent-pane execution**, not AgentReady, to release creator exclusion. Metadata is assigned Some at upstream lib.rs1058, before initial NewTab1099–1118 can execute the pane command. AgentReady remains the separate, stronger agent-startup requirement and its semantics do not change.

The current main-2.kdl44/main-3.kdl55 sequence writes the pane sidecar, calls native rename-pane, then execs pair wrap; wrap starts the agent and publishes AgentReady. Waiting for AgentReady while guarding the earlier rename deadlocks. Replace the shell printf with one internal, non-native publisher at the **same first-command position**, before rename or wrap. It atomically writes the existing `Paths.PaneChecked(agent)` sidecar with its existing pane_id/cwd plus additive exact namespace, scope, tag, agent, session and launch nonce. It resolves and validates the existing launch environment, not arbitrary supplied paths. It never queries Zellij and never acquires the namespace lock. Existing sidecar consumers continue to read pane_id/cwd or existence; the new creator reader requires the exact added identity.

The creator already clears that exact sidecar before spawn. It reads the sidecar under its caller-owned startup deadline, compares the entire expected launch identity, publishes the namespace's positive ready marker itself, then releases exclusion. Thus a late old pane writer cannot publish a namespace marker for a new launch. Only the creator publishes that final marker after matching its own fresh nonce. Subsequent guarded rename can wait for this release without preventing the independent first-pane publisher from running. The right terminal and editor paths use ordinary bounded-command admission; neither must finish before first-pane publication.

#### Governing concepts and integration points

| Pure entity | File | Status | Direct test |
|---|---|---|---|
| ReadyIdentity / ReadyMarker / QualifyReadyMarker | cmd/internal/zellijquery/readiness.go | new | exact identity, missing/stale/malformed/newer additive data |
| PlanProbeAdmission | cmd/internal/zellijquery/readiness.go | new | complete passive candidate set, unknown is never empty; counted8/16 candidates |
| PaneBirthRecord codec | cmd/internal/panebirth/record.go | new | old pane fields plus exact additive launch identity; nested unknown fields |

| Integration | Files | Status | Dependency/fake |
|---|---|---|---|
| Context-bounded stable file lease | cmd/internal/launcher/session_lease.go; cmd/internal/zellijquery/lease.go | modified/new shared extraction | reuse existing kernel flock, CLOEXEC, inode validation and cancellation pacing; subprocess tests |
| ProbeGuard / positive marker store | cmd/internal/zellijquery/readiness_os.go | new | same production seam with a portable private folder and stateful fake native namespace |
| Canonical marker/namespace-lock paths and lifetime declaration | cmd/internal/artifactpath/session_lock.go; manifest.go | modified | effective child environment; transport spelling retained; separate stable lock family |
| First-pane publisher | cmd/internal/panebirth/record_os.go; cmd/pair-go/main.go; zellij/layouts/main-2.kdl; main-3.kdl | new/modified | early internal route analogous to internal exec-alarm; existing pane sidecar, no native effects |
| Creator handoff | cmd/internal/launcher/session_handoff_os.go; session_handoff.go | modified | exact pane record before marker/release, then existing AgentReady and admission |
| Bounded Go native commands | cmd/internal/zellijquery/query.go and callers enumerated below | modified | one guarded invocation seam; independently counted stateful scanner |
| Bounded shell/Lua native adapter | cmd/pair-go/main.go; cmd/internal/zellijquery/command.go; nvim/zellij_trace.lua and callers below | new/modified | internal zellij-command runner, same guard/argv/env and joined child behavior |
| Native qualification | cmd/couch/zellij_startup_readiness_fleet_test.go | new | existing testcouch only; both agent fixtures; caller deadline and exact cleanup |

The shared lease extraction must preserve `launcher.AcquireTerminalLease`'s existing external behavior through delegation; it is not a second lock algorithm. No import cycle or per-session native-lock/conversation policy is introduced. The explicit owned permit is an in-process value; no environment bypass crosses an exec boundary.

#### Lock ownership and completion table (PQ-1)

Lock order for creation is **existing per-session handoff lease, then namespace lease**. A bounded command takes only namespace admission; it must not acquire a per-session handoff lease. A marker/cache read takes neither. Creator preflight queries use its explicit held namespace permit, avoiding recursive flock acquisition. The first-pane publisher takes neither. Failure joins owned native work before releasing its namespace lease; the process-death limit above remains explicit.

| Command/path | Namespace-lock owner and release | Failure/cancellation | Proof/control |
|---|---|---|---|
| New terminal creation | terminalOSAttempt owns from before preflight/native spawn until exact first-pane match and ready-marker publication | cancel/join own client and observers, release; partial server remains uncertain, no new cleanup authority | hold pane publisher, assert zero bounded probes; publish exact record and assert release before AgentReady |
| Creator's final admission | ordinary bounded query after creator release | existing short query budget capped by startup | assert no recursive acquisition; withheld AgentReady remains a startup failure independently |
| Source.readAttempt, including custom Run / InheritedGroup | caller owns across passive marker qualification and the entire joined native command | original single deadline includes lock wait; existing result/evidence and group join preserved | native scanner touches every endpoint even for a named action; second creator cannot overlap |
| Bounded mutation, title, clipboard or shell/Lua action | same guarded command owner through child completion | use caller's existing bound, capped by the shared native-command budget; never silently detach | controlled child waits, cancels and exits; new creator admitted only after join |
| ExecKillSession | exec a Pair internal guarded-command helper first; that helper owns a normal CLOEXEC lease while synchronously running native kill-session, then exits with its status | preserve process-replacement semantics and explicit errors; no inherited lock FD or successful return into restart logic | subprocess proves lock remains while the native child scans and is free after helper/native exit |
| PAIR_KILL_CMD override that is not native zellij | retain existing override execution; it does not claim guarded native scanning | existing override/test contract | regression proves argv and non-returning exit behavior; native zellij overrides go through helper |
| Interactive attach, including native session switch/reconnect | **no namespace lock; residual race explicitly accepted for this contour** | retain current client admission/interactive lifetime and cancellation | negative coverage control shows it is not guarded; independent test proves an attached client never holds creation for its lifetime |
| Creator process death | kernel releases file descriptor automatically | no journal, no orphan latch, no inferred readiness | subprocess owner-death release; expected unknown marker state and documented upstream residual risk |

`syscall.Exec` directly into Zellij cannot carry an ordinary CLOEXEC guard. Executing the Pair helper before acquisition preserves the existing process-replacement boundary, and its synchronously joined native child makes the lock lifetime explicit. The helper accepts native argv as argv, never shell source; it is not a general command executor. Missing binary and native nonzero exits remain explicit.

#### Complete bounded-call sweep and enforcement

The mechanical class is **Pair-originated bounded public Zellij CLI invocations that internally scan the namespace**. Guarding only zellijexec.Run would miss Source.Run injection and raw exec sites. Place source-read admission before Source.readAttempt calls its runner; route raw bounded Go calls through the same command seam without repeating read classification/retry policy. Mutation calls do not inherit read retries.

| Caller family | Production sites to migrate/audit |
|---|---|
| Shared reads | cmd/internal/zellijquery/query.go, including injected and inherited-group runners; launcher ZellijSource and Couch ListingPoller callers derive from this |
| Go mutations | cmd/internal/layoutcmd/layoutcmd.go RunZellijAction; termcmd/run.go runZellij; wrapcmd/wrap.go runZellijAction/floating scrollback; clipcmd/runtime.go SetPaneColor/WritePaneKeyContext; titlepoller/runtime.go RenamePane; couch/titles.go; reviewcmd/runtime.go SpawnReviewPane |
| Retirement/replacement | launcher/osruntime.go ExecKillSession and existing recovery_terminal_os.go native commands; preserve all existing retirement proofs and operation authority |
| Bootstrap/generated shells | zellij/layouts/main-2.kdl, main-3.kdl; workbenchshortcut/terminal_command.go RightTerminalPaneShell; first-pane publisher always precedes any guarded rename |
| Lua bounded actions | nvim/zellij_trace.lua consumers (workbench_route, pair_poke, dom_transport, draft_send); nvim/review/restore_client.lua; init.lua raw vim.fn.system and floating-viewer command sites |
| Explicit non-scans/limits | zellij --version remains lock-free; hidden --server belongs to creator startup, not query admission; ordinary interactive attach/native internal session switching are the declared uncovered class; tests directly invoking Zellij are classified as fixture/qualification inputs |

Internal helper invocation is `pair internal zellij-command -- <native argv>`; a caller with a shorter deadline supplies it through the adapter's existing deadline boundary. It uses the effective inherited environment, keeps argv intact and owns native stdio/exit collection. Shell and Lua consumers route through this helper rather than copy lock paths or shell flock logic. The internal pane writer is `pair internal pane-ready`, resolved before public launch dispatch. Both follow the existing internal exec-alarm route pattern, with no public dispatcher-family growth.

Catalog/AST/source inventory guard in pkg/golint must cite #866, enumerate the above finite escape hatches and reject new unclassified direct native bounded exec sites in Go, generated shell/layout strings and Lua. Classify the interactive exception explicitly rather than suppressing every call in its file. Add negative controls for each source language and for custom Source.Run bypass. Readiness/cancellation code must be reviewed against this inventory before marking the sweep complete.

#### State, format evolution, capacity and retirement (PQ-2)

The positive namespace marker is immutable for one exact launch. No read-modify-write path is needed: observation decodes without writing; same-generation publication keeps an existing compatible record byte-for-byte; replacement is a different launch and may replace only after creator authority/old marker invalidation under the namespace lock. Matching retirement removes the old marker only after the existing lifecycle proof; observation never unlinks an unmarked/unknown server. Stale/external same-name reuse remains the stated limit.

Known JSON fields and duplicates remain strict. Decode preserves unknown raw members at both top level and nested identity in explicit raw-member maps, and encode merges them without dropping fields. The additive pane-sidecar codec does the same for any same-generation rewrite. Tests load a newer record with top-level and nested unknown fields through the older decoder, encode it again and assert semantic equality; same-generation publication also asserts byte-for-byte preservation. Wrong known types, duplicate keys and unsupported safety semantics refuse. A legacy pane record stays readable by existing consumers but cannot satisfy the new exact-launch creator qualification.

Use bounded regular-file metadata helpers and atomic writes. Namespace lock inodes are a fixed256-stripe family and never unlinked; collisions only serialize. The marker store holds only current positive evidence, never a record per retry. Register explicit resolver/retention ownership and integrate matching create/retirement cleanup; do not mistake a manifest label for an implemented sweep. Marker bytes are bounded8KiB. Before admitting a new marker, enforce a complete128-current-session envelope per configured namespace, reclaim only markers proved superseded/retired by existing authority, and refuse capacity/uncertain selection before native creation. Reader overflow returns unknown rather than truncating candidates. This capacity bound covers durable residue even when a process dies; it is not permission to remove an unknown session. Direct positive-marker/cache reads remain lock-free.

Actions are wait/retry the existing operation, cancel the current launch, and use existing explicit recovery; no new recovery command is introduced.

| Internal condition | Visible result | Recovery | Possible loss/limit |
|---|---|---|---|
| live creator holds namespace lock, no exact pane record | bounded command waits within caller deadline; startup pending | wait or cancel launch | no new bounded probe effect |
| exact first-pane record, marker published | release namespace; existing agent readiness/admission continues | ordinary operation | pane readiness proves native metadata, not agent readiness |
| malformed/stale/missing candidate marker | observation unknown/starting; no fabricated empty inventory | existing explicit recovery | legacy sessions may require explicit relaunch before full qualification |
| bounded-command lock wait reaches deadline | no-answer/context cause; no child spawned | retry existing operation | original observation unavailable |
| bounded child cancelled/exits | join child, retire command resources and release | existing caller error/retry | partial native effect remains explicit; no mutation replay |
| creator death or unconfirmed cancellation | kernel unlocks, marker remains absent/unknown | existing explicit recovery | upstream early-removal risk in crash case remains |
| interactive attach/native reconnect overlaps creation | existing attach result; no coverage claim | existing operation recovery | upstream residual race remains; never hold lock for interactive life |
| marker capacity/selection uncertainty | refuse before create, no truncated inventory | existing explicit retirement/recovery | no new session started |

Hot-path cost: for C passive namespace candidates, one complete candidate pass plus at most C bounded marker reads and one guarded native invocation per attempt, O(C), not a native call per candidate. Existing read retries remain at most3 within one original5s/caller deadline. Count8/16 candidate/admission cases and require exactly one candidate traversal, at most C marker reads and zero native scans while the live creator is blocked. No background worker, new timer cadence, lifetime journal or timing-only performance assertion.

#### Single implementation/review boundary

- [ ] Record RED against the production Source seam with a stateful fake that models native global enumeration, withheld first-pane execution and exact identities. Include the named-target/unready-neighbor case, creator publication ordering and bootstrap rename-before-AgentReady cycle.
- [ ] Implement the shared lease extraction, pure codecs/admission and bounded portable store; prove subprocess owner death, inode safety, namespace isolation, existing deadline consumption, capacity refusal and additive nested-field preservation. Do not claim that the accepted crash case is solved.
- [ ] Add the non-native first-pane publisher and creator lifetime integration; keep AgentReady unchanged. Prove that exact pane publication releases exclusion while AgentReady is withheld and that a stale pane generation cannot release a new creator.
- [ ] Complete the bounded Go/shell/Lua caller inventory through the one guarded command seam; add exec-helper lifetime tests, helper argument/exit tests, source-language bypass controls and a negative interactive-coverage test. Run each focused regression RED/GREEN, then the relevant complete packages/races and inventory checks.
- [ ] Qualify on the shared disposable native fleet for Claude and Codex, with ordinary bounded observation traffic concurrent with startup; retain exact cleanup, complete output and hashes. No private server/socket/process-owner experiment and no upstream issue without a standalone reproducer. State historical trigger attribution remains incomplete.
- [ ] Update atlas/startup documentation and durable evidence; push final source; record sdlc local-check run --pre-close, run make test-affected-inventory and use the binary-owned close review/ordinary landing path. #722 remains separately parked for its operator-watched rerun.

**Review disposition request:** PQ-1 is answered by explicit command-class ownership and the TL-authorized interactive limitation, not by inventing an unsupported release event. PQ-2 is answered by immutable same-generation publication plus explicit nested raw-member preservation and round-trip tests. TL should approve/reject this consolidated contour once; another substantive design correction means stop and re-scope per c8f6e5d4.
### 2026-10-24 04:02 PDT — TL-approved recovery scope narrowing

**Reason and authority:** integration exposed that marker-qualified `SessionPresent` runs before `KillServer` in `quiesceAuthorizedSession`, preventing existing explicit recovery of an unmarked endpoint. Authenticated submitted TL delivery e39af197-9300-46c9-8503-a328080ebb48 directs this narrowing and continuing implementation. This revision supersedes the third consolidation's universal marker qualification, not its one-lock ownership, joining, capacity, exact publication or retirement rules.

**One resource, two admissions (ARCH-DRY):** every cooperating bounded native invocation still takes the same namespace lease. Automatic background probes (the ListingPoller and unattended helper reads) additionally require complete positive marker qualification. Explicit recovery (`quiesceAuthorizedSession`, exact `KillServer`, and its confirming `SessionPresent`) retains today's unmarked-endpoint behavior, taking exclusion without marker qualification. Explicit bounded mutations likewise use exclusion; helper reads retain qualification. No new lock, journal, recovery command, serialized bypass or cleanup authority is introduced. The distinction is caller policy carried by the existing in-process Source seam; it does not make a marker authority to kill.

| Integration | Files | Delta | Proof |
|---|---|---|---|
| Shared guard admission | cmd/internal/zellijquery/readiness_os.go; query.go; command.go | factor the same lock into qualified automatic reads and exclusion-only explicit commands/recovery; `Source.ExplicitRecovery` is an explicit trusted caller choice | unmarked private endpoint rejects automatic runner but allows explicit runner; both wait behind live creator; cancellation starts no child |
| Existing recovery observation | cmd/internal/launcher/session_quiescence.go; zellij.go | only explicit recovery's SessionPresent selects exclusion-only reads; default Source and ListingPoller remain qualified | private stateful recovery retires its exact authorized fake server while automatic probe remains refused |
| Bounded helper lifetime | cmd/pair-go/main.go; cmd/internal/launcher/osruntime.go | early helper catches termination, cancels and joins native child; native kill replaces Pair with this helper | argv/exit, subprocess cancellation and helper exclusion through joined native lifetime; non-native override unchanged |

| Internal condition | Visible result | Recovery action | Limit |
|---|---|---|---|
| automatic probe sees unmarked endpoint | unknown/starting; no native probe | wait, cancel current launch, or existing explicit recovery | no fabricated empty inventory |
| explicit recovery sees unmarked endpoint, creator lease held | bounded wait | wait or cancel recovery | no recovery scan inside live cooperating creator window |
| explicit recovery sees unmarked endpoint, creator lease free | existing exact-authorized recovery proceeds | existing recovery | **explicit recovery can still touch an unready server outside the creator window**, including after creator death; upstream panic risk remains accepted |

- [ ] Add exclusion-only admission and Source policy tests RED/GREEN, preserving qualified default behavior and original budgets.
- [ ] Wire existing explicit recovery observations and test authorized retirement plus the automatic-probe refusal control.
- [ ] Complete bounded caller/helper integration, then the still-required third-consolidation inventory, marker retirement, native qualification and one close boundary. No native qualification or stage of an incomplete head.

### 2026-10-24 — implementation ownership and qualification reconciliation

The approved mechanism is implemented and qualified. The actual first-pane IO publisher lives in `cmd/internal/launcher/pane_ready.go`, alongside inherited launcher/artifact binding validation; `cmd/internal/panebirth/record.go` remains the pure additive codec. This replaces the predicted `panebirth/record_os.go` location without adding an authority. The finite source rule belongs in Pair's existing `internal/repolint` extension, rather than editing upstream shared `pkg/golint`; `scripts/test-affected.py` registers its exact scanner tests. Marker retirement lives in `zellijquery/readiness_retirement.go` and is wired to the existing launcher lifecycle/recovery proof boundaries. Artifact families and resolver witnesses are registered and individually tested. Source ownership, lifecycle and limits are mapped in the atlas.

All governing implementation/control tasks above are satisfied by the source and evidence in the issue Log, including the approved explicit recovery narrowing, counted8/16 work, interactive negative coverage and shared Claude/Codex native qualification. Historical unchecked task lists are retained as design history. The remaining work is the recorded pre-close check, this single binary-owned close boundary, and TL-sequenced operator smoke/landing. The native test drives the already-existing public hub form under authenticated TL approval because the shared harness's #863 fix is not yet landed; it does not import or depend on that unlanded implementation.


### 2026-10-24 — close review corrections within the approved scope

The first close review returned REWORK: the earlier completion statement did not account for non-list helper reads. BR-1 is corrected by enumerating the read-only CliAction variants in supported Zellij0.45.1 (tag efd8fd5a89a20c07a111d248ad7fce53848d2c18): list-clients, list-panes, list-tabs, current-tab-info, are-floating-panes-visible, dump-screen, dump-layout and query-tab-names, plus top-level list-sessions. The existing automatic-read marker admission now covers that complete class. The actual early CLI route tests every read against an unmarked endpoint with bare, named and absolute-executable adapters, then verifies positive-marker admission and explicit mutation access. No new mechanism or explicit-recovery restriction is added.

BR-2 replaces the remaining Unix-socket fixture's generic temporary root with existing testpair.ShortRoot, with a long inherited TMPDIR regression. BR-3 adds the README rollout steps and accepted limits. BR-4 removes unused PlanProbeAdmission: the earlier governing concept-table entry is superseded by the implemented pure QualifyReadyMarker/codec functions and the single qualifyCandidates IO loop; counted8/16 tests exercise that production loop. ARCH-DRY favors one policy implementation. The generated close-review ledger preserves the findings; targeted verification, corrected-head native qualification, a fresh recorded pre-close and normal close retry remain required.


### 2026-10-24 — BR-5 authoritative resolver and handoff ownership correction

This append-only correction supersedes the resolver rows in both earlier concept tables (including the governing row formerly at line255). The new resolvers are **not** implemented in artifactpath/session_lock.go; that existing file is unchanged. Git name-status against the issue branch point and direct source inspection establish this replacement mapping:

| Implemented concept | Actual file | Status in this issue | Verification |
|---|---|---|---|
| PairCachePaths.ZellijNamespaceLease, ZellijReadyDirectory, ZellijReadyMarker | cmd/internal/artifactpath/zellij_readiness.go | new | TestZellijReadinessConstructorsUseCanonicalNamespace; TestZellijReadinessKeysRemainBoundedAndCurrent |
| Resolver bindings, source classification and lifecycle declaration | cmd/internal/artifactpath/manifest.go | modified | TestZellijReadinessManifestProtectsLifecycleArtifacts; TestZellijReadinessSourcesArePreciselyClassified |
| Creator handoff and first-pane-to-marker release | cmd/internal/launcher/session_handoff_os.go | modified | existing creator handoff controls, exact identity/release-before-AgentReady regression |
| Existing handoff policy definitions | cmd/internal/launcher/session_handoff.go | unchanged; reused | no new policy implementation here |

The previous implementation reconciliation still governs the first-pane publisher location (new launcher/pane_ready.go), the local repolint inventory ownership and exact retirement file. The BR-4 correction still removes PlanProbeAdmission in favor of the single production qualification loop. All other governing core file/status rows were checked against Git name-status; this revision records location/status corrections only, with no new mechanism, state or source change. ARCH-PURPOSE requires the durable concept map to describe the implementation reviewers can inspect.


### 2026-10-24 — candidate test ownership and atomic rollout rule

Authenticated TL11f5d193 requires the Lua callers and the Pair executable to update together. A staged build updates both from the selected candidate, so normal staged use has no caller/helper version skew. Upgrade/relaunch cooperating processes together; copying only the Lua assets onto an older Pair is unsupported because that older executable may lack internal zellij-command. This records the rollout contract of the approved helper, with no new runtime mechanism.

The test-review and test-lua recipes now build the in-tree Pair and prefix PATH with the suite-owned PAIR_TEST_BINARY directory when supplied, otherwise the in-tree candidate bin directory. This selection survives the existing empty-environment fixture bootstrap; tests no longer resolve the operator's installed Pair accidentally. A regression supplies an incompatible installed Pair first on inherited PATH and checks both recipe targets through that bootstrap, for shared and in-tree candidates. Fixtures also retain private native namespaces and their actual helper/effect assertions. ARCH-PURPOSE and ARCH-CONSTRAINTS make executable and native endpoint ownership part of the integration boundary.


### 2026-10-24 — BR-6 exact candidate alias, independent of basename

The directory-only candidate prefix is superseded: PAIR_TEST_BINARY may name any absolute executable, including plain-pair, and a sibling pair may be a different build. Both public targets now enter the existing with-pair-test-binary.sh owner before their recipe bodies. It uses the existing pair-test-binary.sh builder to alias the exact supplied executable as pair (or build the ordinary fresh suite image when none is supplied), exports that alias on PATH and as PAIR_TEST_BINARY, and cleans it only after the joined target run. Internal recipe targets avoid recursion without another state flag. Standalone public targets select their just-built in-tree candidate; supplied candidates retain exact image ownership.

The Make/bootstrap regression now supplies shared/plain-pair beside an incompatible shared/pair, with another incompatible installed Pair first on inherited PATH. Both review and Lua targets must execute the exact shared candidate; directory-only selection reproduced wrong-sibling output. The coupled Lua/helper rollout rule remains unchanged. This is test orchestration through the existing builder/owner, with no new runtime admission or readiness mechanism.


### 2026-10-24 — dry-run-safe recipe ownership qualification

Public candidate wrappers retain exact executable aliasing but pass the selected Make through an indirect recipe variable. A literal MAKE reference makes GNU Make execute that line even during -n/-p inspection; the existing recipe-ownership test reproduced this real side effect. Its wrapper audit now follows the explicit one-level recipe body, continues checking every script against its group, and checks that the shared candidate wrapper is already a full-selection input. A negative control rejects an unclaimed script inside the wrapper body. No protected affected configuration or runtime admission policy changes. Per TL01e33fe3, all41 prerequisite targets run locally to completion before another recorded suite.


### 2026-10-24 — launch-path coverage and named missing-pane bound

Authenticated TL917953f0/e38364e9 accepts preserving external asset-root selection. Both shipped main-2/main-3 layouts and their embedded copies publish pane-ready before guarded rename/wrap. Every ordinary create, cold resume and restart converges on those selected layouts; warm resume/attach uses create=false, with no creator publication wait. Older/custom files accepted from PAIR_HOME or another asset root remain caller-owned and must include the same first-pane command. Update binary, Lua and layouts together; no fail-fast parser, silent replacement or new admission mechanism is added.

At the existing startup bound, a creator without accepted exact pane evidence reports terminal first-pane readiness unverified, naming its session and retaining the underlying deadline/read error. An accepted pane followed by missing AgentReady keeps the later readiness failure distinct. The shared registration/attach fixture now executes production PublishPaneReady before starting its fake agent, matching the real layout rather than bypassing the creator release witness. RED/GREEN cases defend missing publication, exact identity and the separate AgentReady phase. ARCH-DRY preserves one publisher and ARCH-CONSTRAINTS preserves the existing bound.
