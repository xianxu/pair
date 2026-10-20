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

- [x] Test `ReduceDrain` and `ProjectDrainBoard` with generated, controllably reordered event sequences and independent no-false-drained/owner-isolation invariants; guard each input authority separately with a mutation. Test `DrainNotice` with literal expected v1 output and arbitrary bounded reason data. Count reducer/effect operations at N and 2N to reject repeated fleet scans or per-member queue probes.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestDrain' -count=1`; confirm RED for missing contract, implement the transition/formatter, then GREEN. Commit the self-contained model.

### 2. Queue seam, portable persistence and fake

Files: new `couchcmd/drain_queue*.go`, `drain_store.go` and tests.

- [x] Test `sdlcDrainQueue.Pause`, `.Resume` and `.Status` against the stateful owner/activity fake and executable argv boundary; reorder effect completion versus lost response and independently assert owner isolation. Fuzz `decodeQueueSnapshot` with malformed/oversized JSON and enforce complete-known evidence before absence inference. Test `drainStore.Read`/`.Write` with filesystem failure injection and unknown-field round trips; assert durable intent never vanishes after an uncertain effect.
- [x] Run `go test ./cmd/internal/couchcmd -run 'DrainQueue|DrainStore' -count=1`; demonstrate RED, implement adapter/decoder/storage, then GREEN. Missing #481 binary support must produce unknown/unsupported. Commit.

### 3. Broker/coordinator, authenticated ack and fast interruption

Files: new `drain_service.go`, `drain_cli.go`, `drain_protocol.go`, `drain_endpoint.go`, `wrapcmd/drain_interrupt.go`; named existing wiring files above plus Registry.Liveness's submission observation if required.

- [x] Test `drainCoordinator.start`, `.ack`, `.cancel` and `.reconcile` through the real service with a stateful fleet, portable store and queue fake; control effect order and independently assert exact-generation completion and no early admission release. Test broker `eligibleDrainHead` and `revokeDrainControl` against the actor transition model with held peer FIFO; guard every reserve/commit path with the same generation authority.
- [x] Test wrapper `ReduceDrainInterrupt` and `executeDrainInterrupt` with controllable original-turn/input-lease ordering and recorded-screen evidence; independently count Esc writes and reject writes to any newer turn. Fuzz `ValidateDrainRequest` and endpoint payload validation over malformed/extra fields; the only accepted input effect is the fenced Esc.
- [x] Run targeted tests across couchcmd/couchmessage/wrapcmd; verify RED then GREEN through the actual service/endpoint seam. Mutate ack, identity, queue and reserve guards independently and demonstrate each safety oracle fails. Commit.

### 4. Board UI and operational documentation

Files: `couchtty/console_drain.go`, console/status/tab projection and tests; `README.md`, `atlas/couch.md`, embedded Couch skill/help. Link any new atlas file from index.

- [x] Publish immutable board snapshots through a console event; render per-slot draining/drained/stuck plus global pause/unknown reason. Keep reasons in CLI/JSON status, sanitize terminal text and preserve selected pane, narrow-width state visibility and existing picker indicators.
- [x] Test `Console.SetDrainBoard` and the tab/status renderer with immutable board values and independently measured terminal bounds; assert rendering invokes no live probe. Fuzz `parseDrainCLI` with arbitrary argv and enforce one closed operation shape and configured bounds. Literal help/notice contract checks guard public wording. Run targeted tests and commit docs with behavior.

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

### 2026-10-19 — TL design approval

Canonical submitted TL receipt 0a9b7377-14f4-444f-83ce-3cf1cb86353e approves this plan and instructs change-code. Async start, 10m default/1s–24h deadline, one persisted owner intent, confirmed-pause-before-notice, owner-only cancellation and no timeout release are approved. Visible states are exactly draining, stuck (with reason), drained and cancelled; the table's “inactive: cancelled” is rendered cancelled. Each uses the recovery listed in the table. TL copied the exact notice and queue seam into ariadne#481; real #481 conformance remains the production enablement/close dependency. Execute the approved full flow; the code surface is outside the quick-flow shell even though this design is under its line limit.

### 2026-10-19 — PQ-1 function-level verification strategy

Plan-quality round1 accepted the architecture/ordering but rejected the task sections' prose test-case inventories. Replaced those inventories throughout tasks1–4 with named production functions, an adversarial-input/controllable-ordering strategy and an independent mechanical guard. Governing state/ordering requirements remain requirements rather than a test-case pre-image. This is a verification-description correction, not a change to the approved notice, SDLC seam, visible states or scope.

### 2026-10-20 — re-plan proposal: bounded wrapper generation ownership (awaiting TL approval)

The implemented wrapper retains interrupt receipts and cancellation tombstones in two maps capped at 128 IDs per wrapper incarnation. An isolated probe confirmed that cancellation64 retains128 revocation IDs, cancellation65/66 return unknown, and retrying65 cannot recover. This safely refuses input but can strand a real queue owner. Merely enlarging the cap postpones the same failure; evicting old IDs loses the late-input fence; relaunching a wrapper cannot establish the old incarnation's uncertain effects. These alternatives do not satisfy the approved bounded-current-outcome contract. Do not implement the following replacement until TL approval and a renewed change-code plan gate.

**Revised architecture:** replace the retired-ID lifetime ledger with one monotonically ordered drain generation per wrapper. The existing namespace intent is the sole durable epoch allocator; the wrapper's existing input-admission lease owns its high-water mark and current-generation outcomes. Late lower-generation calls cannot acquire input authority even after their IDs have been discarded (ARCH-STATE, ARCH-FUNERAL, ARCH-ORDER).

| Pure concept | Location | Change |
|---|---|---|
| Typed drain fence: generation UUID, positive uint64 epoch, expected wrapper instance | couchmessage drain protocol/model | new validated value shared by message and endpoint controls |
| Persisted generation epoch | couchcmd drain_service_record.go | modified single intent record; allocate prior epoch+1 for a new generation, never for a duplicate start |
| Wrapper current generation/high-water mark | wrapcmd drain_interrupt.go | modified lease-owned state; current outcomes only |

| Integration | Change |
|---|---|
| Coordinator/store | Persist epoch with intent before pause/input; preserve it across restart and owner-only cancellation; reject overflow before effects |
| Broker/RemoteEndpoint | Carry the same typed fence on notice, interrupt, interrupt-status and revoke; validate returned authority |
| Wrapper input owner | Check instance and epoch under the existing input lease before accepting controls and at every final Esc/paste/submit; advance to a higher epoch atomically and discard retired-generation maps |
| Stateful fleet/transport fakes | Model the same epoch authority, not a separate permissive test path |

