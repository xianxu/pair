---
collaboration-mode: discussion
---

# Per-slot switcher operations Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use bounded subagents for independent exploration; keep the state-machine integration in the main session. Steps use checkbox syntax.

**Goal:** A lifecycle operation on one slot leaves the switcher and other slots usable, with progress and failures attached to the correct row.

**Architecture:** Replace the menu's global operation origin with a pure collection of independently identified row operations. Reuse Console's bounded workers, the authoritative core lifecycle gates, and its single completion owner. Preparation/adoption waits run off the input loop; terminal focus changes remain serialized and require the operator's current navigation intent.

**Tech Stack:** Go reducers, existing operation queue, Console event loop, terminal presenter, stateful FakeRunner and private test stores.

## Evidence and dependencies

Source inspection, independently checked by a read-only subagent, identifies two seams:

- `menu.go:dispatchMenuOperation` silently refuses every operation while the single `MenuState.InFlight` is occupied. Workers already run concurrently (`console.go`, `operation_queue.go`); adding another executor would duplicate them.
- `finishOperation` performs implicit attach synchronously on Console.Run. The installer waits for resize/presenter registration while holding `c.mu`. Moving only the lifecycle call to a goroutine would leave this blockage.

Claim/start-plan completed on the issue branch. No production changes yet. Write failing regressions before implementation to confirm each seam.

Implementation waits for pair#651/PR414 on main. Read-only review used ecf055bee02b7abcc6c738993d49dcd0062d25ed. Preserve recover-park's captured address/nonce, core reservation, cancellation semantics and typed receipt. Do not stack on its branch.

Pair#574/PR419 is also in flight. Preserve its observer seam on integration: one `observeLifecycleCompletion` outside `c.mu`, after final adoption error and before menu reduction/pass-drained observation. Reuse that publisher; add no competing notice writer. The issue's row behavior does not depend on #574 landing first.

## Core concepts

### Pure entities

| Entity | Change | Home | Contract |
|---|---|---|---|
| Menu operation collection | New | `cmd/internal/couchtty/menu_operations.go` | Row-keyed running/result state, immutable origins, attempt matching, progress and busy refusal. |
| MenuState transitions | Modify | `cmd/internal/couchtty/menu.go`, `menu_slot.go` | All routes use the collection; completion cannot restore a superseded frame. |
| Navigation intent | New | `menu_operations.go` and Console focus state | A captured generation and exact target authorize a future landing only until a newer navigation choice. |
| Row operation projection | New | `menu_operations.go`, `menu_render.go` | Derive progress, result and the available recovery action from the row's job and current authoritative inventory. |

Reused without inventing another lifecycle: `ThreadRowKey`, `MenuOperationOrigin`, core ThreadGate, #651 park transactions, ReattachPass, operation declarations, expected-exit tracking and typed recovery offers.

### Integration points

| Boundary | Change | Files | Test seam |
|---|---|---|---|
| Bounded operation execution | Modify | `operation_queue.go`, `console.go`, `console_remote.go` | Channel-controlled stateful dispatcher; reuse queue capacity 16 and existing worker count. |
| Child adoption | Modify | `console.go`, existing attach transaction helpers | FakeRunner typed starts plus controlled registration/attachment completion; private terminal fixture. |
| Terminal selection | Modify | `terminal.go`, `console.go` | Existing serialized terminal commands, checked against current intent immediately before selection. |
| Lifecycle publication | Preserve | #574 Console observer seam, when landed | Existing lifecycle observer and broker tests; exactly one final outcome. |

## Identity, ownership and scheduling

Each running UI operation has one immutable attempt, operation, original address, row key and frame origin. Resolve a valid slot row key at admission for **every** route, including hotkeys and recover-park; do not fill it after admission. Ordinary rows use exact address. Fresh starts without a row use the validated start resolution's worktree/provisioning target as their temporary key; refuse an unidentifiable target rather than using a shared empty address. Whole-Couch leave remains an explicitly exclusive operation, never a fabricated slot.

