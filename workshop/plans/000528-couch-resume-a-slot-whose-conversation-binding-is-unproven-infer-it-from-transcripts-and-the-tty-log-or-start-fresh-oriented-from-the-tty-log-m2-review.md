# Boundary Review — pair#528 (milestone M2)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 525444d45a521bde3dca685d02e6905d2d9bc2e0..332e91c16f697c2071b62825dc7db56ee80cb6c6 |
| command | sdlc milestone-close --issue 528 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T13:26:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation provides explicit candidate selection, bounded evidence collection, durable recovery intent, and original-screen preservation. Two blocking defects remain: native ownership is inferred from the latest tag-level launch rather than the observed wrapper’s generation, and the primary Relaunch gesture bypasses the new recovery preview. Both were reproduced with failing regression tests against the pinned head.

1. **Strengths**

   - Candidate keys bind explicit selections to evidence snapshots; even a perfect match cannot enable automatic recovery.
   - Recovery context survives registration, uncertain outcomes, and rollback. Focused core tests passed.
   - Capture preservation reuses retention leases and the existing publisher, with identity and byte-comparison checks.
   - README and atlas describe the new CLI surface and explicitly disabled automatic inference. The reconciled core-concepts table identifies the implemented owners.

2. **Critical findings**

   - **Wrong launch attributed to a live wrapper** — `cmd/internal/launcher/recovery_native_owner.go:85`. The guard reads `CurrentLaunch` for the wrapper’s scope/tag/agent without linking that launch to its PID/start identity. A newer launch can therefore conceal an older wrapper still using the selected conversation. Bind ownership evidence to the exact wrapper generation; unresolved association must refuse. **ARCH-SECURE, ARCH-ORDER.**
   - **Primary Relaunch bypasses recovery** — `cmd/internal/couchtty/menu.go:864`, `menu_switchagent.go:204`. Recovery is appended as a separate action, while Enter/mouse and the existing Relaunch item retain the old confirmation/executor route (`menu.go:803`, `menu_slot.go:34`). That route still requires an established native binding, so the motivating unproven-binding case cannot recover through its advertised primary action. Route unsafe-row primary actions through the shared preview. **ARCH-PURPOSE.**

3. **Important findings**

   None separately raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused M2 core tests and selected launcher, sessioninventory, threadrecord, couchmessage, orientation, and scrollback tests passed.
   - Added temporary regression `TestReviewRecoveryNativeOwnerDoesNotSubstituteLatestLaunch`: keeping the old wrapper alive while appending a newer same-owner launch incorrectly permits recovery.
   - Added temporary regression `TestReviewNeedsRelaunchPrimaryOpensRecovery`: the primary gesture returns the old confirmation frame with no preview effect.
   - UI/CLI tests initially could not compile because generated runtime assets were absent. Generating them in a disposable pinned checkout resolved compilation; selected existing tests passed.
   - The broader filter failed `TestProvisionRecoveryMarkerPublicationExcludesCleanup` on a missing checkout directory. Full-suite success is not established. OS process conformance was unavailable because `ps` was denied.
   - Temporary files were removed; the original checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing publication, retention, replay, and start-transition owners are reused.
   - **ARCH-PURE — pass:** ranking, selectors, and record validation remain directly testable pure logic.
   - **ARCH-PURPOSE — flag:** primary recovery routing remains incomplete.
   - **ARCH-MOCK — flag:** ownership fixtures do not model an old live wrapper alongside newer same-owner launch evidence.
   - **ARCH-CONSTRAINTS — pass for inspected bounds:** scans enforce deadlines and metadata/root/read limits; representative full-system performance remains unverified.
   - **ARCH-SECURE — flag:** persisted launch evidence lacks exact process-generation attribution.
   - **ARCH-ORDER — flag:** stable wrapper observations do not establish which launch that wrapper owns.
   - **ARCH-FUNERAL — pass:** recovery context is bounded per record, and archived captures retain existing expiry and cleanup ownership.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining exact wrapper-to-launch attribution across both recovery gates and their doubles, and enumerating primary UI entry points—Enter, mouse, and Relaunch—that must invoke the shared recovery preview. Add both regression classes to boundary acceptance.

```findings
findings:
  - id: new
    severity: Critical
    family: exact-process-generation-authority
    title: |
      Native ownership substitutes the latest tag-level launch for the live wrapper's launch
    detail: |
      cmd/internal/launcher/recovery_native_owner.go:85 uses CurrentLaunch(scope, tag, agent) without associating that launch with the observed wrapper PID/start identity. A regression retaining the old wrapper while appending a newer launch targeting another conversation incorrectly returns success. This is the 2nd finding in family `exact-process-generation-authority`. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that every native-ownership assertion must bind to the exact observed process generation; sweep preflight, Pair admission, source exemptions, and their doubles. Missing association must remain unresolved. ARCH-SECURE, ARCH-ORDER, ARCH-MOCK.
  - id: new
    severity: Critical
    family: recovery-primary-action-routing
    title: |
      The advertised primary Relaunch action never opens the new recovery preview
    detail: |
      cmd/internal/couchtty/menu.go:864 handles only the new resume-candidates item, appended separately at menu_switchagent.go:204. Unsafe-row Enter/mouse still select relaunch at menu.go:803 and open the old confirmation through menu_slot.go:34; the old executor requires established binding. The primary-route regression fails with the confirmation frame and no preview effect. Route Enter, mouse, and the unsafe-row Relaunch item through the same evidence preview while preserving ordinary healthy relaunch behavior. ARCH-PURPOSE.
```

