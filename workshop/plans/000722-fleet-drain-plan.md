---
collaboration-mode: discussion
---

# Fleet drain implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Execute under `superpowers-executing-plans`; SDLC owns the mandatory plan and close reviews.

**Goal:** Give the steward one checkpoint-and-stop request and a truthful fleet drain board before restart.

**Architecture:** A pure drain transition model owns one generation. A supervisor coordinator persists intent, pauses SDLC through an owner-scoped adapter, gates dispatch, sends system notices, and projects authenticated acknowledgements plus fresh observed state. Fast mode adds identity-fenced wrapper interruption; neither mode terminates Pair sessions or restarts Couch.

**Tech stack:** Go, existing Couch broker and wrapper endpoint protocol, portable namespace storage, console event loop, and an injected SDLC command adapter.

**Approval:** Draft for TL review; no implementation admitted. TL receipts b2e72204, 205542db and 126cc5c8 approve the ack, owner-scoped queue seam and steward-only control respectively. Async start and the remaining policies below are proposals for this review.

## Scope and alternatives

Recommend a supervisor-owned drain generation with explicit ack and observed completion. A broadcast alone cannot establish completion; automatically parking sessions cannot establish an agent-authored checkpoint and would change the approved idle-at-prompt contract. Reuse `HubOwnership.Main()` for the first-command-line-hub steward, not ordinary per-slot hub ownership (ARCH-DRY).

The operator asks the steward TL to run the command. No outside-Couch operator-shell authority, operator console action, automatic restart, machine shutdown, automatic git writes, filesystem/transcript checkpoint scanner, or fleet-resume implementation is included. Those entry/resume extensions belong outside this issue. Acknowledgement attests the worker's durable checkpoint; the board does not independently prove git contents or remote durability.

Develop against the stateful queue fake while ariadne#481 is unstarted. Production enablement and close require #481's real seam conformance; unsupported old SDLC means stuck/unsupported, never an implicit no-op pause.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `DrainState`, `DrainEvent`, `DrainEffect`, `ReduceDrain` | `cmd/internal/couchcore/drain.go` | new |
| `DrainBoard`, `DrainRow`, `ProjectDrainBoard` | `cmd/internal/couchcore/drain_board.go` | new |
| `DrainNotice` (v1 formatter) | `cmd/internal/couchcore/drain_notice.go` | new |
| `QueueSnapshot` and decoder validation | `cmd/internal/couchcmd/drain_queue_model.go` | new |
| Typed drain CLI/request/response shapes | `cmd/internal/couchcmd/drain_cli.go`, `cmd/internal/couchmessage/drain_protocol.go` | new |

One DrainState owns at most 128 current member slots, keyed by canonical workspace and exact wrapper binding/session token. Each row has one current ack and one notice/interrupt outcome, not an append-only observation history. The board is derived from this model and current evidence, never independently mutated by the CLI or tab bar (ARCH-PURE). QueueSnapshot is immutable input; it represents known/unknown completeness explicitly. Tests for these entities use literal values and event sequences, without filesystem/process mocks.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `drainCoordinator` | `cmd/internal/couchcmd/drain_service.go` | new | model, clock, broker gate, registry facts, queue and store |
| `drainStore` | `cmd/internal/couchcmd/drain_store.go` | new | one portable namespace JSON record and atomic publication |
| `drainQueue` / `sdlcDrainQueue` / `FakeDrainQueue` | `cmd/internal/couchcmd/drain_queue.go`, `cmd/internal/couchcmd/drain_queue_fake_test.go` | new | owned SDLC CLI process boundary |
| Broker drain gate | `cmd/internal/couchmessage/broker.go` | modified | ordinary peer admission and queued reserve/delivery |
| `DrainInterrupt` endpoint and wrapper executor | `cmd/internal/couchmessage/drain_endpoint.go`, `cmd/internal/wrapcmd/drain_interrupt.go` | new | one fenced Esc and its retained outcome |
| Drain request wiring | `cmd/internal/couchcmd/{cli.go,messages.go,message_service.go,run.go}`; `cmd/internal/couchmessage/{protocol.go,endpoint.go}` | modified | CLI, broker and wrapper request routing |
| Console drain view | `cmd/internal/couchtty/console_drain.go`, existing status/tab projection | new/modified | immutable board publication into console loop |

