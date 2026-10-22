# Boundary Review — pair#762 (whole-issue close)

| field | value |
|-------|-------|
| issue | 762 — cast link locks: lifetime guess cap per link and an operator notice on forced code rotation |
| repo | pair |
| issue file | workshop/issues/000762-cast-link-locks-lifetime-guess-cap-per-link-and-an-operator-notice-on-forced-code-rotation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..201f005e08f8bfb425cca988e46d3fd364e956f0 |
| command | sdlc close --issue 762 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:48:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the Spec as clarified by Revisions. The third forced rotation blocks further unlocks, existing sessions survive, and only successful operator rotation restores admission. Notice ordering and documentation are consistent with that behavior. No blocking findings.

Done-when: current — its threshold, refusal, session-survival, recovery, notice, and documentation criteria cover the revised contract; tests additionally pin successful-unlock persistence.

```findings
{}
```

1. **Strengths**
   - Cap enforcement and recovery remain atomic under the existing mutex (`cmd/internal/castlock/castlock.go:162`, `:221`).
   - Tests exercise both code lengths, exact thresholds, correct-code refusal, entropy failures, session survival, and restored budgets.
   - Notices publish under the generation guard, preventing older callbacks from overwriting cap advice (`cmd/internal/couchtty/console_castcodes.go:488`).
   - README and atlas document the lifetime policy and operator recovery.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   
   Passed:
   - Complete `castlock` race suite.
   - Targeted Couch rotation, notice, concurrency, and menu tests under race detection.
   - Broadcast and webcast lock/code/rotation tests under race detection.
   - Pinned-range whitespace check.

   All five live link consumers route through the shared lock. The checkout remained unchanged. No full repository suite was run.

6. **Architectural notes**
   - **ARCH-DRY — pass:** one shared cap implementation; notice publication reuses the existing feed seam.
   - **ARCH-PURE — pass:** state transitions stay in the lock owner with injected randomness, limiter, and callback; Couch retains presentation work.
   - **ARCH-PURPOSE — pass:** all live link kinds inherit enforcement, and notices expose both forced changes and cap recovery.
   - **ARCH-EVOLVE — pass:** no persisted formats change; the counter and event field are in memory.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"72834da7-5a46-4c65-8ff1-98874956be1d","repository":"github.com/xianxu/pair","issue_id":"000762","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/baab0f35b87f11f3","reviewed_head":"201f005e08f8bfb425cca988e46d3fd364e956f0","slot":"pair:4","requested_at":"2026-10-21T09:47:18.781212Z"},"phase":"finished","pid":18130,"started_at":"2026-10-21T09:47:19.024627Z","finished_at":"2026-10-21T09:48:31.796072Z","session_id":"01a15ddb-aa7e-7283-b1b2-0b0261693173","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.395407042s
head: "201f005e08f8bfb425cca988e46d3fd364e956f0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
