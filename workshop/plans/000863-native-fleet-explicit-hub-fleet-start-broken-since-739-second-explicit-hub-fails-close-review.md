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
