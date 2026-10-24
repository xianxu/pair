# Human attention routing implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks, or superpowers-executing-plans for warm integration work. Steps use checkbox syntax for tracking.

**Goal:** The bell carries authenticated hub notifications, failures of the operator's casts/drives, and actionable events with no connected owning hub; other actionable events reach their hub and the feed.

**Architecture:** Give each producer a typed class and feed one pure routing table. Console owns feed/attention projection; a bounded lifetime-owned delivery shell resolves verified ownership and uses the existing generated-system-message ingress. Keep existing occurrence/attempt ownership where a producer already sends to its hub.

**Tech Stack:** Go, existing Console/messageService/broker, stateful fake host and wrapper fixtures.

**Status:** proposed to ops:0; policy approved in delivery 84c76ec1-c90f-40f0-885d-bfeafba9b898. Detailed mechanism awaits approval before change-code. Single atomic close boundary; no Mx tags.

## Scope and alternatives

The authority is the issue's deny-by-default human-attention rule. This covers the bell AttentionList and automatic desktop projections of the same events. Preserve raw child terminal BEL/actor-chip AttentionLedger, blocked chips, operation status and feed copying: they are separate UI facts, not the bell's list. Existing explicit hub --notify validation remains mandatory.

1. **Chosen: typed pure table plus existing delivery seams.** One policy owner, auditable producer enumeration, no English text classification.
2. Filter generic Warn calls by Kind/text. Rejected: status/warning collapse unrelated classes; prompt/question and archive/stall bypass Warn.
3. Move everything through the lifecycle publisher. Rejected: its lifecycle projection, automation generations and restart reconciliation are a different state model. Reuse its broker/permit and receipt mechanics, not its lifecycle state.

No change to admission guards, composer recognition, hub automation or lifecycle retry policy. A connected automation-OFF hub means feed only, no replay and no bell fallback (TL approved). No connected *owning* hub means bell fallback for actionable events even if a different hub is live. Routine feedback stays feed-only. Global process diagnostics use primary hub, matching current archive ownership; source-specific events retain their own hub identity. Do not infer ownership from focused slot or printable label.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| NoticeClass and routing table | cmd/internal/couchcore/notice_route.go | new |
| NoticeRoute input/output | cmd/internal/couchcore/notice_route.go | new |
| NoticeDeliveryState/transition | cmd/internal/couchcore/notice_delivery.go | new |
| Notice | cmd/internal/couchtty/notice.go | modified |

- **NoticeClass** identifies the emitting producer, independently of feed collapse Kind. The table contains every actionable class below; zero/unknown never directly grants bell eligibility. Class metadata describes routine, operator-owned or hub-routed, plus explicit prior-crash/hub-start exceptions. One place widens policy later (ARCH-DRY). Tests enumerate the full table and unknown values.
- **NoticeRoute** consumes class, operation origin, owner readiness and proven existing delivery; returns feed/bell/hub effects. The owner state is booting, connected-enabled, connected-disabled or unavailable. No IO or clocks in the pure decision. All events enter feed once. Explicit hub notify passes its current authenticated admission before projection; a caller cannot self-select that class over a request protocol.
- **NoticeDeliveryState** owns only pending warning routing, not message history. Inputs are occurrence, service-ready, owner snapshot, send-start/result, receipt observation, expiry and shutdown. Outputs request existing broker effects or local projection. One occurrence may have one canonical admission ID and never resend after uncertain admission.
- **Notice** gains typed class and source ownership metadata (exact Thread, known owner root or explicit global scope), plus proven existing hub-effect disposition when applicable. Thread remains the open target, not an ownership alias. Existing feed content/copy bounds remain unchanged; routing body is bounded to 4 KiB.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Console warning projection | cmd/internal/couchtty/console.go and console_notice_route.go | modified/new | feed, attention list, renderer |
| Notice delivery owner | cmd/internal/couchcmd/notice_delivery.go | new | cached identity, broker, receipt snapshots |
| Condition routing | cmd/internal/couchcmd/attention.go, wait_notice.go, message_service.go | modified | picker/question observations |
| Explicit and archive notify | cmd/internal/couchcmd/notify_service.go, archive_notify.go | modified | authenticated notify, existing desktop sink |

