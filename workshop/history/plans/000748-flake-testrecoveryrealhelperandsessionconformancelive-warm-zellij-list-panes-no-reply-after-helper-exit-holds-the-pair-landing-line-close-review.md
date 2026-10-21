# Boundary Review — pair#748 (whole-issue close)

| field | value |
|-------|-------|
| issue | 748 — flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line |
| repo | pair |
| issue file | workshop/issues/000748-flake-testrecoveryrealhelperandsessionconformancelive-warm-zellij-list-panes-no-reply-after-helper-exit-holds-the-pair-landing-line.md |
| boundary | whole-issue close |
| milestone | — |
| window | 127001eae6379ce571b6097f4573deb57ca90416..004fe5d455080cc16330700a1e61a02a3324768a |
| command | sdlc close --issue 748 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T19:23:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The production mitigation appears sound: retries are narrowly allowlisted, share one deadline, preserve per-attempt evidence, and retain existing admission checks. Two verification gaps need correction: the no-retry regression misses forbidden subprocesses, and the native diagnostic does not validate its claimed final answers. No production correctness defect was found.

Done-when: current — deterministic red/green reproduction, 20 consecutive warm runs, and the #636 cause record remain applicable to the mitigation authorized under Revisions.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: subprocess-policy-tests-count-every-attempt
    title: |
      The non-retryable-result regression passes when forbidden retries are enabled
    detail: |
      cmd/internal/launcher/zellij_read_recovery_test.go:54 supplies exactly the expected number of frames, but zellij_empty_reply_test.go:96 rejects exhausted frames before updating the invocation counter at line 110. Extra children therefore remain invisible. Verified with a temporary Go overlay: replacing the retry gate with “if err == nil” makes mutations, unrecognized argv and nonzero failures retry, yet TestZellijKnownBugReadBoundAndNonRetryableResults still passes. Enumerated affected cases in that table: nonzero, mutation, unknown-flags, and crossed exhaustion; malformed/invalid-pane replies return nil boundary errors and do not exercise error retry. Count every child before frame validation, or inject an independently counted collector; assert the expected final error and make this unsafe mutant fail.
  - id: new
    severity: Important
    kind: defect
    family: qualification-validates-returned-evidence
    title: |
      The native qualification counts query errors rather than bad final answers
    detail: |
      workshop/evidence/000748/native-production-query-diagnostic.go.txt:13 discards stdout and increments badFinal only when runContext returns an error. That boundary deliberately returns malformed or missing-decision-field same-kind payloads without a parsing error, leaving validation to callers. Consequently this control cannot establish its “0 bad final answers” claim. ARCH-PURPOSE requires completing the planned final-answer qualification. Validate each returned pane reply with the production requirements, retain validation failures, and rerun the control. This family's claims occur in native-production-query-result.txt, workshop/evidence/000748/README.md:7, and the issue's implementation-and-qualification passage at line 101; refresh them together.
  - id: new
    severity: Minor
    kind: defect
    family: comments-track-query-attempt-semantics
    title: |
      Query comments still describe the former one-attempt contract
    detail: |
      In cmd/internal/launcher/zellij.go, line 28 says the boundary never retries, line 149 promises no automatic retries, lines 51–52, 60–61 and 72–74 describe subprocess costs as one run, and lines 168–169 call the observer error caller-visible even when a later attempt succeeds. These are all instances of the changed logical-read versus subprocess-attempt distinction. Describe bounded attempts and attempt-local observer errors consistently.