One epoch belongs to exactly one generation UUID. Same epoch/different UUID is refused; lower epochs cannot revive old controls or mutate the current generation. A higher epoch retires old effect authority under the input lease before admitting the new control. Pending callback objects retain their frozen fence and revalidate it even if their map entry was retired. Current-generation ID capacity remains bounded as a defensive malformed-producer limit, not a lifetime count of successful drains. The monotonic counter may reset only with a new wrapper incarnation; namespace record loss/reset with live wrappers fails closed rather than manufacturing a new low epoch. Do not add a cleanup timer, retired-generation journal, automatic relaunch, or new queue owner.

An independent fresh-context wrapper read confirmed the capacity reproduction and adds these required fences: bind the epoch to the authenticated endpoint namespace as well as generation UUID; negotiate the capability so older wrappers refuse. Every control outcome echoes namespace/generation/epoch and wrapper instance, which broker/coordinator verify. Read-only observation/status never advances the watermark. Retire old outcome records only after their cancellation dispositions are known; a high-water mark proves no future old write, not that prior input never happened. Advancing an epoch must preserve existing automatic-input quarantine and any composer residue. Test these independently through reservation, enqueue, batching/finalization and every final input write.

Public CLI, UUID acknowledgements, exact notices, #481 argv/JSON, four visible states, timeout and cancellation ordering remain unchanged. Persisted intent becomes schema_version2: an old reader ignoring the new epoch could revive retired controls, so it must refuse the new version. Version1/unsafe epoch data is held/unsupported by the new reader, with no automatic migration of possibly active effects. Additive unknown fields still survive normal version2 rewrites. Wrapper capability negotiation and strict typed fences likewise prevent silent downgrade. No new operator action or live experiment is proposed.

Actions remain wait, inspect status, checkpoint/ack, retry cancel and cancel then start.

| Internal condition | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Valid next epoch / duplicate current epoch+UUID | Existing draining/drained projection | Existing action | None beyond the approved checkpoint contract |
| Late retired call | Cannot grant input or alter current board | Inspect status if current work is affected | Retired receipt detail may be discarded; retired effects cannot be replayed |
| Conflicting UUID, missing/unsafe epoch, storage reset or overflow | Stuck/refused with exact generation reason | Inspect status | No claimed safe restart or input authority |
| Current generation cancellation | Existing cancelling/cancelled projection after control disposition and owner release | Retry cancel | Completed input is not rolled back |

Verification and one close boundary:

- [x] Test pure fence validation and epoch allocation with reordered/duplicate/conflicting values and overflow; independently assert no lower epoch can grant authority and no duplicate start increments the epoch.
- [x] Test wrapper control acceptance plus final Esc/paste/submit through its stateful input owner, keeping old callbacks delayed while generations advance. Count retained maps at N and2N generations (including more than128) and input writes independently; assert constant current residue and no old-generation writes after retirement.
- [ ] Test coordinator/store/broker plus actual RemoteEndpoint transport across repeated graceful/fast/cancel/recover cycles. Require cancellation beyond the former cap, exact returned fence checks and unchanged owner-only queue ordering; preserve all prior unknown-effect and incarnation tests.
- [ ] Update atlas/operating envelope, run focused/race/fuzz and repository inventory/catalog checks, checkpoint, push, then record a fresh full suite on the changed code. Real #481 conformance, operator live board acceptance and the mandatory close review remain required before landing.

Growth drivers: wrapper input events perform O(1) generation checks; retained current controls are bounded independently of total drain count. A generation advance discards only the fixed current map. Coordinator generation allocation is O(1); existing O(N) fanout and one shared queue poll remain. No new idle polling or subprocesses.

### 2026-10-20 — revised mechanism approved; counted lifetime test

Canonical TL receipt68935e6f approves the monotonic epoch/current outcome/input-lease watermark mechanism above, with the public/#481 contracts unchanged, and requires renewed change-code. Its two asks are incorporated here before implementation.

Simplest rejected alternative: refuse a new drain until tombstones are reclaimed; without a generation fence reclaiming permits late input, and refusing forever replaces bounded operation with a lifetime availability limit.

The wrapper resource test must run at least1000 start/cancel cycles (and2000 for its2N comparison), count peak/current retained control outcomes and final input effects, and require a fixed bound independent of total cycles. This is a counted state-growth oracle rather than a timing assertion. The prior cap horizon must fail the test before the epoch fix. A pure transition component owns generation acceptance/retirement; the wrapper shell performs its declared effects under the existing input lease rather than mutating authoritative generation fields directly (ARCH-PURE/ARCH-ORDER). Test read-only status, stale/conflicting epoch, delayed callbacks, recovery and preserved quarantine independently.

### 2026-10-20 — preliminary real #481 consumer evidence

Epoch implementation,1000/2000-cycle counted tests, protocol/wrapper/service/transport races and the recorded8m40s suite have passed (issue Log has exact heads/evidence). Before #481 landing, its owner supplied an M2 executable and isolated queue cohort. Production adapter/strict decoder conformance passed against b39a97b35386 for two-owner/replay/worker refusal, all active kinds, interruption retirement, unknown/corrupt residue and process-boundary lost-response reconciliation. Temporary probes leave production code unchanged. This is preliminary real-seam evidence, not a revision of acceptance: final landed #481/M3 admission/FIFO/process coverage and the authorized operator-present board check still precede close. No live queue/drain or stage was exercised.

### 2026-10-20 — approved review-first ordering and minimal live proposal

TL receipt f35727b5 orders close review now and operator-present live acceptance afterwards, following landed/rebuilt #481. This supersedes the prior live-before-close sequence. No acceptance criterion is dropped. Keep the live and landing checkboxes open; close may acknowledge only the unchecked-plan mechanical guard so its binary-owned fresh review can run now. SHIP does not authorize landing before this live check. Pair rollout belongs to the TL's next pair:0 rebuild and Couch restart; this slot neither stages nor mutates the real fleet without that scheduling.

Proposed single graceful acceptance, for the TL/operator to schedule:

