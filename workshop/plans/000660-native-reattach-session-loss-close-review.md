# Boundary Review — pair#660 (whole-issue close)

| field | value |
|-------|-------|
| issue | 660 — Native direct Zellij session disappears after detach |
| repo | pair |
| issue file | workshop/issues/000660-native-reattach-session-loss.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3c76da091ea324423d755942121418905dd8c625..8022d0d29a753185dacd7e090348c3339ac08f93 |
| command | sdlc close --issue 660 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T06:41:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diagnostic stage preserves the original native assertions and promptly reports dead replacement clients. Deterministic tests pass, including under the race detector. One evidence-retention gap remains: failure reporting can discard the latest output and newly added actor-exit receipt.

1. **Strengths**

   - Dead-client detection uses the existing reaped-child lifecycle and preserves live/unknown outcomes.
   - Seven Python cases compare instrumented termination against baseline raw wait status.
   - Server-log reads are bounded and preserve missing-file errors.
   - No new session-discovery probes, retries, timeout increases or production lifecycle changes were introduced.

2. **Critical findings**

   None.

3. **Important findings**

```findings
findings:
  - id: new
    severity: Important
    family: bounded-latest-diagnostic-evidence
    title: |
      Failure reporting can discard the newly collected exit evidence
    detail: |
      terminal_native_test.go:153–164 still reports the first 4096 bytes of child.raw, parent.raw, wrapper.log and receipts.txt. Consequently, a live replacement client's timeout can omit its latest wire output, and an EXIT receipt appended after 4096 receipt bytes is discarded before scratch removal. I executed the actual Python instrumentation after 4096 padding bytes: the receipt was written, absent from the reported prefix, and present in a bounded tail. The remaining family instances are detach output at line 293, labelled output-tail but also using the prefix helper, and its unbounded failure output at line 295. ARCH-PURPOSE: sweep these diagnostic consumers to bounded latest evidence, reusing nativeChildWireTail/readNativeDiagnosticTail and a shared byte-tail helper where appropriate. Add a regression through the failure-report collection path with oversized streams ending in distinct exit markers; the existing helper-only tests do not catch this wiring gap.
```

4. **Minor findings**

   None.

5. **Test coverage notes**

   Targeted diagnostic tests passed normally and with `-race`; the race run also covered existing evidence lifetime and diagnostic-bound tests. `git diff --check` passed. Original PID:nonce, counter and frame assertions remain intact. I did not repeat the granted native family or full suite; their recorded outcomes were not independently remeasured here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing child lifecycle and environment helpers are reused.
   - **ARCH-PURE — pass:** process/filesystem observations remain in small test-only boundary helpers.
   - **ARCH-PURPOSE — flag:** the finding above identifies consumers that still discard the latest evidence this stage promises.
   - Atlas documentation covers the diagnostic flow. No new user-facing command, flag or configuration surface requires a README update. The plan has no Core concepts table to cross-check.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the complete diagnostic-consumer sweep and the regression proving that oversized failure streams retain their final exit markers. Preserve the explicit statement that the discovery cause remains unresolved.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"edf784e1-3b2c-4333-832c-57d34877b0f9","repository":"github.com/xianxu/pair","issue_id":"000660","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a083832786ba84ec","reviewed_head":"8022d0d29a753185dacd7e090348c3339ac08f93","slot":"pair:3","requested_at":"2026-10-18T13:39:55.761878Z"},"phase":"finished","pid":53672,"started_at":"2026-10-18T13:39:55.961141Z","finished_at":"2026-10-18T13:41:55.452858Z","session_id":"01a14f3d-8d55-7763-a1af-26c62688b559","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.800843083s
head: "8022d0d29a753185dacd7e090348c3339ac08f93"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T07:17:24-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 660 — Native direct Zellij session disappears after detach |
| repo | pair |
| issue file | workshop/issues/000660-native-reattach-session-loss.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3c76da091ea324423d755942121418905dd8c625..b575589513eb20a7eec304f2ef803c55a209a069 |
| command | sdlc close --issue 660 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T07:17:24-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The diagnostic stage matches the revised scope and preserves the original native identity, counter, frame and deadline assertions. BR-1 is addressed across the diagnostic-consumer family, with a regression that fails when prefix reporting is restored. No new blocking findings.

Done-when: current — it covers diagnostic collection, deterministic regressions, native verification and the unresolved-cause handoff described by the revised Spec.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      terminal_native_test.go:154 calls the five-stream failure collector; detach reporting at line 287 also uses the shared bounded-tail rule. TestNativeFailureCollectionRetainsLatestExitMarkers passes at head. A temporary Go overlay restoring prefix collection made it fail for child.raw, parent.raw, detach.output, wrapper.log and receipts.txt. Missing-file errors remain visible, and the unbounded detach failure output is removed.
```

1. **Strengths**

   - Failure reporting collects bounded latest evidence before cleanup.
   - Dead-client reporting reuses the existing reaped-child lifecycle and preserves live and ready outcomes.
   - Seven Python termination cases verify receipts against baseline raw wait status.
   - Original native assertions remain intact; no new discovery probes, retries or timeout increases appear.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**

   Targeted diagnostic and evidence tests passed normally (1.190s) and with `-race` (3.565s). The prefix mutation failed for all five streams. `git diff --check` passed; the checkout remains unchanged. Native conformance and the full suite were not rerun during this review; their recorded results were inspected in the tracker.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared byte-tail, file-tail and collection helpers cover the diagnostic consumers.
   - **ARCH-PURE — pass:** filesystem/process observations remain in small test-only boundary helpers; byte-tail logic is pure.
   - **ARCH-PURPOSE — pass:** the complete BR-1 family is swept, and the diagnostic landing does not claim a causal fix.
   - Atlas documents the flow. No new user-facing surface requires a README change. No Core concepts table is present.

7. **Plan revision recommendations:** None; the existing BR-1 revision describes the delivered sweep and regression.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5350b636-e048-4326-9a66-6c2245298ffc","repository":"github.com/xianxu/pair","issue_id":"000660","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f29968a672b91788","reviewed_head":"b575589513eb20a7eec304f2ef803c55a209a069","slot":"pair:3","requested_at":"2026-10-18T14:16:07.450701Z"},"phase":"finished","pid":86830,"started_at":"2026-10-18T14:16:07.831468Z","finished_at":"2026-10-18T14:17:23.554237Z","session_id":"01a14f5e-b14d-7153-aee4-52657bf51471","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.445491584s
head: "b575589513eb20a7eec304f2ef803c55a209a069"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
