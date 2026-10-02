# Boundary Review — pair#365 (milestone M3)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | b392bd58957f3b1638b9f20c878c9490b127c084..2f3ef5e0b36f922ee1a11ea73f60bab71fc336fa |
| command | sdlc milestone-close --issue 365 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-01T17:20:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M3 does what it set out to do. The registry now uses a "newest admitted wins" rule and has a regression test. Broker tombstones are capped by eviction, and both the eviction and the slot-relaunch retirement are tested. The wrapper keeps its last 64 receipts in memory. `--message-status` now asks connected recipients about IDs the broker has forgotten, with bounded fan-out, and treats a recipient that stays silent as `uncertain`. The crash/interleaving suite runs through the real service and real sockets. The live measurement is careful and honest; it includes a hypothesis that was tested and rejected. Nothing is Critical. Two Important gaps should be closed before shipping:
- **Untested headline guard.** The cross-process "reused ID adopts the retained outcome" path has no test that would fail without it. The test that claims to cover it goes through a different code path.
- **Unchecked receipt provenance.** `StatusContext` records a receipt from any wrapper without checking that the wrapper is that message's recipient. `adoptRetained` does check this.

1. **Strengths**
   - `registry.go:645-656`: `newerAdmittedForSlot` plus the `t < e.Token` displacement. The legal transitions can be read straight off the code, and `TestRegistryUnadmittedNewerSessionDisplacesNothing` covers the ghost → dormant → takeover sequence.
   - `broker.go:111-139`: capping tombstones by eviction removes the cliff where a full table refused new launches. A table full of *connected* actors still refuses, and the test pins that too.
   - `broker.go:197-256`: `StatusContext` runs only on the request path, caps concurrency at 4, gets half the admission budget, waits for every goroutine it starts, and reports silence as `ErrUncertain` → `uncertain` rather than as absence. That is good ARCH-ORDER practice on uncertainty.
   - `recordRecoveredLocked` reports a non-terminal recovered receipt as `Indeterminate` instead of leaving it `Delivering` forever.
   - `message_lifecycle_test.go`: the replacement test runs both orderings (`finishFirst` true and false), and the delayed-frames test runs both forms (`exec` true and false). The measurements file re-took the baseline on the same workload before comparing.

2. **Critical:** none.

3. **Important**
   - **`wrapcmd/peer_recovery_test.go` (TestPeerLostReceiptRecoveredAfterBrokerRestart).** The test calls `broker.StatusContext` first, which writes the receipt into `b.receipts`. The re-send that follows then returns early at `broker.go:427-433` and never reaches the wrapper's `reserve`. So none of the wire path is exercised: `peer_runtime.go:67-71` (`ErrAlreadyCommitted` plus the receipt), `endpoint.go:456-457` (rebuilding `AlreadyCommittedError`) and `adoptRetained` over `RemoteEndpoint`. If the reserve guard is reverted, the test stays green, yet its comment claims this exact guard.
     - Fix: on a fresh broker, send the reused ID *without* a status call first, and assert both that the result was adopted and that there was no second enqueue. Add a case where the retained receipt is still in flight; the plan asked for both terminal and in-flight receipts, and the reserve path only tests terminal ones.
     - This is the 2nd finding in family `test-name-matches-assertion`. Rule: every test added at this boundary must contain an assertion that goes red when the mechanism it names is removed. Mutation-check each new M3 test against its stated mechanism, not just this one.
   - **`broker.go:233-245` (`StatusContext`).** A holder's answer is accepted if `From == caller || To == caller`. Nothing checks that `r.Message.To` is the holder that answered, or that `r.Message.ID == id`. `RemoteEndpoint` checks the ID, but other `ReceiptHolder`s may not. A wrapper's reply crosses a process boundary, so per ARCH-SECURE it is untrusted. As written, any connected wrapper can plant a "Submitted" receipt for a message addressed to a different slot, and the broker records it as evidence. `adoptRetained` (`broker.go:171`) already requires `To == a.binding`.
     - Fix: collect `(binding, holder)` pairs and require `r.Message.ID == id && r.Message.To == binding`. Add a test case in which a non-recipient returns a forged receipt.

