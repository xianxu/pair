# Boundary Review — pair#688 (whole-issue close)

| field | value |
|-------|-------|
| issue | 688 — Codex database snapshots can starve under continuous writes |
| repo | pair |
| issue file | workshop/issues/000688-codex-snapshot-freshness.md |
| boundary | whole-issue close |
| milestone | — |
| window | a6b6973dcf7dab9d192605c4f647fefc9d5d4981..73ef6ffca269065079768347a0cc32efc7d2c3d6 |
| command | sdlc close --issue 688 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T03:09:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The bounded snapshot mechanism, source isolation, and freshness projection are well supported by code and passing tests. One publication failure path still violates the plan’s database-isolation contract: it prevents later databases from publishing and leaves their summaries pending.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: per-database-failure-isolation
    title: |
      Publication failure strands later validated databases
    detail: |
      cmd/internal/historyarchive/continuous.go:240-244 returns from archiveRoot after any ordinary publication error. With multiple fresh databases, a file-specific encoding or bundle-conflict failure therefore prevents later databases from publishing; their summaries remain failed with “snapshot validated; publication pending” after the pass has ended. This contradicts the plan’s partial-failure isolation contract (line 90; ARCH-PURPOSE). Accumulate per-database publication errors and continue remaining snapshot requests unless the context is cancelled. Add a production-pass regression where the first publication fails and a later database archives successfully, with accurate final summaries and an overall error.
```

1. **Strengths**

   - Source acquisition and private validation use separate bounded invocations; staging replacement requires successful validation (`sqlite_snapshot.go:133–179`).
   - Kernel alarms and inherited scratch leases address collector death, with controlled owner-death tests.
   - Real SQLite conformance checks transaction consistency, unknown schema, sparse rowids, and unchanged source DB/WAL bytes.
   - Freshness timestamps reuse the catalog index without changing persisted formats. README and atlas updates cover the new behavior.

2. **Critical findings**

   None.

3. **Important findings**

   The publication-isolation finding above. Fix the return at `continuous.go:244` for per-database failures while preserving immediate cancellation and aggregate error reporting.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `historyarchive`, `execalarm`, `archivecmd`, and `pair-go`
   - `artifactpath` and `testpair`
   - `git diff --check`

   The new counted integration test exercises successful, unchanged, and fully deferred passes. It lacks mixed publication outcomes, which would expose the finding. I inspected the retained passing history-publication log; native acceptance was not rerun.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared runner and catalog projection avoid parallel implementations.
   - **ARCH-PURE — pass:** classification, budget calculation, index projection, and rendering have direct pure tests.
   - **ARCH-PURPOSE — flag:** publication failures still strand unrelated databases.
   - **ARCH-MOCK — pass:** injected stateful runner is supplemented by actual SQLite conformance.
   - **ARCH-CONSTRAINTS — pass:** serial acquisition, bounded streams, size admission, and enforced read budgets.
   - **ARCH-SECURE — pass:** regular-file checks, source identity fences, literal argv, and isolated fixture roots.
   - **ARCH-ORDER — pass:** synchronous process sequencing and explicit cancellation prevent uncertain acquisition from publishing.
   - **ARCH-FUNERAL — pass:** scratch cleanup and inherited leases cover ordinary completion and owner death; staging retains its existing replacement lifecycle.
   - **ARCH-EVOLVE — pass:** persisted formats remain unchanged; opaque SQLite content preservation is tested.

7. **Plan revision recommendations**

   Add a `## Revisions` entry documenting publication-failure isolation and its mixed-outcome regression. The existing contract should remain: unrelated databases continue unless the caller cancels.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1835dbc8-4b3d-499a-844f-d3953869481e","repository":"github.com/xianxu/pair","issue_id":"000688","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9f3934c902e40db2","reviewed_head":"73ef6ffca269065079768347a0cc32efc7d2c3d6","slot":"pair:7","requested_at":"2026-10-19T10:07:19.088536Z"},"phase":"finished","pid":21972,"started_at":"2026-10-19T10:07:19.289723Z","finished_at":"2026-10-19T10:09:56.54869Z","session_id":"01a153a1-432f-77e2-b00c-17997d94ad49","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.273186917s