The coordinator receives stateful fake queue, fake broker/endpoint fleet, fake clock and portable store through the same boundaries as production (ARCH-MOCK). Reuse serviceRig/fakeClock, exact liveCaller/current checks, Registry.Liveness, fakeSlotRunner's explicit completion model and console SetBlockedThreads-style event publication. Do not reuse RequestContinuation's park-and-relaunch execution or classify message receipts as checkpoint proof.

## Public Couch contract

- `couch --drain [--reason TEXT] [--fast] [--timeout DURATION] [--json]`: start; return ID and initial board immediately. Only the currently verified steward may start or cancel. Including the steward in the target set makes waiting synchronously for all-drained a self-deadlock.
- `couch --drain --status [--json]`: read-only board; anyone. Does not send notices, acknowledge, interrupt, resume, or extend the deadline.
- `couch --drain --ack ID [--json]`: the sole worker protocol action after checkpointing; authenticate the current member's exact binding and admitted session token. No separate free-text drained report is required. Ack may arrive during the checkpoint turn; drained waits for a subsequent positive idle observation.
- `couch --drain --cancel [--json]`: steward only; release only this drain's queue owner. Retain Couch's gate while release is uncertain, and until a fresh snapshot confirms this owner absent. Other queue owners can remain paused and must be shown.
- Proposal: default timeout 10m, accepted 1s–24h. Reason is optional, at most 1024 UTF-8 bytes, terminal controls refused. Start returns promptly; the supervisor deadline publishes a terminal stuck list to the board and one console notice. Timeout never releases gates or claims permission to restart.
- A duplicate start while a generation is active returns its existing ID/status without another notice or Esc. A conflicting requested reason/mode is refused toward status/cancel. After timeout the same generation can still become drained as evidence arrives; start does not silently reset its deadline. Cancel then start is an explicit new generation.
- Read/ack/cancel remain usable while draining. Lifecycle operations that would launch/relaunch workers or dispatch new work are refused while active. Existing in-flight effects are joined/observed; they are never assumed cancelled. The first hub's control-plane attachment/recovery is allowed so a surviving drain cannot lock out its own cancellation.

### Exact notice text: shared v1 contract with ariadne#481

The dynamic ID is a canonical UUID; render optional reason as quoted data on its own labelled line. No reason text becomes an argv or an instruction. The formatter is the single production text owner; skill/help describe the contract rather than duplicating the full template.

Graceful:

> Couch drain v1 {ID}: checkpoint and stop. Finish the current atomic step; commit and push the checkpoint; write where you stopped in the issue Log. Start nothing new. Run couch --drain --ack {ID} after the checkpoint is durable, then end your turn and stay idle. Do not send a separate drained report.

Fast:

> Couch drain v1 {ID} (fast): stop now and checkpoint. Do not finish the current step. Write where you stopped in the issue Log; commit and push the checkpoint without discarding unfinished working-tree edits. Start nothing new. Run couch --drain --ack {ID} after the checkpoint is durable, then end your turn and stay idle. Do not send a separate drained report.

These are supervisor-origin notices through Broker.SendSystem with authenticated receipt lookup. A delivery receipt is not an ack. Old or unsupported wrappers remain stuck with a relaunch-needed reason; drain does not relaunch them itself. Exactly one notice is admitted per drain/current incarnation; an uncertain transport result is inspected, not blindly pasted again.

## SDLC seam frozen for ariadne#481

Run argv arrays, never a shell. The queue adapter runs from the verified steward workspace and does not manufacture another slot's environment/authority. Owner token is `couch-drain:{ID}`; create and persist it before the first external pause attempt.

```
["sdlc", "queue", "pause", "--reason", "Couch drain {ID}: {reason}", "--owner", "couch-drain:{ID}"]
# fast appends one argument:
"--cancel-running"
["sdlc", "queue", "resume", "--owner", "couch-drain:{ID}"]
["sdlc", "queue", "status", "--json"]
```

