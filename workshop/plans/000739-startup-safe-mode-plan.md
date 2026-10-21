---
collaboration-mode: discussion
---

# Couch startup safe mode implementation plan

> **For agentic workers:** Consult AGENTS.md §3 for execution strategy. Use the approved plan and SDLC gates; do not implement until the operator approves this startup policy through the hub.

**Goal:** Reconnect surviving terminals without starting agents or recovery, keep startup out of hub composers, and orient each newly started worker once.

**Architecture:** Startup observes, classifies and attaches only positively listed terminals. A memory-only per-hub gate discards automatic system messages until genuine operator submission; ordinary peer reports stay deliverable. Explicit lifecycle actions retain their existing admission. The wrapper owns the generic notice's one-shot memory, and the current broker carries its delivery; the separate Couch-restart resend prompt keeps its existing lifetime.

**Tech stack:** Go, existing Couch core/Console, negotiated wrapper session protocol, existing peer delivery reducer and stateful process/socket fakes.

## Operator review: choices and scope

Source: ops `teams/devinfra/projects/fleet-drain/design.md`, especially the operator's revised point 10. This plan is for review, not permission to code.

| Choice | Proposed behavior | Alternative and cost |
|---|---|---|
| Missing hub at startup | Open the switcher; list retained/missing hubs and slots. Never cold-resume or create an agent, even on an empty installation. | Keep automatic hub creation/resume: convenient, but violates operator-owned recovery. |
| Surviving sessions | Attach warm hubs and workers with existing exact-session guards and worker limits. Never turn a failed warm attach into cold/Fresh. | Disable all attachment: quieter, but needlessly abandons running terminals and changes the approved warm-worker behavior. |
| Restart report | One retained Console status line, readable/copyable in the notice list, never a broker message into a TL composer. | Inject a one-line message: wakes the TL automatically and contradicts safe mode. |
| Hub startup gate | All automatic system-message producers share a per-hub gate, initially quiet. A genuine operator submission opens that hub; suppressed facts never replay. | Suppress only startup lifecycle/baseline facts: smaller, but inherited wait/reminder/resource conditions can still wake the TL. Hub chose the shared gate; operator reviews it. |
| Generic worker cue | Once per wrapper incarnation, with memory owned by that wrapper, including across Couch-only restarts. Fresh/resume/relaunch starts get a cue; reconnecting the same wrapper does not. | Once per incarnation per Couch lifetime: simpler coordinator map, but a surviving agent gets a false "you restarted" cue after every Couch restart. Hub chose the wrapper-owned option; operator may overrule it. |
| Couch-restart resend prompt | Preserve the existing once-per-worker-address-per-Couch-lifetime prompt about an unanswered relevant report. It is distinct from the agent-start cue. | Replace it with the generic cue: loses the restart's lost-report recovery hint. |

Generic cue text: **"You restarted; reread your issue Log and continue where you left off; report to your hub."** The TL may add context in an ordinary separate message. The existing `startupResendPrompt` text remains unchanged.

Covered: implicit `couch`/directory/multi-hub startup, primary and numbered hubs, warm worker attachment, all verified hub identities, wrapper reconnect/replacement, startup summaries and worker cue provenance. Existing explicit operator/TL-authorized start, resume, Fresh, relaunch, park and recover operations remain available; safe mode is startup policy, not a new fleet lock or persisted drain state.

Out: automatic recovery, a recovery script, drain/queue redesign, native-agent selection, durable notice storage, resurrecting old reports or changing wait/stale/resource thresholds and payloads. Their automatic hub delivery does go through the gate. Existing machine-wide passive status, desktop alarm and archive maintenance paths do not become agent input or recovery authority. The hub owns the project and coordinated acceptance. No pair:0 activity is authorized by this plan.

## Current source evidence

