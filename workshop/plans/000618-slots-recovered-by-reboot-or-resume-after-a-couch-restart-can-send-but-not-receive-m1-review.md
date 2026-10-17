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

---

## Re-review — 2026-10-16T16:37:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 618 — Slots recovered by reboot or resume after a Couch restart can send but not receive |
| repo | pair |
| issue file | workshop/issues/000618-slots-recovered-by-reboot-or-resume-after-a-couch-restart-can-send-but-not-receive.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 5254fc022774f55cefafc6496c2bcbeef7b4d13b..19363f2714db86ef31b23e71d3678eb655a88b05 |
| command | sdlc milestone-close --issue 618 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T16:37:21-07:00 |
| verdict | SHIP |

## Review

Checking the send handler path in message_service.go, then I'll run the tests.

Tests pass. Now reverting the BR-2 fix to confirm the regression test goes red.

Verifying what concrete type the supervisor's `c.Artifacts` has in production, to see whether the poller actually gets wired in.

```verdict
verdict: SHIP
confidence: high
```

This range does what M1 set out to do. The supervisor's message authority now reads zellij liveness from one polled snapshot (`launcher.ListingPoller`), so an admission no longer runs zellij itself. A send that is not admitted now leaves a `refused` receipt with its reason. Round 1's Critical (BR-2, the 2 s transport clamp) is fixed on both ends of the socket by one shared per-request budget. I confirmed it by switching the `send` budget off in `requestBudget`: `TestTransportSendOutlastsAdmission` fails with `context deadline exceeded`, then passes again once restored. All affected packages pass: `launcher` and `couchmessage` with `-race`, plus the full `couchcmd` and `couchcore` suites. `go vet` is clean. The only thing still owed is the live AFTER measurement, which the plan now explicitly defers until after landing with a deadline (before M2 closes). What remains is Minor.

### 1. Strengths
- **One budget for both ends.** `requestBudget` (`cmd/internal/couchmessage/transport.go:350-370`) is the single function that both `Call` and `Server.serve` use for a request's time limit. That removes the asymmetry behind BR-2 instead of patching it in one place, and the regression test runs through the real `StartServer` and `Call`.
- **The poller only touches the authority's copy.** `messageListedObserver` (`message_service.go:1301-1314`) attaches the poller to a copy of the artifact checker, so recovery and relaunch keep reading zellij directly. The test pins this with `checker.Zellij.Listings != nil → fail`.
- **Old or missing liveness refuses, visibly.** `ErrListingStale` names the snapshot's age and the last error. Before the first listing completes, a check refuses as "not yet known" rather than blocking or guessing (tested in both `launcher` and `couchcmd`).
- **Refusals are bounded on the write side.** At most 64, kept 10 minutes, swept on every `Send`/`Status`, and never counted against `MaxReceipts`. `TestBrokerRefusalsAreBoundedAndExpire` pins all four rules.
- **The wired-resolver test would catch a regression.** `TestMessageThreadResolverChecksReadOnlyThePoller` uses a zellij stub that takes 1 s per listing. If the resolver stopped using the poller, 50 checks would take about 100 s and the test would fail.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **The send's time budget restarts mid-request** (`message_service.go:829-833`, `broker.go:523`). The exchange limit is `AdmissionTimeout + 1 s`. But `Broker.Send` starts a fresh 2 s admission clock only after the sender check (`authority.current`) has finished, instead of deriving its deadline from the exchange's deadline. If anything before `Send` takes more than 1 s, the server's context expires first and the reply is dropped. This is a sibling of BR-2.
- **`serve` checks the request type three times** (`transport.go:192-212`). It decodes the frame for `requestBudget` and twice more for `requestIsRecoveryPreview`. `Call` also encodes the request twice. The server should apply `requestBudget` once and key only the EOF-watcher branch on the operation (ARCH-DRY).
- **The poller's zellij path isn't taken from the checker** (`message_service.go:331`). It is built from `launcher.ZellijSource{}` rather than `c.Artifacts`' own `Zellij`, so a configured `Path` would be ignored. In production both are empty today.
- **Failed listings are never reported.** The plan says a failed listing "is reported once per failure streak". In the code the failure only shows up inside the stale error text, and nothing logs it.

### 5. Test coverage notes
- **BR-2:** verified by the revert test described above.
- **BR-3:** the resolver test uses a 1 s-per-listing stub, so it would fail if the poller were bypassed. A separate test for the queued-message retry rate isn't needed: once the resolver lists nothing, listings per second depend only on the poller's interval, not on how many checks run. Nothing tests that `startMessageService` passes the poller to the resolver, but that is one line.
- **BR-5:** the fix deletes a blocking call. No cheap test would fail without it. The design behaviour it relies on (a check before the first listing refuses as "not yet known") is tested.
- `TestListingPollerRunStopsWithItsContext` only asserts at least 2 listings. Nothing pins the upper bound on the listing rate; it holds by construction (a synchronous `Refresh` driven by a ticker).

