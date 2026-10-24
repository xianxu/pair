# Boundary Review — pair#863 (whole-issue close)

| field | value |
|-------|-------|
| issue | 863 — Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start |
| repo | pair |
| issue file | workshop/issues/000863-native-fleet-explicit-hub-fleet-start-broken-since-739-second-explicit-hub-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..68bc00e8c2df410b5d53807a9733ef4bb6b0f3ee |
| command | sdlc close --issue 863 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T02:04:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The fresh-start classification, target routing, and empty-result adoption fix are well focused. However, automatic Fresh can replace retained numbered-slot history that the initial namespace snapshot missed. Native acceptance also remains incomplete: one of the issue’s required ExplicitHub tests failed.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: automatic-launch-requires-empty-authority
    title: |
      Automatic fresh startup can replace an existing stopped conversation
    detail: |
      cmd/internal/couchcore/startup.go:147 calls StartFreshSlot, which passes requireEmpty=false (slotrecovery.go:334). The initial observation only discovers local stores for manifest-enrolled repositories (threadstore_location.go:44), so an empty namespace can classify fresh despite a configured numbered hub retaining .couch/thread.json. Startup then archives that stopped conversation and creates another without an explicit Fresh gesture. ARCH-SECURE and ARCH-ORDER: inspect configured local history before granting fresh authority, and enforce an empty-current precondition at mutation time. Add regressions for unenrolled retained history and history appearing after observation.
  - id: new
    severity: Important
    kind: unverifiable
    family: native-acceptance-evidence
    title: |
      The required ExplicitHub native acceptance matrix is not established
    detail: |
      Ran PAIR_NATIVE_COUCH_FLEET=1 SDLC_DEP_SUBSTRATE=/Volumes/TestScratch/review-checkouts/5562c74d00a70f30 go test ./cmd/couch -run '^TestDisposable(FreshHubCaller|EditedRestart|StartupArguments|CouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles)$' -count=1 -timeout=12m. TestDisposableStartupArguments/claude failed at startup_arguments_fleet_test.go:55 with switch-agent receipt Status:failed, Detail:attach terminal input has ended; wrapper evidence also reported terminal listing omitted a live server. Its Codex case was consequently skipped. ARCH-PURPOSE: provide passing acceptance for both personalities, or measured evidence classifying this failure as independent of the change before claiming the issue's Done-when condition.