Pause/resume need only conventional exit status; every successful or uncertain mutation is reconciled through status. Pause idempotently adds that owner; resume idempotently removes that owner and no other. Multiple owners compose: admission stays paused while any owner remains. Fast cancellation records interrupted work, not failed work; owner removal never restarts it. There is no TTL that silently resumes a live pause. The #481 worker rule uses the notice text and ack action above.

Exact required status schema, lower_snake_case JSON keys:

```json
{
  "schema_version": 1,
  "state": "paused",
  "complete": true,
  "pause_owners": [
    {"owner": "couch-drain:00000000-0000-4000-8000-000000000001", "reason": "Couch drain ...", "cancel_running": false}
  ],
  "active": [
    {"id": "opaque-stable-operation-id", "workspace": "pair:5", "kind": "suite", "state": "running", "head": "optional-git-sha"}
  ],
  "detail": ""
}
```

`state` is `running`, `paused` or `unknown`; `kind` is `suite`, `merge` or `landing`; active entry `state` is `running`, `interrupting` or `unknown`. All fields shown are required except `head`. `complete:false` or `state:unknown` means absence cannot be inferred. `detail` explains incomplete/unknown observations. `running` requires zero pause owners; `paused` requires at least one. Entries with unknown workspace use `workspace:""` and block global quiescence. Waiters held before admission are not active. An operation may own several distinct kinds, each represented; do not identify work by PID alone. A complete empty `active` list is authoritative quiescence only at the captured observation; Couch refreshes it while the board is active.

Decoder rejects duplicate keys, malformed required types, unsupported versions, invalid enum combinations and oversized/truncated output. Tolerate unknown additive fields read-only. Limits proposed: 1MiB output, 128 pause owners, 4096 active entries, bounded text fields; overflow becomes unknown/incomplete, never truncated success. Couch performs no PID killing or registry-file edits.

`FakeDrainQueue` is a stateful implementation of the same pause/resume/status interface: owner map, FIFO waiters and exact operation IDs with workspace/kind/head/state; controllable finish, interruption and observation failure. Pause blocks admission, graceful retains running work, fast marks interrupting then explicitly interrupted/removed, resume releases only one owner's hold. Lost replies can occur after an effect; status still sees the applied state. Include a fake executable adapter test that validates exact argv/JSON, in addition to in-process model tests. Live conformance runs in #481's isolated portable queue state root, never against the operator's machine queue during tests.

## State, authority and recovery

Actions: **wait**, **inspect status**, **checkpoint and ack**, **retry cancel**, **cancel then start**, **resume interrupted work after cancellation/restart**. The board has draining/drained/stuck rows and a global queue/control reason. It is not a restart command.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Intent persisted; own pause not yet confirmed | draining: pausing queue | wait | none; no completion claimed |
| Pause call or status unavailable/unknown, or record corrupt | stuck: queue/storage unknown | inspect status | no evidence of safe restart; existing external effect may have happened |
| Pause confirmed; notice queued or turn still active | draining: checkpoint requested / active turn | wait | graceful preserves current step; fast may abandon in-flight reasoning |
| Notice refused, unsupported wrapper or picker/interrupt ambiguity | stuck: exact refusal | inspect status | no checkpoint promise; no automatic keystroke retry |
| Exact ack absent and slot otherwise positively quiet | stuck: no ack (immediate) | checkpoint and ack | durability is unattested |
| Exact ack present; checkpoint turn still running, or suite/merge/landing active | draining: waiting for idle / named work | wait | fast interrupted work will need a same-head rerun |
| Exact ack, positive idle/no picker/no background work, no active owned work; global queue confirmed paused/complete/quiet | drained | wait | none claimed beyond worker attestation and observed quiescence |
| Acked/drained slot becomes busy or receives later operator submission | draining: activity after checkpoint; prior ack invalidated | checkpoint and ack | later work not covered by prior checkpoint |
| Disconnect, replaced incarnation, unknown idle evidence, or activity outside the known fleet | stuck: identity/observation/external-work reason | inspect status | old ack cannot vouch for replacement; absence is not proof of exit |
| Deadline reached with outstanding rows/work | stuck: list every remaining reason | inspect status | none discarded; queue stays paused |
| Cancel requested; queue release pending/unknown | draining: cancelling (dispatch held) | retry cancel | no rollback of completed Esc/checkpoint effects |
| Own pause confirmed absent; another owner remains | inactive: cancelled; queue paused by other owner | inspect status | interrupted work does not restart automatically |
| Restart recovers an active record, old live evidence no longer current | stuck: recovered drain; cancel before a new drain | retry cancel | live ack/idle proof is invalidated; checkpoint data remains |

