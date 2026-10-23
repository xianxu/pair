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