```

1. **Strengths**

   - `StartResult.Started` now requires a handle; tests independently vary record and handle presence.
   - Catalog-backed tests cover both ordinary primary and numbered-slot routing.
   - Fresh launches retain the initial automation policy. Native two-hub acceptance passed for Claude and Codex.
   - README, atlas, and the embedded Couch skill describe the changed startup behavior.

2. **Critical findings**

   `startup.go:147` needs an empty-slot guard. Existing tests confirm that `StartFreshSlot` intentionally replaces stopped conversations, so it grants broader authority than automatic startup should possess. Inspect local history read-only before classification, then revalidate emptiness before committing a launch.

3. **Important findings**

   The native matrix failed after successful initial startup, during worker switch-agent adoption. This is an acceptance-evidence gap; the run does not establish that this diff caused the failure.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Targeted tests passed across couchcore, couchcmd, couchtty, and testcouch. The affected-reader inventory passed. Native FreshHubCaller, EditedRestart, and dirty-file/force acceptance passed; StartupArguments failed as detailed above.

   The plan’s Task 2 also overstates deterministic coverage: fresh queue tests inspect prepared calls and invoke completion callbacks directly. The held console tests use surviving-session startup. Add fresh-path tests with controlled launch/adoption completion, cancellation, and changed local history.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared target-aware startup helper.
   - **ARCH-PURE — pass:** classification and form admission remain directly testable.
   - **ARCH-PURPOSE — flag:** required native acceptance remains incomplete.
   - **ARCH-MOCK — pass:** existing stateful process/store seams are reused.
   - **ARCH-CONSTRAINTS — pass:** serial extra-hub scheduling and existing budgets remain.
   - **ARCH-SECURE — flag:** incomplete inventory is treated as proof of absent history.
   - **ARCH-ORDER — flag:** initial fresh permission can authorize replacement of subsequently observed history.
   - **ARCH-FUNERAL — pass:** existing lifecycle cleanup owns launched processes and records.
   - **ARCH-EVOLVE — pass:** no persisted-format change.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying that fresh authority includes configured local stores independently of namespace enrollment, and that automatic numbered-slot launch requires empty-current revalidation. Correct Task 2’s coverage claim until controlled fresh-path tests exist, and record the failed native acceptance result pending classification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ea273863-bf00-4614-8b89-8827b75709b6","repository":"github.com/xianxu/pair","issue_id":"000863","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8140c6109d7070fe","reviewed_head":"68bc00e8c2df410b5d53807a9733ef4bb6b0f3ee","slot":"pair:7","requested_at":"2026-10-24T08:56:28.831565Z"},"phase":"finished","pid":93491,"started_at":"2026-10-24T08:56:28.99893Z","finished_at":"2026-10-24T09:04:06.01835Z","session_id":"01a16d20-33e9-7433-8e9e-579382765b6d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.00071s
head: "68bc00e8c2df410b5d53807a9733ef4bb6b0f3ee"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000408

---

## Re-review — 2026-10-24T03:39:59-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 863 — Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start |
| repo | pair |
| issue file | workshop/issues/000863-native-fleet-explicit-hub-fleet-start-broken-since-739-second-explicit-hub-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..71bb4b2e719f929a3873145020110dd138634709 |
| command | sdlc close --issue 863 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T03:39:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is addressed: the history checks and locked creation guards are reachable, and their regression tests fail when those protections are disabled. One new correctness bug blocks shipping: failure during post-commit discovery/enrollment strands a numbered hub’s start transaction. BR-2 also remains unresolved; the retained native captures do not establish the required matrix or independence of the original failure.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Configured unenrolled history is inspected, and ordinary/numbered creation rechecks emptiness under publication locks. Targeted tests pass; disabling these protections in a disposable pinned-head copy makes the observation and late-history regressions fail.
  - id: BR-2
    disposition: not-addressed
    note: |
      Inspected branch-required-native-7a71f335.log and the pinned-main comparison captures. StartupArguments/claude fails and skips Codex; EditedRestart/codex also fails in the matrix. The main StartupArguments failure has a different signature from the original attach-input-ended failure, while later passing comparisons do not establish its independence. Passing two-personality acceptance or measured classification of the outstanding failures is still needed; the recorded close approval and follow-up issues do not supply that evidence.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: committed-starts-require-terminal-disposition
    title: |
      Post-commit discovery or enrollment failure strands a creating hub transaction
    detail: |
      cmd/internal/couchcore/slotrecovery.go:491-499 returns directly after claimFreshRecord has published the local creating incarnation and native address claim. Discovery failure, identity disagreement, or enrollment failure bypasses launchClaimedThread and its rollback handling. In a disposable pinned-head copy, injecting a discovery failure after artifact Claim reproduced an error with zero launched children but a persisted creating transaction owned by the live supervisor. Finalize every post-publication failure through exact-record/nonce cleanup that also works before enrollment, and add regressions covering these exits. ARCH-ORDER and ARCH-FUNERAL.
```

1. **Strengths**

   - The shared history reader covers unenrolled local current records, archives, recovery backups, and legacy global records.
   - Empty-only publication shares the existing journal machinery and checks both stores while holding the established lock order.
   - Real `Console.Run` tests hold launch and adoption separately, verify serial extra-hub startup, and confirm cancellation joins.
   - README, atlas, and the embedded Couch skill now describe fresh startup and automation remaining off.

2. **Critical findings**

   - **New: stranded start claim**, [slotrecovery.go:491](/Volumes/TestScratch/review-checkouts/a1b688d778015779/cmd/internal/couchcore/slotrecovery.go:491). The three new error exits occur after durable publication but before lifecycle cleanup takes ownership. Handle these exits as confirmed pre-launch failures against the exact local transaction; preserve any uncertain enrollment progress separately.