---

## Re-review — 2026-10-13T13:39:33-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 525444d45a521bde3dca685d02e6905d2d9bc2e0..5be717b9e09eef04e2223e8a4ae36ed86a3888b7 |
| command | sdlc milestone-close --issue 528 --milestone M2 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T13:39:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

Both prior Critical findings are addressed, with independently confirmed failing mutations. The implementation follows the revised explicit-selection design, preserves recovery evidence, and documents the new commands. One Important architecture gap remains: live conformance does not exercise the new recovery-specific native contract. Full-suite and operator acceptance remain outstanding under the documented boundary exception.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Ownership now resolves the launch ordinal associated with the observed wrapper's live readiness generation. Checked preflight, final Couch check, Pair admission, source exemptions, and the stateful double. TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch passes at head and fails when CurrentLaunch receives the entire tag ledger again.
  - id: BR-7
    disposition: addressed
    note: |
      Unsafe Enter, mouse, Relaunch item, and restart shortcut share the recovery preview route. Production reducer tests pass; disabling the shared routing predicate fails all four cases. Healthy relaunch retains its existing confirmation.
findings:
  - id: new
    severity: Important
    family: external-contract-conformance
    title: |
      Recovery-specific native parsing is outside live conformance
    detail: |
      ARCH-MOCK: cmd/internal/sessioninventory/recovery_candidate.go:192 introduces eligibility based on Claude entrypoint and related provenance, but conformance.go:52 checks only the existing scanners; conformance_live_test.go:21 never invokes the recovery parser or collector. Native changes can therefore leave conformance green while recovery eligibility or tail evidence stops working. Extend the existing read-only conformance path through the production recovery parser, covering provenance, cwd, activity and authored tails, with explicit unsupported/absent-sample outcomes and the existing contract-change/monthly cadence.
```

**1. Strengths**

- Exact-generation ownership is enforced before source exemptions and rechecked after observation (`recovery_native_owner.go:67–91`).
- Primary recovery routes now share one predicate and preview; tests cover all four entry routes and healthy relaunch.
- Recovery intent survives completion and rollback through the existing start transaction machinery.
- Bounded file reads, capture preservation, strict schema migration, and explicit candidate keys have substantive tests. README and atlas cover the new user-facing commands.

**2. Critical findings**

None remaining.

**3. Important findings**

The ARCH-MOCK conformance gap above. Extend `TestLiveNativeSessionShapeConformance` to exercise the new production recovery interpretation, and add a regression demonstrating that recovery-specific contract drift is detected.

**4. Minor findings**

None.

**5. Test coverage notes**

- Passed targeted recovery tests across launcher, couchcore, couchtty, couchcmd, sessioninventory, threadrecord, orientation, and couchmessage.
- Passed full sessioninventory, threadrecord, orientation, scrollbackcmd, and artifactpath suites.
- Independently killed both prior-defect mutations.
- UI/CLI compilation and artifact tests initially encountered absent generated runtime assets. They passed after generating those assets in disposable pinned-head copies; those copies were removed.
- No full repository suite, live acceptance, or independent race rerun was completed.

**6. Architectural notes**

| Marker | Result | Evidence |
|---|---|---|
| ARCH-DRY | Pass | Shared preview, bounded reader, capture publisher, and lifecycle owners. |
| ARCH-PURE | Pass | Parsing, ranking, keys, schema validation, and orientation have direct value tests. |
| ARCH-PURPOSE | Pass | Explicit alternatives and fresh fallback follow the latest revisions; primary routing is repaired. |
| ARCH-MOCK | **Flag** | New recovery interpretation lacks live conformance coverage. |
| ARCH-CONSTRAINTS | Pass | Explicit scan, byte, metadata, candidate, and message bounds; cancellation checks. |
| ARCH-SECURE | Pass | Strict persisted-input validation and exact process/artifact revalidation. |
| ARCH-ORDER | Pass | Start-claim transfer, stale-preview rejection, ownership rechecks, and uncertain-delivery handling. |
| ARCH-FUNERAL | Pass | Bounded retained context and existing capture retention; no new growing suggestion store. |

**7. Plan revision recommendations**

Add a `## Revisions` entry naming recovery-specific conformance coverage, its production parser/collector seam, supported-agent expectations, absent-sample behavior, and execution cadence. The reconciled implementation-owner table otherwise matches the inspected code.
