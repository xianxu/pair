# Boundary Review — pair#544 (whole-issue close)

| field | value |
|-------|-------|
| issue | 544 — couch: let the hub drive a slot's agent keys (switch model/effort remotely), without message chrome |
| repo | pair |
| issue file | workshop/issues/000544-couch-let-the-hub-drive-a-slot-s-agent-keys-switch-model-effort-remotely-without-message-chrome.md |
| boundary | whole-issue close |
| milestone | — |
| window | e7b676abb2083791028bed45eda22515361cca4c..91b007dca5446eea11e9d073879ea06de3fd5b4c |
| command | sdlc close --issue 544 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T23:41:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation has strong input ownership, cancellation and screen-evidence safeguards, and the targeted tests pass. Two contract gaps block this boundary: unknown-model failures discard the visible choices, and the plan’s Core concepts inventory contradicts the implementation.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: preserve-actionable-failure-evidence
    title: |
      Unknown-model recovery discards the choices required by the Spec
    detail: |
      cmd/internal/agentdrive/model.go:425 backs out with only “requested model not among observed choices.” Subsequent recovery screens overwrite Result.Tail at line 369 and ScreenEvidence at line 385, so the terminal result contains neither the observed labels nor the picker once the empty prompt is restored. Preserve bounded model/effort choices independently of the final tail and return them through receipts and CLI output; add regressions asserting choices survive completed recovery for both agents.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-inventory-fidelity
    title: |
      The Core concepts table names missing entities and an unchanged implementation file
    detail: |
      workshop/plans/000544-agent-keys-model-plan.md:36 declares Recipe in recipe.go, but that entity and the promised per-agent recipe tables do not exist; agent decisions are branches in screen.go and model.go. Line 51 names RemoteEndpoint.Drive, whereas production exposes StartDrive, and line 58 marks peer_live_test.go modified although it is unchanged in the pinned range. Reconcile the required recipe structure and correct the inventory through an explicit Revisions entry. ARCH-PURPOSE: the plan must describe the implementation actually delivered.
```

1. **Strengths**

   - `DriveState` encapsulates transitions, and picker-derived effects carry evidence that the wrapper revalidates before input.
   - Production reuses the single input writer and context-aware terminal transport. Cancellation and partial writes retain uncertainty without replay.
   - Session scope defaults safely; unsupported Claude default scope refuses before input.
   - README, atlas, embedded skill, operation declarations and artifact inventory cover the new surface.

2. **Critical findings**

   - `model.go:425`: retain observed choices through backout, including unsupported-effort failures. The existing `TestDriveUnknownChoiceBackoutAndTailBound` checks recovery and size but never checks that choices reach the caller.
   - Plan lines 36–58: reconcile the recipe representation and repair the entity/path/status inventory. This is Critical under the requested Core concepts cross-check.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed targeted tests across agentdrive, couchmessage, couchcmd, couchcore, wrapcmd and couchtty; focused race tests across the drive packages; complete agentdrive, tailmarkup and artifactpath tests; and `git diff --check`.

   Native live conformance and the full repository suite were not rerun. Tracker claims were not treated as independently verified runtime evidence.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared translator, writer, receipt machinery and markup helper.
   - **ARCH-PURE — pass:** parsing and drive decisions remain IO-free.
   - **ARCH-PURPOSE — flag:** required failure diagnostics and declared recipe structure remain incomplete.
   - **ARCH-MOCK — pass:** stateful socket/input-loop fixtures and recorded/native conformance harnesses exist.
   - **ARCH-CONSTRAINTS — pass:** explicit input, navigation, deadline and retention bounds.
   - **ARCH-SECURE — pass:** typed requests, owning-hub checks and exact wrapper identity checks.
   - **ARCH-ORDER — pass:** encapsulated transitions, fresh-screen checks and controlled cancellation tests.
   - **ARCH-FUNERAL — pass:** bounded, expiring in-memory receipts and owner-scoped cancellation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry reconciling the recipe representation, actual endpoint method names and conformance test locations. Update the Core concepts tables accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"85b36c41-2825-44b5-9754-7ddea5208405","repository":"github.com/xianxu/pair","issue_id":"000544","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a64c86cf67e54c02","reviewed_head":"91b007dca5446eea11e9d073879ea06de3fd5b4c","slot":"pair:6","requested_at":"2026-10-16T06:38:51.106993Z"},"phase":"finished","pid":384,"started_at":"2026-10-16T06:38:51.231561Z","finished_at":"2026-10-16T06:41:23.123992Z","session_id":"01a1436f-52ab-76d0-a6eb-af2f1e446c4c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.848505666s
head: "91b007dca5446eea11e9d073879ea06de3fd5b4c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
