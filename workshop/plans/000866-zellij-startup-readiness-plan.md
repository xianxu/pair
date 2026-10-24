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
