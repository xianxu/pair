---
collaboration-mode: discussion
---

# Couch startup safe mode and hub automation implementation plan

> **For agentic workers:** Consult AGENTS.md §3 for execution strategy. The operator-approved contract below supersedes the earlier design in Revisions. SDLC plan-quality and estimate admission still precede code.

**Goal:** Reattach surviving terminals without starting agents/recovery, make hub automation explicit, and orient each newly started worker once.

**Architecture:** One read-only startup observation classifies the fleet and each hub. Only positively proved warm sessions attach. A per-hub memory owner controls Couch-generated input and scheduled runs, independently of human/worker communication. Wrapper-owned cue evidence survives supervisor replacement; the separate restart resend prompt keeps its current lifetime.

**Tech stack:** Go, current Couch/Console lifecycle workers, existing broker/automatic-input reducer, negotiated wrapper sessions and private stateful native/socket fakes.

## Approved contract and alternatives

Source: ops `teams/devinfra/projects/fleet-drain/design.md` and operator decisions relayed by ops:0 on 2026-10-20. The latest wording is **hub automation**, on/off, rather than start gate.

| Decision | Approved behavior | Alternative considered |
|---|---|---|
| Implicit startup | Attach positively surviving sessions only; absent/empty hubs remain in the switcher. Never cold-resume/Fresh or recover automatically. | Automatic hub launch/resume is convenient but defeats operator-owned recovery. |
| Partial fleet | Every surviving hub reattaches; missing hubs remain listed/parked and are named in the single startup line. Focus launch hub if it survived, otherwise switcher; extra-hub completions never steal focus. | Selecting another surviving hub automatically would move the operator without authority. |
| Initial automation | On only for a positively surviving hub on a normal restart. Off for full-shutdown/unknown startup and for every missing hub in a mixed fleet. | Always off was the initial proposal; operator chose continuity for surviving hubs. |
| Explicit control | `couch --automation off|on|status [hub]`; Ctrl+G in the switcher toggles the selected row's owning hub, naming it. Status never changes state; on/off are idempotent. | Automatic opening on operator submission was rejected: debugging must work while off. |
| Authority | CLI mutation is hub-only and own-hub-only. Optional hub must resolve to caller's canonical hub. Operator switcher may control any configured hub. | Cross-hub TL control was explicitly rejected. |
| First operator message | Couch does not interpret it. A separate tl-lead rule tells the TL to run automation on unless the operator says keep it off. | Message parsing or implicit flip inside Couch was rejected. |
| While off | Drop Couch-generated hub input, and skip scheduled due runs like missed runs. No backlog/replay/catch-up when on; schedule resumes at next due time. Running scripts may finish (soft stop). | Baseline-only suppression misses other producers; notice-only scheduler gating still runs unwanted work. |
| Ungated communication | Human input, worker/peer reports, startup/status reads and desktop output are always allowed. Worker-directed cues remain independent. | Muting all hub traffic would hide the evidence needed to debug. |
| Generic worker cue | Once per wrapper launch incarnation, including across supervisor-only restarts; reconnect adds none. | Per-Couch-lifetime memory would falsely tell surviving workers they restarted. |
| Restart resend prompt | Keep existing once-per-worker-address-per-Couch-lifetime request to resend only relevant unanswered reports. | Replacing it with the generic cue loses its separate lost-report purpose. |

Generic cue: **"You restarted; reread your issue Log and continue where you left off; report to your hub."** Only the TL supplies custom work. The startup line says **"hub automation off; say turn it on"** for affected hubs and names missing hubs. The TL-visible status exposes startup kind, missing hubs and each hub's automation state without composer injection.

Scope: implicit bare/directory/multi-hub startup, primary/numbered hubs, warm worker attachment, automatic producer admission, scheduled run admission and wrapper cue reporting. Existing explicit start/resume/recover/park APIs retain their safety guards. Exclude automatic recovery/scripts, drain redesign, native-agent selection, persisted automation/notice storage and the TL's own first-message interpretation. Hub owns project and real rollout; no pair:0 activity is authorized here.

## Starting source and integration ownership

