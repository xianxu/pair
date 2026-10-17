# Boundary Review — pair#634 (whole-issue close)

| field | value |
|-------|-------|
| issue | 634 — every step of a couch send spends the one exchange budget; no fresh timers outside admission |
| repo | pair |
| issue file | workshop/issues/000634-every-step-of-a-couch-send-spends-the-one-exchange-budget-no-fresh-timers-outside-admission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 414195b345eb33569564884e297205cd00f988c0..65ee79d1138e818704d09a3e69b1b89deaa6834a |
| command | sdlc close --issue 634 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T05:20:11-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the issue’s shared-budget and snapshot-reuse requirements. Sender verification, registry posting, and recipient admission remain bounded by the request’s admission deadline, preserving the transport response margin. No blocking defects found.

Done-when: current — its every-step deadline requirement covers the registry-posting revision, and the slow-sender regression verifies delivery of the refusal reason.

```findings
{}
```

1. **Strengths**
   - `message_service.go:818` starts the admission budget before sender verification; the socket regression checks identical sender and recipient deadlines.
   - `broker.go:564` passes admission’s families snapshot into refusal bookkeeping, removing the extra lookup and routing drift.
   - `message_service.go:623` bounds registry posting by caller cancellation and service lifetime.
   - `atlas/couch.md:393` documents the revised budget flow. No new CLI surface requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Passed targeted regressions normally and under race detection; all `TestMessage*` tests under race detection (12.379s); the entire `couchmessage` package under race detection (26.553s). Coverage includes slow authority, stalled registry posting, lookup timeout/cancellation, canonical refusals, and neighboring routing behavior. Repository-wide tests were not rerun. Checkout remained unchanged.

6. **Architecture:** ARCH-DRY passes: refusal bookkeeping reuses the existing snapshot. ARCH-PURE passes: context handling remains in the integration layer with injected authority and endpoint seams. ARCH-PURPOSE passes: the synchronous send-path sweep found no remaining independent budget that extends admission or refusal bookkeeping.

7. **Plan revisions:** None required. The checked implementation items match the code; landing and deferred operator validation remain explicitly separate.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4642de0f-97b0-49a4-a045-943d2191be42","repository":"github.com/xianxu/pair","issue_id":"000634","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e90c3fb728740e8a","reviewed_head":"65ee79d1138e818704d09a3e69b1b89deaa6834a","slot":"pair:3","requested_at":"2026-10-17T12:18:23.212685Z"},"phase":"finished","pid":63445,"started_at":"2026-10-17T12:18:23.354158Z","finished_at":"2026-10-17T12:20:10.063445Z","session_id":"01a149cc-8956-7490-a547-6b184294d50a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.332742875s
head: "65ee79d1138e818704d09a3e69b1b89deaa6834a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
