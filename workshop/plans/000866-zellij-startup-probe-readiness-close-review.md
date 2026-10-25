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

---

## Re-review — 2026-10-24T05:09:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0859007d1bdee7f723eef4ba086e3ec18ff79312..3c25bc2be5b79f3575e6b91ab2abf8b33abaa9ed |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T05:09:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All four prior findings are addressed, including verified regression failures with the two behavioral fixes removed. Runtime and inventory checks passed. One plan-to-code mismatch remains: the governing resolver concept row names an unchanged file. This is blocking under the review’s explicit rule that every core-concept contradiction is Critical; no additional runtime defect was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The actual CLI regression covers all nine supported reads across four adapter spellings, plus qualified-read and explicit-mutation controls. It passes at head and fails for five non-list read types when the old classifier is restored in a disposable copy.
  - id: BR-2
    disposition: addressed
    note: |
      The fixture uses testpair.ShortRoot and the long-TMPDIR regression passes. Restoring os.MkdirTemp reproduces bind: invalid argument in that regression.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:884-897 now documents unmarked legacy sessions, rebuilding/relaunching cooperating clients, authorized recovery and accepted coverage limits, linking the atlas. These statements match acquireNative and the documented interactive/death exclusions.
  - id: BR-4
    disposition: addressed
    note: |
      PlanProbeAdmission is removed. The plan revision at line 364 supersedes its earlier row; production uses qualifyCandidates, directly exercised by the counted 8/16 tests.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      Governing resolver concept row names an unchanged implementation file
    detail: |
      workshop/plans/000866-zellij-startup-readiness-plan.md:255 lists cmd/internal/artifactpath/session_lock.go as modified for the new marker/namespace-lock resolvers, but that file is unchanged across the pinned range. ZellijNamespaceLease, ZellijReadyDirectory and ZellijReadyMarker are instead new in cmd/internal/artifactpath/zellij_readiness.go:12-44. Later revisions correct other implementation locations but not this row. Reconcile this row and append a specific Revisions correction. This is Critical under the supplied core-concepts cross-check rule; it is a traceability defect, not a runtime failure. ARCH-PURPOSE.
```

1. **Strengths**

   - Exact first-pane identity releases creator exclusion before AgentReady, avoiding the bootstrap rename dependency cycle.
   - Helper-read qualification is now checked through the production CLI route, with meaningful negative controls.
   - Strict codecs preserve nested unknown fields; retirement controls preserve successor generations and uncertain evidence.
   - README rollout guidance and atlas coverage limits agree with the implementation.

2. **Critical findings**

   - Plan line 255: replace the incorrect resolver location/status with `artifactpath/zellij_readiness.go` (**new**) and `manifest.go` (**modified**), and record the correction under `## Revisions`.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed complete `zellijquery`, `panebirth`, `artifactpath`, `pair-go` and `repolint` packages.
   - Passed query/panebirth race tests, focused launcher controls and native-admission tests across all seven migrated caller packages.
   - Passed `tests/zellij-trace-test.sh` and `make test-affected-inventory`.
   - Both behavioral corrections fail their regression tests when reverted in the disposable copy; that copy was removed.
   - Native qualification was inspected through retained captures and verified hashes, not rerun.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass — duplicate admission API removed; shared lease implementation retained. |
   | ARCH-PURE | Pass — codecs and identity policy remain pure; filesystem admission is separate. |
   | ARCH-PURPOSE | Flag — governing resolver traceability remains inconsistent. Runtime coverage matches the revised scope. |
   | ARCH-MOCK | Pass — stateful scanner controls exercise the production seam; native conformance evidence retained. |
   | ARCH-CONSTRAINTS | Pass — caller deadlines, candidate bounds and socket-safe fixtures verified. |
   | ARCH-SECURE | Pass — strict metadata handling and conservative read admission verified. |
   | ARCH-ORDER | Pass — scoped permits, exact identity fencing and joined cancellation are exercised. |
   | ARCH-FUNERAL | Pass — fixed lock stripes and exact-generation retirement define resource lifetimes. |
   | ARCH-EVOLVE | Pass — additive top-level and nested fields survive round trips. |