head: "73ef6ffca269065079768347a0cc32efc7d2c3d6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T03:20:48-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 688 — Codex database snapshots can starve under continuous writes |
| repo | pair |
| issue file | workshop/issues/000688-codex-snapshot-freshness.md |
| boundary | whole-issue close |
| milestone | — |
| window | a6b6973dcf7dab9d192605c4f647fefc9d5d4981..ba8519c1beebeb3961cc0891f99515886550cd05 |
| command | sdlc close --issue 688 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T03:20:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is addressed, and all six reviewed package suites pass. One source-isolation defect blocks shipping: staging can overlap a different native source root and overwrite its files. A disposable regression reproduced the overwrite.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      continuous_snapshot_integration_test.go:70 exercises successive publication failures, later success, aggregate errors, terminal summaries and cancellation. Restoring the early return in a disposable checkout makes this regression fail; pinned-head tests pass.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-source-output-isolation
    title: |
      Cross-kind staging overlap can overwrite another native source
    detail: |
      cmd/internal/historyarchive/continuous.go:71-77 checks each native root only against staging for that same kind. With snapshots/codex-database symlinked to home/.claude, both roots pass validation, then sqlite_snapshot.go:182 replaces home/.claude/state.sqlite. A disposable production-pass regression observed its original bytes replaced by "version 0". ARCH-SECURE and ARCH-PURPOSE require every snapshot output to be disjoint from every native source, including filtered kinds. Validate the complete source/output cross-product before mutation and add cross-kind alias and direct-overlap regressions.
```

1. **Strengths**

   - Acquisition and private validation use separate bounded invocations; generic SQLite errors are not guessed to mean contention.
   - Kernel alarms and inherited scratch leases cover collector death, with actual SQLite conformance tests.
   - BR-1 now preserves later publication progress and aggregates failures.
   - README and atlas document the new reader contract, freshness reporting, limits and acceptance procedures.

2. **Critical findings**

   - The cross-kind overlap defect above. Compare every native root against every staging destination and scratch before cleanup or directory creation.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed `historyarchive`, `execalarm`, `archivecmd`, `pair-go`, `artifactpath` and `testpair`. BR-1’s negative control failed as expected. The added disposable cross-source regression failed on pinned head; temporary checkouts were removed. Inspected the retained successful history-publication log; native acceptance was not rerun.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared runner and existing publication/index machinery reused.
   - **ARCH-PURE — pass:** policy and reporting tests run without IO; integration work uses injected seams.
   - **ARCH-PURPOSE — flag:** source preservation remains incomplete across source kinds.
   - **ARCH-MOCK — pass:** stateful fake supplemented by actual SQLite conformance.
   - **ARCH-CONSTRAINTS — pass:** bounded acquisition, validation and output streams; no acquisition retry.
   - **ARCH-SECURE — flag:** cross-kind staging can overwrite native files.
   - **ARCH-ORDER — pass:** synchronous acquisition/validation ordering and controlled cancellation/owner-death tests.
   - **ARCH-FUNERAL — pass:** scratch cleanup respects leases; staging replacement and retained-bundle costs are documented.
   - **ARCH-EVOLVE — pass:** no persisted format changes; opaque-schema preservation is tested.

7. **Plan revision recommendations**

   Add a `## Revisions` entry stating the complete invariant: **every snapshot staging/scratch destination must be canonically disjoint from every native source root, regardless of selected kind**. Require cross-kind direct-path and symlink regressions before any mutation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7ea0d15c-2165-42a1-af19-aa66ba5b0ca8","repository":"github.com/xianxu/pair","issue_id":"000688","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6d197c0735f85358","reviewed_head":"ba8519c1beebeb3961cc0891f99515886550cd05","slot":"pair:7","requested_at":"2026-10-19T10:17:41.607821Z"},"phase":"finished","pid":99585,"started_at":"2026-10-19T10:17:41.840006Z","finished_at":"2026-10-19T10:20:46.381348Z","session_id":"01a153aa-c2a5-7c91-95eb-63c224057db6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.467323083s
head: "ba8519c1beebeb3961cc0891f99515886550cd05"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
