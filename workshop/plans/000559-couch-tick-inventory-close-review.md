# Boundary Review — pair#559 (whole-issue close)

| field | value |
|-------|-------|
| issue | 559 — Stop reparsing inventory on Couch ticks |
| repo | pair |
| issue file | workshop/issues/000559-couch-tick-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6af746c94f18ef4a58cb092a739cc9fe0887124..e541a6d669708ef6743504511c9d9868c53541e3 |
| command | sdlc close --issue 559 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:28:05-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main cache wiring and warm-path performance regressions are sound. Shipping is blocked by incomplete-observation invalidation gaps. The previous-launch cache also retains unnecessary historical payloads, and activity publication lacks the adversarial ordering tests promised by the plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: incomplete-observation-invalidates-evidence
    title: |
      Failed observations can preserve or publish reusable evidence
    detail: |
      cmd/internal/sessioninventory/query.go:370-389 returns on failed listing/stat or missing ledger without invalidating the existing owner entry; after recovery with unchanged metadata, QueryResumeTarget reuses the old parse. Separately, activity_cache.go:378-379 does not record catalog-load failure, which QuerySessionContext intentionally tolerates; successful native fallback can then publish a ready activity entry at lines 212-216 despite incomplete catalog observation. This contradicts the Spec's explicit invalidation rule (ARCH-SECURE, ARCH-ORDER). Invalidate owner evidence on incomplete source observations and prevent activity publication after failed catalog observation; retain uncached fallback behavior. Add failure→recovery regressions proving fresh observation occurs.
  - id: new
    severity: Important
    kind: defect
    family: compact-cache-projections
    title: |
      Previous-launch reuse retains full historical boundary and proof payloads
    detail: |
      cmd/internal/sessioninventory/query.go:476-478 retains the complete previous sessionledger.Current in ownerLaunchEntry.previous (launch_cache.go:34), including launch boundary arrays and binding scanner proofs. ResumeTargetForLaunch consumes only identity/status fields, so these potentially large payloads remain resident unnecessarily across up to 128 entries (ARCH-CONSTRAINTS, ARCH-FUNERAL). Store a compact fallback projection and audit both retained launch projections for unused payloads. Add a substantive large-boundary fixture and retained-memory evidence; the current scale fixture expands files with whitespace and cannot detect this retention.
  - id: new
    severity: Important
    kind: defect
    family: temporal-cache-regression-coverage
    title: |
      Activity publication lacks the promised controlled ordering regressions
    detail: |
      cmd/internal/sessioninventory/activity_cache_test.go:89-106 cancels the purported waiter before Query starts, so it never exercises waiting behind the loader. The activity tests also do not replace dependencies during a blocked load or force repeated drift before publication. Owner/index race tests do not exercise the activity tracker and commit path (ARCH-ORDER). Add barrier-controlled activity mutation, active-loader cancellation, and genuinely queued-waiter tests that fail when the corresponding publication guards are removed.
```

1. **Strengths**

   - Collision-checker copies share constructor-owned index and ledger caches; integration tests exercise both consumers.
   - Cache misses retain the existing strict decoders, launch reducers, and scanner validation.
   - The 20-slot activity regression exercises 600 MB of cold ledger parsing and asserts zero unchanged body-read/decoder work.
   - Atlas documentation explains ownership, invalidation, saturation, and remaining cold costs. No new operator-facing command/configuration surface requires README changes.

2. **Critical findings**

   The incomplete-observation finding above violates the Spec’s explicit “invalidate … any incomplete observation” contract. Sweep all cache observation failure paths when correcting it.

3. **Important findings**

   Compact the retained fallback evidence and complete the activity ordering regressions described above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed launcher, sessioninventory, and threadactivity package tests; targeted cache/wiring tests under `-race`; the named message invariant; and targeted artifact source-inventory checks. `git diff --check` passed.

   The broader package run was interrupted after roughly six minutes while couchcore remained running; it is not a full-suite pass. The checkout remained unchanged.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass: existing parsers, reducers, and validators reused. |
   | ARCH-PURE | Pass: projection/matching functions have direct pure tests; IO remains behind runtime seams. |
   | ARCH-PURPOSE | Flag: invalidation and compact-evidence commitments remain incomplete. |
   | ARCH-MOCK | Pass: injected runtimes and temporary file worlds cover the changed seams; no new external dependency. |
   | ARCH-CONSTRAINTS | Flag: entry-count bounds do not establish compact retained payloads. |
   | ARCH-SECURE | Flag: incomplete observations can become reusable evidence. |
   | ARCH-ORDER | Flag: failure invalidation and activity publication ordering coverage need correction. |
   | ARCH-FUNERAL | Flag: unnecessary previous-generation payloads survive until eviction/caller exit. No new durable file family was introduced. |

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying failure invalidation across each cache, compact current/previous evidence fields, and the concrete activity ordering regressions. Refresh the Core concepts table with actual function names, paths, PURE/INTEGRATION kinds, and new/modified statuses. Keep the affected checklist claims open until those corrections are verified.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.91872875s
head: "e541a6d669708ef6743504511c9d9868c53541e3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