7. **Plan revision recommendation**

   Add a `## Revisions` entry explicitly superseding line 255 with the actual resolver file and its **new** status. This correction is prose-only; source inspection is sufficient verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"123e0044-84d3-45cc-a374-e7fc3877b8a0","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c43bff9a480c2b5f","reviewed_head":"3c25bc2be5b79f3575e6b91ab2abf8b33abaa9ed","slot":"pair:5","requested_at":"2026-10-24T12:05:23.802307Z"},"phase":"finished","pid":63142,"started_at":"2026-10-24T12:05:23.999499Z","finished_at":"2026-10-24T12:09:52.708006Z","session_id":"01a16dcd-2986-7543-a798-40fe67ccfbcc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.968876s
head: "3c25bc2be5b79f3575e6b91ab2abf8b33abaa9ed"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-24T05:15:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0859007d1bdee7f723eef4ba086e3ec18ff79312..bdbf7367ba61debbe2017f5cf5656e38a7c2e829 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T05:15:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the governing plan revisions. BR-5’s explicit correction now maps the resolvers and handoff to their actual files and change statuses. No new blocking findings emerged. Executed checks passed; retained native qualification supports the documented bounded protection and its accepted limits.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The passing actual-CLI regression covers all nine supported native reads, four adapter spellings, positive-marker admission and explicit-mutation controls; command.go applies qualification to that class.
  - id: BR-2
    disposition: addressed
    note: |
      The recovery fixture uses testpair.ShortRoot; the long inherited TMPDIR regression passes in the complete zellijquery package.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:884-897 documents legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with acquireNative and the interactive/death controls.
  - id: BR-4
    disposition: addressed
    note: |
      PlanProbeAdmission is absent from production. The revision at plan line 364 supersedes its earlier row; counted 8/16 tests exercise the production qualifyCandidates loop.
  - id: BR-5
    disposition: addressed
    note: |
      Plan lines 367-378 explicitly supersede both earlier resolver rows. Source and pinned name-status confirm zellij_readiness.go is new, manifest.go and session_handoff_os.go are modified, and session_lock.go and session_handoff.go are unchanged.
