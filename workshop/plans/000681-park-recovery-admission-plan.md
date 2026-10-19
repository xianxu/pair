---
collaboration-mode: discussion
---

# Recover park receipt admission implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development if suitable, otherwise superpowers-executing-plans. Steps use checkbox syntax for tracking.

**Goal:** Return the existing recovery receipt before slow fleet inspection, while preserving selection of one exact park transaction.

**Architecture:** Bind the selected address and nonce from the console's already cached inventory, admit one bounded receipt and queue job, then perform full current-world preparation on that owned queue. Retain the existing selection guard at dispatch and RecoverParkExpected's nonce check under the lifecycle hold. Do not move the first transaction selection to execution time.

**Tech Stack:** Go, existing Unix message transport, Couch console operation queue, cached actionable inventory and lifecycle fake/store fixtures.

## Decision and alternatives

1. Recommended: cheap cached selection before receipt, slow validation after receipt. This preserves selection authority without fleet IO in the two-second exchange; unknown/absent/ambiguous cached rows refuse with refresh-and-retry advice.
2. Queue the first inventory and select later: fewer changes, but may silently select a replacement thread or newer park while queued. Rejected.
3. Increase admission timeout or spawn a separate preparer: enlarges a global latency budget or duplicates queue ownership/cancellation. Rejected.

The receipt remains an admission acknowledgment, not proof recovery has happened. Cached rows are untrusted observations: stale selection can refuse, never authorize changed state. The same address with an already-settled original park stays idempotent. A cached empty nonce cannot recover a newly opened park.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Cached slot-row selection | cmd/internal/couchcore/slot_operation.go | new helper, using existing SelectSlotRow |
| validateParkRecoveryTarget | cmd/internal/couchcmd/console_slot_operations.go | existing contract preserved |
| ApplyReceiptEvent | cmd/internal/couchmessage/operation.go | reused unchanged |

Cached selection accepts rows, canonical repository primary path and slot number. For :0 use the primary path with SelectSlotRow. For numbered slots find the unique matching Target.Slot.PrimaryRoot and Number to obtain the cached WorktreeRoot, then use SelectSlotRow; no workspace/Git discovery. Refuse no match or ambiguity, including duplicate roots/rows; never guess a conventional path. Reuse ActorOperationArgs for exact scope/tag/park-nonce arguments. Keep this helper pure and test it directly.

Relationships: each admitted request retains one immutable selected OperationCall; duplicate IDs share its existing receipt/job. Receipt state stays owned by ApplyReceiptEvent. The existing operation queue owns one pending canonical slot key until completion.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| slotOperations.handle/admit | cmd/internal/couchcmd/slot_operations.go | modified ordering | caller validation, receipt mutex, bounded queue admission |
| consoleSlotOperations.preparePark | cmd/internal/couchcmd/console_slot_operations.go | modified | cached console selection then queued current-world validation |
| Console.ResourceInventory | cmd/internal/couchtty/console_resources.go | reused unchanged | locked immutable snapshot of already published inventory |
| EnqueueRemoteOperation | cmd/internal/couchtty/console_remote.go | reused unchanged | owned console lifetime, one queue job and terminal completion |
| Stateful service/console fixture | cmd/internal/couchcmd/slot_operations_acceptance_test.go | extended | real Unix transport, fake terminal/process/slot state, temporary stores |

Existing fakeSlotRunner models canonical pending keys, retained callbacks and effect counts. Real console acceptance uses FakeHost/FakeChild, FakeRunner and SlotCatalogFake with private temporary roots. No production namespace or operator slots are test inputs.

## Exact ordering and ownership