3. **Important findings**

   - **BR-2 remains open.** The retained matrix passes fresh two-hub callers and dirty/force scenarios on both personalities, but fails StartupArguments/Claude and skips its Codex cohort. The documented comparisons do not resolve the original failure’s cause. See [the plan’s acceptance disposition](/Volumes/TestScratch/review-checkouts/a1b688d778015779/workshop/plans/000863-fresh-hub-startup-plan.md:160).

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Targeted startup tests passed across `couchcore`, `couchcmd`, `couchtty`, and `testcouch`.
   - Additional store, tag, Fresh-slot, recovery, and transaction tests passed.
   - BR-1 negative controls failed as expected in a temporary pinned-head copy.
   - A temporary injected-failure regression reproduced the new stranded transaction.
   - Native evidence was inspected; no additional native run was performed. Temporary copies were removed, and the reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared history checks and journal publication avoid parallel implementations.
   - **ARCH-PURE — pass:** classification remains pure; storage and launch effects stay at integration boundaries.
   - **ARCH-PURPOSE — flag:** BR-2’s acceptance requirement remains unestablished.
   - **ARCH-MOCK — pass:** orchestration uses the production dispatcher with stateful process fakes and controllable ordering.
   - **ARCH-CONSTRAINTS — pass:** directory inspection is bounded and context-aware; launch deadlines were not increased.
   - **ARCH-SECURE — pass:** retained or unsafe history denies creation authority.
   - **ARCH-ORDER — flag:** post-publication failures escape without a terminal transaction disposition.
   - **ARCH-FUNERAL — flag:** those exits leave a durable claim without completion or cleanup.
   - **ARCH-EVOLVE — pass:** no persisted format change was introduced.

   The plan’s named entities exist at their stated locations; the revised integration boundaries match the implementation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating every failure between successful claim publication and launch ownership, specifying exact local cleanup and tests for discovery failure, identity change, enrollment failure, and cancellation. Retain BR-2 as incomplete until its evidence requirement is satisfied.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7afa2210-0e71-49d9-bf5a-37a5751ec1f2","repository":"github.com/xianxu/pair","issue_id":"000863","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a1b688d778015779","reviewed_head":"71bb4b2e719f929a3873145020110dd138634709","slot":"pair:7","requested_at":"2026-10-24T10:35:51.33727Z"},"phase":"finished","pid":68024,"started_at":"2026-10-24T10:35:51.432715Z","finished_at":"2026-10-24T10:39:57.239786Z","session_id":"01a16d7b-2e29-7410-b27e-d01255a430f4","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.041710583s
head: "71bb4b2e719f929a3873145020110dd138634709"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T03:58:58-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 863 — Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start |
| repo | pair |
| issue file | workshop/issues/000863-native-fleet-explicit-hub-fleet-start-broken-since-739-second-explicit-hub-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..3a25379a588a7d6eff2fc7d4d918cea9e4e67de6 |
| command | sdlc close --issue 863 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T03:58:58-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned change implements fresh startup with empty-only creation authority and fixes BR-3’s stranded claim. Focused tests and inventory checks passed; independently removing BR-3’s cleanup made its regression fail. BR-2 is withdrawn under the documented narrowed acceptance scope. The retained evidence supports that scope, but does not establish a complete native matrix pass or explain the original attach failure.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained as addressed. Empty-only publication guards remain in both target paths; focused history, appeared-history, and FreshHub regression tests passed.
  - id: BR-2
    disposition: withdrawn
    note: |
      Withdrawn under the pinned plan's explicit narrowed acceptance revision. Inspected captures confirm FreshHubCaller and dirty/force pass for both personalities, matching lib.rs:1462 Zellij panics on baseline and branch, and passing baseline/instrumented-branch Codex comparisons. These measurements do not explain the original attach-input-ended signature or establish independence of the single idle refusal; the revised plan preserves those limits rather than claiming full-matrix acceptance.
  - id: BR-3
    disposition: addressed
    note: |
      enrollFreshStartupClaim defers exact-local-store rollback across discovery, identity, enrollment, and cancellation failures. The focused cohort passed. An independent Go overlay removing only that defer made TestFreshStartupPostClaimDiscoveryFailureRollsBack fail with a retained creating record and native marker.
