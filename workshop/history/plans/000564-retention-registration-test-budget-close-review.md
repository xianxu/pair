# Boundary Review — pair#564 (whole-issue close)

| field | value |
|-------|-------|
| issue | 564 — Retention resume test expires its registration budget under load |
| repo | pair |
| issue file | workshop/issues/000564-retention-registration-test-budget.md |
| boundary | whole-issue close |
| milestone | — |
| window | e85237e7e1951bc61ce3d9527ed7ad2dfb6e47d3..f6ba7bcb5fd0f9339ad2b5542916c87c7e12a260 |
| command | sdlc close --issue 564 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T02:15:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies the issue’s Spec and Plan. It removes the retention test’s incidental 200 ms registration deadline while preserving bounded cancellation, both retention modes, and all original assertions. No defects or documentation gaps found.

```findings
{}
```

1. **Strengths**
   - `retention_resume_test.go:12–28` reuses the existing fixture and deadline seam, with the caller bounded by `testwait.Budget`.
   - `retention_resume_test.go:33–50` deterministically reproduces expiry before any completed ownership observation.
   - `retention_test.go:351–407` preserves exact last-use, process-list, and environment assertions for explicit and background resume.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently passed targeted semantic, timeout, and lifecycle tests (7.093 s), targeted race tests (11.805 s), and the full couchcore package (408.393 s). Deadline inheritance and cancellation are exercised through the real resume path. Diff whitespace checks passed; checkout remained clean.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Reuses `warmDetachedThread`, `newSpendableBudget`, and `testwait.Budget`.
   - **ARCH-PURE: pass.** Test timing policy stays in the fixture; existing injected boundaries exercise integration behavior.
   - **ARCH-PURPOSE: pass.** Covers controlled expiry, parent deadline propagation, cancellation, and both retention modes. Production timeouts remain unchanged.
   
   No new architectural or user-facing surface requires atlas or README updates.

7. **Plan revision recommendations:** None. The original broad-run failure remains documented separately from passing reruns.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.968875209s
head: "f6ba7bcb5fd0f9339ad2b5542916c87c7e12a260"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T05:18:55-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 564 — Retention resume test expires its registration budget under load |
| repo | pair |
| issue file | workshop/issues/000564-retention-registration-test-budget.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9fdd548406119c50ab8a42a9ac4e63b2605568a3..8e46802a30902440f303211c2a029a8a16fc239a |
| command | sdlc close --issue 564 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T05:18:55-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies the Spec and revised Plan. It removes the retention test’s incidental 200 ms deadline, preserves cancellation and both retention modes, and leaves production behavior unchanged. Independent verification found no blocking defects.

Done-when: current — its clauses still cover expiry characterization, bounded semantic testing, and lifecycle/package verification after the attachment-route revision.

```findings
{}
```

1. **Strengths**
   - `retention_resume_test.go:13–28` reuses the existing fixture and context seam with an enclosing test budget.
   - `retention_resume_test.go:34–77` deterministically characterizes expiry in both current attachment and ownership-registration paths.
   - `retention_resume_test.go:133–141` verifies cancellation before cleanup can mask detached context ancestry.
   - `retention_test.go:351–407` preserves exact last-use, process-list, and child-environment assertions for both modes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused semantic, timeout, and lifecycle tests passed three repetitions: 9.708 s; under `-race`: 13.408 s.
   - Full couchcore package passed: 416.480 s.
   - Temporary overlays independently confirmed that restoring the 200 ms timer fails deadline inheritance, and detaching the parent fails cancellation propagation.
   - Checkout remained clean at the pinned head.

6. **Architectural notes**
   - **ARCH-DRY: pass.** Reuses existing fixture, spendable-budget, and test-budget helpers.
   - **ARCH-PURE: pass.** Integration tests use injected boundaries; no production logic or IO coupling added.
   - **ARCH-PURPOSE: pass.** Addresses the timing hazard while retaining both semantic modes and intentional timeout coverage.
   - No new architectural or user-facing surface requires atlas or README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6eca8fb3-d11f-4652-b8d8-fb8163f4435d","repository":"github.com/xianxu/pair","issue_id":"000564","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/48d5081c1083f537","reviewed_head":"8e46802a30902440f303211c2a029a8a16fc239a","slot":"pair:3","requested_at":"2026-10-15T12:10:26.498956Z"},"phase":"finished","pid":97439,"started_at":"2026-10-15T12:10:26.792723Z","finished_at":"2026-10-15T12:18:54.620725Z","session_id":"01a13f78-8c2d-7ee3-b3f6-4b7a3db1f390","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.198585625s
head: "8e46802a30902440f303211c2a029a8a16fc239a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