- Console publication commits feed plus a bounded routing fact under its mutex, with no callback IO, broker call or repaint under locks. Project eligible local bell events once using existing IDs/bounds. Route decisions for hub-dependent events are returned to Console after unlocking; never publish feed again on completion. Convert the two direct locked warning sites (cast code and continuation scan), not just Warn.
- Start the delivery owner with the Console lifetime before startup warnings. It collects boot events, then binds the service after authority/registry initial reconciliation. A nil service while booting is not unavailable. Explicit startup failure seals readiness as unavailable; shutdown drops pending work and joins workers.
- Resolve exact source owner from cached verified workspace identity and authority.owner/hubBindingAt. Capture source ownership before pane/metadata removal; use root identity for detached targets. Revalidate exact hub binding and automation epoch at admission, respecting holdHubEffect ordering and sendAutomaticSystem as the sole generated-input ingress. Global process warnings use primary, never an arbitrary live hub.
- Use stateful messageWorld/newServiceRig and hubMessageRig fixtures (real local registry/broker sockets and controlled endpoints), lifecycleManualPublisher ordering helpers, and fake Console hosts. No live fleet mutation is required for tests.

## One producer routing table

`B` = bell + feed, `H` = enabled owning hub + feed, fallback bell only for unavailable owner; disabled owner is feed-only. `F` = feed-only. Existing canonical hub delivery suppresses a second send without suppressing fallback on a proven unavailable owner. Desktop is retained for explicit authenticated notify and already-supported operator-owned events only; automatic H classes lose unconditional desktop output.

| Typed class (one table row per class) | Current producer anchor | Route |
|---|---|---|
| HubRequested | couchcmd/notify_service.go handleNotify | B after authenticated admission |
| CastStartFailed | couchtty/console_broadcast.go startBroadcast | B, operator origin |
| CastEndedUnexpectedly | console_broadcast.go broadcastEnded | B, operator origin |
| ChromeStartFailed | console_webcast.go startWebcast | B, operator origin |
| ChromeEndedUnexpectedly | console_webcast.go webcastEnded | B, operator origin |
| ChromeRecovery | console_webcast.go webcastUpdated | B, operator origin |
| CastDiagnostic | console_webcast.go SetWebcast Warnf | B, operator origin |
| CastCodeLockout | console_castcodes.go codeRotated | B, current generation only |
| OperatorCapabilityFailed | cast_control.go onWebCapabilityClick | B; expected races remain F |
| OperatorDriveFailed | console_drive.go driveResult | B; expected races remain F |
| ChildExited | console.go onExit | H; expected park remains F |
| ExitCleanupFailed | console.go onExit forget | H |
| InputTraceFailed | console.go SetInputTrace | H |
| MouseTraceFailed | console.go SetMouseTrace | H |
| EventTraceFailed | trace.go SetEventTrace | H |
| ContinuationScannerFailed | console_continuation.go acceptContinuationRequests | H |
| ContinuationMissed | console_continuation.go finishContinuationOperation | H |
| ContinuationFailed | console_continuation.go finishContinuationOperation | H |
| LifecycleAttemptDiagnostic | console_lifecycle_failure.go finishLifecycleAttempt | H |
| LifecycleResultWarning | console.go finishOperation | H |
| RemoteLifecycleDiagnostic | couchcmd/lifecyclefailure_queue.go | H |
| DispatcherDiagnostic | couchcmd/run.go wireConsoleOperations | H |
| FailedAttachCleanup | run.go dispatcher LiveOwner | H |
| StartupConfigurationWarning | run.go beginConsole | H, buffer until ready |
| HubStartFailed | couchcmd/hubs.go startExplicitHubs | B, failed owner cannot act |
| HubClaimConflict | couchcmd/hub_move.go | H |
| AgentPriorityWarning | couchcmd/renicer.go | H |
| ScratchMetadataWarning | couchcmd/scratch.go | H |
| SupervisionEscalation | couchcmd/supervision.go tick | H |
| StartupResendUnavailable | supervision.go reconcile/tick | H |
| WorkerCueUnavailable | worker_cue_coordinator.go | H |
| AutomaticInputStall | automatic_input_stall.go | H, reuse reserved occurrence |
| LifecyclePublisherDiagnostic | lifecycle_notices.go | H |
| ArchiveInventoryWarning | archive_schedule.go warnOnce/archive_notify.go | H |
| ArchiveMaintenanceWarning | archive_schedule.go tick/diagnostic | H |
| PriorCouchCrash | run.go installCrashReport | B, Couch itself crashed |
| CrashCaptureSetupFailed | run.go installCrashReport | H |
| LivePicker | message_service.go publishBlocked | H, condition keyed by exact thread |
| JudgedQuestion | attention.go publishQuestions/wait_notice.go tick | H, reuse existing wait delivery |
| Routine | Notify/setNotice/progress, expected stops/races, borrowed attach, freshness skip | F, even without hub |
| Unknown | unclassified future warning | H, never direct B |