```

1. **Strengths**

   - Exact first-pane evidence releases namespace exclusion before AgentReady, avoiding the bootstrap rename dependency cycle.
   - Admission precedes injected runners, so tests and production share the guard.
   - Retirement preserves successor markers and removes only unchanged, confirmed generations.
   - README and atlas explain rollout requirements and residual risks.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed complete `zellijquery`, `panebirth`, `artifactpath`, `pair-go`, and `repolint` packages; targeted launcher controls; race tests for `zellijquery` and `panebirth`; Lua tracing tests; affected-reader inventory; and source whitespace checks.

   Inspected retained Claude/Codex native qualification and exact cleanup evidence. No fresh native fleet run or mutation testing was performed this round.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared lease and admission implementations.
   - **ARCH-PURE — pass:** codecs and identity qualification separate from filesystem/process effects.
   - **ARCH-PURPOSE — pass:** bounded caller inventory is migrated and enforced.
   - **ARCH-MOCK — pass:** stateful dependency controls share production seams; native conformance evidence exists.
   - **ARCH-CONSTRAINTS — pass:** deadlines, candidate counts, metadata size and capacity are bounded.
   - **ARCH-SECURE — pass:** strict metadata parsing, exact identities and argv preservation.
   - **ARCH-ORDER — pass:** stale publication, cancellation, owner death and successor retirement are exercised.
   - **ARCH-FUNERAL — pass:** stable lock bounds, marker retirement and joined child lifetimes are explicit.
   - **ARCH-EVOLVE — pass:** legacy compatibility and nested unknown-field preservation are tested.

7. **Plan revision recommendations:** None; the latest revisions reconcile the implementation map.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cd15e139-b7ee-420f-abbc-db816abf46c2","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5e464e32d6e377bc","reviewed_head":"bdbf7367ba61debbe2017f5cf5656e38a7c2e829","slot":"pair:5","requested_at":"2026-10-24T12:11:42.02409Z"},"phase":"finished","pid":58653,"started_at":"2026-10-24T12:11:42.139833Z","finished_at":"2026-10-24T12:15:00.400979Z","session_id":"01a16dd2-ee29-7ef1-b3fd-fc3631fb06fd","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.347180541s
head: "bdbf7367ba61debbe2017f5cf5656e38a7c2e829"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T06:11:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 77e6bdd8de0437b8a0124c2a00c1011b066422a2..0465f3846b724d833335a1540709f2d3e4327c40 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T06:11:50-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned interdiff fixes eager launcher lookup without changing the approved scope. Identity/RPC commands retain their argv; native commands still traverse the guarded helper. No new blocking defects found. No prior findings remain open; BR-1 through BR-5 retain their recorded dispositions.

```findings
{}
```

1. **Strengths**

   - `nvim/review/restore_client.lua:24` classifies commands before consulting the native launcher provider.
   - `nvim/review/restore_client_test.lua:6` covers identity, RPC, and native routing through the actual run method.
   - Updated integration fixtures execute the current Pair helper with private native namespaces.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed restore-client, restore-admission, DOM CLI, route/poke, workbench routing, changed repolint controls, tracked-source enforcement, and affected-reader inventory checks. Substituting the old eager lookup in memory makes the new regression fail. Whitespace checks passed; checkout remains clean. No full-suite or native-fleet rerun performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses existing adapter and test-binary builder.
   - **ARCH-PURE — pass:** routing remains in the thin process boundary; injected tests exercise it directly.
   - **ARCH-PURPOSE — pass:** covers both non-native command classes while preserving guarded native execution.
   - **ARCH-EVOLVE — pass:** no persisted-format semantics change.

   No new user-facing or architectural surface requires README or atlas changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"00d15af6-47e8-43b7-a559-82a493327a24","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/110b3f66cbea9318","reviewed_head":"0465f3846b724d833335a1540709f2d3e4327c40","slot":"pair:5","requested_at":"2026-10-24T13:09:37.978742Z"},"phase":"finished","pid":25720,"started_at":"2026-10-24T13:09:38.122543Z","finished_at":"2026-10-24T13:11:47.508785Z","session_id":"01a16e07-fb57-75a0-8cac-b98165870ab6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 27.301496166s
head: "0465f3846b724d833335a1540709f2d3e4327c40"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413

---

## Re-review — 2026-10-24T08:16:30-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4851d20b3a46d52348c455316bf7084348f4f4cf..bb896e034063c6891c3639ae7ee7ef1418eef261 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T08:16:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The interdiff repairs the four failing fixtures, preserves their effect assertions, and passes the complete review and Lua targets. Prior dispositions remain supported. One Important gap remains: the new Make selector can execute an installed Pair instead of a valid `PAIR_TEST_BINARY` whose filename differs from `pair`. No scope expansion was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The complete native-read classifier remains intact; TestInternalZellijCommandQualifiesEveryNativeRead passes at pinned head.
  - id: BR-2
    disposition: addressed
    note: |
      Recovery still uses testpair.ShortRoot; the long inherited TMPDIR regression passes.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:884-897 retains legacy-session refusal, rebuild/relaunch guidance, authorized recovery and coverage limits, consistent with the admission implementation.
  - id: BR-4
    disposition: addressed
    note: |
      PlanProbeAdmission remains absent. The plan identifies the production qualifyCandidates loop; its counted qualification tests pass.
  - id: BR-5
    disposition: addressed
    note: |
      Plan lines 367-378 retain the corrected resolver mapping, supported by constructors in artifactpath/zellij_readiness.go and bindings in manifest.go.
findings:
  - id: new
    severity: Important
    kind: defect
    family: exact-test-candidate-ownership
    title: |
      Make selects a candidate directory rather than the exact supplied executable
    detail: |
      Makefile.local:325 prefixes dirname(PAIR_TEST_BINARY), but native callers execute bare pair. The existing builder accepts any absolute executable filename, and README.md:2208 illustrates plain-pair. Running the new Make/bootstrap regression with shared/plain-pair as the candidate produces ['installed', 'installed'] for the two targets instead of ['shared', 'shared']; a sibling pair could likewise select a different build. ARCH-PURPOSE: create a suite-owned pair alias to the exact candidate using the existing builder, and extend the regression to cover candidate basenames other than pair.
```