- `couchcore/startup.go:StartInteractive` currently falls back from listed warm hubs to proven cold resume or Fresh, for primary and managed hubs. Replace only implicit startup selection, preserving explicit actions.
- `couchcmd/run.go:beginConsole` currently skips arming warm workers without an initial handle. `hubs.go:startExplicitHubs/queueExplicitHubs` uses the declared `start-hub` path for extras. Install ownership before pass seeding and run a usable no-handle switcher.
- `couchtty/menu_reattach.go` already skips configured hubs and uses bounded `resume --warm-only`; preserve its reservation through adoption and scheduler capacity.
- `couchtty/console_lifecycle.go` and `couchcmd/lifecycle_notices.go` produce initial observations and restart summaries. Retire startup-origin facts, including if automation turns on while the warm pass is still held; do not retain a later summary backlog.
- `couchmessage/session_protocol.go`/`session_transport.go` negotiate hello-v4→v3→v2→v1 after definite no-ack. New state must negotiate, not add fields to strict old frames.
- `wrapcmd/peer_delivery.go` owns actual native input/Submitted confirmation. `wrap.go:freshAgentInvocation` prepares a new launch ordinal before re-exec: PID/birth/binding can survive exec, so exact binding alone is not a new-start key.
- #737 owns the unlanded scheduler. Hub relays the agreed predicate contract to pair:1: check before each due script; off consumes/skips that due time; on admits next due only; already-running work finishes. Exit-10 notices use the same automation owner. No stacked branch; integrate main after #737 lands before wiring/qualifying that consumer. #739 cannot claim scheduler coverage complete before that integration.

## Core concepts

### Pure entities

| Name / risky functions | Lives in | Status |
|---|---|---|
| `StartupObservation`; `ClassifyStartup` | new `couchcore/startup_policy.go` | new |
| `HubAutomation`; `AdvanceHubAutomation`, `AutomationStatus` | new `couchcore/hub_automation.go` | new |
| `WorkerCueReport`; `AdvanceWorkerStartNotice`, `WorkerStartNoticeID`, `ValidateWorkerStartNotice` | new `couchmessage/worker_start_notice.go` | new |
| `LifecycleNoticeState`; `AdvanceLifecycleNotices` suppression/discard transitions | `couchcore/lifecycle_notice.go` | modified |
| `ResourceEpisode`; `AdvanceResourceEpisode` suppression and volatile original hub/epoch | `couchcore/resource_episode.go` | modified |

Startup values use existing parsed ThreadRecord and SessionObservation. A positively attributed live fleet session supports normal restart; complete absence supports full shutdown. Unknown/unreadable/incomplete/orphaned evidence without a proved survivor stays unknown. Per-hub positively surviving proof is needed to initialize that hub on; an absent hub stays off even when other workers survived. Parked counts include verified parked records only; unknown is not zero. These are current observations, not historical proof of host power-off, and never authorize a later attachment without its current exact-session guard.

`HubAutomation` is a bounded map over configured canonical hub roots. Initialization is a single startup transition. Only explicit authenticated on/off events change it afterward; status, input, submit, connect, restart summaries and time cannot. Capture automated occurrences under the same owner's serialized decision: an event suppressed while off stays retired even if later dispatch races with on. Off never revokes human/worker communication or cancels already-running scripts. There is no new recovery permission or fleet lock.

`WorkerStartNotice` belongs to one wrapper/launch ordinal, not Couch. Needed → committed → delivered/unconfirmed is monotonic for that launch. A constant-size sticky receipt witnesses actual confirmation, survives ordinary receipt expiration and is reported on reconnect; committed/uncertain never resets to needed. ID uses exact binding plus existing launch ordinal. A new launch gets one opportunity even across same-PID exec. Only canonical server-origin input can change the witness; copied peer prose cannot. Missing ordinal/capability remains unknown.

### IO seams

