# Boundary Review — pair#579 (whole-issue close)

| field | value |
|-------|-------|
| issue | 579 — pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation |
| repo | pair |
| issue file | workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6dc6a2c36b1dc033c35def4f621aef33f3ac7e6b..bd641b5d9b551a6baee522fb955c14ce86efb27a |
| command | sdlc close --issue 579 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T18:01:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the governing scope revisions: buffered diagnostics, compact append reservations, ordered capture flushing, shared generation limits, and preservation behind the backup fence. No new blocking findings. The explicitly post-landing live IO measurement remains outstanding.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      EOF drain and controlled late-resize regressions pass. Replacing finishCapture with flushCapture in a temporary overlay makes both regressions fail, including 50 stranded sidecar bytes.
  - id: BR-2
    disposition: addressed
    note: |
      The real rename/sync-failure/retry regression passes. Removing unchanged-entry sync reconfirmation makes it fail because the retry incorrectly returns nil.
```

1. **Strengths**

   - The real-writer invariant measured **1.0199× logical amplification**, with three writes, one checkpoint rename, and zero hot syncs.
   - Append recovery preserves uncertain partial bytes in marked quarantines instead of adopting or truncating them.
   - Capture shutdown closes admission before its flush owner ends, including deliberately delayed resize completions.
   - README and atlas explain crash exposure, compatibility, and intentionally unbounded retained history.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Eight package runs passed: diagnosticlog, wrapcmd, adapt, artifactpath, crashreport, gcruntime, couchcore, and diagnosticcmd. Focused race tests passed for batching, capture, shutdown, and registry retries. Both prior fixes were independently verified with failing mutations in temporary overlays; repository source remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared buffering, generation evidence comparison, and retention authority.
   - **ARCH-PURE — pass:** batch transitions and classification logic have direct tests without IO.
   - **ARCH-PURPOSE — pass:** delivers the revised scope; raw rotation is explicitly assigned to #582.
   - **ARCH-MOCK — pass:** stateful sink/process doubles share production seams; storage fixtures use portable folders.
   - **ARCH-CONSTRAINTS — pass:** admission bounds include in-flight bytes; counted amplification passes.
   - **ARCH-SECURE — pass:** bounded strict decoding, identity checks, and conservative uncertainty handling.
   - **ARCH-ORDER — pass:** controlled partial-progress, close, retry, and late-completion tests exercise production paths.
   - **ARCH-FUNERAL — pass:** reservation reuse/retirement and worker teardown are explicit; retained-history growth is expressly accepted behind the backup fence.

7. **Plan revision recommendations:** None. Existing revisions reconcile the implementation names, restored capture batching, quarantine behavior, and backup fence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f08979a2-aaab-4357-8813-3e893092bce3","repository":"github.com/xianxu/pair","issue_id":"000579","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7ba4870511f3d0f1","reviewed_head":"bd641b5d9b551a6baee522fb955c14ce86efb27a","slot":"pair:3","requested_at":"2026-10-16T00:52:10.217112Z"},"phase":"finished","pid":39631,"started_at":"2026-10-16T00:52:10.417048Z","finished_at":"2026-10-16T01:01:20.117013Z","session_id":"01a14231-edcc-72f3-adb7-d6bad94a4f90","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.4579425s
head: "bd641b5d9b551a6baee522fb955c14ce86efb27a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
