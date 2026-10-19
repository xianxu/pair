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
