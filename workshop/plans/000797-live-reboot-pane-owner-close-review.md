# Boundary Review — pair#797 (whole-issue close)

| field | value |
|-------|-------|
| issue | 797 — Live reboot refuses owned pane as ambiguous |
| repo | pair |
| issue file | workshop/issues/000797-live-reboot-pane-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dd027f15ac0f24f76b2c1188043642f71822f97..003e3ba8deb1dd28cedb89b58774a1295b7916e6 |
| command | sdlc close --issue 797 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T10:21:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The ownership correction is supported by passing causal regressions, package tests, and native Claude acceptance. Two blockers remain: the plan misclassifies an IO-dependent validator as pure, and native Codex acceptance failed on the pinned head after retirement, during replacement startup. The latter’s cause is unresolved.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-purity-classification
    title: |
      ARCH-PURE: checkConfirmed is incorrectly classified as pure
    detail: |
      workshop/plans/000797-live-reboot-pane-owner-plan.md:40 lists RecoveryTerminalPlan / checkConfirmed under Pure decisions. checkConfirmed reads readiness files at recovery_terminal_confirmed.go:278 and :291; readRecoveryReady defaults to osfs.ReadRegularMetadata, and confirmedRecoveryFixture injects d.read. This is the sole instance of this classification mismatch in the window. Split the pure plan data from the integration validator and record the correction under ## Revisions; the checklist explicitly makes this contradiction Critical.
  - id: new
    severity: Critical
    kind: unverifiable
    family: boundary-acceptance-evidence
    title: |
      ARCH-PURPOSE: required native Codex reboot acceptance failed
    detail: |
      PAIR_LIVE_RECOVERY_OWNER=1 go test ./cmd/internal/couchcmd -run '^TestLiveRecoveryOwnerReboot/codex$' -count=1 -timeout=5m failed at recovery_owner_live_test.go:112 on head 003e3ba8. The old agent exited, but confirmation returned failed: Pair exited before registering (exit 1); retained output reports Lost connection to the Zellij server and terminal client exited before admission was verified. Cleanup confirmed all owned generations absent. This is the sole failed personality in the two-personality acceptance family; Claude passed. Diagnose this failure and provide measured pinned-head evidence establishing successful Codex confirmation, replacement delivery, and cleanup, or evidence explaining the environmental failure; package tests and prior logged passes do not resolve this observation.
```

1. **Strengths**

   - `recovery_slot_tree.go:103` preserves pane evidence by quiescing before added wrapper signals.
   - `artifactcollision.go:183` consolidates all three scoped deletion consumers and refuses unsupported name-only deleters.
   - Replacement, PID reuse, cancellation, invalid proof, and unknown-owner tests exercise meaningful refusal paths.
   - Captured native replies preserve omitted command fields and remain rejected by strict classification.

2. **Critical findings**

   Both are detailed above. The classification correction can be documentation-only; the Codex failure needs investigation and acceptance evidence.

3. **Important findings**

   None. The atlas documents the new internal handoff. No new product command, flag, or configuration surface requires a README change.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed all four affected packages: `launcher`, `couchcore`, `testcouch`, and `couchcmd`. Focused regressions and the production-decoder fixture catalog also passed. Native Claude reboot, replacement delivery, and cleanup passed; native Codex failed as recorded above. No retry was used to replace that failure.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared owned-quiescence helper and existing authorization loop.
   - **ARCH-PURE — flag:** plan classification mismatch above.
   - **ARCH-PURPOSE — flag:** both-personality acceptance remains unverified.
   - **ARCH-EVOLVE — pass:** no persisted format or existing field meaning changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry separating PURE `RecoveryTerminalPlan` data from INTEGRATION `checkConfirmed` and its readiness-reader seam. Record the Codex acceptance failure and its eventual diagnosis before declaring verification complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e6e327bd-8e14-40e6-8890-cf3dabb7a88f","repository":"github.com/xianxu/pair","issue_id":"000797","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0dd285368b839e51","reviewed_head":"003e3ba8deb1dd28cedb89b58774a1295b7916e6","slot":"pair:2","requested_at":"2026-10-22T17:11:56.270733Z"},"phase":"finished","pid":56565,"started_at":"2026-10-22T17:11:56.58375Z","finished_at":"2026-10-22T17:21:58.476814Z","session_id":"01a16499-1829-7561-b781-9852d0ae2bbb","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.281907791s
head: "003e3ba8deb1dd28cedb89b58774a1295b7916e6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
