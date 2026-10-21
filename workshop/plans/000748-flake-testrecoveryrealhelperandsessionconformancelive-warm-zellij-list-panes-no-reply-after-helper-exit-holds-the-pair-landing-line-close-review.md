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