1. Confirm the intended Pair build/supervisor and current wrapper capability after the scheduled rollout. At a quiet moment, read `sdlc queue status --json`: require complete evidence and no active suite, merge or landing; record existing pause owners. Existing unknown legacy owners must settle normally or be explicitly reconciled, never ignored to claim quiet.
2. Steward starts `couch --drain --reason 'pair#722 live acceptance' --timeout 10m --json` and records the returned ID. Observe draining; verify `sdlc queue status --json` reports the exact `couch-drain:{ID}` owner and paused state. No fast interrupt is part of this check.
3. Every admitted slot follows the notice: finish its atomic checkpoint, commit/push and log where it stopped, run `couch --drain --ack ID`, then end its turn idle. Steward is included and acks last. The operator observes all current rows/global board reach drained with fresh idle and complete queue quiescence. Read-only queue observation can be made without starting a new agent turn; starting a new acknowledged turn would invalidate that ack.
4. After observing drained, the operator explicitly asks the steward to cancel. Starting that turn may invalidate the steward's prior ack, which does not prevent cancellation. Steward runs `couch --drain --cancel --json`; observe cancelled, the matching owner absent from queue status, any original owners preserved, and ordinary dispatch restored. A single harmless dispatch can verify delivery after cancellation if the TL includes it in the scheduled check.
5. Record build identities, drain ID, observed board/queue transitions and the operator's rendering/selection attestation in the issue Log. Only then proceed through the remaining merge gates. Timeout, unknown or missing ack remains an honest stuck result; inspect/cancel explicitly, with no claimed restart permission.

### 2026-10-20 06:25 PDT — close round 1 corrections

The binary-owned close review returned REWORK (BR-1 acknowledgement-observation-ordering, Critical; BR-2 external-dependency-conformance, Important). Actual measurement succeeded after the hub repaired provenance. The queued combined-head suite was cancelled before execution because these corrections change code. No staging or live effect has occurred.

BR-1 preserves the approved checkpoint meaning: sample authoritative registry submission and identity-fenced wrapper input at ack time, carry that activity baseline into the pure acknowledgement event, and require a subsequent observation before sealing idle. Discard observations started before that ack using the existing ack revision fence. Enumerate registry activity, wrapper input, pre-ack delayed observation, post-ack activity, duplicate ack, missing evidence and identity/generation replacement; test each at the reducer/service seam (ARCH-ORDER/ARCH-PURPOSE). This corrects evidence ordering without a new public state or control mechanism.

BR-2 retains an opt-in real executable conformance runner through the production adapter/decoder, with newly created private queue/capacity/fleet roots only. Cover two-owner isolation, complete versus unknown observations, active suite/merge/landing retirement/interruption, worker authority and lost-response reconciliation. Document execution on queue-contract/dependency changes and before closing drain adapter changes (ARCH-MOCK); normal tests continue to use the stateful fake. Live acceptance and landing remain open, and staging remains conditional on SHIP.

### 2026-10-20 06:37 PDT — approved fast-mode Spec revision (BR-3)

Canonical submitted TL receipt e881ba7d-158e-4a5a-812f-335f0a0fb030 explicitly approves this revision, superseding the original Spec promise that fast drain cancels merges. Fast cancels running suites and records them interrupted for a later explicit rerun. An admitted merge or landing finishes or retires at its own safe boundary; killing integration or a push is riskier than waiting. The landed #481 admission gate already stops a merge at its next suite boundary. Fast never authorizes killing a merge or landing process. The board remains draining with named active work until authoritative complete active=[] and all member checkpoint/idle conditions hold; timeout remains stuck without release.

| Queue operation | Fast pause | Completion evidence |
| --- | --- | --- |
| suite | Request interruption; preserve interrupted outcome; no restart on resume | Suite retires, no active row |
| merge | Preserve running work; finish/retire at its safe boundary, including the next suite admission boundary | Merge retires, no active row |
| landing | Preserve running work; finish/retire at its safe boundary | Landing retires, no active row |

The fake must match the real seam across this complete kind enumeration and owner replay/resume; the retained real conformance already asserts the suite-only interruption policy. Add fake and coordinator regressions that require live merge/landing rows to keep the board draining even after every agent has acked/settled and the suite retired. The public CLI, notice text, JSON schema, owner-scoped hold and approved graceful-only live check are unchanged. BR-1 and BR-2 were disposed addressed by round2, including reviewer-run mutation controls and real conformance; only BR-3 remains. Cancelled this slot's still-queued prior-head suite because the fake corrections change the tested head. No stage or live effect occurred.

### 2026-10-20 06:48 PDT — approved transition-authority correction (BR-4)

