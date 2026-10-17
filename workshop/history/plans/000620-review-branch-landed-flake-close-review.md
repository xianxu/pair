# Boundary Review — pair#620 (whole-issue close)

| field | value |
|-------|-------|
| issue | 620 — Review branch restore intermittently misses landed acknowledgment |
| repo | pair |
| issue file | workshop/issues/000620-review-branch-landed-flake.md |
| boundary | whole-issue close |
| milestone | — |
| window | f99b62872d0d5816ad7e6eedaa9729ffdc5e3442..12d613dad701c542e87572a66425d5ddebd84f05 |
| command | sdlc close --issue 620 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T10:59:54-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change matches the revised contract: the fixture derives its completion bound from real resolver deadlines, retains the landed-artifact assertions, and captures failure evidence. No blocking defects found. The historical failure’s exact cause remains appropriately unclaimed.

Done-when: current — its reproducer, regression, diagnostics, and before/after requirements still cover the revised implementation.

```findings
{}
```

1. **Strengths**
   - Reuses the existing observation trace and real fixture.
   - Preserves context, branch restoration, CRLF/BOM, and recovery assertions (`tests/review-branch-restore-test.sh:130`).
   - Publication-refusal control verifies missing acknowledgment still fails with diagnostic evidence (`tests/review-branch-restore-diagnostics-test.py:49`).
   - Atlas documentation accurately explains the completion bound.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Normal fixture passed; acknowledgment took 0.758 seconds.
   - Both diagnostic controls passed; delayed acknowledgment took 6.54 seconds.
   - Restoring the five-second bound in a disposable copy made the delayed-success regression fail at the acknowledgment assertion. Copy removed afterward.
   - Both fixture guard tests and diff whitespace checks passed.
   - Inspected the recorded full-suite log and verified its SHA256 matches the tracker. Full suite was not rerun during review.
   - Tracked checkout remains clean.

6. **Architecture**
   - **ARCH-DRY: pass** — shared fixture and trace reused.
   - **ARCH-PURE: pass** — controlled IO stays in integration fixtures; production logic unchanged.
   - **ARCH-PURPOSE: pass** — measured timing mismatch is reproduced, repaired, and checked against publication failure.
   - Docs gate passes: atlas updated; no new user-facing surface requires README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"53d39fbf-6f01-4c70-ad29-833939118b2f","repository":"github.com/xianxu/pair","issue_id":"000620","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/05558da86317eebe","reviewed_head":"12d613dad701c542e87572a66425d5ddebd84f05","slot":"pair:3","requested_at":"2026-10-17T17:56:58.439351Z"},"phase":"finished","pid":9438,"started_at":"2026-10-17T17:56:58.719515Z","finished_at":"2026-10-17T17:59:52.833358Z","session_id":"01a14b02-8e2f-79e2-8187-30dd7e418dd0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.257207708s
head: "12d613dad701c542e87572a66425d5ddebd84f05"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