- `couchcore/startup.go:StartInteractive` attaches a listed hub, otherwise selects proven cold resume or Fresh. The managed-slot branch can also cold-resume/Fresh. Both implicit fallback classes must go.
- `couchcmd/run.go:beginConsole` currently arms the warm pass only after an initial child exists. Safe startup must also run a switcher and arm its warm-only pass when the hub has no handle.
- `couchcmd/hubs.go:startExplicitHubs/queueExplicitHubs` queues extra hubs through `start-hub`; `operationdispatch.go` currently delegates that to `StartInteractive`. Every configured hub must use the same attachment-only startup policy. Ownership must be installed before pass seeding.
- `couchtty/menu_reattach.go` already skips configured hubs and executes worker `resume` with `warm-only=true`; reuse its scheduler and per-slot reservation through adoption.
- `couchtty/console_lifecycle.go` brackets the warm pass and emits automatic recovery completions plus initial `needs-relaunch` observations. `couchcmd/lifecycle_notices.go` currently turns these into a hub summary. Drop only startup-origin facts, not explicit actions or subsequent independently observed events.
- `couchcmd/supervision_model.go` currently owns the resend prompt's per-address attempted set. Keep it for that prompt; it cannot own a cross-supervisor worker-start bit.
- `couchmessage/session_protocol.go` and `session_transport.go` already negotiate hello-v4→v3→v2→v1 by redial after a definite no-ack. Extend that seam; never relax strict decoders or add new fields to old-version frames.
- `wrapcmd/peer_delivery.go` owns the automatic input writer, retained receipts and confirmed Submitted outcome. This is the one authority that knows whether the cue actually reached the native agent.
- Filtering lifecycle alone is insufficient: `waitWatcher` inherits pre-start quiet time; `sessionStaleWatcher` may remind a hub about its own document; resource sampling starts immediately; a wrapper stall can arrive at any time. There is no current startup gate. Broker submission counters advance on accepted observations, including initial reconciliation, so that first observation must establish a baseline rather than open the gate.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `StartupObservation` / restart-kind projection | `cmd/internal/couchcore/startup_policy.go` | new |
| startup attachment selection | `cmd/internal/couchcore/startup.go` | modified |
| `HubStartupGate` | `cmd/internal/couchcmd/startup_gate.go` | new |
| worker start-notice state and canonical identity/text | `cmd/internal/couchmessage/worker_start_notice.go` | new |
| startup provenance of lifecycle observations | `cmd/internal/couchtty/console_lifecycle.go` | modified |

`StartupObservation` is a point-in-time value: restart kind (surviving fleet terminal / none / unknown), verified parked count and uncertainty. Use existing `SessionObservation` and parsed thread records; do not infer missing sessions from zero values or unreadable records. Any positively attributed present terminal proves a normal restart; no present terminal plus unresolved/orphaned/incomplete evidence is unknown, not full shutdown. Fully observed absence gives the full-shutdown projection. An empty known fleet gives no sessions/zero parked, not permission to Fresh.

`HubStartupGate` has quiet and active states, 1:1 with each configured hub in this supervisor lifetime. A verified initial wrapper observation establishes its submission baseline; it is never a release event. Only a subsequently observed genuine operator submission by the current verified hub wrapper opens the gate. Worker submissions, automated paste, peer/system messages, inherited initial counters, stale old-session frames and reconnect do not. If no hub wrapper is connected, it stays quiet. Active is monotonic for that configured hub until Couch exits; this is not a recovery permission and does not auto-start anything. Pure transitions are owned under the existing service/session serialization; a common generated-message entry point checks the exact target's current hub identity.

