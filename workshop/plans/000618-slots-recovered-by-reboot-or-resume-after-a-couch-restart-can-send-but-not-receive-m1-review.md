# Boundary Review — pair#618 (milestone M1)

| field | value |
|-------|-------|
| issue | 618 — Slots recovered by reboot or resume after a Couch restart can send but not receive |
| repo | pair |
| issue file | workshop/issues/000618-slots-recovered-by-reboot-or-resume-after-a-couch-restart-can-send-but-not-receive.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 5254fc022774f55cefafc6496c2bcbeef7b4d13b..c09811358886808dd2f508e7441125e0055cf01e |
| command | sdlc milestone-close --issue 618 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T16:17:20-07:00 |
| verdict | REWORK |

## Review

I'm checking the CLI and protocol paths for refused receipts, then I'll run the tests.

Test suite is slow; I'll run the targeted packages without race while it finishes.

```verdict
verdict: REWORK
confidence: high
```

Most of M1 is sound. The `ListingPoller` is small, injectable and well tested. The authority's checker copy is a value copy, so the supervisor's own recovery and relaunch reads still go to zellij directly. Refused receipts have their own bounds, expire, and don't count against `MaxReceipts`. The broker tests show that a check failure now produces a `refused` status with its reason. One fix in the plan does not work, though: "the CLI waits a second past the admission budget". `couchmessage.Call` cuts every exchange to `TransportTimeout` (2 s). The server's `serve` gives its handler the same 2 s and returns without writing a reply once that context has expired. So when admission runs out of time, which is the failure measured in the issue, the sender still sees "uncertain" and never sees the reason. The CLI test passes only because it injects its own `call` and never goes through the real transport. The atlas states the false claim as fact. This blocks the boundary. Separately, the Task 2 admission and promote-rate tests were not delivered, and the M1 live measurement is not recorded anywhere.

**1. Strengths**
- `cmd/internal/launcher/zellij_listing_poller.go`: the clock and the listing call are injected. A failed listing keeps the last good pair until it ages out, and a stale read fails closed with its age and the last error. The tests cover all of this with no zellij (`zellij_listing_poller_test.go:35`).
- `message_service.go:1297` `messageListedObserver`: the poller goes only into a copy of the checker. The test confirms the supervisor's own checker never receives it (`message_listing_test.go:125`).
- `refusal.go`: refusals have their own retention, cap and sweep. A resend with a refused ID is a fresh attempt, because `receiptLocked` never reads `refusals`. Status still checks the caller.
- The poller goroutine joins `service.workers` and stops with `service.lifetime`, so its lifetime has a clear end.

**2. Critical**
- `cmd/internal/couchcmd/messages.go:87`: setting `AdmissionTimeout+time.Second` has no effect.
  - `transport.go:Call` applies `context.WithTimeout(ctx, TransportTimeout)` with `TransportTimeout` = 2 s.
  - `Server.serve` runs the handler, and therefore `Broker.Send`, under a 2 s context. When admission hits its deadline, `ctx.Err() != nil`, so serve returns without a response.
  - Result: the "admission timed out" refusal is stored and `--message-status` shows it, but the send prints the old "uncertain / query status" text.
  - Fix sketch: give send requests a transport budget of `AdmissionTimeout` plus slack on both the server ctx and the `Call` clamp, or start Send's admission budget earlier than the transport deadline.
  - Add a test that uses the real `Server` and `Call` with a broker whose admission times out, and assert that the reason is on the last line.
  - Correct the atlas sentence at `atlas/couch.md:16`.

**3. Important**
- The Task 2 tests were not delivered.
  - Missing: an admission through the production `messageThreadResolver` and broker with a 1 s stub, landing inside `AdmissionTimeout`.
  - Missing: a test that the promote retry over 5 s causes at most about 5 listing pairs.
  - `message_listing_test.go` calls the observer directly, so it would not notice if `startMessageService` stopped passing `listings`.
- The M1 live step (Task 5) has no evidence.
  - There is no measurement in the issue Log of the zellij listing rate or CPU before and after.
  - The "2 per second" claim is unmeasured (ARCH-CONSTRAINTS).
- `message_service.go:157-158`: the first listing runs synchronously on `context.Background()` while `startMessageService` is starting up.
  - Under the measured load, the two serial listings (the nil-poller `listings` path) can each take up to `zellijQueryTimeout` (5 s), so startup can block for up to about 10 s and cannot be cancelled.
  - `Run` then lists again immediately.
  - Fix: refresh only inside `Run` and accept a brief "not yet known" refusal at startup, or bound the first refresh with the service context.

**4. Minor**
- `refusal.go:430` `LatestRefusals` has no production consumer in M1, because Task 4 moved to M2. Its supersede check compares `route.Target` against `Message.To.Slot` and derives the refusal time from `RetainUntil - RefusalRetention`. Store a `RefusedAt` time instead.
- `refusal.go:405`: for a family route, `To: Binding{Slot: route.Target}` puts a family name where a slot belongs.
- `message_service.go:157`: the poller is built from `launcher.ZellijSource{}`, not from the checker's `Zellij`. That is harmless today because production leaves `Path` empty, but the two can drift apart.
- The plan's checkboxes are still unticked for Tasks 1–3, though its Revisions say they are implemented.
- The plan says `--message-status` prints `refused: <reason>`. The CLI actually prints a status line and then the detail.

