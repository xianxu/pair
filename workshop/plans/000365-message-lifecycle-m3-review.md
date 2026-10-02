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

---

## Re-review — 2026-10-01T17:23:39-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | b392bd58957f3b1638b9f20c878c9490b127c084..c0cf69af95eceb7b39feb3e8e7a3fe498a605c88 |
| command | sdlc milestone-close --issue 365 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-01T17:23:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Of the three open findings, the two Important ones are fixed and I checked the evidence by reverting each fix; the Minor one is still open.

- **BR-13 (receipt provenance):** fixed. `StatusContext` now records which wrapper sent each answer. It accepts a receipt only when `Message.ID == id` and `Message.To` is that answering wrapper. A new forged-answer case fails when this check is removed.
- **BR-12 (wire adoption path):** fixed. The lost-receipt test now re-sends on a fresh broker before any status query. That forces the send through the wrapper's reserve over the real socket, the `already-committed` reply, and `adoptRetained`. The test fails if either the wrapper-side conversion (`peer_runtime.go`) or the endpoint-side rebuild (`endpoint.go`) is reverted.
- **BR-14 (plan drift):** still open. Commit c0cf69af did not touch the plan file. Every Chunk 3 line is still `[ ]`. The service-level `TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate` and the redraw-bytes experiment are neither delivered nor given a Revisions disposition. The lesson item is now delivered. This is Minor and does not block the gate, but it is the third round this family has been raised.

Nothing is Critical.

1. **Strengths**
   - `broker.go:622-661`: each holder now travels with its binding, so the answer is tied to its source structurally rather than by a string check. Concurrency stays capped at 4 and the timeout is unchanged.
   - `peer_recovery_test.go:80-101`: one fresh broker per path (re-send, then status) removes the cache short-circuit outright, instead of reordering calls and hoping.
   - `lessons.md` gained two rules that cover these classes in general: "a test that warms a cache tests the cache" and "a component answering for others must prove entitlement".
   - Mutation checks I ran myself in a scratch copy:
     - removing the provenance check makes `TestBrokerStatusAsksRecipientsForForgottenIDs` fail;
     - disabling `peer_runtime.go`'s `AlreadyCommitted` conversion makes `TestPeerLostReceiptRecoveredAfterBrokerRestart` fail;
     - removing `endpoint.go`'s `AlreadyCommittedError` rebuild makes the same test fail.
   - The `couchmessage`, `wrapcmd` and `couchcmd` packages pass. I ran them outside the sandbox because the test writes under `/tmp`.

2. **Critical:** none.

3. **Important:** none new.

4. **Minor**
   - **BR-14, not addressed.** The plan needs a `## Revisions` entry that ticks or disposes every line in Tasks 3.1–3.4:
     - lost-receipt coverage moved to `wrapcmd/peer_recovery_test.go` and `couchmessage/broker_recovery_test.go`;
     - the redraw-bytes experiment is N/A because Zellij queries dropped to 0;
     - the lesson is added;
     - the operator smoke test and the close step stay open.
   - **Vacuous assertion.** `peer_recovery_test.go:102-108` claims to check "the wrapper accepted a second delivery" by testing `current.Message.ID != m.ID`. A duplicate with the same ID would leave `current.Message.ID == m.ID`, so this assertion can never fail. The test still catches the bug through `Send`'s error, but this line is misleading.
   - **Two places check provenance.** `StatusContext` (`To == a.from`) and `adoptRetained` (`To == a.binding`) each check it separately. A shared `vouchedBy(r, id, binding)` helper would keep the two from drifting apart.

5. **Test coverage notes**
   - Over the wire, only a finished ("terminal") retained receipt is exercised. A receipt still in flight is covered only through the fake-endpoint `StatusContext` test.
   - It still guards the shared `recordRecoveredLocked` conversion to `Indeterminate`, so this is acceptable. Task 3.2's "test both terminal and in-flight" line should say so in the plan revision (part of BR-14).

6. **Architecture, per principle**

   | Principle | Result | Note |
   |---|---|---|
   | ARCH-DRY | pass | The duplicated provenance check is noted under Minor. |
   | ARCH-PURE | pass | |
   | ARCH-PURPOSE | pass | |
   | ARCH-MOCK | pass | The real socket path is now exercised end to end. |
   | ARCH-CONSTRAINTS | pass | Fan-out and timeout are unchanged. |
   | ARCH-SECURE | pass | Answers from other processes are now tied to the answering wrapper; this closes BR-13. |
   | ARCH-ORDER | pass | The test controls ordering with a fresh instance per path. |
   | ARCH-FUNERAL | pass | Nothing new is persisted. |

7. **Plan revision recommendations:** the BR-14 Revisions entry described under Minor.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Fresh-broker re-send reaches reserve then wire already-committed then adoptRetained; reverting the peer_runtime.go conversion or the endpoint.go rebuild turns the test red (verified in a scratch copy). In-flight is covered only through the shared recordRecoveredLocked via the StatusContext test.
  - id: BR-13
    disposition: addressed
    note: |
      broker.go:658 requires ID == id and To == the answering binding; the forged-answer case in broker_recovery_test.go fails when that check is removed (verified).
  - id: BR-14
    disposition: not-addressed
    note: |
      Plan file untouched in c0cf69af; all Chunk 3 lines still unticked; no Revisions disposition for TestLifecycleLostReceiptAfterBrokerRestartNoDuplicate or the redraw-bytes rerun. Only the lesson item is now delivered.
findings:
  - id: new
    severity: Minor
    family: test-name-matches-assertion
    title: |
      TestPeerLostReceiptRecoveredAfterBrokerRestart final check (current ID == m.ID) can never fail
    detail: |
      3rd in family. A duplicate delivery with the same ID leaves current.Message.ID == m.ID, so the assertion cannot catch it; the test catches the bug only through Send's error. Rule (already in lessons.md): every assertion must go red when the mechanism its message names is removed. Assert an enqueue or commit counter instead, or delete the line.
```
