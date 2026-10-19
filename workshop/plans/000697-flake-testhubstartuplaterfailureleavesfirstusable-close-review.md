# Boundary Review — pair#697 (whole-issue close)

| field | value |
|-------|-------|
| issue | 697 — flake: TestHubStartupLaterFailureLeavesFirstUsable |
| repo | pair |
| issue file | workshop/issues/000697-flake-testhubstartuplaterfailureleavesfirstusable.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8c2b04a6c81bb4aae43df070110866ebdd8a38e5..f6c15b64cd32332e81f197a16300fdabde99fbd3 |
| command | sdlc close --issue 697 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T07:44:05-07:00 |
| verdict | SHIP |

## Review

Tests pass. Next I'm putting back the old order-dependent injection to check that the new variant fails without the fix.

```verdict
verdict: SHIP
confidence: high
```

This fixes the flake at its cause. `FakeRunner.FailNextStart` failed whichever child started next. The test has two independent starters, the explicit hub queue and the warm-only pass, so the warm worker sometimes took the injected failure. The new `FailNextStartIn(dir, err)` fails only a start in the given directory, and the test now aims it at `paths[1]`. I checked this against the code myself:
- `go test -run TestHubStartup -count=5` passes, and `go vet` is clean.
- With the old `FailNextStart(...)` call put back, the new held-order variant fails 2 out of 2 runs with the CI message (`second launch did not report its injected refusal: <nil>`). So the regression test does fail without the fix. I restored the file afterwards.
- The log says there are 14 other `FailNextStart` callers. A grep agrees: 15 hits, one of which is the definition.

Only Minor test-fixture nits remain. Nothing blocks shipping.

**1. Strengths**
- `cmd/internal/couchcore/runner_fake.go:84-91`: the failure is tied to the start it is meant to fail, not to start order. The fix is in the fixture, and production ordering is left alone, which is correct since the two starters are independent by design.
- `hubs_startup_test.go:196-206`: both start orders are covered. The original test keeps the natural order, and the new variant forces the order that failed in CI. Both run through one shared function, `testHubStartupLaterFailure`, so the assertions are not duplicated.
- The held-order variant fails every time on the old code, so the regression is reproducible rather than a timing guess.
- The class sweep in the Log is accurate. The other 14 callers each arm the failure before a single synchronous start, so none can hit this race.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- `hubs_startup_test.go:211-223`: the warm-first wrapper wraps `r.console.Ops()`. That is already the rig's wrapper (`hubs_startup_test.go:125-133`), which records history and sends to `r.calls`. The new wrapper sends to `r.calls` a second time, so every dispatched call lands in the channel twice. This is harmless today because there are few calls, `await` skips non-matching names and the buffer holds 32. It is still redundant: drop the extra send and just `return actual(call)`, as `TestHubStartupDetachedExtraLaunchesWithoutDuplicateWarmAttempt` does at line 247. This is the only instance in the window.
- `hubs_startup_test.go:214`: `testwait.Until(t, …)` runs inside the operation dispatcher, which is called from the console's operation goroutine, not the test goroutine. If it times out, `t.Fatalf` → `FailNow` runs off the test goroutine, which Go's testing package does not allow. The test would only fail later through `await`'s timeout, with a misleading message. Use a non-fatal `testwait.Wait` (or a channel barrier) inside the dispatcher instead. Only one instance.
- `runner_fake.go:81`: the doc says "the next Start in dir", but the check sits in the shared `start` helper, so it also covers `StartBlocked`. Wording nit.

**5. Test coverage**
- The fix's regression test is shown to fail without the fix.
- `FailNextStartIn` has no direct unit test in couchcore, for example one that starts in another directory and checks it is not consumed. The couchcmd test covers it end to end, and the helper is small enough that this is acceptable.

**6. Architecture**
- **ARCH-DRY: pass.** The two test variants share one function. The dispatcher-wrapping pattern repeats from the rig but with different behaviour each time, and the duplicate send is noted under Minor.
- **ARCH-PURE: pass.** The change only touches a test double and a test, with no business logic mixed with IO.
- **ARCH-PURPOSE: pass.** The root cause is fixed with no retry loop and no longer timeout, and the class sweep has been enumerated and checked.
- **ARCH-EVOLVE: pass.** No persisted format is touched.

**7. Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: test-dispatcher-wrap-reuse
    title: |
      Warm-first dispatcher wrapper sends each call to r.calls twice
    detail: |
      The wrapper at hubs_startup_test.go:211-223 wraps r.console.Ops(), which is already the rig wrapper at lines 125-133 that sends to r.calls, so it should just return actual(call), as line 247 does. This is the only instance in the window.
  - id: new
    severity: Minor
    kind: defect
    family: fatal-off-test-goroutine
    title: |
      testwait.Until (t.Fatalf) is called from the console operation goroutine
    detail: |
      At hubs_startup_test.go:214 the barrier runs inside the dispatcher, off the test goroutine, so a timeout calls FailNow from the wrong goroutine. Use non-fatal testwait.Wait or a channel barrier there. This is the only instance in the window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6a5fbe0d-2ea5-4bcf-9d2e-bba29b183ffb","repository":"github.com/xianxu/pair","issue_id":"000697","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/759bfdce717ff848","reviewed_head":"f6c15b64cd32332e81f197a16300fdabde99fbd3","slot":"pair:2","requested_at":"2026-10-19T14:42:10.053409Z"},"phase":"finished","pid":68726,"started_at":"2026-10-19T14:42:10.659876Z","finished_at":"2026-10-19T14:44:04.327197Z","session_id":"81164074-ff57-422e-a534-ea29187ee4ae","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 5.071623292s
head: "f6c15b64cd32332e81f197a16300fdabde99fbd3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
