# Boundary Review — pair#658 (whole-issue close)

| field | value |
|-------|-------|
| issue | 658 — pair test suites write ~20 MB/s (couchcmd.test): measure per package and cut, with a counted budget |
| repo | pair |
| issue file | workshop/issues/000658-pair-test-suites-write-20-mb-s-couchcmd-test-measure-per-package-and-cut-with-a-counted-budget.md |
| boundary | whole-issue close |
| milestone | — |
| window | b08a927f6fed942fdbb4aef84b09cf07b2a32f47..e5e0d8cc6bdbbed0cc93e94df928dc1b97380a0d |
| command | sdlc close --issue 658 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T02:06:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the revised scope without weakening existing fixture sizes or assertions. Focused tests passed, raw measurement totals agree with the reports, and the remaining attribution work exists as open, unclaimed pair#663. No blocking findings.

Done-when: current — it still covers package measurements, the couchcmd reduction, counted enforcement, whole-device comparison, and the remaining-share follow-up.

```findings
{}
```

1. **Strengths**

   - Ledger construction retains exact padding and records while using private clones; fallback, collisions, partial failures, and mutation independence have focused coverage (`cmd/internal/workbenchperftest/ledger_writer.go:30`).
   - Budget enforcement preserves test failures and rejects supported-counter errors, counter regression, and excess writes (`cmd/internal/testdisk/run.go:21`).
   - Real Git constructors enforce inventory caps before returning. Independently rerun N=8/16 tests passed with zero sample hooks (`cmd/internal/couchcore/fixture_budget_test.go:28`).
   - Independently summed 116 measurement rows confirm 1,640,353,792→445,779,968 bytes. Couchcmd’s recorded 625,799,168→56,639,488 exceeds the required reduction. Device-wide limitations remain explicit.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed fresh runs of testdisk, testgit, workbenchperftest, and testpair; real constructor and helper-cleanup regressions; shell bootstrap/cache contracts; native excess-write refusal; and CGO-disabled portable controls. The cache regression exercises 32 independent homes. I inspected retained full-suite evidence rather than rerunning the full suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared cache resolver, budget runner, and inventory comparison.
   - **ARCH-PURE — pass:** arithmetic and verdict logic remain directly testable without IO.
   - **ARCH-PURPOSE — pass:** all five ledger consumers derive from the shared writer; all seven selected packages enforce budgets.
   - **ARCH-MOCK — pass:** stateful counter/cache doubles share the relevant seams; real Git, filesystem, and build controls remain.
   - **ARCH-CONSTRAINTS — pass:** explicit platform/filesystem envelope, checked repetition scaling, and counted growth tests.
   - **ARCH-SECURE — pass:** owned-tree audits reject symlinks; explicit environment values retain literal argument boundaries.
   - **ARCH-ORDER — pass:** synchronous setup/run/cleanup is bounded; exec helper roles bypass ownership that cannot survive exec.
   - **ARCH-FUNERAL — pass:** new template and seed artifacts have scoped cleanup owners.

   Atlas documents the new internal surfaces. No new user-facing command, flag, or configuration key requires a README update. Core-concept locations and classifications match the implementation.

7. **Plan revision recommendations:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"61a71b45-6ee6-4700-a0ad-5fdfba8e5c5e","repository":"github.com/xianxu/pair","issue_id":"000658","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c5982f6848f1da90","reviewed_head":"e5e0d8cc6bdbbed0cc93e94df928dc1b97380a0d","slot":"pair:3","requested_at":"2026-10-18T09:01:00.023115Z"},"phase":"finished","pid":87066,"started_at":"2026-10-18T09:01:00.245331Z","finished_at":"2026-10-18T09:06:42.255689Z","session_id":"01a14e3e-2fa6-7400-b56e-942cf51e5d99","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.139111s
head: "e5e0d8cc6bdbbed0cc93e94df928dc1b97380a0d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
