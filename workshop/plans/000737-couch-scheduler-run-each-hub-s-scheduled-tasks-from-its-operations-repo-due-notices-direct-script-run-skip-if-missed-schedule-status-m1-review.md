# Boundary Review — pair#737 (milestone M1)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 2d298efea85f7a623d0a8d4a65b3260e0c626f18..e5547ba789573abd77c9af33d67c993b8ca5a636 |
| command | sdlc milestone-close --issue 737 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T20:12:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 delivers substantial working foundations: captured builds, owned execution, durable run records, retention integration, and read-only status. Three reproduced correctness failures block admission. The broader tests also expose invalid skill commands, and the cache format and core-concepts inventory need correction. Automatic dispatch and notices remain appropriately scoped to M2.

```findings
findings:
  - id: new
    severity: Critical
    family: task-refusal-isolation
    title: |
      One invalid task name makes status and manual commands unusable
    detail: |
      cmd/internal/couchschedule/status.go:86 rejects invalid names even for refused tasks intentionally retained by Catalog.Discover at catalog.go:111. A temporary overlay test discovering BadTask reproduced “invalid schedule status task identity”; runScheduleCLI validates this snapshot before status, history, or manual execution. Represent malformed-entry refusals safely without invalidating the whole snapshot, and test refusal isolation through the CLI/protocol boundary. ARCH-PURPOSE, ARCH-SECURE.
  - id: new
    severity: Critical
    family: durable-artifact-crash-recovery
    title: |
      An abandoned cache publication blocks subsequent builds across the namespace
    detail: |
      cmd/internal/couchschedule/cache.go:304 creates .publication-* directories whose only cleanup is deferred, while inspectCache at line 137 rejects them as unknown entries. A process crash before rename leaves this production-created shape behind; an overlay test reproduced subsequent Build failure with “unknown cache entry: .publication-123456789”. Add identity-safe recovery under the maintenance lock and sweep interrupted publication, manifest-update, and eviction shapes; test abrupt termination rather than only returned errors. ARCH-ORDER, ARCH-FUNERAL.
  - id: new
    severity: Critical
    family: persisted-state-invariants
    title: |
      Accepted persisted phase combinations can panic the supervisor during recovery
    detail: |
      cmd/internal/couchschedule/state_json.go:78 validates stage names independently of source and required companion state. EncodeLedger and DecodeLedger accept a manual notifying attempt with Last=nil; Advance(OwnerUnknown) then dereferences state.Last at model.go:310, reachable from schedule_service.go:135. An overlay test reproduced the nil-pointer panic. Validate legal state combinations at decode and enforce them through transition-owned state updates; malformed storage must visibly block scheduling instead of crashing Couch. ARCH-SECURE, ARCH-ORDER.
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      The core-concepts inventory does not match the delivered M1 entities
    detail: |
      workshop/plans/000737-couch-scheduler-plan.md:33 places TaskKey in spec.go although it is defined in model.go; lines 35–36 list ScheduleState, Occurrence, and BuildKey, which have no corresponding definitions. Integration rows similarly use TaskCatalog/TaskBuilder instead of Catalog/Builder. Correct the greppable inventory and explicitly distinguish delivered M1 entities from M2 entities in a Revisions entry. This is a plan correction, classified Critical under the requested core-concepts gate. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: executable-documentation-validity
    title: |
      New skill examples break the existing command-validity test
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:86–93 embeds literal optional-argument notation and uppercase TASK placeholders in couch commands. Both the full package run and isolated TestSkillDocumentsRecovery fail because “couch --schedule --status [--json]” does not parse; TASK also violates ValidTaskName. Use runnable examples with a valid task slug, explain optional flags separately, and rerun the existing test. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: additive-persisted-format-compatibility
    title: |
      Cache manifests reject additive fields and block both reuse and rebuilding
    detail: |
      cmd/internal/couchschedule/cache.go:149 decodes BuildManifest with strictjson.Decode, and maintainCache rewrites it through json.Marshal without unknown-member preservation. An overlay test adding future_metadata to an otherwise valid manifest reproduced “json: unknown field future_metadata” on the next Build. Preserve unknown members at the manifest and input-entry levels while retaining strict owned-field validation, and test older-reader read-modify-write compatibility. ARCH-EVOLVE.
```

1. **Strengths**

   - Captured-source tests verify dependency/embed changes, cache reuse, and compilation from captured bytes after live source changes.
   - Native execution tests exercise cancellation, resistant descendants, parent death, and cleanup before permit release.
   - Manual completion publishes immutable history before clearing authority; recovery tests preserve uncertain ownership without replay.
   - README and atlas updates explain M1 availability, trusted execution, ownership, and the M2 boundary.