| Integration / functions | Lives in | Status | External/state owner |
|---|---|---|---|
| `ObserveStartupContext` | new `couchcore/startup_policy.go` | new | read-only Preview snapshot + bulk SessionPresenceResolver |
| implicit `StartInteractive` | `couchcore/startup.go` | modified | existing current-session guards and warm attachment |
| initial/extra hub adoption, `beginConsole` | `couchcmd/run.go`, `hubs.go`, Console reattach | modified | existing terminal/lifecycle workers, no new pool |
| `sendAutomaticSystem`, `automationAdmissionGuard`, automation CLI/UI controller | new `couchcmd/automation.go`, current generated-send consumers | new/modified | verified hub bindings + one memory owner + normal broker |
| `AdmissionGuard`; broker `tryAdmission`, `enqueueLocked` | `couchmessage/broker.go` | modified | broker then automation lock over first in-memory mailbox commit only; no endpoint IO |
| `runAutomationCLI`, `handleAutomation` | new `couchcmd/automation.go`, CLI/request dispatch | new | authenticated own-hub control; no process effect |
| lifecycle automation admission and suppression ownership, `lifecyclePublisher.observe/run` | `couchcmd/lifecycle_notices.go` | modified | existing cancellable publisher workers, owner epochs and broker; Console observation provenance unchanged |
| negotiated cue reporting | session protocol/transport/registry + `message_service.go` | modified | versioned private Unix session connection |
| cue confirmation/publication | `wrapcmd/peer_delivery.go`, `peer_runtime.go`; coordinator | modified | existing sole input writer and post-submit confirmation |
| scheduler due admission | #737 scheduler as landed | modified after main integration | injected automation predicate, existing due/missed-run policy |

Use current FakeThreadArtifactCollisionChecker/FakeRunner, private ThreadStore/SlotCatalogFake, hubStartupRig, fake host/terminal, peer-delivery harness and real private Unix sockets. Only live acceptance uses a coordinated disposable fleet/real restart. No fixture reads operator HOME, credentials or current sessions. Memory owners die with wrapper or supervisor; no new files/processes. Current registry-owned state is bounded at its existing 128 actors, never an unbounded incarnation history.

## Visible states and recovery

Actions: **inspect Slots/status**, **say turn it on**, **explicit automation on/off**, **operator chooses recovery**, **TL explicitly supplies orientation**.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Launch hub survives normal restart | Launch hub focused; automation on, missing hubs named | inspect Slots/status | Unavailable slots remain untouched |
| Full shutdown/unknown/missing hub | Switcher if launch hub absent; affected hubs say automation off; say turn it on | say turn it on | Missing processes not recreated; unknown stays unknown |
| Automation explicitly off | Hub can converse/debug; automatic input absent, due scripts skipped | explicit automation on | Suppressed notices and due occurrences intentionally lost, state queryable |
| Automation on | Future automatic input/due runs eligible | explicit automation off | Prior skipped work never returns |
| Warm attachment proof/adoption fails | Existing failed row/reason; no cold fallback | inspect Slots/status | Survivor may still exist; no replacement started |
| Worker cue held | Existing held/queued receipt | inspect Slots/status | Orientation delayed, never forced |
| Worker cue refused/uncertain/unsupported | Bounded per-slot status, no duplicate native input | TL explicitly supplies orientation | Generic cue may not arrive |
| Shutdown interrupts work | Existing partial/cancel/uncertain receipts; new supervisor re-observes | inspect Slots/status | Ephemeral uncommitted work only; acknowledged prefix retained |

The one startup status is retained/readable/copyable, not pasted into hub input. Status and desktop diagnostics are never automation-gated. Report emission does not create recovery work. Explicit on/off has no replay effect and no implicit focus change.

## Producer sweep and no replay

Every automatic conversation input goes through the shared policy, with a typed terminal suppressed disposition, never a fabricated success or retryable busy error. Each producer consumes its suppressed occurrence in its own reducer. Retire before any delayed effect can escape; retaining current queryable evidence is not retaining a pending notification. Pin all production SendSystem sites to this owner.

