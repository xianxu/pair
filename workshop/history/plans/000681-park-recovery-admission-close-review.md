# Boundary Review — pair#681 (whole-issue close)

| field | value |
|-------|-------|
| issue | 681 — Recover park inventory exceeds CLI admission deadline |
| repo | pair |
| issue file | workshop/issues/000681-park-recovery-admission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 455737dcd073ae5a8610f9b9e7483203c7708f07..7a5a81e82ff6a71893d6380211b8bd60a38f37dc |
| command | sdlc close --issue 681 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T18:47:16-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation passes focused behavioral and race checks. Receipt admission precedes slow preparation, and exact transaction checks remain in place. One Core concepts table contradiction blocks SHIP under this review’s explicit traceability rule; no executable correctness defect was found.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-location-status
    title: |
      Core concepts table claims an extension in an unchanged file
    detail: |
      workshop/plans/000681-park-recovery-admission-plan.md:45 lists the stateful service/console fixture in cmd/internal/couchcmd/slot_operations_acceptance_test.go as extended, but that file is unchanged in the pinned range. The new fixture is newParkAdmissionRig in cmd/internal/couchcmd/park_recovery_admission_test.go:70. This is the sole location/status contradiction found across the table rows. Correct the row to identify the new fixture and add a Revisions entry recording the implementation location.
```

1. **Strengths**

   - Cached selection reuses `SelectSlotRow` and `ActorOperationArgs`, preserving shared selection and identity rules.
   - The real Unix-transport regression holds preparation beyond two seconds while verifying readable receipts and exactly one recovery.
   - Replacement tests exercise both preparation and dispatch boundaries; cancellation tests verify joined work and no late mutation.
   - Retained receipt/conflict handling avoids fresh catalog IO.

2. **Critical findings**

   - **Plan:45 — incorrect fixture location/status.** The table names an unchanged acceptance-test file instead of the newly added fixture. Update the row and record the correction under `## Revisions`. Severity follows the required Core concepts contradiction rule.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Core selection/recovery tests: `4.007s`.
   - Command recovery/slot-operation tests: `5.048s`.
   - Focused race tests: `6.327s`.
   - Pinned-range `git diff --check`.

   Command tests required `TMPDIR=/tmp`; the initial run exceeded the existing macOS Unix-socket path limit during fixture setup. The recorded full suite was not independently run in this review.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Existing selection, argument mapping, receipt transitions and queue ownership are reused.
   - **ARCH-PURE: pass.** Cached selection remains deterministic; IO stays in the console adapter and queued preparation.
   - **ARCH-PURPOSE: pass.** The transport regression demonstrates the intended admission fix while retaining exact nonce protection.

   Atlas documents the changed flow. No new command, flag, configuration key or wire format requires a README surface entry.

7. **Plan revision recommendation**

   Add a `## Revisions` entry stating that production-transport coverage uses the new `newParkAdmissionRig` in `park_recovery_admission_test.go`, and correct the Core concepts row accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c6c72929-1ec8-404f-b79f-16fa67aa6a2f","repository":"github.com/xianxu/pair","issue_id":"000681","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/de29162d72cd3890","reviewed_head":"7a5a81e82ff6a71893d6380211b8bd60a38f37dc","slot":"pair:5","requested_at":"2026-10-19T01:45:11.520515Z"},"phase":"finished","pid":67411,"started_at":"2026-10-19T01:45:11.932634Z","finished_at":"2026-10-19T01:47:00.534249Z","session_id":"01a151d5-8e10-7cb2-8dbb-8e13145a2b17","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.059078333s
