# Boundary Review — pair#884 (whole-issue close)

| field | value |
|-------|-------|
| issue | 884 — flake: TestSessionTunnelExitEndsSession |
| repo | pair |
| issue file | workshop/issues/000884-flake-testsessiontunnelexitendssession.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4ae4b15e1377c9c1bb3913505da18cac9f264b9a..1650258c2af2a887e8d177088bed34e5bfe31844 |
| command | sdlc close --issue 884 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-25T08:20:18-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fixes shutdown ownership with a small, consistent implementation. Independent tests confirm the regression detects the original defect and passes with the fix. No blocking findings.

Done-when: current — it covers shutdown joining, repeated calls, generation isolation, and retained lifecycle behavior; the checklist revision does not expand executable scope.

```findings
{}
```

1. **Strengths**
   - `host.go:254–258` retains the exact close receipt before publishing mount loss.
   - `host.go:365–378` joins outside the mutex and preserves concurrent idempotence.
   - `host_shutdown_test.go:29` deterministically covers tunnel loss, server loss, last unmount, repeated callers, and generation isolation.
   - Atlas documentation describes the new lifetime invariant. No new commands, flags, or configuration require README changes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Eight focused lifecycle tests passed normally and under race detection, each at `count=1`.
   - Six additional shared-host, rotation, and cancellation tests passed.
   - The new regression failed against base `host.go` in all three shutdown cases using a temporary Go overlay.
   - Reader inventory and diff whitespace checks passed. Checkout remains unchanged.
   - The separately queued affected run remains a landing prerequisite; its completion was not verified here.

6. **Architectural notes**
   - **ARCH-DRY: pass** — reuses the existing close attempt and fake handle.
   - **ARCH-PURE: pass** — change stays within the existing lifecycle shell; tests inject the close boundary.
   - **ARCH-PURPOSE: pass** — both routing-removal paths retain the applicable receipt.
   - **ARCH-EVOLVE: pass** — no persisted formats change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4872d1c3-0b9c-46ad-af86-03ebec49d17d","repository":"github.com/xianxu/pair","issue_id":"000884","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/44ffc73c2e0aa7c5","reviewed_head":"1650258c2af2a887e8d177088bed34e5bfe31844","slot":"pair:9","requested_at":"2026-10-25T15:19:03.449372Z"},"phase":"finished","pid":64721,"started_at":"2026-10-25T15:19:03.562139Z","finished_at":"2026-10-25T15:20:16.633933Z","session_id":"01a173a4-d209-7130-b705-f7a6ae75a768","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.260011125s
head: "1650258c2af2a887e8d177088bed34e5bfe31844"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