| Producer / risky function | Location | Strategy and mechanical guard |
|---|---|---|
| `lifecyclePublisher.observe/run` | `lifecycle_notices.go`; Console lifecycle/native exit | Stateful observation/admission ordering around off→on; suppressed revisions retire and startup facts never flush; count actual hub input |
| `waitWatcher.tick` | `wait_notice.go` | Inherited quiet episodes under controlled ticks; consume suppression once and require a fresh episode for later input |
| `sessionStaleWatcher.tick` / `advanceStaleReminder` | session_stale files | Stateful private document/owner transitions; suppressed reminder cannot remain a pending replay; worker-directed input independent |
| `resourceAlertDispatcher.dispatch` / `AdvanceResourceEpisode` | resource alert/reducer files | Episode event sequences across off/on; suppressed effect terminal without delivered claim or pending retry |
| `handleAutomaticInputStall` | `automatic_input_stall.go` | Sequence/provenance/reconnect adversaries; one reserved occurrence, gated hub input, ungated desktop/diagnostic |
| `supervisionCoordinator.tick` / supervision state | `supervision.go`/model | Controlled report episodes; ordinary worker report arrives while off, generated nudge does not replay |
| worker cue and restart resend prompt | cue adapter / supervision | Exact-role/lifetime adversaries; neither targets a hub; two distinct once rules remain independent |
| #737 due admission and exit-10 wake | landed scheduler seam | Fake clock/script receipts while off/on; skipped due runs never start/catch up; exit-10 respects automation |
| archive warning output | archive_notify/archive_schedule | Assert no broker agent-input call; existing status/attention/desktop and machine backup maintenance remain independent |

## Test strategy by risky function

| Functions | Adversarial class → mechanical guard |
|---|---|
| `ClassifyStartup` | Arbitrary typed presence/record populations → direct table/property tests; unknown never fabricates absence; candidates require positive exact-session evidence |
| `ObserveStartupContext`, `StartInteractive` | Mutating native/record evidence at selection/effect boundary → stateful fakes, unchanged retained records, zero cold/Fresh/native-target reads and counted bulk calls |
| `AdvanceHubAutomation`, `AutomationStatus` | Arbitrary explicit/irrelevant event sequences across independent hubs → pure transition properties, no input-driven change or implicit replay |
| `handleAutomation`, `runAutomationCLI` | Forged/current/stale caller and hub aliases → actual private socket/CLI and strict decoder, own-hub mutation only |
| `automationAdmissionGuard`, broker `tryAdmission`/`enqueueLocked` | Controlled policy changes during families/Observe/Reserve and both queue paths → original hub/epoch/exact Binding checked at commit under broker then automation; TryLock and receipt-publication assertions pin lock extent; prior canonical receipts remain unchanged |
| Console automation key/adoption | Incrementally delivered key encodings and completion reordering → fake host/input plus real reducer; outside-switcher bytes preserved; focus never stolen |
| `AdvanceWorkerStartNotice`, `WorkerStartNoticeID` | Launch identity/receipt uncertainty/reconnect sequences → pure properties plus one sticky witness; count native submissions, not only send calls |
| `EncodeSessionFrame`, `DecodeSessionFrame`, `SessionClient.session` | Malformed/version-conflicting bounded frames → seeded fuzz plus stateful negotiation sockets; no new field sent to old peers, no timeout downgrade |
| `peerDelivery.enqueueLocked`, `proxy.dispatchPeer`, cue coordinator | Supervisor death between commit and confirmation, recipient replacement, receipt expiry/re-exec → independent rendezvous controls; at most one native cue per launch and no uncertainty retry |

N/2N qualification: at 20/40 records, one added bulk startup classification call and linear reduction, zero added per-row client/native-resume/process-table queries; existing selected warm-attachment guards retain their own work. At 20/40 workers and unchanged tick sequences, at most one generic cue per needed launch, zero re-cue after same-launch reconnect, zero new filesystem work per automation/cue tick. Preserve existing native timeouts, broker/worker bounds and overload behavior; no timing claims or increased constants.

## Implementation tasks — one close boundary

### 1. Read-only startup policy and warm-only hubs

Files: new startup_policy.go/test; startup.go and existing startup tests; couchcmd run/hubs and startup rigs.

- [x] Write semantic reds against StartInteractive and pure classification, using the function strategies above; verify failure before code.
- [x] Implement bulk read-only collection, attachment-only implicit selection, per-hub initial automation/focus evidence and one report. Preserve explicit lifecycle operations.
- [x] Update Console no-handle initialization and extra-hub warm path; preserve background worker scheduler/reservations. Run targeted core/startup/reattach tests and commit.

### 2. Hub automation owner, CLI/UI, producer integration

Files: new core hub_automation.go/test, new couchcmd automation.go/test; request/CLI and Console key dispatch; generated producers and their existing reducers.

