# Boundary Review — pair#579 (milestone M1)

| field | value |
|-------|-------|
| issue | 579 — pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation |
| repo | pair |
| issue file | workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 32a0818a32ffb6ae4764aa587a3193959a4ad295..b13ee4118af26fb38f981c319255d7f7bc4e0a13 |
| command | sdlc milestone-close --issue 579 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T16:07:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 substantially delivers compact diagnostic appends, bounded batching, registry deduplication, and ordered capture buffering. The focused package tests and selected race tests pass. One reproduced lifecycle gap blocks SHIP: capture flushing stops before all accepted bytes are drained.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: drain-before-owner-stop
    title: |
      Drain capture before stopping its flush owner
    detail: |
      cmd/internal/wrapcmd/wrap.go:3144 stops the capture ticker when masterPump returns, but capture.Close runs only after run returns, beyond cmd.Wait at line 3041. A child that closes its PTY and keeps running leaves accepted capture bytes buffered indefinitely despite healthy IO, violating the declared 250ms envelope (ARCH-CONSTRAINTS, ARCH-ORDER). A temporary overlay regression reproduced pump completion with offset=12 and an empty raw sink. Flush capture on every pump exit before waiting for the child, report failures, and retain a deterministic EOF-before-tick regression.
```

1. **Strengths**

   - Compact reservations eliminate payload duplication while retaining legacy-intent recovery and exact full-tail verification.
   - Batch tests exercise FIFO progress, partial writes, admission bounds, and Close during controlled in-flight IO.
   - Capture overflow tests verify raw-before-sidecar publication. README and atlas updates explain buffering and durability limits.

2. **Critical findings**

   - **Drain before stopping the capture owner:** [wrap.go:3144](/Users/xianxu/Library/Caches/sdlc/review-checkouts/252bef2d378bc31d/cmd/internal/wrapcmd/wrap.go:3144). Drain on pump exit, before the subsequent child wait. PTY closure does not establish child termination.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed all tests in `diagnosticlog`, `wrapcmd`, `adapt`, and `artifactpath`.
   - Selected batching, recovery, quarantine, registry, and capture tests passed under `-race`.
   - The temporary EOF regression failed deterministically: accepted offset `12`, raw sink empty after pump return. Its overlay was removed; the checkout remains unchanged.
   - Full repository suite, build, and live physical measurements were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** quarantine reuses durable rotation; diagnostic emitters share one buffered owner.
   - **ARCH-PURE — pass:** batch transitions and reservation classification have direct tests without IO.
   - **ARCH-PURPOSE — pass:** implementation follows revised M1 scope; shared retention limits remain M2 work.
   - **ARCH-MOCK — pass:** stateful sinks, manual ticks, process inspection doubles, and temporary storage exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** pump exit suspends the capture flush envelope.
   - **ARCH-SECURE — pass:** bounded strict parsing and identity/hash checks preserve uncertain evidence.
   - **ARCH-ORDER — flag:** the flush owner ends while accepted effects remain pending.
   - **ARCH-FUNERAL — pass:** reservation retirement and quarantined-generation expiry have implementation and coverage.

7. **Plan revision recommendations**

   None required. The correction should enforce the existing capture lifecycle and flush contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c86498bf-dff7-434c-820d-d058f439327e","repository":"github.com/xianxu/pair","issue_id":"000579","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/252bef2d378bc31d","reviewed_head":"b13ee4118af26fb38f981c319255d7f7bc4e0a13","slot":"pair:3","requested_at":"2026-10-15T23:04:52.074265Z"},"phase":"finished","pid":83077,"started_at":"2026-10-15T23:04:52.230515Z","finished_at":"2026-10-15T23:07:46.430936Z","session_id":"01a141cf-b25b-7fa3-8966-1276c481fff8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.679020875s
head: "b13ee4118af26fb38f981c319255d7f7bc4e0a13"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T16:13:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 579 — pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation |
| repo | pair |
| issue file | workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 32a0818a32ffb6ae4764aa587a3193959a4ad295..aa22ef251aaa6d0510e4e9a9c480cb506dd005b2 |
| command | sdlc milestone-close --issue 579 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T16:13:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-1 is addressed with verified red/green evidence. The M1 implementation preserves capture ordering, reduces diagnostic amplification, and passes focused package and race checks. One Important gap remains: the promised regression for registry durability reconfirmation is missing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      wrap.go:3077 flushes capture on every pump exit. TestMasterPumpDrainsCaptureBeforeReturningOnEOF passes at head; removing the defer in a temporary overlay makes both drainage and error-reporting cases fail.
findings:
  - id: new
    severity: Important
    kind: defect
    family: durability-reconfirmation
    title: |
      Add the promised registry durability-retry regression
    detail: |
      cmd/internal/diagnosticlog/registry_test.go:247 checks unchanged declarations and inode preservation, but never exercises publication followed by failed directory sync and retry. The M1 plan explicitly requires this case at workshop/plans/000579-wrapper-log-io-plan.md:120. Replacing registry.go:62's durability reconfirmation with return nil still passes every TestRegistry test. Add an injected sync-failure sequence proving an equal declaration reconfirms durability, propagates sync failure, and avoids content replacement.
```

1. **Strengths**

   - Capture EOF drainage and error reporting have meaningful regression coverage.
   - The real-writer amplification test preserves all 1,000 records in order and measured **1.0188×**, three writes, one checkpoint rename, and zero hot syncs.
   - Compact reservations validate persisted input strictly; partial-quarantine tests preserve uncertain bytes and exercise interrupted rotation recovery.
   - README and atlas updates explain buffering, crash exposure, and retained synchronization boundaries.