The model tags phases (preparing, active, cancelling, cancelled/recovered) rather than deriving lifecycle from a constellation of nullable flags. Row status is a projection of evidence. One supervisor coordinator owns the active generation and gates every dispatch reserve plus automatic worker launch; a pre-check of an empty queue is not the invariant. #481's owner pause is the separate authority preventing new SDLC admission machine-wide (ARCH-STATE).

Sequence policy:

- Persist start intent and close Couch admission before invoking pause; send notices only after own pause is confirmed. Failure leaves a visible stuck generation; do not rollback away a possibly applied pause.
- Ack binds drain ID, canonical slot, exact Binding and Registry SessionToken. A stale ack, receipt, queue response, interrupt completion or console publication cannot mutate a newer generation. Ack while busy is provisional until the next positive idle state; once sealed, subsequent busy/operator-submission evidence invalidates it. Preserve monotonic submission count through the registry if current projections omit it.
- Freeze the initial connected inventory including every hub and steward. Newly admitted live slots join the same bounded generation and need their own notice/ack; replacing a member invalidates its prior facts. Overflow means stuck, never silently omitted. Pre-existing in-flight deliveries and lifecycle operations must finish or be reported unknown before all-drained.
- Gate ordinary peer send and queued delivery reservations while draining; preserve already queued messages for cancel. Only this generation's drain notice bypasses the gate. Do not dispatch ordinary lifecycle/system work that could start another turn. An already handed delivery remains observed; later activity invalidates completion.
- Fast interruption is a distinct wrapper capability, not ordinary idle-only `--keys`. At most one Esc per ID/incarnation, under the wrapper's existing input ownership mutex/lease, only for a positively recognized active turn with no competing automatic input/picker/buffered input. Revalidate generation immediately before writing. Idle means no Esc. Lost confirmation remains unknown with a retained interrupt receipt; no repeated write. Stop confirmation does not replace checkpoint ack or idle evidence.
- Cancellation closes the coordinator's child work, joins bounded effects, reconciles only its queue owner, then removes dispatch holds. A late pause completion after cancel is reconciled/released, not allowed to repause an inactive generation. Stopping the CLI does not stop the supervisor-owned drain.

## Operating envelope and residue

Proposed budgets (explicit choices for review): at most 128 member slots (existing Broker.MaxActors); fan-out 4; one active coordinator tied to supervisor context; at most one queue-status subprocess every 2s while active. Each command has a 5s context, output cap and bounded process-group cleanup; UI/status reads use the published board and spawn no process. Fast endpoint budget 5s, fail closed on slow/unknown response. No per-byte work, transcript scans or new always-on idle polling. Queue model events examine at most current members plus returned activities, O(N+A). Counted tests at 8 and 16 members require one census, at most one notice/interrupt per current member and one shared queue observation per poll; no N status subprocesses.

Persist one bounded `drain.json` under the existing canonical Couch namespace (mode0600), atomic replace and context-aware serialization. Store at most the current or last completed generation (<=1MiB), including owner token and pending effect IDs before external actions. A new generation replaces the cancelled record; cancel retains one bounded last outcome. No new journal/history family. Existing supervisor namespace lock serializes writers; never hold a storage lock across external calls. Recovered intent closes dispatch before startup effects, rechecks the queue owner, and grants no inherited all-drained proof. Missing record means inactive; malformed/newer unsafe record means held/unknown with a diagnostic, not inactive.

New persisted format: known keys strict (including duplicate rejection); unknown top-level and nested fields survive read-modify-write via raw extension maps. Explicit version refusal protects interpretation of a pause/interrupt intent; unknown optional fields remain round-trippable. Tests cover crash after intent/before pause, after pause/before response, after resume/before record update, and old-reader rewrite preserving new fields (ARCH-SECURE, ARCH-EVOLVE, ARCH-FUNERAL).

