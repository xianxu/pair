# Boundary Review — pair#775 (whole-issue close)

| field | value |
|-------|-------|
| issue | 775 — Standalone Zellij false-miss reproduction |
| repo | pair |
| issue file | workshop/issues/000775-zellij-false-miss-repro.md |
| boundary | whole-issue close |
| milestone | — |
| window | eaa758a6f0cd38837655be5334400098849410ec..7ce32158d781a5f1c71a0f5107aa3ebf0b686414 |
| command | sdlc close --issue 775 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T11:50:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The standalone workload reproduces the exit-zero false miss, preserves raw evidence, and cleans up successfully. SHIP is blocked by signature matching that accepts extra stderr and missing documentation required by the boundary gate.

```findings
findings:
  - id: new
    severity: Critical
    family: exact-diagnostic-contract
    title: |
      Both signature checks accept additional stderr instead of requiring exact replies
    detail: |
      scripts/repro/zellij-false-miss.sh:57 and :60 use grep -Fxq, which matches any complete line rather than the complete stderr stream. These are both instances in this window. Executing the actual worker with the expected diagnostic followed by "unrelated error" produced counts "1 0 0 0" for exit0 and "0 0 1 0" for exit1. This violates the Spec's exact-signature counting contract (ARCH-PURPOSE). Compare the complete stderr bytes and add regressions covering exact positives and extra-line negatives for both signatures, including confirmed and unconfirmed exit1 liveness.
  - id: new
    severity: Important
    family: new-surface-documentation
    title: |
      The standalone reproduction lacks top-level README and atlas updates
    detail: |
      scripts/repro/README.md:3 introduces a runnable diagnostic with query/worker arguments, private-root conventions and named counters, but neither README.md nor atlas/ changes in this range. These are the two missing documentation consumers for this new surface. Add a top-level usage pointer and document the standalone diagnostic and its liveness limits in atlas/zellij-reply-safety.md.
  - id: new
    severity: Minor
    family: evidence-reference-accuracy
    title: |
      The run manifest points to the wrong log for the initial setup failure
    detail: |
      workshop/evidence/000775/run.json:19 says native.log retains the initial setup failure. That file contains the successful measured run; initial-setup-failure.log contains the failure. This is the only incorrect evidence-file reference found in the window. Correct the manifest note.
```

1. **Strengths**

   - Script and log SHA-256 values match the committed manifest.
   - All four query variants run without measured-query retries.
   - Private HOME/XDG/config/socket roots keep the diagnostic independent of Pair and Couch.
   - Raw anomalies remain visible alongside separate confirmed and unconfirmed exit1 counters.

2. **Critical findings**

   - Script lines 57 and 60 need complete-stream diagnostic matching and adversarial regression coverage, as detailed above.

3. **Important findings**

   - Add the README and atlas documentation for the new runnable surface.

4. **Minor findings**

   - Correct the manifest’s initial-failure log reference.

5. **Test coverage notes**

   - `bash -n` passed; tested zero, over-limit, nonnumeric and malformed arguments were refused.
   - Fresh native run: **1,000 reads, four workers; exit0=1, confirmed exit1=0, unconfirmed exit1=0, other nonzero=0**.
   - Cleanup reported pass; a subsequent process search found no matching private server.
   - Actual-worker adversarial checks demonstrated both overly permissive stderr matches. No committed regression tests cover those boundaries.
   - `git diff --check` reports trailing whitespace in retained raw evidence; preserving those bytes is appropriate.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one worker owns the query schedule and classification.
   - **ARCH-PURE — pass:** IO is localized to the diagnostic shell; deterministic signature predicates would improve direct testing.
   - **ARCH-PURPOSE — flag:** exact classification is part of the diagnostic’s purpose and remains incomplete.
   - **ARCH-EVOLVE — pass:** no existing persisted reader or rewrite path changes.

7. **Plan revision recommendations**

   - Add a `## Revisions` entry requiring complete-stream signature matching, deterministic positive/negative coverage for both signatures, and the missing README/atlas updates before marking verification complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bc22fde9-cb5e-47d3-b63b-4c3be918a01c","repository":"github.com/xianxu/pair","issue_id":"000775","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/337e65e19b2af3e2","reviewed_head":"7ce32158d781a5f1c71a0f5107aa3ebf0b686414","slot":"pair:5","requested_at":"2026-10-21T18:48:48.973935Z"},"phase":"finished","pid":38225,"started_at":"2026-10-21T18:48:49.075857Z","finished_at":"2026-10-21T18:50:19.751472Z","session_id":"01a15fcb-6bbd-72d0-8211-f564aefae6cc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.800049708s
head: "7ce32158d781a5f1c71a0f5107aa3ebf0b686414"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
