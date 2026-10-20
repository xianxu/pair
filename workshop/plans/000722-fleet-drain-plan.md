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