- [x] Write pure event and actual CLI/UI reds; qualify authorization, status and no implicit changes.
- [x] Implement one shared off/on owner and explicit controls, initially seeded from startup evidence. Suppressed outcome is typed and terminal.
- [x] Sweep every producer in the table through it, with its function-level controlled regression and no-replay invariant; retain passive and human/peer communication.
- [x] Publish the tested #737 getter contract. The second lander wires and qualifies due-run/script/notice consumers on main; unlanded scheduler coverage is not claimed here (see the 2026-10-21 ordering Revision).
- [x] Run focused automation, producer, lifecycle and Console suites/race tests; commit coherent behavior.

### 3. Wrapper-owned generic cue

Files: new worker_start_notice.go/test; session protocol/transport/registry and focused negotiation tests; wrapper delivery/runtime and cue tests; coordinator.

- [x] Write cross-supervisor red with one live fake wrapper and real private sockets; use the cue function strategies to prove native-input counts.
- [x] Implement the fixed-size wrapper witness and hello-v5 reporting with existing negotiation fallback. Old unsupported wrappers remain unknown; ordinary messaging stays live.
- [x] Wire canonical system-origin cue admission to exact current worker/owner and launch ordinal. Keep the separate restart resend prompt intact.
- [x] Run focused protocol/session/supervision/peer tests, bounded frame fuzz and race checks; commit.

### 4. Documentation, gates and operator rollout