2. **Critical findings**

   The four Critical findings above require fixes and another review. The first three have failing behavioral reproductions; the inventory correction requires concrete source-to-table reconciliation.

3. **Important findings**

   Repair the runnable skill examples and additive cache-manifest handling before crossing the boundary.

4. **Minor findings**

   None.

5. **Test coverage notes**

   The six-package command passed `couchschedule`, `operationalrun`, `gcruntime`, `couchmessage`, and `artifactpath`; `couchcmd` failed `TestSkillDocumentsRecovery`. Its isolated rerun also failed. Four temporary overlay tests reproduced the three correctness defects and additive-manifest refusal. Overlay files were removed; the checkout remains unchanged. `git diff --check` passed.

   Add retained regressions for those failures, including actual crash recovery. Existing returned-error publication tests execute deferred cleanup and therefore miss abandoned artifacts. The real heavy-lock qualification is opt-in and was skipped in this run.

6. **Architectural notes for upcoming work**

   | Marker | Assessment |
   |---|---|
   | ARCH-DRY | Pass: manual execution shares the builder, executor, store, and retention machinery. |
   | ARCH-PURE | Pass: recurrence and transition decisions accept explicit observations; IO remains in adapters. |
   | ARCH-PURPOSE | Flag: refusal isolation, executable documentation, and inventory discrepancies above. M2 deferral is documented. |
   | ARCH-MOCK | Pass for inspected M1 execution seam: stateful runtime fake plus native qualification tests; heavy native qualification remains opt-in. |
   | ARCH-CONSTRAINTS | Pass: explicit task/input/output/cache bounds and two shared execution permits are implemented and tested. |
   | ARCH-SECURE | Flag: accepted persisted combinations can crash recovery; malformed names invalidate the status boundary. |
   | ARCH-ORDER | Flag: persisted-state invariants and interrupted cache mutations need enforcement and recovery coverage. |
   | ARCH-FUNERAL | Flag: cache publication residue lacks recovery. Run bundles otherwise have explicit collection. |
   | ARCH-EVOLVE | Flag: cache manifests lack additive compatibility; authority and operational-record codecs have preservation tests. |

   Before M2 consumes this API, move controller identity assignment into a reducer event: `schedule_manual.go:132–134` currently mutates authoritative attempt fields directly despite the plan’s transition-ownership contract.

7. **Plan revision recommendations**

   Add `## Revisions` entries that reconcile the core-concepts table with actual symbols and paths, define persisted phase invariants, and specify recovery for every interrupted cache mutation. Update the compatibility description to include preserving cache-manifest codecs.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"beea5dbc-2dca-42f7-ba3d-461ee3c6c965","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/21f3e34c061adac5","reviewed_head":"e5547ba789573abd77c9af33d67c993b8ca5a636","slot":"pair:1","requested_at":"2026-10-21T03:08:13.834329Z"},"phase":"finished","pid":8223,"started_at":"2026-10-21T03:08:14.053418Z","finished_at":"2026-10-21T03:12:30.290329Z","session_id":"01a15c6e-4b34-7e53-be6a-cc43c9dd732c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.763477417s
head: "e5547ba789573abd77c9af33d67c993b8ca5a636"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T20:40:38-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 737 — couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status) |
| repo | pair |
| issue file | workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 2d298efea85f7a623d0a8d4a65b3260e0c626f18..e34a33888d9441e83bc815b5a8f4f6ddcf28d979 |
| command | sdlc milestone-close --issue 737 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T20:40:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All six prior findings are addressed, with regression evidence for the executable corrections. M1’s manual execution, durable authority, status/history, and documentation are substantially delivered; automatic scheduling remains explicitly scoped to M2. One additional crash-cleanup gap should be fixed before crossing the boundary.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Socket/CLI refusal-isolation and valid-manual-run regressions pass. Disabling diagnostic-entry validation support makes TestScheduleMixedRefusalStatusAndHistoryThroughSocket fail with invalid schedule status task identity.
  - id: BR-2
    disposition: addressed
    note: |
      Cache recovery covers publication, legacy staging, touch and eviction interruptions with abrupt subprocess exits. Disabling recoverCacheLocked makes TestCacheAbruptMutationRecovery/publication-task-written fail on retained aged residue. The new finding concerns staging outside this repaired cache path.
  - id: BR-3
    disposition: addressed
    note: |
      Encode/decode and Advance share phase validation. Source/stage/companion regressions pass; removing persistence-boundary validation makes TestStateJSONRefusesInconsistentCompanions/notice_missing_summary fail.
  - id: BR-4
    disposition: addressed
    note: |
      Inspected the before/after inventory and source declarations. The Core concepts tables now name TaskKey in model.go, Catalog, Builder and scheduleCoordinator, and explicitly label scheduleNotice and FinalizeNotice as planned M2. The Revisions passage records that distinction.
  - id: BR-5
    disposition: addressed
    note: |
      Skill examples now use runnable landing-plan commands and explain optional flags separately. TestSkillDocumentsRecovery passes at head; overlaying the previous skill makes it fail on couch --schedule --status [--json].
  - id: BR-6
    disposition: addressed
    note: |
      Manifest and input codecs preserve unknown members through reuse and rebuilding. TestCacheFutureManifestMetadataSurvivesReuseAndRebuild passes; restoring unknown-member rejection makes the regression fail.