**5. Test coverage**
- Passed (no race detector): `go test ./cmd/internal/launcher ./cmd/internal/couchmessage`.
- Passed: the targeted `couchcmd` tests (`Listing|ThreadResolver|SendWaits|Unfocused`).
- Under `-race`, `couchcore` hit the default 10-minute timeout. That package is unchanged in this diff, so I don't attribute the failure to it.
- The CLI-timeout test checks the deadline handed to an injected fake. It cannot catch the transport clamp, which is a mock repeating the implementation's assumption.

**6. Architecture**
- **ARCH-DRY: pass.** One poller feeds every authority read, and the `listings` helper keeps a single source for both listing forms.
- **ARCH-PURE: pass.** The poller logic is pure apart from the injected clock and listing call.
- **ARCH-PURPOSE: flag.** The "send's last line says why" purpose is not met for the timeout case (Critical above).
- **ARCH-MOCK: flag.** The CLI test's injected `call` replaces the real transport boundary where the bug is.
- **ARCH-CONSTRAINTS: flag.** The budget claims (listing rate, timeout slack) are unmeasured or not enforced end to end.
- **ARCH-SECURE: pass.** Refusal details are shown only to the sender or recipient.
- **ARCH-ORDER: pass, with a note.** Refusal and success writes happen under `mu` inside the coalesced admission. A deadline that passes after `Reserve` is still a definite refusal, because only `Admit` fills an inbox.
- **ARCH-FUNERAL: pass.** Refusals are capped at 64 and expire after 10 minutes, and the poller goroutine is joined on shutdown.

**7. Plan revisions**
- Add a Revisions entry: the CLI timeout slack also needs the server's handler budget and `Call`'s `TransportTimeout` clamp raised for sends. Name the end-to-end transport test.
- Record the Task 2 tests as delivered, or state that they are deferred and why.
- Record the M1 live listing-rate and CPU measurement, or state that it moves to M2.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: transport-budget-clamps-caller-timeout
    title: |
      CLI send's AdmissionTimeout+1s is clamped to TransportTimeout (2s) by Call, and the server drops the reply when its 2s ctx expires, so a timed-out admission never reports its refusal reason
    detail: |
      couchmessage.Call wraps ctx with TransportTimeout=2s, and Server.serve runs the handler under the same 2s budget, returning with no reply once ctx.Err() is set. The "admission timed out" refusal is stored, but the send still prints the uncertain/query-status text. TestMessageCLISendWaitsForTheBrokersVerdict injects a fake call, so it cannot catch this, and atlas/couch.md states the false claim. Fix the send transport budget on both sides and add a test that goes through the real Server and Call.
  - id: new
    severity: Important
    kind: defect
    family: plan-test-not-delivered
    title: |
      Task 2 tests are missing: no admission through the wired resolver under a slow zellij, and no bound on listings caused by promote retries
    detail: |
      message_listing_test.go calls messageListedObserver directly. Nothing checks that startMessageService passes the poller to messageThreadResolver, and nothing checks that a 5s queued-message promote retry causes at most about 5 listing pairs, both of which the plan lists as failing tests for Task 2.
  - id: new
    severity: Important
    kind: defect
    family: unmeasured-operating-claim
    title: |
      M1 live step (Task 5) has no recorded listing-rate or CPU measurement
    detail: |
      The plan's M1 includes Task 5 live: under a full suite, sends land, plus the supervisor's zellij listings per second and CPU before and after. The issue Log records no such measurement, so the 2-per-second claim is unsupported (ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    kind: defect
    family: blocking-startup-io
    title: |
      startMessageService runs a synchronous, uncancellable listing refresh on context.Background before the service starts
    detail: |
      listings.Refresh(context.Background()) at message_service.go:158 runs two zellij listings serially, each bounded only by zellijQueryTimeout (5s), so supervisor startup can block for up to about 10s under the load this issue measured. Run then lists again immediately. Bound it with the service context, or let Run do the first refresh.
  - id: new
    severity: Minor
    kind: defect
    family: unconsumed-api-surface
    title: |
      LatestRefusals has no M1 consumer and infers refusal time from RetainUntil minus retention
    detail: |
      Task 4 moved to M2. Store an explicit RefusedAt time and compare it against resolved slot bindings when --list starts consuming this.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9d0b3176-5f36-4a8b-9f9b-e2053b2a5f11","repository":"github.com/xianxu/pair","issue_id":"000618","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f48f7c976a894f08","reviewed_head":"c09811358886808dd2f508e7441125e0055cf01e","slot":"pair:2","requested_at":"2026-10-16T23:03:17.155788Z"},"phase":"finished","pid":12544,"started_at":"2026-10-16T23:03:17.468156Z","finished_at":"2026-10-16T23:17:19.332222Z","session_id":"dc455cf2-a491-4438-bd15-9974924e1e9d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.889670792s
head: "c09811358886808dd2f508e7441125e0055cf01e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
