# Boundary Review — pair#866 (whole-issue close)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0859007d1bdee7f723eef4ba086e3ec18ff79312..50db793d663d30bd059e6358e2e54652395f8b31 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T04:50:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The creator lease, exact first-pane witness, and joined native-command seam provide solid startup protection. However, the helper misclassifies several reads and admits them against unmarked endpoints, violating the documented conservative-read contract. A reproducible test portability failure and missing README migration guidance also need correction.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: native-read-admission
    title: |
      Non-list helper reads bypass positive readiness qualification
    detail: |
      cmd/internal/zellijquery/command.go:86-101 classifies only list-sessions and action list-* as reads. Against a private unmarked Unix endpoint, the actual CLI helper refused list-clients but invoked native commands for are-floating-panes-visible, current-tab-info and dump-layout. The migrated review client uses are-floating-panes-visible at nvim/review/restore_client.lua:166. This contradicts the governing plan's requirement that helper reads retain qualification. Enumerate the complete supported read class and add production-helper regression tests proving every read refuses unmarked endpoints while explicit mutations/recovery retain exclusion-only admission. ARCH-PURPOSE, ARCH-SECURE.
  - id: new
    severity: Important
    kind: defect
    family: unix-fixture-path-budget
    title: |
      The new recovery regression fails with a long inherited temporary directory
    detail: |
      cmd/internal/zellijquery/guard_controls_test.go:303 creates its socket root using os.MkdirTemp("", "q-"), which does not ensure a short pathname. The default command go test ./cmd/internal/zellijquery -run '^TestExplicitRecoveryPreservesUnmarkedEndpointAccess$' -count=1 reproducibly fails at line 318 with bind: invalid argument under this review environment's TMPDIR; the same test passes with TMPDIR=/tmp. Use the existing testpair.ShortRoot helper, already used by the launcher recovery regression. ARCH-CONSTRAINTS.
  - id: new
    severity: Important
    kind: defect
    family: user-facing-migration-docs
    title: |
      README startup and rollout guidance was not updated
    detail: |
      README.md:873-882 remains unchanged across the range despite the plan requiring README startup semantics. Automatic observations now refuse namespaces containing unmarked sessions, and protection depends on rebuilding cooperating clients. Document the legacy-session impact, supported recovery/relaunch action and coverage limits near the existing startup guidance, linking the updated atlas. ARCH-PURPOSE.
  - id: new
    severity: Minor
    kind: defect
    family: single-admission-policy
    title: |
      PlanProbeAdmission is an unused parallel admission implementation
    detail: |
      cmd/internal/zellijquery/readiness.go:66-77 has no production or test callers; production instead uses qualifyCandidates in readiness_os.go:182-205. The governing concept table nevertheless claims direct counted tests for PlanProbeAdmission. Remove the unused API and correct the table, or make one pure policy serve production and direct tests. ARCH-DRY, ARCH-PURE.
```

1. **Strengths**

   - Exact pane evidence releases namespace exclusion before AgentReady, avoiding the guarded-rename bootstrap cycle. The launcher regression exercises stale evidence and withheld AgentReady.
   - Lease extraction preserves stable inode and close-on-exec behavior.
   - Marker retirement preserves successor generations and uncertain evidence.
   - Retained native captures match their documented hashes and record both agent cases passing with exact cleanup.

2. **Critical findings**

   - `command.go:86`: replace the name-prefix read classifier with complete read classification. Test the actual helper against unmarked endpoints; the CLI reproduction above demonstrates the current failure.

3. **Important findings**

   - `guard_controls_test.go:303`: allocate a socket-safe private root.
   - `README.md:873`: add the promised startup compatibility and rollout guidance.

4. **Minor findings**

   - Remove or integrate the unused `PlanProbeAdmission` policy and reconcile its plan row.

5. **Test coverage notes**

   - Passed complete `zellijquery`, `panebirth`, `artifactpath` and `repolint` packages with `TMPDIR=/tmp`.
   - Passed focused launcher/CLI tests and native-admission tests across all seven migrated caller packages.
   - Passed `tests/zellij-trace-test.sh`, `make test-affected-inventory`, and source diff whitespace checks.
   - The default temporary-directory recovery test fails as reported above.
   - Native qualification was inspected through retained captures; it was not rerun. No prior findings required disposition.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | **Flag:** unused parallel admission policy. |
   | ARCH-PURE | **Flag:** the declared pure admission helper is unused; codecs and identity qualification are properly pure. |
   | ARCH-PURPOSE | **Flag:** helper-read coverage is incomplete. |
   | ARCH-MOCK | **Pass:** stateful scanner controls use the production seam; native conformance evidence is retained. |
   | ARCH-CONSTRAINTS | **Flag:** socket fixture exceeds the pathname envelope. Runtime deadlines and candidate bounds are otherwise enforced. |
   | ARCH-SECURE | **Flag:** some reads admit unverified endpoints. Strict metadata decoding and unknown-field preservation are sound. |
   | ARCH-ORDER | **Pass:** creator phases, scoped permits, stale identity refusal and joined cancellation are explicit. Accepted interactive/death gaps are documented. |
   | ARCH-FUNERAL | **Pass:** fixed lock stripes and exact-generation marker retirement name resource lifetimes. |
   | ARCH-EVOLVE | **Pass:** additive top-level and nested fields survive codec round trips and compatible publication. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating the complete helper-read class and its regression controls. Reconcile the `PlanProbeAdmission` row with the chosen implementation, and correct the “all governing tasks satisfied” claim until README guidance is delivered.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"037e5dc7-d71f-41fb-84cc-8706e978e6da","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e29ceee8e8c2fbbb","reviewed_head":"50db793d663d30bd059e6358e2e54652395f8b31","slot":"pair:5","requested_at":"2026-10-24T11:44:01.332137Z"},"phase":"finished","pid":78764,"started_at":"2026-10-24T11:44:01.509011Z","finished_at":"2026-10-24T11:50:01.495891Z","session_id":"01a16db9-976a-73d1-b461-2171a522f9ba","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1m37.283524042s
head: "50db793d663d30bd059e6358e2e54652395f8b31"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000415
