# Boundary Review — pair#683 (whole-issue close)

| field | value |
|-------|-------|
| issue | 683 — Persist every lifecycle failure reason |
| repo | pair |
| issue file | workshop/issues/000683-park-finalization-diagnostics.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5afc269d3add99de21b5f894d8436ac6da15c796..3b9370348abba854cd5be0fa7d5c676be7ef3db7 |
| command | sdlc close --issue 683 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T03:24:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The approved failure-persistence scope is substantially implemented, with strong format, retention, and projection coverage. Shipping is blocked by a reproduced recovery regression, gaps in terminal recording coverage, and a failing required source guard. Park finalization remains explicitly assigned to #695/#684.

```findings
findings:
  - id: new
    severity: Critical
    family: diagnostics-preserve-recovery-contracts
    title: |
      Diagnostic target binding prevents recovery of unreadable records
    detail: |
      cmd/internal/couchcore/lifecyclefailure_record.go:44 requires GetThread to decode successfully before reboot can execute. Reboot intentionally supports archiving unreadable records; TestAnUnreadableRecordCanBeRebootedThroughTheRuntime and TestARefusalsNamedCommandsActuallyWork both reproducibly fail with "exact thread binding unavailable ... invalid JSON value". Preserve that recovery path using verified exact-address evidence without requiring a decoded slot path, while retaining guards against replacement targets. ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: terminal-owner-coverage
    title: |
      Admitted queue operations can silently run without terminal recording
    detail: |
      cmd/internal/couchcmd/lifecyclefailure_queue.go:35-43 leaves attempt nil when cached target lookup fails, but still enqueues the operation. Preparation failures then remain transient; successful dispatch can finish its own attempt before adoption/readiness, whose later failures are unrecorded. An enumerable sibling is hub_move.go:95-105: its "hub-move" key is rejected by the parser, and its payload-only dispatcher call supplies no fallback target. Pass verified diagnostic targets independently of scheduling keys and establish terminal ownership across every admitted producer, with missing-cache and post-dispatch failure regressions. ARCH-PURPOSE, ARCH-ORDER.
  - id: new
    severity: Important
    family: single-terminal-publication
    title: |
      Standalone switch failures allocate and finish multiple attempts
    detail: |
      cmd/internal/couchcore/launchfailure_capture.go:155-164 calls captureLaunchFailure when no launchFailureError exists, which already creates and finishes a standalone attempt at lines 139-145, then calls finishLaunchFailureCapture again. Existing standalone launch failures likewise publish before final switch enrichment and publish again afterward. Keep one enclosing attempt through capture and enrichment, then finish once; strengthen TestLaunchFailureCaptureSwitchFinalError to assert exactly one publication. ARCH-ORDER.
  - id: new
    severity: Important
    family: required-source-guard-compliance
    title: |
      The artifact source guard fails on three newly introduced symbols
    detail: |
      go test ./cmd/internal/artifactpath -count=1 fails TestNoProductionSymbolIsReferencedOnlyByTests for NewLifecycleFailureStore and NewFakeLifecycleFailureStore at cmd/internal/couchcore/lifecyclefailure.go:163-166, and ActiveSlots at operationattempt_record.go:103. Remove unnecessary exports, use the intended production surface, or document legitimate test-observation exceptions in the existing allowlist, then rerun the guard.
```

1. **Strengths**

   - The codec preserves nested opaque JSON values through real rewrites while rejecting duplicate keys and malformed known fields.
   - Retention tests cover the 65th maximum-size entry and supersession within the same byte budget.
   - Admission-order tests cover late failures, duplicate completion, and newer success without a store write.
   - README and atlas document the new diagnostics surface and its limits.

2. **Critical findings**

   - Restore unreadable-record reboot behavior without weakening exact-target safety.
   - Close the terminal-owner coverage gaps across cached-target misses and hub moves.

3. **Important findings**

   - Give standalone switch capture one terminal publication.
   - Resolve the three source-guard failures.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused lifecycle/launch/receipt/recovery/adoption/readiness tests passed across all four packages after using `TMPDIR=/tmp` to avoid macOS socket-path limits.
   - Core `OperationAttempt|LifecycleFailure` race tests passed.
   - Both unreadable-record recovery tests failed again in an isolated rerun.
   - Artifact guards failed as detailed above.
   - Broader message and console suites passed; command tests exposed the recovery failures. The optional broad core run was interrupted after 355 seconds, so full-core success is unverified.
   - Existing standalone switch coverage checks that records exist, rather than asserting one publication.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared history, codec, and projection helpers.
   - **ARCH-PURE — pass:** deterministic transitions and projections remain separated from storage and queue adapters.
   - **ARCH-PURPOSE — flag:** recovery regresses, and admitted failures still escape persistence.
   - **ARCH-MOCK — pass:** stateful fakes and portable real-store/private-service tests.
   - **ARCH-CONSTRAINTS — pass:** bounded history and metadata caching have relevant coverage.
   - **ARCH-SECURE — pass:** strict parsing, file validation, and holder-kind filtering remain intact.
   - **ARCH-ORDER — flag:** terminal ownership is missing on some paths and duplicated on standalone switch failures.
   - **ARCH-FUNERAL — pass:** bounded durable history and transient coordinator cleanup.
   - **ARCH-EVOLVE — pass:** additive fields and nested unknown-value preservation are tested.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining:

   - Exact-address diagnostic handling for unreadable-record recovery.
   - An exhaustive queue-producer/target-binding table, including missing-cache and hub-move paths.
   - One terminal owner for standalone switch capture and enrichment.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e70bee73-e1f3-4b8d-9f61-4ca79ec49278","repository":"github.com/xianxu/pair","issue_id":"000683","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fd7443b0a18d13be","reviewed_head":"3b9370348abba854cd5be0fa7d5c676be7ef3db7","slot":"pair:5","requested_at":"2026-10-19T10:16:09.387885Z"},"phase":"finished","pid":27646,"started_at":"2026-10-19T10:16:09.501685Z","finished_at":"2026-10-19T10:24:36.922835Z","session_id":"01a153a9-5951-7b93-9aa0-36212748b2cf","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.803072291s
head: "3b9370348abba854cd5be0fa7d5c676be7ef3db7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
