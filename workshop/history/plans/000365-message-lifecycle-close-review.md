# Boundary Review — pair#365 (whole-issue close)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | whole-issue close |
| milestone | — |
| window | f42189564ba340eb017d4292dc74a73c8f37e7f5..c44c786ef19a798509d47c58e7cbab714f58cff2 |
| command | sdlc close --issue 365 |
| reviewer | claude |
| timestamp | 2026-10-01T17:25:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue does what it set out to do. Messaging now runs on lifecycle events instead of polling. The registry reducer, session transport, pane mailbox and wrapper-retained receipts are all in place and tested. The live measurement reports zero messaging spawns and Couch CPU down 63%. Since round 7 the only change is c44c786e, which touches the ledger and review notes and no code.

Of the seven open findings, four are now fixed: BR-6, BR-8, BR-10 and BR-11. Three are still open, all Minor:
- **BR-9 and BR-14:** the plan file has not caught up with the code.
- **BR-15:** a test assertion that can never fail.

None of these blocks shipping. Running the couchmessage, couchcmd and wrapcmd lifecycle, registry, broker, session, recovery and peer tests outside the sandbox: all pass. Inside the sandbox they fail with a `/tmp` mkdir permission error, which is the environment, not the code.

1. **Strengths**
   - **Registry displacement:** `registry.go:190` now requires the newer session to be *admitted* before it displaces an older one. `TestRegistryUnadmittedNewerSessionDisplacesNothing` checks liveness (the older session stays connected, the ghost goes dormant) and that an admitted newer session does take over.
   - **Tombstone removal:** handled at the writer. A relaunch retires the slot's other incarnations (`broker.go:149-156`), and a full table evicts the oldest tombstone (`broker.go:182`). It is pinned in `broker_test.go:325`.
   - **Send path:** `message_service.go:544-563` no longer spawns a goroutine. `post(..., false)` only waits for the loop to accept the event. The alias-target limitation is stated where the code is.
   - **Probe trace cleanup:** `run.sh disarm` now deletes the trace, so the trace file has a removal path.
   - **Retained receipts:** the wrapper keeps at most 64. The broker's `receipts` table is pruned and capped by `MaxReceipts`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** BR-9, BR-14 and BR-15 are re-raised as not-addressed (details in the findings block). BR-9 and BR-14 both belong to `plan-revision-drift`. The rule that covers both: before `sdlc close`, sweep the whole plan once. Every Core-concepts row's path must exist in the tree, and every Plan line must either be ticked or have a Revisions disposition. Do one sweep, not per-item patches.

5. **Test coverage**
   - Still untested:
     - the helper-subprocess kill test in `session_transport_test` (planned in Task 2.3, never written);
     - a test that wires `startPeerRuntime` to the registry socket (its only caller is `wrap.go:2831`);
     - a service-level lost-receipt test (planned as `TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate`).
   - The wire adoption path is pinned at broker level (BR-12).

6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. The Registry is a pure reducer and the service executes its effects.
   - **ARCH-PURPOSE:** pass. The measurement is delivered.
   - **ARCH-MOCK:** pass. Tests use real sockets and a fake authority.
   - **ARCH-CONSTRAINTS:** pass. Status fan-out is bounded (concurrency 4) and activity frames are coalesced.
   - **ARCH-SECURE:** pass. Peer credentials are checked, and recovery answers must come from the recipient.
   - **ARCH-ORDER:** pass. The ordering test drives random interleavings through `Advance`.
   - **ARCH-FUNERAL:** pass now that tombstone eviction and the trace removal are in.

7. **Plan revisions needed**
   - Fix the Core-concepts row that names `backoff.go`: `ReconnectBackoff` lives in `session_protocol.go`, and `backoff.go` does not exist.
   - Record that the Task 2.3 helper-subprocess kill test and the Task 2.6 wrapper-level real-broker test were not delivered.
   - Tick or dispose every Chunk 3 line:
     - Task 3.1: `TestLifecyclePartialInputIsIndeterminate` (Revisions says it is covered elsewhere) and `TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate` (absent).
     - Task 3.3: the planned test names were replaced by the eviction test.
     - Task 3.4: the redraw-bytes rerun is not recorded in the measurements file.
     - The remaining delivered lines just need ticking.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      run.sh disarm now rm -f the trace; window keeps its own summary, so the armed-only growth has a removal path.
  - id: BR-8
    disposition: addressed
    note: |
      registry.go:190 newerAdmittedForSlot; TestRegistryUnadmittedNewerSessionDisplacesNothing asserts liveness of the older session and ghost goes Dormant.
  - id: BR-9
    disposition: not-addressed
    note: |
      M3 Revisions says ReconnectBackoff "exists as named" but the Core-concepts row (plan line 45) still names nonexistent backoff.go; Task 2.3 subprocess-kill test and Task 2.6 startPeerRuntime wiring test are still absent and unrecorded.
  - id: BR-10
    disposition: addressed
    note: |
      message_service.go:562 posts without a goroutine (post waits only for inbox acceptance); alias-target limitation is documented in place.
  - id: BR-11
    disposition: addressed
    note: |
      broker.go:149-156 retires the slot's other incarnations on Register, and evictOldestTombstoneLocked bounds the table; broker_test.go:325 pins it.
  - id: BR-14
    disposition: not-addressed
    note: |
      Plan unchanged since 311937fb; every Chunk 3 checkbox still unticked; no disposition for TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate (absent), the redraw-bytes rerun (not in measurements), or the Task 3.3 test names. Fix by the one-sweep rule, not per item.
  - id: BR-15
    disposition: not-addressed
    note: |
      peer_recovery_test.go:102-107 still asserts current ID == m.ID, which a duplicate delivery of the same ID also satisfies; replace it with an enqueue or commit counter, or delete it.
```