CastDiagnostic must enumerate Warnf's current callers: remote-driving setup/fidelity failures, actionable HTTP failures (503 remains informational), recorder open/write/close/snapshot/capability failures, and board diagnostics. Current casts are UI initiated; remote hub agent-drive requests retain caller origin and are H, not operator B. Origin is captured at operation admission, never guessed from eventual completion text.

## State, sequencing and operating envelope

ARCH-PURE / ARCH-STATE: keep legal transitions in the pure delivery reducer. No new durable queue, log or replay store.

| State/event | Decision and effect |
|---|---|
| booting + H occurrence | Feed once; retain bounded event until readiness |
| any + Routine | Feed once; no retained routing obligation |
| ready + owner enabled + new H | Freeze exact owner/epoch and occurrence ID; request one system admission |
| ready + owner disabled | Feed only, retire occurrence; off/on never replays it |
| ready + owner unavailable | One bell item for actionable event; no arbitrary-hub reroute |
| pending + owner moved/replaced before admission | Re-resolve only before effect starts; stale exact-binding/epoch guard refuses admission |
| admitted/result uncertain | Retain canonical ID/receipt status; inspect memory receipt without resubmitting |
| refused/not admitted | Feed diagnostic, no automatic bell while connected owner exists; retire bounded attempt |
| late/duplicate result | Match token/generation/ID; ignore superseded occurrence, never duplicate feed or bell |
| expiry or shutdown | Cancel unstarted/local waits; no replay; join owned workers; broker remains owner of any already-admitted message |

Proposed load budget (design assumption for rare asynchronous diagnostics, subject to TL approval): at most 128 pending occurrences, 4 KiB copied routing body each, four concurrent admission attempts and a two-second admission context; coalesce repeated pending class/source condition updates. Four consumers bound worst queued admission waiting at roughly 64 s for a full queue of deadline-limited attempts; ordinary bell-eligible local events bypass hub IO. Pending events expire at two minutes, chosen consistently with existing ordinary lifecycle wait. The implementation must measure/tests count at N and 2N and stop if actual broker semantics invalidate this envelope. No process/Git probes or per-event goroutine growth.

At capacity, retain feed and increment one bounded, aggregated delivery-loss diagnostic; one bell item per dispatcher lifetime identifies that the hub cannot receive the omitted diagnostics, with a count updated on the same item. This is an explicit class-3 fallback requiring TL approval, not a new per-slot warning storm. No recursive routing of router errors. Expiry/refusal while hub is still live stays feed-only. Reconcile uncertain receipts in memory at a bounded tick and retire local tracking by the same expiry; never convert an unknown effect into a retry.

Condition reconciliation must include ownership/availability changes even when picker/question set is unchanged. Preserve blocked chips. If no hub exists, wait classification must still create its question episode (currently it continues before that); route the condition through the same table and clear it when resolved. Existing wait/supervision sends remain the single hub send per episode. A live hub picker cannot send a notice to itself in a way that depends on clearing that picker: treat hub-can-act from actual recipient capability; bring ambiguity to TL rather than bypassing input guards.