The collection reserves the target before enqueue and releases it only after final adoption/result processing. A repeated effect on the same target returns `busy: <operation>`, without enqueue or supersession. Core gates remain the mutation authority across UI, socket and automatic work; the UI reservation covers its queue/adoption gaps and never grants authority over a remote holder. Preserve remote key admission and synchronous CLI completion callbacks.

Different targets use the existing bounded worker pool and finite queue. If full, show a refusal and release the UI reservation without effects. Keep one worker's capacity available for navigation to an already hosted pane: local switching must not wait behind lifecycle workers. A switch that requires resume uses the lifecycle budget. No goroutine per input event, unbounded waiting list or automatic lifecycle retry.

Once admitted, close that action/form back to the root list immediately. Root navigation, menu opening/closing, mouse selection, hotkeys and hub switching continue. A busy-row Enter/click selects its progress and records a landing intent, rather than starting a second lifecycle action. A second mutation still refuses with the running operation's name.

Starting/resuming/relaunching a slot records a landing intent. Completion attaches the child in the background first; it selects it only if that exact intent remains current. Choosing another row/actor/hub, opening another navigation flow or closing the switcher invalidates the old intent. Selecting the busy row creates a new intent for its existing attempt. Thus waiting on the row lands when ready, but a job never pulls focus back after the operator leaves. Preserve notification-hop attention capture for actual switch landings.

## State and interrupting events

The collection uses a tagged phase (`Running`, `Succeeded`, `Failed`, `Unconfirmed`, `NeedsConfirmation`), not combinations of nullable flags. `Running` includes queue wait, operation and adoption; progress text names the current step. Typed receipts distinguish a confirmed refusal from an unconfirmed external outcome.

| State/event | Next state and effects |
|---|---|
| No running target + confirmed action | Reserve attempt; enqueue immutable call; return root with progress. |
| Running + same-target mutation | Keep owner; emit busy refusal; no second work. |
| Running A + action/switch B | Admit B independently; newer navigation intent invalidates A's landing. |
| Running + progress | Update only matching attempt; stale progress ignored. |
| Running + inventory replacement/removal | Keep original reservation and row key; refresh is evidence, never completion. |
| Operation returns started child | Keep Running/Attaching; prepare adoption off the input loop; commit only after revalidation. |
| Matching final success | Record row result, release running reservation; refresh inventory; land only if intent still current. |
| Matching failure/uncertainty | Record typed failure and recovery advice; release UI reservation only after owned work returns. No rollback claim or automatic retry. |
| Recovery confirmation required | Store offer on its row; do not open a modal over newer navigation; selecting row exposes existing explicit confirmation. |
| Duplicate/old completion | Ignore state/focus changes; never adopt a child twice; preserve exactly-once disposal and remote callback ownership. |
| Operator switches/menu closes | Invalidate landing intent; do not cancel or release admitted lifecycle/park work. |
| Console shutdown | Cancel lifetime; stop admissions; tracked workers finish/clean up before teardown joins. No persisted UI jobs. |

For process death or connection loss, core receipts and subsequent inventory determine the real lifecycle outcome; the UI must retain uncertainty and offer the current declared recovery action. In particular an unfinished park offers Recover park with the captured nonce. No UI timeout releases #651's shared worker reservation. Newer attempts and out-of-order results are fenced by attempt plus stable target; address changes alone cannot steal another job's result.

### Operator projection and one next action

| Internal phase | Row / visible behavior | One next action |
|---|---|---|
| Queued/executing/adopting | `Starting…`, `Relaunching…`, `Attaching…` with current step | Work elsewhere, or select this row to wait for its landing. |
| Successful final result | Brief operation result, then refreshed ordinary row state | Enter to switch. |
| Safe refusal/failure | `failed: <reason>` retained per row; full detail reachable through existing notice-copy/details surface | Choose the row's declared retry action. |
| Partial park / uncertain lifecycle | Reason plus current recovery advice; no fabricated success | Recover park, or the existing typed recovery action applicable to the evidence. |
| Confirmation required | Row says confirmation needed, keeps exact offer | Select the row to review/confirm. |
| Target disappeared | Exact target and result in the retained global notice; unrelated rows remain usable | Refresh the switcher to inspect current state. |

