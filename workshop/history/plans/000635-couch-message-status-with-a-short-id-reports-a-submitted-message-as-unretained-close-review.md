# Boundary Review — pair#635 (whole-issue close)

| field | value |
|-------|-------|
| issue | 635 — couch --message-status with a short ID reports a submitted message as unretained |
| repo | pair |
| issue file | workshop/issues/000635-couch-message-status-with-a-short-id-reports-a-submitted-message-as-unretained.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5d2145d5445b06786704af921311360f696dace0..890c9f34b9d4927eb56cc42d89f0696812925b24 |
| command | sdlc close --issue 635 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T08:10:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fulfills the issue’s purpose: visible prefixes resolve correctly, ambiguity lists only visible full IDs, and misses no longer imply lost delivery. Exact-ID recovery and uncertainty handling remain intact. No blocking findings.

Done-when: current — its three clauses still cover the implemented acceptance behavior; the revision clarifies gate actions without changing scope.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/couchmessage/broker.go:401` centralizes visibility and waiting projections for exact and prefix reads.
   - `cmd/internal/couchmessage/broker.go:856` checks exact matches first, resolves prefixes under the lock, and sorts ambiguity candidates.
   - `cmd/internal/couchcmd/message_status_prefix_test.go:17` verifies submitted full-ID and eight-character-prefix queries over the real Unix socket.
   - README, atlas, and CLI help document the new lookup behavior and recovery limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Full `couchmessage` race suite passed: 27.514 seconds.
   - All `TestMessage*` CLI tests passed under race detection: 12.412 seconds.
   - Tests cover visibility, exact precedence, sorted ambiguity, unknown lookup, terminal/refusal expiry, queue projections for sender and recipient, and existing recovery behavior.
   - All four new tests failed with the base broker implementation substituted through a temporary Go overlay, confirming regression sensitivity.
   - Repository-wide tests were not rerun; the checkout remained unchanged.

6. **Architectural notes**
   - **ARCH-DRY: pass** — both lookup paths share visibility and projection logic.
   - **ARCH-PURE: pass** — the change remains within the broker’s existing integration boundary, uses injected endpoints, and adds no IO to prefix selection.
   - **ARCH-PURPOSE: pass** — both retained receipt stores participate; the change delivers resolution, ambiguity, and unknown handling.
   - No Core concepts table requires reconciliation. README and atlas gates are satisfied.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6cc50b19-9a6e-4773-8b91-ed227af2d9f6","repository":"github.com/xianxu/pair","issue_id":"000635","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/35881cb76fe388e9","reviewed_head":"890c9f34b9d4927eb56cc42d89f0696812925b24","slot":"pair:3","requested_at":"2026-10-17T15:08:23.724102Z"},"phase":"finished","pid":84243,"started_at":"2026-10-17T15:08:23.84652Z","finished_at":"2026-10-17T15:10:06.55039Z","session_id":"01a14a68-2eb6-78d1-8874-37d4b104b183","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.74861375s
head: "890c9f34b9d4927eb56cc42d89f0696812925b24"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
