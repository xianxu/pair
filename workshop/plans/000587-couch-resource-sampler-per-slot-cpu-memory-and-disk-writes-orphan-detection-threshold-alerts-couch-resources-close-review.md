# Boundary Review — pair#587 (whole-issue close)

| field | value |
|-------|-------|
| issue | 587 — Couch resource sampler: per-slot CPU, memory and disk writes, orphan detection, threshold alerts, couch --resources |
| repo | pair |
| issue file | workshop/issues/000587-couch-resource-sampler-per-slot-cpu-memory-and-disk-writes-orphan-detection-threshold-alerts-couch-resources.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99e11ecb8c72b7e5325a58692c1efd679e50710c..0da6c7c91b78394c755bf593d735c5cdef1275cc |
| command | sdlc close --issue 587 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:38:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation provides useful separation between sampling, cached queries, persistence, and alert delivery. Targeted tests, race checks, and measured overhead passed. Shipping is blocked by oversized responses, permanent alert disablement after snapshot corruption, and ownership decisions made without complete inventory evidence.

1. **Strengths**

   - Native observations count failed attempts and fence process generations.
   - Alert envelopes are persisted before sending; real-broker tests cover self-delivery, queued admission, and uncertain outcomes.
   - History and staging files have explicit bounds. README, Atlas, and storage documentation cover the new surface.
   - Measured near-cap sampling cost passed: mean **67.492 ms**, maximum **68.272 ms**, or **0.112486%** of one core at default cadence.

2. **Critical findings**

   The machine-readable block below contains three runtime defects and the core-concepts table contradiction, classified Critical under the requested gate rule.

3. **Important findings**

   Production delivery bypasses the pure episode transition model; see the finding below.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed `make test-make-env`.
   - Passed targeted `Resource|ProductionArtifact` tests across all six affected packages.
   - Passed resource race tests in `couchcmd` and `couchmessage`.
   - Passed native owned-child conformance and the opt-in near-cap cost check.
   - Stopped the optional broader package run after several minutes without completion; no full-suite pass is claimed.
   - Missing regressions: maximum-size responses through the real transport, recovery from corrupt history with a valid checkpoint, and sampling before inventory discovery completes.
   - Coordinated live acceptance remains explicitly deferred in the plan.

6. **Architectural notes**

   - **ARCH-DRY — flag:** admission transitions are duplicated between the pure model and dispatcher.
   - **ARCH-PURE — flag:** delivery policy mutates authoritative episode state inside the IO adapter.
   - **ARCH-PURPOSE — flag:** snapshot corruption can permanently disable the alerting purpose.
   - **ARCH-MOCK — pass:** native seams, controllable endpoints, and real-broker tests provide useful coverage.
   - **ARCH-CONSTRAINTS — flag:** response size exceeds transport capacity; inventory truncation loses coverage information.
   - **ARCH-SECURE — pass:** bounded strict decoding, generation fences, private files, and sanitized labels are present.
   - **ARCH-ORDER — flag:** incomplete inventory becomes confirmed absence, and production bypasses the transition model.
   - **ARCH-FUNERAL — pass:** snapshots, stages, caches, and worker lifetimes have explicit bounds or cleanup owners.

7. **Plan revision recommendations**

   Add `## Revisions` entries documenting inventory completeness, independent snapshot recovery, the resource response byte envelope, and delivery outcomes flowing through the pure transition model. Correct the fallback filename in the core-concepts table. Keep live acceptance and full-suite qualification outstanding until evidenced.

```findings
findings:
  - id: new
    severity: Critical
    family: response-envelope-bounds
    title: |
      Resource responses exceed the existing socket frame limit within the supported process envelope
    detail: |
      cmd/internal/couchcmd/message_service.go:773 returns the entire ResourceView without a transport byte bound. ResourceMaxProcesses permits 8192 process rows, while couchmessage.MaxFrameBytes is 590848 bytes; a conservative serialization containing only a subset of the required process fields already occupies 1760214 bytes at that count. transport.go:213 rejects the response, so both resources CLI forms fail precisely when a large process fleet needs inspection. Add pagination or an explicitly bounded projection that preserves slot aggregates and discloses omitted details, and test maximum-size views through the real socket. ARCH-CONSTRAINTS.
  - id: new
    severity: Critical
    family: snapshot-failure-recovery
    title: |
      A corrupt history file permanently disables alerts despite a valid checkpoint
    detail: |
      cmd/internal/couchcore/resource_store.go:32 discards the successfully read checkpoint when history validation fails. cmd/internal/couchcmd/resources_persist.go:64 then returns on the latched loadError on every subsequent pass, with no reload or repair path; neither history nor alerts recover during that supervisor lifetime. This contradicts the plan's fresh-baseline recovery contract. Load the artifacts independently, preserve valid checkpoint suppression state, and rebuild invalid aggregate history with visible continuity loss. Add a regression using valid checkpoint state plus corrupt history that proves subsequent persistence and eligible alert delivery recover without duplicate admission. ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: ownership-absence-requires-evidence
    title: |
      Incomplete slot inventory is treated as confirmed ownership loss
    detail: |
      cmd/internal/couchcore/resource.go:240 treats a missing slot as known ownership loss. At startup, message_service.go:474 starts inventory discovery concurrently with the immediately sampling worker, so persisted live processes can be reported as proven orphans before discovery finishes. resources_inventory.go:109 and :199 also silently omit slots beyond the cap, hiding incompleteness from the reducer and its overflow diagnostics. Carry explicit inventory completeness and omission evidence into sampling; missing owners must remain unknown until authoritative inventory establishes absence. Add barrier-controlled restart and over-cap inventory regressions. ARCH-ORDER, ARCH-CONSTRAINTS.
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      The core-concepts table names a nonexistent platform adapter
    detail: |
      workshop/plans/000587-couch-resource-sampler-plan.md:41 names resource_other.go, but the pinned implementation supplies cmd/internal/procutil/resource_unsupported.go. Under the requested core-concepts gate this table/code contradiction requires correction. Update the table and record the actual fallback path in Revisions; the implemented unsupported capability itself exists.
  - id: new
    severity: Important
    family: authoritative-state-transition-boundary
    title: |
      Production alert admission bypasses the pure episode transition model
    detail: |
      cmd/internal/couchcmd/resources_alert.go:145, :152 and :172 assign episode phases directly. The Admitted event in couchcore.AdvanceResourceEpisode is exercised only by tests, so production admission does not pass through the model those tests validate. Move delivery-outcome transitions into the pure model, have the dispatcher return outcomes through that boundary, and prevent direct authoritative phase mutation. Cover admission, refusal, uncertainty and restart reconciliation through the production transition path. ARCH-ORDER, ARCH-PURE, ARCH-DRY.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"17675bd2-ffe6-4f8a-a3d4-0071384371a5","repository":"github.com/xianxu/pair","issue_id":"000587","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/192561dfb76ffa63","reviewed_head":"0da6c7c91b78394c755bf593d735c5cdef1275cc","slot":"pair:1","requested_at":"2026-10-16T00:31:58.515171Z"},"phase":"finished","pid":98119,"started_at":"2026-10-16T00:31:58.613029Z","finished_at":"2026-10-16T00:38:20.215669Z","session_id":"01a1421f-6f6f-7c32-8689-4af719225b42","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.354492458s
head: "0da6c7c91b78394c755bf593d735c5cdef1275cc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
