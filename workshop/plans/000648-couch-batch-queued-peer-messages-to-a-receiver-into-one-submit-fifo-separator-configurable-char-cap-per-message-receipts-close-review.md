# Boundary Review — pair#648 (whole-issue close)

| field | value |
|-------|-------|
| issue | 648 — Couch: batch queued peer messages to a receiver into one submit (FIFO, separator, configurable char cap), per-message receipts |
| repo | pair |
| issue file | workshop/issues/000648-couch-batch-queued-peer-messages-to-a-receiver-into-one-submit-fifo-separator-configurable-char-cap-per-message-receipts.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0dc92957af516764a525718ae52f86e493ff86a3..1d76e4b122effc4511a7011f11849de1991ab2f6 |
| command | sdlc close --issue 648 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T13:47:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation matches the revised Claude-hub scope: readiness freezes a capped FIFO group, one confirmation determines every member’s receipt, and uncertain exchanges are not replayed. All affected-package tests and batch race tests passed. One Important documentation gap remains.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-surface-documentation
    title: |
      README update appears missing for the batch configuration variable
    detail: |
      cmd/internal/couchcmd/message_batch.go:12 introduces COUCH_PEER_BATCH_MAX_CHARS, but README.md is unchanged in the pinned range and contains no documentation for it. Add the Claude-hub scope, 16384-code-point default, accepted range of 1–32768, invalid-value startup failure, and over-cap singleton behavior to README.md; atlas and embedded skill updates do not satisfy the separate README gate.
```

1. **Strengths**

   - `couchmessage/broker_batch.go:7` freezes membership under the broker mutex without endpoint IO, preserving reports arriving during the busy wait.
   - `wrapcmd/peer_batch.go:33` validates immutable membership and rejects reused follower IDs before enabling input.
   - The real-socket test verifies five FIFO envelopes, exactly one submit, independent receipts, and follower recovery without replay.
   - Combined Pair-log recording preserves the actual submitted turn and enforces shared count and byte budgets.

2. **Critical findings:** None.

3. **Important findings:** Document `COUCH_PEER_BATCH_MAX_CHARS` in README.md as specified above. This is a prose correction; a wording-presence test is unnecessary.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Full tests for `couchmessage`, `wrapcmd`, and `couchcmd`.
   - Batch-focused tests across all three packages, including `-race`.
   - Range `git diff --check`.

   Native client measurements and the deferred ten-report live-hub check were not rerun. The tracker explicitly records that remaining live-check arrangement. The checkout remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared envelope rendering and delivery accounting.
   - **ARCH-PURE — pass:** selection, validation, and transitions remain directly testable without IO.
   - **ARCH-PURPOSE — pass:** the motivating busy Claude hub is covered, including readiness-time arrivals.
   - **ARCH-MOCK — pass:** stateful receiver seams and real-socket wrapper integration exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** character, encoded-frame, membership, and recording bounds are enforced.
   - **ARCH-SECURE — pass:** identities, bodies, duplicate IDs, and frozen membership are validated.
   - **ARCH-ORDER — pass:** explicit preparation phases and reducer events govern ownership; tests control cancellation and late-arrival ordering.
   - **ARCH-FUNERAL — pass:** batching adds no durable queue; receipts reuse bounded retention, and increased log payload has an explicit writer bound.

   Core-concept locations and classifications match the implementation.

7. **Plan revision recommendation**

   Add a `## Revisions` entry extending Task 3’s documentation scope to include README.md and the new configuration variable.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"810aada4-9ecc-4a57-bcc3-0ab26234d4bf","repository":"github.com/xianxu/pair","issue_id":"000648","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c1bdf9599d08207f","reviewed_head":"1d76e4b122effc4511a7011f11849de1991ab2f6","slot":"pair:3","requested_at":"2026-10-17T20:43:21.780812Z"},"phase":"finished","pid":34557,"started_at":"2026-10-17T20:43:22.035574Z","finished_at":"2026-10-17T20:47:08.244932Z","session_id":"01a14b9a-dbe6-7ee0-b67c-35cba9c0e9d1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.295664958s
head: "1d76e4b122effc4511a7011f11849de1991ab2f6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T13:50:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 648 — Couch: batch queued peer messages to a receiver into one submit (FIFO, separator, configurable char cap), per-message receipts |
| repo | pair |
| issue file | workshop/issues/000648-couch-batch-queued-peer-messages-to-a-receiver-into-one-submit-fifo-separator-configurable-char-cap-per-message-receipts.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0dc92957af516764a525718ae52f86e493ff86a3..ace7fd5eea5fa5e1a3dce037856832607ea8f361 |
| command | sdlc close --issue 648 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T13:50:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised Claude-hub scope: FIFO batches freeze at readiness, preserve original envelopes, and share confirmation while retaining per-message receipts. BR-1 is addressed. No new blocking findings surfaced. The authorized post-restart busy-hub check remains outstanding.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:858–867 now documents Claude-hub scope, the 16,384-code-point default, range 1–32,768, invalid-value startup failure and over-cap singleton behavior. These statements match message_batch.go:11–21, message_service.go:755–758 and batch.go:34–46. This is a prose-only correction; no runtime regression test is required.
```

1. **Strengths**

   - Shared rendering preserves exact headers and separators across input and Pair-log recording.
   - Preparation prevents input before membership freezes; finalization preserves the writer’s current safety checks.
   - Controlled tests cover group cancellation, late arrivals, uncertain exchanges, per-member accounting and receipt recovery through real Unix sockets.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - All tests in `couchmessage`, `wrapcmd` and `couchcmd`.
   - Batch tests under `-race` across those packages.
   - Exhaustive production artifact classification.
   - Pinned-range `git diff --check`.

   Native client conformance and the deferred ten-report hub check were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared rendering, accounting and existing readiness logic are reused.
   - **ARCH-PURE — pass:** Selection and group transitions remain in the pure core; integration stays behind endpoint seams.
   - **ARCH-PURPOSE — pass:** Readiness-time collection includes reports arriving during the busy head’s wait.
   - **ARCH-MOCK — pass:** Stateful recipient and terminal fakes exercise production boundaries; native conformance is available.
   - **ARCH-CONSTRAINTS — pass:** Character, encoded-frame, membership and logging bounds are enforced.
   - **ARCH-SECURE — pass:** Requests validate identities and membership before input; uncertain transport failures cannot authorize fallback.
   - **ARCH-ORDER — pass:** Preparation and group transitions are explicit; controlled tests cover cancellation, late arrivals and immutable membership.
   - **ARCH-FUNERAL — pass:** Batch state follows existing lifetimes, receipts remain bounded, and logging has a writer-side byte budget.

7. **Plan revision recommendations:** None. The plan records the Claude-only rollout and explicitly leaves restart acceptance and landing steps open.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1c306711-f6f3-473d-8c68-6f27995065b9","repository":"github.com/xianxu/pair","issue_id":"000648","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/14a49dc854c739ac","reviewed_head":"ace7fd5eea5fa5e1a3dce037856832607ea8f361","slot":"pair:3","requested_at":"2026-10-17T20:48:21.518589Z"},"phase":"finished","pid":44759,"started_at":"2026-10-17T20:48:21.678496Z","finished_at":"2026-10-17T20:50:27.149941Z","session_id":"01a14b9f-6db6-76a2-85d3-187ee55024cc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.115357417s
head: "ace7fd5eea5fa5e1a3dce037856832607ea8f361"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
