# Boundary Review — pair#860 (whole-issue close)

| field | value |
|-------|-------|
| issue | 860 — Multi-hub test fixture mutates authority after message readers start |
| repo | pair |
| issue file | workshop/issues/000860-multi-hub-fixture-authority-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..17ce2edb890108642629051661d7866b103882a0 |
| command | sdlc close --issue 860 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T00:49:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main multi-hub repair is structurally sound, and the selected changed tests passed repeated race testing. However, the child-proof extension leaves an executable sibling of the same race unfixed: native-exit tests still replace authority after wrappers connect. Done-when also omits the added child-proof scope.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: child-authority-before-readers
    title: |
      ARCH-PURPOSE: child-proof publication sweep leaves native-exit fixtures racing
    detail: |
      The child-proof extension at cmd/internal/couchcmd/worker_cue_coordinator_test.go:62 fixes newWorkerCueRig but leaves the same live-wrapper pattern at native_exit_test.go:41, :147 and :167. Running go test -race ./cmd/internal/couchcmd -run '^TestNativeExit(AdmitsOnlyTheCurrentChild|WithoutChildEvidenceIsUnsupported|ReachesTheHubAsOneLifecycleNotice)$' -count=20 -timeout=120s failed: newExitRig's write at :41 races messageService.execute at message_service.go:896, where the value-receiver call copies authority. These are all remaining connected-fixture child assignments; worker_cue_coordinator_test.go:309 is the documented no-wrapper control. Compose all three native-exit variants before service construction through the existing decorator, preserve their assertions, and repeat this regression command. Add a Revisions entry completing the demonstrated child-proof family.
  - id: new
    severity: Important
    kind: defect
    family: acceptance-criteria-track-contract
    title: |
      Done-when does not cover the added child-proof contract
    detail: |
      workshop/issues/000860-multi-hub-fixture-authority-race.md:53-57 covers the original notification race, multi-hub startup policy and hub/hubs assignments, but neither child-proof revision at :96 and :104 has corresponding acceptance criteria. These are both instances of the same contract extension, implemented in worker_cue_coordinator_test.go. Update Done-when to require pre-start child-proof composition and repeated race coverage preserving the default, role-change, committed-report and bounded-proof controls, including the native-exit siblings identified above.