### 6. Architecture
- **ARCH-DRY — pass, with a Minor.** `ZellijSource.listings` is the single place both listing outputs come from. The serve-path decoding is the Minor above.
- **ARCH-PURE — pass.** Clock and runner are injected into the poller. The refusal logic is broker memory, tested without IO.
- **ARCH-PURPOSE — pass for M1 as revised.** The authority path reads only the poller. Refusals reach the send's last line and `--message-status`. `--list` (Task 4) moved to M2 by plan revision, at the TL's request.
- **ARCH-MOCK — pass.** The zellij stub scripts and the injected `list` function sit behind the same `ZellijSource` boundary production uses.
- **ARCH-CONSTRAINTS — pass, with a deferral.** The rate is fixed by design: 2 listings per second, at most 2 processes at once, no overlap. The CPU and listing-rate AFTER numbers are deferred by revision. That measurement must happen before M2 closes, and should include the idle cost: the poller now lists even when no messages flow.
- **ARCH-SECURE — pass.** Refusal reasons carry zellij error text and ownership-check messages, but no secrets.
- **ARCH-ORDER — pass.** Turning an admission timeout into a definite refusal is justified. `Admit` (or `Enqueue`) is the only way into an inbox, and the `stored` check keeps any receipt that was actually written. One note for later: on timeout, `Release` runs on an already-expired context, so a wrapper reservation can outlive a refusal. The CLI mints a fresh ID per send, so this only matters if same-ID resends are ever used.
- **ARCH-FUNERAL — pass.** Refusals are capped by count and age. The poller goroutine is joined through `service.workers` on the service's lifetime.

### 7. Plan revision recommendations
- Task 2: name the test file actually used (`message_listing_test.go`), and state the decided startup behaviour: a check refuses as "zellij liveness is not yet known" until the poller's first listing.
- Core concepts, ListingPoller: either drop "reported once per failure streak" or implement it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Decided as a distinct refusal ("zellij liveness is not yet known"); covered in zellij_listing_poller_test.go and TestMessageThreadResolverChecksReadOnlyThePoller.
  - id: BR-2
    disposition: addressed
    note: |
      requestBudget shared by Call and Server.serve; disabling the send case turns TestTransportSendOutlastsAdmission red (context deadline exceeded), restored is green.
  - id: BR-3
    disposition: addressed
    note: |
      TestMessageThreadResolverChecksReadOnlyThePoller drives messageThreadResolver(c, listings) with a 1s-per-listing stub; 50 checks run zero listings, so the retry-driven listing rate is bounded by the poller interval.
  - id: BR-4
    disposition: addressed
    note: |
      BEFORE numbers are in the Log (6.1/s, 68% CPU); a plan revision defers AFTER to the post-landing restart and requires it before M2 closes. Hold M2 close to it.
  - id: BR-5
    disposition: addressed
    note: |
      The startup Refresh on context.Background is gone; Run does the first listing in a worker on service.lifetime (message_service.go:475-476); checks refuse not-yet-known until then.
  - id: BR-6
    disposition: addressed
    note: |
      refusal now carries an explicit at time used by eviction and LatestRefusals; comparing against slot bindings belongs to the M2 --list consumer.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: transport-budget-clamps-caller-timeout
    title: |
      Broker.Send restarts a fresh AdmissionTimeout after the sender check instead of deriving from the exchange deadline
    detail: |
      2nd finding in this family. Rule: any budget nested inside a socket exchange must derive from that exchange's deadline (minus the reply margin), never start its own clock. Today the margin holds only because authority.current is cheap with the poller (message_service.go:829-833, broker.go:523).
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-request-classification
    title: |
      Server.serve decodes the request three times to pick its budget; Call marshals the request twice
    detail: |
      requestBudget already returns RecoveryPreviewTimeout for resume-candidates; serve should apply it once and key only the EOF-watcher branch on the op (transport.go:192-212, 251-254). ARCH-DRY.
  - id: new
    severity: Minor
    kind: defect
    family: plan-test-not-delivered
    title: |
      Plan says a failed listing is reported once per failure streak; the poller never reports it
    detail: |
      2nd finding in this family. Rule: every behaviour the plan's Core concepts table promises is either implemented with a test or removed by a Revisions entry at the same boundary. Here the failure only appears inside ErrListingStale text (zellij_listing_poller.go:68-77).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9398105f-5f66-4b2b-a9cf-c86a2be21a6e","repository":"github.com/xianxu/pair","issue_id":"000618","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/76b4167f1c28ebd3","reviewed_head":"19363f2714db86ef31b23e71d3678eb655a88b05","slot":"pair:2","requested_at":"2026-10-16T23:22:24.320167Z"},"phase":"finished","pid":24980,"started_at":"2026-10-16T23:22:24.648152Z","finished_at":"2026-10-16T23:37:19.413739Z","session_id":"f0d70dad-e3df-4600-921d-82e29411f88f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.713318709s
head: "19363f2714db86ef31b23e71d3678eb655a88b05"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