1. **Strengths**

   - All four changed fixtures retain their original query, delivery, trace and overlay assertions.
   - Fixtures reuse `pair-test-binary.sh` and own private native namespaces.
   - The Make regression exercises real Make export and bootstrap behavior. Removing the PATH fix makes it fail with installed-Pair selection.

2. **Critical findings:** None.

3. **Important findings**

   - [Makefile.local:325](/Volumes/TestScratch/review-checkouts/154ac811aefe243f/Makefile.local:325): resolve the exact supplied executable, as detailed above. The regression currently assumes every candidate is named `pair`.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed complete shared-candidate `test-review` and `test-lua`, both environment-test invocations, affected-reader inventory, focused prior-finding regressions, shell syntax and diff whitespace checks. The renamed-candidate counterexample fails reproducibly. Real-docflow smoke skipped because its external script was unavailable; no native-fleet rerun was performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** fixtures reuse the existing candidate builder.
   - **ARCH-PURE — pass:** changes remain in test orchestration and integration fixtures.
   - **ARCH-PURPOSE — flag:** directory selection does not guarantee exact candidate ownership.
   - **ARCH-EVOLVE — pass:** no persisted-format semantics change.

   This window adds no user-facing or architectural surface requiring README or atlas updates.

7. **Plan revision recommendations**

   Append a `## Revisions` correction stating that supplied candidates resolve through an exact executable alias, independently of filename, with a regression covering a differently named candidate ahead of an incompatible installed Pair.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d4465ca7-60a6-433d-9d80-c6c8a2123a77","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/154ac811aefe243f","reviewed_head":"bb896e034063c6891c3639ae7ee7ef1418eef261","slot":"pair:5","requested_at":"2026-10-24T15:11:46.355224Z"},"phase":"finished","pid":84044,"started_at":"2026-10-24T15:11:46.485313Z","finished_at":"2026-10-24T15:16:21.773702Z","session_id":"01a16e77-cbea-79a0-bddd-b83b217b261c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.143508167s