```

Done-when: stale — the child-proof revisions introduce behavior and controls beyond the hub/hubs acceptance criteria.

1. **Strengths**

   - `multiHubWaitRig` constructs ownership before starting readers and reuses `decorateAuthority`.
   - The new startup-policy regression checks both configured hubs, initially and after restart (`supervision_test.go:449`).
   - Live ownership transitions use the existing mutex; notification and bounded-title assertions remain intact.
   - Production code, wire formats and timing budgets are unchanged.

2. **Critical findings**

   The remaining native-exit assignments reproduce the exact publication race being repaired. Sweep the three sites together using pre-start composition.

3. **Important findings**

   Align Done-when with both child-proof revisions and their preserved controls.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Selected notification, multi-hub, coordinator, guard, recovery, archive and lifecycle tests passed 20 race repetitions: **39.663s**.
   - Selected admission, receipt-scope and prepared-switch tests passed 20 race repetitions: **34.582s**.
   - Native-exit sibling qualification failed with a measured data race: **13.901s**.
   - `git diff --check` passed. No source files were modified; reverse-fix testing was not performed.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing composition seam reused.
   - **ARCH-PURE: pass** — changes remain in integration fixtures; no business logic added to IO handlers.
   - **ARCH-PURPOSE: flag** — the demonstrated child-proof family remains incomplete.
   - **ARCH-EVOLVE: pass** — no persisted format changes.
   
   No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating all connected child-authority assignments, moving native-exit setup before construction, and recording repeated race qualification. Update Done-when to include that scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"26ceec7f-df70-453d-a570-bea58f9fe642","repository":"github.com/xianxu/pair","issue_id":"000860","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9dae7d0c560f4a31","reviewed_head":"17ce2edb890108642629051661d7866b103882a0","slot":"pair:9","requested_at":"2026-10-24T07:47:33.703031Z"},"phase":"finished","pid":2096,"started_at":"2026-10-24T07:47:33.84405Z","finished_at":"2026-10-24T07:49:33.89961Z","session_id":"01a16ce1-1aa1-79a0-b7bb-40fb6f5364ec","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.364010292s
head: "17ce2edb890108642629051661d7866b103882a0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T01:04:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 860 — Multi-hub test fixture mutates authority after message readers start |
| repo | pair |
| issue file | workshop/issues/000860-multi-hub-fixture-authority-race.md |
| boundary | whole-issue close |
| milestone | — |
| window | 99846a5015f753f05edaf4778618849f377a2f71..67a53ea2b4d26e3ed053150decb566c473e45068 |
| command | sdlc close --issue 860 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T01:04:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are addressed. The fixtures now compose child and hub authority before publication, and the acceptance criteria cover the expanded contract. Regression controls fail against the unfixed sources. No new blocking findings were identified; the recorded pre-close suite remains a separate landing requirement.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      All three native-exit assignments now use pre-construction decorators (native_exit_test.go:38, :152, :166). Twenty race repetitions passed. The new source guard passes on head and fails against the prior native-exit source, identifying exactly the three original writes. The issue's Revisions section records the complete family.
  - id: BR-2
    disposition: addressed
    note: |
      Done-when at workshop/issues/000860-multi-hub-fixture-authority-race.md:53-54 now requires pre-construction child authority, the named proof and native-exit controls, and mechanical enumeration. These requirements match the worker-cue/native-exit fixtures and authority_source_test.go; repeated race checks preserve their assertions.
```

1. **Strengths**

   - Existing `decorateAuthority` composition is reused across the fixture constructors.
   - `supervision_test.go:449` checks automation policy for both hubs, initially and after restart.
   - `authority_source_test.go:14` rejects direct child/hub/hubs assignments outside four individually checked exceptions.
   - Ownership transitions retain mutex protection; production code and existing behavioral assertions are unchanged.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None requiring action.

5. **Test coverage notes**

   - Child-proof, native-exit, notification, supervision and startup-policy controls passed 20 race repetitions.
   - Changed guard, recovery, archive and lifecycle controls passed 20 race repetitions.
   - Admission controls passed 20 race repetitions.
   - The startup-policy regression fails with the original multi-hub helper: `/repo3` lacks automation policy.
   - Affected-reader inventory and all 13 affected-selector tests passed.
   - An expanded selection encountered an unchanged startup test’s visible-refusal timeout, then exhausted its 180-second budget. That startup test passed in isolation; the changed controls passed separately.
   - Full package qualification was not rerun during this review. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing composition and synchronization seams reused.
   - **ARCH-PURE: pass** — changes remain in integration fixtures and source-checking glue.
   - **ARCH-PURPOSE: pass** — the enumerated connected child/hub/hubs assignment family is covered.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

   No new user-facing or architectural surface requires README or atlas updates. No Core concepts table is present.

7. **Plan revision recommendations**

   None; the existing revisions cover the delivered scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6cd55195-e21a-4508-a3fa-f2d9474724d6","repository":"github.com/xianxu/pair","issue_id":"000860","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b6dc7b87fc54ff65","reviewed_head":"67a53ea2b4d26e3ed053150decb566c473e45068","slot":"pair:9","requested_at":"2026-10-24T07:58:01.103807Z"},"phase":"finished","pid":20255,"started_at":"2026-10-24T07:58:01.283889Z","finished_at":"2026-10-24T08:03:13.892754Z","session_id":"01a16cea-adc5-78b3-9958-bc4742719fa7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.043943583s
head: "67a53ea2b4d26e3ed053150decb566c473e45068"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
