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

---

## Re-review — 2026-10-19T06:55:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 700 — Warn when a database backup is older than its freshness threshold |
| repo | pair |
| issue file | workshop/issues/000700-database-backup-freshness-alert.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8b7e155fd77906abd9823491ff9b510a798cfab8..47fd390f227e85d08c6caa0008ef50f263e8d484 |
| command | sdlc close --issue 700 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T06:55:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the documented freshness policy, bounded report transport, daily warning admission, and operator delivery. BR-1 is verified addressed with a failing negative control. No new blocking findings were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      ContinuousPass.now normalizes injected and production clocks to UTC. The mixed-result and N/2N pass regressions exercise actual producer evidence through encoding and decoding with non-UTC clocks. Removing only normalization in a disposable pinned-head checkout made these tests fail with invalid UTC evidence time; pinned-head historyarchive tests pass. The disposable checkout was removed.
```

1. **Strengths**

   - Freshness policy is shared by CLI and Couch, preserves catalog timestamps, and exempts validated unchanged content (`cmd/internal/historyarchive/continuous_snapshot_model.go:50`).
   - Subprocess handling preserves valid partial reports independently of process failure and rejects overflow entirely (`cmd/internal/couchcmd/archive_schedule.go:86`).
   - Admission occurs before notification effects, with exact source keys, UTC rollover, rollback protection, and a bounded reservation set (`cmd/internal/couchcmd/archive_alert_model.go:29`).
   - README and atlas updates document the threshold, quiet-database exception, delivery ownership, and reservation lifetime.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed package tests for `historyarchive`, `archivecmd`, and `artifactpath`.
   - Passed the complete `couchcmd` package with `TMPDIR=/tmp`. The initial run encountered host temporary-path lengths exceeding the Unix socket limit; the short-root rerun passed.
   - BR-1’s negative control failed in the mixed-result regression and both counted-pass cases when normalization was removed.
   - `git diff --check` passed. The repository-wide recorded suite and natural desktop acceptance were not independently rerun during this review.

6. **Architectural notes**

   | Principle | Result |
   | --- | --- |
   | ARCH-DRY | Pass — one freshness predicate serves both consumers. |
   | ARCH-PURE | Pass — policy, codec, and admission tests run without IO; effects remain at integration seams. |
   | ARCH-PURPOSE | Pass — stale and never-verified rows reach warnings, including partial failed passes. |
   | ARCH-MOCK | Pass — stateful subprocess fixtures and injected notification outputs exercise production boundaries. |
   | ARCH-CONSTRAINTS | Pass — report capture and admission are bounded; N/2N checks cover work scaling. |
   | ARCH-SECURE | Pass — external reports receive structural and semantic validation; notification text is sanitized. |
   | ARCH-ORDER | Pass — reservation transitions are centralized and tested across duplicates, rollback, rollover, restart, and uncertain delivery. |
   | ARCH-FUNERAL | Pass — reservations expire on rollover or process exit; the existing latest-pass log is overwritten. |
   | ARCH-EVOLVE | Pass — persisted formats remain unchanged; additive runtime fields are accepted and original report bytes retained in logs. |

7. **Plan revision recommendations:** None. The existing BR-1 revision matches the correction and regression evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0cad523e-bc90-4ae0-84c4-f3c4288422e2","repository":"github.com/xianxu/pair","issue_id":"000700","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/49f729cc0d309bee","reviewed_head":"47fd390f227e85d08c6caa0008ef50f263e8d484","slot":"pair:7","requested_at":"2026-10-19T13:52:01.251488Z"},"phase":"finished","pid":68414,"started_at":"2026-10-19T13:52:01.390913Z","finished_at":"2026-10-19T13:55:30.716818Z","session_id":"01a1546e-fb0a-71d2-a15c-c0966d3b8061","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.643824042s
head: "47fd390f227e85d08c6caa0008ef50f263e8d484"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
