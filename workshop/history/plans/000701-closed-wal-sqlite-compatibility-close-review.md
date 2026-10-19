# Boundary Review — pair#701 (whole-issue close)

| field | value |
|-------|-------|
| issue | 701 — Read closed WAL databases without sidecars on macOS SQLite |
| repo | pair |
| issue file | workshop/issues/000701-closed-wal-sqlite-compatibility.md |
| boundary | whole-issue close |
| milestone | — |
| window | fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812..9961f834ac6b08fbe2350a3a0872bfbfeea8e2f7 |
| command | sdlc close --issue 701 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T12:32:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The compatibility route succeeds on macOS SQLite 3.54.0, preserves opaque schema and rowids, and retains active-writer progress. Both reviewed packages pass. One inexpensive correction remains: distinguish proven unsafe paths from uncertain capture evidence, as the plan requires.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: failure-classification-preserves-evidence
    title: |
      Proven unsafe paths are classified as temporary capture uncertainty
    detail: |
      cmd/internal/historyarchive/sqlite_snapshot.go:146-149 unconditionally changes every cold-path error to SnapshotDeferred. observeColdSource explicitly detects nonregular sidecars at sqlite_cold.go:62-63, but an existing -journal symlink consequently becomes deferred rather than rejected; sqlite_cold_test.go:331-335 asserts this incorrect classification. The plan's Operator projection requires proven unsafe paths/configuration to be rejected/failed. Preserve a distinct error category for proven unsafe paths and local output failures, defer only uncertain source evidence/deadlines, and update the regression to distinguish these outcomes while retaining prior-stage preservation.
```

1. **Strengths**

   - Descriptor ownership, matching full-read digests and source/parent metadata fences provide the documented conservative capture contract.
   - Controlled real-writer tests cover commits and rollbacks during both reads, including transient sidecars.
   - Both acquisition routes reuse private validation, atomic staging and leased cleanup.
   - README and atlas describe the new route, filesystem assumptions and retry action.

2. **Critical findings**

   None.

3. **Important findings**

   The failure-classification finding above is localized to `sqlite_snapshot.go:146–149`. Use typed errors to preserve the distinction between uncertainty and established unsafe configuration. The existing journal-symlink fixture should expect rejection and fail against the current implementation.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - `go test ./cmd/internal/historyarchive ./cmd/internal/artifactpath -count=1` passed: 131.493s and 16.838s.
   - Focused SQLite/cold-route qualification passed in 11.953s.
   - Native characterization independently reproduced the closed-WAL refusal on configured SQLite and `/usr/bin/sqlite3`; production snapshots succeeded.
   - Active-writer conformance completed a coherent snapshot while the writer committed 280 transactions.
   - Opt-in native acceptance tests remained skipped. The recorded whole-branch suite remains an explicitly pending delivery obligation.
   - Checkout unchanged; `git diff --check` passed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared hashing, validation, publication and cleanup.
   - **ARCH-PURE — pass:** header/stamp rules have direct tests without IO; filesystem operations stay in integration code.
   - **ARCH-PURPOSE — flag:** the compatibility goal is delivered, but failure classification differs from the approved operator contract.
   - **ARCH-MOCK — pass:** stateful mutation fixtures share the production seam; real SQLite conformance checks the modeled behavior.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, fixed buffers, size caps and counted scaling tests.
   - **ARCH-SECURE — pass:** source descriptors remain read-only, unsafe sidecars prevent publication, and fixtures use disposable storage.
   - **ARCH-ORDER — pass:** capture state is invocation-local; controlled gates exercise mutation during both reads and prevent fallback after uncertainty.
   - **ARCH-FUNERAL — pass:** existing defer, lease and sweep mechanisms own private residue.
   - **ARCH-EVOLVE — pass:** no persisted codec changes; opaque SQL content survives validation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry documenting the corrected error categories and regression expectations. Update qualification checkboxes when their recorded evidence is complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"92354b2b-68ac-48c1-b086-4d124c99b1f5","repository":"github.com/xianxu/pair","issue_id":"000701","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/23bf2b92c83c638c","reviewed_head":"9961f834ac6b08fbe2350a3a0872bfbfeea8e2f7","slot":"pair:7","requested_at":"2026-10-19T19:29:48.035044Z"},"phase":"finished","pid":47644,"started_at":"2026-10-19T19:29:48.326347Z","finished_at":"2026-10-19T19:32:46.650706Z","session_id":"01a155a4-3b14-7130-bca6-e3e0974910fc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.119849042s
head: "9961f834ac6b08fbe2350a3a0872bfbfeea8e2f7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T12:38:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 701 — Read closed WAL databases without sidecars on macOS SQLite |
| repo | pair |
| issue file | workshop/issues/000701-closed-wal-sqlite-compatibility.md |
| boundary | whole-issue close |
| milestone | — |
| window | fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812..3375a20570c743da2cf33165e0ef26fc468b73c1 |
| command | sdlc close --issue 701 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T12:38:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The cold-WAL route matches the documented conservative capture contract and preserves the existing active-source route. BR-1 is addressed with executable regression evidence. Both reviewed packages pass; no new blocking findings. The recorded whole-branch suite remains a documented landing obligation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Typed unsafe-source/output errors preserve rejection while uncertain capture evidence defers. TestSQLiteColdRefusalPreservesPreviousStage/unsafe-sidecar passes at pinned HEAD and fails when unconditional deferral is restored in a disposable checkout. Bounds, output-collision, write-provenance and uncertainty controls also pass.
```