head: "7a5a81e82ff6a71893d6380211b8bd60a38f37dc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T18:51:02-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 681 — Recover park inventory exceeds CLI admission deadline |
| repo | pair |
| issue file | workshop/issues/000681-park-recovery-admission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 455737dcd073ae5a8610f9b9e7483203c7708f07..bd39d8c6b9760ae977e866c3c26eb8604fce30b6 |
| command | sdlc close --issue 681 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T18:51:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is addressed, and the focused tests pass. However, the production hub admission path still performs workspace discovery before allocating a receipt. Enabling that wiring in the new fixture reproduces the two-second timeout; the unblocked control succeeds. This blocks SHIP.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The pinned correction changes plan line 45 from an extended slot_operations_acceptance_test.go fixture to newParkAdmissionRig in the new park_recovery_admission_test.go, where the fixture exists at line 70. The Revisions entry at line 139 records that implementation choice. This is a prose-only correction.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: receipt-admission-before-workspace-io
    title: |
      Production hub authorization still performs expensive target discovery before receipt admission
    detail: |
      cmd/internal/couchcmd/park_recovery_admission_test.go:175 initializes the service without authority.hubs or the production hubTargetResolver, bypassing the pre-admission target authorization path. Production handleHubSlotOperation calls guard before slotOps.handle (hub_guards.go:130-133); that guard invokes WorkspaceReferencePath, repository Discover, and ResolveWorkspace through hubTargetResolver (lines 45-66). A temporary Go overlay enabling guardHubState(t) and hubTargetResolver(r.couch) on the existing fixture passes with its catalog barrier released, but fails with the barrier held: socket recover-park: context deadline exceeded after 2.16s. ARCH-PURPOSE: the documented zero-workspace-probe admission contract is therefore not delivered for configured hubs. The family includes new recover-park admissions and retained duplicate/conflicting-ID requests, all of which traverse this guard before receipt lookup; its workspace IO sites are caller workspace resolution and target resolution/discovery. Preserve caller authentication, derive admission target ownership from the bound cached identity, and retain full queued/dispatch authorization. Extend the transport regression with service-side hub authority enabled and cover retained requests before fresh target lookup.
```

1. **Strengths**

   - Cached selection reuses `SelectSlotRow` and `ActorOperationArgs`.
   - Replacement tests exercise both preparation and dispatch boundaries.
   - Cancellation tests verify joined work and no late mutation.
   - The corrected Core concepts table matches the implementation.

2. **Critical findings**

   - **Production admission still blocks on workspace IO.** See the finding above. The fixture enables dispatch-side hub wiring at line 127, but omits service-side hub authority at line 175, masking this path.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed core selection/recovery tests, command recovery/slot-operation tests, focused race tests, and pinned-range `git diff --check`. Command tests required `TMPDIR=/tmp` for Unix socket path length.

   The additional production-wiring overlay passed its unblocked control and failed its blocked case. Temporary overlay files were removed; the checkout remains clean. The recorded full suite was not independently run.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Existing selectors, argument mapping, receipts, and queue ownership are reused.
   - **ARCH-PURE: pass.** Selection remains deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: flag.** Configured-hub admission retains the blocking IO path described above.

   Atlas documents the intended flow. No new command syntax, flag, or configuration key requires a README entry.

7. **Plan revision recommendations**

   Add a `## Revisions` entry covering cached admission ownership checks versus queued current-world authorization. Update the production-transport checklist to require service-side hub authority and its real target resolver, then record the regression evidence after correction.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7a225fee-5546-491f-998a-0a249cf5343e","repository":"github.com/xianxu/pair","issue_id":"000681","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c5e9c6c4ede0f8b2","reviewed_head":"bd39d8c6b9760ae977e866c3c26eb8604fce30b6","slot":"pair:5","requested_at":"2026-10-19T01:48:33.040226Z"},"phase":"finished","pid":1158,"started_at":"2026-10-19T01:48:33.231446Z","finished_at":"2026-10-19T01:51:01.34778Z","session_id":"01a151d8-a003-7713-b927-89475596e5b0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.924799875s
