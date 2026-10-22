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

---

## Re-review — 2026-10-22T10:33:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 797 — Live reboot refuses owned pane as ambiguous |
| repo | pair |
| issue file | workshop/issues/000797-live-reboot-pane-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dd027f15ac0f24f76b2c1188043642f71822f97..e93447cbaedb8696d93135ba7ae3a28c444efa5e |
| command | sdlc close --issue 797 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T10:33:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The ownership correction preserves strict validation and exact server authorization, and focused regressions pass. BR-1 is corrected. BR-2 remains open: native Claude acceptance passed, but Codex confirmation failed on this pinned head before retirement. No new production defect is inferred from that refusal.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The pinned before/after diff separates PURE RecoveryTerminalPlan at plan:40 from INTEGRATION recoveryTerminalDriver.checkConfirmed at plan:48 and records the correction under Revisions at plan:135. This matches recovery_terminal_confirmed.go:95-99 and :278-291, where the validator reads readiness metadata through its injected reader. This is a prose-only correction.
  - id: BR-2
    disposition: not-addressed
    note: |
      Verified the retained diagnostic log's SHA256 bd819f74b2424feaa2cf0886e4571d5a4581732bc7b9f72afd547fa3f9155b73: it records successful Codex confirmation, replacement delivery and cleanup on 003e3ba8 with the diagnostic overlay. That measurement is valid, but does not explain the original startup disconnect. On pinned head e93447cb, PAIR_LIVE_RECOVERY_OWNER=1 go test ./cmd/internal/couchcmd -run '^TestLiveRecoveryOwnerReboot/codex$' -count=1 -timeout=5m failed at recovery_owner_live_test.go:112 after 30.706s: confirmation returned “recovery screen changed; refresh resume candidates,” nonce operation-9c891ab72f318d49ea9c781291ee0682. The old generation remained alive; cleanup confirmed all owned generations absent. Claude passed separately in 33.733s. The acceptance family now contains two observed Codex failure modes: replacement startup disconnect and pre-retirement capture-change refusal. Provide a diagnosis or measured environmental explanation and reproducible pinned-head acceptance without weakening either guard; another unexplained passing retry does not resolve the family.
