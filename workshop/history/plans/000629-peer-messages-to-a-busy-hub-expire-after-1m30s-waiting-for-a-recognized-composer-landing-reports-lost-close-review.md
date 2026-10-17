# Boundary Review — pair#629 (whole-issue close)

| field | value |
|-------|-------|
| issue | 629 — Peer messages to a busy hub expire after 1m30s waiting for a recognized composer; landing reports lost |
| repo | pair |
| issue file | workshop/issues/000629-peer-messages-to-a-busy-hub-expire-after-1m30s-waiting-for-a-recognized-composer-landing-reports-lost.md |
| boundary | whole-issue close |
| milestone | — |
| window | 628ab0cadba9a7027130383d886fff443708c948..80be425bef48935ed3af1151c4eb49280aef1f54 |
| command | sdlc close --issue 629 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T22:12:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main timeout removal, FIFO behavior, receipt projection, and documentation are consistent with the intended busy-receiver behavior. One correctness bug blocks shipping: sender disconnect can classify a handed commit as safely cancelled. Regression coverage also fails to detect restoration of either old outer timeout.

Done-when: current — its clauses still cover connection-scoped retention, queue progress, and disconnect outcomes; the added diagnostic-history bound supports that retention contract.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: cancellation-preserves-input-uncertainty
    title: |
      Sender disconnect classifies handed commits as cancelled
    detail: |
      cmd/internal/couchmessage/model.go:409 now preserves Queued progress, but broker.go:269-273 decides cancellation safety from the displayed receipt status. After a handed commit reports Queued, sender disconnect records Cancelled even though the wrapper may have pasted since that observation; the terminal receipt cannot subsequently be corrected. Family enumeration: both Queued and Held interim receipts take this unsafe path. A disposable regression using newBusyReceiverFixture reproduced Cancelled for both. Classify uncertainty from delivery ownership rather than the last displayed status, and test both interim states plus genuinely uncommitted cancellation. This violates the plan's uncertainty contract and ARCH-PURPOSE.
  - id: new
    severity: Important
    kind: defect
    family: expiry-owner-regression-coverage
    title: |
      Regression tests do not detect restoration of either outer timeout
    detail: |
      The affected owners are Broker.deliver at cmd/internal/couchmessage/broker.go:955 and RemoteEndpoint.Deliver at endpoint.go:217. Restoring both previous outer deadlines in a disposable copy still passed all TestBusyReceiver and TestRemoteEndpoint cases (7.407s). The broker fixture's clock jump does not advance an already-created real context timer, and the endpoint exchange completes far before its former horizon. Add deterministic coverage that fails when either owner independently regains a pre-paste timeout, while preserving parent cancellation and bounded commit/RPC behavior.
```

1. **Strengths**

   - Reuses the existing queue and input owner; introduces no parallel retry store.
   - Busy-receiver tests exercise reservation refusal, committed queued/held waits, FIFO drain, replay, and exact-incarnation replacement.
   - Picker-history regression covers resource growth during indefinite waits.
   - README, atlas, and embedded skill document the changed waiting semantics.

2. **Critical findings**

   The cancellation finding above requires an ownership-based classification at `broker.go:269`, with coverage for both queued and held progress.

3. **Important findings**

   Both outer timeout removals need independently effective regression tests, as detailed above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed locally:

   - Full `couchmessage` package: 17.022s.
   - Wrapper `TestPeer` cases: 6.327s.
   - Targeted CLI receipt and supervision provenance tests: 0.345s.

   The additional sender-disconnect regression failed for both interim states. Restoring both old outer deadlines left the relevant existing tests green. Disposable copies were removed; the reviewed checkout remains unchanged.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY: pass** — existing queue, receipt, and input machinery are reused.
   - **ARCH-PURE: pass** — model transitions and receipt projection remain pure; endpoint and wrapper effects stay at existing IO seams. Core-concept paths and classifications match the implementation.
   - **ARCH-PURPOSE: flag** — retention is delivered, but cancellation no longer consistently preserves input uncertainty.

   Documentation gates are satisfied. The Revisions section explicitly supersedes the Release-lock work and pre-landing live check.

7. **Plan revision recommendations**

   Add a `## Revisions` entry covering ownership-based sender-disconnect classification for queued/held handed commits, and deterministic regression evidence for both outer timeout owners.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"294c4d88-035e-4b39-9236-eb6f9c6ce0cd","repository":"github.com/xianxu/pair","issue_id":"000629","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7c096cf9632e86dd","reviewed_head":"80be425bef48935ed3af1151c4eb49280aef1f54","slot":"pair:3","requested_at":"2026-10-17T05:09:51.014808Z"},"phase":"finished","pid":2165,"started_at":"2026-10-17T05:09:51.2458Z","finished_at":"2026-10-17T05:12:16.519808Z","session_id":"01a14844-3417-7460-8774-2dc1e678971f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.013544792s
