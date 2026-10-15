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
