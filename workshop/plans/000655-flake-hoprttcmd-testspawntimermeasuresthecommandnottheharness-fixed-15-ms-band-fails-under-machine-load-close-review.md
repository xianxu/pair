# Boundary Review — pair#655 (whole-issue close)

| field | value |
|-------|-------|
| issue | 655 — Flake: hoprttcmd TestSpawnTimerMeasuresTheCommandNotTheHarness fixed 15 ms band fails under machine load |
| repo | pair |
| issue file | workshop/issues/000655-flake-hoprttcmd-testspawntimermeasuresthecommandnottheharness-fixed-15-ms-band-fails-under-machine-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | d5cc10b581191ca33f59f8fe0f066002fd52c4bc..2481722ff4e7b7ddd282c0d54b375289679f65f5 |
| command | sdlc close --issue 655 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T18:15:12-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change removes the machine-speed assumption while preserving production timing and failure semantics. Independent tests confirm the regression detects incorrect timer boundaries and subprocess clocks. No blocking findings.

Done-when: current — both load independence and subprocess-clock rejection remain applicable and were independently verified.

```findings
{}
```

1. Strengths
   - `hoprtt.go:117–135` shares the actual production sampling loop with the regression test.
   - `hoprtt_test.go:59–95` checks exact durations, sequential execution, and failure accounting across three preparation delays.
   - The shipped-binary spawn smoke retains output validation without a latency ceiling.
   - Atlas documentation accurately describes the new seam; no user-facing surface requires a README update.

2. Critical findings: none.

3. Important findings: none.

4. Minor findings: none.

5. Test coverage notes
   - Package race tests passed.
   - Regression passed 1,000 repetitions alongside 20 live CPU workers; all workers were reaped.
   - Temporary-overlay mutations including preparation time, omitting execution, and using subprocess clocks each failed as expected.
   - Full repository suite was not rerun during this review. Checkout remained unchanged.

6. Architectural notes
   - **ARCH-DRY: pass** — production and tests use one sampling loop.
   - **ARCH-PURE: pass** — command creation remains a thin adapter; execution is injected into the clock-dependent sampler.
   - **ARCH-PURPOSE: pass** — both original acceptance requirements are satisfied.

7. Plan revision recommendations: none.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"eb63c2a7-209b-44c1-8aa7-5a82df31bb27","repository":"github.com/xianxu/pair","issue_id":"000655","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e9998489f02bb29f","reviewed_head":"2481722ff4e7b7ddd282c0d54b375289679f65f5","slot":"pair:3","requested_at":"2026-10-18T01:13:59.508035Z"},"phase":"finished","pid":68389,"started_at":"2026-10-18T01:13:59.736776Z","finished_at":"2026-10-18T01:15:11.423781Z","session_id":"01a14c92-a055-7ef1-9975-5ece89d08c89","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.393851666s
head: "2481722ff4e7b7ddd282c0d54b375289679f65f5"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