## Implementation tasks — one close boundary

### 1. Pure model, board and notice contract

Files: new `couchcore/drain*.go` and colocated tests.

- [ ] Write literal/event-sequence tests first: `TestDrainGenerationSequences`, `TestDrainBoardTruth`, `TestDrainNoticeV1`, `TestDrainCountedGrowth`. Independently assert no all-drained without same-generation ack, positive idle and complete paused/quiet queue; cover no-ack quiet, expiry, later activity, replacement and reordered completion.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestDrain' -count=1`; confirm RED for missing contract, implement the transition/formatter, then GREEN. Commit the self-contained model.

### 2. Queue seam, portable persistence and fake

Files: new `couchcmd/drain_queue*.go`, `drain_store.go` and tests.

- [ ] Write `TestDrainQueueOwnerIsolation`, `TestDrainQueueLostReplies`, `TestDrainQueueCLIContract`, `TestDrainStoreCrashRecovery`, `TestDrainStoreUnknownFields`. Use a portable folder and stateful owner/activity transitions, not function-return call scripts.
- [ ] Run `go test ./cmd/internal/couchcmd -run 'TestDrainQueue|TestDrainStore' -count=1`; demonstrate RED, implement adapter/decoder/storage, then GREEN. Missing #481 binary support must produce unknown/unsupported. Commit.

### 3. Broker/coordinator, authenticated ack and fast interruption

Files: new `drain_service.go`, `drain_cli.go`, `drain_protocol.go`, `drain_endpoint.go`, `wrapcmd/drain_interrupt.go`; named existing wiring files above plus Registry.Liveness's submission observation if required.

- [ ] Write `TestDrainFleetGraceful`, `TestDrainFleetFast` with idle, mid-turn and running-suite members; explicitly advance notices, checkpoint acks, turn completion and queue operations. Cover steward self-ack, other-hub refusal, start idempotency, status read-only, queued dispatch holding/release, late join, replacement, partial pause/resume and cancellation races.
- [ ] Write `TestDrainInterruptOwnership` and recorded-screen controls: busy supported turn, idle, picker, buffered input, old wrapper, superseded incarnation, operator input race, lost write reply and duplicate request. Missing/protocol-extra fields fail closed; known allowed raw payload is Esc only.
- [ ] Run targeted tests across couchcmd/couchmessage/wrapcmd; verify RED then GREEN through the actual service/endpoint seam. Mutate ack, identity, queue and reserve guards independently and demonstrate each safety oracle fails. Commit.

### 4. Board UI and operational documentation

Files: `couchtty/console_drain.go`, console/status/tab projection and tests; `README.md`, `atlas/couch.md`, embedded Couch skill/help. Link any new atlas file from index.

- [ ] Publish immutable board snapshots through a console event; render per-slot draining/drained/stuck plus global pause/unknown reason. Keep reasons in CLI/JSON status, sanitize terminal text and preserve selected pane, narrow-width state visibility and existing picker indicators.
- [ ] Add `TestDrainConsoleProjection` and CLI/parser/help contract tests. Exercise malformed/duplicate flags, excessive reason/deadline, all mode combinations, JSON bounds, no live probes on status rendering. Run targeted tests and commit docs with behavior.

### 5. Real seam conformance, suite, review and landing

- [ ] Once #481 lands, run real queue adapter conformance under its isolated state root: two owners, graceful running work, fast interruption evidence, lost-response reconciliation, FIFO resume and exact JSON contract. No production pause/drain as a test.
- [ ] Run relevant race tests and headless three-wrapper integration with real endpoint transport and isolated namespace. Assert board transitions and no extra messages/input. Stage the build through `sdlc stage --build` only after hub authorization; send loaded revisions and arrange operator observation of board rendering. Any real fleet drain/fast interruption needs a fresh explicit steward/operator request, not a test that surprises working slots.
- [ ] Push with `sdlc pr`, queue `sdlc local-check run --name suite` on the pushed head, preserve every failure, then `sdlc close --issue 722 --verified '<returned evidence>'`. Binary owns the single fresh-context close review; fix classes and update lessons.
- [ ] On SHIP run `sdlc merge --yes`, respect parked #723/#622 and landing gates, clear any owned stage, verify actual PR landing and report outcome to hub. No regenerated workflow files in this PR (#699).