head: "80be425bef48935ed3af1151c4eb49280aef1f54"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T23:12:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 629 — Peer messages to a busy hub expire after 1m30s waiting for a recognized composer; landing reports lost |
| repo | pair |
| issue file | workshop/issues/000629-peer-messages-to-a-busy-hub-expire-after-1m30s-waiting-for-a-recognized-composer-landing-reports-lost.md |
| boundary | whole-issue close |
| milestone | — |
| window | a991ba8fb2b1e30b199013aebdcdab9d237d7ed9..acce298254c4f151c1c355f0dccd8a6084eef0dd |
| command | sdlc close --issue 629 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T23:12:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the revised busy-receiver contract. Accepted messages survive connected composer waits, drain in FIFO order, and retain conservative uncertainty after handoff. Both prior findings are verified addressed; no new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      broker.go:266 classifies sender disconnect from active delivery ownership. TestBusyReceiverSenderDisconnectPreservesHandoffUncertainty covers handed queued/held receipts and uncommitted cancellation controls. Restoring receipt-based classification in a disposable pinned-head copy failed both handed cases.
  - id: BR-2
    disposition: addressed
    note: |
      delivery_lifetime_test.go:31 and :70 exercise broker and endpoint context lifetimes at their IO boundaries, including receiver cancellation, parent cancellation and bounded commit admission. Independently restoring each former outer timeout failed its corresponding regression.
```

Done-when: current — the clauses still cover prolonged connected waits, delivery once readiness returns, queue progress, and receiver-loss outcomes. The revisions clarify cancellation and uncertainty without changing those acceptance goals.

1. **Strengths**
   - Cancellation uses delivery ownership rather than displayed receipt status (`broker.go:266`).
   - Stateful receiver tests cover reservation refusal, queued/held handoffs, long waits, FIFO delivery, replay and exact-binding replacement (`busy_receiver_wait_test.go`).
   - Receipt projection remains pure and bounds detail without losing queue position (`model.go:151`).
   - README, atlas and embedded skill consistently describe the new timing contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Full couchmessage race suite passed: 25.009s.
   - Wrapper `TestPeer` race tests passed: 21.530s.
   - Targeted CLI receipt and supervision provenance tests passed.
   - Three independent mutation checks failed as expected: sender classification, broker outer timeout and endpoint outer timeout.
   - Diff whitespace checks passed. Disposable copy removed; reviewed checkout unchanged.
   - Live hub qualification remains explicitly deferred to the operator’s next restart.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Reuses existing queues, transitions and transport; no parallel delivery mechanism.
   - **ARCH-PURE: pass.** State transitions and receipt projection remain IO-free; lifecycle and transport checks use injected integration seams. Core-concept classifications match the code.
   - **ARCH-PURPOSE: pass.** The timing-consumer sweep covers model, broker, endpoint and wrapper. Both prior finding families are addressed across their enumerated cases.

7. **Plan revision recommendations:** None. Existing revisions explicitly supersede the Release-lock work and pre-landing staging tasks.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f8951c81-eb97-43ba-9040-5a3da343aac9","repository":"github.com/xianxu/pair","issue_id":"000629","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/2efff9e29ed44569","reviewed_head":"acce298254c4f151c1c355f0dccd8a6084eef0dd","slot":"pair:3","requested_at":"2026-10-17T06:10:31.672616Z"},"phase":"finished","pid":77655,"started_at":"2026-10-17T06:10:31.77058Z","finished_at":"2026-10-17T06:12:01.322797Z","session_id":"01a1487b-bfcc-7473-a0a0-8726fcca1273","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.828657291s
head: "acce298254c4f151c1c355f0dccd8a6084eef0dd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
