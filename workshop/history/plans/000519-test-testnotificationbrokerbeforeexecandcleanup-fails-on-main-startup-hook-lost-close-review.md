# Boundary Review — pair#519 (whole-issue close)

| field | value |
|-------|-------|
| issue | 519 — test: TestNotificationBrokerBeforeExecAndCleanup fails on main ('startup hook lost') |
| repo | pair |
| issue file | workshop/issues/000519-test-testnotificationbrokerbeforeexecandcleanup-fails-on-main-startup-hook-lost.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1ef66ff23734f63b866852c147cbe78f4c356704..cbae39e8c2d71559725afa9c0e5c92a4ebf3a351 |
| command | sdlc close --issue 519 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T18:53:24-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The fix addresses both notification-loss boundaries: kernel-queued datagrams and buffered messages at wrapper EOF. Independent tests confirm the intended behavior and fail when the fixes are reverted. No blocking findings.

1. **Strengths**
   - `notifytransport/transport.go:133` shares validation and admission between normal reception and shutdown.
   - `notifytransport/transport.go:183` joins the receiver before draining, preserves bounded admission, and retains idempotent cleanup.
   - `wrapcmd/wrap.go:3304` consumes pending notifications before finishing stdout through the existing output owner.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Both affected packages passed. The startup test passed 20 consecutive repetitions; focused broker tests also passed 20 repetitions. Reverting behavior through an overlay made both kernel-backlog and EOF regressions fail. Missing generated assets were generated only in a disposable copy, subsequently removed. The reviewed checkout remains unchanged.

6. **Architectural notes:** **ARCH-DRY: pass**—shared validation. **ARCH-PURE: pass**—transport IO remains contained; no business logic moved into it. **ARCH-PURPOSE: pass**—both pending-work boundaries are covered without timing relaxation. Atlas documents shutdown semantics; no new README-facing surface exists.

7. **Plan revision recommendations:** None. The implementation matches the revised plan; source history supports the documented introduction and cause.

```findings
{}
```