```

1. **Strengths**

   - `recovery_slot_tree.go:103` obtains pane ownership proof before its added wrapper signals destroy that evidence.
   - `artifactcollision.go:183` shares exact-proof deletion across `Quiesce`, `QuiesceNamed` and `TriggerBoundQuit`, refusing unsupported deleters.
   - Replacement, PID reuse, cancellation and invalid-proof tests assert meaningful refusal behavior.
   - Captured native replies preserve omitted command fields and remain rejected by the strict classifier.

2. **Critical findings**

   BR-2 remains blocking at `recovery_owner_live_test.go:112`. Trace the capture identity change through `recovery_evidence_os.go:120`; establish reliable acceptance while preserving the safety check.

3. **Important findings**

   None new. The atlas covers the internal handoff. No new user command, flag or configuration key requires a README update.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused ownership, causal recovery, evidence-rendering, normal-park and confirmed-retirement tests passed. The production-decoder fixture catalog passed. Native Claude passed; native Codex failed as recorded above. The full repository suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared deletion helper and existing authorization loop.
   - **ARCH-PURE — pass:** corrected classification; pure decisions remain separate from injected IO.
   - **ARCH-PURPOSE — flag:** both-personality acceptance remains unresolved under BR-2.
   - **ARCH-EVOLVE — pass:** no persisted format or existing field-meaning changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording this pinned-head capture-change refusal and its eventual diagnosis. Keep acceptance incomplete until the evidence resolves BR-2.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"05e71f80-8499-4887-9039-1a96976d9129","repository":"github.com/xianxu/pair","issue_id":"000797","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c4d0865ba80089b9","reviewed_head":"e93447cbaedb8696d93135ba7ae3a28c444efa5e","slot":"pair:2","requested_at":"2026-10-22T17:31:30.605286Z"},"phase":"finished","pid":95268,"started_at":"2026-10-22T17:31:30.719444Z","finished_at":"2026-10-22T17:33:53.29451Z","session_id":"01a164ab-01aa-7a20-8100-75d8c280ded4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.231818625s
head: "e93447cbaedb8696d93135ba7ae3a28c444efa5e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T10:43:19-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 797 — Live reboot refuses owned pane as ambiguous |
| repo | pair |
| issue file | workshop/issues/000797-live-reboot-pane-owner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dd027f15ac0f24f76b2c1188043642f71822f97..716e5f77d63ecceb0121a3dd796f203722959e7b |
| command | sdlc close --issue 797 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T10:43:19-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The correction preserves ownership evidence until quiescence and carries the exact observed server generation into deletion. Both native personalities passed on pinned head `716e5f77`, including replacement delivery and complete cleanup. No new blocking findings remain. The earlier startup disconnect remains unexplained; this review confirms successful acceptance without claiming its root cause is resolved.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan lines 40 and 48 correctly separate immutable RecoveryTerminalPlan data from INTEGRATION checkConfirmed; the revision at line 137 matches the readiness-file IO in recovery_terminal_confirmed.go:278-291. This is a prose-only correction.
  - id: BR-2
    disposition: addressed
    note: |
      On pinned head 716e5f77, the exact Codex acceptance command passed in 51.871s: confirmation succeeded, old generation 70179/1792690846.889015 disappeared, replacement 80421/1792690861.235185 accepted delivery, and cleanup confirmed all owned generations absent with child and IO joined. Claude independently passed in 58.382s. Verified all three retained transcript hashes against the tracker. The round-2 capture refusal matches the unchanged identity-comparison guard at recovery_evidence_os.go:104-125; its unchanged/changed-file regression passed. The original replacement-start disconnect remains explicitly unclassified, rather than represented as fixed.
```

1. **Strengths**

   - Ownership proof precedes added wrapper signals at `recovery_slot_tree.go:103`.
   - All three scoped deletion consumers share `quiesceOwned`, with explicit refusal for unsupported deleters.
   - `lifecycle_os.go:21` validates the proof and reuses immutable authorized quiescence.
   - Captured native replies preserve missing-command evidence without weakening classification.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused ownership, recovery, diagnostic, capture-change and park tests passed. `make test-affected-inventory` passed.

   Temporary overlays demonstrated meaningful failures without the correction: the old ordering reproduced stopped-pane ownership refusal; all six scoped handoff cases failed; restoring name-based re-admission failed replacement, PID-reuse and socket-change cases. Overlays were removed; the checkout remains clean. The full repository suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared helper covers indexed quiescence, named quiescence and bound quit.
   - **ARCH-PURE — pass:** classification remains IO-free; effects use injected integration seams. Corrected concept classifications match their implementations.
   - **ARCH-PURPOSE — pass:** both public reboot personalities completed confirmation, replacement delivery and cleanup; negative ownership cases remain refused.
   - **ARCH-EVOLVE — pass:** no production persisted-format or field-meaning change.
   
   Atlas documents the new internal flow. No new command, flag, keybinding or configuration surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"b75a6003-8cc1-4863-b4db-d428524b6f1a","repository":"github.com/xianxu/pair","issue_id":"000797","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4c28c2ca579e7e8c","reviewed_head":"716e5f77d63ecceb0121a3dd796f203722959e7b","slot":"pair:2","requested_at":"2026-10-22T17:39:38.603261Z"},"phase":"finished","pid":24429,"started_at":"2026-10-22T17:39:38.779302Z","finished_at":"2026-10-22T17:43:17.774218Z","session_id":"01a164b2-744f-7bc1-a88b-e501d4ad5555","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.97433275s
head: "716e5f77d63ecceb0121a3dd796f203722959e7b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
