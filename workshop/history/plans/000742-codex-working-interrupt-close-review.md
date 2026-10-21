# Boundary Review — pair#742 (whole-issue close)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9c926136d44af5bad734eaef7986e327a06a1965..a16907dd04af7aa327515f3778a85ed7459ffb1b |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T13:35:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation passes focused tests and race checks, and the new regressions fail against the base behavior. No executable correctness defect was established. The blocking issue is stale acceptance criteria: the revised Codex contract intentionally overrides cached stop grace, while “Done when” still requires unconditional refusal.

Done-when: stale — the stop-grace refusal clause does not distinguish Claude’s lifecycle requirement from Codex’s current native work proof.

```findings
findings:
  - id: new
    severity: Important
    family: acceptance-criteria-match-contract
    title: |
      Update the stop-grace acceptance criterion for per-agent work proof
    detail: |
      workshop/issues/000742-codex-working-interrupt.md:50 requires stop grace to refuse without a write, but the revised Spec at lines 30–32 and cmd/internal/wrapcmd/agent_drive.go:73–83 authorize current recognized Codex work despite cached stop grace. The sole instance in this window is this unconditional Done-when clause. Qualify refusal by agent and current evidence, and explicitly exercise Codex working evidence during cached grace alongside the existing Claude grace-refusal test.
  - id: new
    severity: Minor
    family: plan-completion-matches-evidence
    title: |
      The completed capture item overstates the native Waiting evidence
    detail: |
      workshop/issues/000742-codex-working-interrupt.md:55 marks native Working and background-wait capture complete, although its Revisions and testdata/drive/codex/0.163.0/metadata.json:8 explicitly identify Waiting fixtures as synthetic. This is the sole completed-plan evidence overstatement in this window. Describe captured Working/running-shell evidence and reported Waiting wording with synthetic fixtures separately.
```

1. **Strengths**

   - Lifecycle observation and interrupt admission share `RecognizeCodexWorking`.
   - Production-output tests exercise the actual mailbox/writer, one Ctrl+C, fresh idle confirmation and second-interrupt refusal.
   - README and atlas explain the changed work proof and confirmation requirement.

2. **Critical findings:** None.

3. **Important findings:** Revise the stop-grace acceptance criterion and cover both agents’ intended grace behavior, as detailed above.

4. **Minor findings:** Correct the completed capture item’s description of synthetic Waiting evidence.

5. **Test coverage notes**

   Focused interrupt/Codex tests passed in both packages, including under `-race`. Temporary Go overlays restoring base behavior made the production-output regressions and unknown-screen confirmation regression fail; the checkout remained unchanged. Recorded local and hub live checks remain pending in the pinned tracker.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both work-proof consumers use the shared recognizer.
   - **ARCH-PURE: pass.** Recognition and reduction remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: pass for implementation.** The change addresses native status recognition, stale lifecycle evidence and positive completion.
   - **ARCH-EVOLVE: pass.** No persisted runtime format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry distinguishing Claude grace refusal from Codex’s current-work override, and update “Done when” accordingly. Correct the capture checklist wording to match the documented native/synthetic evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"45a3559d-2d42-45f0-a44b-bac679c1b3fa","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5369351013e3462e","reviewed_head":"a16907dd04af7aa327515f3778a85ed7459ffb1b","slot":"pair:2","requested_at":"2026-10-20T20:33:24.151072Z"},"phase":"finished","pid":56505,"started_at":"2026-10-20T20:33:24.26066Z","finished_at":"2026-10-20T20:35:43.863355Z","session_id":"01a15b04-d05b-7ce2-ba97-aa5e60a99725","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.95535975s
head: "a16907dd04af7aa327515f3778a85ed7459ffb1b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-20T13:41:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9c926136d44af5bad734eaef7986e327a06a1965..69bd42d1ae642733a110eac1d680b042c7abc146 |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T13:41:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are addressed. The implementation matches the revised per-agent contract, focused tests pass with race detection, and mutation checks demonstrate meaningful regression coverage. No new defects found. The tracker still explicitly leaves the recorded local and hub live checks pending before landing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue line 50 now distinguishes Claude grace refusal from Codex current-work override, matching agent_drive.go:69–83. codex_interrupt_test.go:113–139 exercises cached stop grace through admission and the writer; restoring the cached-working requirement makes its stop_grace subtest fail. Existing Claude grace-refusal coverage passes.
  - id: BR-2
    disposition: addressed
    note: |
      Issue line 55 now separates captured Working/running-shell frames from synthetic Waiting fixtures. The pinned before/after correction agrees with metadata.json:6–9 and the synthetic fixture description in codex_interrupt_test.go:38–39.
