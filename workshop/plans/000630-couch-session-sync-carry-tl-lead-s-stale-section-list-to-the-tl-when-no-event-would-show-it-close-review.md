# Boundary Review — pair#630 (whole-issue close)

| field | value |
|-------|-------|
| issue | 630 — couch session-sync: carry tl-lead's stale-section list to the TL when no event would show it |
| repo | pair |
| issue file | workshop/issues/000630-couch-session-sync-carry-tl-lead-s-stale-section-list-to-the-tl-when-no-event-would-show-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | f823355f9b9988dfb9888eed1bac54de44edc10f..4b330574b832255236b1d146d82c4b278e48e735 |
| command | sdlc close --issue 630 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T07:01:19-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main reminder flow matches the Spec, and both affected packages and focused race tests pass. One blocking edge remains: registered-only inspection can enter the legacy document path after a registration disappears, introducing an uncancellable wait into the worker and service shutdown.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: registered-inspection-fails-closed
    title: |
      Reject missing registrations before entering legacy document IO.
    detail: |
      cmd/internal/sessionsync/registry.go:36 calls withDocument, which falls back to locked(path, fn) when the registration is absent (lines 581–583). The rejection at line 37 therefore occurs only after legacy file loading and an unbounded LOCK_EX acquisition (sync.go:145–150). If a file is forgotten between worker enumeration and inspection and its legacy flock is held, the reminder worker blocks while holding the registry lock; neither its 30-second context nor service cancellation can interrupt it. ARCH-CONSTRAINTS/ARCH-ORDER: add a registered-only route that rejects absence under the registry lock before legacy IO, and regress inspection of an unregistered file with a held legacy flock.
```

1. **Strengths**

   - Freshness policy remains owned by tl-lead; Couch only validates output framing.
   - The production transition model directly tests cooldowns, invalidated observations, cancellation, and uncertain sends.
   - Registered-document tests compare document, shadow, state, and registry snapshots to establish baseline preservation.
   - README and atlas document cadence, lock suppression, command bounds, and restart behavior.

2. **Critical findings**

   - [registry.go:36](cmd/internal/sessionsync/registry.go:36): missing registration rejection happens after the legacy fallback. Add a registered-only mode to the shared loader, preserving shared metadata-loading logic.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Complete `sessionsync` and `couchcmd` package tests.
   - Focused `InspectRegistered|SessionStale` tests, including race checks.
   - `git diff --check`.

   Existing unknown/forgotten-registration tests assert eventual refusal but do not hold the legacy flock, so they miss the blocking path. The opt-in live conformance fixture was inspected but not rerun; this checkout lacks its required already-built Couch binary.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses freshness, identity, broker, and registry authorities.
   - **ARCH-PURE — pass:** classification and scheduling transitions have direct tests without IO.
   - **ARCH-PURPOSE — pass:** production startup and wait ticks reach the reminder worker.
   - **ARCH-MOCK — pass:** injected checker, stateful endpoint fixtures, and documented real conformance triggers.
   - **ARCH-CONSTRAINTS — flag:** legacy fallback defeats the worker’s bounded execution.
   - **ARCH-SECURE — flag:** absent registration must be rejected before entering the legacy path’s unbounded file reads.
   - **ARCH-ORDER — flag:** registration disappearance can strand cancellation and shutdown.
   - **ARCH-FUNERAL — pass:** cooldowns are pruned, runtime state dies with the service, and command descendants are collected.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that registered inspection rejects missing authority before legacy IO, with a held-flock regression covering prompt refusal and worker cancellation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0c1056f8-af6c-42a9-9643-95d3219af226","repository":"github.com/xianxu/pair","issue_id":"000630","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7d6f54f42e7a59c1","reviewed_head":"4b330574b832255236b1d146d82c4b278e48e735","slot":"pair:6","requested_at":"2026-10-17T13:59:03.900214Z"},"phase":"finished","pid":81446,"started_at":"2026-10-17T13:59:04.170244Z","finished_at":"2026-10-17T14:01:18.498979Z","session_id":"01a14a28-b70e-7df3-b7f9-816ce57d9a31","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.7881455s
head: "4b330574b832255236b1d146d82c4b278e48e735"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
