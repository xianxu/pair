# Boundary Review — pair#579 (milestone M2)

| field | value |
|-------|-------|
| issue | 579 — pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation |
| repo | pair |
| issue file | workshop/issues/000579-pair-wrap-writes-200-more-than-it-appends-4-mb-s-per-slot-per-event-sync-on-scrollback-wrap-events-logs-also-no-rotation.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | fe07bef1e2d604b11ce14a3ddcab1dd2aa1deea0..2723d3189cc221fc748a2ed3a014126b70737482 |
| command | sdlc milestone-close --issue 579 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:51:18-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches the governing Spec and Plan revisions: shared generation-size limits, diagnostic preservation pending backup, and capture shutdown that drains buffers and closes admission. No blocking defects found. README and atlas describe the revised policy, including intentionally unbounded retained history. The post-landing live measurement remains explicitly outstanding.

```findings
{}
```

1. **Strengths**

   - Persisted limits prevent conflicting handles and CLI writers from widening the cap (`cmd/internal/diagnosticlog/policy.go:33`, `writer.go:315`).
   - The backup fence covers managed/legacy collection, count pruning, crash sweeping, and pending deletion replay (`collect.go:183`).
   - Pump exit atomically closes capture admission and drains accepted bytes; the regression controls a resize completing after EOF (`wrap.go:3077`, `diagnostic_buffer_test.go:220`).
   - Compatibility coverage exercises actual park, witnessed capture identities, retained-reader leases, and previous-capture replay during diagnostic rotation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   All nine selected packages passed. Focused race checks passed for diagnosticlog, wrapcmd, diagnosticcmd, and couchcore.

   Temporary source overlays confirmed meaningful regression failures:
   - Disabling the backup fence caused generation loss, expired-history deletion, and pending-delete replay.
   - Replacing capture Close with Flush left 50 sidecar bytes buffered after pump exit.

   Overlays were removed; the checkout is clean. I did not rerun the full SDLC suite, build, or live physical-IO measurement.

6. **Architecture**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass — shared policy and existing rotation/recovery owners are reused. |
   | ARCH-PURE | Pass — limit resolution and generation selection have direct IO-free tests. |
   | ARCH-PURPOSE | Pass — implementation follows the revised backup-preservation contract across consumers. |
   | ARCH-MOCK | Pass — integration fixtures use portable storage and injected process evidence. |
   | ARCH-CONSTRAINTS | Pass — shared byte caps hold; rotation proceeds beyond former inventory limits. |
   | ARCH-SECURE | Pass — identity checks, strict schema handling, and uncertain-evidence refusal remain intact. |
   | ARCH-ORDER | Pass — controlled late-resize and deletion-interruption tests exercise production transitions. |
   | ARCH-FUNERAL | Pass — retained history’s unbounded lifetime and backup dependency are explicitly documented and approved. |

7. **Plan revision recommendations:** None. Existing revisions reconcile the implementation filenames, scope changes, and backup fence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"38e1b21a-6591-4991-8e38-6f6c04ae5ae1","repository":"github.com/xianxu/pair","issue_id":"000579","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9bab4852b0f25499","reviewed_head":"2723d3189cc221fc748a2ed3a014126b70737482","slot":"pair:3","requested_at":"2026-10-16T00:41:23.615513Z"},"phase":"finished","pid":10037,"started_at":"2026-10-16T00:41:23.910932Z","finished_at":"2026-10-16T00:51:16.863979Z","session_id":"01a14228-11bb-7431-9dbf-632315ce22fc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.337179292s
head: "2723d3189cc221fc748a2ed3a014126b70737482"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
