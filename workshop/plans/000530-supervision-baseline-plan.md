# Session freshness and stuck-slot supervision Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans (otherwise). Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the hub timely, explainable nudges from observed activity, without retaining or replaying failed messages.

**Architecture:** Keep the session-file ping in Parley's existing watcher. Pair exposes factual file-write observations, logs bounded feature samples, and extends the existing Couch supervision tick using a pure transition model. A one-shot startup prompt asks connected workers to reconsider their own unanswered reports; it never replays those reports.

**Tech Stack:** Go, existing Couch broker/actor snapshots, sessionsync files, diagnosticlog; Lua consumer in separately tracked parley.nvim#321.

---

> **Superseded draft:** The sections below preserve the original proposal. The
> 2026-10-13 Revision at the end is the authoritative rules-only implementation
> plan. Fact logging is now pair#543; none of the online sampling or immediate
> broker admission proposals below will be implemented.

## Approval and boundaries

Draft for TL approval; no implementation or change-code yet. The operator's
ephemeral-message correction supersedes the old offline queue proposal in the
issue. Existing #492 busy-recipient queues and existing wait-notice retries stay
unchanged (TL receipt 9afd9c9b-0949-45d1-9b98-f673f228f51f).

Do not modify attach, resume, recovery or slot-state transitions (#528). Read
only existing connected/liveness/workspace snapshots, broker facts and genuine
operator-submission observations. No transcript crawling, process polling,
inference service, automatic slot recovery, or learned weights.

The existing baseline consumer is Parley's `session_sync.lua:probe`, currently
five minutes from oldest Owed.At. Companion **parley.nvim#321**, filed with
`deps: [pair#530]`, changes that one watcher. Pair must not add a competing file
ping. Pair lands the producer/supervision work first; the cross-repo baseline is
complete only after the companion lands and their shared scenarios pass.

## Alternatives

1. **Recommended: explicit rules plus bounded observations.** Use existing
   factual inputs and a deterministic state machine. Explain why each notice
   fired; collect data for a later comparison without changing the rule online.
2. Only adjust Parley's timeout. Smallest change, but misses stuck workers and
   lost restart reports, and produces no comparison data required by the issue.
3. Train/rank slots now. Rejected: no labeled baseline yet, more moving parts,
   and contrary to the operator's rule-based-first instruction.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| FileWriteObservation | cmd/internal/sessionsync/sync.go | modified Status/result metadata |
| SupervisionState / Event / Effect | cmd/internal/supervision/model.go | new, private state with exported transition entry |
| FeatureVector and report classification | cmd/internal/supervision/features.go | new |
| Sample lifecycle | cmd/internal/supervision/sampling.go | new |

- FileWriteObservation carries the latest successful agent write and observed
  document mtime (`FileWriteAt` in Status). Invalid/missing timestamps are unknown;
  never manufacture an old timestamp. New fields are additive; Parley falls
  back to LastWrite plus its existing observed file mtime with older Couch.
- One SupervisionState per Couch lifetime owns per-binding report episodes,
  per-file sampling state and one startup-prompt attempt per slot address. Only
  its transition method mutates these; adapters return effect outcomes as events.
- A report's kind is a deterministic hint from its bounded first line:
  explicit landed/completion wording, explicit waiting-for-operator wording,
  explicit progress wording, otherwise unknown. No semantic reply detection.
  Test negated/quoted/ambiguous wording as unknown. Counts retain uncertainty.
- FeatureVector contains counts by hint, owed-delivery count, changed-slot count,
  minutes since file write/slot action, operator-submitted-since, delivery-failure
  or uncertainty evidence, and known/unknown flags. No message body, transcript,
  draft, code, link, token, or raw file path is logged.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Session-file telemetry adapter | cmd/internal/couchcmd/session_sync.go; new session_observation.go | modified/new | successful local Sync.Write/Status and one bounded broker request |
| Session observation request | cmd/internal/couchmessage/protocol.go | modified | strict additive operation shape |
| Supervision coordinator | cmd/internal/couchcmd/supervision.go; wait_notice.go; message_service.go | new/modified | existing supervision tick, broker outcomes and cached snapshots |
| Immediate prompt admission | cmd/internal/couchmessage/broker.go | modified | existing caller/recipient authority, reservation and one delivery attempt |
| Feature logger | cmd/internal/couchcmd/supervision_log.go | new | existing diagnosticlog writer and retention |
| Operator escalation | cmd/internal/couchtty existing notice API, wired by couchcmd | modified adapter only | one visible Couch notice, no desktop alert |
| Baseline ping consumer | parley.nvim/lua/parley/session_sync.lua | separate issue #321 | existing status/actor probes and buffer lifetime |

Use existing stateful broker/endpoint/host fakes. The observation adapter and
logger use private temporary stores and real Unix sockets in integration tests.
No fixture may reach the live Couch socket, operator session files or desktop.

## Rules and person-visible projection

Proposed slot thresholds: first stuck nudge at 15 minutes, operator escalation
at 30 minutes without an observed response/action. These are explicit initial
defaults for TL review, not learned weights. Preserve the existing faster picker
and ordinary wait notices; a notice already sent for the same episode satisfies
the first nudge so the two paths cannot spam duplicate reports.

| Observed condition | What the person/agent sees | One action |
|------|------|------|
| Session file has owed deliveries and no write for 15 minutes | Existing Parley ping names file, age and owed count | Owner writes/folds the session file |
| Quiet settled worker off resting branch, report to hub with no later hub message/operator submission | Hub gets one notice naming observed age and missing action | Hub inspects that slot with couch --peek |
| Unanswered explicit completion report after worker returns to resting branch | Same hub notice; completion must not disappear merely because landing returned the worker to main | Hub inspects that slot |
| Hub already nudged, still no observed action at escalation threshold | One Couch notice names slot and age; no automatic desktop notification | Operator inspects that slot |
| Activity/branch/ownership observation unknown | No stuck classification; feature row marks unknown | Hub can inspect if needed |
| Hub available after Couch startup | Each connected nonhub worker gets one resend-request attempt | Worker resends only a still-relevant unanswered report, otherwise does nothing |
| Startup prompt refused/uncertain | Coalesced Couch notice lists missed/unconfirmed slots | Hub sends those slots a fresh status request if needed |

“Unanswered” in a notice must be phrased as **no later hub message or operator
submission observed**. Receipt submission is not an answer. Output, redraws and
progress chatter alone are not a response/action and do not clear this fact.
Genuine operator input/submission or a later hub-to-worker delivery clears the
episode; a new worker report starts a new generation. Unknown observations never
count as confirmed action or failed delivery.

Parley #321 owns the exact baseline computation: owed nonempty, latest valid
LastWrite/FileWriteAt at least 15 minutes old, owner live and quiet, operator lock
not held. One ping per write generation in the existing buffer lifetime. A write
postpones the timer even if older owed rows remain; more deliveries cannot rearm
it. Preserve configured threshold overrides and leave the visual stale band alone.

## Startup resend contract

At the first available hub in each supervisor lifetime, consider connected
nonhub slots; late-admitted slots are considered when first seen. Prompt text:

> Couch restarted and the hub is available. If your last report to the hub is
> still unanswered and relevant, resend it now. Otherwise do nothing.

Keep only attempted slot addresses, not original reports. Mark attempted before
effect submission. Ordinary reconnect or a replacement binding at that address
does not trigger another prompt in this lifetime. A new supervisor has a new
attempt set. Never persist this set as delivery work.

Existing Broker.Send can queue and its delivery worker can retry, so it is not
sufficient unchanged. Add an immediate, exact-binding admission policy that
reuses normal authorization/reservation/receipt machinery but refuses a busy
recipient and performs at most one endpoint delivery attempt. Do not put it in
the busy queue or held/retry path. Verify the expected recipient binding at
admission; a replacement cannot inherit an earlier prompt. A missing reply is
unconfirmed, never grounds for retry. Normal sends retain their existing policy.
Prompt/system notices are excluded from report counts and owed-file obligations
so they cannot generate their own reminders.

## Observation and sampling contract

1. Successful session-sync status/write operations offer a small typed telemetry
   observation to the live supervisor. Include opaque canonical file identity,
   owner, observed write timestamp, owed count and uncertainty; a write includes
   pre-write obligation metadata returned by Sync.Write from inside its existing
   transaction, so folding does not erase the positive sample and an unlocked
   preliminary read cannot race it.
   No directory-wide discovery. Standalone file operations still work without
   Couch: telemetry failure cannot turn an already successful write into failure.
2. Reuse the broker's caller identity for agent writes. Status probes may run in
   the operator's editor: their read-only observation is limited to the existing
   local owner-only socket boundary and cannot mutate files or slot state. Bound
   the operation shape, reject malformed metadata, and keep it separate from
   peer-send input/allowance. One best-effort call, at most 250 ms, no retry or
   background goroutine. Return telemetry coverage separately from file success;
   routine status polling must not repeat warnings for an older supervisor.
   A write may report unavailable telemetry once in its result; the supervisor
   coalesces received coverage-gap diagnostics by generation.
3. The coordinator snapshots existing actor/workspace/liveness facts on the
   existing 15-second supervision tick. Compare cached facts to count changes;
   do not add new Git/process probes. Observe admitted worker/hub message outcomes
   and genuine operator submissions. A failed socket connection while Couch is
   absent cannot be observed there; startup resend requests cover that blind spot.
4. Emit a positive feature row at a successful write, before resetting counters.
   Emit the one-minute-after-write negative only if no intervening write has
   superseded that generation. Random sample candidates use an injected seeded
   draw (initially one randomly selected observed entity per minute); label negative only
   after ten minutes with no write/action. An intervening positive invalidates
   the overlapping candidate. Apply the same scheme to hub/operator action on
   a previously silent worker. System-generated prompts are never positives.
   First observation has unknown prior coverage, not zero activity. Duplicate
   write generations are idempotent. An external mtime change with no successful
   write observation censors overlapping samples; it is not a fully observed
   positive. Document this coverage limit rather than claim every filesystem
   write is observed. Status observations cannot themselves count as writes.
5. Keep at most 10 pending candidates per entity and 64 observed file identities
   globally; reject excess telemetry with an explicit capacity diagnostic. Slots
   are bounded by broker MaxActors. File observations expire after one hour
   without a status/write observation; incomplete samples then become censored,
   not negatives. Process restart censors pending samples. Out-of-order or clock
   regression observations are marked unknown and cannot create early nudges.
6. Feature JSONL uses the existing diagnostic writer, schema version, supervisor
   generation and opaque entity ID. Reuse its size rotation, ownership and GC;
   set a 4 MiB active-generation limit, document existing fleet retention rather
   than introduce another retention engine. Logs are diagnostic data only and
   never consulted to reconstruct or resend messages. Log overflow/write failure
   cannot block UI or delivery; expose a dropped-sample/coverage-gap diagnostic.

No logistic fit, trainer, model switch, or accuracy claim ships here. Logged
known/unknown/censored facts make a later offline baseline comparison possible.

## State transitions and invariants

| State/event | Next state and effects |
|------|------|
| File observation/write | Update factual generation; write emits positive; old pending negatives invalidated |
| Eligible sample candidate | Pending until due; no nudge effect |
| Pending sample + ten minutes with known no action | Emit negative, remove candidate |
| Candidate + action/restart/expiry/unknown interval | Invalidate or censor; never label absence as negative |
| Worker report + known quiet state at threshold | Awaiting-action → nudged; one hub notice, coalesced with existing wait episode |
| Nudged + continued lack of action at escalation threshold | Escalated; one visible operator notice |
| Any report episode + observed hub reply/operator action/new incarnation | Clear or replace episode; late effects cannot affect successor |
| Hub first available + unattempted worker address | Mark attempted then one immediate prompt effect |
| Prompt outcome success/refused/uncertain | Record outcome only; never transition back to pending |
| Shutdown | Cancel/join watcher and bounded calls, close logger, discard volatile state |

The pure model owns these transitions. Effects carry generation/recipient
identity; IO results for superseded generations are ignored. No message body is
durable state. There is no automatic operator-facing desktop notification.

## Implementation tasks (one close boundary, after approval)

### 1. Factual file contract and pure supervision model
- [ ] Add failing sessionsync tests for LastWrite/FileWriteAt, nil/invalid state,
  successful write and partial state-write failure; extend pure model fixtures
  for exact thresholds, reset/dedup, unknown observations and stale generations.
- [ ] Implement the minimal factual contract and private transition state; no IO
  in rule/classification/sampling functions. Run
  `go test ./cmd/internal/sessionsync ./cmd/internal/supervision -count=1` green.
- [ ] Add seeded sampling sequences: positives, one-minute negative, ten-minute
  maturation, overlapping writes/actions, censoring, restart and capacity limits.
  Commit the model and producer contract.

### 2. Bounded adapters and one-shot restart prompt
- [ ] Add red real-socket/broker tests for registered caller, malformed metadata,
  absent hub, reconnect order, replacement, busy recipient, uncertain delivery,
  and exactly one attempt; ordinary queue semantics must remain covered.
- [ ] Wire existing supervision tick and cached activity; add immediate-admission
  policy through existing broker machinery. No attach/resume/state code edits.
- [ ] Add file telemetry and bounded diagnostic writer; tests use portable temp
  stores, injected clocks and realistic partial failures. Verify a failed optional
  telemetry write never invalidates the actual session-file write.
- [ ] Run `go test ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/sessionsync -count=1`
  and focused race checks; commit adapters.

### 3. Scenario verification, companion handoff and close
- [ ] Build a synthetic replay of the reported overnight pattern: worker reports,
  hub unavailable/restart, no observed response, later human action. Label it a
  scenario reconstructed from the incident, not a captured historical trace.
  Verify one hub nudge, one escalation, no payload replay and no duplicate prompt.
- [ ] Test no changes to ordinary picker/wait notices, normal busy queues, message
  authority and receipt uncertainty; ensure supervision cannot feed itself.
- [ ] Update README, atlas/couch.md, sessionsync contract docs, embedded Couch
  skill and artifact inventory. Publish the exact producer seam to TL for
  parley.nvim#321; do not edit its implementation from this slot.
- [ ] Run the full queued `make -k` to completion; no shortened run or gate
  bypass. Schedule operator smoke through TL only, including startup prompt and
  visible escalation with private short thresholds if approved.
- [ ] Close through mandatory SDLC review; on SHIP merge main, open PR, wait CI,
  `sdlc merge --yes`, report to hub and stop. Report the companion baseline as
  outstanding until its own acceptance, not as delivered by Pair alone.

## Architectural accounting

- ARCH-DRY: Parley remains the sole file-ping watcher; reuse Couch tick, broker,
  file state and diagnostic retention, not a second recovery subsystem.
- ARCH-PURE: one private transition model plus pure feature/sampling rules.
- ARCH-PURPOSE: separate companion explicitly covers the actual baseline
  consumer; completion reports still matter after a worker returns to main.
- ARCH-MOCK: existing stateful endpoint/host fixtures, real private socket/store,
  fake clock/random sequences; TL-managed live smoke confirms integration.
- ARCH-CONSTRAINTS: 15-second cached tick, bounded metadata/calls, 64 file watches,
  10 pending candidates/entity, broker-bounded slots, rotating diagnostic output.
- ARCH-SECURE: typed untrusted observations, current caller/recipient identity,
  no payload/path/secret logging, read-only local observation cannot change slots.
- ARCH-ORDER: explicit phases and effect generations; uncertain outcomes never
  trigger replay or become confirmed action; all work belongs to service lifetime.
- ARCH-FUNERAL: volatile episodes expire on action/disconnect/shutdown; file
  watches expire after one hour; logs use existing diagnostic retention/GC.


## Revisions

### 2026-10-13 20:53 PDT — approved simplification and separate landing

TL receipts ddcbf61f-f57a-4aa9-a740-f4614830f5c3 and
addd69f2-20e6-4697-8404-8fa5396f0d52 supersede the original scope, concepts,
transitions and tasks above. #530 is single-pass rules only, one sdlc close.
Fact logging is separately filed as **pair#543**, depends on pair#530, with the
revised design in its Spec. No feature logging or sampling boundary remains in
#530. No new immediate-admission broker policy. Normal Send is the single owner
of admission and its existing in-memory queue remains unchanged (ARCH-DRY).

#### Current core concepts

| Name | Lives in | Status |
|------|----------|--------|
| supervisionState, reportEpisode, supervisionEvent/effect | cmd/internal/couchcmd/supervision_model.go | new PURE |
| completionHint | cmd/internal/couchcmd/supervision_model.go | new PURE |
| Candidate operator submission observation | cmd/internal/couchmessage/routing.go | modified PURE data |

One state owns current-binding report episodes and per-lifetime attempted slot
addresses. State changes only through transition methods. No IO or clock reads
inside transitions. Reports contain time, ID and completion hint, never retained
body text. The bounded first line recognizes explicit landed/merged wording;
negated, quoted or ambiguous text remains noncompletion. A hint only preserves
eligibility on main; it is never proof that work landed.

| Integration | Lives in | Status | Wraps |
|------|------|------|------|
| supervision coordinator | cmd/internal/couchcmd/supervision.go | new | mutex around pure transitions, effects outside lock |
| report attempt/reply hooks | cmd/internal/couchcmd/message_service.go | modified | authenticated send entry and confirmed endpoint delivery |
| submission facts | cmd/internal/couchmessage/broker.go | modified | copy existing accepted submission counter into candidate snapshot |
| common tick and coalescing | cmd/internal/couchcmd/wait_notice.go | modified | existing 15-second watcher and wait episode |
| visible escalation | cmd/internal/couchcmd/message_service.go | modified | existing console.Notify status notice |

No registry lifecycle, attach, resume or recovery mutation. No extra polling or
background workers. Coordinator uses the existing waitWatcher tick and service
lifetime. Endpoint callbacks and ticks serialize model transitions with one
mutex, releasing it before Send/Notify. Effects carry current binding and report
ID; stale outcomes cannot mark a successor report notified.

#### Rules and recovery

- Authenticate a worker send before recording its report attempt. Reserved hub
  targets are recorded even if hubRecipient refuses because the hub is absent;
  an exact current hub slot target is also recognized. Malformed/unauthenticated
  requests do not create reports. Deduplicate the same message ID.
- Record confirmed hub-to-worker endpoint submissions as action. Queued,
  cancelled, expired or indeterminate messages are not answers. Supervisor
  prompts and wait notices must not enter report/action hooks or file obligations.
- On first snapshot capture the existing operator submission counter; only a
  subsequent increase clears an existing episode. Key by full binding so a
  replacement never inherits old observations. A submission racing a new report
  must be compared against the report's captured submission generation.
- At 15 minutes since a report, nudge if the current worker is known, settled,
  quiet at least 30 seconds, with no pending incoming delivery, and is off its
  resting branch OR has an explicit completion hint. Unknown facts suppress it.
  Text says “no later hub message or operator submission observed”; one action
  is couch --peek SLOT. Output/redraw alone does not clear a report.
- Preserve earlier ordinary wait/picker notices and their retries. A successfully
  admitted wait notice for the same report generation counts as the first nudge;
  a rules nudge likewise suppresses a duplicate ordinary wait notice for that
  episode. Do not treat old notices preceding the report as coalesced. At 30
  minutes without action emit one operator Couch notice, never desktop notify.
- Successful hub nudge admission advances notified state; failure permits the
  ordinary next-tick retry. Operator notice is emitted once. A new report or
  confirmed response resets the episode; disconnect discards per-binding state.
- When a hub is available, each connected nonhub slot address gets one startup
  prompt attempt per supervisor lifetime. Mark attempted before normal Send;
  reconnect/replacement cannot prompt twice. Busy queueing is allowed within
  this lifetime. There is no new retry by the coordinator, persistence or
  replay across restart. Late-connected workers receive their one attempt too.
- Prompt: “Couch restarted and the hub is available. If your last report to the
  hub is still unanswered and relevant, resend it now. Otherwise do nothing.”
  Identify supervisor-generated messages structurally, not by user-forgeable
  prefixes; system effects bypass report/action accounting. A prompt failure
  produces one readable Couch notice advising the hub to request fresh status.

#### Current implementation tasks — one close boundary

- [x] Write failing pure tests in supervision_model_test.go: exact 15/30-minute
  boundaries, unknown/active/pending suppression, completion after main, normal
  report on main ignored, response/submission reset, duplicate IDs, stale effects,
  notice coalescing and once-per-address startup attempt across replacements.
- [x] Implement supervision_model.go minimally; run
  `go test ./cmd/internal/couchcmd -run 'TestSupervision' -count=1` green.
- [x] Extend existing stateful wait-notice and message-service fixtures in
  supervision_test.go and message_service_test.go. Red tests cover authenticated
  failed hub send, confirmed vs queued reply, genuine operator submission,
  startup normal busy queue, reconnect and lost acknowledgement without a second
  coordinator send. Tests use private sockets and injected clocks, not live Couch.
- [x] Implement supervision.go and the hook/snapshot/coalescing changes named
  above, then run focused couchcmd and couchmessage tests and relevant race tests.
- [x] Add the reconstructed overnight scenario: completion report, return to
  main, no hub action, one nudge, one escalation; a restart produces one fresh
  resend request without original report replay. Label reconstruction honestly.
- [x] Update README, atlas/couch.md, embedded Couch skill and artifact inventory.
  Parley #321 remains the sole file watcher; its change can use existing LastWrite
  and observed mtime without waiting for #543 telemetry. Do not implement it here.
- [ ] Run full `make -k`, complete all targets, cite any known failures. Request
  live smoke through TL only if needed. Run one sdlc close; on SHIP merge main,
  PR/CI/sdlc merge --yes, report and stop. #543 waits for this landing.

ARCH-PURE/ORDER: one private transition owner, effects after decisions, exact
binding/report generations for outcomes. ARCH-MOCK: reuse stateful endpoint and
service fixtures. ARCH-CONSTRAINTS/FUNERAL: only connected report state plus the
required per-lifetime attempted-address set; no report body retention, no disk
queue, no new watcher. ARCH-SECURE: authenticated observations and existing Send
authority; supervisor effects cannot masquerade as worker reports or hub replies.


### 2026-10-13 — implementation seam refinement

The broker exposes a read-only Submission(binding) accessor rather than adding
operator counters to the public Candidate wire shape. Existing wait-notice-off
also mutes the new report reminders; picker/prompt notices retain their behavior.
A bounded volatile send-ID table distinguishes genuine hub replies from generated
prompts and deduplicates observed CLI attempts; it uses the broker receipt cap and
a two-receipt-lifetime retention window (queue plus operator hold can exceed one
hour), never message bodies or persistence. The coordinator rechecks report IDs
before applying effects. No feature-log machinery was added.


### 2026-10-13 — final rules inventory and acceptance

The current pure implementation names are supervisionState, reportEpisode,
supervisionFacts and supervisionEffect in supervision_model.go, plus completionHint.
Events enter through report/reply/nudged/tick transition methods; no separate
supervisionEvent struct or Candidate wire field was necessary. Integration uses
supervisionCoordinator and Broker.Submission as described above. The historical
original-draft concepts and task checkboxes are retired, not unfinished work.

TL receipt 018dd090-c2ec-45bc-8425-b9c1394204d9 accepts the private-socket/clock
scenarios; no stage or operator restart is required. The full command is
make -k test (bare make -k prints help here). The machine gate queues it; no
custom wait loop or resource override is used.


### 2026-10-13 — full-run result

make -k test ran every target to completion. The only failure was the new
completion-after-main fixture racing its startup delivery. The submitted-receipt
barrier fixed it; all integration cases passed 50 repetitions, focused race tests
passed, and the full affected package passed afterward. The snapshot ordering
regression also passed red/green. Mandatory close review is next; the original
full-run exit remains nonzero and is recorded honestly in the issue Log.


### 2026-10-13 — boundary-review corrections

BR-1/2/3/4: replace the capped two-hour send-ID history with pending-send metadata
only. The broker remains the one capacity/receipt owner. Generated waiting
notices use the same normal Send adapter and provenance exclusion as startup
prompts. New read-only Broker.ReceiptSnapshot exposes retained terminal outcomes
to supervision even when a caller disconnected; it changes neither the wire nor
admission. The existing tick reconciles submitted/cancelled/expired/indeterminate
receipts and then discards metadata. Unconfirmed startup delivery emits one
notice with the recovery action “hub should request fresh status”; no prompt is
regenerated. The primary issue Done when now states the approved rules-only
acceptance, with deferred file pings/logging linked to their companion issues.