ARCH-CONSTRAINTS: asynchronous path, no UI blocking; bounded memory/parallelism above, no network calls added outside existing broker. ARCH-SECURE: typed local classes are internal, not caller-controlled protocol assertions; identities must come from verified cache and be checked again at effect. Automation OFF never grants fallback eligibility. Preserve body sanitization and thread-safe attention bounds.

ARCH-FUNERAL: the lifetime owner removes pending/tracking records on terminal result/expiry/shutdown; state dies with Console, no restart replay. Existing AttentionList capacity 50 and 12-hour age remain, and feed stays bounded. ARCH-EVOLVE: no wire or durable record format changes planned; added fields are ephemeral Go values. If implementation needs persisted/protocol fields, stop and revise before widening schema. ARCH-MOCK: reuse persisted fixture workspaces and real local broker/registry with stateful endpoints. ARCH-PURPOSE: qualify all producers and bypasses in this table, not only the previously edited Warn calls.

## Executable test contract

The following names are the production API to implement; unit tests call these functions, not a second model. Scenario details belong in executable fixtures rather than a prose case inventory.

| Production function | Adversarial input and mechanical guard |
|---|---|
| couchcore.RouteNotice(NoticeRouteInput) NoticeRoute | Exhaust the typed class/origin/owner domains and fuzz unknown enum values; assert independently specified human-eligibility and feed-once invariants. |
| couchcore.AdvanceNoticeDelivery(NoticeDeliveryState, NoticeDeliveryEvent) (NoticeDeliveryState, []NoticeDeliveryEffect) | Generate bounded event sequences with duplicate/reordered completions and ownership epochs; assert at-most-one admission ID per occurrence, no resend on uncertainty, disabled-hub non-replay and resource caps after every transition. |
| couchcmd.resolveNoticeOwner(event) noticeOwner | Stateful multi-hub authority fixtures vary source lifetime and ownership while controlled barriers pause admission; assert exact-owner provenance and unchanged binding/epoch guards, never focused-slot or alternate-hub substitution. |
| couchcmd.runNoticeDelivery(ctx) | Controlled stateful broker endpoints release or fail effects in arbitrary order; count active admissions, queue bytes, IDs and retained records against independent bounds at N and 2N, and prove cancellation joins all local workers. |
| couchtty.publishWarningLocked(Notice) bool | Race-enabled fake Console host drives concurrent publication/close and locked callers; assert feed-once, memory-only handoff and existing lock order with delayed external effects. |
| couchtty.applyNoticeRoute(result) | Fake host/real presenter drives route completions after source replacement; assert exact target identity, bounded attention projection and no feed republish or post-shutdown paint. |
| couchcmd.publishBlocked and waitWatcher.publishQuestions | Statefully vary readiness/ownership independently of a fixed condition set; assert blocked-view independence, condition reconciliation and one canonical hub effect per episode. |
| couchcmd.handleNotify, notifyArchiveWarning and handleAutomaticInputStall | Existing authenticated service fixtures observe all effect sinks; assert each class uses the table and at most one existing canonical hub delivery, without broadening caller privileges or unconditional desktop output. |
| Each typed producer in the routing table | Production-path fixture plus source inventory contract proves the producer emits its declared class/origin/source; a newly introduced unclassified actionable producer fails the inventory. |

## Implementation tasks (after detailed approval and change-code)

### Task 1: pure policy and delivery state

**Files:** create cmd/internal/couchcore/notice_route.go, notice_route_test.go, notice_delivery.go, notice_delivery_test.go; modify cmd/internal/couchtty/notice.go.

- [x] Write failing tests for RouteNotice and AdvanceNoticeDelivery using the executable test contract.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestNotice(Route|Delivery)' -count=1` and verify the missing-feature failure.
- [x] Implement the typed table and pure transition API; run the same tests green, then commit with issue and agent trailers.

### Task 2: Console and service integration

**Files:** create cmd/internal/couchtty/console_notice_route.go and tests; create cmd/internal/couchcmd/notice_delivery.go and tests; modify couchtty/console.go, couchcmd/run.go, message_service.go and automation.go only if the existing ingress seam needs exposure.

