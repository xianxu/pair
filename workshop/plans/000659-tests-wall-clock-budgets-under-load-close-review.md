# Boundary Review — pair#659 (whole-issue close)

| field | value |
|-------|-------|
| issue | 659 — tests: seven wall-clock budgets on external tools (git worktree, zellij IPC, lsof, a rate-limit window) fail at fleet load |
| repo | pair |
| issue file | workshop/issues/000659-tests-wall-clock-budgets-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6ae7e32b0e128f95f6ca3ae0c1541b79560ac40..1a1f729229e3539cd0dfca7524ae6d11d5631eb0 |
| command | sdlc close --issue 659 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T23:57:08-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The timing corrections preserve production budgets and pass focused verification. Three contract/documentation gaps remain: the plan misclassifies an IO-dependent verdict helper as PURE, the acceptance checklist does not reflect the revised load requirement, and README omits the new conformance switch.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-purity-classification
    title: |
      ARCH-PURE: the probe verdict is classified PURE but depends on IO
    detail: |
      workshop/plans/000659-test-budgets-plan.md:27 labels the optional/required verdict PURE, but requireLsofObservation reads os.Getenv and can execute HostState (proof_test.go:145–154); its policy tests launch subprocesses (lines 167–201). This is the sole contradictory PURE row in this window. Promote it to INTEGRATION and record a plan Revisions entry, or extract a deterministic classifier taking explicit inputs and test that directly without IO.
  - id: new
    severity: Important
    family: acceptance-checklist-contract-sync
    title: |
      Done when retains the superseded load-acceptance contract
    detail: |
      workshop/issues/000659-tests-wall-clock-budgets-under-load.md:104–111 still requires deliberate fleet-load repetitions and a full suite under that same load, while Revisions:194 accepts injected-delay proof supported by observed-load repetitions. The recorded repetition and suite cohorts have different loads. Both load-related checklist clauses belong to this family; update them to express the accepted evidence requirements without implying another load campaign is necessary.
  - id: new
    severity: Important
    family: readme-new-run-surface
    title: |
      README update appears missing for required lsof conformance mode
    detail: |
      cmd/internal/diagnosticlog/proof_test.go:151 introduces PAIR_REQUIRE_LSOF_CONFORMANCE=1, and atlas/couch.md:26 documents its invocation, but README.md is unchanged. This is the only new operator-set configuration surface in this window; both native tests consume it through the shared helper. Add the invocation and optional-skip versus required-failure behavior to README's testing guidance.
```

Done-when: stale — its load clauses still describe the original acceptance contract rather than the revised evidence standard.

1. **Strengths**

   - The HTTP rate test freezes logical time, checks exact limiting, and verifies refill while socket deadlines retain real time.
   - Recovery ownership tests use catalog state independently of native Git latency; the new deadline test checks refusal and retained-state preservation.
   - Lsof policy coverage exercises optional and required modes, including wrong-holder and wrong-identity failures.

2. **Critical findings**

   The PURE classification contradiction above requires correction under the supplied core-concepts rule.

3. **Important findings**

   Synchronize the acceptance checklist and document the conformance switch in README.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused tests passed across testwait, broadcast, couchcore, couchcmd, couchtty, and diagnosticlog. Both native lsof tests also passed with required mode enabled. Independently parsed the saved soak artifact: every named function passed 20 times, with no failures or skips; its SHA256 matches the tracker. The saved suite log hash also matches. No files were modified.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Existing clocks, fakes, and patience helpers are reused.
   - **ARCH-PURE: flag.** Correct the verdict helper’s classification.
   - **ARCH-PURPOSE: pass for implementation coverage.** All seven groups are addressed; acceptance wording needs reconciliation.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording the verdict helper’s actual integration boundary and the accepted injected-delay/observed-load evidence standard. Update the issue’s corresponding Done-when clauses.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"90a121e7-4277-4c77-8846-742c2097d6b4","repository":"github.com/xianxu/pair","issue_id":"000659","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/95545d166b3092e1","reviewed_head":"1a1f729229e3539cd0dfca7524ae6d11d5631eb0","slot":"pair:7","requested_at":"2026-10-18T06:55:28.420392Z"},"phase":"finished","pid":45524,"started_at":"2026-10-18T06:55:28.638764Z","finished_at":"2026-10-18T06:57:07.005056Z","session_id":"01a14dcb-42ee-70f3-891a-e307fc9b79b7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.339555291s
head: "1a1f729229e3539cd0dfca7524ae6d11d5631eb0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T00:01:29-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 659 — tests: seven wall-clock budgets on external tools (git worktree, zellij IPC, lsof, a rate-limit window) fail at fleet load |
| repo | pair |
| issue file | workshop/issues/000659-tests-wall-clock-budgets-under-load.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6ae7e32b0e128f95f6ca3ae0c1541b79560ac40..8f59f920b2064566f2da4baddf36d0f56cd6041f |
| command | sdlc close --issue 659 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T00:01:29-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation covers all seven affected groups while preserving product deadlines. All three prior findings are addressed by documentation corrections verified against the code. Focused tests and required native lsof conformance passed; retained repetition and full-suite evidence match the tracker.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan's Core concepts table now classifies requireLsofObservation as integration code, matching its environment reads, testing verdicts, host diagnostics and subprocess checks in proof_test.go:145–201. The Revisions entry records the correction; executable behavior is unchanged.
  - id: BR-2
    disposition: addressed
    note: |
      Both load-related Done when clauses at issue:104–114 and the active plan tasks now match the accepted measured-load Revision: injected-delay proof, observed-load repetitions and a separately recorded passing suite. The superseded clauses remain identified as historical.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:1848 documents the required-mode invocation and optional-skip versus required-failure behavior. Both native tests consume the shared helper; policy tests cover both modes and wrong answers, and fresh required-mode native tests passed.
```

1. **Strengths**

   - The pointer test checks exact burst limiting and refill using injected logical time; socket deadlines retain real time.
   - Recovery tests separate ownership semantics from native Git latency, with a dedicated timeout test checking retained state and absence of launches or repair.
   - Lsof policy tests cover unavailable, incomplete, malformed, wrong-holder and wrong-identity outcomes in both modes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused tests passed across testwait, broadcast, couchcore, couchcmd, couchtty and diagnosticlog. Both native lsof tests passed with required mode enabled. Independently parsed the retained soak log: each targeted function passed 20 times. Soak and full-suite log hashes match the tracker. The working tree remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Existing clocks, catalog fakes, timeout seams and patience helpers are reused.
   - **ARCH-PURE: pass.** Deterministic rate decisions remain separate from clock delivery and test IO; the corrected plan reflects this.
   - **ARCH-PURPOSE: pass.** All seven groups are covered, including native conformance enforcement and failure diagnostics.

7. **Plan revision recommendations:** None outstanding.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"68b12d5a-f24b-434d-b55f-c42690e9b276","repository":"github.com/xianxu/pair","issue_id":"000659","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/6d3b5441507ed34f","reviewed_head":"8f59f920b2064566f2da4baddf36d0f56cd6041f","slot":"pair:7","requested_at":"2026-10-18T07:00:28.060288Z"},"phase":"finished","pid":98361,"started_at":"2026-10-18T07:00:28.289018Z","finished_at":"2026-10-18T07:01:28.859056Z","session_id":"01a14dcf-d5c0-7fa2-8276-2289094a05d9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.096277125s
head: "8f59f920b2064566f2da4baddf36d0f56cd6041f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