```

1. **Strengths**

   - The shared timeout surrounds the entire attempt loop; cancellation is checked before subsequent children (`zellij.go:193`).
   - Reply classification reuses existing strict parsers and leaves independent agreement checks intact.
   - Recovery refusal tests retain durable-state and no-effects assertions.
   - Atlas updates document ownership, limits, and the workaround’s removal condition.

2. **Critical findings:** None.

3. **Important findings:** The two verification gaps above. Both have concrete, limited fixes.

4. **Minor findings:** Update the enumerated query comments.

5. **Test coverage notes:** Focused launcher/couchcore/repolint checks and the broader selected launcher tests passed. One live warm recovery passed on Zellij 0.45.1. The recovery regression failed against the base implementation, confirming red-before-green behavior. Temporary overlays were removed; the checkout remains unchanged.

6. **Architectural notes:** ARCH-DRY passes through parser reuse and centralized retry ownership. ARCH-PURE passes with pure classification and an injected collector. ARCH-PURPOSE is flagged for incomplete native answer qualification. ARCH-EVOLVE passes: no persisted-format reader or rewrite behavior changed. Atlas coverage is present; no new user-facing command, flag, or configuration requires a root README update. No Core concepts table was supplied.

7. **Plan revision recommendations:** Add a `## Revisions` entry recording corrected subprocess accounting and strict native final-answer qualification, with refreshed measured results. No production design revision is needed.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7705cc97-8675-4aae-bd13-3d1a95616701","repository":"github.com/xianxu/pair","issue_id":"000748","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/495753bc7a03398e","reviewed_head":"004fe5d455080cc16330700a1e61a02a3324768a","slot":"pair:3","requested_at":"2026-10-21T02:20:04.314444Z"},"phase":"finished","pid":41384,"started_at":"2026-10-21T02:20:04.488621Z","finished_at":"2026-10-21T02:23:20.484678Z","session_id":"01a15c42-3362-77b3-a2c9-0e9bdf1f8cde","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.469332375s
head: "004fe5d455080cc16330700a1e61a02a3324768a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T19:35:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 748 — flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line |
| repo | pair |
| issue file | workshop/issues/000748-flake-testrecoveryrealhelperandsessionconformancelive-warm-zellij-list-panes-no-reply-after-helper-exit-holds-the-pair-landing-line.md |
| boundary | whole-issue close |
| milestone | — |
| window | 127001eae6379ce571b6097f4573deb57ca90416..fb794ec500174a160dd447ac51b4695972c48cb0 |
| command | sdlc close --issue 748 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T19:35:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the revised bounded-mitigation contract, and all three prior findings are addressed. Deterministic regressions, refusal checks, exact park→warm, and 20 consecutive warm runs passed. Native stress results remain mixed: one 2,000-read run reported one bad final answer; a subsequent isolated run reported zero. The revision explicitly permits final refusal, so this limits reliability claims without demonstrating a contract violation.

Done-when: current — deterministic red/green evidence, 20 consecutive warm passes, and the #636 cause follow-through still cover the revised mitigation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      zellij_empty_reply_test.go:98 counts children before frame validation, and the regression asserts counts and final errors. An independently applied unsafe retry mutation fails nonzero, mutation, and unknown-flags cases with three calls instead of one.
  - id: BR-2
    disposition: addressed
    note: |
      native-production-query-diagnostic.go.txt:13 validates returned panes using production requirements and rejects empty inventory. Injected malformed output fails this oracle while the former error-only oracle passes. Retained results and associated claims were refreshed; independent native runs produced both a failure and a pass, as reported here.
  - id: BR-3
    disposition: addressed
    note: |
      The pinned correction updates zellij.go timeout, inventory-cost, ClientCountContext, and observer comments to describe bounded logical reads and attempt-local errors, consistent with runContext:197 and runQueryAttempt:215.
