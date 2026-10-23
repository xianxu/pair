# Boundary Review — pair#728 (whole-issue close)

| field | value |
|-------|-------|
| issue | 728 — couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo) |
| repo | pair |
| issue file | workshop/issues/000728-couch-switch-agent-refuses-no-saved-configuration-for-codex-even-after-the-operator-s-switch-form-switch-in-that-repo-echo.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc..6a854ad5a7a774534b7312d85872310e6d42d19c |
| command | sdlc close --issue 728 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T22:04:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The repository preference authority, guarded restart/fallback flow, and documentation largely match the governing revisions. Two issues remain: form preparation can recover persisted state before confirmation, contrary to the read-only contract, and a new console acceptance fixture exhibited uncontrolled startup contention.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      menu_switchagent.go renders borrowed donor, conflict, own and absent provenance; TestSwitchAgentConfigurationProvenanceRequiresExplicitAcceptance passed and asserts those distinctions.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1359 describes the Couch repository document and independent Pair defaults; README.md:1380 describes final-attachment publication. These match repopreferences_store.go, launcher/createflow.go and couchcmd/run.go:1029.
  - id: BR-3
    disposition: addressed
    note: |
      TestSpawnPreparedRefusesDriftByFingerprint and TestNestedRepositoryStorageRemainsIndependentOfOuterSlot now exercise repository preferences and independently passed together.
  - id: BR-4
    disposition: addressed
    note: |
      repopreferences_adoption.go:56 shares the migration/publication deadline. Deadline and competing-writer regressions passed, as did TestRepositoryPublicationKeepsConsoleResponsiveAndRemotePending.
  - id: BR-5
    disposition: addressed
    note: |
      SwitchAgent retains returned source tokens through the final guard. TestSwitchAgentPinsLiveSourceAcrossFinalGuard passed and requires refusal before effects for local/remote ordinary/cooperative reconnection.
  - id: BR-6
    disposition: addressed
    note: |
      cancelSwitchForceConfirmation copies the prepared preview before clearing Force. Input/snapshot preservation and concurrent-render cancellation regressions passed under the race detector.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: read-only-preview-boundary
    title: |
      Switch-form preparation can replay a pending journal before confirmation
    detail: |
      cmd/internal/couchcore/switchagent.go:260 calls GetThread from the common preparation path used by PrepareAgentSwitch, PrepareKnownAgentSwitch and PrepareForcedAgentSwitch. On the production writable store, GetThread reaches withLock, which registers retention and invokes recoverStoreJournalLockedChecked; that applies journal images and removes journal.json before the form is confirmed. This contradicts the explicit no-initialization/no-retention/no-journal-recovery contract at workshop/plans/000728-shared-agent-startup-preferences-plan.md:230 (ARCH-PURPOSE, ARCH-SECURE, ARCH-ORDER). Use PreviewThreadContext for preparation and audit its transitive reads across all three entry points. Add full preparation/form regressions with pending journals in global and routed slot backends, asserting refusal and byte-identical storage through cancellation; the current migration-preview test exercises only previewRepositoryPreferences.
  - id: new
    severity: Important
    kind: defect
    family: test-fixture-initialization-ordering
    title: |
      Edited-restart acceptance setup races the already-running console
    detail: |
      cmd/internal/couchcmd/editedrestart_console_acceptance_test.go:71 starts newColdResumeRig, which starts Console.Run, before editedRestartConsoleSource performs its claim/register/park mutations at lines 30–50. Only the actionable provider is gated. The focused multi-package review run failed TestEditedRestartConsoleRemoteAdoption/codex/resume/attach-refused at line 123 with “lock thread store: resource temporarily unavailable” during fixture construction. The isolated case passed once and across ten repetitions, and a subsequent service selection passed, demonstrating schedule-dependent coverage rather than a stable failure oracle (ARCH-ORDER, ARCH-MOCK). Finish source setup before starting console workers, or explicitly fence every competing store observer; exercise the competing ordering deterministically.
```

1. **Strengths**

   - Repository resolution, migration decisions, and raw-preserving codecs are separated from storage IO.
   - Edited restart requires positive absence and an exact cleanup receipt before fresh fallback; replacement and uncertain evidence remain refusals.
   - Final adoption owns preference publication, with shared warning propagation and no rollback of an adopted child.
   - README and atlas document the new flags, ownership, borrowing, force confirmation, and fallback behavior.

2. **Critical findings**

   - [switchagent.go:260](/Volumes/TestScratch/review-checkouts/6fa35332b55fb2e9/cmd/internal/couchcore/switchagent.go:260): replace the preparation-time recovering getter with the existing preview seam. Audit the complete preparation path, rather than only preference loading.

3. **Important findings**

   - [editedrestart_console_acceptance_test.go:71](/Volumes/TestScratch/review-checkouts/6fa35332b55fb2e9/cmd/internal/couchcmd/editedrestart_console_acceptance_test.go:71): establish fixture state before console startup and control competing observations.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused core selection passed: **184.670s**.
   - Restart/reducer/codec selection passed: **46.583s**.
   - UI cancellation/action tests passed under race detection.
   - Deadline, contention, and preview-journal tests passed.
   - Reader inventory and diff whitespace checks passed.
   - The first multi-package selection failed in the fixture described above; subsequent focused service testing passed.
   - No independent full-suite or live-agent qualification was run. Checkout remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared preference policy and existing launch/cleanup owners.
   - **ARCH-PURE — pass:** pure decisions/codecs have direct tests; IO adapters remain separate.
   - **ARCH-PURPOSE — flag:** complete form-preview purity is not delivered.
   - **ARCH-MOCK — flag:** acceptance setup leaves an uncontrolled real-store interaction.
   - **ARCH-CONSTRAINTS — pass:** explicit document, argument, discovery, and publication bounds.
   - **ARCH-SECURE — flag:** preview crosses into persisted-state mutation without confirmation.
   - **ARCH-ORDER — flag:** preview recovery and fixture startup need explicit ordering.
   - **ARCH-FUNERAL — pass:** bounded replacement document and request-local capabilities/receipts.
   - **ARCH-EVOLVE — pass:** nested unknown fields survive; strict validation and round-trip tests exist.

   The governing concept mappings otherwise match the delivered files; reviewed pure entities require no IO.

7. **Plan revision recommendations**

   Add a `## Revisions` entry naming the read-only preparation seam, all three preparation consumers, and global/routed pending-journal tests. Record the acceptance fixture’s initialization-before-workers invariant and deterministic ordering test.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b6766e61-ff9e-4b18-8101-6212703f30cd","repository":"github.com/xianxu/pair","issue_id":"000728","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6fa35332b55fb2e9","reviewed_head":"6a854ad5a7a774534b7312d85872310e6d42d19c","slot":"pair:3","requested_at":"2026-10-23T04:58:18.497677Z"},"phase":"finished","pid":4617,"started_at":"2026-10-23T04:58:18.694389Z","finished_at":"2026-10-23T05:04:22.104385Z","session_id":"01a1671f-cabd-7ba1-a3ef-ec58e4c94643","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.819738584s
head: "6a854ad5a7a774534b7312d85872310e6d42d19c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
