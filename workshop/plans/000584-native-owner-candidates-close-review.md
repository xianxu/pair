# Boundary Review — pair#584 (whole-issue close)

| field | value |
|-------|-------|
| issue | 584 — Native owner scan blocks fleet resume on unrelated processes |
| repo | pair |
| issue file | workshop/issues/000584-native-owner-candidates.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3c7f6cb43d05f889aaf852cb61f61ec4fb005051..935fd8c6820b93349468454644e83ebbe693ab39 |
| command | sdlc close --issue 584 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:33:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The candidate filtering, ownership fences, and bounded scheduling passed the checks run here. Two architecture-contract gaps block SHIP: the plan misclassifies IO orchestration as pure, and the new `lsof` dependency lacks repeatable live conformance coverage.

```findings
findings:
  - id: new
    severity: Critical
    family: pure-integration-classification
    title: |
      Core concepts classifies IO-driven retry and ownership proof as PURE
    detail: |
      workshop/plans/000584-native-owner-candidates-plan.md:15–16 lists observation retry and legacy source proof as pure entities. Their implementations invoke injected observation, metadata reads, and held-file probes (recovery_native_owner.go:74 and recovery_native_legacy.go:44); their tests require those fakes. ARCH-PURE and the Core concepts gate require promoting these orchestration entities to INTEGRATION and naming the actual deterministic helpers separately. Add a Revisions entry correcting the table.
  - id: new
    severity: Important
    family: external-dependency-conformance
    title: |
      Writable-file ownership proof has no repeatable live lsof conformance check
    detail: |
      cmd/internal/procutil/procutil.go:105 introduces an admission-critical external contract, but lsof_writable_test.go:14 replaces lsof with scripted output throughout its tests. No committed live check or recurring invocation exercises this contract against real descriptors; the issue's manual observations do not provide ongoing drift detection. ARCH-MOCK: add an opt-in conformance test using temporary files opened read-only, writable, and then closed, call the production LsofWritableNames boundary, and connect it to the live-check workflow with a documented cadence.
```

1. **Strengths**

   - Candidate filtering preserves uncertainty for possible wrappers while excluding unrelated services and zombies.
   - Shared exact-ordinal ledger projection avoids separate modern, legacy, and orphan implementations.
   - Scheduler tests control completion order and verify cancellation joins all workers.
   - Couch diagnostic tests verify preview reads preserve pending journals and storage.

2. **Critical findings**

   The Core concepts classification mismatch above. Correct the table and identify concrete pure helpers; this does not inherently require changing runtime behavior.

3. **Important findings**

   The missing `lsof` conformance check above. Exercise real descriptor changes through the same boundary used by admission.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Full tests for launcher, procutil, sessionledger, strictjson, and sessioninventory.
   - Couch legacy-slot advice regression.
   - Scheduler and concurrent memo tests under `-race`, three repetitions.
   - JSON differential fuzzing: 42,275 inputs.
   - `go build ./cmd/...` and pinned-range whitespace checks.

   The full recorded repository suite was not independently rerun. No files were changed.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared ledger projection and existing filename codecs.
   - **ARCH-PURE — flag:** classification mismatch described above.
   - **ARCH-PURPOSE — pass:** candidate, legacy, modern-unbound, and orphan paths are covered.
   - **ARCH-MOCK — flag:** missing recurring real-`lsof` comparison.
   - **ARCH-CONSTRAINTS — pass:** bounded workers, subprocess output, deadlines, and retry count.
   - **ARCH-SECURE — pass:** malformed or incomplete evidence refuses admission.
   - **ARCH-ORDER — pass:** deterministic result consumption, evidence comparisons, cancellation and joining.
   - **ARCH-FUNERAL — pass:** memo and worker lifetimes end with admission; no new durable artifacts.

   Atlas coverage is present. No new command, flag, or configuration surface requiring a README update was identified.

