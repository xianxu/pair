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
