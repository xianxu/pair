# Boundary Review — pair#162 (whole-issue close)

| field | value |
|-------|-------|
| issue | 162 — Enter sends when the composer line is addressed to the harness |
| repo | pair |
| issue file | workshop/issues/000162-enter-sends-harness-addressed-lines.md |
| boundary | whole-issue close |
| milestone | — |
| window | 09c8964de39cdb5e6bd9db20601c232446af6949..8b74232d951f9480e947e88fe8c832cbed063dab |
| command | sdlc close --issue 162 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T07:17:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared grammar, explicit command-wait phase, ownership checks and immutable-frame Return decision are sound approaches. Existing tests pass, and README/atlas updates are present. Two reproduced correctness gaps block shipping: unaccepted suggestions can authorize submission, and valid commands ending in spaces cannot satisfy automated echo proof.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: authored-input-proof
    title: |
      Unaccepted faint suggestions can authorize native submission
    detail: |
      cmd/internal/wrapcmd/peer_composer.go:186 copies faint suggestion cells into command text; composerPlaceholder at line 211 rejects suggestions only at the empty input origin. A temporary regression using an authored "/" followed by faint "model", with the cursor immediately after "/", produced text="/model" and native CR. This violates the Spec's suggestion-negative contract (ARCH-SECURE, ARCH-PURPOSE). Refuse command authority when suggestion cells contribute to the proof, preserve Active, and add mixed-style regressions across the supported profiles.
  - id: new
    severity: Critical
    kind: defect
    family: command-proof-representation
    title: |
      Trailing authored spaces make command echo proof impossible
    detail: |
      cmd/internal/agentdrive/command.go:84 retains "/model " as the expected command, while cmd/internal/wrapcmd/peer_composer.go:191 projects its fully rendered echo as "/model". The strict comparison at cmd/internal/agentdrive/model.go:441 therefore waits until the deadline without emitting Enter. A temporary reducer regression reproduced this for a valid supported text-plus-enter sequence (ARCH-PURPOSE). Make authored targets and observed proof represent trailing whitespace consistently without weakening leading-whitespace, multiline or ownership checks, and add a projector-to-writer regression.
```

1. **Strengths**

   - Command Enter has an explicit reducer phase and remains within the existing deadline.
   - Final command validation and Return translation share one captured frame in `agent_drive.go`.
   - General command evidence remains separate from exact `/model` cleanup authority.
   - The slash-producer sweep, public `submit` compatibility, README and atlas changes agree.

2. **Critical findings**

   - **Suggestion authority:** [peer_composer.go:186](/Volumes/TestScratch/review-checkouts/336b7fa19b26cf4d/cmd/internal/wrapcmd/peer_composer.go:186). Preserve activity while declining text proof containing unaccepted suggestions.
   - **Whitespace representation:** [command.go:84](/Volumes/TestScratch/review-checkouts/336b7fa19b26cf4d/cmd/internal/agentdrive/command.go:84). Reconcile authored and observed trailing whitespace before exact-echo comparison.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full tests passed for `agentdrive`, `wrapcmd`, `couchcmd`, `testcouch`, `repolint` and `artifactpath`. The plan’s focused test command also passed. Both additional adversarial probes failed as described above; they used temporary Go overlays, which were removed. The checkout remains clean.

   Existing suggestion tests cover wholly faint placeholders, missing mixed authored/suggested text. Expectation tests miss the projection mismatch for trailing spaces. Native agents were not launched; committed native captures were replayed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared grammar, geometry and projector.
   - **ARCH-PURE — pass:** deterministic recognition and reducer logic remain separate from IO.
   - **ARCH-PURPOSE — flag:** the two documented input classes above remain incorrect.
   - **ARCH-MOCK — pass:** stateful delayed-echo and writer tests exercise the production seam.
   - **ARCH-CONSTRAINTS — pass:** bounded input, existing deadlines and linear projection remain.
   - **ARCH-SECURE — flag:** suggestion provenance is lost before submission authority is granted.
   - **ARCH-ORDER — pass:** explicit pending-command phase, cancellation and final revalidation.
   - **ARCH-FUNERAL — pass:** new runtime evidence follows existing drive/receipt lifetimes.
   - **ARCH-EVOLVE — pass:** no persisted-format change introduced.

7. **Plan revision recommendations**

   Add `## Revisions` entries requiring mixed authored/faint suggestion regressions and end-to-end trailing-whitespace proof coverage. Retain the accepted scope and existing submission safeguards.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"681d624d-c5ae-49bc-b93c-f5360a7fc527","repository":"github.com/xianxu/pair","issue_id":"000162","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/336b7fa19b26cf4d","reviewed_head":"8b74232d951f9480e947e88fe8c832cbed063dab","slot":"pair:2","requested_at":"2026-10-23T14:11:30.411362Z"},"phase":"finished","pid":90061,"started_at":"2026-10-23T14:11:30.571211Z","finished_at":"2026-10-23T14:16:19.49773Z","session_id":"01a1691a-4239-7462-b8e4-db30d16555f0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.295660583s