7. **Plan revision recommendations**

   Add `## Revisions` entries that:

   - Reclassify observation retry and legacy proof orchestration as INTEGRATION, naming deterministic helpers separately.
   - Specify the real-`lsof` conformance fixture, invocation, and recurring cadence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"06df5d05-36f0-43a1-bcf1-389b396d4207","repository":"github.com/xianxu/pair","issue_id":"000584","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5020d8595e076223","reviewed_head":"935fd8c6820b93349468454644e83ebbe693ab39","slot":"pair:2","requested_at":"2026-10-16T00:30:42.69503Z"},"phase":"finished","pid":83515,"started_at":"2026-10-16T00:30:42.868494Z","finished_at":"2026-10-16T00:33:02.679394Z","session_id":"01a1421e-47f2-7be3-8a5f-e9c14906927a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.926174542s
head: "935fd8c6820b93349468454644e83ebbe693ab39"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T17:40:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 584 — Native owner scan blocks fleet resume on unrelated processes |
| repo | pair |
| issue file | workshop/issues/000584-native-owner-candidates.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3c7f6cb43d05f889aaf852cb61f61ec4fb005051..44b10b71dd206c8f694724df6137f54b8c9f7cbc |
| command | sdlc close --issue 584 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T17:40:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the Spec and revised Plan. Both prior findings are addressed, and no new blocking defects were found. Candidate filtering preserves uncertainty for possible owners, legacy exemptions retain generation fences, and concurrent proofs cancel and join before returning.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Inspected the pinned before/after table correction. Plan lines 15–29 now name deterministic helpers separately from IO orchestration; the Revisions passage at line 90 explains the correction. The referenced implementations support those classifications.
  - id: BR-2
    disposition: addressed
    note: |
      TestLsofWritableNamesLiveConformance exercises production LsofWritableNames with writable, read-only, closed and reopened descriptors; it passed against real lsof. Makefile.local:136 connects it to the existing weekly/manual macOS workflow, and plan line 92 documents cadence. This adds conformance coverage without changing production behavior.
```

1. **Strengths**
   - Candidate filtering precedes argv certainty while retaining actionable PID evidence for uncertain candidates.
   - Shared exact-ordinal ledger projection avoids divergent modern, legacy and orphan rules.
   - Scheduler tests control completion order and verify cancellation joins every worker.
   - Migration-advice tests verify preview reads preserve storage and pending journals.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Passed affected launcher, procutil, sessionledger, strictjson and sessioninventory packages; focused Couch ownership tests; three race repetitions of ownership/scheduling regressions; real lsof conformance; and `go build ./cmd/...`. Differential JSON fuzzing passed 29,038 executions. The full recorded repository suite was not rerun during this review. No mutation check was performed under the read-only constraint.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared ledger projection and existing filename codecs.
   - **ARCH-PURE — pass:** corrected classifications match deterministic helpers and injected IO adapters.
   - **ARCH-PURPOSE — pass:** candidate, legacy, modern-unbound and orphan consumers are covered.
   - **ARCH-MOCK — pass:** stateful ownership fixtures plus recurring production-boundary conformance.
   - **ARCH-CONSTRAINTS — pass:** bounded retries, deadlines, output and four-worker concurrency.
   - **ARCH-SECURE — pass:** malformed or partial evidence refuses; metadata retains regular-file and generation checks.
   - **ARCH-ORDER — pass:** both observation fences and reproducible cancellation/order tests remain enforced.
   - **ARCH-FUNERAL — pass:** memo lifetime is one admission; workers are joined; no new durable production artifacts.

   Atlas documents the new proof flow. The relaunch command already appears in README; no new user-facing syntax requires an update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7889d3ca-4b86-4aea-b34b-2440c876d92f","repository":"github.com/xianxu/pair","issue_id":"000584","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ac00987f608f6cd7","reviewed_head":"44b10b71dd206c8f694724df6137f54b8c9f7cbc","slot":"pair:2","requested_at":"2026-10-16T00:38:35.514066Z"},"phase":"finished","pid":39249,"started_at":"2026-10-16T00:38:35.697515Z","finished_at":"2026-10-16T00:40:33.636236Z","session_id":"01a14225-7ee2-7aa1-973e-3577d784d6b8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.761782083s
head: "44b10b71dd206c8f694724df6137f54b8c9f7cbc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