The worker cue state belongs 1:1 to `peerDelivery`'s wrapper incarnation. Model needed → committed → delivered or unconfirmed; a retained one-shot receipt witnesses the monotonic "already delivered" bit and prevents an accepted/uncertain attempt becoming needed again. This is constant memory, not another history. The wrapper reports its current state on connect and on changes; normal reconnect does not reset it. A new wrapper begins needed. Use a shared deterministic message identity from the exact binding **and existing launch ordinal**, and a shared body definition, separate from the random-ID resend prompt. `wrap.go:freshAgentInvocation` already prepares a new launch ordinal before re-exec: PID/birth and even some binding fields can survive exec, so binding equality alone is not the start identity. Carry that ordinal in the negotiated cue report and validate it against the existing recorded-child authority; unsupported/missing evidence is unknown, not a made-up start. Only a server-origin cue can change this state; an ordinary peer copying the prose cannot.

Do not change the existing thread lifecycle vocabulary or persist a startup/recovery permission bit. The observation never authorizes later effects; selected warm attachment still revalidates the exact terminal through existing guards.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| read-only startup collection | `couchcore/startup.go`, existing `SessionPresenceResolver` | modified | Preview records + bulk bound-session presence |
| safe Console startup | `couchcmd/run.go`, `hubs.go`, `couchtty/console_reattach.go` | modified | initial/extra warm attachment, switcher and retained status |
| generated-message policy | new `couchcmd/startup_gate.go`; `message_service.go`, `supervision.go`, `lifecycle_notices.go` | new/modified | verified target ownership, operator submissions, existing Broker.SendSystem |
| worker session reporting | `couchmessage/session_protocol.go`, `session_transport.go`, `registry.go`; `couchcmd/message_service.go` | modified | negotiated hello-v5 plus session updates |
| cue admission | `couchcmd/supervision.go` and a focused worker-cue adapter | modified | verified hub ownership + exact current worker + normal Broker.SendSystem |
| wrapper cue witness | `wrapcmd/peer_delivery.go`, `peer_runtime.go` | modified | existing queue/input/confirmation and session client |
| lifecycle startup filtering | `couchcmd/lifecycle_notices.go`, `couchtty/console_lifecycle.go` | modified | existing arm/drain and source provenance |

Fakes: reuse `FakeThreadArtifactCollisionChecker` (present/absent/orphan/unknown and mutation hooks), `FakeRunner`, private ThreadStore/SlotCatalogFake, `hubStartupRig`, fake host/terminal, real private Unix session sockets and existing stateful peer-delivery harness. No operator HOME/session/process is a fixture. The wrapper's native process lifetime, not a supervisor map, is the resource that enforces no second cue.

## State and failure projection

Actions: **inspect Slots**, **operator chooses recovery**, **operator/TL explicitly supplies orientation**. Attaching a surviving terminal is presentation of existing work; it is not permission to restart an agent.

| Internal state / failure | Visible state | One recovery action | Possible loss |
|---|---|---|---|
| One or more verified attachable fleet sessions; other slots may be absent | `Couch restart: surviving sessions attached; N slots parked; recovery waits for operator` | inspect Slots | Any unrecovered process work was already absent; Couch creates nothing to replace it |
| Fully observed absence of attachable fleet sessions | `Full shutdown detected: N slots parked; recovery waits for operator` | operator chooses recovery | Native processes are absent; retained records/transcripts remain |
| Listing timeout/no reply, unreadable identity, orphan or ambiguous ownership without a proved survivor | `Startup state unknown; no recovery started; inspect Slots` | inspect Slots | Observation incomplete; do not claim absence or kill survivors |
| Warm proof disappears / adoption fails | Existing failed attachment row with its reason; no cold fallback | inspect Slots | Terminal may still survive; no new agent or replacement |
| Hub quiet; automatic notice generated | One startup status says recovery waits for operator; no injected message and no backlog | operator submits the next instruction to that hub | Suppressed automatic notices are intentionally lost; current state remains queryable |
| Genuine later operator submission opens this hub | Existing hub conversation; future automatic notices follow normal rules | no recovery needed | Prior suppressed notices stay discarded |
| Generic cue needed, owner hub not connected yet | Worker remains unprompted; cue waits for verified owner | operator/TL explicitly supplies orientation if needed | Orientation delayed, never sent as the worker's own authority |
| Cue accepted and held behind composer/picker | Existing queued/held message status | inspect Slots | None yet; no forced paste |
| Cue Submitted | No extra persistent UI state; wrapper remembers delivered | no recovery needed | None |
| Cue refused, incomplete or uncertain | One Console notice identifying slot and uncertainty, without re-injection | operator/TL explicitly supplies orientation | Generic orientation may not have arrived; never duplicate on uncertainty |
| Old wrapper has no cue-state capability | One status naming the compatibility limit; ordinary messaging still works | operator/TL explicitly supplies orientation | Cannot prove whether an old worker has received a cue; do not guess |
| Shutdown during collection/attachment/send | Existing cancellation/partial-result receipts; no restart replay store | inspect Slots on next startup | Only ephemeral uncommitted attempts; preserve acknowledged input |

