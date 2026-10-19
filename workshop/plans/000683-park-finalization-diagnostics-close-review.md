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

---

## Re-review — 2026-10-19T03:47:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 683 — Persist every lifecycle failure reason |
| repo | pair |
| issue file | workshop/issues/000683-park-finalization-diagnostics.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5afc269d3add99de21b5f894d8436ac6da15c796..6fcefcb8258797ae53642ae6c844f04b3e5a373d |
| command | sdlc close --issue 683 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T03:47:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four prior findings are addressed, with regression tests that fail when the old behavioral paths are restored. The completion recorder, bounded preserving codec, and recovery behavior are sound under the focused checks. One Important display inconsistency remains: inspecting a resolved slot can omit a failure that its switcher row displays.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      BeginOperationAttempt no longer requires GetThread. Both unreadable-record recovery regressions pass; restoring the old binding prerequisite makes both fail with invalid JSON value.
  - id: BR-2
    disposition: addressed
    note: |
      Terminal ownership is independent of cache hits, and hub moves retain submitted targets. Missing-cache preparation/adoption/readiness and hub-move regressions pass; restoring cache-dependent ownership makes all fail.
  - id: BR-3
    disposition: addressed
    note: |
      Standalone switches retain one enclosing attempt through enrichment. Both switch publication regressions pass; restoring the old capture/switch paths produces two publications and fails them.
  - id: BR-4
    disposition: addressed
    note: |
      The three unused symbols are removed, and the full artifactpath package passes, including TestNoProductionSymbolIsReferencedOnlyByTests.
findings:
  - id: new
    severity: Important
    kind: defect
    family: shared-diagnostic-visibility
    title: |
      Show omits reference-only failures when inspecting the resolved slot
    detail: |
      cmd/internal/couchcore/launchfailure_show.go:57 matches target-only evidence only against the literal show reference, although lifecyclefailureRowTargets matches equivalent slot references. A disposable regression confirms that Target="pair:8" appears in the row and show pair:8, but disappears when show uses the same resolved slot's path or exact thread tag. ARCH-DRY/ARCH-PURPOSE: share target selection and visibility across both views; cover reference-only evidence through slot-name, primary-root reference, path, and exact-address inspection.
```

1. **Strengths**

   - Diagnostics no longer add a decoding prerequisite to recovery.
   - Once-only terminal ownership covers preparation through adoption/readiness, including missing-cache operations.
   - Codec tests verify nested opaque-byte preservation, strict known fields, atomic refusal, and bounded retention.
   - README and atlas document the new surface and historical-evidence semantics.

2. **Critical findings:** None.

3. **Important findings:** The row/show mismatch above. Consolidate matching before filtering and selecting the latest failure; add cross-reference parity tests.

4. **Minor findings:** None.

5. **Test coverage notes**

   - The plan’s focused four-package suite passed with `TMPDIR=/tmp`.
   - Focused race checks passed for core, command, console, and message packages.
   - Full artifactpath and testpair guards passed.
   - Controlled restorations reproduced BR-1, BR-2, and BR-3.
   - The new display regression failed on pinned head.
   - The optional unrestricted run was stopped; its default temporary path caused socket and registration failures. Focused reruns with a short temporary path passed. Full-suite success is not claimed.
   - Disposable files were removed; the tracked checkout remains unchanged.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | **Flag:** separate row/show selection policies diverge. |
   | ARCH-PURE | Pass: pure codec/projection logic remains separate from storage and completion adapters. |
   | ARCH-PURPOSE | **Flag:** shared diagnostic visibility is incomplete. |
   | ARCH-MOCK | Pass: existing stateful fakes and portable service/store fixtures are reused. |
   | ARCH-CONSTRAINTS | Pass: writer bounds, cached reads, and bounded admission are exercised. |
   | ARCH-SECURE | Pass: strict decoding, safe-path checks, and holder-kind allowlisting are covered. |
   | ARCH-ORDER | Pass: terminal ownership and duplicate publication have regression evidence. |
   | ARCH-FUNERAL | Pass: bounded history and existing staging cleanup govern residue. |
   | ARCH-EVOLVE | Pass: nested unknown members survive real rewrites byte-equal. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the shared row/show matching correction. Label the replacement concept map explicitly PURE/INTEGRATION and clearly mark the original coordinator tables as superseded.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d7cb8070-9207-48cb-9c5c-6f742bb4a3fc","repository":"github.com/xianxu/pair","issue_id":"000683","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/37797697dc98bac4","reviewed_head":"6fcefcb8258797ae53642ae6c844f04b3e5a373d","slot":"pair:5","requested_at":"2026-10-19T10:41:24.736133Z"},"phase":"finished","pid":41913,"started_at":"2026-10-19T10:41:24.832787Z","finished_at":"2026-10-19T10:46:42.908626Z","session_id":"01a153c0-7856-7501-8ca8-016cac21fa39","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.50210625s
head: "6fcefcb8258797ae53642ae6c844f04b3e5a373d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-19T03:57:52-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 683 — Persist every lifecycle failure reason |
| repo | pair |
| issue file | workshop/issues/000683-park-finalization-diagnostics.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5afc269d3add99de21b5f894d8436ac6da15c796..ba4820caa5a8270bcbea67bb4767121a9748188b |
| command | sdlc close --issue 683 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T03:57:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-5 is addressed: row and show now share resolved-slot target selection, and removing the fix makes its regression tests fail. The implementation follows the approved completion-log design, preserves recovery behavior and updates README/atlas. One Important responsiveness issue remains: diagnostic publication can block the console event loop.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Diagnostic binding is absent; both unreadable-record recovery regressions pass at the pinned head.
  - id: BR-2
    disposition: addressed
    note: |
      Missing-cache preparation/adoption/readiness and hub-move tests pass; attempts retain dispatched evidence independently of cache hits.
  - id: BR-3
    disposition: addressed
    note: |
      Standalone switches retain one enclosing attempt; launch-capture tests asserting single publication pass.
  - id: BR-4
    disposition: addressed
    note: |
      The full artifactpath package passes, including source and symbol guards.
  - id: BR-5
    disposition: addressed
    note: |
      Resolved-target selection uses lifecycleFailureRowTargets. Both target-only regression tests pass at head and fail when launchfailure_show.go is restored from 6fcefcb82 in a disposable checkout.
findings:
  - id: new
    severity: Important
    kind: defect
    family: optional-io-off-console-event-loop
    title: |
      Terminal diagnostic publication blocks console input and rendering
    detail: |
      cmd/internal/couchtty/console_lifecycle_failure.go:41 calls Finish synchronously from finishOperation, invoked by the console event loop at console.go:1088. Finish performs Save, including up to ten seconds waiting for the diagnostic lock and filesystem IO afterward; a controlled blocked-store probe kept finishOperation blocked throughout a 100ms interval. Remote completions without a readiness wait also reach synchronous publication through couchcmd/lifecyclefailure_queue.go:49. ARCH-CONSTRAINTS: perform terminal publication after adoption on bounded, joined worker paths, then deliver the finalized outcome to the event loop; cover menu, reattach and remote failures with a stalled-store responsiveness regression.
```

