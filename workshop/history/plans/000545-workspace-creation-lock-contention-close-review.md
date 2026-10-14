# Boundary Review — pair#545 (whole-issue close)

| field | value |
|-------|-------|
| issue | 545 — couchcore: eliminate spurious workspace creation lock contention |
| repo | pair |
| issue file | workshop/issues/000545-workspace-creation-lock-contention.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1b371441bf14dffef0480d7bb2643b46c2daa4b8..ff8a74266d307053656a68326072e243a3ae06fb |
| command | sdlc close --issue 545 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T10:03:50-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The change implements the revised release-aware retry contract while preserving kernel-enforced exclusion. Fresh targeted tests pass, and the recorded comparison logs support the reported counts. The controlled fixture demonstrates the fork-inheritance failure mechanism; the historical natural failures were not reproduced, so the evidence does not establish a natural failure-rate improvement. No blocking findings.

```findings
{}
```

1. **Strengths**
   - Only successful `flock` admits creation; release hints affect waiting, never exclusion (`cmd/internal/couchcore/provision_lock_unix.go:56`).
   - Retry policy is pure and tested at both exact deadlines, including owner turnover and repeated release observations (`cmd/internal/couchcore/provision_retry_test.go:16`).
   - Held-marker failure closes the descriptor; release-marker failure still closes it and returns an error (`cmd/internal/couchcore/provision_lock_unix.go:26`, `:82`).
   - The native fixture exercises actual fork-to-exec inheritance. Existing tests retain intentional inheritance and parent-death coverage.
   - Atlas documentation and the exhaustive production-source inventory were updated.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Fresh provisioning, retry, and inventory checks passed: couchcore **16.608s**, artifactpath **5.131s**.
   - Inspected logs confirm natural baseline/fixed counts of **0/400 failures each**, and controlled-fork counts of **50/50 baseline failures versus 0/50 fixed failures**.
   - Independently reran the archived baseline regression binary; it failed with the expected busy error.
   - Inspected the final complete-suite log: no failure entries; couchcore passed in **378.533s**.
   - The checkout remains unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Existing lock inode and acquisition path remain the shared authority; no parallel locking mechanism.
   - **ARCH-PURE — pass:** Deterministic retry decisions are separated from filesystem, clock, sleep, and flock operations.
   - **ARCH-PURPOSE — pass:** The released-owner fork window is covered while live producers remain excluded. Natural and forced evidence are appropriately distinguished.
   - The plan’s revisions explicitly reconcile the moved pure function and per-invocation fixture compilation. No new command, flag, or configuration surface requires a README update.

7. **Plan revision recommendations:** None.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.474196666s
head: "ff8a74266d307053656a68326072e243a3ae06fb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
