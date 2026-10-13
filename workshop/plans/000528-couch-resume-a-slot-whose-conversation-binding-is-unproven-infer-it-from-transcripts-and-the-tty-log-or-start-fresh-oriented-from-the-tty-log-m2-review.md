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