```

1. **Strengths**

   - [slotrecovery.go:524](cmd/internal/couchcore/slotrecovery.go:524) gives the published claim one cleanup owner before launch handoff, preserving the original error alongside cleanup failures.
   - [startup_creation.go:15](cmd/internal/couchcore/startup_creation.go:15) checks history while holding publication locks, closing the gap between observation and mutation.
   - Cleanup regressions exercise real local storage, successor preservation, enrollment publication failure, and successful ownership transfer.
   - README, atlas, and the embedded Couch skill describe fresh startup and conservative recovery behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Focused startup, fresh creation, adoption, orchestration, and harness tests passed across `couchcore`, `couchcmd`, `couchtty`, and `testcouch`.
   - `make test-affected-inventory` passed.
   - BR-3’s independent negative control failed as expected; repository source remained unchanged.
   - Native evidence captures and hashes were inspected. Native scenarios and the full suite were not rerun in this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing journal, reservation, and rollback mechanisms are reused.
   - **ARCH-PURE — pass:** classification stays pure; storage and lifecycle effects remain at integration boundaries.
   - **ARCH-PURPOSE — pass:** both ordinary and numbered targets enforce empty-only startup.
   - **ARCH-MOCK — pass:** deterministic tests use stateful boundary fakes; retained native tests exercise the shared production path.
   - **ARCH-CONSTRAINTS — pass:** history traversal has entry/context bounds; extra hubs launch serially.
   - **ARCH-SECURE — pass:** unsafe paths, pending journals, and unreadable evidence deny creation.
   - **ARCH-ORDER — pass:** publication, enrollment, and launch ownership have explicit handoff and tested failure paths.
   - **ARCH-FUNERAL — pass:** failed pre-handoff claims receive exact cleanup; uncertain successors retain occupancy.
   - **ARCH-EVOLVE — pass:** no persisted format change; extracted writers retain existing encoding behavior.

7. **Plan revision recommendations:** None. Existing revisions cover the final implementation and narrowed native acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ca899fb3-0809-4535-b97e-a29140749b2a","repository":"github.com/xianxu/pair","issue_id":"000863","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/7736e71951190822","reviewed_head":"3a25379a588a7d6eff2fc7d4d918cea9e4e67de6","slot":"pair:7","requested_at":"2026-10-24T10:52:34.151222Z"},"phase":"finished","pid":76258,"started_at":"2026-10-24T10:52:34.308649Z","finished_at":"2026-10-24T10:55:04.405115Z","session_id":"01a16d8a-7c2b-79b2-9011-f6143591e671","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.754480542s
head: "3a25379a588a7d6eff2fc7d4d918cea9e4e67de6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T04:56:39-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 863 — Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start |
| repo | pair |
| issue file | workshop/issues/000863-native-fleet-explicit-hub-fleet-start-broken-since-739-second-explicit-hub-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | 060628d56ad8f81df7fad0709aa0552170370484..db3c572e729063328540b68d194bed58547055a1 |
| command | sdlc close --issue 863 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T04:56:39-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff repairs stale test fixtures and expectations without changing production behavior or expanding scope. Targeted tests, installed-command scenarios, and the reader inventory passed. No new blocking defects were found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained as addressed. Empty-only creation guards are unchanged; FreshHub history-preservation regressions passed.
  - id: BR-2
    disposition: withdrawn
    note: |
      Retained as withdrawn under the previously reviewed narrowed acceptance revision. This interdiff adds no full native-matrix acceptance claim.
  - id: BR-3
    disposition: addressed
    note: |
      Retained as addressed. The cleanup implementation and regression tests are unchanged; post-claim failure, uncertainty, and ownership-transfer tests passed.
```

1. **Strengths**

   - `cmd/couch/main_test.go:22` now independently exercises retained-history shutdown and zero-history automatic startup, preserving launch-count, attachment, and pipe-refusal assertions.
   - `cmd/internal/couchtty/console_relaunch_chord_test.go:382` supplies handles to positive adoption fixtures while explicitly rejecting metadata-only and empty results.
   - `cmd/internal/couchcmd/hubs_startup_test.go:625` preserves the complete joined diagnostic and existing no-effects assertions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - All changed command and console tests passed.
   - Both installed-command startup scenarios passed.
   - Focused `StartResult`, FreshHub, and post-claim cleanup regressions passed.
   - `make test-affected-inventory` and `git diff --check` passed.
   - Full-suite and native acceptance runs were not repeated. The tracker accurately retains the failed earlier suite and pending final verification.

