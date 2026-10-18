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
