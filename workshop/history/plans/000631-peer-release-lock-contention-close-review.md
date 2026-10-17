# Boundary Review — pair#631 (whole-issue close)

| field | value |
|-------|-------|
| issue | 631 — Peer promotion holds broker lock during wrapper release RPC |
| repo | pair |
| issue file | workshop/issues/000631-peer-release-lock-contention.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2c2d6a4a3192ad9dd289a0d13441a64209f2863e..2e1ab24607e87b232bcb6846081e5dcce09acad3 |
| command | sdlc close --issue 631 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T02:01:49-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change removes promotion cleanup IO from the shared mutex while preserving reservation ownership, endpoint identity, and retry behavior. The regression tests detect both the original contention and premature reservation clearing. No blocking findings.

Done-when: current — the criteria still cover independent Send/Status progress and promotion, reconnect, duplicate-ID, and race controls; the revisions add verification details without changing the delivered behavior.

```findings
{}
```

1. **Strengths**
   - Cleanup releases and reacquires the mutex around only the bounded RPC ([broker.go:801](/Users/xianxu/Library/Caches/sdlc/review-checkouts/c7e9f43b993f14b2/cmd/internal/couchmessage/broker.go:801)).
   - Deferred cleanup clears only the matching reservation on the original actor ([broker.go:776](/Users/xianxu/Library/Caches/sdlc/review-checkouts/c7e9f43b993f14b2/cmd/internal/couchmessage/broker.go:776)).
   - Stateful gated tests exercise unrelated receiver progress, sender cancellation, and replacement during cleanup.
   - The atlas accurately documents the resulting ownership and concurrency behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed `go test -race ./cmd/internal/couchmessage -timeout=120s` in 23.669s.
   - Temporary overlays independently confirmed that restoring the original code fails the contention regression, and restoring early reservation clearing fails the ownership regression.
   - The claimed full-suite tested commit differs from the reviewed head only in issue documentation. Its recorded suite log was unavailable locally; I did not independently reproduce that full-suite result.

6. **Architecture**
   - **ARCH-DRY: pass.** Reuses the existing actor reservation, endpoint interface, and stateful test helpers.
   - **ARCH-PURE: pass.** IO remains in broker orchestration; pure model transitions remain unchanged.
   - **ARCH-PURPOSE: pass.** Inspected all four broker `Release` sites; none now holds the shared mutex during release IO.
   - Atlas coverage is present. No new command, flag, configuration, or usage surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"424f0d30-a15f-407b-8802-30c85c651391","repository":"github.com/xianxu/pair","issue_id":"000631","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c7e9f43b993f14b2","reviewed_head":"2e1ab24607e87b232bcb6846081e5dcce09acad3","slot":"pair:3","requested_at":"2026-10-17T09:00:44.695181Z"},"phase":"finished","pid":74010,"started_at":"2026-10-17T09:00:44.951705Z","finished_at":"2026-10-17T09:01:48.286663Z","session_id":"01a14917-982a-7993-a1f1-465b0fb8949a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.952318666s
head: "2e1ab24607e87b232bcb6846081e5dcce09acad3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
