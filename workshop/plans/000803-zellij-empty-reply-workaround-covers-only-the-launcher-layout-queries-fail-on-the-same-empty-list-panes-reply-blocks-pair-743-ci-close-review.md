# Boundary Review — pair#803 (whole-issue close)

| field | value |
|-------|-------|
| issue | 803 — Zellij empty-reply workaround covers only the launcher; layout queries fail on the same empty list-panes reply (blocks pair#743 CI) |
| repo | pair |
| issue file | workshop/issues/000803-zellij-empty-reply-workaround-covers-only-the-launcher-layout-queries-fail-on-the-same-empty-list-panes-reply-blocks-pair-743-ci.md |
| boundary | whole-issue close |
| milestone | — |
| window | 365fd86e7ed8895236ba4b0c67cf278168e54d53..191e8491b7de6972df06e420d378346e8d5cc8d9 |
| command | sdlc close --issue 803 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T09:52:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared read seam fulfills the approved Go scope and preserves strict confirmation, bounded retries, and mutation behavior. Native conformance passed. One reproducible launcher test failure blocks close: an existing assertion still expects the pre-extraction attempt count.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: recovery-contract-test-alignment
    title: |
      Native name-absence regression still requires one attempt after migration to shared recovery
    detail: |
      cmd/internal/launcher/osruntime.go:169 routes name probes through shared recovery, but cmd/internal/launcher/native_name_absence_test.go:79 still requires calls == 1 for every case. Both the full launcher package and the isolated requested_identity_present subtest fail with "probe calls=3". Give that false-miss case an explicit three-attempt expectation, preserve its refusal assertion and the one-attempt expectations for other cases, then rerun the launcher package.
```

1. **Strengths**

   - `zellijquery/query.go:82` centralizes the original deadline and three-attempt bound.
   - `layoutcmd/query_adapter_test.go:12` exercises empty-reply recovery through the actual OS adapter while retaining two independent confirmations and zero effects.
   - Raw process errors remain separate from retry classification, preserving diagnostic authority.
   - Atlas documents the shared owner, process-group requirements, and approved Lua/diagnostic exclusions.

2. **Critical findings:** None.

3. **Important findings:** The launcher regression above fails deterministically. Update its case-specific attempt expectation without weakening refusal behavior.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: full zellijquery, layoutcmd, repolint, and artifactpath packages; targeted regressions across all five other adapters; affected-reader inventory.
   - Passed: focused race checks for shared reads and title process-group/EOF cleanup.
   - Native terminal-recovery conformance passed with `TMPDIR=/tmp`; the initial invocation failed before queries because the review temporary path exceeded Zellij’s socket-name allowance.
   - Full launcher tests failed only at the reported assertion; isolated reproduction confirmed it.
   - Checkout remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one recovery owner; launcher forwards to it.
   - **ARCH-PURE — pass:** classification stays pure; collection and socket observation remain boundary operations.
   - **ARCH-PURPOSE — pass:** the sweep covers approved Go consumers.
   - **ARCH-MOCK — pass:** production adapters use stateful executable fixtures; native conformance exercises the shared seam.
   - **ARCH-CONSTRAINTS — pass:** sequential attempts, shared deadline, and counted work bounds are exercised.
   - **ARCH-SECURE — pass:** strict parsing and process-error provenance remain intact.
   - **ARCH-ORDER — pass:** cancellation prevents subsequent attempts; EOF cleanup is exercised.
   - **ARCH-FUNERAL — pass:** no new durable artifacts or background workers.
   - **ARCH-EVOLVE — pass:** no persisted-format changes.

7. **Plan revision recommendations:** None. Complete the existing verification checklist after repairing and rerunning the launcher test.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e158f44e-93eb-4b61-8f69-871582d4f53a","repository":"github.com/xianxu/pair","issue_id":"000803","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b558995a857b628e","reviewed_head":"191e8491b7de6972df06e420d378346e8d5cc8d9","slot":"pair:6","requested_at":"2026-10-22T16:49:40.770087Z"},"phase":"finished","pid":23613,"started_at":"2026-10-22T16:49:40.964767Z","finished_at":"2026-10-22T16:52:49.602402Z","session_id":"01a16484-b675-7012-a074-9561954ded05","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.436884s
head: "191e8491b7de6972df06e420d378346e8d5cc8d9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