The line describes the startup observation, not proof that the host powered off. Parked counts count verified parked slots only; an unavailable count is displayed as unknown, never zero. Multiple hubs share the startup observation and retain their configured ownership even when their terminals are absent.

### Ordering and compatibility

1. Resolve configured hub ownership, collect a bounded read-only startup observation, then perform only positively proved warm attachments. Do not enroll/provision a missing hub or parse native resume targets merely to classify startup.
2. Install ownership before initial menu inventory and arm the warm pass even without an attached initial hub. Extra-hub requests remain on the existing lifecycle lane; no new executor. A warm refusal is final for that request.
3. Publish one startup status through Console, never `SendSystem` to a hub. Install the per-hub gate before starting producers. Route every automatic system send through one policy entry; normal worker peer reports bypass it. A suppressed outcome is typed and terminal for that occurrence, not busy/unavailable and not a successful delivery. Each producer retires its suppressed occurrence without buffering or retrying it when the gate opens. Retain queryable current state. No timer releases the gate.
4. Baseline initial inventory and background warm completions must not later flush as a hub summary. Retire startup-origin facts at observation, even while a hub is absent. Explicit lifecycle facts observed while quiet are suppressed/retired too; after operator submission only new facts may publish. Test opening the gate while startup work is held and after drain; neither releases a startup backlog. Genuine later operator actions/native exits retain their existing independent notice semantics.
5. Session hello-v5 carries cue state; updates stay on the existing coalesced activity channel. Strictly validate shape and known states. New wrapper→old supervisor falls back through existing hello versions, omitting cue fields on older frames. Old wrapper→new supervisor remains connected with unknown cue state, never treated as needed. No change to public CLI/resource JSON is required.
6. Current verified ownership and exact recipient incarnation gate the cue. Do not require the hub composer to be idle or its startup gate open, only its current connected authority: the target is a worker. No self-sent worker cue when the hub is missing. Recheck current incarnation at admission and let existing broker/receiver guards reject replacement.
7. Atomically claim the one-shot cue at wrapper commit, before any PTY write. Report committed while held; set delivered only at the existing post-submit confirmation. A supervisor restart between commit and confirmation must see committed/delivered/unconfirmed, never needed. Preserve the fixed witness past normal receipt-history expiry. No uncertain write is retried, and no accepted cue is replaced by another random ID.
8. Duplicate hello/activity, same-incarnation reconnect (including Couch replacement), hub replacement, late old-session frames and uncertain admission cannot create another generic cue. Fresh/resume/relaunch/re-exec yields a new launch incarnation and one new opportunity, even when PID/birth survives exec. The independent resend prompt still runs once per address per new Couch lifetime.

### Complete automatic producer sweep

All conversation-bound system messages use the shared target gate. Audit production `SendSystem` call sites and pin the inventory so a new direct call cannot bypass it.