- [x] Update README/atlas/index and shipped Couch skill/help for automation controls, initial state, status/report surface, suppression loss and the two worker notice lifetimes. tl-lead first-message rule is a separately owned ariadne issue, not Couch parsing.
- [ ] Audit source readers; run make test-affected-inventory and git diff --check; push tested head, run recorded local-check affected (known #738 graph fallback may select full suite). Classify failures; no gate waiver or blind rerun.
- [ ] Close once via SDLC fresh review after verification. Estimate only after plan-quality passes; no single-pass milestone tags.
- [ ] Hub/operator coordinate disposable-fleet acceptance and real restart: measured input/launch/notice counts and focus, plus manual automation controls and scheduled-run predicate. No real slot/restart/stage without concrete hub authorization. Record the agreed live evidence before claiming acceptance.
- [ ] On SHIP, sdlc merge --yes, verify Git/PR outcome, report hub and stop. Hub owns project updates.

## Architecture and lifecycle

ARCH-DRY/PURE: reuse presence, warm attachment, broker/receiver and negotiation; policy and transitions are pure, typed and singly owned. ARCH-PURPOSE: sweep primary/managed launch fallbacks and all automatic producers, including actual scheduled-run admission. ARCH-MOCK: private portable stores, stateful native/terminal doubles and real private sockets; coordinated live conformance. ARCH-CONSTRAINTS: new work is bounded startup/current-session work; no hot-path discovery, new executor or timeout increase. ARCH-SECURE: strict parsed records/frames, exact current caller/owner authority, unknown never authorizes input, no credentials. ARCH-ORDER: explicit automation events and per-launch committed cue witness; suppression is occurrence-owned and uncertainty never resets it. ARCH-FUNERAL: no durable files/processes; fixed wrapper witness dies with wrapper, bounded automation map with supervisor, no incarnation history. ARCH-EVOLVE: no persisted-format change; negotiate session v5 and omit new fields for old peers, retaining strict known-field validation. New automation status is specific to its new command, not an added non-omitted field on ordinary legacy responses.

## Approval

Operator approved the current contract through ops:0, including own-hub CLI authority and operator-only cross-hub switcher control. The final tl-lead rule operates the explicit control on the operator's first message unless told keep off; Couch itself never flips on input. #737 integration waits for its main landing and hub-relayed seam coordination. Plan-quality passed CLEAN and change-code admitted implementation on 6695cb081. Component verification is recorded in the issue Log; recorded checks and live rollout remain pending.

## Revisions

### 2026-10-20 23:42 PDT — operator-approved explicit start gate

The operator rejected the earlier first-submission release: they must be able to talk to the TL and debug while the gate remains closed. Name it the **start gate**, open only by authenticated `couch --start-gate open` after "let it start", or an explicit switcher key. Add `couch --start-gate status`; Ctrl+G is switcher-only and opens the selected row's owning hub through the same owner. No submissions, timers or reconnects release it and no suppressed work replays. This replaces, rather than supplements, the earlier counter/baseline mechanism.

Every surviving hub reattaches and works with its own gate closed. Missing hubs stay listed/parked; the single startup status names them and says the start gate is closed and to say "let it start". Initial focus remains on the launch hub if it survived, otherwise the switcher; a later extra-hub completion never steals it. Add mixed-fleet and multi-hub focus regressions. Scheduled task exit-10 wake notices (#737) stay gated (option a), including no replay on opening. The operator approved the remaining design; the hub instructed change-code and implementation.


### 2026-10-20 23:44–23:48 PDT — final hub automation policy

Operator follow-ups supersede always-closed/monotonic start-gate wording above: call it hub automation; initialize on for positively surviving hubs after a normal restart, off for full shutdown/unknown/missing hubs, and allow explicit off as a soft stop. Expose automation off/on/status [hub], plus switcher control. No automatic submission release. While off, skip scheduled due runs and generated wake notices; on resumes at the next due time without catch-up. Already-running scripts finish; human/worker input and desktop/status output are never gated. CLI names only the caller's own canonical hub; cross-hub control belongs to the operator switcher. tl-lead first-message interpretation is a separate ariadne change owned by the hub.

PQ-1 required named risky functions and adversarial strategies rather than prose case inventories. The active plan now has one function-level strategy table and a producer contract table, without duplicated per-case task lists. The previous reviewer completed before the follow-ups and refused only that format issue; no code was admitted. Re-run plan-quality against this committed revision.

### 2026-10-21 — scheduler seam and implementation ordering

The hub relayed pair:1's concrete #737 seam: messageService owns one
hubAutomationOn(hubRoot string) bool getter, injected at construction into the
scheduler and scheduleNotice. reserveDue and reserved-script authorization both
check it; off skips and advances NextDue; notice admission checks immediately
before generated send. The second issue to land wires and qualifies those
consumers, with no stacking. Thus if #739 lands first, its delivered surface is
the tested provider; scheduler coverage remains explicitly owed by #737 before
that feature lands. This supersedes the earlier ordering sentence that required
#739 to wait for #737 main integration in every case.

Implementation retains an owner epoch per hub. Delayed occurrences bind their
observation's epoch; off/on cannot revive them. Epochs are memory-only and do
not change the operator-visible two states or introduce recovery permissions.

### 2026-10-21 — empty configured hub projection

Final integration proved a never-started configured hub has no ThreadRecord and
therefore no ordinary switcher actor row. Hub approval f7d34e39-4724-43c5-bba1-
32359ac86584 permits a display-only parked/missing hub entry with one explicit
recovery action, without a new selectable hub-row model. The existing Ctrl+Space
Start form accepts the exact hub path and previews before starting. The entry
names that action; no fake conversation address or primary-as-numbered identity
is introduced. Bare-cwd startup separately binds the same verified HubOwnership
to routing and Console, so Ctrl+G works there too; failed resolution stays
unknown with a usable switcher and an explicit diagnostic.


### 2026-10-21 — BR-1 concept traceability correction

The first close review found a stale design row claiming a PURE change in
console_lifecycle.go, which is unchanged by this issue. The delivered pure
transitions are in couchcore/lifecycle_notice.go; lifecycle_notices.go owns
stateful admission, cancellation and publication IO. Split those rows, and split
new ObserveStartupContext in startup_policy.go from modified StartInteractive
in startup.go. Name WorkerCueReport and its receiver validation explicitly. The
starting-source survey remains historical design context, not a claim that its
pre-implementation fallbacks remain. These are documentation corrections only;
no new behavior or scope is introduced (ARCH-PURE).


### 2026-10-21 — BR-4 remove the unused selection helper

The second review reproduced the production dead-symbol guard: SelectStartupHub
had only test callers. Remove that unused helper instead of adding another
selection layer. ClassifyStartup's tests directly assert exact hub candidates;
production StartInteractive retains its existing candidate ranking and current
proof checks. Correct the plan/atlas claim that the report selector drove
attachment. This deletes unused surface without changing startup behavior
(ARCH-PURPOSE).


### 2026-10-21 01:59 PDT — proposed BR-5 admission ordering, awaiting hub

The third close review disposed BR-4 and reproduced two remaining races:
automation off in the broker families callback still produced a queued system
receipt, and a resource alert captured its epoch but lost it across recipient
resolution, so off/on revived the old alert. A pre-send check is not an
admission boundary. Stop production edits pending approval of this revision.

Proposed smallest correction (ARCH-ORDER, ARCH-DRY):

- The resource making the invariant one-step is the existing hub-automation
  owner's mutex. Every generated hub occurrence carries its expected canonical
  hub, automation epoch and exact recipient binding through routing, observation
  and reservation. The same permit must reach the mailbox commit, never be
  recaptured as current after slow work. Worker-directed messages remain
  independent; no persistent state or new visible state.
- Extend only the internal generated-send seam with an optional admission
  guard. At each broker mailbox mutation (direct admission, already-busy
  enqueue, and busy-after-observation enqueue), under the existing broker
  mutex, the guard verifies the actual exact recipient, locks the automation
  owner, and checks enabled + expected epoch. Keep that lock through the
  in-memory mailbox mutation, then release it. No routing, endpoint IO, native
  write or callbacks with IO occur while the automation mutex is held.
  Lock order is broker then automation; control mutates automation only and
  never acquires the broker, avoiding an inverse dependency.
- Off and first mailbox admission become ordered: off first suppresses the
  occurrence, including an intervening off/on; admission first keeps its
  canonical receipt and existing delivery/uncertainty rules. Already-admitted
  work is not retracted or falsely reported undelivered. Receipt lookup and
  wrapper-retained committed receipts retain their current semantics.
- Sweep lifecycle, wait, stale reminder, resource, stall and supervision
  adapters to transport the occurrence's original permit. Resource dispatch
  must pass its stored epoch through resolve/send. Unknown or invalidated
  permits retire through each producer's existing suppressed disposition, not
  busy/retry. Scheduler #737 receives this same seam on second landing; no
  independent policy stack.
- Controlled red/green tests hold routing, Observe and Reserve, then complete
  off or off/on before releasing. Cover direct and both enqueue paths,
  unchanged-policy success, binding replacement, and the inverse ordering
  (admission first remains canonical), plus per-producer epoch transport.
  Preserve counts, worker/human exemptions, native timeouts, queue bounds and
  no-replay behavior. No automatic retry or extra executor.

Simpler alternative: another policy check immediately before SendSystem is
insufficient, because broker routing/endpoint IO follows it (the measured red).
Holding the automation mutex across all broker IO would serialize off behind
slow external work and deadlock callbacks that control automation; reject it.
The proposed guard holds it only over the actual in-memory commit.

Visible states remain hub automation on/off with explicit on/off recovery; no
new user state or data-loss mode beyond intentional retirement of suppressed
occurrences. Live acceptance remains owed through #744 after #761.


### 2026-10-21 02:01 PDT — BR-5 correction approved and cross-issue lock order

Hub b88aa936-7bcc-40c4-a9f6-e4d7432377f8 approves the preceding proposal. The
required total order with #737 is hub-effect gate, scheduler authority, broker,
automation. The scheduler may read automation under earlier locks, but no
automation-held callback acquires those earlier locks or performs IO. Atlas
records the order; broker tests assert broker/policy ownership through receipt
publication, while service tests assert automation is not retained across
routing. The second lander verifies the actual combined scheduler wiring.
Implementation does not claim unlanded #737 behavior.

Resource pending policy retains its original hub alongside its existing epoch,
both json-excluded volatile fields; changing hub while pending retires the old
occurrence. No persisted format or new UI state. Already-admitted receipts
remain canonical after off; the guard governs only new mailbox mutations.


### 2026-10-22 — operator-approved maintenance during the acceptance shutdown

Per hub b6bfeab3, pair:0 remains with #753 until the explicit #739 stage go. The coordinated full-shutdown checklist now includes the operator deleting the old AgentScratch volume and running brew upgrade go (1.27.1→1.27.3) while Couch is fully down, before starting Couch again. After restart, the hub dispatches one rebuild per slot. Those are operator machine changes, not worker commands. Then verify retained startup status plus automation OFF, followed by the first genuine operator message and the TL's explicit automation ON unless told to keep it off. No implicit flip, no replay, and no change to positively surviving normal-restart ON semantics.