1. **Strengths**

   - Shared display target enumeration and visibility resolve BR-5 across slot names, primary-root references, paths, tags and empty slots (`launchfailure_show.go:36`, `lifecyclefailure_projection.go:142`).
   - Real-store tests verify byte-preserving unknown members through rewrites and enrichment (`lifecyclefailure_codec_test.go:14`).
   - Success performs no diagnostic IO; typed failures and receipt correlation survive terminal recording.
   - README and atlas document the new surface, retention limits and record-derived visibility.

2. **Critical findings**

   None.

3. **Important findings**

   The console-blocking publication above needs correction. Keep once-only terminal ownership, but finish storage work outside the event loop. The regression should prove input and stop events remain processable while publication is stalled.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused lifecycle, launch, attempt, recovery and receipt tests passed.
   - Focused race tests passed for core, command and console packages.
   - Full artifactpath, testpair and termcmd guards passed.
   - Private-service coverage passed with `TMPDIR=/tmp`; the default temporary path exceeded the Unix socket path limit.
   - BR-5 mutation failed as expected. The new responsiveness probe also failed as expected. Disposable checkouts were removed; the reviewed checkout remains unchanged.
   - The broad unfiltered package run was interrupted; no full-suite pass is claimed.

6. **Architectural notes**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass: shared target enumeration, recency and visibility helpers. |
   | ARCH-PURE | Pass: deterministic codec/history/projection functions remain separate from IO adapters. |
   | ARCH-PURPOSE | Pass: approved persistence scope covers failures before and after dispatch; BR-5 is corrected. |
   | ARCH-MOCK | Pass: stateful storage fakes and portable real-service coverage share production seams. |
   | ARCH-CONSTRAINTS | **Flag:** optional publication blocks the console event loop. |
   | ARCH-SECURE | Pass: strict decoding, safe file handling and holder-kind allowlisting. |
   | ARCH-ORDER | Pass: action-local once-only completion ownership retains adoption/readiness outcomes. |
   | ARCH-FUNERAL | Pass: writer-enforced count/byte bounds and existing staging cleanup. |
   | ARCH-EVOLVE | Pass: additive decoding preserves nested unknown bytes through persisted rewrites. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that diagnostic publication runs outside the console event loop, names the bounded worker owner for each completion path, and requires stalled-store responsiveness coverage. Retain the approved separation from #695 park finalization.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9653df2e-c9a6-42c0-8d45-b037b3fc4261","repository":"github.com/xianxu/pair","issue_id":"000683","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ac26d412ace7f42e","reviewed_head":"ba4820caa5a8270bcbea67bb4767121a9748188b","slot":"pair:5","requested_at":"2026-10-19T10:53:02.846002Z"},"phase":"finished","pid":46516,"started_at":"2026-10-19T10:53:02.956433Z","finished_at":"2026-10-19T10:57:51.506438Z","session_id":"01a153cb-1f6d-7ca0-9fec-a43db6b0ff53","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.644756541s
head: "ba4820caa5a8270bcbea67bb4767121a9748188b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