2. **Critical findings**

   None.

3. **Important findings**

   The registry durability regression above. The production reconfirmation at `cmd/internal/diagnosticlog/registry.go:62` appears correct; the missing test leaves a specifically identified failure mode unprotected.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed all four focused packages: `diagnosticlog`, `wrapcmd`, `adapt`, and `artifactpath`. Selected buffering, recovery, quarantine, capture, and wrapper-wiring tests also passed under `-race`. Temporary mutation overlays were removed; the checkout remains unchanged.

   Inspected the scratch measurement harness and result artifacts supporting the reported **61.0× → 1.32×** physical-write comparison. Build and full recorded-suite execution were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** quarantine reuses rotation recovery; diagnostic emitters share one buffered owner.
   - **ARCH-PURE — pass:** batch transitions and reservation classifications have direct tests without IO.
   - **ARCH-PURPOSE — pass for M1:** batching and compact append metadata are delivered; shared retention limits remain M2.
   - **ARCH-MOCK — pass:** stateful sinks, manual clocks, process-inspection doubles, and portable filesystem fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** admission bounds include in-flight bytes; counted tests verify amplification.
   - **ARCH-SECURE — pass:** bounded strict decoding and identity checks preserve refusal of changed evidence.
   - **ARCH-ORDER — flag:** registry durability-retry coverage is missing, as detailed above. Capture-owner shutdown is verified.
   - **ARCH-FUNERAL — pass for M1:** reservation storage is bounded and retired; quarantines use existing generation collection. Count retention remains M2.

7. **Plan revision recommendations**

   None. Implement the existing M1 durability-retry test requirement; no scope revision is needed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"02b08413-075b-4baf-8715-7ad5e37fd6fd","repository":"github.com/xianxu/pair","issue_id":"000579","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3b6b0ac70846a5ed","reviewed_head":"aa22ef251aaa6d0510e4e9a9c480cb506dd005b2","slot":"pair:3","requested_at":"2026-10-15T23:10:04.444581Z"},"phase":"finished","pid":86366,"started_at":"2026-10-15T23:10:04.629976Z","finished_at":"2026-10-15T23:12:41.627737Z","session_id":"01a141d4-7466-7cd2-ad0d-b9e233dae31a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.246468375s
head: "aa22ef251aaa6d0510e4e9a9c480cb506dd005b2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T16:18:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 579 — pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation |
| repo | pair |
| issue file | workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 32a0818a32ffb6ae4764aa587a3193959a4ad295..8a91523741f6fcefc1be498fdc3418cb74a38905 |
| command | sdlc milestone-close --issue 579 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T16:18:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised M1 scope. BR-2 now has meaningful regression coverage, verified by mutation testing. No new blocking findings emerged. Shared retention bounds remain M2 work; this verdict covers M1 only.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      registry_test.go:388 exercises publication followed by failed directory sync, failing identical retry, and successful reconfirmation while preserving bytes/inode/mtime. Replacing registry.go:68 with return nil makes the regression fail.
  - id: BR-1
    disposition: addressed
    note: |
      The EOF drain/error-reporting regression passes. Removing masterPump's deferred flush makes both cases fail, confirming the production fix remains reachable and necessary.
```

1. **Strengths**

   - Compact reservations preserve recovery evidence without duplicating payload.
   - The real-writer invariant measured **1.0187×** logical amplification, one checkpoint rename, and zero hot syncs.
   - Capture tests verify raw-before-sidecar ordering, unchanged witnessed bytes, sticky failures, and EOF drainage.
   - README and atlas document the new buffering behavior and crash limitations.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Full tests passed for `diagnosticlog`, `wrapcmd`, `adapt`, and `artifactpath`. Selected race tests passed. Both prior fixes failed their targeted tests when removed through temporary overlays. Repository source remained unchanged.

   Inspected the scratch measurement harness and saved results supporting the reported physical IO reduction. Build, full recorded suite, and live measurements were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** quarantine reuses durable rotation and collection.
   - **ARCH-PURE — pass:** batch transitions and reservation classification have direct pure tests.
   - **ARCH-PURPOSE — pass for M1:** implementation matches the governing scope revisions.
   - **ARCH-MOCK — pass:** stateful sinks and process inspection share production seams; storage tests use temporary folders.
   - **ARCH-CONSTRAINTS — pass:** admission and flight are bounded; deterministic counters exercise amplification.
   - **ARCH-SECURE — pass:** strict parsing, identity checks, and foreign-tail refusal preserve uncertainty.
   - **ARCH-ORDER — pass:** confirmed progress, concurrent admission, close, and EOF drainage are exercised.
   - **ARCH-FUNERAL — pass for M1:** reservations are bounded and retired; quarantines use existing expiry collection. Count bounds remain M2.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e5a38a5a-2ae3-458f-9505-59e2f4daac2d","repository":"github.com/xianxu/pair","issue_id":"000579","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a59b97b959cd8ced","reviewed_head":"8a91523741f6fcefc1be498fdc3418cb74a38905","slot":"pair:3","requested_at":"2026-10-15T23:16:33.651203Z"},"phase":"finished","pid":74691,"started_at":"2026-10-15T23:16:33.792007Z","finished_at":"2026-10-15T23:18:49.84123Z","session_id":"01a141da-6495-7962-89af-4ca4f1f66590","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.746207709s
head: "8a91523741f6fcefc1be498fdc3418cb74a38905"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