head: "bb896e034063c6891c3639ae7ee7ef1418eef261"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T09:19:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 29fc24fe76967b6495959485a1eed02a5377ca03..456228a42c042e58b6a068fce31a913526e93a00 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T09:19:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-6 is addressed: both public test targets now alias the exact supplied executable through the existing builder. The regression passes with `plain-pair` beside a wrong sibling `pair`; restoring directory-only selection in a temporary fixture makes both targets select the wrong sibling and fails the test. No new blocking defect or scope expansion was found in the interdiff.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Makefile.local:329 routes both targets through with-pair-test-binary.sh, which aliases the exact candidate and exports its PATH. The regression at tests/make-test-env-test.sh:79 passes; a directory-only mutation fails with ['wrong-sibling', 'wrong-sibling'].
```

1. **Strengths**

   - Exact executable ownership reuses the shared builder and cleanup lifecycle (`tests/lib/with-pair-test-binary.sh:10`).
   - The Make ownership audit follows the wrapped recipes and rejects an unclaimed inner script (`tests/test-affected-test.py:198`).
   - Provider qualification preserves pinned source and module metadata together.
   - Copy-on-select completion now belongs only to the orchestrator; the real reset-helper control exercises that distinction.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed `make-test-env-test.sh` under both outer Make configurations, all 14 affected-runner tests, all four provider qualification tests with the pinned dependency, `go test ./cmd/internal/testpair`, copy-on-select, and Zellij tracing. The candidate-selection mutation failed as expected. Diff whitespace checks passed; the checkout remained unchanged. Full native startup qualification was not repeated for this test-orchestration interdiff.

6. **Architectural notes**

   - **ARCH-DRY — pass:** candidate placement remains centralized in the existing builder.
   - **ARCH-PURE — pass:** changes remain in test orchestration and IO fixtures; no production business logic moved into IO.
   - **ARCH-PURPOSE — pass:** both affected targets enforce exact candidate ownership, including arbitrary executable basenames.
   - **ARCH-EVOLVE — pass:** no persisted format or rewrite contract changes.

7. **Plan revision recommendations:** None. The appended corrections accurately describe the implementation. This window introduces no new user-facing or architectural surface requiring additional README or atlas changes.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ef69f78a-f7e0-4d79-a720-280dd9334065","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/3a3a88c83cff4ad0","reviewed_head":"456228a42c042e58b6a068fce31a913526e93a00","slot":"pair:5","requested_at":"2026-10-24T16:17:48.701721Z"},"phase":"finished","pid":99461,"started_at":"2026-10-24T16:17:48.808862Z","finished_at":"2026-10-24T16:19:39.63765Z","session_id":"01a16eb4-425b-79c2-b605-d06a19c34953","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.103957125s
head: "456228a42c042e58b6a068fce31a913526e93a00"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T11:56:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | b118e76f4fc1748df91dfd878ef4c6e5f150543d..abd991c4b2486fc9fafad261550ad79de9d614e6 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T11:56:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The fixture repairs have meaningful regression evidence, and focused tests pass. However, this interdiff adds a production startup diagnostic that none of BR-1–BR-6 requested. The narrow-review contract requires a full close review for that scope expansion. No prior findings remain open; their recorded dispositions remain unchanged.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: post-close-scope-control
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      cmd/internal/launcher/session_handoff_os.go:197 adds a new phase/session error wrapper, documented as new behavior in workshop/plans/000866-zellij-startup-readiness-plan.md:404. None of BR-1–BR-6 requested this runtime diagnostic; its regression fails against the base implementation, confirming an executable change. Run a full sdlc close covering the expanded scope, or remove this addition from the narrow fix bundle.
```

1. **Strengths**

   - The registration fixture reuses production `PublishPaneReady` before starting its agent.
   - Namespace isolation is tested against an owned poisoned inherited namespace.
   - Effect coverage now recognizes `RunNativeCommand` and rejects unclassified uses.

2. **Critical findings:** None.

3. **Important findings:** The post-close scope expansion above requires a full review. This is a boundary-scope finding; no correctness defect was identified in the diagnostic itself.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused launcher, Couchcore, effect-coverage, and fixture-decoder tests passed. Temporary Go overlays confirmed regressions fail without the diagnostic, pane publisher, and namespace binding. Inventory and diff whitespace checks passed. The complete launcher package passed; the broader package sweep was stopped during Couchcore and supplies no full-suite pass claim. Checkout unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses the existing publisher and fixture serializer.
   - **ARCH-PURE — pass:** changes remain in IO fixtures and the existing admission boundary.
   - **ARCH-PURPOSE — pass:** fixture repairs exercise actual readiness ordering.
   - **ARCH-EVOLVE — pass:** no persisted-format semantics changed.

7. **Plan revision recommendations**

   Add a `## Revisions` entry identifying the diagnostic as post-close scope and recording the required full-close review.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6fd7dee6-abe7-493d-b4b3-a18a32dd7b49","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"fix-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8a0049f20efdc5c2","reviewed_head":"abd991c4b2486fc9fafad261550ad79de9d614e6","slot":"pair:5","requested_at":"2026-10-24T18:48:19.838584Z"},"phase":"finished","pid":171,"started_at":"2026-10-24T18:48:20.002599Z","finished_at":"2026-10-24T18:56:48.699083Z","session_id":"01a16f3e-0e84-7191-8a48-1b1eb3cb934d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.828055s
