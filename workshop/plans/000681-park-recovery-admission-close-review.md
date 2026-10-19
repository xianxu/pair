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
