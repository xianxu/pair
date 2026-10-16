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

---

## Re-review — 2026-10-15T17:53:40-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 587 — Couch resource sampler: per-slot CPU, memory and disk writes, orphan detection, threshold alerts, couch --resources |
| repo | pair |
| issue file | workshop/issues/000587-couch-resource-sampler-per-slot-cpu-memory-and-disk-writes-orphan-detection-threshold-alerts-couch-resources.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99e11ecb8c72b7e5325a58692c1efd679e50710c..5e7cf327faa8276d1ceb4fa07a0948c071ab5992 |
| command | sdlc close --issue 587 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:53:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All five prior findings are addressed, with meaningful regression evidence. Two blockers remain: the checkpoint cannot accommodate the supported process count, preventing alert delivery, and the core-concepts table still contradicts implemented entity names. README and atlas updates are present.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Cached byte-budgeted projection preserves slot aggregates and discloses omitted details. TestResourceProjectionTransportCapacity passes through a real socket and fails with response exceeds frame limit when projection is disabled.
  - id: BR-2
    disposition: addressed
    note: |
      Snapshot loading preserves healthy sibling artifacts and rebuilds corrupt history. TestResourcePersistenceCorruptHistoryPreservesCheckpointAndAlerts passes; restoring all-or-nothing loading makes it fail on lost suppression state.
  - id: BR-3
    disposition: addressed
    note: |
      Inventory completeness and omission evidence reach the reducer. Startup-barrier, over-cap, cancellation and positive-root-death tests pass. Removing the completeness guard makes both the barrier and pure ownership regressions fail.
  - id: BR-4
    disposition: addressed
    note: |
      The pinned plan diff corrects its NativeResourceReader row to resource_unsupported.go at line 41. That file exists and supplies the !darwin || !cgo fallback; Revisions records the correction.
  - id: BR-5
    disposition: addressed
    note: |
      Production delivery observations invoke AdvanceResourceEpisode. Admission, refusal, uncertainty and restart reconciliation tests pass; substituting the previous dispatcher makes all six delivery-outcome cases fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: persistence-envelope-bounds
    title: |
      Supported process counts exceed the checkpoint cap and prevent alert admission
    detail: |
      cmd/internal/couchcmd/resources_alert.go:200 rejects the entire checkpoint above 2 MiB, while resources_persist.go:116 and resources_alert.go:171 require that checkpoint before sending. A disposable pinned-head regression with 8192 valid provenance rows, a short /f/pair workspace and one pending alert serialized to 3718809 bytes; persist returned resource checkpoint exceeds byte cap and left the alert pending without sending. Repeated workspace metadata makes this reachable within the advertised operating envelope. Normalize or compact persisted ownership, reserve capacity for suppression/envelopes, and test maximum-cardinality persistence plus real-broker admission. ARCH-CONSTRAINTS, ARCH-PURPOSE.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      Additional core-concepts rows still name entities absent from their stated files
    detail: |
      workshop/plans/000587-couch-resource-sampler-plan.md:26 names ResourceObservation instead of ResourceUsageObservation; line 45 names ResourceStore instead of ResourceSnapshotStore; line 42 names ResourceWorld instead of resourceFakeWorld. The naming Revisions paragraph already recognizes the first two renames, but the tables remain contradictory. This is the 2nd finding in family core-concept-traceability. Earlier rounds fixed an instance: apply the rule that every table entity resolves to its implemented declaration, enumerate and sweep every row, and record the corrected names/classifications in Revisions. Under the requested core-concepts gate these contradictions are Critical. ARCH-PURPOSE.
```

1. **Strengths**

   - The transport regression exercises actual socket framing at maximum cardinality.
   - Recovery tests prove valid alert suppression survives corrupt history while new alerts remain eligible.
   - Barrier-controlled inventory tests distinguish unknown ownership from confirmed absence.
   - README, atlas, storage retention documentation and artifact registration cover the new surface.

2. **Critical findings**

   - Checkpoint capacity prevents alerts within the supported process envelope. Compact the persisted representation and add a maximum-cardinality admission regression.
   - Sweep and correct the entity tables, including the three mismatches identified above.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Resource/ProductionArtifact checks passed across procutil, couchcore, couchcmd, couchmessage, couchtty and artifactpath. Negative controls confirmed the four behavioral corrections fail without their fixes. The new checkpoint reproduction fails against restored pinned-head code. The disposable copy was removed; the reviewed checkout remains unchanged.

   Full recorded-suite qualification and coordinated live acceptance remain outstanding in the tracker; neither is counted as passed here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing broker, atomic writer and ownership seams are reused.
   - **ARCH-PURE — pass:** attribution, episodes and history policy are tested separately from IO.
   - **ARCH-PURPOSE — flag:** checkpoint capacity prevents required alert behavior.
   - **ARCH-MOCK — pass:** injected readers, controlled broker endpoints and owned-child conformance exercise production boundaries.
   - **ARCH-CONSTRAINTS — flag:** wire bounds are corrected; persistence bounds remain incompatible with supported cardinality.
   - **ARCH-SECURE — pass:** bounded validation, unsafe-path rejection and terminal sanitization are present.
   - **ARCH-ORDER — pass:** production delivery outcomes enter the pure reducer; uncertainty and incomplete inventories remain explicit.
   - **ARCH-FUNERAL — pass:** bounded snapshots, fixed replacement stages, history pruning and joined worker lifetimes provide cleanup paths.

7. **Plan revision recommendations**

   Add `## Revisions` entries documenting the checkpoint representation and its verified capacity, and the complete entity-table sweep with actual names and classifications. Preserve the outstanding live-acceptance and recorded-suite requirements.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ffa6bd7c-d63f-4826-a6c8-48b568e981ab","repository":"github.com/xianxu/pair","issue_id":"000587","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f82c5d161af4a6df","reviewed_head":"5e7cf327faa8276d1ceb4fa07a0948c071ab5992","slot":"pair:1","requested_at":"2026-10-16T00:48:44.849916Z"},"phase":"finished","pid":18945,"started_at":"2026-10-16T00:48:45.153315Z","finished_at":"2026-10-16T00:53:28.945182Z","session_id":"01a1422e-cc5d-7022-a5bc-24a013651a31","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.895139667s
head: "5e7cf327faa8276d1ceb4fa07a0948c071ab5992"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