```

1. **Strengths**

   - Lifecycle observation and interrupt admission share `RecognizeCodexWorking`.
   - Production-output tests verify exactly one Ctrl+C, fresh idle confirmation, and second-interrupt refusal.
   - Negative cases cover partial frames, pickers, quoted text, startup status, ownership conflicts, and changed evidence before writing.
   - README and atlas document the revised proof and confirmation behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused interrupt and Codex tests passed in both packages, normally and under `-race`. All eight reader-inventory checks passed. Temporary overlays demonstrated failures when restoring the cached-working requirement, base recognizer, or confirmation without positive idle evidence. Temporary files were removed; the checkout remains unchanged.

   Native Waiting behavior remains covered synthetically; the recorded local and hub live checks remain pending in the tracker.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both work-proof consumers derive from the shared recognizer.
   - **ARCH-PURE: pass.** Recognition and reduction remain deterministic; terminal IO stays in adapters.
   - **ARCH-PURPOSE: pass.** The repair covers composer-relative recognition, stale lifecycle evidence, and positive completion.
   - **ARCH-EVOLVE: pass.** No runtime persisted-format changes.

7. **Plan revision recommendations:** None; the existing revisions resolve both prior documentation inconsistencies.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1d9c66f2-8661-438f-bd23-8b0c92ae9b14","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bc570e8dbdff0c85","reviewed_head":"69bd42d1ae642733a110eac1d680b042c7abc146","slot":"pair:2","requested_at":"2026-10-20T20:39:00.987108Z"},"phase":"finished","pid":57766,"started_at":"2026-10-20T20:39:01.189062Z","finished_at":"2026-10-20T20:41:00.999526Z","session_id":"01a15b09-f4dd-7652-b170-9b330ff3f579","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.786300333s
head: "69bd42d1ae642733a110eac1d680b042c7abc146"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T13:59:51-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 65354ae1f7ed9ef6df4a8b909bee29cd87b08e58..3027d14a2cb85212e8a07e65202f416fb8457baa |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T13:59:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior documentation fixes remain correct, and focused tests pass. However, this window introduces a new recognition contract beyond either prior finding. The fix-review rules require a full `sdlc close` before admission.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue line 51 retains the per-agent grace distinction, unchanged from the pinned base and consistent with agent_drive.go:69–83. Retained Codex stop-grace and Claude refusal tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Issue line 56 retains the corrected original-capture description. metadata.json distinguishes the original synthetic Waiting evidence from the newly added operator viewport reconstructions.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: post-close-scope-requires-full-review
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      cmd/internal/wrapcmd/codex_working.go:41–44 removes duration and background-summary validation and broadens Waiting recognition to any "Waiting for " prefix; codex_interrupt_test.go:168–174 explicitly accepts formerly rejected metadata and an additional Waiting form. Issue Revisions lines 87–95 authorize this new executable contract, rather than recording either BR-1 or BR-2's documentation correction. The narrow fix review cannot admit this change; run a full sdlc close covering the revised contract and implementation.
```

1. **Strengths**

   - New operator fixtures exercise production output, admission, exactly one Ctrl+C, and fresh-idle confirmation.
   - Added negatives cover missing hints and verb lookalikes.
   - README and atlas both document the changed recognition surface.

2. **Critical findings**

   - The scope change above requires full close review. Approval recorded in the issue does not expand this review’s permitted scope.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   `go test ./cmd/internal/wrapcmd ./cmd/internal/agentdrive -run 'TestCodex|Test.*Interrupt' -count=1` passed. `git diff --check` passed. No mutation checks were performed; the checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both consumers retain the shared recognizer.
   - **ARCH-PURE: pass.** Recognition remains deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: pass.** The repair targets the reported native Waiting refusal.
   - **ARCH-EVOLVE: pass.** No runtime persisted-format change.

7. **Plan revision recommendations**

   Retain the new stable-marker revision and submit it through full `sdlc close`, with the Spec reconciled to the broadened recognition contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fbfce350-482b-4530-b48b-5eea94d53ce9","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0967515e4150aaf3","reviewed_head":"3027d14a2cb85212e8a07e65202f416fb8457baa","slot":"pair:2","requested_at":"2026-10-20T20:59:09.329269Z"},"phase":"finished","pid":15028,"started_at":"2026-10-20T20:59:09.425581Z","finished_at":"2026-10-20T20:59:50.685766Z","session_id":"01a15b1c-63f2-7821-b066-ad58baae271e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.68244875s
head: "3027d14a2cb85212e8a07e65202f416fb8457baa"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-20T14:02:56-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9c926136d44af5bad734eaef7986e327a06a1965..f684f972951b18ec530dd8acdc012bd6b1031680 |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T14:02:56-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised Spec, and this full-range review resolves BR-3. No new defects found. The recorded suite and hub live acceptance remain explicit requirements before landing; this review does not establish those results.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue line 51 distinguishes Claude grace refusal from Codex current-work override, matching agent_drive.go:69–83. The retained stop-grace test passes; restoring cached-working dependence makes it fail.
  - id: BR-2
    disposition: addressed
    note: |
      Issue line 56 correctly separates original native Working/running-shell captures from synthetic Waiting evidence, consistent with metadata.json:6–9 and syntheticCodexWork's description.
  - id: BR-3
    disposition: addressed
    note: |
      This whole-issue review inspects the full pinned base-to-head range against the reconciled Spec, including broadened status recognition. Restoring the earlier parser makes the operator truncated-Waiting regression fail.