1. Existing message validation/authentication/owning-hub admission runs. Require confirmation before cached binding; retain one policy owner rather than copying confirmation rules.
2. An already-held request ID returns its existing receipt (or conflict) without binding or scanning again, even if its old thread has gone away.
3. Resolve repository aliases using the existing bounded RepositoryNamesContext path (500ms lock budget); canonicalize the pending slot key. Reuse that resolved identity for cached row selection rather than making another repository read.
4. Read cached ResourceInventory: no external workspace resolver, native discovery, journal replay or full inventory. If unknown, absent or ambiguous, refuse explicitly: refresh the switcher inventory, inspect the row, retry confirmed recovery. Capture exact address plus park nonce once.
5. Existing admit allocates its <=64-entry receipt and calls ONLY a nonblocking enqueue under its mutex. It returns accepted with the receipt before any full inventory begins. Do not call slow preparation in admit's enqueue callback.
6. The existing owned queue marks running, checks current hub authorization, runs full PrepareSlotOperation and compares it to the captured identity with validateParkRecoveryTarget. A changed/ambiguous/missing target refuses through the receipt. Preserve already-settled idempotence.
7. Dispatch retains the guard, and RecoverParkExpected checks the same nonce again under its lifecycle hold. Recovery never starts a child. All completion/error/cancellation reporting uses existing started/finished ownership and receipt transitions.

No new goroutine, worker, transport timeout, durable file, status enum or public wire field. Socket disconnection after admission does not cancel an already accepted job (existing semantics); the receipt remains the outcome authority. Console shutdown cancels the owned operation and waits for its normal completion path; no detached IO wrapper.

## Visible states and recovery

Actions: inspect receipt, refresh switcher inventory and retry, inspect changed row and confirm a new request, follow returned next command.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Cached inventory unknown/missing/ambiguous | Explicit pre-admission refusal, refresh advice | Refresh inventory, inspect, retry | None; no recovery queued |
| Accepted, queued or preparing beyond 2s | Existing queued/running receipt | Inspect same receipt | None; recovery not yet confirmed |
| Same ID repeated | Same receipt/job | Inspect receipt | None; no duplicate effect |
| Different intent with same ID | Existing id-conflict refusal | Inspect old receipt; new confirmed request if desired | None |
| Target/nonce changes after cached selection | Refused receipt | Inspect current row before confirming a new request | None; replacement not recovered |
| Original park settles before execution | Idempotent outcome on same identity | Follow returned next command | None |
| IO/preparation failure or cancellation | Failed/refused receipt with existing detail | Inspect receipt and state before retry | Underlying park remains explicitly unresolved |
| Recovery succeeds | Existing recovery result + next command | Follow next command | Existing incomplete-scrollback warning preserved |
| Reply lost/restart/receipt expiry | Existing unknown/uncertain outcome | Inspect current row before a new request | Outcome knowledge only; never infer no effect |

## Operating envelope and principles

- ARCH-ORDER: immutable selection precedes receipt; receipt precedes full fleet IO; final nonce CAS owns mutation. The canonical queue key owns duplicate concurrency, not repeated absence probes.
- ARCH-DRY: reuse receipt machine, ActorOperationArgs, SelectSlotRow and current target guard. No parallel recover executor or lifecycle policy.
- ARCH-PURE: cached row selection is deterministic; IO remains in adapters.
- ARCH-MOCK: stateful fake plus actual Unix transport/console wiring; fixture storage is portable private folders.
- ARCH-CONSTRAINTS: admission keeps existing two-second budget; bounded repository-name lookup plus O(N) cached row copy/selection, zero native/workspace/full-inventory calls. Full preparation costs the existing fleet scan on the owned queue; no extra full scan compared with current code. Count scans and jobs at N=8 and 2N=16 rows; exactly zero full scans before acceptance, one job per ID, constant full validation pass count independent of row count.
- ARCH-SECURE: cache only selects an identity and never authorizes an effect. Recheck live caller/hub and current row before mutation. Wire values remain parsed/validated by the existing protocol; no new credentials or persisted artifact.
- ARCH-FUNERAL: create no new durable family. Receipt cap64, terminal retention5m, console stop disposes queue; reuse these bounds.
- ARCH-PURPOSE: fixes the actual >2s production admission failure while keeping exact transaction protection. #683 diagnostics and original failure cause remain separate.

## Implementation and verification steps

### Cached selection and admission ordering

Files: cmd/internal/couchcore/slot_operation.go, slot_operation_test.go; cmd/internal/couchcmd/slot_operations.go, console_slot_operations.go, park_recovery_surface_test.go.

