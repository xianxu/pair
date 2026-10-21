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

---

## Re-review — 2026-10-21T12:05:36-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 775 — Standalone Zellij false-miss reproduction |
| repo | pair |
| issue file | workshop/issues/000775-zellij-false-miss-repro.md |
| boundary | whole-issue close |
| milestone | — |
| window | eaa758a6f0cd38837655be5334400098849410ec..6515126f10857f851ca746ca5295e70ef91967d2 |
| command | sdlc close --issue 775 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T12:05:36-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, including regression tests that fail with the old signature matcher. A fresh native run confirms the diagnostic works and cleans up. One input-validation defect remains: accepted numeric arguments can silently produce a different workload than the reported values.

Done-when: current — syntax, invalid-input refusal and native qualification still cover the revised complete-stderr contract.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Both signatures call exact_error, which compares complete bytes. All 10 tests pass; replacing that helper with the old grep matcher produces eight failing subcases and zero errors. Exact positives and confirmed/unconfirmed exit1 outcomes are exercised.
  - id: BR-2
    disposition: addressed
    note: |
      The pinned correction adds the runnable README.md usage pointer and atlas/zellij-reply-safety.md:290-311 documentation of private roots, counters and liveness limits, supported by scripts/repro/zellij-false-miss.sh.
  - id: BR-3
    disposition: addressed
    note: |
      The pinned before/after diff changes run.json's reference from native.log to initial-setup-failure.log. The latter contains the setup refusal and cleanup result; native.log contains the successful measured run.
