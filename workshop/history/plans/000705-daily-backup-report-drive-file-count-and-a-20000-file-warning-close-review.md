# Boundary Review — pair#705 (whole-issue close)

| field | value |
|-------|-------|
| issue | 705 — Daily backup report: Drive file count and a 20000-file warning |
| repo | pair |
| issue file | workshop/issues/000705-daily-backup-report-drive-file-count-and-a-20000-file-warning.md |
| boundary | whole-issue close |
| milestone | — |
| window | 424e6c7c0c917ec4af21ffc2c802cbe74babd8d8..7a71608c5bb57a4a05ab2c5dc87089482be9eeb1 |
| command | sdlc close --issue 705 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T09:13:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes fulfill #705’s implementation requirements: both reports use existing listings, unknown counts remain distinct from zero, and warnings share the destination-specific daily admission path. No blocking defects found. The recorded full-suite and landing steps remain explicitly unchecked.

1. **Strengths**

   - `cmd/internal/historyarchive/drive.go:138` captures valid listing evidence immediately; a failed final listing preserves the labeled before observation.
   - `cmd/internal/historyarchive/drive_count_model.go:36` counts duplicate file entries while keeping directories in existing capacity accounting.
   - `cmd/internal/couchcmd/archive_schedule.go:160` shares warning delivery and reserves admission before effects, preserving suppression after uncertain delivery.
   - README and atlas document the threshold, observation semantics, suppression, and operator action.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed historyarchive, archivecmd, couchcmd, stateful FakeRclone tests, installed-rclone local-backend conformance, selected source/reference/dead-symbol/docs guards, and diff whitespace checks.

   The initial Couch run failed because temporary socket paths exceeded 103 bytes. Both targeted archive tests and the complete Couch package passed with `TMPDIR=/tmp`.

   Coverage exercises duplicate names, directories, exact remote-call counts, partial copies, cancellation, retained observations, invalid thresholds, malformed reports, and destination/day admission sequences. No prior findings required regression-fix verification.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared count/rendering model and warning delivery helper.
   - **ARCH-PURE — Pass:** model and codec tests run without IO; adapters own effects.
   - **ARCH-PURPOSE — Pass:** human report, typed report, and scheduler warning are connected.
   - **ARCH-MOCK — Pass:** production runner seam uses the stateful fake; local conformance passed.
   - **ARCH-CONSTRAINTS — Pass:** bounded listings remain authoritative; counted tests confirm no additional remote calls.
   - **ARCH-SECURE — Pass:** observation validation rejects malformed known data; display text is bounded and sanitized.
   - **ARCH-ORDER — Pass:** existing reservation transition governs warnings; uncertainty does not rearm admission.
   - **ARCH-FUNERAL — Pass:** no new durable artifact family; observations expire with the pass and existing gate/log lifetimes remain.
   - **ARCH-EVOLVE — Pass:** additive runtime fields; persisted formats are unchanged, and production logging retains original protocol bytes.

   Core-concept paths and PURE/INTEGRATION classifications match the code.

7. **Plan revision recommendations:** None.

```findings
{}
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bc90ffb5-1bd4-4700-a58e-c9be66636613","repository":"github.com/xianxu/pair","issue_id":"000705","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0327042a8fa97b00","reviewed_head":"7a71608c5bb57a4a05ab2c5dc87089482be9eeb1","slot":"pair:7","requested_at":"2026-10-19T16:07:49.437276Z"},"phase":"finished","pid":18570,"started_at":"2026-10-19T16:07:49.968478Z","finished_at":"2026-10-19T16:13:07.931435Z","session_id":"01a154eb-5364-72d3-8d12-b63768226fde","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.496701375s
head: "7a71608c5bb57a4a05ab2c5dc87089482be9eeb1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
