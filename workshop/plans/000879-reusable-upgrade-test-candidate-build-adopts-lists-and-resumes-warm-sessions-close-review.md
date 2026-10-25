# Boundary Review — pair#879 (whole-issue close)

| field | value |
|-------|-------|
| issue | 879 — Reusable upgrade test: candidate build adopts, lists and resumes warm sessions created by main |
| repo | pair |
| issue file | workshop/issues/000879-reusable-upgrade-test-candidate-build-adopts-lists-and-resumes-warm-sessions.md |
| boundary | whole-issue close |
| milestone | — |
| window | 597d8b49bc89498fd6fa37094b3aac0723f1c360..ee8116e1ac241a77254c4ea015ffd57594c72427 |
| command | sdlc close --issue 879 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T18:07:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The replacement seam and historical qualification have solid evidence: focused tests pass, and the retained native logs match their recorded hashes. SHIP is blocked by a legacy-only assertion in the reusable fixture, a contradictory core-concepts table, and missing README guidance.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: qualification-baseline-independence
    title: |
      The reusable upgrade oracle requires readiness markers to remain absent.
    detail: |
      cmd/couch/upgrade_acceptance_test.go:295-299 rejects any nonempty zellij-ready-v1 directory on every snapshot, including candidate snapshots. A baseline that legitimately writes readiness receipts, or a compatible candidate that establishes them during adoption, therefore fails regardless of successful adoption/list/resume/attach. ARCH-PURPOSE: separate the historical pre-#866 absence proof from generic cohort snapshots; accept genuine marker-bearing baselines and legitimate candidate publication without manufacturing or deleting receipts. Add regression coverage for both cases.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-table-code-agreement
    title: |
      The core-concepts table names the wrong file for the pure cohort comparison.
    detail: |
      workshop/plans/000879-native-upgrade-plan.md:21 places the comparison in cmd/internal/testcouch/upgrade.go, but that file contains replacement IO; compareUpgradeCohort actually lives at cmd/couch/upgrade_acceptance_test.go:304. The later Revisions passage acknowledges the move but leaves the authoritative table contradictory. ARCH-PURE: update the table to the actual function and location, retaining its PURE classification. This is Critical under the requested core-concepts cross-check.
  - id: new
    severity: Important
    kind: defect
    family: user-surface-readme-coverage
    title: |
      README guidance is missing for the new upgrade qualification invocation.
    detail: |
      atlas/couch-testing.md:200 introduces TestDisposableCouchBuildUpgrade and PAIR_NATIVE_COUCH_UPGRADE, PAIR_COUCH_UPGRADE_BASE_SOURCE, and PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE, but README.md is unchanged. Add a discoverable invocation or concise link beside the existing disposable-fleet guidance at README.md:2694, including the generated-runtime preparation requirement.
```

1. **Strengths**

   - `ReplaceBuild` stages physical copies, excludes other admitted operations, and fences further admission after partial publication.
   - Attachment checks combine retained drafts, native clients, complete bindings, process generations, and socket identities.
   - Historical evidence is verifiable: the healthy and RED logs exist and match the documented SHA256 hashes. The RED log shows initialization refusal before timeout and confirmed cleanup.

2. **Critical findings**

   - Separate historical marker expectations from the reusable oracle.
   - Correct the plan table’s function location. The implementation’s pure comparison itself needs no IO.

3. **Important findings**

   - Add README discovery and usage guidance for the new test surface. The atlas update is present.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused replacement tests, their race run, cohort/receipt controls, artifact-source classification checks, and `git diff --check`. The native test skipped by default; I inspected the retained native runs rather than rerunning them. Marker-bearing baseline and candidate-publication cases lack coverage.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing copier, fleet lifetime, and cleanup authority are reused.
   - **ARCH-PURE — flag:** core-concepts table contradicts the delivered location; pure tests themselves are appropriate.
   - **ARCH-PURPOSE — flag:** unconditional legacy-marker absence restricts the reusable qualification.
   - **ARCH-MOCK — pass:** existing stateful agent fixtures and native conformance seams are reused.
   - **ARCH-CONSTRAINTS — pass:** sequential personalities, bounded contexts, and bounded executable copies.
   - **ARCH-SECURE — pass:** explicit source paths and private runtime destinations retain existing validation.
   - **ARCH-ORDER — pass:** partial publication preserves cleanup authority and closes admission before unlocking; deterministic failure controls exercise it.
   - **ARCH-FUNERAL — pass:** staging directories and candidate builds have cleanup owners; uncertain fleet cleanup stops the series.
   - **ARCH-EVOLVE — pass:** this range introduces no production persisted-format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry separating historical unmarked-session qualification from the reusable baseline contract, and update the core-concepts table to name `compareUpgradeCohort` at its actual path.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6252fb16-3a54-49f9-9be1-f4c3f4481874","repository":"github.com/xianxu/pair","issue_id":"000879","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/61e7d34fcc74b1be","reviewed_head":"ee8116e1ac241a77254c4ea015ffd57594c72427","slot":"pair:9","requested_at":"2026-10-25T01:05:38.322254Z"},"phase":"finished","pid":58544,"started_at":"2026-10-25T01:05:38.432676Z","finished_at":"2026-10-25T01:07:15.882042Z","session_id":"01a17097-7efc-7e51-93bc-cb52a8308dbc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.790841375s
head: "ee8116e1ac241a77254c4ea015ffd57594c72427"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