4. **Minor**
   - **Plan drift.** Every Task 3.1–3.4 line in the plan file is still `[ ]`. Three items are neither delivered nor revised:
     - `TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate` was never written as a service-level test; the coverage moved to wrapcmd and couchmessage.
     - The "isolated Zellij query experiment / redraw bytes per query" re-run is absent from the measurements file. It is moot now that queries are at 0, but the plan should say so.
     - No lessons entry is recorded.
     - This is the 3rd finding in family `plan-revision-drift`. Rule: before `milestone-close`, every Plan line in the closing milestone is either ticked or has a `## Revisions` disposition. Sweep all of Chunk 3, not only these items.
   - **`broker.go:130`.** An actor that is not connected but never passed through `disconnectLocked` has a zero `disconnectedAt`, so it sorts as the oldest and is evicted first. That is harmless, but worth a comment.
   - **Live send smoke test.** The project note says a live send between relaunched slots was not done. Ask the operator to run it before merge, per the dogfood rule.

5. **Test coverage notes:** the registry reducer, the `RecentDeliveries` bound, tombstone eviction and the uncertain/absent split in `StatusContext` are well pinned with in-package fakes. The gaps are the wire-level reserve adoption and the provenance check, both covered under Important.

6. **Architectural notes**

   | Principle | Result | Note |
   |---|---|---|
   | ARCH-DRY | pass | `retainedLocked` is shared by `reserve`, `enqueue` and `status`. |
   | ARCH-PURE | pass | `RecentDeliveries` and the registry are pure. |
   | ARCH-PURPOSE | pass | The plan revision explains why the dropped per-send family fan-out guards a path the CLI cannot take. The residual risk is stated in the atlas. |
   | ARCH-MOCK | pass | The fake endpoint models retained state and silence. |
   | ARCH-CONSTRAINTS | pass | `StatusContext` has bounded fan-out and timeout; the gain is measured live (−63% Couch CPU, 0 messaging spawns). |
   | ARCH-SECURE | flag | Receipt provenance in `StatusContext` (Important above). |
   | ARCH-ORDER | pass | The removed goroutine around `post(SendTargeted)` is now a synchronous hand-off with no wait for the reply, so nothing outlives the request. The transitions are explicit. |
   | ARCH-FUNERAL | pass | Receipts are memory-only and capped at 64. Tombstones are evicted. Recovered receipts carry `RetainUntil`. |

7. **Plan revision recommendations**
   - Add a `## Revisions` entry that ticks or disposes every Chunk 3 line. Say that lost-receipt coverage lives in `wrapcmd/peer_recovery_test.go` and `couchmessage/broker_recovery_test.go`. Mark the redraw-bytes experiment N/A because zellij queries went to 0. Record that no lesson was added, or add one.

```findings
findings:
  - id: new
    severity: Important
    family: test-name-matches-assertion
    title: |
      Lost-receipt recovery test re-sends via the b.receipts short-circuit; the wire reserve→AlreadyCommitted adoption path is unpinned
    detail: |
      TestPeerLostReceiptRecoveredAfterBrokerRestart calls StatusContext first, which records the receipt, so the re-send returns early at broker.go:427 and never reaches the wrapper's reserve; peer_runtime.go:67-71, endpoint.go:456-457 and adoptRetained over RemoteEndpoint can be reverted with the test green. 2nd in family. Rule: each test added at this boundary needs an assertion that goes red when its named mechanism is removed. Mutation-check all new M3 tests, and add a fresh-broker reused-ID send plus an in-flight retained receipt case.
  - id: new
    severity: Important
    family: untrusted-receipt-provenance
    title: |
      StatusContext records a wrapper-supplied receipt without checking the answering holder is its recipient
    detail: |
      broker.go:239 accepts a receipt if From or To equals the caller; it never checks r.Message.To equals the answering actor's binding (or ID == id for non-Remote holders). Any connected wrapper can plant an outcome for another slot's message. adoptRetained already checks To == a.binding. Carry the binding with each holder, require To == binding and ID == id, and test a forged answer.
  - id: new
    severity: Minor
    family: plan-revision-drift
    title: |
      Chunk 3 plan lines are unticked; lost-receipt service test, redraw-bytes experiment and lesson are neither delivered nor revised
    detail: |
      3rd in family. Rule: before milestone-close, every Plan line in the closing milestone is ticked or has a Revisions disposition. Sweep all of Chunk 3, not only the named items.
```