head: "bd39d8c6b9760ae977e866c3c26eb8604fce30b6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T18:58:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 681 — Recover park inventory exceeds CLI admission deadline |
| repo | pair |
| issue file | workshop/issues/000681-park-recovery-admission.md |
| boundary | whole-issue close |
| milestone | — |
| window | 455737dcd073ae5a8610f9b9e7483203c7708f07..13bab42b594631c392438fcb89aac5837d3dece8 |
| command | sdlc close --issue 681 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T18:58:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation fixes receipt admission for configured hubs while preserving exact transaction binding and queued authorization. BR-2 is independently verified by a regression that fails with the old guard restored. No new findings. Required recorded-suite success and the documented deferred live acceptance remain separate obligations.

Done-when: current — it still covers prompt receipts, duplicate handling, replacement refusal, inspectable preparation failures, cancellation ownership, and final verification gates.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Production-wired transport enables service hub authority and hubTargetResolver. Restoring the pre-admission full guard through a temporary Go overlay makes TestRecoverParkAdmissionPrecedesSlowQueuedPreparation fail with context deadline exceeded after 2.27s; pinned code passes. Retained duplicate/conflict tests also verify no fresh caller/target workspace probes. Queued and dispatch authorization remain enforced.
  - id: BR-1
    disposition: addressed
    note: |
      The Core concepts row at workshop/plans/000681-park-recovery-admission-plan.md:45 correctly identifies newParkAdmissionRig in the newly added park_recovery_admission_test.go. The pinned diff and fixture definition support the corrected location/status.
```

1. **Strengths**

   - Cached selection reuses `SelectSlotRow` and captures immutable recovery arguments (`cmd/internal/couchcore/slot_operation.go:77`).
   - Admission ownership uses the verified binding cache while retaining live caller authentication (`cmd/internal/couchcmd/hub_guards.go:72`).
   - Transport tests cover nonce replacement, caller replacement, and ownership movement at preparation and dispatch boundaries.
   - Retained requests bypass unavailable target discovery (`cmd/internal/couchcmd/park_recovery_hub_admission_test.go:145`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Focused core selection/recovery tests: 3.489s.
   - Focused command recovery/hub/slot-operation tests: 6.024s.
   - Focused command race tests: 9.338s.
   - `git diff --check`.

   Restoring the old admission guard produced the expected deadline failure. Temporary overlay files were removed; the repository remains unchanged. Transport tests required `TMPDIR=/tmp` because the default macOS temporary path exceeded the Unix socket length limit. The full required suite was not run in this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared selection, confirmation, receipt transitions, and authorization policies are reused.
   - **ARCH-PURE — pass:** selection and identity comparison remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE — pass:** fresh and retained recovery admissions avoid workspace discovery before receipt lookup/allocation.
   - **ARCH-MOCK — pass:** production transport and queue share stateful external seams with private fixture storage.
   - **ARCH-CONSTRAINTS — pass:** the two-second timeout remains unchanged; controlled barriers and 8/16-row tests exercise admission and scan growth.
   - **ARCH-SECURE — pass:** cached observations select identity; live checks and nonce validation authorize effects.
   - **ARCH-ORDER — pass:** existing receipt transitions and queue ownership remain authoritative; reproducible barriers exercise intervening changes and cancellation.
   - **ARCH-FUNERAL — pass:** no new durable artifact family; existing receipt caps, retention, and queue shutdown remain applicable.

   Atlas documents the changed flow. README already documents the command; no new command, flag, configuration key, or installation step is introduced.

7. **Plan revision recommendations:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7f7abc49-1682-472d-97fd-37e2964ce29a","repository":"github.com/xianxu/pair","issue_id":"000681","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7204542613547955","reviewed_head":"13bab42b594631c392438fcb89aac5837d3dece8","slot":"pair:5","requested_at":"2026-10-19T01:56:48.941623Z"},"phase":"finished","pid":33243,"started_at":"2026-10-19T01:56:49.062601Z","finished_at":"2026-10-19T01:58:39.260355Z","session_id":"01a151e0-3069-7052-a3f2-c7fd01c226c1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.697567875s
head: "13bab42b594631c392438fcb89aac5837d3dece8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