Do not append guessed retry instructions to arbitrary errors. Derive the action from current row/action policy and typed recovery evidence. This changes no viewer/cast surface.

## Adoption and completion boundary

Keep one final completion owner for adoption, observer, menu result, expected exits, continuation and remote callback. Split the slow child-preparation portion from final reduction: workers perform dispatcher attach/preparation without blocking Console.Run; reserve any installation identity under `c.mu`, perform resize/registration outside it, then revalidate lifetime, endpoint identity and duplicate ownership before a short commit. Use existing terminal-command serialization only for terminal selection, not lifecycle waits. If registration succeeds but commit is rejected, retire that owned registration and leave typed start rollback with its existing owner. Never close an already accepted child.

The UI reservation spans this stage. Remote callbacks and #574 final observations run after final adoption, including errors. Preserve expected-exit markers across both exit-before-result and result-before-exit orderings for **all** running origins, not whichever row is selected. Reattach's Background flag remains specific to the startup pass; do not reuse it for operator jobs.

## Operating envelope and cleanup

- Keystroke path performs pure transitions plus bounded queue admission; no lifecycle, readiness or adoption waits.
- Reuse queue capacity 16 and `LifecycleParallelism+1` worker envelope. Navigation to a hosted pane has reserved execution capacity; overload is an explicit refusal.
- Running entries are bounded by admitted workers plus queue; terminal outcomes retain at most one per currently listed row. Prune inactive missing rows after publishing their final notice. Replace outcomes on next attempt; release all state at Console death. Do not retain an event history.
- Progress stores one bounded detail per job, using existing notice limits; no new log family, timer, polling loop or durable file.
- Existing lifecycle deadlines remain; no new automatic retries or cancellation on navigation. Shutdown cancellation does not assert native effects were undone.

## Architecture checks

- **ARCH-DRY:** reuse workers, lifecycle gates, declared actions, row keys, typed recovery and #574 publisher; remove every global-InFlight consumer together.
- **ARCH-PURE:** admission, matching, intent invalidation and row projection are pure reducer functions; IO reports events.
- **ARCH-PURPOSE:** cover root Enter/mouse, action/confirmation, start/add-slot, recovery, switch-agent and hotkeys, including #651 recover-park and completion/adoption.
- **ARCH-MOCK:** use the existing stateful FakeRunner/private stores and real Console/terminal fixture; control dispatch/adoption with channels, not sleeps. No new external command surface.
- **ARCH-CONSTRAINTS:** existing bounded pool/queue, one target owner, explicit saturation; responsive hosted-pane navigation even with lifecycle work held.
- **ARCH-ORDER:** exact attempt/row transitions above; test rejection, cancellation, missing/replaced rows and all relevant completion/exit orders. All collection changes go through reducer helpers, never IO-side map writes.
- **ARCH-FUNERAL:** one current outcome per listed row; prune missing inactive rows, lifetime-owned workers/registrations, no new persistence.
- **ARCH-SECURE:** inventory and external results remain typed evidence, not authority; preserve action admission, exact identity/nonce and hub checks at execution. Fixtures use private roots and fake tools, never live slots. No new credential access or external parser.

## Implementation tasks (one atomic close)

### 1. Per-target reducer and rendering

Files: create `cmd/internal/couchtty/menu_operations.go` and `menu_operations_test.go`; modify `menu.go`, `menu_slot.go`, `menu_render.go`, `menu_actions.go` only as needed for shared row projection; extend `menu_render_test.go`.