```

1. **Strengths**

   - One deadline surrounds all attempts, and cancellation is checked before another child starts (`cmd/internal/launcher/zellij.go:197`).
   - Exact argv classification and existing strict parsers constrain replay (`cmd/internal/launcher/zellij_read_recovery.go:27`).
   - Recovery tests preserve durable-state and no-effects assertions after exhaustion.
   - Atlas documentation records ownership, bounds, and the workaround’s removal condition.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused launcher, couchcore, and lint checks passed.
   - The recovery regression fails against the base implementation.
   - The unsafe retry mutation fails the corrected regression.
   - The corrected diagnostic rejects malformed successful output; the previous oracle accepts it.
   - Exact park→warm passed; 20 consecutive warm runs passed.
   - Native stress: first run had 2,016 attempts, 17 attempt errors, and one bad final answer; isolated second run had 2,007 attempts, seven attempt errors, and zero bad final answers. These samples do not establish failure-free native operation.
   - Temporary overlays were removed; tracked files remain unchanged.

6. **Architectural notes for upcoming work**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass — centralized attempt ownership and parser reuse. |
   | ARCH-PURE | Pass — pure classification; injected process boundary. |
   | ARCH-PURPOSE | Pass — implements the expressly revised mitigation and qualification controls. |
   | ARCH-MOCK | Pass — production boundary is exercised through stateful fixtures and native controls. |
   | ARCH-CONSTRAINTS | Pass — three sequential attempts, shared deadline, counted 8/16-read bounds. |
   | ARCH-SECURE | Pass — external replies retain strict validation; native fixtures use private roots. |
   | ARCH-ORDER | Pass — bounded sequential attempts, controllable fault schedules, cancellation and exhaustion checks. |
   | ARCH-FUNERAL | Pass — no new persistent runtime artifact; fixture cleanup owns temporary resources. |
   | ARCH-EVOLVE | Pass — no persisted-format reader or rewrite changes. |

   Atlas coverage is present. No new user command, flag, or configuration requires a README update. No Core concepts table was supplied.

7. **Plan revision recommendations:** None required. Preserve the mixed native stress outcomes when reporting qualification; the evidence supports bounded mitigation, not elimination of native failures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"54f24f66-54f2-4de5-9fd6-98d52251edfb","repository":"github.com/xianxu/pair","issue_id":"000748","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b1cf8b85ec2d9523","reviewed_head":"fb794ec500174a160dd447ac51b4695972c48cb0","slot":"pair:3","requested_at":"2026-10-21T02:30:13.228808Z"},"phase":"finished","pid":87844,"started_at":"2026-10-21T02:30:13.459682Z","finished_at":"2026-10-21T02:35:30.758634Z","session_id":"01a15c4b-7e8b-70d3-a4b1-bc9aeba716a4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.029232542s
head: "fb794ec500174a160dd447ac51b4695972c48cb0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T19:51:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 748 — flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line |
| repo | pair |
| issue file | workshop/issues/000748-flake-testrecoveryrealhelperandsessionconformancelive-warm-zellij-list-panes-no-reply-after-helper-exit-holds-the-pair-landing-line.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8e51ba7464c95b79e976f9df040aed35fa2ba27d..8ebb557f6406d40da5512e16a1e320daae225433 |
| command | sdlc close --issue 748 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T19:51:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly adds the missing source-inventory classification. Its existing guard passes at head and fails without the added entry. Qualification records accurately retain native failures without broadening the retry contract. No new defects or scope changes found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Previously disposed correction remains intact: the fake counts every child before schedule validation. This interdiff does not alter that behavior.
  - id: BR-2
    disposition: addressed
    note: |
      The retained diagnostic validates final payloads with production pane requirements. New evidence explicitly records two failed final reads, consistent with the retained events and result.
  - id: BR-3
    disposition: addressed
    note: |
      Query comments retain the corrected distinction between logical reads, bounded attempts, and attempt-local observer errors; this interdiff leaves them unchanged.
```

1. **Strengths**
   - `cmd/internal/artifactpath/manifest.go:1593` correctly classifies the reply-validation helper as a non-artifact source.
   - Native evidence preserves all 14 erroneous attempts and both final failures.
   - Tracker revisions record qualification and the inventory correction while preserving the existing contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - `TestProductionArtifactReferencesAreExactlyClassified` passed.
   - A temporary Go overlay restoring the base manifest made that test fail specifically for the missing helper entry.
   - Evidence checks confirmed 14 erroneous attempts, final failures at logical reads 223 and 510, and 20 retained warm passes.
   - `git diff --check` passed; tracked files remain unchanged. Native controls were not rerun.

6. **Architectural notes**
   - **ARCH-DRY:** Pass — uses the existing authoritative inventory.
   - **ARCH-PURE:** Pass — the classified helper performs byte validation without IO.
   - **ARCH-PURPOSE:** Pass — repairs exhaustive classification without weakening enforcement.
   - **ARCH-EVOLVE:** Pass — no persisted-format reader or rewrite changes.
   - No new architectural or user-facing surface requires atlas or root README updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6f174145-b748-40c2-a25b-57ea131eeb95","repository":"github.com/xianxu/pair","issue_id":"000748","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/35101e2fe2f85718","reviewed_head":"8ebb557f6406d40da5512e16a1e320daae225433","slot":"pair:3","requested_at":"2026-10-21T02:49:51.827743Z"},"phase":"finished","pid":52766,"started_at":"2026-10-21T02:49:52.037059Z","finished_at":"2026-10-21T02:51:04.27938Z","session_id":"01a15c5d-7ad6-7292-a525-b3b7b0883a6c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.890526375s
head: "8ebb557f6406d40da5512e16a1e320daae225433"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