head: "abd991c4b2486fc9fafad261550ad79de9d614e6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-24T12:01:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0859007d1bdee7f723eef4ba086e3ec18ff79312..f443e10748b4febc409571bd29ace9835dec3369 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T12:01:59-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the governing plan revisions. No new blocking defects were found. BR-7 is addressed by this full whole-issue review, including the added startup diagnostic; its regression fails when that diagnostic is removed through a temporary overlay.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The complete native-read classifier and real CLI admission regression remain intact; the pair-go package passes.
  - id: BR-2
    disposition: addressed
    note: |
      Recovery uses testpair.ShortRoot, and the query package passes its long inherited TMPDIR regression.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:884-897 documents legacy-session refusal, coordinated rollout, explicit recovery and accepted coverage limits consistently with the implementation.
  - id: BR-4
    disposition: addressed
    note: |
      PlanProbeAdmission is absent; production uses qualifyCandidates, exercised by the passing counted-work controls.
  - id: BR-5
    disposition: addressed
    note: |
      The authoritative plan revision at lines 367-378 matches the pinned file statuses and actual resolver and handoff implementations.
  - id: BR-6
    disposition: addressed
    note: |
      Both Make targets use the exact-candidate alias owner. The Make/bootstrap regression passes with plain-pair beside an incompatible sibling pair.
  - id: BR-7
    disposition: addressed
    note: |
      This full whole-issue close review covers the expanded diagnostic scope recorded at plan lines 400-408. TestCreatorMissingPaneReadinessNamesSessionAtBound passes at head and fails with the diagnostic wrapper removed through a temporary Go overlay.
