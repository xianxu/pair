# Boundary Review — pair#726 (whole-issue close)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | whole-issue close |
| milestone | — |
| window | e2ac2a19f253961b715b6edf104ca82d72b5c40a..dd8a86665c176e52ab7195a431622aada3b1ef06 |
| command | sdlc close --issue 726 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T09:52:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Tests for the memo and admission pass. Now I'm reverting the BR-8 and BR-12 fixes to confirm their regression tests fail without them.

Both fixes go red when reverted, and the checkout is restored. Next I'm checking which couchcore test failed.

The only couchcore failure is the sandbox refusing `git init`. That is the known confinement issue (BR-3, already withdrawn). Last, a quick look at the concurrent Discover and the scan-skip code.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Two of the three open findings (BR-8, BR-12) are now fixed. Each has a regression test that I confirmed goes red when its fix is reverted in this checkout. Once the fix was restored, the memo and admission tests passed. BR-16 is fixed in the atlas and in the Plan's M2 bullet. But the finding's own rule ("grep the old phrase") still finds one stale copy, the doc comment of the `--resume-candidates` wording test, so I'm keeping it open. It is Minor and does not block. The only couchcore test failure is the sandbox refusing `git init` outside its directories. That is the known confinement issue already disposed as BR-3, so I'm not raising it again. No new findings this round.

1. **Strengths**
   - `hub_target_memo.go:35-46`: each entry keeps a `started` time, used for ordering (BR-10), and a separate `finished` time, used for aging. A slow failed resolution now gives its own error for the full 30 s after it ends.
   - `hub_target_memo.go:104-107`: `Warm` caps its own wait at `slotAdmissionBudget`, so every caller is bounded no matter what context it passes. The new receipt-filter and move callers get this for free.
   - Both regression tests are deterministic. `TestHubTargetMemoAgesFailuresFromTheirEnd` uses a fake clock. In `TestHubTargetMemoWarmCapsItsOwnWait` the `time.After` is only a safety bound, never the test's oracle.
   - `recovery_evidence_os.go:199-209`: the capture-identity fence now has one site, and the skipped scan goes through `collectCandidates` (`!SkipCandidates`).
   - `slotcatalog.go`: the concurrent row verification gives a deterministic outcome (it reports the lowest-numbered fatal error) and still checks the replacement fence on every row.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `resume_candidates_test.go:194`: the doc comment still says "affects only the inferred choices", which is the wording BR-16 removed (see the BR-16 disposition).
   - Optional: `hub_guards.go:250` and `hub_move.go:88` still wrap their own `slotAdmissionBudget`, which is now redundant with `Warm`'s cap. It is harmless and you may keep it on purpose.
   - Optional: `slotcatalog.go` doesn't cancel the other rows after a fatal one. The run is bounded by `MaxSlotCandidates` and the 8-wide gate, so this only matters for speed.

5. **Test coverage**
   - Mutation checks on `hub_target_memo.go`:
     - Aging failures from `started` turns `TestHubTargetMemoAgesFailuresFromTheirEnd` red.
     - Removing `Warm`'s `WithTimeout` turns `TestHubTargetMemoWarmCapsItsOwnWait` red (it waited 30 s).
   - With the fixes restored, the `couchcmd` tests (HubTargetMemo, HubAdmission, HubMove, ResumeCandidates) pass, and so do `couchcore`'s SkipCandidates and PreparedRecovery tests.
   - `TestDiscoverVerifiesSlotsConcurrently` can't run here because of the sandbox's git restriction (the BR-3 environment issue).

6. **Architecture**
   - **Pass:** ARCH-DRY (one `checkHubSlotOperation`, one capture fence); ARCH-PURE (the memo takes its resolver and clock as injected functions); ARCH-PURPOSE (admission stays inside its budget on every request-path read of a target slot); ARCH-CONSTRAINTS (1.5 s budget below the 2 s exchange; flights and fan-out capped at 8); ARCH-SECURE (a stale memo can only admit a job that is re-proved fresh in prepare and at the effect gate; a move re-resolves its slots before the effect).
   - **ARCH-ORDER: pass.** A cancelled resolution isn't remembered, and an older resolution can't overwrite a newer one.
   - **ARCH-FUNERAL: pass.** The memo is swept on write and capped at 256 entries; flights are joined at `Close`.
   - **ARCH-MOCK, ARCH-EVOLVE:** N/A for these fixes. The memo is in memory only, and no persisted format changed.