```

1. **Strengths**

   - One recognizer supplies lifecycle and interrupt proof ([codex_working.go:34](/Volumes/TestScratch/review-checkouts/8b3e9deee711237a/cmd/internal/wrapcmd/codex_working.go:34)).
   - Production-output tests exercise admission, exactly one Ctrl+C, fresh-idle confirmation, and second-interrupt refusal.
   - Unknown screens cannot confirm interruption; README and atlas document the revised behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed focused Codex/interrupt tests with race detection, broader drive-related wrapper tests, the full agentdrive package, vet, eight reader-inventory checks, and diff whitespace checks. Temporary overlays demonstrated failures without the status-parser, positive-idle, and stale-grace corrections. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both consumers use the shared recognizer and existing composer authority.
   - **ARCH-PURE: pass.** Recognition and reduction remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: pass.** Captured Waiting failure, stale lifecycle evidence, and completion safety are covered.
   - **ARCH-EVOLVE: pass.** No runtime persisted-format changes.

7. **Plan revision recommendations:** None. The remaining landing checks are accurately left pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0b58a7d3-cd13-4213-b528-dd18073c3c79","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"small-diff-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8b3e9deee711237a","reviewed_head":"f684f972951b18ec530dd8acdc012bd6b1031680","slot":"pair:2","requested_at":"2026-10-20T21:00:53.184972Z"},"phase":"finished","pid":42579,"started_at":"2026-10-20T21:00:53.275079Z","finished_at":"2026-10-20T21:02:55.494578Z","session_id":"01a15b1d-f98e-7212-8791-4b9be3b1909c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.550558792s
head: "f684f972951b18ec530dd8acdc012bd6b1031680"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T22:04:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 742 — Couch interrupt refuses a working Codex turn |
| repo | pair |
| issue file | workshop/issues/000742-codex-working-interrupt.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9c926136d44af5bad734eaef7986e327a06a1965..e767dc64d4e161b2539ae87251d276872b9a7bee |
| command | sdlc close --issue 742 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T22:04:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

No new defects found. The implementation matches the revised Spec, preserves input and ownership fences, and has meaningful regression coverage. The pinned tracker still requires the new recorded suite, CI, and hub live acceptance before landing; this review does not establish those results.

Done-when: current — working-turn admission, refusal safeguards, positive completion, and pre-landing validation still cover the revised redraw-tolerant contract.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue line 51 distinguishes Claude grace refusal from Codex current-work override. Both are tested; restoring cached-working dependence makes the Codex stop-grace regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      Issue line 56 distinguishes the original captured Working/running-shell evidence from synthetic Waiting fixtures, consistent with testdata/drive/codex/0.163.0/metadata.json and the fixture descriptions.
  - id: BR-3
    disposition: addressed
    note: |
      This full whole-issue review covers the pinned base-to-head range and reconciled Spec. Restoring the base recognizer makes the operator truncated-Waiting regression fail.
```

1. **Strengths**

   - Lifecycle and interrupt admission share `RecognizeCodexWorking`.
   - One sequence selector serves observation, start admission, and writer admission.
   - Production-order tests cover redraws, input invalidation, unparsed output, exactly one Ctrl+C, fresh-idle confirmation, and second refusal.
   - README and atlas describe the changed behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed focused tests and race checks across wrapcmd, agentdrive, couchcmd, and couchmessage; broader wrapper/reducer drive tests; vet; all eight reader-inventory checks; and diff whitespace checks.

   Temporary overlays produced expected regression failures for cached-grace dependence, output-sequence fencing, missing input-generation stamps, missing unparsed-output protection, missing positive-idle confirmation, and the base recognizer. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Shared recognizer and sequence selector cover their consumers.
   - **ARCH-PURE: pass.** Recognition and reduction remain deterministic; IO stays in adapters.
   - **ARCH-PURPOSE: pass.** The repair covers recognition, redraw-tolerant rechecks, and safe completion.
   - **ARCH-EVOLVE: pass.** No persisted runtime format changes; sequence remains an opaque token.

7. **Plan revision recommendations:** None. Remaining landing checks are explicitly pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fd40622f-0b21-4e1d-b394-21e0487cd50a","repository":"github.com/xianxu/pair","issue_id":"000742","recipe":"small-diff-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b36bd268d5a271aa","reviewed_head":"e767dc64d4e161b2539ae87251d276872b9a7bee","slot":"pair:2","requested_at":"2026-10-21T05:01:21.806889Z"},"phase":"finished","pid":79235,"started_at":"2026-10-21T05:01:21.956019Z","finished_at":"2026-10-21T05:03:59.425099Z","session_id":"01a15cd5-de2d-73a1-83d8-8aec90206479","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.863449875s
head: "e767dc64d4e161b2539ae87251d276872b9a7bee"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