```

1. **Strengths**

   - Exact first-pane identity releases exclusion independently of AgentReady, avoiding the bootstrap rename deadlock (`cmd/internal/launcher/session_handoff_os.go:205`).
   - Admission retains the namespace lease through qualification and joined native execution; unknown evidence refuses conservatively.
   - Strict codecs preserve nested unknown fields, with round-trip and malformed-input coverage.
   - Retirement preserves successor generations and requires existing authorized lifecycle confirmation.
   - README and atlas describe the new surfaces and their accepted limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed complete packages: `zellijquery`, `panebirth`, `artifactpath`, `launcher`, `pair-go`, and `internal/repolint`. Query and codec race checks passed. Native-admission regressions passed across all seven migrated caller packages. Make environment/candidate checks and `make test-affected-inventory` passed.

   The diagnostic removal control failed as expected. Retained Claude/Codex native qualification results were inspected, and their capture SHA256 matched `016720b0388837bb50cb1faa75a448d6a1033575005e6591885107ba3e2676e1`. Native qualification and the entire repository suite were not rerun during this review. The checkout remains unchanged.

6. **Architectural notes**

   | Principle | Result | Evidence |
   |---|---|---|
   | ARCH-DRY | Pass | Shared lease implementation and one production qualification loop. |
   | ARCH-PURE | Pass | Identity/codec logic is directly tested without IO; filesystem/process work stays at integration seams. |
   | ARCH-PURPOSE | Pass | Covers global native scanning across migrated bounded callers. |
   | ARCH-MOCK | Pass | Stateful initialization controls share production admission; retained native qualification checks conformance. |
   | ARCH-CONSTRAINTS | Pass | Existing deadlines include lock waits; capacity, record size and counted work are bounded. |
   | ARCH-SECURE | Pass | Strict metadata parsing, exact identities, inode checks and private test roots. |
   | ARCH-ORDER | Pass | Encapsulated creator phases and controlled stale-publication, cancellation, owner-death and successor tests. |
   | ARCH-FUNERAL | Pass | Fixed lock stripes, capped markers, exact retirement and joined helper lifetimes. |
   | ARCH-EVOLVE | Pass | Additive records preserve unknown members and reject malformed known fields and duplicates. |

7. **Plan revision recommendations:** None. Existing superseding revisions reconcile the implementation locations, accepted limits, rollout requirements and expanded review scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d35ecbff-f66a-494a-b2d1-df3abeb3096b","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"milestone-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ab6df6f7c13e4c05","reviewed_head":"f443e10748b4febc409571bd29ace9835dec3369","slot":"pair:5","requested_at":"2026-10-24T18:58:39.592094Z"},"phase":"finished","pid":32410,"started_at":"2026-10-24T18:58:39.715448Z","finished_at":"2026-10-24T19:01:54.589022Z","session_id":"01a16f47-8346-7ab3-9015-39cb423e050a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.548567s
head: "f443e10748b4febc409571bd29ace9835dec3369"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-24T18:05:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 866 — Wait for Zellij readiness before startup probe clients |
| repo | pair |
| issue file | workshop/issues/000866-zellij-startup-probe-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0859007d1bdee7f723eef4ba086e3ec18ff79312..2af137563f9f375555febae4129e2eada17a78f7 |
| command | sdlc close --issue 866 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T18:05:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the governing lease-only revision: live creators exclude bounded native commands, while legacy sessions remain observable without receipts. No new blocking findings. This verdict covers the reviewed range; the separately recorded suite and integration gates remain required.

```findings
{}
```

1. **Strengths**
   - Shared admission preserves caller deadlines and validates scoped permits (`cmd/internal/zellijquery/readiness_os.go:171`).
   - Exact first-pane identity releases exclusion before the independent AgentReady phase (`cmd/internal/launcher/session_handoff_os.go:226`).
   - Additive codecs preserve nested unknown fields while rejecting malformed known fields and duplicate keys.
   - Source guards include negative controls for Go, Lua, shell and layout bypasses. README and atlas explain rollout and accepted limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   
   Passed complete query, panebirth, artifactpath, repolint and pair-go packages; targeted launcher controls; native-mutation regressions across seven caller packages; Lua restore-client tests; and `make test-affected-inventory`.

   Independently inspected and hash-verified the retained warm-upgrade driver and native log. Both agents preserved their original bindings and socket identities without receipt synthesis, with exact cleanup. Native qualification was not rerun during this review. Previously disposed findings remain disposed.

6. **Architecture**
   - **ARCH-DRY — Pass:** one shared lease and admission implementation.
   - **ARCH-PURE — Pass:** codecs and identity validation remain separate from IO.
   - **ARCH-PURPOSE — Pass:** creator exclusion and legacy compatibility are both delivered.
   - **ARCH-MOCK — Pass:** stateful controls exercise the production seam; retained native evidence checks real behavior.
   - **ARCH-CONSTRAINTS — Pass:** existing deadlines, bounded records and capacity limits remain enforced.
   - **ARCH-SECURE — Pass:** exact identities, bounded metadata reads and argv-based execution protect the boundaries.
   - **ARCH-ORDER — Pass:** controls cover stale publication, cancellation, owner death and release before AgentReady.
   - **ARCH-FUNERAL — Pass:** fixed lock stripes and exact-generation marker retirement define artifact lifetimes.
   - **ARCH-EVOLVE — Pass:** additive round trips preserve unknown members with strict validation.

7. **Plan revision recommendations:** None. The appended governing revisions reconcile the implementation locations, admission policy and accepted coverage limits.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"84022fc7-53fa-4aa7-a385-f73130744617","repository":"github.com/xianxu/pair","issue_id":"000866","recipe":"milestone-review","round":9,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/836662232fea2cb2","reviewed_head":"2af137563f9f375555febae4129e2eada17a78f7","slot":"pair:5","requested_at":"2026-10-25T00:58:10.5387Z"},"phase":"finished","pid":32626,"started_at":"2026-10-25T00:58:10.672629Z","finished_at":"2026-10-25T01:00:57.870273Z","session_id":"01a17090-aa45-7a72-80c6-45c92495ce40","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.89607625s
head: "2af137563f9f375555febae4129e2eada17a78f7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