| Producer | Current source | Quiet-gate behavior and required regression |
|---|---|---|
| Lifecycle summaries, later lifecycle/native-exit facts | `lifecycle_notices.go`; `console_lifecycle.go`, `native_exit.go` | Retire suppressed revisions without retry/diagnostic injection; initial warm/baseline facts never become later backlog. Test quiet, open-after-drain, and a new post-open explicit action. |
| Worker wait notice | `wait_notice.go:waitWatcher.tick` | Consume the suppressed wait episode without claiming it was delivered. Test inherited pre-start quiet time, gate opening, unchanged episode (still zero writes), then a new episode. |
| Session-file freshness reminder (including a hub's own file) | `session_stale.go` / `session_stale_model.go` | Consume suppressed reminder, keep document evidence queryable. Test immediate first check, gate opening/no replay, then a new reminder condition; worker-directed reminders remain independent. |
| Resource episode | `resources_alert.go:resourceAlertDispatcher.dispatch` / resource episode reducer | Terminal suppressed disposition for that effect, no pending retry. Test a pending startup episode, unchanged condition after opening, and a distinct new episode after opening. |
| Automatic-input stall tripwire | `automatic_input_stall.go` | Gate the hub message; retain the existing occurrence reservation and diagnostic/desktop evidence. Test arrival before hub submission, no replay on opening and one new occurrence afterward. |
| Unanswered-report supervision nudge | `supervision.go` / `supervision_model.go` | Gate hub-directed nudge and consume the suppressed report episode's notice effect. Test a worker report accepted while hub is quiet, no automatic nudge replay after opening, and a later new report episode. Ordinary worker report itself is delivered. |
| Generic worker cue and existing restart resend prompt | new cue adapter; `supervision.go` | Target workers only; test every configured hub is excluded and both distinct lifetimes work while the owner hub is quiet. |
| Archive freshness/Drive warnings | `archive_notify.go` / `archive_schedule.go` | Not a broker conversation producer: status/attention/desktop only. Do not turn it into agent input or auto-recovery. Pin the distinction; machine-wide backup work remains independent. |

The pending #737 scheduler must use this same entry point for any hub conversation input once integrated. Its branch is not a base for #739; coordinate after landing. A peer seam inquiry was refused without a receipt; no delivery or agreement is claimed.

No durable schema changes: no drain, marker, startup receipt file or automatic recovery intent. The session protocol is version-negotiated precisely because adding fields to strict old frames is unsafe (ARCH-EVOLVE). No native credentials are read.

## Operating envelope and invariants

- This is startup/session-connect work, not a UI tick or message-admission discovery loop. Reuse one bulk presence observation for the startup classification; do not ask every row for clients or native transcript ownership. Existing selected warm-attach revalidation retains its own calls and limits.
- Input size is the bounded existing fleet/store inventory. At N=20 and 2N=40, count one classification bulk call, one pass over records and zero cold-resume/Fresh/native-binding probes; no new process-table scans per row. Existing warm attachment work scales with attached terminals through the unchanged lifecycle worker limit/cap, not a new pool.
- Per tick/session update, cue decisions use memory. At 20/40 current workers, count at most one cue admission per needed incarnation, zero admissions after delivered/reconnect, zero hub-directed startup admissions. Simulate replacements and N/2N unchanged ticks; no per-tick filesystem scan or goroutine is added.
- Per-hub gate checks and release use cached verified identity and genuine submission evidence only; no native query on a generated send. The bounded hub table dies with the supervisor. Initial submission counts and a reconnect with the same count cannot open it. Connect/observation/submit ordering is exercised by held callbacks, not clock sleeps.
- Keep existing Zellij logical-read timeout, broker capacity and delivery/input deadlines; no new retry or deadline extension. A failed observer returns unknown with a usable switcher.
- One wrapper witness dies with the wrapper. Supervisor current-incarnation bookkeeping is bounded by existing registry capacity (128) and replaced/retired with its registry owner; no incarnation-history map that grows with relaunches. Existing notice and receipt bounds remain.

## Implementation tasks (one final close boundary)

### 1. Qualify safe startup selection and report

Files: `couchcore/startup.go`, new `startup_policy.go`/`startup_policy_test.go`, existing `startup_test.go`, `startup_observation_test.go`, `startup_proof_test.go`, `couchcmd/startup_observation_test.go`.

- [ ] Write reds through actual `StartInteractive`: absent primary hub, absent numbered hub, empty fleet and proven cold-resume evidence all produce zero agent starts and unchanged retained records. Known warm hub still attaches. Missing extra hub must not call Fresh/resume. Use fixture gates, not sleeps.
- [ ] Add direct projection cases: live+absent mixed fleet, all absent, exited rows, orphan, missing/unreadable record/index, duplicate attribution and listing no-reply/malformed. Unknown never becomes full shutdown.
- [ ] Implement the narrow observation/attachment-only path using existing read-only snapshot and session projections. Keep explicit start/resume/recovery implementations reachable only from explicit actions; remove implicit cold/Fresh fallback.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestStartup|TestOccupancy|TestScopeHolds' -count=1`; expect the new semantic reds before implementation and pass afterward. Commit source/tests with issue/model trailers.

### 2. Keep the hub silent and the Console usable

Files: `couchcmd/run.go`, `hubs.go`, `hubs_startup_test.go`, `startup_warm_pass_test.go`, `lifecycle_notices.go`/tests; `couchtty/console_reattach.go`, `console_lifecycle.go`/tests.

- [ ] Add real-Console/declared-dispatch reds for no-handle startup, multiple warm/absent hubs, one readable status line, unchanged focus and zero cold/Fresh workers. Exercise blocked attachment/cancellation and a vanished session after selection.
- [ ] Write a pure per-hub quiet→active transition suite: initial counter reconciliation does not open; current genuine operator submit does; peer/system/worker input, repeated counts, old session frames, reconnect and one hub's submission opening another are negatives. Hold admission while input arrives to qualify the actual initial-baseline boundary conservatively.
- [ ] Count zero hub-composer startup writes across every producer in the sweep, with lifecycle notices both on/off. Open the gate with pending observations and prove zero replay; only a new episode/fact after opening publishes. Hold the warm pass while an explicit action occurs after opening and prove that new action publishes once without flushing startup facts.
- [ ] Wire ownership before inventory, arm the pass with no initial handle, apply the shared warm-only hub policy and common message gate, and retire suppressed producer occurrences explicitly. Preserve existing shared queue/adoption reservations. The gate never blocks ordinary peer reports or worker-directed cues.
- [ ] Run all targeted `couchcmd` startup/lifecycle and `couchtty` reattach/lifecycle tests, including existing focus and cancellation cases; commit the coherent startup behavior.

### 3. Deliver the worker cue once across supervisor replacement

Files: new `couchmessage/worker_start_notice.go`/tests; `session_protocol.go`, `session_transport.go`, `registry.go` and protocol/negotiation tests; `wrapcmd/peer_delivery.go`, `peer_runtime.go` and focused cue tests; `couchcmd/supervision.go`/model/tests and focused adapter.

- [ ] Write the cross-lifetime red first: one wrapper delivers a cue, replace only the private supervisor/registry socket, expire ordinary recent receipts, reconnect the same wrapper and observe zero further cue writes. Then replace the wrapper and observe exactly one new cue.
- [ ] Qualify hello-v5 state validation, old/new fallback in both directions, duplicate/reordered session events and no new fields sent to old peers. All external-process fakes keep state across the modeled reconnects.
- [ ] Enumerate cue transitions using the existing delivery reducer: needed/commit/held/confirmed/uncertain/disconnect/cancel; gate commit and confirmation independently. A supervisor death between them, a busy composer, changed hub/worker incarnation, refused admission and lost acknowledgment must not duplicate native input. Add a same-PID/birth re-exec with a new launch ordinal and a late prior-ordinal update. Count actual writes, not only send calls.
- [ ] Add negatives: hub never receives the generic cue, missing hub cannot impersonate it, an ordinary peer copying cue text cannot set the bit, unsupported old wrapper remains unknown. Assert the separate restart resend prompt survives and is not counted as a new generic cue.
- [ ] Implement one constant-size wrapper witness plus negotiated reporting and current-owner admission, reusing normal broker/provenance/automatic input. No retry loop or new executor. Run focused protocol/session/supervision/peer tests with race detection and commit.

### 4. Document, verify, review and coordinate rollout

- [ ] Update README startup, lifecycle and supervision passages plus `atlas/couch.md`/index and shipped Couch skill/help text where the old behavior is described. Explain the per-hub gate and loss of suppressed notices, both worker notice lifetimes, no automatic cold/Fresh, unknown status, old-wrapper limits and operator-owned recovery. Add lessons only for an actual reviewed mistake class.
- [ ] Audit added/changed source readers and run `make test-affected-inventory` plus `git diff --check` before review. Run N/2N counted cases and targeted packages; push the final head and run `sdlc local-check run --name affected` (known #738 graph fallback may run the suite). Gate failures are evidence to classify, not permission to waive.
- [ ] Close once through SDLC's fresh review after required verification. No milestone tags for this atomic behavior change. Estimate only after plan-quality passes, not during this operator review.
- [ ] Coordinate real acceptance with the hub/operator: disposable owned fleet first, then a scheduled restart. Prove warm restart has no new agent and no startup hub input; full-shutdown fixture starts no agent and waits in switcher; operator explicitly opens/recoveries a hub and worker; worker cue once; supervisor-only reconnect no repeat; explicit worker relaunch one new cue. No staging, restart or real slot action without the hub's concrete authorization.
- [ ] On SHIP and the agreed acceptance evidence, `sdlc merge --yes`, verify Git/PR results, report to hub and stop. Hub owns project updates.

## Architecture rationale

- **ARCH-DRY/PURE:** reuse typed presence, warm attachment, broker/receiver and session negotiation. Selection, projection and cue transitions stay pure; one source defines cue identity/text.
- **ARCH-PURPOSE:** remove both primary and managed implicit launch fallbacks and startup hub-summary producers; do not merely replace one prompt string.
- **ARCH-MOCK:** private folder-backed stores, stateful native/terminal fakes and real private Unix sockets cover restarts; live qualification is coordinated, not performed on the operator while planning.
- **ARCH-CONSTRAINTS:** bound new work to startup/current sessions and counted linear passes; preserve existing native timeouts, worker capacity and overload handling.
- **ARCH-SECURE:** external records/listings remain strictly parsed, missing evidence remains unknown, exact verified hub/worker bindings gate system origin; no user prose is authority.
- **ARCH-ORDER:** the wrapper owns the one-shot commit witness through reconnect; delayed/uncertain outcomes never reset it. Startup provenance, not elapsed time, separates initial observation from later lifecycle changes.
- **ARCH-FUNERAL:** create no durable artifacts or new process. Fixed wrapper witness dies on exec/exit; registry-owned current state dies with its session/supervisor; no history growth per reconnect.
- **ARCH-EVOLVE:** no persisted-format change; new session shape is version-negotiated with strict known fields and old-version fallback. Do not change CLI JSON to carry internal cue bookkeeping.

## Approval and open decisions

The hub confirmed attachment-only startup, per-wrapper cue ownership, the independent Couch-restart resend prompt, and a per-hub quiet gate until genuine operator submission with no replay. **Operator approval is still pending for the whole plan**, including first-ever startup staying in the switcher, the non-injected report surface, suppressed-notice loss, protocol compatibility fallback and one-shot failure recovery. No production code or tests have been changed.
