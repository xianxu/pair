# Boundary Review — pair#686 (whole-issue close)

| field | value |
|-------|-------|
| issue | 686 — session-watch busy-loops on freshly started Codex threads (45-80% CPU each) |
| repo | pair |
| issue file | workshop/issues/000686-session-watch-busy-loops-on-freshly-started-codex-threads-45-80-cpu-each.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5e899a871e27dd8126c512da08e069e6c3a99708..620dd8b23f01b05b94e5f6ab05430c4e5fbbd3b0 |
| command | sdlc close --issue 686 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:30:56-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation removes corpus discovery from bound follow, preserves guarded transcript reads, and passes all four relevant package suites. One documentation contradiction blocks this boundary under the explicitly required Core concepts cross-check; I found no additional runtime defect. Live CPU acceptance remains deferred by the plan’s approved post-landing revision.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-code-traceability
    title: |
      Core concepts table contradicts the delivered entity locations and ownership
    detail: |
      workshop/plans/000686-bound-watcher-plan.md:23 places lifecycle cadence in sessionwatch/cadence.go or run.go, but nextLifecycleDelay is implemented in cmd/internal/sessionwatch/bound_state.go:84. Line 24 also assigns scanner state to the cursor, whereas TranscriptCursor explicitly excludes it and boundCheckpoint owns it. Update the table and add a Revisions entry naming the actual entities, locations and PURE/INTEGRATION split (ARCH-PURE). This is Critical under the review's explicit table/code contradiction rule; it is not a demonstrated runtime failure.
```

1. **Strengths**

   - Bound follow exits discovery permanently and reads only the authorized transcript (`bound_follow.go:28`).
   - The cursor freezes each read endpoint and enforces the payload budget while preserving partial records (`transcript_cursor.go:134`).
   - Publication advances the acknowledged prefix and commits the cursor only after the complete batch (`bound_state.go:68`).
   - Exact-owner caching retains metadata fencing inside the existing writer lock; append and replacement regressions exercise rejection of stale authority.

2. **Critical findings**

   - The Core concepts mismatch above needs a plan-only correction. Cadence belongs at `cmd/internal/sessionwatch/bound_state.go:84`; scanner ownership belongs to `boundCheckpoint`, separate from `TranscriptCursor`.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/sessioninventory ./cmd/internal/artifactpath`
   - `git diff --check` for the pinned range.

   Coverage includes counted 4k/8k idle workloads, partial framing, cancellation, mutation guards, stale acknowledgments, publication uncertainty, and owner-cache invalidation. I did not independently run the repository-wide recorded suite or live CPU acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** title projection and lifecycle follow share the guarded cursor and owner reducer.
   - **ARCH-PURE — flag:** implementation separates pure transitions from IO; the Core concepts table misstates that ownership.
   - **ARCH-PURPOSE — pass:** addresses measured bound-follow work within the approved scope.
   - **ARCH-MOCK — pass:** relevant IO uses existing runtime seams and stateful filesystem fakes.
   - **ARCH-CONSTRAINTS — pass:** bounded payload reads, idle backoff, and counted workload growth checks.
   - **ARCH-SECURE — pass:** identity, mutation, owner and lock fences remain exercised.
   - **ARCH-ORDER — pass:** production publication uses the reducer’s staged cursor and acknowledged-prefix transitions.
   - **ARCH-FUNERAL — pass:** new cursor/cache state dies with its owner; no new durable artifact family.

   Atlas documents the new flow and latency. No new user command, flag or configuration key requires a README update.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting the Core concepts table to the delivered locations and ownership, including `reduceBoundFollow`, `nextLifecycleDelay`, `boundCheckpoint`, and the cursor’s separate IO methods. No wording-presence test is needed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d685340f-132b-490d-8c2c-d0eadb2721dd","repository":"github.com/xianxu/pair","issue_id":"000686","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c04461cb41ebdc50","reviewed_head":"620dd8b23f01b05b94e5f6ab05430c4e5fbbd3b0","slot":"pair:7","requested_at":"2026-10-19T02:28:41.501493Z"},"phase":"finished","pid":19411,"started_at":"2026-10-19T02:28:41.609157Z","finished_at":"2026-10-19T02:30:45.92295Z","session_id":"01a151fd-5f12-7963-a09e-54fd4a655755","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.718496792s
head: "620dd8b23f01b05b94e5f6ab05430c4e5fbbd3b0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