head: "8b74232d951f9480e947e88fe8c832cbed063dab"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T07:27:22-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 162 — Enter sends when the composer line is addressed to the harness |
| repo | pair |
| issue file | workshop/issues/000162-enter-sends-harness-addressed-lines.md |
| boundary | whole-issue close |
| milestone | — |
| window | 09c8964de39cdb5e6bd9db20601c232446af6949..6e9bd23145854d1db1c88cd7886824782b6cc2cd |
| command | sdlc close --issue 162 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T07:27:22-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior fixes are addressed and backed by regression tests that fail when their fixes are disabled. The shared grammar, reducer proof and immutable-frame writer are sound approaches. One newly reproduced defect blocks shipping: blank continuation rows can disappear from command proof, allowing a multiline draft to submit.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestSlashReturnDeclinesMixedFaintSuggestion passes across all six profiles and fails when the hasSuggestion refusal is disabled through a temporary compiler overlay.
  - id: BR-2
    disposition: addressed
    note: |
      Target, reducer and projector-to-writer trailing-space regressions pass. TestAgentDriveCommandTrailingSpacesReachNativeReturn fails when authored-target trimming is removed through a temporary compiler overlay.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: command-proof-representation
    title: |
      Blank continuation rows below the cursor are erased before single-line proof
    detail: |
      cmd/internal/wrapcmd/peer_composer.go:197 removes blank trailing rows below the cursor before joining lines. A temporary regression using slashComposerPaint(agent, "/model", "") with the cursor on the first row reproduced known text="/model" and native submission for Claude, Qoder, Muse, Agy and Grok. This violates the multiline-negative contract (ARCH-PURPOSE, ARCH-SECURE). This is the 2nd finding in family command-proof-representation: define and enforce the normalization rule across consumers, rather than fixing this instance. Terminal padding normalization must preserve authored line structure; ambiguous continuation rows must decline command authority. Sweep all six profiles and add cursor-movement and blank/space-only continuation regressions.
```

1. **Strengths**

   - Activity and command-text authority remain separate, preserving newline behavior when text proof is unavailable.
   - Final command matching and Return translation share one immutable frame.
   - Exact `/model` cleanup authority remains separate from general command evidence.
   - README, atlas, CLI help and embedded usage examples cover the changed Enter behavior.

2. **Critical findings**

   The finding above originates at [peer_composer.go:197](/Volumes/TestScratch/review-checkouts/4a0278ed5ef70b85/cmd/internal/wrapcmd/peer_composer.go:197) and reaches submission at [harness_tty.go:196](/Volumes/TestScratch/review-checkouts/4a0278ed5ef70b85/cmd/internal/wrapcmd/harness_tty.go:196). Preserve line-count evidence independently of normalized comparison text.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Full tests passed for `agentdrive`, `wrapcmd`, `couchcmd`, `testcouch`, `repolint` and `artifactpath`. The plan’s focused test command and affected-reader inventory also passed. Both prior-fix mutations failed as expected. The temporary blank-continuation regression failed in five profiles; existing tests cover a blank continuation only while the cursor remains on that continuation. Native agents were not launched during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared grammar, geometry and Return decision.
   - **ARCH-PURE — pass:** deterministic decisions remain separate from IO; listed core entities exist.
   - **ARCH-PURPOSE — flag:** multiline input can acquire submission authority.
   - **ARCH-MOCK — pass:** stateful terminal/writer fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** bounded requests and projection work; existing deadlines retained.
   - **ARCH-SECURE — flag:** lossy normalization converts ambiguous input into positive command evidence.
   - **ARCH-ORDER — pass:** explicit command-wait phase and controllable proof/write ordering.
   - **ARCH-FUNERAL — pass:** new evidence follows existing bounded drive lifetimes.
   - **ARCH-EVOLVE — pass:** no persisted-format change introduced.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining permissible command-proof normalization, preserving line structure independently of cursor position, and requiring a six-profile sweep covering blank and space-only continuation rows before and after cursor movement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f175f5ab-f3f1-4807-8d84-e5d85d214eba","repository":"github.com/xianxu/pair","issue_id":"000162","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4a0278ed5ef70b85","reviewed_head":"6e9bd23145854d1db1c88cd7886824782b6cc2cd","slot":"pair:2","requested_at":"2026-10-23T14:22:24.339415Z"},"phase":"finished","pid":72744,"started_at":"2026-10-23T14:22:24.504881Z","finished_at":"2026-10-23T14:27:21.349403Z","session_id":"01a16924-3cc8-7fd0-9cc5-15bf88ba6738","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.161550333s
head: "6e9bd23145854d1db1c88cd7886824782b6cc2cd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T07:46:23-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 162 — Enter sends when the composer line is addressed to the harness |
| repo | pair |
| issue file | workshop/issues/000162-enter-sends-harness-addressed-lines.md |
| boundary | whole-issue close |
| milestone | — |
| window | 09c8964de39cdb5e6bd9db20601c232446af6949..51f0a6e0cdaf9847904dff59821237bb8bb3e926 |
| command | sdlc close --issue 162 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T07:46:23-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the current Spec and approved Plan. BR-3 is addressed by a shared structural-row rule, with regression coverage across all six profiles. Mutation checks confirm the tests fail without the corrections. No new blocking findings emerged; the documented native-conformance limitations remain explicit.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      peer_composer.go:217–240 preserves every bounded input row and declines footerless Codex slash authority. TestSlashReturnPreservesBlankContinuationAfterCursorMove passes at head and fails in all six profiles when cursor-based trimming is restored; removing the footerless guard also fails its dedicated regression. Disposable mutation checkouts were removed.
  - id: BR-1
    disposition: addressed
    note: |
      Mixed faint suggestion cells remain unknown authored input. Removing the suggestion guard makes TestSlashReturnDeclinesMixedFaintSuggestion fail across all six profiles.
  - id: BR-2
    disposition: addressed
    note: |
      Command expectations validate authored grammar before normalizing trailing ASCII spaces. Removing normalization makes TestCommandExpectationUsesProjectedTrailingSpaceRepresentation fail; reducer and writer regressions pass at head.
```