findings:
  - id: new
    severity: Important
    family: bounded-decimal-input
    title: |
      Numeric arguments undergo octal interpretation and integer overflow
    detail: |
      scripts/repro/zellij-false-miss.sh:38-40 accepts digit strings directly into Bash arithmetic. A traced native invocation with arguments 010 1 exits successfully, reports queries=010, and executes only eight queries. Invocations with 18446744073709551617 as either argument also succeed and execute a one-query/one-worker workload because the value wraps to one. Both numeric inputs, n and parallel, are the complete family in this window; their unchecked representations feed worker and launch-loop arithmetic. Validate bounded decimal strings before arithmetic, normalize accepted values consistently for execution/reporting, and add regressions for both arguments covering leading zeros, overflow, limits and one-past-limit refusal.
  - id: new
    severity: Minor
    family: new-surface-documentation
    title: |
      Atlas overstates the liveness response validation
    detail: |
      atlas/zellij-reply-safety.md:301-302 says the response begins with layout { and malformed replies remain unconfirmed, but scripts/repro/zellij-false-miss.sh:27 accepts a matching line anywhere. An actual-worker check with "unrelated diagnostic\nlayout { pane; }\n" increments the confirmed counter. This is the 2nd finding in family new-surface-documentation. Apply the rule that documented predicates must match enforced predicates across all four new documentation consumers: README.md, scripts/repro/README.md, atlas/index.md and atlas/zellij-reply-safety.md. The inaccurate predicate claims are confined to atlas lines 301-302; describe the implemented line-presence check accurately.
```

1. **Strengths**

   - One shared `exact_error` helper enforces both complete-stderr signatures.
   - Actual-worker tests verify classification and CLI call sequences, including independent liveness checks.
   - Native evidence hashes match the committed manifests.
   - Private roots, unretried measured queries and cleanup preserve the diagnostic’s independence from Pair and Couch.

2. **Critical findings**

   None.

3. **Important findings**

   - Script lines 38–40 need bounded decimal validation before arithmetic. The current behavior can invalidate the reported experiment size.

4. **Minor findings**

   - Atlas lines 301–302 describe stronger liveness validation than the worker implements.

5. **Test coverage notes**

   - `make test-zellij-false-miss`: **10 tests passed**.
   - Old-matcher mutation: **eight failing subcases, zero errors**; temporary mutation removed.
   - `bash -n` and ordinary malformed/out-of-range refusal checks passed.
   - Fresh native run: **1,000 queries, four workers; exit0=3, confirmed exit1=0, unconfirmed exit1=0, other nonzero=0; cleanup passed**.
   - Traced native runs confirmed the numeric-input defects. Committed tests currently cover the worker rather than CLI numeric validation.
   - Repository working tree remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** both signature consumers derive from one comparison helper.
   - **ARCH-PURE — pass:** external IO stays within the standalone diagnostic; worker integration tests inject the CLI boundary.
   - **ARCH-PURPOSE — pass:** the native reproduction, raw evidence and separate liveness outcomes are delivered.
   - **ARCH-EVOLVE — pass:** no existing persisted-format reader or rewrite path changes.
   - No Core concepts table is present in this quick plan.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring bounded decimal parsing for both CLI arguments, normalized workload reporting, and regressions for leading-zero and overflowing inputs. Correct the atlas’s liveness-predicate description.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e4ee2189-1000-4da4-aa88-9ddd545cc800","repository":"github.com/xianxu/pair","issue_id":"000775","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/521044732775f127","reviewed_head":"6515126f10857f851ca746ca5295e70ef91967d2","slot":"pair:5","requested_at":"2026-10-21T19:02:54.338212Z"},"phase":"finished","pid":89849,"started_at":"2026-10-21T19:02:54.486838Z","finished_at":"2026-10-21T19:05:36.126992Z","session_id":"01a15fd8-526d-7e71-840d-6ddebc26ced6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.894542792s
head: "6515126f10857f851ca746ca5295e70ef91967d2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T12:17:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 775 — Standalone Zellij false-miss reproduction |
| repo | pair |
| issue file | workshop/issues/000775-zellij-false-miss-repro.md |
| boundary | whole-issue close |
| milestone | — |
| window | eaa758a6f0cd38837655be5334400098849410ec..b88131bf4093ef81422cd19f71f18d328863c2c0 |
| command | sdlc close --issue 775 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T12:17:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both open findings are addressed. Regression tests fail when the old numeric validation is restored, documentation matches the implemented liveness predicate, and a fresh native run completed with cleanup passing. No new findings.

Done-when: current — syntax checks, invalid-input refusal and native qualification still cover the revised diagnostic.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Both signatures use complete-byte comparison. Restoring the old grep matcher produces eight failing regression subcases.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:2435 provides the invocation; atlas/zellij-reply-safety.md:290 documents the standalone diagnostic, counters and liveness limits.
  - id: BR-3
    disposition: addressed
    note: |
      run.json names initial-setup-failure.log, which contains the setup refusal and passing cleanup; native.log contains the successful measured run.
  - id: BR-4
    disposition: addressed
    note: |
      Both arguments use bounded_decimal before arithmetic. Tests cover normalization, overflow, exact limits, one-past-limit refusal and actual request counts. Restoring the old CLI validation produces nine failing subcases.
  - id: BR-5
    disposition: addressed
    note: |
      The pinned correction replaces beginning-of-response and malformed-layout claims with line-presence wording at atlas/zellij-reply-safety.md:301-304, matching the worker at scripts/repro/zellij-false-miss.sh:35. All four documentation consumers were checked.
```

1. **Strengths**

   - Shared helpers enforce both numeric bounds and exact diagnostic comparisons.
   - Tests exercise actual main/worker paths with an injected executable and private sockets.
   - Final native evidence hashes match, and its script is identical to the pinned head.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Bash syntax check and all **15 tests passed**.
   - Both mutation checks failed as expected; temporary copies were removed.
   - Fresh native run: **1,000 queries, four workers; exit0=5, confirmed exit1=0, unconfirmed exit1=0, other nonzero=0; cleanup passed**.
   - Exit1 outcomes are covered deterministically. Repository working tree remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** both argument consumers and both signature consumers share their respective helpers.
   - **ARCH-PURE — pass:** deterministic numeric validation is separate; native IO remains within the diagnostic boundary.
   - **ARCH-PURPOSE — pass:** standalone reproduction, raw anomalies and distinct liveness outcomes are delivered.
   - **ARCH-EVOLVE — pass:** no existing persisted-format reader or rewrite changes.
   - No Core concepts table is present.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"087979cd-61a7-4798-9873-b14dd1884ebe","repository":"github.com/xianxu/pair","issue_id":"000775","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e7651d89c6c2b72a","reviewed_head":"b88131bf4093ef81422cd19f71f18d328863c2c0","slot":"pair:5","requested_at":"2026-10-21T19:16:11.966725Z"},"phase":"finished","pid":91910,"started_at":"2026-10-21T19:16:12.142327Z","finished_at":"2026-10-21T19:17:40.011644Z","session_id":"01a15fe4-7e74-79e1-bf91-ba9aafeda67e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.949398917s