## Architecture lens summary

ARCH-DRY: current steward/broker/registry/input ownership and one board/notice authority. ARCH-PURE: transition/board/decoder rules use values; IO returns typed events. ARCH-PURPOSE: both drain modes, timeout, cancel and truthful UI ship together; real #481 conformance is required. ARCH-MOCK: stateful queue, wrapper/fleet and portable persistence seams. ARCH-CONSTRAINTS: bounded membership/effects/status polling and counted growth. ARCH-SECURE: authenticated steward/member and exact incarnation, no assertion of operator authority. ARCH-STATE: generation/owner lease and late/unknown/cancel sequences. ARCH-FUNERAL: one record, one coordinator, bounded current outcomes, no new growing log. ARCH-EVOLVE: additive preserved unknowns, strict known keys and unsafe-version hold.

## Revisions

### 2026-10-19 — advisory sequencing corrections (governing)

One fresh-context advisory review identified three material ambiguities in the initial draft above. These requirements supersede its queue-head, interrupt/notice ordering and cancellation wording; they add no new user actions or queue argv/schema. No code exists yet.

1. **Drain control bypasses held peer FIFO structurally.** Existing Broker.promote selects only ActorState.QueueHead, so merely allowing SendSystem while ordinary delivery is held would strand a notice behind an older peer. Give each actor one typed current-generation drain-control entry, selected ahead of the held ordinary queue. Only the internal coordinator can create this entry; OriginSystem alone is insufficient. Preserve the original order of all ordinary peer messages for cancellation. Centralize eligibility selection and use it in reserve, commit/revalidation, batching and retained-receipt reconciliation, not only the initial promotion. Do not batch a drain control with ordinary peer messages. Test `TestDrainNoticeBypassesHeldPeerFIFO` with pre-existing queued peers and an idle target, then cancel and require the original peer FIFO order.

2. **Fast interrupt precedes checkpoint notice and names the original turn.** Freeze exact wrapper instance, admitted session and logical turn generation before fast interruption. The endpoint observation exposes a monotonic wrapper-owned turn generation (updated under the existing lifecycle/input ownership), not a guessed timestamp or only a busy boolean. Recheck that generation and active-turn predicate atomically immediately before the single Esc. If the original turn ended or changed, skip the Esc; do not interrupt the replacement turn. Complete or reconcile the original interrupt outcome before enabling this member's drain-notice delivery. Unknown interruption keeps the row stuck and the notice held. Only after known interrupt completion/skip may the notice start the checkpoint turn. Test `TestDrainInterruptCannotHitCheckpointTurn`, including a delayed effect, ordinary turn completion, operator submission and a replacement turn; assert zero Esc to the new turn. The same protocol capability retains outcome by drain/member ID for reconciliation without reissuing input.

3. **Cancel revokes control work before release.** Cancel invalidates the generation, removes its queued broker control entries and revokes any uncommitted wrapper reservation through the same automatic-input lease used for the final delivery commit. Revoke returns known not-committed, already-committed with retained outcome, or unknown; it never claims a paste can be undone. Already-committed/submitted notices are recorded as such and their effects are not rolled back. Keep Couch dispatch held until all admitted control delivery/interrupt effects have known terminal dispositions and this queue owner is confirmed released. An uncertain revoke shows stuck:cancellation unconfirmed; retry cancel reconciles it without another paste/Esc. Do not allow a new generation while any old handoff remains unknown. Test `TestDrainCancelRevokesNotice`, covering cancel-before-reserve, reserve-before-commit, already committed, lost reply after commit, and immediate cancel/start with a delayed old completion. A stale ack cannot satisfy a new generation.

Task3 includes the production broker/endpoint transition changes and all three tests. Portable persistence keeps only the current generation's pending control identities; there is no growing retired-generation ledger. These are ARCH-STATE/ARCH-DRY corrections to the single control authority, rather than three independent mechanisms.
