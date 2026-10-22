# Boundary Review — pair#773 (whole-issue close)

| field | value |
|-------|-------|
| issue | 773 — startup: terminal readiness truncates the caller to a fixed 10-second bound |
| repo | pair |
| issue file | workshop/issues/000773-startup-terminal-readiness-truncates-the-caller-to-a-fixed-10-second-bound.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7702a3dddbfdb5c7d8a021bb8949df98f0a6b282..e9c154b231493652ebe6e2c285ce0e57e624c625 |
| command | sdlc close --issue 773 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T06:08:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The tracked launch path removes the fixed readiness cutoff and preserves interactive lifetime correctly. SHIP is blocked by an unswept public warm-attach path, incomplete console stage timing, and warm-adoption tests whose failure injections now expire during preparation.

```findings
findings:
  - id: new
    severity: Critical
    family: startup-budget-consumer-coverage
    title: |
      Public warm attachment does not transport the caller-owned startup deadline
    detail: |
      cmd/internal/couchcore/resume.go:452 routes ordinary listed-terminal resume through attachListedTerminalAdmitted, bypassing launchTrackedThread. warm_attach.go:92 supplies no PAIR_STARTUP_DEADLINE, and line 116 retains the relative ten-second helper timeout. Consequently the helper cannot reject delayed exec against the caller's absolute deadline, and Pair selects a fresh fallback. ARCH-PURPOSE: complete the plan's explicit warm-attach sweep by selecting and transporting one deadline while preserving the separate interactive lifetime. Add regression coverage through this public entry, including delayed helper execution and deadline preservation.
  - id: new
    severity: Important
    family: stage-progress-consumer-coverage
    title: |
      Local console progress displays operation age instead of current-stage age
    detail: |
      cmd/internal/couchcore/startup_progress.go:21 forwards only stage text to the console reporter. couchtty/menu_operations.go:311 updates that text, but lines 329–332 continue measuring from the operation's StartedAt. After twenty seconds of preparation and five seconds of readiness, the console displays twenty-five seconds beside readiness; the remote CLI correctly displays stage age. ARCH-PURPOSE: retain a stage-start timestamp in the console's pure progress transition, preserve it on repeated reports, and test stage changes with controlled time.
  - id: new
    severity: Important
    family: deterministic-failure-injection
    title: |
      The shared startup timer invalidates warm-adoption failure fixtures
    detail: |
      cmd/internal/couchcore/startup_budget.go:18–28 turns resumeRegistrationTimeout into a real timer before preparation. warm_failure_test.go:274–279 still supplies a 200 ms sentinel and explicitly expects its registrationDeadline factory to avoid a timer. The broader package run failed TestAFailedTrackedWarmAdoptionRetireStillLeavesTheRecordRecoverable and two TestAFailedTrackedWarmAdoptionKeepsItsSession cases; one expired during spawn, and the retire case reached zero intended session reads. ARCH-ORDER: separate budget-propagation assertions from deterministic failure injection so these tests reach and verify their named transitions without depending on preparation speed.
```

1. **Strengths**

   - Production-boundary virtual-time tests cover readiness at 12 seconds, absent readiness, cancellation, and admitted-client survival.
   - The shared codec rejects malformed, noncanonical, expired, and explicitly zero deadline inputs.
   - Receipt progress preserves stage age and rejects terminal and retired-request updates.
   - README and atlas updates describe the deadline, observation, recovery, and rollout contracts.

2. **Critical findings**

   Public warm attachment remains outside the shared deadline contract. Preserve its existing short query budget while adding absolute deadline transport; do not deadline-bound the admitted client.

3. **Important findings**

   Implement console stage age and repair the warm-adoption fixtures described above.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed focused startup/helper/handoff/progress tests, full startupbudget, launcher, couchmessage, couchcmd, artifactpath and repolint packages, affected-reader inventory, and diff checks.

   The broader Couch core run failed after 340.6 seconds with the three warm-adoption failures above. No retry was performed. The retained native Claude/Codex acceptance log’s SHA-256 matches its evidence record; subsequent commits contain no runtime behavior changes.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared deadline/default/codec ownership.
   - **ARCH-PURE — pass:** pure selection and receipt transitions remain separate from IO.
   - **ARCH-PURPOSE — flag:** warm attachment and console stage timing remain incomplete.
   - **ARCH-MOCK — pass:** existing stateful seams and native conformance harness reused.
   - **ARCH-CONSTRAINTS — flag:** warm helper execution lacks the caller’s absolute bound.
   - **ARCH-SECURE — pass:** transported timestamps are bounded, strictly parsed, and consumed.
   - **ARCH-ORDER — flag:** warm-adoption fixtures no longer reliably reach their intended transitions.
   - **ARCH-FUNERAL — pass:** no new growing runtime artifact family; observer/client cleanup remains bounded.
   - **ARCH-EVOLVE — pass:** no persisted-format change; coordinated ephemeral receipt rollout is documented.