head: "b88131bf4093ef81422cd19f71f18d328863c2c0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T13:08:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 775 — Standalone Zellij false-miss reproduction |
| repo | pair |
| issue file | workshop/issues/000775-zellij-false-miss-repro.md |
| boundary | whole-issue close |
| milestone | — |
| window | 644a01cfea85504a86d02b7e9ce98c800d1366d6..d9bc8999768f3d49bad8cb003b6e7bf51672845c |
| command | sdlc close --issue 775 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:08:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The targeted tests pass, retained evidence hashes match, and prior corrections remain intact. However, this window introduces behavior beyond the prior findings, requiring a full close review under the supplied scope rule. The new Make target also lacks the required root README update.

Done-when: current — syntax, invalid-input refusal and native qualification still cover the standalone reproduction.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Complete-stream comparisons remain unchanged at scripts/repro/zellij-false-miss.sh:30,33; exact-signature regression tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      The existing standalone invocation remains documented at README.md:2435 and atlas/zellij-reply-safety.md:290. The newly introduced Make target has a separate documentation gap below.
  - id: BR-3
    disposition: addressed
    note: |
      workshop/evidence/000775/run.json still identifies initial-setup-failure.log as the setup-failure evidence.
  - id: BR-4
    disposition: addressed
    note: |
      Both arguments retain bounded_decimal validation before arithmetic; all 15 main/worker regression tests pass.
  - id: BR-5
    disposition: addressed
    note: |
      Both pinned versions of atlas/zellij-reply-safety.md:301-304 describe heading-line presence, matching the worker predicate at scripts/repro/zellij-false-miss.sh:35.
findings:
  - id: new
    severity: Critical
    family: post-close-scope-control
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      Makefile.local:625-628 removes the default regression dependency and adds native execution through a new manual target. The issue's Revisions entry at workshop/issues/000775-zellij-false-miss-repro.md:86 explicitly adopts this behavior, which none of BR-1 through BR-5 requested. Under the supplied narrow-review rule, this requires a full close review rather than admission through this fix interdiff.
  - id: new
    severity: Important
    family: new-surface-documentation
    title: |
      README update appears missing for make zellij-false-miss
    detail: |
      Makefile.local:626 introduces a command users run, but root README.md is unchanged; its reproduction section at line 2435 documents only the Bash invocation. This is the 3rd finding in family new-surface-documentation. Apply the rule across every added or changed runnable surface: update the root README alongside the relevant guide and atlas documentation, including this target's regression-plus-native behavior.
```

1. **Strengths**

   - All 15 diagnostic regressions and all 12 affected-runner tests pass.
   - Atlas and repro-guide updates accurately describe the explicit target.
   - Manual-run log and script hashes match the manifest; the recorded Makefile matches the pinned head.

2. **Critical findings**

   - `Makefile.local:625-628`: post-close behavior expansion requires full review, as detailed above.

3. **Important findings**

   - `README.md:2435`: document the new manual target using the family-wide rule above.

4. **Minor findings:** None.

5. **Test coverage notes**

   Bash syntax and Make dry-run checks pass. Recipe ownership passes at this head. No source files were changed. Native execution and the full suite were not rerun; retained native evidence was checked.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Make delegates to existing tests and script.
   - **ARCH-PURE — pass:** orchestration remains thin.
   - **ARCH-PURPOSE — flag:** expanded behavior requires review against the complete contract.
   - **ARCH-EVOLVE — pass:** no persisted-format reader or rewrite changes.

7. **Plan revision recommendations**

   Record the full-close review requirement in `## Revisions` and replace the entry’s instruction to perform only a narrow re-close.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e6e4fa5b-7c0e-4bd9-aecf-e2ac7b762e48","repository":"github.com/xianxu/pair","issue_id":"000775","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4a73011cb725c0f8","reviewed_head":"d9bc8999768f3d49bad8cb003b6e7bf51672845c","slot":"pair:5","requested_at":"2026-10-21T20:07:33.707895Z"},"phase":"finished","pid":64367,"started_at":"2026-10-21T20:07:33.819787Z","finished_at":"2026-10-21T20:08:44.344151Z","session_id":"01a16013-83e4-7e10-bbe7-242d0d29ef5b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.00242425s
head: "d9bc8999768f3d49bad8cb003b6e7bf51672845c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