7. **Plan revisions:** none. The Plan's M2 bullet at issue line 78 already has the corrected wording.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      Warm caps its own wait at slotAdmissionBudget (hub_target_memo.go:104-107); without it TestHubTargetMemoWarmCapsItsOwnWait fails after 30 s.
  - id: BR-12
    disposition: addressed
    note: |
      Entries now carry started (ordering) and finished (expiry), and live() ages failures from finished; reverting to started turns TestHubTargetMemoAgesFailuresFromTheirEnd red.
  - id: BR-16
    disposition: not-addressed
    note: |
      atlas/couch.md:83-85 and the issue M2 bullet (line 78) are fixed, but grepping the old phrase still finds the resume_candidates_test.go:194 doc comment ("affects only the inferred choices"); reword it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"357bd11f-8d43-4d2f-b999-75401f4c938e","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":6,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/029b9f6d5e0fd8a1","reviewed_head":"dd8a86665c176e52ab7195a431622aada3b1ef06","slot":"pair:2","requested_at":"2026-10-20T16:50:23.206698Z"},"phase":"finished","pid":50256,"started_at":"2026-10-20T16:50:23.41367Z","finished_at":"2026-10-20T16:52:58.921833Z","session_id":"aac39a45-9f3f-4335-8d47-bd8400a32dc7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.944583667s
