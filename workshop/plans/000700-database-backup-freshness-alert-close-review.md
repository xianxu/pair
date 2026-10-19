# Boundary Review — pair#700 (whole-issue close)

| field | value |
|-------|-------|
| issue | 700 — Warn when a database backup is older than its freshness threshold |
| repo | pair |
| issue file | workshop/issues/000700-database-backup-freshness-alert.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8b7e155fd77906abd9823491ff9b510a798cfab8..bc0803dc3f63f88c9fdc939393abb75bd8733094 |
| command | sdlc close --issue 700 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T06:49:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation largely follows the approved plan: shared freshness policy, bounded reports, daily admission, and existing notification surfaces. One reproduced timezone bug blocks shipping: a successful snapshot on a non-UTC host can prevent the entire report—including stale database warnings—from reaching Couch.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: runtime-evidence-time-normalization
    title: |
      Normalize observation timestamps before enforcing UTC report validation
    detail: |
      cmd/internal/historyarchive/continuous_report.go:117-119 rejects non-UTC CheckedAt values, but continuous.go:188 assigns p.now(), whose production default returns local time.Now(). On a non-UTC host, any validated snapshot therefore makes EncodeContinuousReport reject the entire report; archivecmd/continuous.go:122-123 emits no JSON, losing stale/never-verified rows alongside it. A temporary overlay exercising the existing fake-backed pass at UTC−7 failed for both 3 and 6 databases with "invalid UTC evidence time"; the same regression passed with row.CheckedAt = p.now().UTC(). Normalize producer timestamps and retain a regression covering non-UTC pass results through encoding, including mixed healthy/stale rows. [ARCH-PURPOSE, ARCH-CONSTRAINTS]
```

1. **Strengths**

   - Freshness derives from catalog evidence without another snapshot or catalog scan.
   - Daily admission explicitly handles duplicates, rollback, capacity, and reservation before notification IO.
   - Subprocess tests exercise partial failures, oversized output, cancellation, and inherited descriptors through the production launch seam.
   - README and atlas document the threshold, quiet-database exception, routing, and warning lifetime.

2. **Critical findings**

   - [Report validation rejects production local timestamps](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ddcdc8e52bf2d941/cmd/internal/historyarchive/continuous_report.go:117). Normalize `CheckedAt` to UTC at its producer and add the regression described above.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full `historyarchive`, `archivecmd`, `couchcmd`, and `artifactpath` package tests passed. Couch tests required `TMPDIR=/tmp` to satisfy macOS Unix-socket path limits.

   Existing fixtures consistently use UTC and do not carry actual pass results through encoding, leaving the reproduced gap. Temporary overlays demonstrated failure on pinned code and success with timestamp normalization. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared freshness policy.
   - **ARCH-PURE — pass:** pure policy/codec/admission separated from IO.
   - **ARCH-PURPOSE — flag:** report rejection prevents the promised database warnings.
   - **ARCH-MOCK — pass:** stateful subprocess fixtures exercise the production seam.
   - **ARCH-CONSTRAINTS — flag:** normal non-UTC environments break report delivery.
   - **ARCH-SECURE — pass:** malformed reports fail visibly; warnings are bounded and sanitized.
   - **ARCH-ORDER — pass:** admission transitions cover rollover, rollback, duplicates, and uncertain delivery.
   - **ARCH-FUNERAL — pass:** reservations are capped/pruned; logs and attention reuse existing lifetimes.
   - **ARCH-EVOLVE — pass:** persisted formats remain unchanged; runtime unknown fields survive in raw logs.

   Core-concept locations and classifications match the implementation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring non-UTC producer-to-report qualification, including a pass containing both validated and stale database rows.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"67270aa8-11b6-4106-81c2-0ce4b677a628","repository":"github.com/xianxu/pair","issue_id":"000700","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ddcdc8e52bf2d941","reviewed_head":"bc0803dc3f63f88c9fdc939393abb75bd8733094","slot":"pair:7","requested_at":"2026-10-19T13:42:38.76386Z"},"phase":"finished","pid":50396,"started_at":"2026-10-19T13:42:38.858368Z","finished_at":"2026-10-19T13:48:42.831369Z","session_id":"01a15466-64fc-7293-9992-a3c1b3d446c6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.600699959s
head: "bc0803dc3f63f88c9fdc939393abb75bd8733094"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
