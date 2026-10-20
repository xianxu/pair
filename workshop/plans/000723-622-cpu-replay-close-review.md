# Boundary Review — pair#723 (whole-issue close)

| field | value |
|-------|-------|
| issue | 723 — pair#622 CPU acceptance: controlled replay A/B |
| repo | pair |
| issue file | workshop/issues/000723-622-cpu-replay.md |
| boundary | whole-issue close |
| milestone | — |
| window | 21bdc299427b3f824cfb426bad756cbed04ad2d6..2226be35ea2d7bd40dbddf738a2fe3a1c2b12486 |
| command | sdlc close --issue 723 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-19T22:42:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The replay evidence checks out: retained artifacts match the report, both controls pass, the missing-Sink variant fails, and independently recomputed medians match. No runtime defect was found. Before close, reconcile the Spec’s outstanding console-attestation requirement with its stated ownership under #622.

Done-when: current — the accounting, publication, parity, retention, and uncertainty clauses cover the corrected experiment.

1. **Strengths**

   - The committed driver and runner exactly match the retained source hashes.
   - Capture witness, file identities, selected timestamp events, geometry, and frozen prefix hash agree.
   - All twelve retained observations match their logs; corrected binary hashes and both attempts’ medians verify.
   - Publication delivery, reply flushing, capture, and validation occur inside the corrected CPU bracket. The report appropriately limits the speedup claim.

2. **Critical findings**

   None.

3. **Important findings**

   The remaining acceptance-scope inconsistency is detailed below.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Both retained binaries passed `Test622ReplayControls`. The retained missing-Sink variant failed with `invalid accounting`, confirming detection of the omitted publication stage. Source inspection confirms both asynchronous queues retain in-flight work until acknowledgment. No additional measurement trials were run.

6. **Architectural notes**

   - **ARCH-DRY: pass.** The experiment reuses the production Child/Endpoint/parser path.
   - **ARCH-PURE: pass.** Experiment integration code owns IO; no business logic was moved into production handlers.
   - **ARCH-PURPOSE: flag.** Controlled measurement is delivered, but the written acceptance ownership remains inconsistent.
   - **ARCH-EVOLVE: pass.** No existing persisted format or rewrite path changes.

   No new product or architectural surface requires README or atlas updates.

7. **Plan revision recommendation**

   Add a `## Revisions` entry explicitly assigning live consumer/console acceptance to #622 and align the Spec accordingly, or supply the attestation required by the existing Spec.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: acceptance-scope-consistency
    title: |
      Reconcile the console-attestation requirement before whole-issue close
    detail: |
      workshop/issues/000723-622-cpu-replay.md:42 still requires appending operator console attestation before close, but none is supplied. All instances of this scope inconsistency are the Spec requirement at line 42, the Problem assigning deployment/consumer conformance to #622 at line 26, the log explicitly leaving console attestation pending under #622 at line 64, and the completed close-preparation Plan item at line 58. ARCH-PURPOSE: either provide the required attestation or explicitly revise the Spec and Revisions to establish that this acceptance belongs exclusively to #622; the completed replay does not itself satisfy that clause.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"43575c01-f599-4fa6-993e-050b92301b05","repository":"github.com/xianxu/pair","issue_id":"000723","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/6a51ecab054d0c30","reviewed_head":"2226be35ea2d7bd40dbddf738a2fe3a1c2b12486","slot":"pair:5","requested_at":"2026-10-20T05:40:20.517208Z"},"phase":"finished","pid":49350,"started_at":"2026-10-20T05:40:20.661606Z","finished_at":"2026-10-20T05:42:02.961046Z","session_id":"01a157d3-31c3-7ff1-b0b2-b7688c0794ce","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.833116584s
head: "2226be35ea2d7bd40dbddf738a2fe3a1c2b12486"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