6. **Architectural notes**

   - **ARCH-DRY — pass:** the installed-command scenarios share one fixture helper.
   - **ARCH-PURE — pass:** IO remains in integration fixtures; pure adoption policy is tested directly.
   - **ARCH-PURPOSE — pass:** corrections preserve the original assertions and add negative controls.
   - **ARCH-EVOLVE — pass:** no persisted format or reader/writer behavior changes.

   No new architectural or user-facing surface requires atlas or README updates.

7. **Plan revision recommendations:** None. The added revision records fixture corrections without changing the approved contract.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8ccea2bc-e2c4-4379-bda4-547adc436a58","repository":"github.com/xianxu/pair","issue_id":"000863","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1b990a29c5239c8a","reviewed_head":"db3c572e729063328540b68d194bed58547055a1","slot":"pair:7","requested_at":"2026-10-24T11:54:16.910347Z"},"phase":"finished","pid":47527,"started_at":"2026-10-24T11:54:17.052861Z","finished_at":"2026-10-24T11:56:11.650163Z","session_id":"01a16dc2-fbed-7623-8343-58145074d7da","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.352540417s
head: "db3c572e729063328540b68d194bed58547055a1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T07:02:46-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 863 — Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start |
| repo | pair |
| issue file | workshop/issues/000863-native-fleet-explicit-hub-fleet-start-broken-since-739-second-explicit-hub-fails.md |
| boundary | whole-issue close |
| milestone | — |
| window | a02d54abfccd88b47c8f4063bd7e9b269c023905..8b13b536b24b1a30bf122d7c40acbdc4dd37aafe |
| command | sdlc close --issue 863 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T07:02:46-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides’ intent in every listed conflict resolution. Updated tests retain meaningful behavioral assertions, reader certificates match their sources, and focused checks passed. No new blocking defect or scope expansion was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained as addressed. Empty-only creation guards and regression tests are unchanged; retained-history and history-appearance tests passed.
  - id: BR-2
    disposition: withdrawn
    note: |
      Retained as withdrawn under the previously reviewed acceptance revision. This window changes neither the acceptance contract nor its documented native-evidence limits.
  - id: BR-3
    disposition: addressed
    note: |
      Retained as addressed. Exact-store rollback and its regressions are unchanged; post-claim failure, uncertainty, and ownership-transfer tests passed.
```

1. **Strengths**

   - `cmd/internal/couchcmd/hubs.go:115` preserves warning attention routing while forwarding the original startup observation.
   - `cmd/internal/couchcmd/hubs_startup_test.go:695` checks both visible diagnostic causes while retaining exact clipboard and no-effects assertions.
   - Fleet conflict resolutions preserve `StartCold` behavior; all three refreshed reader certificates match their source hashes.
   - `atlas/index.md` retains both fresh-start and asleep/wake descriptions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:

   - Affected startup, attention, and summary tests: 8.534s.
   - Fresh orchestration, cancellation, and known-shutdown tests: 16.024s.
   - History-preservation and post-claim cleanup regressions: 35.062s.
   - Complete `testcouch` package: 52.607s.
   - Reader inventory and diff whitespace checks.

   Full-suite, native acceptance, and fix-removal controls were not repeated.

6. **Architectural notes**

   - **ARCH-DRY — pass:** resolutions retain shared orchestration and fleet helpers.
   - **ARCH-PURE — pass:** integration seams remain unchanged; no business logic is added to IO.
   - **ARCH-PURPOSE — pass:** resolutions preserve fresh-start authority and warning attention behavior.
   - **ARCH-EVOLVE — pass:** no persisted format or reader/writer semantics change.

   No new surface requires additional README or atlas documentation.

7. **Plan revision recommendations:** None. The durable implementation plan is unchanged.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c507b4ef-b524-4e88-a82d-d1a4f5ac1c4c","repository":"github.com/xianxu/pair","issue_id":"000863","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/ec9601e1806bd23d","reviewed_head":"8b13b536b24b1a30bf122d7c40acbdc4dd37aafe","slot":"pair:7","requested_at":"2026-10-24T14:00:29.43935Z"},"phase":"finished","pid":76103,"started_at":"2026-10-24T14:00:29.623831Z","finished_at":"2026-10-24T14:02:42.434109Z","session_id":"01a16e36-8a6d-7ca2-b42d-2c6604eeb1d3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.554193208s
head: "8b13b536b24b1a30bf122d7c40acbdc4dd37aafe"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