- [ ] Add red pure selection cases for :0, numbered slots, aliases resolved to canonical repository, absent/ambiguous/unknown cache, unrelated subdirectory rows, empty nonce and immutable captured arguments.
- [ ] Add red ordering tests: unconfirmed/unauthenticated requests never bind; duplicate/conflicting IDs never rebind; alias spellings share the existing pending key.
- [ ] Implement the small cached selector and adapt preparePark to use it, sharing canonical repository resolution with queue admission. Preserve selection checks before queued effects and remove full preparation from socket admission.
- [ ] Run focused tests; expected PASS, zero full-inventory calls in admission and one job per ID. Commit code plus tests once green.

### Production transport and adversarial sequences

Files: cmd/internal/couchcmd/slot_operations_acceptance_test.go or focused park_recovery_admission_test.go, existing service/console fixtures; existing couchcore park recovery tests as needed.

- [ ] Build a real Unix-transport fixture with a published cached exact row, real console queue, private thread/lifecycle stores and stateful external fakes. Place a controlled barrier in queued full inventory preparation.
- [ ] Send a production recover-park request with the actual two-second admission context. Keep preparation blocked beyond that deadline (one bounded >2s interval using testwait-owned synchronization), prove receipt accepted and status readable before release, then release and prove exactly one recovery effect and terminal result. Prove test fails against the old ordering.
- [ ] Exercise replacement before queued prepare and between prepare/dispatch; each refuses without mutation. Exercise same original park already settled, new nonce after cached empty nonce, and stale cached address.
- [ ] Exercise preparation error, console cancellation while blocked, duplicate during blocked preparation, callback completion and pending-key release. Assert cancellation settles once, no late effect, and no worker survives fixture shutdown.
- [ ] Assert O(N) cached selection and constant full-pass/effect counts at8 and16 rows, no per-row external query. Retain existing authentication, confirmation, max receipt and queue overload tests.
- [ ] Run targeted packages and focused race coverage. Commit verification evidence and update atlas/couch.md plus the issue Log with the receipt/selection ordering and refresh refusal.

Commands after approval and change-code:

- go test ./cmd/internal/couchcore -run 'SelectSlot|RecoverPark' -count=1
- go test ./cmd/internal/couchcmd -run 'RecoverPark|SlotOperation' -count=1
- go test -race ./cmd/internal/couchcmd -run 'RecoverPark|SlotOperationReceiptsConcurrent' -count=1
- sdlc pr; sdlc local-check run --name suite on the pushed final head.
- sdlc close --issue 681 --verified exact evidence (TL owns project update); fix findings, then sdlc merge --yes promptly on SHIP. Do not chase main manually.

### Live conformance / completion

- [ ] Ask hub for a disposable interrupted-park fixture and staging window before touching native/operator slots. Verify confirmed CLI returns a receipt during slow preparation, operation-status remains readable, exact recovery succeeds and a changed nonce refuses. Do not reuse archived production victims or change :0 without hub scheduling. If hub defers to next restart, record the acceptance owed explicitly.
- [ ] Verify actual merge on origin/main and report PR/merge to hub with submitted receipt. Then start #683 part2 on fresh main; do not stack or close its root-cause scope.

## Approval

Pending hub approval. Important product delta: cached unknown/missing/ambiguous selection refuses with refresh-and-retry advice; it never defers the first identity selection to queued execution. No implementation begun.

## Revisions

2026-10-18 — Hub approved the cached-selection design and tests in delivery53e81e0b-ada3-4d7b-b830-7f30238b7ef2. Exact address/nonce binding must precede admission; unknown/missing/ambiguous cache refuses with refresh advice. Live disposable slot is pair:8, idle Qoder with no in-flight work; ask hub before staging on pair:0 to avoid operator trial collision. No live action has run.

2026-10-18 — Execution clarified existing shutdown semantics: Console.Stop cancels and joins queued work while the server lifetime ends; it does not invent a retained terminal receipt after shutdown. The transport regression will prove joined workers/no late effects on shutdown and separately prove preparation errors retain a failed receipt/key release. This preserves the table's lost-reply/restart unknown-outcome rule rather than widening #681 into receipt persistence or console shutdown redesign (#683 owns diagnostics).

2026-10-18 — Ad-hoc review caught a pre-existing alias lookup before held receipt lookup. The approved duplicate-first contract requires shared admission to inspect held receipts/conflicts under the mutex before any catalog IO, then recheck after unlocked lookup; regression covers catalog failure and a changed-target ID conflict. Review confirmed the fix; no new identity or state policy.