- [ ] Write red tests: A relaunch followed by B/hub switch emits B's effect; same-A mutation yields busy; two attempts finish in either order with isolated progress/failure.
- [ ] Run `go test ./cmd/internal/couchtty -run 'TestMenu.*(Operation|Background|Busy|Progress)' -count=1`; verify the new assertions fail on the current global guard.
- [ ] Implement the pure collection, target admission before enqueue, immediate root return and row projection. Convert every `InFlight` consumer, including reconciliation and attention bookkeeping.
- [ ] Add tests for changed frame/row ordering, missing/replaced address, stale progress/result, hidden recovery offer and busy-row landing intent. Render at minimum geometry and check full failure access.
- [ ] Run the reducer/render tests green; commit with issue and Agent trailers.

### 2. Console execution, adoption and navigation

Files: modify `console.go`, `console_remote.go`, `operation_queue.go`, `terminal.go` as required; extend `console_menu_operation_test.go`, `console_run_menu_test.go`, `console_attach_transaction_test.go`, `operation_queue_test.go`; create `console_slot_jobs_test.go`.

- [ ] Write channel-handshake red tests through Console.Run: hold A's lifecycle call, select B then hub, observe terminal cells/input delivery before releasing A. Hold adoption separately and repeat. No sleep-based ordering.
- [ ] Implement bounded worker/preparation and short validated adoption commit; keep one final completion path and reserved navigation capacity.
- [ ] Test same-row duplicate before execution, during effect and during adoption; independent B progress; queue overload; shutdown with work held; duplicate/stale result and cleanup ownership.
- [ ] Test choosing busy A then remaining there lands after completion; navigating away before completion/adoption never steals focus or restores an old frame.
- [ ] Test exit/result in both orders, startup reattach and continuation unaffected, and remote CLI callback waits through adoption while other UI input proceeds.
- [ ] Run `go test ./cmd/internal/couchtty ./cmd/internal/couchcmd ./cmd/internal/couchcore -run 'Test.*(Menu|OperationQueue|Attach|Reattach|Continuation|Remote|RecoverPark|Lifecycle)' -count=1`; expected pass. Commit.

### 3. Integration, documentation and acceptance

Files: `atlas/couch.md`, README's switcher/lifecycle guidance if behavior is described there; issue Log/plan.

- [ ] Merge main after #651 lands before production work; recheck exact recover-park tests and preserve #574 observer ordering whenever it arrives. Do not build from peer branches.
- [ ] Enumerate root Enter/mouse, confirmations, hotkeys, recovery, start/add-slot, switch-agent and leave; prove each uses the shared admission/ownership rules. Check no stale global-InFlight comments or consumers remain.
- [ ] Run targeted packages normally and focused race tests for new ordering cases. Update atlas and operator guidance with progress, busy refusal and no focus stealing.
- [ ] Push head; run recorded full suite via `sdlc local-check run --name suite` (the configured suite is make -k); follow the hub's flake rule.
- [ ] Proposed live acceptance, only when hub schedules it: start a slot relaunch, switch another slot and hub immediately, return to see progress/result; attempt duplicate action; observe a controlled refusal without losing UI. No pair:0 staging without hub instruction. Headless stateful fixtures remain the deterministic proof of held lifecycle/adoption orderings.
- [ ] Record evidence, close through mandatory fresh review, land on SHIP with `sdlc merge --yes`, report real merge/PR to hub and stop.

## Revisions

### 2026-10-17 — hub design approval and display rule

Hub approved this design in delivery 24b5e642-2afe-4954-8922-b53c54a01ea1. Keep visible states few: a busy row shows its current operation/step **and elapsed time**, while a finished or failed action retains **one result line** on its row. Capture start time at admission from the existing injected clock/event time; derive elapsed time from existing menu ticks, never add a timer or wall-clock read to the pure reducer. Test progress age and result retention with explicit times. The earlier projection's “brief operation result” is superseded: keep the one-line result until the next attempt or inactive-row removal, with fresh authoritative row state still available. Live acceptance rides the hub's next restart. Do not stage. Wait for the hub to announce #651 landed before code.