- [x] Write failing tests for resolveNoticeOwner, runNoticeDelivery, publishWarningLocked and applyNoticeRoute using stateful fixtures and controlled effect barriers.
- [x] Implement their IO shell and lifecycle integration according to the bounds and state table; preserve the existing guarded broker ingress.
- [x] Run `go test -race ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestNotice(Route|Delivery)|TestConsoleNotice' -count=3`; green then commit.

### Task 3: producer and bypass integration

**Files:** producer anchors in the routing table, couchcmd/attention.go, wait_notice.go, notify_service.go, archive_notify.go, automatic_input_stall.go; associated tests.

- [x] Add failing producer-inventory and production-function tests from the executable test contract before rewiring their effects.
- [x] Wire the typed sources and existing canonical delivery provenance into the one routing policy, keeping source identity through callbacks.
- [x] Run `go test -race ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test.*(Notice|Attention|Warning|Picker|Question|Notify|AutomaticInputStall|Startup)' -count=1`; classify unrelated baseline failures with exact evidence.
- [x] Audit each changed reader, refresh only its named certificate via the inventory's exact repair command, run `make test-affected-inventory`, then commit the integrated sweep.

### Task 4: documentation and close

**Files:** relevant atlas/couch attention documentation and atlas/index.md; this issue/plan.

- [x] Document policy-table anchor, operator eligibility, ownership, automation-off and bounded delivery semantics at map level.
- [x] Run `git diff --check`, targeted verification above and inventory; push final code head and start `sdlc local-check run --pre-close`.
- [x] Finish the pre-close reader inventory and focused verification evidence in the issue Log.

Workflow postconditions: await the recorded pass on final code and TL-coordinated live acceptance before landing; use measured actuals and the binary-owned close review; address its findings through the ledger, then land through sdlc PR/merge and report verified PR/archive/runner completion to hub. Stage only if TL requests a live check.

## Approval request / remaining decisions

Approve the typed table, four-worker/128-entry dispatcher and the single aggregated capacity-loss fallback. Approve primary ownership for truly global diagnostics (preserves current archive contract). The live-hub-picker case needs an explicit interpretation: connected owner but blocked composer cannot safely consume its own prompt; keep feed and chip only under the approved connected-hub rule, rather than expanding class 3 to busy hubs. Other disabled/busy admission failures remain feed-only and do not weaken guards. No implementation starts until ops:0 answers the detailed plan.


## Revisions

- 2026-10-24: ops:0 approved the entire design in authenticated delivery 12d7cfb1-b715-4546-b159-69f6c8cde4bb, including all 39 classes, dispatch bounds, global primary ownership, aggregated capacity-loss bell and connected-hub picker rule. This supersedes the proposed/awaiting-approval status and approval-request wording above. Implementation is authorized after the change-code gate. Use the full flow because the planned reducer, dispatcher and producer sweep exceed the 100-added-code-line quick shell.

- 2026-10-24: PQ-1 test-contract refinement names production functions and one adversarial strategy/mechanical guard per risky function; compressed procedural scenario inventories. Generated event sequences exercise the production reducer with independent effect-count, epoch and resource invariants, and controlled shell completion ordering. Approved behavior and bounds unchanged.

- 2026-10-24: implementation retained the existing wait-watcher ownership for picker/question messages, including timing/mute/deduplication; HubHandled describes canonical effect ownership, not a submitted receipt. No immediate duplicate picker send was added. Publication freezes memory-only readiness/epoch before asynchronous work, tested against OFF/ON ordering.

- 2026-10-24: moved future close/landing/report actions out of implementation checkboxes; those are workflow postconditions, not deliverables that can truthfully be ticked before the close gate itself (lesson #871).

- 2026-10-24: separated recorded-check start (completed implementation task) from awaiting its passing result (mandatory pre-landing postcondition), following tl-worker's explicit allowance for close review to overlap a queued check. Ops:0 assigned the live look to pair:4's combined stage after review and green suite; no unilateral pair:0 stage.
