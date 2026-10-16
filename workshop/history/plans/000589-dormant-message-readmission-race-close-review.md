# Boundary Review — pair#589 (whole-issue close)

| field | value |
|-------|-------|
| issue | 589 — Dormant message readmission test races its last asynchronous attempt |
| repo | pair |
| issue file | workshop/issues/000589-dormant-message-readmission-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1a1047c89126c68fa1f9c2651bff36663f6ab656..8a00404d7ae9a70daba1485be3b595fb6661f7f8 |
| command | sdlc close --issue 589 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T02:09:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The fixture now establishes send-triggered readmission before restoring healthy launch evidence, removing the elapsed-sleep race. Production behavior is unchanged. The implementation matches the Spec and Plan; no blocking findings remain.

```findings
{}
```

1. **Strengths**
   - Retry callbacks synchronize the ladder without a fixed delay (`message_service_test.go:465`).
   - The admission barrier ensures the triggering send returns `unavailable` before connection can succeed (`:443`, `:480–487`).
   - Deferred release and context cancellation bound cleanup and failure paths (`:436`, `:447`, `:463`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed 100 isolated repetitions under the race detector.
   - Message and Registry tests passed under the race detector.
   - Independently reran the controlled late-final-retry reproduction: the original assertion failed with `accepted`, as expected.
   - Independently disabled targeted readmission through an overlay: the fixed test failed within its deadline, confirming meaningful coverage.
   - Inspected the recorded full-suite log and verified its SHA-256. Its tested revision has no non-tracker differences from the pinned head.

6. **Architecture**
   - **ARCH-DRY: pass** — reuses existing authority and retry seams.
   - **ARCH-PURE: pass** — integration synchronization stays in the fixture; production separation is preserved.
   - **ARCH-PURPOSE: pass** — demonstrates the race and verifies unavailable-send followed by successful readmission.
   - No new architectural or user-facing surface requires atlas or README changes.

7. **Plan revisions:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"31e0553c-15d9-45c4-8bac-6be815d9aab6","repository":"github.com/xianxu/pair","issue_id":"000589","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/725b8bb27e877b99","reviewed_head":"8a00404d7ae9a70daba1485be3b595fb6661f7f8","slot":"pair:1","requested_at":"2026-10-16T09:07:41.866516Z"},"phase":"finished","pid":93098,"started_at":"2026-10-16T09:07:42.049664Z","finished_at":"2026-10-16T09:09:02.072231Z","session_id":"01a143f7-989a-7930-83dc-9971b46e55c3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.273484958s
head: "8a00404d7ae9a70daba1485be3b595fb6661f7f8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