findings:
  - id: new
    severity: Important
    kind: defect
    family: durable-artifact-crash-recovery
    title: |
      Build captures and ledger staging still have only deferred cleanup
    detail: |
      cmd/internal/couchschedule/build.go:86 creates couch-build-* directories outside the managed cache, and store_unix.go:132 creates .state-*.tmp snapshots; both rely solely on deferred removal. Temporary overlay subprocess tests exiting abruptly reproduced both abandoned shapes, and a subsequent successful StateStore.Update left the old snapshot untouched. This is the 2nd finding in family durable-artifact-crash-recovery, with two remaining staging families found. State the rule that every scheduler-owned staging artifact needs owner-safe crash cleanup, enumerate all staging writers, and sweep the class in this round. Add identity-fenced recovery under the appropriate locks and abrupt-death regressions; build captures and repeated ledger residues otherwise accumulate outside the declared retention bounds. ARCH-FUNERAL, ARCH-PURPOSE.
```

1. **Strengths**

   - Refused filesystem entries now remain diagnostic rows without gaining runnable task identities.
   - Cache mutation tests exercise actual abrupt process termination, including partially written publications and eviction.
   - Pure phase validation is shared by transitions and persisted-state codecs.
   - README and atlas document the new commands, execution privileges, retention, and M1/M2 boundary.

2. **Critical findings**

   None.

3. **Important findings**

   The staging-cleanup finding above. Recover abandoned build captures and ledger temporary files without removing live or replaced artifacts.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Scheduler, operational-run, GC runtime, message protocol, and artifact inventory package checks passed. Targeted schedule/skill CLI checks passed. Temporary overlays demonstrated regression failures for BR-1, BR-2, BR-3, BR-5, and BR-6. The additional abrupt-exit probes confirmed both remaining residue shapes. `git diff --check` passed; the checkout remains unchanged.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared builder, executor, codecs, and phase validator.
   - **ARCH-PURE — pass:** declaration, calendar, transition, manifest-key, and status logic remain directly testable without IO.
   - **ARCH-PURPOSE — flag:** crash recovery still needs the complete staging-artifact sweep identified above.
   - **ARCH-MOCK — pass:** injected stateful execution fake, portable stores, and native boundary qualifications support M1.
   - **ARCH-CONSTRAINTS — pass:** explicit task/input/output/cache limits and two execution permits bound active work.
   - **ARCH-SECURE — pass:** refusal isolation, strict persisted-field parsing, argv execution, and ownership checks are supported by tests.
   - **ARCH-ORDER — pass:** durable reservations precede authorization; malformed phases and uncertain outcomes retain authority.
   - **ARCH-FUNERAL — flag:** build captures and ledger staging lack crash cleanup.
   - **ARCH-EVOLVE — pass:** inspected preserving codecs and nested unknown-member regression coverage.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating scheduler staging families and specifying each family’s owner, crash-recovery lock, identity checks, removal trigger, and regression evidence. Include captured build views and temporary ledger snapshots alongside cache and run-bundle staging.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bce4ef5d-9dfc-49ed-9f85-e4c114d7300e","repository":"github.com/xianxu/pair","issue_id":"000737","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f599858499d90cc7","reviewed_head":"e34a33888d9441e83bc815b5a8f4f6ddcf28d979","slot":"pair:1","requested_at":"2026-10-21T03:36:38.96183Z"},"phase":"finished","pid":65864,"started_at":"2026-10-21T03:36:39.154577Z","finished_at":"2026-10-21T03:40:38.048196Z","session_id":"01a15c88-4f9b-72f2-ae3a-e3f25dcac1a4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.840563542s
head: "e34a33888d9441e83bc815b5a8f4f6ddcf28d979"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