7. **Plan revision recommendations**

   Add `## Revisions` entries explicitly enumerating tracked launches and public borrowed-client attachment as separate deadline consumers, identifying both console and remote progress consumers, and recording the fixture-timing correction. Reopen the affected completion checkboxes until these changes and regression checks pass.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d323f9de-6dc5-40d4-b8cf-4f890b2bf591","repository":"github.com/xianxu/pair","issue_id":"000773","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5a1b4c03b2d10c9d","reviewed_head":"e9c154b231493652ebe6e2c285ce0e57e624c625","slot":"pair:7","requested_at":"2026-10-22T13:00:37.331191Z"},"phase":"finished","pid":20120,"started_at":"2026-10-22T13:00:37.483234Z","finished_at":"2026-10-22T13:08:43.656937Z","session_id":"01a163b3-0070-7372-97d3-e816903d9f5c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.87639125s
head: "e9c154b231493652ebe6e2c285ce0e57e624c625"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-22T07:19:05-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 773 — startup: terminal readiness truncates the caller to a fixed 10-second bound |
| repo | pair |
| issue file | workshop/issues/000773-startup-terminal-readiness-truncates-the-caller-to-a-fixed-10-second-bound.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7702a3dddbfdb5c7d8a021bb8949df98f0a6b282..f3c423c6462058da7433262a1617a9fb42ab3386 |
| command | sdlc close --issue 773 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T07:19:05-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range fulfills the startup-budget contract and resolves all three prior findings. Package tests and focused race checks passed. Each correction’s regression test failed when its fix was removed through a temporary Go overlay. No new blocking findings emerged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      warm_attach.go selects and transports one absolute deadline while preserving interactive lifetime. TestPublicWarmAttachmentTransportsCallerStartupBudget passes and fails against the previous implementation, detecting missing transport, the ten-second helper timeout, and registration renewal. Helper tests separately cover delayed acknowledgement and pre-exec expiry.
  - id: BR-2
    disposition: addressed
    note: |
      menu_operations.go retains stage-start time, preserves it on repeated reports, and uses it for row and notice rendering. TestMenuOperationStageAgeProjectsRowAndNotice passes; removing the timestamp correction reproduces the incorrect 25s readiness display.
  - id: BR-3
    disposition: addressed
    note: |
      General warm-failure fixtures no longer install the preparation-sensitive 200ms timer. Dedicated propagation controls use virtual time. TestTrackedWarmFailureInjectionSurvivesPreparationDelay passes; restoring the sentinel fails before the intended registration transition.
```

1. **Strengths**

   - Deadline selection and strict exec encoding share one pure owner in `cmd/internal/startupbudget/budget.go`.
   - Production handoff tests exercise readiness at virtual 12 seconds, cancellation, absent readiness, remaining-budget enforcement, and admitted-client survival.
   - Console and remote progress retain stage age and reject stale updates.
   - README and atlas document the fallback, separate lifetimes, recovery limits, and coordinated deployment.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed complete packages: `startupbudget`, `couchmessage`, `couchtty`, `couchcore`, `launcher`, and `couchcmd`.
   - Passed focused race checks across the five integration/progress packages and the admission-budget lint regression.
   - All three temporary negative controls failed as expected; overlays were removed and the checkout remains unchanged.
   - Retained cold/warm native logs match their recorded SHA-256 hashes and ancestor heads. Native acceptance was not rerun; the opt-in test compiled and skipped normally.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared selector/default/codec; both launch owners reuse it.
   - **ARCH-PURE — pass:** Deadline values and progress transitions remain separate from clock and IO glue.
   - **ARCH-PURPOSE — pass:** Tracked launches, public warm attachment, and direct handoffs carry the startup budget.
   - **ARCH-MOCK — pass:** Existing stateful runner/runtime seams exercise production boundaries; retained native conformance supplements them.
   - **ARCH-CONSTRAINTS — pass:** Absolute startup bounds coexist with short query budgets; virtual-time tests check polling growth.
   - **ARCH-SECURE — pass:** Transported deadlines receive bounded, canonical validation and are consumed before descendants launch.
   - **ARCH-ORDER — pass:** Controlled cancellation/readiness/exit sequences preserve fail-closed admission and joined cleanup.
   - **ARCH-FUNERAL — pass:** No new runtime artifact family; timer, observer, and child cleanup remain bounded.
   - **ARCH-EVOLVE — pass:** No persisted-format change; ephemeral receipt compatibility limits are documented.

7. **Plan revision recommendations:** None. The correction revision matches the delivered implementation.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a0ca37da-1daf-4bb6-af4d-a55bc7629653","repository":"github.com/xianxu/pair","issue_id":"000773","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/beb50a2909b4ab64","reviewed_head":"f3c423c6462058da7433262a1617a9fb42ab3386","slot":"pair:7","requested_at":"2026-10-22T14:15:15.614014Z"},"phase":"finished","pid":55129,"started_at":"2026-10-22T14:15:15.705453Z","finished_at":"2026-10-22T14:19:04.594664Z","session_id":"01a163f7-5507-71f0-a0e8-910505da279a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.779678833s
head: "f3c423c6462058da7433262a1617a9fb42ab3386"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