Canonical submitted TL receipt e9934f5f-75ed-411a-a949-86a7a8273a05 approves the correction. Every authoritative phase/error update is reducer-owned. Add a generation-fenced broker-release result: only queue-release-finished intent accepts it; failed/unknown release returns to the existing cancelling/error recovery, explicit cancel retries the existing owner-release sequence, and successful release clears the failure. Active intent, stale ID and unrelated outcomes cannot change it. The coordinator applies results only to its exact current unreleased intent and never marks a replacement intent released. Storage, dispatch-hold and member-cap failures use DrainFailed rather than direct Error writes. A local Go AST catalog rule (pair#722) rejects direct coordinator phase/error assignment and common aliases; constructors and whole-state reducer results remain allowed. This enumerates existing production failure paths without changing public states, CLI, persisted schema or the #481 seam. Verify release failure/retry/stale outcomes in reducer/service tests, and exercise the lint's positive/negative fixtures and actual repository scan before another close. The round3 FIX-THEN-SHIP verdict was not finalized because the ledger correctly retained this Important blocker; do not bypass it. Live/stage/landing holds remain unchanged.

Broker-release failure also invalidates the prior queue observation: retry needs fresh queue evidence. A storage error after queue release stays visible as stuck instead of being hidden by the terminal projection. The public states/recovery actions remain the approved set.

### 2026-10-20 11:07 PDT — failed fleet acceptance: recovery-first re-plan (awaiting TL approval)

This Revision supersedes the original early dispatch hold, all-wrapper cancellation prerequisite, recovered-record hold and repeated per-chip drain projection. Prior implementation/review/suite evidence remains historical; it does not satisfy this revised acceptance. TL dispatch31d0d041 explicitly requires re-planning before code. The failed live run is not a successful drain or a successful live check.

#### Evidence and remaining uncertainty

Read the operator record's “What happened” and “After the reboot” in ops/teams/devinfra/tl-status-xian-xu.side/2026-10-20-09-drain-for-reboot.md. Observed: checkpoint acknowledgement refused; ordinary messages blocked before checkpoint confirmation; cancellation never confirmed, including after reboot; quit/detach blocked; repeated status/reason overwhelmed the tab bar. The operator preserved drain.json.wedged-1020; treat it read-only. Its SHA256 is7d479da411031c27440020854d452b55978f78b893a04cb4db6a84c0bedcc691. It contains epoch1, cancelling/unreleased, ResumeIssued=false,27 members, zero acks and zero NoticeStarted/InterruptStarted flags. Seventeen rows have unknown cancel observations and ten have changed-wrapper outcomes.

Source confirms: start holds the broker before queue/notice/ack; preparing requires complete known own-pause evidence before sending notices; cancellation unnecessarily revokes even controls never started and waits for all wrappers before queue release; recovery restores the hold; the broad process guard blocks detach; tab rendering appends a state to every chip. The ariadne:4 checkpoint request attested by the TL is not explained by this saved record's zero notice-start flags. Do not claim identity corruption or no delivered request: obtain generation/caller/receipt evidence in the disposable reproduction. Acknowledgement's broad error currently combines several predicates and must become diagnostic without weakening authentication.

#### Chosen design and alternatives

Keep the pure reducer, exact caller/wrapper identity and epoch fences, owner-scoped SDLC seam, bounded intent, authenticated worker ack and four public states. Change the authority of message suppression: a volatile broker hold is acquired only after every current member has acknowledged its durable checkpoint and fresh idle/no-background observations plus a complete quiet paused queue establish readiness. Before that barrier, ordinary coordination continues. Delivery or activity invalidates the corresponding ack; an invalidated barrier releases its hold and returns to checkpoint collection. Drain state never owns quit, detach, status, ack, cancel, receipt lookup or the tab bar.

Alternatives rejected: (1) patch only the ack predicate and keep the early global hold—unknown queue/wrapper evidence still makes recovery unreachable; (2) remove all coordination holds permanently—usable, but cannot stabilize the final observed checkpoint boundary against newly admitted messages. The selected late hold has one lifetime owner, the active supervisor's generation-scoped broker lease, and cancellation/shutdown closes it locally without consulting wrappers. This replaces a distributed absence proof with one owned resource (ARCH-STATE/ORDER).

Start performs a read-only queue preflight before persisting a new intent or adding a pause owner: require schema-valid known complete running status and no unknown active operation. Healthy running suites/merges/landings are allowed; unknown/unmanaged work, incomplete status, another pause owner or failed observation refuses start with the exact reason. No notices, Esc, pause or message hold follow a refusal. Preflight does not prove later quiescence; after intent persistence, pause and subsequent snapshots still establish the active barrier. Preflight all member capabilities/identities too; unsupported members produce a refusal rather than an unrecoverable half-start.

Cancel first revokes the local generation's authority to issue new controls and releases ordinary delivery. It cancels queued uncommitted notices, joins in-flight local effects under their existing deadline, and records uncertain already-handed effects as warnings. Releasing this drain's SDLC owner does not depend on wrapper observation, ack, or revocation response. Reconcile uncertain pause/resume through the unchanged owner-scoped status seam; never remove another owner. A failed/unconfirmed queue release is visibly stuck with retry-cancel recovery, while messages, quit and detach stay available. A cancel with nothing active is an idempotent no-op. Already committed checkpoint notices or one Esc cannot be undone; retain that limitation explicitly, never claim drained from cancellation. No automatic replay of a notice or Esc occurs after cancel/restart. A new start refuses unresolved old control authority rather than silently adopting it; exact old-wrapper death/replacement or a confirmed endpoint disposition can retire that uncertainty without input to a new wrapper.

On supervisor or whole-fleet restart, load the bounded intent for diagnosis/owner reconciliation only: discard all inherited ack/idle proof and never reconstruct a broker hold from disk. Recovered state permits ordinary coordination, quit/detach and explicit cancel immediately. Reconciliation releases only the retained owner; no old-wrapper liveness prerequisite. Preserve uncertain outcomes and extension fields; no automatic notice/interrupt replay or automatic work restart. Keep epoch monotonicity and stale-result fences. Persistence errors must report retained ownership and permitted recovery rather than install a synthetic global hold. All authoritative phase/error transitions remain reducer-owned and covered by the existing AST lint (ARCH-PURE/SECURE/EVOLVE/FUNERAL).

#### Revised concept map and state table

These rows replace the affected original Core concepts entries; original rows remain provenance.

| Pure entity | Lives in | Status |
|---|---|---|
| DrainState/events/reducer and checkpoint-barrier predicate | cmd/internal/couchcore/drain.go | modified |
| DrainBoard/row projection, cancellation warnings | cmd/internal/couchcore/drain_board.go | modified |
| Concise drain summary, no per-actor decoration | cmd/internal/couchtty/console_drain.go and reserve.go | modified |

One state owns one generation and at most128 exact members. The barrier is derived from reducer state; it is not a second mutable readiness flag. The concise summary derives from the board. No filesystem or endpoint calls belong in these functions.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Coordinator preflight/cancel/recovery and intent store | cmd/internal/couchcmd/drain_service.go, drain_store.go | modified | queue, persistence, exact fleet and effect lifetimes |
| Generation-scoped broker hold and pending control cancellation | cmd/internal/couchmessage/drain_broker.go | modified | actual message admission/delivery |
| Explicit launch-only drain guard | cmd/internal/couchcmd/drain_wiring.go and hubs.go | modified | process-operation admission; escape operations bypass it |
| Disposable installed-Couch fleet harness | cmd/couch/drain_acceptance_test.go and testdata/drain fixture helpers | new | real Couch/Pair/wrapper/CLI/PTY, isolated stateful executable queue |

Actions: wait; checkpoint-and-ack; inspect status; retry cancel; cancel then explicitly resume work. Ordinary sends, status, cancel, quit and detach remain available throughout collection, failure and recovery; the final readiness hold affects ordinary dispatch only.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Preflight unknown/incomplete/paused queue or unsupported member | Start refused; no active drain | inspect status | no drain effect; existing work continues |
| Intent persisted, pause pending/unknown | draining or stuck with queue reason; messages open | retry cancel | pause may have applied; never safe-restart proof |
| Pause known, checkpoint notices/acks/idle still pending | draining; missing/unknown member reason | checkpoint-and-ack | fast can abandon in-flight reasoning; durable edits retained |
| Missing ack, replaced/disconnected member, stale observation or deadline | stuck with bounded summary | inspect status | no checkpoint proof; ordinary coordination remains usable |
| Every exact checkpoint/idle confirmed; queue paused/complete/quiet; final local hold confirmed | drained | wait | worker attestation, not independent Git inspection |
| Activity/handed message invalidates checkpoint/barrier | draining; hold released | checkpoint-and-ack | later activity lies outside earlier checkpoint |
| Cancel accepted; local authority revoked; queue release pending/unknown | draining or stuck: releasing own pause; messages open | retry cancel | committed notice/Esc may complete; no new control permitted |
| Own owner absent, irrespective of old wrapper availability | cancelled; other owners/warnings shown in status | explicitly resume work | interrupted suites require explicit rerun |
| Restart/corrupt record/old in-flight outcomes | stuck: recovered/unknown; messages and escape paths open | retry cancel | no inherited all-drained authority; preserve uncertain owner/control facts |

The transition tests enumerate cancellation before pause, pause reply lost after effect, late pause after cancel, notice handed before cancel, old-wrapper disconnect/replacement, stale result after a new epoch, store-write failure and reboot in each phase. Cancellation is bounded local revocation, followed by owner reconciliation; it is not proof that a remote action never committed. No new public CLI, queue argv/schema or worker notice/ack contract is proposed.

#### Disposable Couch acceptance harness and test strategy

Reuse cmd/couch/main_test.go installedEnv/installedExecutables and TestBareCouchInstalledCommand, and the existing native terminal fixture/registration handshake. Run a real supervisor with one steward plus two independently admitted throwaway workers. The installed binary, CLI, actual caller identity, real registration/delivery, real wrapper drain endpoints and PTY rendering are in scope; fake service endpoints alone are insufficient. Extend the controlled agent fixture to checkpoint a marker and invoke the public ack CLI from its genuine admitted environment. When native Zellij behavior is material to quit/detach, reuse pairlifecycletest.StartControlledZellijWithOptions in the same disposable fleet; do not substitute handler mocks for the public lifecycle action.

Each case owns a new root with COUCH_ISOLATED_ROOT, COUCH_STORE_DIR, COUCH_IDENTITY_DIR, PAIR_DATA_DIR, private HOME/XDG paths, ZELLIJ_SOCKET_DIR/config/runtime, and private WF_SDLC_QUEUE_ROOT/WF_TEST_LOCK_FILE/WF_SDLC_QUEUE_FLEET_ROOT. Refuse empty or non-owned paths and inherited live COUCH_TREE/binding/socket references before launching; build fixture identities from its real disposable repositories. A stateful executable SDLC fixture implements the existing exact production argv/schema, pause-owner isolation, active operation kinds and controllable lost replies. Retain the opt-in real SDLC conformance runner on newly created private roots. Couch message sockets are hash-disjoint by private canonical namespace under the existing per-UID socket parent; no existing socket-root override is claimed. Cleanup verifies and removes only exact owned fixture socket/session/process identities, joining all children; no glob cleanup or process-name killing. Prefer TestScratch when available, with ordinary TMPDIR fallback. No production drain, queue pause or slot lifecycle operation is used.

| Public production path / proposed named test | Adversarial strategy and independent oracle |
|---|---|
| TestDisposableCouchDrainGraceful | Hold each worker ack in turn. Confirm real system receipt/generation and CLI ack success; ordinary sends succeed before the barrier. Both worker durable markers and steward-last ack precede drained. New activity invalidates it; cancel restores dispatch. |
| TestDisposableCouchDrainFast | Real wrapper active-turn fixture records at most one interrupt; isolated suite retires interrupted, admitted merge/landing survives until safe retirement. No early drained and no automatic restart on cancel. |
| TestDisposableCouchDrainRefusesStuckQueue | Unknown/incomplete/legacy-active/other-paused status refuses before intent/pause/notices/Esc/hold. Independent ordinary message succeeds; compare owner/state bytes before and after. |
| TestDisposableCouchDrainCancel | Cancel before any notice, during held handoff, after ack, with missing/replaced/disconnected wrapper, unknown status and lost mutation reply. Observe bounded return and ordinary-send/quit/detach usability independent of remote revoke. Only own owner is removed; late callbacks cannot rehold dispatch or issue controls. |
| TestDisposableCouchDrainReboot | Restart supervisor on the same private state with retained and replaced workers, including crash after intent/pause/control/cancel. Old ack never seals; no notice/Esc replay; messages/quit/detach work before old-wrapper observation; public cancel reconciles only its owner. |
| TestDisposableCouchDrainEscapeAndTabs | Exercise public quit and detach in preparing, active, stuck, cancelling and recovered states. Capture narrow/wide terminal frames: one bounded drain badge, unchanged selectable chips, no repeated full reason or per-chip suffix. Detailed reasons remain in --drain --status. |
| TestDisposableCouchDrainIsolation | Two concurrent fixture namespaces/queue roots cannot observe or cancel one another. Assert no live paths in argv/identity/manifests and no surviving fixture-owned writers after cleanup. |

Add predicate-specific ack refusal diagnostics and a production-path regression for each predicate: wrong ID, unadmitted/stale binding/session, unsupported/unknown protocol, absent notice submission, epoch/instance mismatch and non-active phase. Preserve the earlier ack-baseline ordering/alias/extension/race controls. Capture attempted ID, authoritative ID, canonical member, wrapper capability and predicate without dumping environment/secrets. The ariadne:4 failure is considered explained only after a matching production-path reproduction or exact retained receipt evidence.

#### Execution and acceptance order — no code until TL approval

- [ ] Reopen #722 through the SDLC setter from the owning slot after this Revision is approved; pass renewed full-flow change-code. Do not hand-edit the card mirror or reuse the former SHIP as approval of changed behavior. Derive the revised estimate after plan-quality, not now.
- [ ] Write the disposable harness and regressions first in cmd/couch/drain_acceptance_test.go using the existing installed fixture; demonstrate the original failures on the unchanged production behavior. Keep normal/fast/stuck/cancel/reboot fixtures and their JSON/PTY evidence. Run `go test ./cmd/couch -run '^TestDisposableCouchDrain' -count=1` with bounded fixture deadlines; expected RED for the affected old behaviors.
- [ ] Correct queue/member preflight, barrier ordering and detailed ack rejection through the pure model/coordinator, then local cancellation/owner reconciliation/restart. Exercise reducer/core and real endpoint tests before the full disposable matrix; expected no false drained, no early message hold, and cancellation independent of old wrappers. Commit each coherent correction with its tests.
- [ ] Replace the broad lifecycle guard with explicit launch admission; keep quit/detach/cancel/read-only paths unconditional. Replace per-chip drain state/reason repetition with one compact global badge capped by available width; detailed reasons only in status. Prove public actions and rendered frames through the disposable fleet.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/wrapcmd -run 'TestDrain|Test.*Drain' -count=1`, focused races, repository AST/artifact/slot-state audits, then the disposable installed-binary matrix. Run retained real SDLC adapter conformance with matching clean executable/source and private roots; no live pause.
- [ ] Update README, atlas/couch maps, couch skill and tl-worker shared-rule owner as needed for the approved ordering/recovery; coordinate cross-repo rule changes with TL rather than edit another issue's branch. Add prevention lesson: green component fakes do not qualify actual caller/registration/lifecycle composition. Keep whole-path independent oracles; no tests that merely mirror a phase assignment.
- [ ] Push and run the required recorded suite, then mandatory close review. Only after SHIP and all disposable cases pass: seek TL's operator-present live schedule and stage through sdlc. The live start refuses stuck queue before any effect; record actual ack, graceful/fast policy, cancel and escape outcomes. Failed live acceptance reopens the work; no merge before successful watched acceptance.

Budgets: retain128 members, fan-out4 and one queue-status operation per2s,5s per external operation. UI/status reads perform no IO. Quit/detach do not join drain workers or queue subprocesses. Cancel releases local dispatch authority immediately and bounds local effect joining by existing operation deadlines; queue failure remains inspectable/retryable. Count census/control/status operations at8/16 members and retain1000/2000-cycle bounded-state tests. The disposable three-member fixture has a per-case90s deadline and owned child cleanup; suite selection follows machine caps. Store one bounded current/last intent plus epoch watermark, never a new per-retry history. Preserve unknown fields on rewrite and retain the operator's wedged evidence untouched. Network/production workload throughput is N/A: no per-byte drain work; cost is O(members+active operations) per poll. These are proposed budgets, not measured acceptance results.

### 2026-10-20 11:09 PDT — TL approval and recorded opt-in acceptance matrix

Canonical submitted TL receipt71ddbf62 approves Revision11:07 and renews change-code after supported reopen. Two governing corrections: the seven installed-fleet cases are excluded from ordinary suite runs; pure and endpoint regressions remain in the normal suite. Declare a named opt-in local check `drain-acceptance` whose command is `make test-drain-acceptance`; that target enables `PAIR_DRAIN_ACCEPTANCE=1` and runs `go test ./cmd/couch -run '^TestDisposableCouchDrain' -count=1 -timeout 15m`. Ordinary tests skip these cases unless explicitly enabled. Run/record it through `sdlc local-check run --name drain-acceptance` on the pushed code head, require current-code evidence in the close verification, and recheck that recorded evidence before staging. If the repository local-check declaration is a local-gates change, request the operator's label through the hub and never self-label. This check is required for this issue's close and every future drain pre-stage step, not a machine-wide extra suite on unrelated changes. Update the helper/rules/runbook to enforce that pre-stage evidence rather than relying on historical prose.

The ariadne:4 discrepancy is a first-class acceptance row: preserve the reported delivered request and contradictory zero NoticeStarted flags, attempt reproduction through the installed caller/receipt/persistence path, and record the exact outcome. A non-reproduction must be stated explicitly in the close evidence and surfaced to TL for live scheduling; no claim of resolved cause or omission of this acceptance item is permitted. Watched live acceptance remains scheduled through TL with the operator present; plan approval is not live approval.

### 2026-10-20 11:23 PDT — approved recording dependency

TL5c8b8b29 confirms the separate local-gates boundary. Unclaimed #734 owns the optional main declaration of drain-acceptance; it is #722's dependency for named-check recording. Until #734 lands, run the named opt-in make target explicitly and retain exact pushed Git head, argv/environment isolation, exit, duration, complete output path and SHA256 in #722's Log by hand. This is the approved interim recording mechanism; it does not waive the matrix or pre-stage requirement. The harness/product target remains here, and no config/label is changed on this branch. Reopen/change-code from canonical pair:5 after #622's landing runner releases the checkout, as TL explicitly directs.


### 2026-10-20 11:36 PDT — HOLD pending operator's simpler contour

Canonical submitted TL receipt379c72ba supersedes the earlier permission to reopen/change-code: HOLD #722; do not reopen, enter change-code or implement the approved Revision. The operator is shaping a steward-driven in-memory per-slot flag and ack model with messages remaining available and no persisted drain state. This is a proposal, not an implementation contract; TL will return with what survives and what is dropped. Preserve current code and evidence unchanged meanwhile. The compact badge and ack diagnostics may survive, but are not independently dispatched. No product code changed after the failed live check, and the attempted auxiliary-worktree reopen never changed the card. Pair:0's stage was cleared/rebuilt earlier. #734's recording declaration was filed unclaimed before this hold; its relevance awaits the revised contour. Continue only #622's already-authorized landing in canonical pair:5.

### 2026-10-20 12:18 PDT — steward-driven transient flags (awaiting operator plan review)

**Authority and scope.** TL4160a300 conveys the operator-reviewed ten points and settled answers in ops/teams/devinfra/tl-status-xian-xu.side/2026-10-20-17-drain-redesign.md. This Revision replaces every earlier drain mechanism and acceptance proposal above. Keep this issue/branch, as the fresh dispatch directs; the side doc's earlier replacement-issue suggestion is superseded. The operator reviews this implementation plan through TL before code. #622's running train takes priority; do not switch its checkout or restart its runner to implement this plan.

**Purpose.** Couch marks requested worker posture and delivers ordinary system notices. The steward and agents judge progress. A flag does not attest checkpoint durability, idle, acknowledgement or permission to restart. There is no aggregate completion result. Agents' “acknowledge message received, will resume processing after drain is lifted” is ordinary prose, never parsed or counted by Couch.

**Public contract.** `couch --drain [--fast]` sets in-memory per-slot flags and sends notices. `couch --drain --status [--json]` reports flagged slots using existing cached activity/idle observations. `couch --undrain [--message TEXT] [--json]` clears flags and sends exactly the generic “drain lifted; continue your current work” followed by the optional steward note as data. Preserve conventional JSON output for these CLI requests. Remove old --ack, --cancel, --timeout and --reason grammar with a clear unsupported-option error pointing to current help. Start/lift authorization reuses the verified first-command-line-hub check; status is read-only for anyone. No outside-Couch mutation lane is added.

Graceful notice tells agents to finish/wait for their current background work, commit and push their branch, Log the exact next step, and stay idle. Fast notice tells agents to stop their own background work through supported owner-scoped controls, preserve unfinished edits, then checkpoint and stay idle. Both say keep receiving messages, acknowledge new work with the quoted ordinary reply, and defer processing until lifted. Couch sends no Esc or process signal. An optional completion reply is for the steward, not a machine gate. The worker rule must avoid acknowledgement loops: receipt/ack-only messages do not require another courtesy ack. The ariadne-owned tl-worker/tl-lead rule update owns the supported background-work stop and manual queue procedure; coordinate its landing with TL rather than edit a peer issue's unlanded branch.

The steward separately runs existing SDLC pause/resume, peeks slots/observes existing idle notices, and notifies the operator when it judges restart appropriate. Couch does not inspect or mutate the queue. Safe startup/restart-kind and worker startup notices are #739; global park qualification is #740. No automatic recovery, global park, scripts orchestrating the fleet, new idle sampler, drain file, persistent epoch, formal ack, completion barrier or message/lifecycle hold belongs here.

#### Concepts and implementation seams

These rows replace the original Core concepts; the previous tables are historical design, not implementation requirements.

| Pure entity | Lives in | Change |
|---|---|---|
| DrainFlags value and flag/clear/select helpers | cmd/internal/couchcore/drain.go | replace the old multi-phase reducer with transient set operations |
| DrainStatus/row projection | cmd/internal/couchcore/drain_board.go | replace completion board with requested flag and observed quiet/idle facts |
| Drain/undrain notice formatter | cmd/internal/couchcore/drain_notice.go | replace old v1 ack-command text with the operator contract |
| Compact badge projection | cmd/internal/couchtty/console_drain.go, reserve.go | one global width-bounded count; no actor suffix |

One immutable flag value contains at most Broker.MaxActors canonical slot names. Set/clear operations return copied values; a single service mutex owns updates. Observation snapshots and message receipts remain distinct inputs/results. There is no second readiness state or stored idle clock. Pure tests cover set operations, copy isolation, stable ordering, unknown observations, duration arithmetic and bounded badge formatting without IO mocks (ARCH-PURE/DRY).

| Integration | Lives in | Reuse / removal |
|---|---|---|
| Transient flag owner and authenticated request handling | cmd/internal/couchcmd/drain_wiring.go, message_service.go, drain_cli.go | messageService supervisor lifetime, switchCaller and first-hub authorization; no coordinator worker |
| Ordinary system delivery | existing couchmessage.Broker.SendSystem | existing route admission, sender provenance, FIFO and receipts; no drain-only transport |
| Cached status | existing Registry.Liveness/Connected and broker activity snapshots | existing LastWorkAt/LastActivity and nullable settled/picker/background observations; no filesystem/queue/subprocess work |
| Console publication | existing console event publication | one immutable snapshot/count; no drain-based operation guard |
| Disposable installed-Couch test | cmd/couch/drain_acceptance_test.go plus installed fixture helpers | actual public CLI, Pair/wrapper registration and PTY under private roots |

`Broker.SendSystem(ctx, verifiedSteward, requestID, Route{Target: slot}, body)` routes to the current actor at admission. A receipt identifies its actual recipient; do not pretend the earlier liveness snapshot pins that incarnation. Request IDs are ordinary message IDs, not drain generations. Flag changes are immediate posture changes, independent of notice outcome. Return per-target receipt/refusal and an overall snapshot; queued/submitted/refused never means checkpointed. Limit a request to the existing actor cap and a proposed five-second total notification-admission context; use existing broker queue/backpressure rather than waiting for delivery or adding a background coordinator. Any per-target failure remains inspectable and cannot block ordinary messages, quit or detach.

#### Proposed defaults for the operator's plan review

The ten points settle the mechanism but not these edges. These are explicit proposed defaults, not previously approved facts:

- Select currently admitted connected slots, including all hubs and the steward. Set flags on canonical slot names, not wrapper incarnations. A disconnect does not erase a flag; status then reports idle unavailable. A new/replacement wrapper does not inherit checkpoint proof because there is none.
- A later newly admitted slot starts unflagged. No persistent fleet-wide mode, automatic notice replay or special relaunch gate is added. The steward can issue drain again or message a slot; #739 owns generic startup behavior. Confirm this finite-cohort choice rather than silently inventing an inheritance protocol.
- Repeating drain is an explicit re-notification of currently connected slots, with the selected graceful/fast text; existing flags stay true. Fast is notice policy, not another stored lifecycle. CLI replies include ordinary delivery IDs so an uncertain invocation is inspected before an intentional repeat.
- Undrain snapshots all flagged slot names, clears them first, then attempts ordinary generic/custom notification to those targets. Disconnected/refused recipients stay cleared; report the delivery failure. With no flags it is a no-op and sends no unsolicited continuation. No automatic retry or reflag on error.
- Status describes observed quiet time using the same existing activity clock as supervision, plus current settled/picker/background evidence when available. Active is active, and unknown/zero/absent timestamp is unavailable; never synthesize zero idle or imply safe restart. Repeated status reads do not advance a stored observation or start a sampler.

#### State table and recovery

Actions: drain/re-notify; inspect status/receipts/peek; undrain; steward separately resumes work/queue. Quit and detach are unconditional ordinary operations, never recovery gates.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Unflagged | no drain badge/row | drain | none |
| Flagged; notice queued/submitted; actor active or quiet | one badge; per-slot observed status | inspect status/peek | no durability/readiness promise |
| Flagged; notice refused/unknown or actor disconnected | same requested flag; delivery/idle unavailable | inspect receipt, explicitly re-notify if appropriate | request may not have reached a worker |
| Undrain; continuation queued/refused/unknown | flags already cleared; ordinary delivery result | inspect receipt or explicitly message slot | committed notices cannot be undone; continuation may be missed |
| Couch stops/restarts while flagged or sending | next supervisor unflagged; normal delivery receipts remain normal evidence | steward inspects/re-notifies if needed | transient flags intentionally lost; no inherited posture/proof |

The only flag lifecycle is absent → present by drain, present → absent by undrain or supervisor death. Message outcomes never change that flag implicitly. Delayed old drain/undrain notices are ordinary queued messages; they may arrive after another request, so notice wording must identify it as a steward request and the steward can inspect receipts/send a correction. Do not reintroduce cross-wrapper tombstones, revocation or a completion protocol to hide this ordinary-message limitation (ARCH-STATE/SECURE).

#### Deletion and reuse tasks — one renewed close boundary

- [ ] After operator plan approval and after #622 completes: use the supported #722 reopen setter from canonical pair:5, then renewed full-flow change-code. Derive a new cumulative estimate only after plan-quality; former9.6185h/SHIP does not admit this redesign.
- [ ] Write cheap failing tests for the new public parser and pure set/status/notice behavior. Replace couchcore/drain.go, drain_board.go and drain_notice.go contracts. Test flags never mean “drained,” duration unknown stays unknown, repeated commands follow approved defaults, and copies cannot mutate authoritative state. Run `go test ./cmd/internal/couchcore -run 'TestDrain|TestUndrain' -count=1`; old behavior must fail the new oracle before replacement.
- [ ] Replace the coordinator with a small messageService-owned transient value and ordinary SendSystem handling. Delete drain_service.go, drain_service_record.go, drain_store.go, drain_queue*.go, their obsolete tests and testdata/drainqueue. Update CLI/request/response wiring in couchcmd/{drain_cli.go,drain_wiring.go,cli.go,messages.go,message_service.go,run.go} and couchmessage protocol shapes. Add service/public-route tests using actual inherited caller environment; no hand-constructed Binding as the acceptance path.
- [ ] Delete couchmessage/drain_broker*, drain_endpoint*, drain_generation* and the drain-only fields, fences and branches in broker/model/batch/endpoint code. Delete wrapcmd/drain_interrupt*, drain_epoch_test.go and drain-generation/input/reservation hooks in peer_delivery.go, peer_batch.go and peer_runtime.go. Remove unused registry Submission barrier fields only after a consumer search; retain independent activity/liveness facts. Remove hubRuntime.drainGuard, process wrapping and shutdown joins. Preserve unrelated output-ring/terminal drain functions. Verify ordinary messaging and process lifecycle suites, not only names containing Drain.
- [ ] Replace the console projection with one compact global badge (proposed text “draining N”), no full reason in the tab strip and no per-actor suffix. Status retains slot detail. Exercise width/selection at narrow and wide terminal sizes. Remove the obsolete drain-coordinator reducer AST rule and only its catalog entry when that coordinator no longer exists; update artifactpath inventory and slot-state audit for removed/new files. Keep general lint protections unchanged.
- [ ] Add a single bounded opt-in installed composition test, `TestDisposableCouchDrainFlags`, behind `PAIR_DRAIN_ACCEPTANCE=1`, invoked by `make test-drain-acceptance`. Keep pure/service/parser/endpoint/renderer tests in the ordinary suite; do not add a protected local-check declaration or SDLC fake. Run the target explicitly before close/pre-stage and retain exact Git head, argv, isolation manifest, exit, duration, full log and SHA256 in the issue Log. #734 is wontfix/archived and no longer a dependency.
- [ ] Update README, atlas/couch.md and the Couch help/skill to the same commands and scope; coordinate ariadne-owned worker/steward rule updates through TL. Preserve the failed morning evidence and state that old ariadne:4 ack refusal was not reproduced if it remains unexplained; formal ack code is removed rather than credited as fixed. The new acceptance is real message delivery from genuine launched environments, independent of the old saved flags.
- [ ] Run focused races for the new shared flag owner and existing message/lifecycle seams; run repository lint/artifact/slot audits and public installed test. Push and run the required declared suite on the tested branch, then mandatory close review. Stage only after SHIP and recorded disposable evidence, with TL authorization. Watched live acceptance is scheduled through TL/operator and exercises the new posture/notice/undrain/status model; steward queue operations are external, not a fake claim of Couch integration. Merge only after that acceptance.

#### Bounded installed test and independent oracles

Reuse cmd/couch/main_test.go installedEnv/installedExecutables and terminal fixture registration, extended to one steward and two independent worker checkouts. Keep actual Couch and actual Pair/wrapper caller registration; fixture agents execute public CLI subprocesses with their real launched COUCH_STORE_DIR/COUCH_THREAD_SCOPE/COUCH_THREAD_TAG/PAIR_SESSION_NAME/PAIR_LAUNCH_NONCE. Do not synthesize the accepted caller identity. If the existing terminal fixture substitutes the wrapper implementation, wire the real wrapper for this acceptance path; handler or fake-endpoint success alone cannot qualify it.

Within one proposed90s case: flag both active/quiet workers and steward, observe authentic notices and status; submit ordinary peer messages while flagged and confirm submitted receipts; lift with generic and custom text and confirm cleared flags/delivery; test public detach and quit while flagged; restart the real supervisor on the same private roots and require no flags and usable messaging; capture narrow/wide rendered frames with one badge and selectable ordinary chips. Use deterministic held fixture events instead of timing-only assertions. Negative controls must fail if ordinary delivery is gated, flags are serialized/restored, or caller routing is bypassed.

Own private COUCH_ISOLATED_ROOT/store/identity/Pair-data/home/XDG and Zellij/runtime roots under TestScratch when available, with TMPDIR fallback. Refuse live/empty roots and inherited live binding/socket references; construct identities from actual disposable repositories. Couch message socket addresses are hash-disjoint under the existing per-UID parent, not a nonexistent socket-root override. Join all fixture children and remove only exact owned process/session/socket identities. Keep before/after legacy drain.json fixture bytes unchanged and prove flags are not serialized anywhere new; never inspect or migrate the operator's retained wedged evidence. No real queue, production Couch control or worker lifecycle action is exercised.

Operating envelope: at most existing MaxActors flags; one O(N) set/project/copy and optional O(N log N) stable status ordering, no per-byte work. Count projections/sends at8/16 members to reject per-slot census or subprocess calls; expect one snapshot and at most one ordinary send attempt per selected slot per explicit command. No idle poller or durable residue is added: flags die with messageService; normal receipt/transcript retention remains owned by existing messaging. Five-second notification-admission and90s installed-test budgets are proposed limits to validate, not claimed measurements. If they cannot qualify the real path, return with evidence before widening them (ARCH-CONSTRAINTS/FUNERAL).

### 2026-10-20 14:14 PDT — operator approval and issuing-steward exclusion

Canonical submitted TL831e9502 confirms operator approval of the 12:18 Revision with edge1 changed: select currently connected slots except the authenticated steward hub issuing the drain. Other hubs are flagged normally. The issuing steward receives no drain notice and keeps driving under ariadne#506's tl-lead rule. Edges2–5 are approved unchanged. This supersedes the earlier proposed inclusion of the steward, including that assertion in the disposable test: the installed three-slot case must flag exactly the two workers, leave the issuing steward unflagged, and keep its commands/messages usable. Both graceful and fast commands apply this exclusion. No new mechanism or lifecycle is introduced.

#622 landed as PR474 in train194. Canonical pair:5 is free; the #722 reopen setter succeeded. The auxiliary documentation checkout was clean and removed before switching this branch into canonical pair:5. Proceed through renewed full-flow change-code, deriving the redesigned estimate after plan-quality. The former persisted-state SHIP and failed live check remain historical evidence, not current acceptance. No staging or fleet mutation is authorized by this approval; watched live acceptance still goes through TL/operator.