**1. Strengths**

- The shared projector preserves authored line structure while keeping established peer-padding behavior ([peer_composer.go](/Volumes/TestScratch/review-checkouts/51e89e5e16463423/cmd/internal/wrapcmd/peer_composer.go:217)).
- Final command proof and Return emission consume the same immutable frame, with ownership and completeness checks ([agent_drive.go](/Volumes/TestScratch/review-checkouts/51e89e5e16463423/cmd/internal/wrapcmd/agent_drive.go:395)).
- README, atlas, help, and concrete slash producers consistently describe and use `enter`; public `submit` retains its existing meaning.

**2. Critical findings:** None.

**3. Important findings:** None.

**4. Minor findings:** None.

**5. Test coverage notes**

Passed the plan’s focused test command, complete `agentdrive`, `wrapcmd`, `couchcmd`, `testcouch`, `repolint`, `artifactpath`, and `termcmd` packages, plus affected-reader inventory validation. Native capture replay passed. Mutation checks demonstrated failures for BR-1, BR-2, bounded blank-row trimming, and footerless Codex authority.

Live agents and the opt-in public fleet acceptance test were not rerun. Qoder’s approved trust gap and the recorded public-oracle limitation remain documented.

**6. Architectural notes**

| Principle | Result |
|---|---|
| ARCH-DRY | Pass — shared geometry, projector, grammar, and Return decision. |
| ARCH-PURE | Pass — deterministic recognition and reducer logic; thin writer boundary. |
| ARCH-PURPOSE | Pass — operator Return and concrete automated slash producers are covered. |
| ARCH-MOCK | Pass — stateful drive fixtures, delayed echo, and retained native replay. |
| ARCH-CONSTRAINTS | Pass — bounded requests, existing deadlines, counted projection work. |
| ARCH-SECURE | Pass — ambiguous, incomplete, or suggested input grants no command authority. |
| ARCH-ORDER | Pass — explicit pending-command phase; cancellation, stale proof, and changed ownership tested. |
| ARCH-FUNERAL | Pass — no new runtime artifact family or background worker. |
| ARCH-EVOLVE | Pass — no persisted-format change; public submit compatibility preserved. |

**7. Plan revision recommendations:** None. The BR-3 structural-row revision matches the implementation and verified regressions.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"792d09a5-7ae6-4a22-bcca-2211795709dc","repository":"github.com/xianxu/pair","issue_id":"000162","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/51e89e5e16463423","reviewed_head":"51f0a6e0cdaf9847904dff59821237bb8bb3e926","slot":"pair:2","requested_at":"2026-10-23T14:41:34.466267Z"},"phase":"finished","pid":46716,"started_at":"2026-10-23T14:41:34.626595Z","finished_at":"2026-10-23T14:46:21.766034Z","session_id":"01a16935-c960-79e3-a741-02c26b97cc24","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.2841445s
head: "51f0a6e0cdaf9847904dff59821237bb8bb3e926"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
