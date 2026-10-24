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