1. **Strengths**

   - Held descriptors, metadata fences and matching full-read digests implement observed quiescence without opening the cold native database through SQLite (`sqlite_cold.go:95`).
   - Controlled real writers exercise commits and rollbacks during both reads, including transient sidecars.
   - Both routes share private validation, atomic staging and leased cleanup (`sqlite_snapshot.go:138`).
   - README and atlas explain the compatibility route, assumptions and recovery actions.

2. **Critical findings**

   None.

3. **Important findings**

   None remaining.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - `go test ./cmd/internal/historyarchive ./cmd/internal/artifactpath -count=1` passed: 33.777s and 7.247s.
   - BR-1 mutation check passed at HEAD and failed after restoring blanket deferral; the disposable checkout was removed.
   - Coverage includes native CLI qualification, opaque schema/rowid preservation, controlled mutation sequences, counted costs, cancellation, previous-stage retention and continuous publication.
   - Pinned-range `git diff --check` passed; review checkout unchanged.
   - The recorded whole-branch suite was not verified by this review.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** hashing, validation, staging and cleanup remain shared.
   - **ARCH-PURE — pass:** header/stamp rules have direct tests without IO; filesystem work stays in integration code.
   - **ARCH-PURPOSE — pass:** delivers closed-WAL compatibility while retaining active-writer backup behavior and honest failure categories.
   - **ARCH-MOCK — pass:** stateful fixtures share production seams; real SQLite conformance checks their assumptions.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, fixed buffers, existing size/deadline limits and counted scaling tests.
   - **ARCH-SECURE — pass:** read-only source descriptors, nofollow checks, exclusive private output and disposable test storage.
   - **ARCH-ORDER — pass:** capture state is invocation-local; controlled gates exercise both reads, and uncertainty cannot trigger fallback.
   - **ARCH-FUNERAL — pass:** existing defer, lease and sweep mechanisms own private residue.
   - **ARCH-EVOLVE — pass:** persisted codecs are unchanged; opaque SQL content survives private validation.

7. **Plan revision recommendations**

   No new `## Revisions` entry required. The BR-1 revision accurately describes the correction. Complete outstanding qualification checkboxes when their evidence is available.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"01e61bd3-7bb3-4457-9105-c5236cf09a76","repository":"github.com/xianxu/pair","issue_id":"000701","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5df0b937ad9819fa","reviewed_head":"3375a20570c743da2cf33165e0ef26fc468b73c1","slot":"pair:7","requested_at":"2026-10-19T19:36:56.50717Z"},"phase":"finished","pid":30706,"started_at":"2026-10-19T19:36:56.6156Z","finished_at":"2026-10-19T19:38:05.86683Z","session_id":"01a155aa-c328-7161-8be4-03981b7fba0a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.565610042s
head: "3375a20570c743da2cf33165e0ef26fc468b73c1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