head: "dd8a86665c176e52ab7195a431622aada3b1ef06"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T11:30:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | whole-issue close |
| milestone | — |
| window | b812d578e3f000f1a724db6b7bf4fa4b2fa09724..e75f282584c7efc19abebc5d6ab999a1b0148091 |
| command | sdlc close --issue 726 |
| reviewer | codex (openai; author anthropic,openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:30:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-16 is addressed, and the focused tests pass. However, this interdiff adds repository-wide discovery, startup prewarming, and batch acceptance behavior beyond the prior findings. The supplied fix-review contract requires a full close review for that scope change.

```findings
dispose:
  - id: BR-16
    disposition: addressed
    note: |
      atlas/couch.md:83-85 and the issue's M2 bullet at line 79 correctly attribute the confirmed choice to the established binding, matching resume_candidates.go:123-126. The remaining stale test comment is corrected by this interdiff at resume_candidates_test.go:194-196; this is a prose-only correction.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: boundary-review-scope
    title: |
      scope changed after close — run a full `sdlc close`
    detail: |
      The issue's new Revisions entry at line 105 introduces repository-wide single-flight discovery, startup prewarming, and Tier A batch acceptance beyond the outstanding wording finding. Executable changes in hub_target_memo.go:108-274, message_service.go:701-709, and slotcontext.go:203-222 implement that expansion. Run a full sdlc close covering the expanded scope; this narrow fix review cannot authorize it.
```

1. **Strengths**

   - BR-16’s atlas, plan, and test-comment wording now agree with the actual CLI output.
   - Repository discovery and name lookup are injected into the memo.
   - Batch acceptance tests exercise the broker socket, CLI, console recovery, and persisted live incarnations.

2. **Critical findings**

   - The scope expansion identified above requires a full close review. This is a boundary coverage failure; the focused tests passing does not resolve it.

3. **Important findings**

   - None additionally raised.

4. **Minor findings**

   - None additionally raised.

5. **Test coverage notes**

   - Passed focused tests in `couchcmd` and `couchcore`: memo, batch socket acceptance, repository-slot resolution, and incomplete-scan wording.
   - BR-16 requires prose inspection rather than a runtime mutation test.
   - These checks do not constitute full validation of the expanded scope.

6. **Architectural notes**

   - **ARCH-DRY — pass:** repository name resolution reuses `ResolveRepositoryName`; flight admission shares `startFlight`.
   - **ARCH-PURE — pass:** the added IO sources and clock are injected into the memo.
   - **ARCH-PURPOSE — pass within inspected behavior:** batch tests check successful resumes and retain fresh authority proofs. Expanded requirements still need full review.
   - **ARCH-EVOLVE — pass:** this interdiff introduces no persisted-format change.
   - Atlas documents the added warming flow. No new command, flag, or configuration key triggers a README update.

7. **Plan revision recommendations**

   - The new `## Revisions` entry records substantive scope expansion. Carry it into the required full close review and reconcile M1’s older “only the existing fresh resolver fills it” and per-reference warming descriptions with the repository-wide implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"df361ea6-3d73-4cc7-b2df-0900e748f8f9","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"fix-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/256b88f12cc8bd75","reviewed_head":"e75f282584c7efc19abebc5d6ab999a1b0148091","slot":"pair:2","requested_at":"2026-10-20T18:29:16.820695Z"},"phase":"finished","pid":42033,"started_at":"2026-10-20T18:29:16.923724Z","finished_at":"2026-10-20T18:30:27.376277Z","session_id":"01a15a93-2cfc-7fc1-a3cb-f76a6638a9ed","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.598512917s
head: "e75f282584c7efc19abebc5d6ab999a1b0148091"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T11:36:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 726 — Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis |
| repo | pair |
| issue file | workshop/issues/000726-hub-lifecycle-ops-time-out-client-side-resume-pair-6-relaunch-parley-nvim-4-context-deadline-exceeded-with-no-diagnosis.md |
| boundary | whole-issue close |
| milestone | — |
| window | e2ac2a19f253961b715b6edf104ca82d72b5c40a..06fde08625b18c4fbbfb7ab4e84d8dbf37a59524 |
| command | sdlc close --issue 726 |
| reviewer | codex (openai; author anthropic,openai) |
| isolation | fenced |
| timestamp | 2026-10-20T11:36:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-17 is addressed by this full-range review. All three changed-package test suites, focused race tests, and vet pass. Two correctness gaps remain: the admission budget excludes work before and after the guard, and repository warming can resolve a reference differently from the existing fresh resolver.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Failure entries retain the resolver error; TestHubTargetMemoRemembersALateFailure covers late completion and retry.
  - id: BR-2
    disposition: addressed
    note: |
      Move target resolution uses Warm; TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh covers admission and fresh revalidation.
  - id: BR-3
    disposition: withdrawn
    note: |
      Discover concurrency and replaced-host tests ran successfully under the race detector in this environment.
  - id: BR-4
    disposition: addressed
    note: |
      Background flights and Discover verification have eight-worker caps, covered by focused race tests.
  - id: BR-5
    disposition: addressed
    note: |
      Memo access and writes consistently use hubTargetKey.
  - id: BR-6
    disposition: addressed
    note: |
      Admission and fresh guards share checkHubSlotOperation with injected sources.
  - id: BR-7
    disposition: addressed
    note: |
      Discover concurrency uses an in-flight rendezvous rather than elapsed time as its success oracle.
  - id: BR-8
    disposition: addressed
    note: |
      Receipt filtering bounds its memo wait; Warm also enforces its own budget.
  - id: BR-9
    disposition: addressed
    note: |
      The issue Revisions records failure retention, move revalidation, caps, ordering, and additional visible states.
  - id: BR-10
    disposition: addressed
    note: |
      remember rejects older-started results; late individual and repository completion regressions pass.
  - id: BR-11
    disposition: addressed
    note: |
      Writes sweep expired entries and cap the map at 256; the sweep/cap regression passes.
  - id: BR-12
    disposition: addressed
    note: |
      Failure expiry uses finished time; TestHubTargetMemoAgesFailuresFromTheirEnd passes.
  - id: BR-13
    disposition: addressed
    note: |
      The oversized-store fixture exercises metadata and record limits, compares counted bytes, and passes with the routing regression.
  - id: BR-14
    disposition: addressed
    note: |
      ReadRequest has one Source-nil capture fence after optional candidate collection.
  - id: BR-15
    disposition: addressed
    note: |
      resume_candidates.go attributes the confirmed choice to the established binding without claiming all incomplete observations affect only inference.
  - id: BR-16
    disposition: addressed
    note: |
      atlas/couch.md:83-85 and the Plan M2 bullet match the CLI wording and established-binding recovery path.
  - id: BR-17
    disposition: addressed
    note: |
      This review inspected the complete pinned range, including repository batching, startup prewarm, production wiring, and Tier A acceptance. The issue Revisions at line 121 explicitly supersedes the original per-reference design.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: admission-sync-discover-over-exchange
    title: |
      The admission budget does not cover the complete admission path
    detail: |
      hub_guards.go:162-168 starts the 1.5 s budget after handleSlotOperation has authenticated the caller, then cancels it before slotOps.handle. Earlier liveCaller reads can wait up to one second for the store lock (message_service.go:1256; couchcore/retention.go:299), while later queueTarget performs another repository-name read with a separate 500 ms allowance. A caller check that succeeds after 700 ms followed by a cold target resolution reaches the parent 2 s deadline before the inner budget expires; transport.go then drops the refusal. This is the 3rd finding in family admission-sync-discover-over-exchange. Earlier rounds fixed instances; establish one deadline for the entire lifecycle admission path, including authentication, target resolution, queue-key lookup, and enqueue, and sweep slot, move, and status paths against that rule (ARCH-CONSTRAINTS, ARCH-PURPOSE). Add a socket regression combining delayed successful authentication with blocked target resolution, plus delayed queue-name lookup.
  - id: new
    severity: Critical
    kind: defect
    family: canonical-reference-resolution
    title: |
      Repository warming bypasses sibling-directory precedence
    detail: |
      hub_target_memo.go:119 resolves repository names directly against enrollment, then returns the batched slot at lines 135-146. The existing fresh resolver instead gives an exact sibling directory precedence over enrolled names and prefixes (couchcore/slotcontext.go:152-156), explicitly tested by TestExistingSiblingDirectoryBeatsPrefix. With an existing sibling parley and an enrolled parley.nvim, Warm("parley:1") can return parley.nvim:1 while Fresh resolves parley:1. If those repositories belong to different hubs, admission refuses a valid operation before fresh authority can correct the snapshot. Derive memo repository identity through the same reference-resolution rule, retaining batching after identity is established; cover cold and prewarmed references, including prefixes and aliases (ARCH-DRY, ARCH-PURPOSE).
```

1. **Strengths**

   - Admission snapshots remain separate from fresh prepare/effect authority checks.
   - Flights share a capacity limit, service cancellation, and shutdown joining.
   - Tier A tests exercise real broker sockets, CLI polling, console recovery, and persisted live incarnations.
   - Candidate-scan skipping preserves capture fencing and non-candidate recovery evidence.

2. **Critical findings**

   - Two, detailed above. Both need regression coverage with the correction.

3. **Important findings**

   - None additional.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/artifactpath`
   - Focused hub and couchcore tests under `-race`
   - Existing sibling-precedence regression
   - `go vet ./cmd/internal/couchcmd ./cmd/internal/couchcore`

   Current tests do not combine admission-stage delays or compare memo resolution against sibling precedence. Live acceptance was not performed.

6. **Architectural notes**

   - **ARCH-DRY — flag:** memo and fresh resolution implement different reference-selection rules.
   - **ARCH-PURE — pass:** resolver, repository sources, and clock are injected.
   - **ARCH-PURPOSE — flag:** complete admission still lacks the promised bounded response.
   - **ARCH-MOCK — pass:** tests use shared production seams and stateful portable fixtures; live Tier B remains explicitly separate.
   - **ARCH-CONSTRAINTS — flag:** budgets cover individual stages rather than the complete exchange.
   - **ARCH-SECURE — pass:** fresh authority and capture-identity checks remain intact.
   - **ARCH-ORDER — pass:** late-result ordering and cancellation/join behavior have controlled tests.
   - **ARCH-FUNERAL — pass:** entries are swept and capped; flights end with the service.
   - **ARCH-EVOLVE — pass:** no persisted-format change.

   Atlas covers the new internal flow. No new command, flag, or configuration key requires a README addition. The Plan contains no Core concepts table.

7. **Plan revision recommendations**

   Add `## Revisions` entries establishing:

   - One admission deadline spanning authentication through enqueue, with controlled multi-stage delay tests.
   - One canonical repository-selection rule shared by memo and fresh resolution, including sibling precedence and cold/prewarmed equivalence tests.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f58d3a93-3d1e-4e01-94e6-eb42bb68ca43","repository":"github.com/xianxu/pair","issue_id":"000726","recipe":"milestone-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/63fd352e96effb43","reviewed_head":"06fde08625b18c4fbbfb7ab4e84d8dbf37a59524","slot":"pair:2","requested_at":"2026-10-20T18:31:55.466403Z"},"phase":"finished","pid":64696,"started_at":"2026-10-20T18:31:55.556003Z","finished_at":"2026-10-20T18:36:34.392281Z","session_id":"01a15a95-989b-7e30-b96e-d1102c354c12","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.477914166s
head: "06fde08625b18c4fbbfb7ab4e84d8dbf37a59524"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
